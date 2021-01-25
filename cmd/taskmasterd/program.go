package main

import (
	"bufio"
	"errors"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"syscall"

	"github.com/VisorRaptors/taskmaster/machine"
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
	log.Printf("Starting program '%s'", program.Config.Name)
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
	log.Printf("Stopping program '%s'", program.Config.Name)
	for _, process := range program.Processes {
		_, err := process.Machine.Send(ProcessEventStop)
		if err != nil {
			log.Println(err)
		}
	}
}

func (program *Program) Restart() {
	log.Printf("Restarting program '%s'", program.Config.Name)
	program.Stop()
	program.Start()
}

func (program *Program) GetState() machine.StateType {
	starting := 0
	running := 0
	backoff := 0
	stopping := 0
	stopped := 0
	exited := 0
	fatal := 0
	unknown := 0

	for _, process := range program.Processes {
		switch process.Machine.Current() {
		case ProcessStateStarting:
			starting++
		case ProcessStateRunning:
			running++
		case ProcessStateBackoff:
			backoff++
		case ProcessStateStopping:
			stopping++
		case ProcessStateStopped:
			stopped++
		case ProcessStateExited:
			exited++
		case ProcessStateFatal:
			fatal++
		default:
			unknown++
		}
	}

	if unknown > 0 {
		return ProcessStateUnknown
	}
	if fatal > 0 {
		return ProcessStateFatal
	}
	if starting > 0 {
		return ProcessStateStarting
	}
	if stopping > 0 {
		return ProcessStateStopping
	}
	if backoff > 0 {
		return ProcessStateBackoff
	}
	if stopped == len(program.Processes) {
		return ProcessStateStopped
	}
	if exited == len(program.Processes) {
		return ProcessStateExited
	}
	if running > 0 {
		return ProcessStateRunning
	}
	return ProcessStateUnknown
}

func (program *Program) GetProcessById(id string) *Process {
	process, ok := program.Processes[id]
	if !ok {
		return nil
	}
	return process
}

func (program *Program) GetSortedProcesses() []*Process {
	processIds := []string{}

	for id, _ := range program.Processes {
		processIds = append(processIds, id)
	}

	sort.Strings(processIds)

	processes := []*Process{}
	for _, id := range processIds {
		processes = append(processes, program.GetProcessById(id))
	}

	return processes
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

func programsParse(programManager *ProgramManager, configs ProgramsConfiguration) Programs {
	parsedPrograms := make(Programs)

	for name, config := range configs {
		program := programParse(programManager, config)
		parsedPrograms[name] = program
		if program.Config.Autostart {
			program.Start()
		}
	}

	return parsedPrograms
}
