package machine_test

import (
	"errors"
	"testing"

	"github.com/42Taskmaster/taskmaster/machine"
)

func TestOnOffMachine(t *testing.T) {
	const (
		OnState  machine.StateType = "on"
		OffState machine.StateType = "off"

		OnEvent  machine.EventType = "on"
		OffEvent machine.EventType = "off"
	)

	onOffMachine := machine.Machine{
		Initial: OffState,

		StateNodes: machine.StateNodes{
			OnState: machine.StateNode{
				On: machine.Events{
					OffEvent: OffState,
				},
			},
			OffState: machine.StateNode{
				On: machine.Events{
					OnEvent: OnState,
				},
			},
		},
	}
	onOffMachine.Init()

	if currentState := onOffMachine.Current(); currentState != OffState {
		t.Fatalf(
			"machine is in incorrect state %v; expected %v",
			currentState,
			OffState,
		)
	}

	nextState, err := onOffMachine.Send(OnEvent)
	if err != nil {
		t.Fatalf(
			"transition returned an unexpected error %v",
			err,
		)
	}
	if nextState != OnState {
		t.Fatalf(
			"machine is in incorrect state %v; expected %v",
			nextState,
			OnState,
		)
	}
	if currentState := onOffMachine.Current(); currentState != OnState {
		t.Fatalf(
			"machine is in incorrect state %v; expected %v",
			currentState,
			OnState,
		)
	}

	nextState, err = onOffMachine.Send(OnEvent)
	if err == nil {
		t.Error("returned no error when we expected one")
	}
	if !errors.Is(err, machine.ErrInvalidTransitionNotImplemented) {
		t.Fatalf(
			"returned error is not caused by what we expected %v; expected %v",
			err,
			machine.ErrInvalidTransitionNotImplemented,
		)
	}

	nextState, err = onOffMachine.Send(OffEvent)
	if err != nil {
		t.Fatalf(
			"transition returned an unexpected error %v",
			err,
		)
	}
	if nextState != OffState {
		t.Fatalf(
			"machine is in incorrect state %v; expected %v",
			nextState,
			OffState,
		)
	}
	if currentState := onOffMachine.Current(); currentState != OffState {
		t.Fatalf(
			"machine is in incorrect state %v; expected %v",
			currentState,
			OffState,
		)
	}
}
