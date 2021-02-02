package main

import (
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/VisorRaptors/taskmaster/machine"
	"github.com/VisorRaptors/taskmaster/parser"
)

func ProcessStartAction(context machine.Context) (machine.EventType, error) {
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
		process.Context,
		parsedCommand.Cmd,
		parsedCommand.Args...,
	)

	cmd.Env = config.CreateCmdEnvironment()
	cmd.Stdin = nil

	stdout, err := config.CreateCmdStdout(process.ID)
	if err != nil {
		return ProcessEventFatal, nil
	}
	cmd.Stdout = stdout

	stderr, err := config.CreateCmdStderr(process.ID)
	if err != nil {
		return ProcessEventFatal, nil
	}
	cmd.Stderr = stderr

	cmd.Dir = config.Workingdir

	process.Cmd = cmd

	// TODO: set umask
	if err := process.Cmd.Start(); err != nil {
		// reset umask
		log.Printf("Error starting process '%s' of program '%s': %s", config.Name, process.ID, err.Error())
		return ProcessEventFatal, &ErrProcessAction{
			ID: process.ID,
			Err: &ErrProcessStarting{
				Err: err,
			},
		}
	}
	// TODO: reset umask

	process.DeadCh = make(chan struct{})

	go func() {
		select {
		case <-time.After(time.Duration(config.Starttime) * time.Second):
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
			log.Printf("Process '%s' of program '%s' has exited with code %d", process.ID, config.Name, process.Cmd.ProcessState.ExitCode())
			event = ProcessEventExit
		}

		_, err := process.Machine.Send(event)
		if err != nil {
			log.Printf("expected no error to be returned but got %v\n", err)
		}
	}()

	return machine.NoopEvent, nil
}

func ProcessStopAction(context machine.Context) (machine.EventType, error) {
	var (
		processContext = context.(*ProcessMachineContext)
		process        = processContext.Process
	)

	config, err := process.GetConfig()
	if err != nil {
		return machine.NoopEvent, err
	}

	err = process.Cmd.Process.Signal(config.Stopsignal.ToOsSignal())
	if err != nil {
		return machine.NoopEvent, &ErrProcessAction{
			ID:  processContext.Process.ID,
			Err: err,
		}
	}

	go func() {
		select {
		case <-time.After(time.Duration(config.Stoptime) * time.Second):
			process.Kill()
		case <-process.DeadCh:
		}
	}()

	return machine.NoopEvent, nil
}

func ProcessBackoffAction(context machine.Context) (machine.EventType, error) {
	processContext := context.(*ProcessMachineContext)
	process := processContext.Process

	config, err := process.GetConfig()
	if err != nil {
		return machine.NoopEvent, err
	}

	switch config.Autorestart {
	case AutorestartOn:
		processContext.Starttries++
		if processContext.Starttries == config.Startretries {
			log.Printf("Process '%s' of program '%s' has exceeded start tries", process.ID, config.Name)
			return ProcessEventFatal, nil
		}
		log.Printf("Trying to restart process '%s' of program '%s'...", process.ID, config.Name)
		return ProcessEventStart, nil
	case AutorestartUnexpected:
		exitcode := process.Cmd.ProcessState.ExitCode()
		for _, allowedExitcode := range config.Exitcodes {
			if exitcode == allowedExitcode {
				return machine.NoopEvent, nil
			}
		}
		processContext.Starttries++
		if processContext.Starttries == config.Startretries {
			log.Printf("Process '%s' of program '%s' has exceeded start tries", process.ID, config.Name)
			return ProcessEventFatal, nil
		}
		log.Printf("Trying to restart process '%s' of program '%s'...", process.ID, config.Name)
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
