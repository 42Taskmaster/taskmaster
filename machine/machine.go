package machine

import (
	"errors"
	"fmt"
	"sync"
)

// ErrUnexpectedBehavior represents an invalid operation we could not perform nor identify.
var ErrUnexpectedBehavior = errors.New("unexpected behavior")

// InvalidTransitionReason represents the reason why a transition could not be performed.
type InvalidTransitionReason string

// All reasons why a transition could not be performed.
const (
	InvalidTransitionInvalidCurrentState     InvalidTransitionReason = "current state is unexpected"
	InvalidTransitionFinalState              InvalidTransitionReason = "final state reached"
	InvalidTransitionNotImplemented          InvalidTransitionReason = "transition not implemented"
	InvalidTransitionNextStateNotImplemented InvalidTransitionReason = "next state is not implemented"
)

// ErrInvalidTransition means a transition could not be performed, holding the reason why.
type ErrInvalidTransition struct {
	Reason InvalidTransitionReason
}

func (err ErrInvalidTransition) Error() string {
	return "invalid transition: " + string(err.Reason)
}

// ErrTransition represents an error that occured while transitioning to a new state.
type ErrTransition struct {
	Event EventType
	Err   error
}

func (err *ErrTransition) Unwrap() error {
	return err.Err
}

func (err *ErrTransition) Error() string {
	return "could not transition to: " + string(err.Event) + ": " + err.Err.Error()
}

// StateType represents a state described in the state machine.
type StateType string

// Basic states.
const (
	IdleState StateType = ""
	NoopState StateType = "noop"
)

// EventType represents an event transitioning from one state to another one.
type EventType string

// Basic events.
const (
	NoopEvent EventType = "noop"
)

// Context holds data passed to actions functions.
// It can contain what user wants.
type Context interface{}

// An Action is performed when the machine is transitioning to the node where it's defined.
// It takes machine context and returns an event to send to the machine itself, or NoopEvent.
type Action func(Context) EventType

// Events map holds events to listen with the state to transition to when triggered.
type Events map[EventType]StateType

// A StateNode is a node of the state machine.
// It has actions and events to listen to.
//
// No actions can be specified.
// When no events are specified, the state node is of *final* type, which means once reached, the state
// machine can not be transitioned anymore.
type StateNode struct {
	Actions []Action
	On      Events
}

// A StateNodes holds all state nodes of a machine.
type StateNodes map[StateType]StateNode

// A Machine is a simple state machine.
type Machine struct {
	Context Context

	Previous StateType
	Current  StateType

	StateNodes StateNodes

	lock sync.Mutex
}

func (machine *Machine) getNextState(event EventType) (StateType, error) {
	currentState, ok := machine.StateNodes[machine.Current]
	if !ok {
		return NoopState, &ErrInvalidTransition{
			Reason: InvalidTransitionInvalidCurrentState,
		}
	}

	if currentState.On == nil {
		return NoopState, &ErrInvalidTransition{
			Reason: InvalidTransitionFinalState,
		}
	}

	fmt.Printf("currentState, event = %+v, %+v\n", currentState, event)

	nextState, ok := currentState.On[event]
	if !ok {
		return NoopState, &ErrInvalidTransition{
			Reason: InvalidTransitionNotImplemented,
		}
	}

	return nextState, nil
}

func (machine *Machine) executeActions(stateNode StateNode) EventType {
	for _, actionToRun := range stateNode.Actions {
		stateToReach := actionToRun(machine.Context)
		if stateToReach == NoopEvent {
			continue
		}

		return stateToReach
	}

	return NoopEvent
}

// Send an event to the state machine.
// Returns the new state and an error if one occured, or nil.
func (machine *Machine) Send(event EventType) (StateType, error) {
	machine.lock.Lock()
	defer machine.lock.Unlock()

	for {
		nextState, err := machine.getNextState(event)
		if err != nil {
			return NoopState, &ErrTransition{
				Event: event,
				Err:   err,
			}
		}

		nextStateNode, ok := machine.StateNodes[nextState]
		if !ok {
			return NoopState, &ErrTransition{
				Event: event,
				Err: &ErrInvalidTransition{
					Reason: InvalidTransitionNextStateNotImplemented,
				},
			}
		}

		machine.Previous = machine.Current
		machine.Current = nextState

		if len(nextStateNode.Actions) == 0 {
			return nextState, nil
		}

		eventToSend := machine.executeActions(nextStateNode)
		if eventToSend == NoopEvent {
			return nextState, nil
		}

		event = eventToSend
	}
}
