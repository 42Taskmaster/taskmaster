package main

import (
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/VisorRaptors/taskmaster/machine"
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
	Process *Process
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
		Context: ProcessMachineContext{
			Process: process,
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
					ProcessEventStarted:        ProcessStateRunning,
					ProcessEventStop:           ProcessStateStopping,
					ProcessEventExitedTooEarly: ProcessStateBackoff,
				},
			},

			ProcessStateBackoff: machine.StateNode{
				On: machine.Events{
					ProcessEventStart: ProcessStateStarting,
					ProcessEventFatal: ProcessStateFatal,
				},
			},

			ProcessStateRunning: machine.StateNode{
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

func (process *Process) Init() {
	commandChunks := strings.Split(process.Program.Config.Cmd, " ")

	cmd := exec.Command(commandChunks[0], commandChunks[1:]...)
	cmd.Env = process.Program.Cache.Env
	cmd.Stdin = nil
	cmd.Stdout = process.Program.Cache.Stdout
	cmd.Stderr = process.Program.Cache.Stderr

	process.Cmd = cmd
}

func (process *Process) Start() error {
	process.Init()

	if err := process.Cmd.Start(); err != nil {
		return &ErrProcessStarting{
			Err: err,
		}
	}

	go func() {
		process.Cmd.Wait()

		event := ProcessEventStopped
		if process.Cmd.ProcessState.Exited() {
			event = ProcessEventExit
		}

		_, err := process.Machine.Send(event)
		if err != nil {
			log.Panicf("expected no error to be returned but got %v\n", err)
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
	processContext := context.(ProcessMachineContext)

	err := processContext.Process.Start()
	if err != nil {
		return machine.NoopEvent, &ErrProcessAction{
			ID:  processContext.Process.ID,
			Err: err,
		}
	}

	return ProcessEventStarted, err
}

func ProcessStopAction(context machine.Context) (machine.EventType, error) {
	var processContext = context.(ProcessMachineContext)

	log.Printf("processContext = %#v\n", processContext)
	if processContext.Process == nil {
		log.Panic("processContext.Process is nil")
	}
	if processContext.Process.Program == nil {
		log.Panic("processContext.Process.Program is nil")
	}

	var (
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
				Action:    ProgramTaskActionKill,
				Program:   processContext.Process.Program,
				ProcessID: processContext.Process.ID,
			}
		case <-deadCh:
		}
	}()

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
