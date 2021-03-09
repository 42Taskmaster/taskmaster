package main

import (
	"github.com/42Taskmaster/taskmaster/machine"
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
	ProcessEventStart   machine.EventType = "start"
	ProcessEventStarted machine.EventType = "started"
	ProcessEventStop    machine.EventType = "stop"
	ProcessEventStopped machine.EventType = "stopped"
	ProcessEventFatal   machine.EventType = "fatal"
)

type ProcessMachineContext struct {
	Process    Processer
	Starttries int
	LastError  error
}

func NewProcessMachine(process *Process) *machine.Machine {
	machine := &machine.Machine{
		Context: &ProcessMachineContext{
			Process:    process,
			Starttries: 0,
		},

		Initial: ProcessStateStopped,

		StateNodes: machine.StateNodes{
			ProcessStateStopped: machine.StateNode{
				Actions: []machine.Action{
					PrintCurrentStateAction,
				},

				On: machine.Events{
					ProcessEventStart: ProcessStateStarting,
				},
			},

			ProcessStateStarting: machine.StateNode{
				Actions: []machine.Action{
					PrintCurrentStateAction,
					ProcessStartAction,
				},

				On: machine.Events{
					ProcessEventStarted: ProcessStateRunning,
					ProcessEventStop:    ProcessStateStopping,
					ProcessEventStopped: ProcessStateBackoff,
				},
			},

			ProcessStateBackoff: machine.StateNode{
				Actions: []machine.Action{
					PrintCurrentStateAction,
					ProcessBackoffAction,
				},

				On: machine.Events{
					ProcessEventStart: ProcessStateStarting,
					ProcessEventFatal: ProcessStateFatal,
				},
			},

			ProcessStateRunning: machine.StateNode{
				Actions: []machine.Action{
					PrintCurrentStateAction,
					ProcessResetStarttriesAction,
				},

				On: machine.Events{
					ProcessEventStop:    ProcessStateStopping,
					ProcessEventStopped: ProcessStateExited,
				},
			},

			ProcessStateStopping: machine.StateNode{
				Actions: []machine.Action{
					PrintCurrentStateAction,
					ProcessStopAction,
					ProcessResetStarttriesAction,
				},

				On: machine.Events{
					ProcessEventStopped: ProcessStateStopped,
				},
			},

			ProcessStateExited: machine.StateNode{
				Actions: []machine.Action{
					PrintCurrentStateAction,
					ProcessExitedAction,
				},

				On: machine.Events{
					ProcessEventStart: ProcessStateStarting,
				},
			},

			ProcessStateFatal: machine.StateNode{
				Actions: []machine.Action{
					PrintCurrentStateAction,
					ProcessResetStarttriesAction,
				},

				On: machine.Events{
					ProcessEventStart: ProcessStateStarting,
				},
			},
		},
	}

	machine.Init()

	return machine
}
