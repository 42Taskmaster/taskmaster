package main

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
			programTask.Execute()
		}
	}()
}

// func (programManager *ProgramManager) GetProgramByName(name string) *Program {
// 	for _, program := range programs {
// 		if program.yaml.Name == name {
// 			return program
// 		}
// 	}
// 	return nil
// }

// func (programManager *ProgramManager) StartProgramByName(name string) error {
// 	program := GetProgramByName(name)
// 	if program == nil {
// 		return fmt.Errorf("Program not found: \"%s\"", name)
// 	}
// }

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
