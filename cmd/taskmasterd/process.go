package main

import (
	"context"
	"os/exec"

	"github.com/VisorRaptors/taskmaster/machine"
)

type Process struct {
	ID string

	Context         context.Context
	ProgramTaskChan chan<- Tasker

	TaskActionChan chan TaskAction
	Cmd            *exec.Cmd
	Machine        machine.Machine

	DeadCh chan struct{}
}

type NewProcessArgs struct {
	ID              string
	Context         context.Context
	ProgramTaskChan chan<- Tasker
}

func NewProcess(args NewProcessArgs) *Process {
	process := &Process{
		ID:              args.ID,
		Context:         args.Context,
		ProgramTaskChan: args.ProgramTaskChan,

		TaskActionChan: make(chan TaskAction),
	}

	process.Machine = NewProcessMachine(process)

	go process.Monitor()

	return process
}

func (process *Process) Monitor() {
	for {
		select {
		case <-process.Context.Done():
			return
		case action := <-process.TaskActionChan:
			switch action {
			case ProcessTaskActionStart:
				process.Machine.Send(ProcessEventStart)
			case ProcessTaskActionStop:
				process.Machine.Send(ProcessEventStop)
			}
		}
	}
}

func (process *Process) Start() {
	process.TaskActionChan <- ProcessTaskActionStart
}

func (process *Process) Stop() {
	process.TaskActionChan <- ProcessTaskActionStart
}

func (process *Process) Restart() {
	process.Start()
	process.Stop()
}

func (process *Process) Kill() {
	process.TaskActionChan <- ProcessTaskActionKill
}

func (process *Process) GetConfig() ProgramConfiguration {
	responseChan := make(chan interface{})

	process.ProgramTaskChan <- ProgramTaskWithResponse{
		ProgramTask: ProgramTask{
			TaskBase: TaskBase{
				ProgramTaskActionGetConfig,
			},
		},

		ResponseChan: responseChan,
	}

	res := <-responseChan
	config := res.(ProgramConfiguration)

	return config
}
