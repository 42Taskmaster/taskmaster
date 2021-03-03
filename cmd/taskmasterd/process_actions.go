package main

import (
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
		log.Printf(
			"Error starting process '%s' of program '%s': %s",
			config.Name,
			serializedProcess.ID,
			err.Error(),
		)

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

		_, err := stateMachine.Send(ProcessEventStopped)
		if err != nil {
			log.Printf("expected no error to be returned but got %v\n", err)
		}
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

	serializedProcess := process.Serialize()

	if processContext.Starttries >= config.Startretries {
		log.Printf(
			"Fatal: could not start process '%s' of program '%s' (%d tries)",
			serializedProcess.ID,
			config.Name,
			processContext.Starttries,
		)
		return ProcessEventFatal, nil
	}

	log.Printf(
		"Trying to restart process '%s' of program '%s'...",
		serializedProcess.ID,
		config.Name,
	)

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

	serializedProcess := process.Serialize()

	switch config.Autorestart {
	case AutorestartOn:
		log.Printf(
			"Trying to restart process '%s' of program '%s'...",
			serializedProcess.ID,
			config.Name,
		)
		return ProcessEventStart, nil
	case AutorestartUnexpected:
		exitcode := process.GetCmd().ProcessState.ExitCode()
		for _, allowedExitcode := range config.Exitcodes {
			if exitcode == allowedExitcode {
				return machine.NoopEvent, nil
			}
		}

		log.Printf(
			"Trying to restart process '%s' of program '%s'...",
			serializedProcess.ID,
			config.Name,
		)
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
