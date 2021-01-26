package main

type ProgramTaskAction string

const (
	ProgramTaskActionStart   ProgramTaskAction = "START"
	ProgramTaskActionStop    ProgramTaskAction = "STOP"
	ProgramTaskActionRestart ProgramTaskAction = "RESTART"
	ProgramTaskActionKill    ProgramTaskAction = "KILL"
)

type ProgramTask struct {
	Action  ProgramTaskAction
	Program *Program
	Process *Process
}
