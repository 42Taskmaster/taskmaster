package main

import (
	"log"
	"os/exec"
	"time"

	"github.com/VisorRaptors/taskmaster/machine"
	"github.com/VisorRaptors/taskmaster/parser"
)

const (
	ProcessStateStarting machine.StateType = "STARTING"
	ProcessStateBackoff  machine.StateType = "BACKOFF"
	ProcessStateRunning  machine.StateType = "RUNNING"
	ProcessStateStopping machine.StateType = "STOPPING"
	ProcessStateStopped  machine.StateType = "STOPPED"
	ProcessStateExited   machine.StateType = "EXITED"
	ProcessStateFatal    machine.StateType = "FATAL"
	ProcessStateUnknown  machine.StateType = "UNKNOWN"
)

const (
	ProcessEventStart          machine.EventType = "start"
	ProcessEventStarted        machine.EventType = "started"
	ProcessEventStop           machine.EventType = "stop"
	ProcessEventExit           machine.EventType = "exit"
	ProcessEventExitedTooEarly machine.EventType = "exited-too-early"
	ProcessEventStopped        machine.EventType = "stopped"
	ProcessEventFatal          machine.EventType = "fatal"
)

type ProcessMachineContext struct {
	Process    *Process
	Starttries int
}

type Process struct {
	ID string

	Program *Program

	Cmd     *exec.Cmd
	Machine machine.Machine

	DeadCh chan struct{}
}

func NewProcess(id string, program *Program) *Process {
	process := &Process{
		ID:      id,
		Program: program,
	}

	process.Machine = machine.Machine{
		Context: &ProcessMachineContext{
			Process:    process,
			Starttries: 0,
		},

		Initial: ProcessStateStopped,

		StateNodes: machine.StateNodes{
			ProcessStateStopped: machine.StateNode{
				On: machine.Events{
					ProcessEventStart: ProcessStateStarting,
				},
			},

			ProcessStateStarting: machine.StateNode{
				Actions: []machine.Action{
					ProcessStartAction,
				},

				On: machine.Events{
					ProcessEventStarted: ProcessStateRunning,
					ProcessEventStop:    ProcessStateStopping,
					ProcessEventStopped: ProcessStateBackoff,
					ProcessEventExit:    ProcessStateBackoff,
				},
			},

			ProcessStateBackoff: machine.StateNode{
				Actions: []machine.Action{
					ProcessBackoffAction,
				},

				On: machine.Events{
					ProcessEventStart: ProcessStateStarting,
					ProcessEventFatal: ProcessStateFatal,
				},
			},

			ProcessStateRunning: machine.StateNode{
				Actions: []machine.Action{
					ProcessRunningAction,
				},

				On: machine.Events{
					ProcessEventStop: ProcessStateStopping,
					ProcessEventExit: ProcessStateExited,
				},
			},

			ProcessStateStopping: machine.StateNode{
				Actions: []machine.Action{
					ProcessStopAction,
				},

				On: machine.Events{
					ProcessEventStopped: ProcessStateStopped,
				},
			},

			ProcessStateExited: machine.StateNode{
				On: machine.Events{
					ProcessEventStart: ProcessStateStarting,
				},
			},

			ProcessStateFatal: machine.StateNode{
				On: machine.Events{
					ProcessEventStart: ProcessStateStarting,
				},
			},
		},
	}
	process.Machine.Init()

	return process
}

func (process *Process) Init() error {
	parsedCommand, err := parser.ParseCommand(process.Program.Config.Cmd)
	if err != nil {
		return err
	}

	cmd := exec.Command(parsedCommand.Cmd, parsedCommand.Args...)
	cmd.Env = process.Program.Cache.Env
	cmd.Stdin = nil
	cmd.Stdout = process.Program.Cache.Stdout
	cmd.Stderr = process.Program.Cache.Stderr
	cmd.Dir = process.Program.Config.Workingdir

	process.Cmd = cmd

	return nil
}

