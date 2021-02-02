package main

import "errors"

var (
	ErrChannelClosed = errors.New("channel has been closed")
)

type TaskAction string

const (
	TaskmasterdTaskActionGet    TaskAction = "TASKMASTERD_GET"
	TaskmasterdTaskActionGetAll TaskAction = "TASKMASTERD_GET_ALL"
	TaskmasterdTaskActionAdd    TaskAction = "TASKMASTERD_ADD"
	TaskmasterdTaskActionRemove TaskAction = "TASKMASTERD_REMOVE"

	ProgramTaskActionGet        TaskAction = "PROGRAM_GET"
	ProgramTaskActionGetAll     TaskAction = "PROGRAM_GET_ALL"
	ProgramTaskActionStart      TaskAction = "PROGRAM_START"
	ProgramTaskActionStartAll   TaskAction = "PROGRAM_START_ALL"
	ProgramTaskActionStop       TaskAction = "PROGRAM_STOP"
	ProgramTaskActionStopAll    TaskAction = "PROGRAM_STOP_ALL"
	ProgramTaskActionKill       TaskAction = "PROGRAM_KILL"
	ProgramTaskActionRestart    TaskAction = "PROGRAM_RESTART"
	ProgramTaskActionRestartAll TaskAction = "PROGRAM_RESTART_ALL"
	ProgramTaskActionRemove     TaskAction = "PROGRAM_REMOVE"
	ProgramTaskActionSetConfig  TaskAction = "PROGRAM_SET_CONFIG"
	ProgramTaskActionGetConfig  TaskAction = "PROGRAM_GET_CONFIG"

	ProcessTaskActionGet     TaskAction = "PROCESS_GET"
	ProcessTaskActionStart   TaskAction = "PROCESS_START"
	ProcessTaskActionStop    TaskAction = "PROCESS_STOP"
	ProcessTaskActionRestart TaskAction = "PROCESS_RESTART"
	ProcessTaskActionKill    TaskAction = "PROCESS_KILL"
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

type TaskmasterdTask struct {
	TaskBase

	ProgramID string
}

type TaskmasterdTaskGet struct {
	TaskBase

	ProgramID    string
	ResponseChan chan Program
}

type TaskmasterdTaskGetAll struct {
	TaskBase

	ProgramID    string
	ResponseChan chan map[string]Program
}

type TaskmasterdTaskAdd struct {
	TaskBase

	Program Program
}

type ProgramTask struct {
	TaskBase

	ProgramID string
}

type ProgramTaskWithResponse struct {
	ProgramTask

	ResponseChan chan interface{}
}

type ProgramTaskRootAction struct {
	TaskBase
}

type ProgramTaskRootActionWithResponse struct {
	ProgramTaskRootAction

	ResponseChan chan interface{}
}

type ProgramTaskRootActionWithPayload struct {
	ProgramTaskRootAction

	Payload interface{}
}

type ProcessTaskWithReponse struct {
	ProcessTask

	ResponseChan chan interface{}
}

type ProcessTask struct {
	TaskBase

	ProcessID string
}
