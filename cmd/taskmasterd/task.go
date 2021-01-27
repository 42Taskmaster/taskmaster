package main

type TaskAction string

const (
	ProcessTaskActionStart   TaskAction = "START"
	ProcessTaskActionStop    TaskAction = "STOP"
	ProcessTaskActionRestart TaskAction = "RESTART"
	ProcessTaskActionKill    TaskAction = "KILL"

	ProgramTaskActionStart   TaskAction = "START_ALL"
	ProgramTaskActionStop    TaskAction = "STOP_ALL"
	ProgramTaskActionRestart TaskAction = "RESTART_ALL"
	// ProgramTaskActionKill    TaskAction = "KILL_ALL"

	ProgramTaskActionGetConfig TaskAction = "GET_CONFIG"
)

type Tasker interface {
	GetAction() TaskAction
}

type TaskBase struct {
	Action TaskAction
}

func (task TaskBase) GetAction() TaskAction {
	return task.Action
}

type ProcessTask struct {
	TaskBase

	ProcessID string
}

type ProcessTaskWithReponse struct {
	ProcessTask

	ResponseChan chan interface{}
}

type ProgramTask struct {
	TaskBase
}

type ProgramTaskWithResponse struct {
	ProgramTask

	ResponseChan chan interface{}
}
