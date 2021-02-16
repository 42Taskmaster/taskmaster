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
		return ProcessEventFatal, err
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
		return ProcessEventFatal, nil
	}
	cmd.Stdout = stdout

	stderr, err := config.CreateCmdStderr(serializedProcess.ID)
	if err != nil {
		return ProcessEventFatal, nil
	}
	cmd.Stderr = stderr

	process.SetStdoutStderrCloser(stdout.Close, stderr.Close)

	cmd.Dir = config.Workingdir

	process.SetCmd(cmd)

	SetUmask(config.Umask)
	if err := cmd.Start(); err != nil {
		log.Printf(
			"Error starting process '%s' of program '%s': %s",
			config.Name,
			serializedProcess.ID,
			err.Error(),
		)

		ResetUmask()

		return ProcessEventFatal, &ErrProcessAction{
			ID: serializedProcess.ID,
			Err: &ErrProcessStarting{
				Err: err,
			},
		}
	}
	ResetUmask()

	process.StartChronometer()

	deadCh := process.CreateNewDeadChannel()

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

		if err := process.CloseFileDescriptors(); err != nil {
			log.Printf(
				"error while closing opened file descriptors of stdout and stderr: %v\n",
				err,
			)
		}

		event := ProcessEventStopped
		if cmd.ProcessState.Exited() {
			log.Printf(
				"Process '%s' of program '%s' has exited with code %d",
				serializedProcess.ID,
				config.Name,
				cmd.ProcessState.ExitCode(),
			)
			event = ProcessEventExit
		}

		_, err := stateMachine.Send(event)
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

	switch config.Autorestart {
	case AutorestartOn:
		processContext.Starttries++
		if processContext.Starttries == config.Startretries {
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
		return ProcessEventStart, nil
	case AutorestartUnexpected:
		exitcode := process.GetCmd().ProcessState.ExitCode()
		for _, allowedExitcode := range config.Exitcodes {
			if exitcode == allowedExitcode {
				return machine.NoopEvent, nil
			}
		}
		processContext.Starttries++
		if processContext.Starttries == config.Startretries {
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
		return ProcessEventStart, nil
	default:
		return machine.NoopEvent, nil
	}
}

func ProcessRunningAction(stateMachine *machine.Machine, context machine.Context) (machine.EventType, error) {
	processContext := context.(*ProcessMachineContext)
	processContext.Starttries = 0

	return machine.NoopEvent, nil
}
