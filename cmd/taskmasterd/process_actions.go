package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/42Taskmaster/taskmaster/machine"
	"github.com/42Taskmaster/taskmaster/parser"
)

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

type ErrStrartretriesReached struct {
	Startretries int
}

func (err *ErrStrartretriesReached) Error() string {
	return fmt.Sprintf("reached maximum startries: %d", err.Startretries)
}

func ProcessStartAction(stateMachine *machine.Machine, context machine.Context) (machine.EventType, error) {
	var (
		processContext = context.(*ProcessMachineContext)
		process        = processContext.Process
	)

	config, err := process.GetConfig()
	if err != nil {
		return machine.NoopEvent, err
	}

	expandedCommand := os.ExpandEnv(config.Cmd)
	parsedCommand, err := parser.ParseCommand(expandedCommand)
	if err != nil {
		return ProcessEventStopped, nil
	}

	cmd := exec.CommandContext(
		process.GetContext(),
		parsedCommand.Cmd,
		parsedCommand.Args...,
	)

	cmd.Env = config.CreateCmdEnvironment()
	cmd.Stdin = nil

	serializedProcess := process.Serialize()

	stdout, err := config.CreateCmdStdout(serializedProcess.ID)
	if err != nil {
		return ProcessEventStopped, nil
	}
	cmd.Stdout = stdout

	stderr, err := config.CreateCmdStderr(serializedProcess.ID)
	if err != nil {
		return ProcessEventStopped, nil
	}
	cmd.Stderr = stderr

	process.SetStdoutStderrCloser(stdout, stderr)

	cmd.Dir = config.Workingdir

	process.SetCmd(cmd)

	deadCh := process.CreateNewDeadChannel()

	SetUmask(config.Umask)
	if err := cmd.Start(); err != nil {
		processContext.LastError = err

		ResetUmask()

		close(deadCh)

		return ProcessEventStopped, nil
	}
	ResetUmask()

	process.StartChronometer()

	go func() {
		select {
		case <-time.After(time.Duration(config.Starttime) * time.Second):
			stateMachine.Send(ProcessEventStarted)
		case <-deadCh:
			return
		}
	}()

	go func() {
		// We must close the channel after the state machine has reached another
		// state. Without that we encounter race conditions on the restart function.
		defer close(deadCh)

		cmd.Wait()

		process.StopChronometer()

		stateMachine.Send(ProcessEventStopped)
	}()

	return machine.NoopEvent, nil
}

func ProcessStopAction(stateMachine *machine.Machine, context machine.Context) (machine.EventType, error) {
	var (
		processContext = context.(*ProcessMachineContext)
		process        = processContext.Process
	)

	config, err := process.GetConfig()
	if err != nil {
		return machine.NoopEvent, err
	}

	serializedProcess := process.Serialize()

	err = process.GetCmd().Process.Signal(config.Stopsignal.ToOsSignal())
	if err != nil {
		return machine.NoopEvent, &ErrProcessAction{
			ID:  serializedProcess.ID,
			Err: err,
		}
	}

	go func() {
		select {
		case <-time.After(time.Duration(config.Stoptime) * time.Second):
			process.Kill()
		case <-process.GetDeadChannel():
		}
	}()

	return machine.NoopEvent, nil
}

func ProcessBackoffAction(stateMachine *machine.Machine, context machine.Context) (machine.EventType, error) {
	processContext := context.(*ProcessMachineContext)
	process := processContext.Process

	config, err := process.GetConfig()
	if err != nil {
		return machine.NoopEvent, err
	}

	if processContext.Starttries >= config.Startretries {
		processContext.LastError = &ErrStrartretriesReached{
			Startretries: processContext.Starttries,
		}

		return ProcessEventFatal, nil
	}

	processContext.Starttries++

	return ProcessEventStart, nil
}

func ProcessExitedAction(stateMachine *machine.Machine, context machine.Context) (machine.EventType, error) {
	processContext := context.(*ProcessMachineContext)
	process := processContext.Process

	config, err := process.GetConfig()
	if err != nil {
		return machine.NoopEvent, err
	}

	switch config.Autorestart {
	case AutorestartOn:
		return ProcessEventStart, nil

	case AutorestartUnexpected:
		exitcode := process.GetCmd().ProcessState.ExitCode()
		for _, allowedExitcode := range config.Exitcodes {
			if exitcode == allowedExitcode {
				return machine.NoopEvent, nil
			}
		}

		return ProcessEventStart, nil

	default:
		return machine.NoopEvent, nil
	}
}

func ProcessResetStarttriesAction(stateMachine *machine.Machine, context machine.Context) (machine.EventType, error) {
	processContext := context.(*ProcessMachineContext)
	processContext.Starttries = 0

	return machine.NoopEvent, nil
}

func PrintCurrentStateAction(stateMachine *machine.Machine, context machine.Context) (machine.EventType, error) {
	var (
		processContext = context.(*ProcessMachineContext)
		process        = processContext.Process
	)

	defer func() {
		processContext.LastError = nil
	}()

	sentenceForState := map[machine.StateType]string{
		ProcessStateStarting: "is starting",
		ProcessStateBackoff:  "is backing off",
		ProcessStateRunning:  "is running",
		ProcessStateStopping: "is stopping",
		ProcessStateStopped:  "is stopped",
		ProcessStateExited:   "has exited",
		ProcessStateFatal:    "has fataly exited",
	}

	config, err := process.GetConfig()
	if err != nil {
		return machine.NoopEvent, err
	}

	serializedProcess := process.Serialize()
	currentState := stateMachine.UnsafeCurrent()

	if err := processContext.LastError; err != nil {
		log.Printf(
			"Process '%s' of program '%s' %s (%s)\n",
			serializedProcess.ID,
			config.Name,
			sentenceForState[currentState],
			err,
		)
	} else {
		log.Printf(
			"Process '%s' of program '%s' %s\n",
			serializedProcess.ID,
			config.Name,
			sentenceForState[currentState],
		)
	}

	return machine.NoopEvent, nil
}
