package main

type ProgramTaskAction string

const (
	ProgramTaskActionStart   ProgramTaskAction = "START"
	ProgramTaskActionStop    ProgramTaskAction = "STOP"
	ProgramTaskActionRestart ProgramTaskAction = "RESTART"
)

type ProgramTask struct {
	programTaskAction ProgramTaskAction
	programName       string
}

func NewProgramTask(program *Program, programTaskAction ProgramTaskAction) ProgramTask {
	return ProgramTask{
		programTaskAction: programTaskAction,
		programName:       program.yaml.Name,
	}
}

func (programTask *ProgramTask) Execute() {

}