func (process *Process) Start() error {
	if err := process.Init(); err != nil {
		return err
	}

	process.Program.ProgramManager.Taskmasterd.SetUmask(process.Program.Config.Umask)
	if err := process.Cmd.Start(); err != nil {
		process.Program.ProgramManager.Taskmasterd.ResetUmask()
		return &ErrProcessStarting{
			Err: err,
		}
	}
	process.Program.ProgramManager.Taskmasterd.ResetUmask()

	process.DeadCh = make(chan struct{})

	go func() {
		select {
		case <-time.After(time.Duration(process.Program.Config.Starttime) * time.Second):
			process.Machine.Send(ProcessEventStarted)
		case <-process.DeadCh:
			return
		}
	}()

	go func() {
		process.Cmd.Wait()

		close(process.DeadCh)

		event := ProcessEventStopped
		if process.Cmd.ProcessState.Exited() {
			event = ProcessEventExit
		}

		_, err := process.Machine.Send(event)
		if err != nil {
			log.Printf("expected no error to be returned but got %v\n", err)
		}
	}()

	return nil
}

func (process *Process) Stop(signal StopSignal) error {
	err := process.Cmd.Process.Signal(signal.ToOsSignal())
	if err != nil {
		return &ErrProcessStopping{
			Err: err,
		}
	}
	return nil
}

func ProcessStartAction(context machine.Context) (machine.EventType, error) {
	processContext := context.(*ProcessMachineContext)

	err := processContext.Process.Start()
	if err != nil {
		return machine.NoopEvent, &ErrProcessAction{
			ID:  processContext.Process.ID,
			Err: err,
		}
	}

	return machine.NoopEvent, err
}

func ProcessStopAction(context machine.Context) (machine.EventType, error) {
	var (
		processContext   = context.(*ProcessMachineContext)
		stopsignal       = processContext.Process.Program.Config.Stopsignal
		stoptime         = processContext.Process.Program.Config.Stoptime
		deadCh           = processContext.Process.DeadCh
		programTasksChan = processContext.Process.Program.ProgramManager.ProgramTaskChan
	)

	err := processContext.Process.Stop(stopsignal)
	if err != nil {
		return machine.NoopEvent, &ErrProcessAction{
			ID:  processContext.Process.ID,
			Err: err,
		}
	}

	go func() {
		select {
		case <-time.After(time.Duration(stoptime) * time.Second):
			programTasksChan <- ProgramTask{
				Action:  ProgramTaskActionKill,
				Program: processContext.Process.Program,
				Process: processContext.Process,
			}
		case <-deadCh:
		}
	}()

	return machine.NoopEvent, nil
}

func ProcessBackoffAction(context machine.Context) (machine.EventType, error) {
	processContext := context.(*ProcessMachineContext)
	process := processContext.Process

	switch process.Program.Config.Autorestart {
	case AutorestartOn:
		processContext.Starttries++
		if processContext.Starttries == process.Program.Config.Startretries {
			return ProcessEventFatal, nil
		}
		return ProcessEventStart, nil
	case AutorestartUnexpected:
		exitcode := process.Cmd.ProcessState.ExitCode()
		for _, allowedExitcode := range process.Program.Config.Exitcodes {
			if exitcode == allowedExitcode {
				return machine.NoopEvent, nil
			}
		}
		processContext.Starttries++
		if processContext.Starttries == process.Program.Config.Startretries {
			return ProcessEventFatal, nil
		}
		return ProcessEventStart, nil
	default:
		return machine.NoopEvent, nil
	}
}

func ProcessRunningAction(context machine.Context) (machine.EventType, error) {
	processContext := context.(*ProcessMachineContext)
	processContext.Starttries = 0

	return machine.NoopEvent, nil
}

type ErrProcessAction struct {
	ID  string
	Err error
}

func (err *ErrProcessAction) Unwrap() error {
	return err.Err
}

func (err *ErrProcessAction) Error() string {
	return "error in process id: " + err.ID + ": " + err.Err.Error()
}

type ErrProcessStarting struct {
	Err error
}

func (err *ErrProcessStarting) Unwrap() error {
	return err.Err
}

func (err *ErrProcessStarting) Error() string {
	return "starting: " + err.Err.Error()
}

type ErrProcessStopping struct {
	Err error
}

func (err *ErrProcessStopping) Unwrap() error {
	return err.Err
}

func (err *ErrProcessStopping) Error() string {
	return "stopping: " + err.Err.Error()
}
