package main

import "fmt"

type ProgramManager struct {
	programs        Programs
	programTaskChan chan ProgramTask
}

func NewProgramManager() *ProgramManager {
	var programManager ProgramManager
	programManager.Init()
	return &programManager
}

func (programManager *ProgramManager) Init() {
	programTaskChan := make(chan ProgramTask)

	go func() {
		for programTask := range programTaskChan {
			if programTask.action == ProgramTaskActionStart {
				programTask.program.Start()
			} else if programTask.action == ProgramTaskActionStop {
				programTask.program.Stop()
			} else if programTask.action == ProgramTaskActionRestart {
				programTask.program.Restart()
			}
		}
	}()
}

func (programManager *ProgramManager) GetProgramByName(name string) *Program {
	for _, program := range programManager.programs {
		if program.Config.Name == name {
			return program
		}
	}
	return nil
}

func (programManager *ProgramManager) StartAllPrograms() {
	for _, program := range programManager.programs {
		programManager.programTaskChan <- NewProgramTask(program, ProgramTaskActionStart)
	}
}

func (programManager *ProgramManager) StopAllPrograms() {
	for _, program := range programManager.programs {
		programManager.programTaskChan <- NewProgramTask(program, ProgramTaskActionStop)
	}
}

func (programManager *ProgramManager) StartProgramByName(name string) error {
	program := programManager.GetProgramByName(name)
	if program == nil {
		return fmt.Errorf("Program not found: \"%s\"", name)
	}
	programManager.programTaskChan <- NewProgramTask(program, ProgramTaskActionStart)
	return nil
}

func (programManager *ProgramManager) StopProgramByName(name string) error {
	program := programManager.GetProgramByName(name)
	if program == nil {
		return fmt.Errorf("Program not found: \"%s\"", name)
	}
	programManager.programTaskChan <- NewProgramTask(program, ProgramTaskActionStop)
	return nil
}

// func (programManager *ProgramManager) StartProgram(program *Program) {
// 	programManager.programTaskChan <- NewProgramTask(program, ProgramTaskActionStart)
// }

// func (programManager *ProgramManager) StopProgram(program *Program) {
// 	programManager.programTaskChan <- NewProgramTask(program, ProgramTaskActionStop)
// }

// func (programManager *ProgramManager) RestartProgram(program *Program) {
// 	programManager.programTaskChan <- NewProgramTask(program, ProgramTaskActionRestart)
// }

// func (programManager *ProgramManager) StartPrograms(programs *Programs) {
// 	for _, program := range *programs {
// 		programManager.StartProgram(program)
// 	}
// }

// func (programManager *ProgramManager) StopPrograms(programs *Programs) {
// 	for _, program := range *programs {
// 		programManager.StopProgram(program)
// 	}
// }

// func (programManager *ProgramManager) RestartPrograms(programs *Programs) {
// 	for _, program := range *programs {
// 		programManager.StartProgram(program)
// 	}
// }
