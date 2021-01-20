package main

type ProgramTaskAction string

const (
	ProgramTaskActionStart   ProgramTaskAction = "START"
	ProgramTaskActionStop    ProgramTaskAction = "STOP"
	ProgramTaskActionRestart ProgramTaskAction = "RESTART"
)

type ProgramTask struct {
	Action  ProgramTaskAction
	Program *Program
}
