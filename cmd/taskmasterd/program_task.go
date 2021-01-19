package main

type ProgramTaskAction string

const (
	ProgramTaskActionStart   ProgramTaskAction = "START"
	ProgramTaskActionStop    ProgramTaskAction = "STOP"
	ProgramTaskActionRestart ProgramTaskAction = "RESTART"
)

type ProgramTask struct {
	action  ProgramTaskAction
	program *Program
}

func NewProgramTask(program *Program, programTaskAction ProgramTaskAction) ProgramTask {
	return ProgramTask{
		action:  programTaskAction,
		program: program,
	}
}
