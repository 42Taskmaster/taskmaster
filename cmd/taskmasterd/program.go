package main

import (
	"bufio"
	"errors"
	"io"
	"log"
	"os"
	"syscall"
)

type Programs map[string]*Program

type Program struct {
	Processes []*Process
	Config    ProgramConfig
}

type ProgramState string

const (
	ProgramStateStarting ProgramState = "STARTING"
	ProgramStateBackoff  ProgramState = "BACKOFF"
	ProgramStateRunning  ProgramState = "RUNNING"
	ProgramStateStopping ProgramState = "STOPPING"
	ProgramStateStopped  ProgramState = "STOPPED"
	ProgramStateExited   ProgramState = "EXITED"
	ProgramStateFatal    ProgramState = "FATAL"
	ProgramStateUnknown  ProgramState = "UNKNOWN"
)

func (program *Program) Start() {
	log.Println("start is called")
	for _, process := range program.Processes {
		_, err := process.Machine.Send(ProcessEventStart)
		if err != nil {
			log.Println(err)
		}
		if errors.Is(err, syscall.ENOENT) {
			log.Println("can not launch process as the command does not exist")
		}
	}
}

func (program *Program) Stop() {
	for _, process := range program.Processes {
		_, err := process.Machine.Send(ProcessEventStop)
		if err != nil {
			log.Println(err)
		}
	}
}

func (program *Program) Restart() {
	program.Stop()
	program.Start()
}

func (program *Program) GetState() string {
	return "OK"
}

func programParse(config ProgramConfig) *Program {
	var stdoutWriter, stderrWriter io.Writer

	if len(config.Stdout) > 0 {
		stdoutFile, err := os.OpenFile(config.Stdout, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		stdoutWriter = bufio.NewWriter(stdoutFile)
	}
	if len(config.Stderr) > 0 {
		stderrFile, err := os.OpenFile(config.Stderr, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		stderrWriter = bufio.NewWriter(stderrFile)
	}

	env := os.Environ()
	for name, value := range config.Env {
		concatenatedKeyValue := name + "=" + value

		env = append(env, concatenatedKeyValue)
	}

	processes := []*Process{}

	for index := 0; index < config.Numprocs; index++ {
		process := NewProcess(NewProcessArgs{
			ID:         index,
			Cmd:        config.Cmd,
			Env:        env,
			Stdout:     stdoutWriter,
			Stderr:     stderrWriter,
			StopSignal: config.Stopsignal,
		})

		processes = append(processes, process)
	}

	return &Program{
		Processes: processes,
		Config:    config,
	}
}

func programsParse(config ProgramsConfiguration) Programs {
	parsedPrograms := make(Programs)

	for name, program := range config {
		parsedPrograms[name] = programParse(program)
	}

	return parsedPrograms
}
