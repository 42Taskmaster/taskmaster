package main

import (
	"bufio"
	"errors"
	"io"
	"log"
	"os"
	"strconv"
	"syscall"
)

type Programs map[string]*Program

type Program struct {
	ProgramManager *ProgramManager
	Processes      map[string]*Process
	Config         ProgramConfig
	Cache          ProgramCache
}

type ProgramCache struct {
	Env            []string
	Stdout, Stderr io.Writer
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

func (program *Program) GetProcessById(id string) *Process {
	process, ok := program.Processes[id]
	if !ok {
		return nil
	}
	return process
}

func programParse(programManager *ProgramManager, config ProgramConfig) *Program {
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

	program := &Program{
		ProgramManager: programManager,
		Processes:      make(map[string]*Process),
		Config:         config,
		Cache: ProgramCache{
			Env:    env,
			Stdout: stdoutWriter,
			Stderr: stderrWriter,
		}}

	for index := 1; index <= config.Numprocs; index++ {
		id := strconv.Itoa(index)
		program.Processes[id] = NewProcess(id, program)
	}

	return program
}

func programsParse(programManager *ProgramManager, config ProgramsConfiguration) Programs {
	parsedPrograms := make(Programs)

	for name, program := range config {
		parsedPrograms[name] = programParse(programManager, program)
	}

	return parsedPrograms
}
