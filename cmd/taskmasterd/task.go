package main

import (
	"errors"
	"io"
)

var (
	ErrChannelClosed = errors.New("channel has been closed")
)

type Tasker interface {
	GetAction() TaskAction
}

type TaskAction string

func (task TaskAction) GetAction() TaskAction {
	return task
}

const (
	TaskmasterdTaskActionGet                                       TaskAction = "TASKMASTERD_GET"
	TaskmasterdTaskActionGetAll                                    TaskAction = "TASKMASTERD_GET_ALL"
	TaskmasterdTaskActionAdd                                       TaskAction = "TASKMASTERD_ADD"
	TaskmasterdTaskActionRemove                                    TaskAction = "TASKMASTERD_REMOVE"
	TaskmasterdTaskActionAddProgram                                TaskAction = "TASKMASTERD_ADD_PROGRAM"
	TaskmasterdTaskActionDeleteProgram                             TaskAction = "TASKMASTERD_DELETE_PROGRAM"
	TaskmasterdTaskActionEditProgram                               TaskAction = "TASKMASTERD_REPLACE_PROGRAM_CONFIGURATIONS"
	TaskmasterdTaskActionReplaceProgramsConfigurations             TaskAction = "TASKMASTERD_REPLACE_PROGRAMS_CONFIGURATIONS"
	TaskmasterdTaskActionRefreshConfigurationFromConfigurationFile TaskAction = "TASKMASTERD_REFRESH_CONFIGURATION_FROM_CONFIGURATION_FILE"
	TaskmasterdTaskActionRefreshConfigurationFromReader            TaskAction = "TASKMASTERD_REFRESH_CONFIGURATION_FROM_READER"
	TaskmasterdTaskActionGetProgramsConfigurations                 TaskAction = "TASKMASTERD_GET_PROGRAMS_CONFIGURATIONS"
	TaskmasterdTaskActionPersistProgramsToDisk                     TaskAction = "TASKMASTERD_PERSIST_PROGRAMS_TO_DISK"

	ProgramTaskActionGet            TaskAction = "PROGRAM_GET"
	ProgramTaskActionGetAll         TaskAction = "PROGRAM_GET_ALL"
	ProgramTaskActionStart          TaskAction = "PROGRAM_START"
	ProgramTaskActionStartAll       TaskAction = "PROGRAM_START_ALL"
	ProgramTaskActionStop           TaskAction = "PROGRAM_STOP"
	ProgramTaskActionStopAll        TaskAction = "PROGRAM_STOP_ALL"
	ProgramTaskActionStopAllAndWait TaskAction = "PROGRAM_STOP_ALL_AND_WAIT"
	ProgramTaskActionKill           TaskAction = "PROGRAM_KILL"
	ProgramTaskActionRestart        TaskAction = "PROGRAM_RESTART"
	ProgramTaskActionRestartAll     TaskAction = "PROGRAM_RESTART_ALL"
	ProgramTaskActionRemove         TaskAction = "PROGRAM_REMOVE"
	ProgramTaskActionSetConfig      TaskAction = "PROGRAM_SET_CONFIG"
	ProgramTaskActionGetConfig      TaskAction = "PROGRAM_GET_CONFIG"

	ProcessTaskActionGetContext                  TaskAction = "PROCESS_GET_CONTEXT"
	ProcessTaskActionSerialize                   TaskAction = "PROCESS_SERIALIZE"
	ProcessTaskActionCreateNewDeadChannel        TaskAction = "PROCESS_CREATE_NEW_DEAD_CHANNEL"
	ProcessTaskActionGetProgramConfig            TaskAction = "PROCESS_GET_PROGRAM_CONFIG"
	ProcessTaskActionGetCmd                      TaskAction = "PROCESS_GET_CMD"
	ProcessTaskActionGetDeadChannel              TaskAction = "PROCESS_GET_DEAD_CHANNEL"
	ProcessTaskActionGetStateMachineCurrentState TaskAction = "PROCESS_GET_STATE_MACHINE_CURRENT_STATE"
	ProcessTaskActionSetCmd                      TaskAction = "PROCESS_SET_CMD"
	ProcessTaskActionSetStdoutStderrCloser       TaskAction = "PROCESS_SET_STDOUT_STDERR_CLOSER"
	ProcessTaskActionStart                       TaskAction = "PROCESS_START"
	ProcessTaskActionStop                        TaskAction = "PROCESS_STOP"
	ProcessTaskActionRestart                     TaskAction = "PROCESS_RESTART"
	ProcessTaskActionKill                        TaskAction = "PROCESS_KILL"

	ProcessTaskActionStartChronometer TaskAction = "PROCESS_START_CHRONOMETER"
	ProcessTaskActionStopChronometer  TaskAction = "PROCESS_STOP_CHRONOMETER"
)

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

type TaskmasterdTaskAddProgram struct {
	TaskBase

	ProgramConfiguration ProgramYaml
	ErrorChan            chan<- error
}

type TaskmasterdTaskEditProgram struct {
	TaskBase

	ProgramId            string
	ProgramConfiguration ProgramYaml
	ErrorChan            chan<- error
}

type TaskmasterdTaskDeleteProgram struct {
	TaskBase

	ProgramId string
	ErrorChan chan<- error
}

type TaskmasterdTaskReplaceProgramsConfigurations struct {
	TaskBase

	ProgramsConfigurations ProgramsYaml
}

type TaskmasterdTaskGetProgramsConfigurations struct {
	TaskBase

	ProgramsConfigurationsChan chan<- ProgramsYaml
}

type TaskmasterdTaskRefreshConfigurationFromReader struct {
	TaskBase

	Reader    io.Reader
	ErrorChan chan<- error
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

type ProcessTaskWithResponse struct {
	ProcessTask

	ResponseChan chan interface{}
}

type ProcessTask struct {
	TaskBase

	ProcessID string
}

type ProcessInternalTaskWithResponse struct {
	TaskBase

	ResponseChan chan interface{}
}

type ProcessInternalTaskWithPayload struct {
	TaskBase

	Payload interface{}
}
