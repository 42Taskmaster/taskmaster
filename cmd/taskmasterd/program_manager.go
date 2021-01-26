package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"syscall"
)

type ProgramManager struct {
	Programs        ProgramMap
	ProgramTaskChan chan ProgramTask
}

func NewProgramManager() *ProgramManager {
	var programManager ProgramManager
	programManager.Init()
	return &programManager
}

func (programManager *ProgramManager) Init() {
	programManager.ProgramTaskChan = make(chan ProgramTask)

	go func() {
		for programTask := range programManager.ProgramTaskChan {
			switch programTask.Action {
			case ProgramTaskActionStart:
				programTask.Program.Start()
			case ProgramTaskActionStop:
				programTask.Program.Stop()
			case ProgramTaskActionRestart:
				programTask.Program.Restart()
			case ProgramTaskActionKill:
				programTask.Process.Cmd.Process.Signal(syscall.SIGKILL)
			}
		}
	}()
}

func (programManager *ProgramManager) GetProgramByName(name string) *Program {
	var ret *Program = nil
	programManager.Programs.Range(func(key string, program *Program) bool {
		if key == name {
			ret = program
			return false
		}
		return true
	})
	return ret
}

func (programManager *ProgramManager) StartPrograms() {
	programManager.Programs.Range(func(key string, program *Program) bool {
		programManager.ProgramTaskChan <- ProgramTask{
			Action:  ProgramTaskActionStart,
			Program: program,
		}
		return true
	})
}

func (programManager *ProgramManager) StopPrograms() {
	programManager.Programs.Range(func(key string, program *Program) bool {
		programManager.ProgramTaskChan <- ProgramTask{
			Action:  ProgramTaskActionStop,
			Program: program,
		}
		return true
	})
}

func (programManager *ProgramManager) StartProgramByName(name string) error {
	program := programManager.GetProgramByName(name)
	if program == nil {
		return fmt.Errorf("Program not found: \"%s\"", name)
	}
	programManager.ProgramTaskChan <- ProgramTask{
		Action:  ProgramTaskActionStart,
		Program: program,
	}
	return nil
}

func (programManager *ProgramManager) StopProgramByName(name string) error {
	program := programManager.GetProgramByName(name)
	if program == nil {
		return fmt.Errorf("Program not found: \"%s\"", name)
	}
	programManager.ProgramTaskChan <- ProgramTask{
		Action:  ProgramTaskActionStop,
		Program: program,
	}
	return nil
}

func (programManager *ProgramManager) RestartProgramByName(name string) error {
	program := programManager.GetProgramByName(name)
	if program == nil {
		return fmt.Errorf("Program not found: \"%s\"", name)
	}
	return nil
}

func (programManager *ProgramManager) GetSortedPrograms() []*Program {
	programsKeys := []string{}

	programManager.Programs.Range(func(key string, program *Program) bool {
		programsKeys = append(programsKeys, key)
		return true
	})

	sort.Strings(programsKeys)

	programs := []*Program{}
	for _, key := range programsKeys {
		program := programManager.GetProgramByName(key)
		if program != nil {
			programs = append(programs, programManager.GetProgramByName(key))
		} else {
			log.Panic("GetSortedPrograms(): program is nil")
		}
	}

	return programs
}

func ProgramListContainsProgram(programList []string, programToFind string) bool {
	for _, program := range programList {
		if program == programToFind {
			return true
		}
	}
	return false
}

func (programManager *ProgramManager) LoadConfiguration(programsConfiguration ProgramsConfiguration) {
	configPrograms := make([]string, 0, len(programsConfiguration))
	for name, programConfiguration := range programsConfiguration {
		configPrograms = append(configPrograms, name)
		program := programManager.GetProgramByName(name)
		if program != nil {
			programManager.UpdateProgram(program, programConfiguration)
		} else {
			programManager.AddProgram(programConfiguration)
		}
	}

	programManager.Programs.Range(func(name string, program *Program) bool {
		if !ProgramListContainsProgram(configPrograms, name) {
			programManager.RemoveProgramByName(name)
		}
		return true
	})
}

func (programManager *ProgramManager) UpdateProgram(program *Program, newProgramConfig ProgramConfig) {
	program.Config = newProgramConfig

	program.SetProgramStds(program.Config.Stdout, program.Config.Stderr)

	numProcesses := program.Processes.Length()
	if numProcesses > program.Config.Numprocs {
		for index := numProcesses; index > program.Config.Numprocs; index-- {
			process, _ := program.Processes.Get(strconv.Itoa(index))
			process.Machine.Send(ProcessEventStop)

			go func() {
				<-process.DeadCh

				program.Processes.Delete(process.ID)
			}()
		}
	} else if numProcesses < program.Config.Numprocs {
		for index := numProcesses; index <= program.Config.Numprocs; index++ {
			id := strconv.Itoa(index)
			program.Processes.Add(id, NewProcess(id, program))
			if program.Config.Autostart {
				programManager.ProgramTaskChan <- ProgramTask{
					Action:  ProgramTaskActionStart,
					Program: program,
				}
			}
		}
	}
}

func (programManager *ProgramManager) AddProgram(programConfig ProgramConfig) {
	env := os.Environ()
	for name, value := range programConfig.Env {
		concatenatedKeyValue := name + "=" + value

		env = append(env, concatenatedKeyValue)
	}

	program := &Program{
		ProgramManager: programManager,
		Config:         programConfig,
		Cache: ProgramCache{
			Env: env,
		},
	}

	program.SetProgramStds(programConfig.Stdout, programConfig.Stderr)

	programManager.Programs.Add(program.Config.Name, program)

	for index := 1; index <= programConfig.Numprocs; index++ {
		id := strconv.Itoa(index)
		process := NewProcess(id, program)
		program.Processes.Add(id, process)
	}

	if program.Config.Autostart {
		programManager.ProgramTaskChan <- ProgramTask{
			Action:  ProgramTaskActionStart,
			Program: program,
		}
	}
}

func (programManager *ProgramManager) RemoveProgramByName(name string) {
	program := programManager.GetProgramByName(name)

	log.Printf("Remove Program %s", name)
	programManager.ProgramTaskChan <- ProgramTask{
		Action:  ProgramTaskActionStop,
		Program: program,
	}

	go func() {
		log.Printf("Waiting for processes' deadCh")
		program.Processes.Range(func(key string, process *Process) bool {
			<-process.DeadCh
			return true
		})

		log.Printf("Sending remove action")
		programManager.Programs.Delete(program.Config.Name)
	}()
}
