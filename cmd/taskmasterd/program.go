package main

import (
	"errors"
	"io"
	"log"
	"os"
	"sort"
	"syscall"

	"github.com/VisorRaptors/taskmaster/machine"
)

type Program struct {
	ProgramManager *ProgramManager
	Processes      ProcessMap
	Config         ProgramConfig
	Cache          ProgramCache
}

type ProgramCache struct {
	Env            []string
	Stdout, Stderr io.WriteCloser
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
	program.Processes.Range(func(key string, process *Process) bool {
		_, err := process.Machine.Send(ProcessEventStart)
		if err != nil {
			log.Println(err)
		}
		if errors.Is(err, syscall.ENOENT) {
			log.Println("can not launch process as the command does not exist")
		}
		return true
	})
}

func (program *Program) Stop() {
	program.Processes.Range(func(key string, process *Process) bool {
		_, err := process.Machine.Send(ProcessEventStop)
		if err != nil {
			log.Println(err)
		}
		return true
	})
}

func (program *Program) Restart() {
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

	program.Processes.Range(func(key string, process *Process) bool {
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

		return true
	})

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
	if stopped == program.Processes.Length() {
		return ProcessStateStopped
	}
	if exited == program.Processes.Length() {
		return ProcessStateExited
	}
	if running > 0 {
		return ProcessStateRunning
	}
	return ProcessStateUnknown
}

func (program *Program) GetProcessById(id string) *Process {
	process, ok := program.Processes.Get(id)
	if !ok {
		return nil
	}
	return process
}

func (program *Program) GetSortedProcesses() []*Process {
	processIds := []string{}

	program.Processes.Range(func(key string, process *Process) bool {
		processIds = append(processIds, process.ID)
		return true
	})

	sort.Strings(processIds)

	processes := []*Process{}
	for _, id := range processIds {
		processes = append(processes, program.GetProcessById(id))
	}

	return processes
}

func openStdFile(path string) (io.WriteCloser, error) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	return file, err
}

func (program *Program) SetProgramStds(stdout string, stderr string) {
	if program.Config.Stdout != stdout {
		program.Cache.Stdout.Close()

		if len(stdout) > 0 {
			stdoutFile, err := openStdFile(stdout)
			if err != nil {
				log.Fatal(err)
			}
			stdoutWriter := stdoutFile
			program.Cache.Stdout = stdoutWriter
		}
	}

	if program.Config.Stderr != stderr {
		program.Cache.Stderr.Close()

		if len(stderr) > 0 {
			stderrFile, err := openStdFile(stderr)
			if err != nil {
				log.Fatal(err)
			}
			stderrWriter := stderrFile
			program.Cache.Stderr = stderrWriter
		}
	}
}
