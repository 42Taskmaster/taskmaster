package main

import "github.com/VisorRaptors/taskmaster/machine"

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

func NewProcessMachine(process *Process) *machine.Machine {
	machine := &machine.Machine{
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

	machine.Init()

	return machine
}
