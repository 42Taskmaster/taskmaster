package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"syscall"
)

type ProgramManager struct {
	Programs        Programs
	ProgramTaskChan chan ProgramTask
}

func NewProgramManager() *ProgramManager {
	var programManager ProgramManager
	programManager.Init()
	return &programManager
}

func (programManager *ProgramManager) Init() {
	programManager.Programs = make(Programs)
	programManager.ProgramTaskChan = make(chan ProgramTask)

	go func() {
		for programTask := range programManager.ProgramTaskChan {
			if programTask.Action == ProgramTaskActionStart {
				programTask.Program.Start()
			} else if programTask.Action == ProgramTaskActionStop {
				programTask.Program.Stop()
			} else if programTask.Action == ProgramTaskActionRestart {
				programTask.Program.Restart()
			} else if programTask.Action == ProgramTaskActionKill && programTask.ProcessID != "" {
				process := programTask.Program.GetProcessById(programTask.ProcessID)
				if process != nil {
					process.Cmd.Process.Signal(syscall.SIGKILL)
				}
			} else if programTask.Action == ProgramTaskActionRemove {
				delete(programManager.Programs, programTask.Program.Config.Name)
			}
		}
	}()
}

func (programManager *ProgramManager) GetProgramByName(name string) *Program {
	for _, program := range programManager.Programs {
		if program.Config.Name == name {
			return program
		}
	}
	return nil
}

func (programManager *ProgramManager) StartPrograms() {
	for _, program := range programManager.Programs {
		programManager.ProgramTaskChan <- ProgramTask{
			Action:  ProgramTaskActionStart,
			Program: program,
		}
	}
}

func (programManager *ProgramManager) StopPrograms() {
	for _, program := range programManager.Programs {
		programManager.ProgramTaskChan <- ProgramTask{
			Action:  ProgramTaskActionStop,
			Program: program,
		}
	}
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

	for key := range programManager.Programs {
		programsKeys = append(programsKeys, key)
	}

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
			// le program existe déjà, il faut le mettre à jour
			programManager.UpdateProgram(program, programConfiguration)
		} else {
			// le program n'existe pas, il faut l'ajouter
			programManager.AddProgram(programConfiguration)
		}
	}

	for name := range programManager.Programs {
		if !ProgramListContainsProgram(configPrograms, name) {
			// le program n'existe plus, il faut le retirer
			programManager.RemoveProgramByName(name)
		}
	}
}

func (programManager *ProgramManager) UpdateProgram(program *Program, programConfig ProgramConfig) {

}

func (programManager *ProgramManager) AddProgram(programConfig ProgramConfig) {
	var stdoutWriter, stderrWriter io.Writer

	if len(programConfig.Stdout) > 0 {
		stdoutFile, err := os.OpenFile(programConfig.Stdout, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		stdoutWriter = bufio.NewWriter(stdoutFile)
	}
	if len(programConfig.Stderr) > 0 {
		stderrFile, err := os.OpenFile(programConfig.Stderr, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		stderrWriter = bufio.NewWriter(stderrFile)
	}

	env := os.Environ()
	for name, value := range programConfig.Env {
		concatenatedKeyValue := name + "=" + value

		env = append(env, concatenatedKeyValue)
	}

	program := &Program{
		ProgramManager: programManager,
		Processes:      make(map[string]*Process),
		Config:         programConfig,
		Cache: ProgramCache{
			Env:    env,
			Stdout: stdoutWriter,
			Stderr: stderrWriter,
		},
	}

	for index := 1; index <= programConfig.Numprocs; index++ {
		id := strconv.Itoa(index)
		program.Processes[id] = NewProcess(id, program)
	}

	programManager.Programs[programConfig.Name] = program
}

func (programManager *ProgramManager) RemoveProgramByName(name string) {
	program := programManager.GetProgramByName(name)
	programManager.ProgramTaskChan <- ProgramTask{
		Action:  ProgramTaskActionStop,
		Program: program,
	}

	go func() {
		for _, process := range program.Processes {
			<-process.DeadCh
		}
		// check si tous les process sont stop

		programManager.ProgramTaskChan <- ProgramTask{
			Action:  ProgramTaskActionRemove,
			Program: program,
		}
	}()
}
