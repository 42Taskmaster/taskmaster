package main

import (
	"io"
	"log"
	"os/exec"
	"strconv"
	"strings"

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

type NewProcessArgs struct {
	ID             int
	Cmd            string
	Env            []string
	Stdout, Stderr io.Writer
	StopSignal     StopSignal
}

type ProcessMachineContext struct {
	Process *Process
}

type Process struct {
	ID int

	Cmd     *exec.Cmd
	Machine *machine.Machine

	StopSignal StopSignal
}

func NewProcess(args NewProcessArgs) *Process {
	commandChunks := strings.Split(args.Cmd, " ")

	cmd := exec.Command(commandChunks[0], commandChunks[1:]...)
	cmd.Env = args.Env
	cmd.Stdin = nil
	cmd.Stdout = args.Stdout
	cmd.Stderr = args.Stderr

	process := &Process{
		ID:         args.ID,
		Cmd:        cmd,
		StopSignal: args.StopSignal,
	}

	machine := &machine.Machine{
		Context: ProcessMachineContext{
			Process: process,
		},

		Current: ProcessStateStopped,

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

	process.Machine = machine

	return process
}

type ErrProcessAction struct {
	ID  int
	Err error
}

func (err *ErrProcessAction) Unwrap() error {
	return err.Err
}

func (err *ErrProcessAction) Error() string {
	return "error in process id: " + strconv.Itoa(err.ID) + ": " + err.Err.Error()
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

func (process *Process) Start() error {
	log.Println("before start is called")
	if err := process.Cmd.Start(); err != nil {
		return &ErrProcessStarting{
			Err: err,
		}
	}
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
	processContext := context.(ProcessMachineContext)

	err := processContext.Process.Stop(processContext.Process.StopSignal)
	if err != nil {
		return machine.NoopEvent, &ErrProcessAction{
			ID:  processContext.Process.ID,
			Err: err,
		}
	}

	return machine.NoopEvent, nil
}
