package main

type ProgramTaskAction string

const (
	ProgramTaskActionStart             ProgramTaskAction = "START"
	ProgramTaskActionStop              ProgramTaskAction = "STOP"
	ProgramTaskActionRestart           ProgramTaskAction = "RESTART"
	ProgramTaskActionGetMachineCurrent ProgramTaskAction = "GET_MACHINE_CURRENT"
)

type ProgramTask struct {
	Action  ProgramTaskAction
	Program *Program

	ResponseCh chan<- interface{}
}

type NewProgramTaskArgs struct {
	ProgramTaskAction ProgramTaskAction
	Program           *Program

	ResponseCh chan<- interface{}
}
