package main

import (
	"context"
	"os/exec"
	"syscall"

	"github.com/42Taskmaster/taskmaster/machine"
)

type Process struct {
	ID string

	Context         context.Context
	ProgramTaskChan chan<- Tasker

	TaskActionChan           chan TaskAction
	Cmd                      *exec.Cmd
	stdoutClose, stderrClose func() error
	Machine                  *machine.Machine

	DeadCh *chan struct{}
}

type NewProcessArgs struct {
	ID              string
	Context         context.Context
	ProgramTaskChan chan<- Tasker
}

func NewProcess(args NewProcessArgs) Process {
	process := Process{
		ID:              args.ID,
		Context:         args.Context,
		ProgramTaskChan: args.ProgramTaskChan,

		TaskActionChan: make(chan TaskAction),
	}

	process.Machine = NewProcessMachine(&process)

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
			case ProcessTaskActionKill:
				process.Cmd.Process.Signal(syscall.SIGKILL)
			}
		}
	}
}

func (process *Process) Start() error {
	select {
	case process.TaskActionChan <- ProcessTaskActionStart:
		return nil
	case <-process.Context.Done():
		return ErrChannelClosed
	}
}

func (process *Process) Stop() error {
	select {
	case process.TaskActionChan <- ProcessTaskActionStop:
		return nil
	case <-process.Context.Done():
		return ErrChannelClosed
	}
}

func (process *Process) Restart() error {
	if err := process.Start(); err != nil {
		return err
	}
	if err := process.Stop(); err != nil {
		return err
	}
	return nil
}

func (process *Process) Kill() error {
	select {
	case process.TaskActionChan <- ProcessTaskActionKill:
		return nil
	case <-process.Context.Done():
		return ErrChannelClosed
	}
}

func (process *Process) GetConfig() (ProgramConfiguration, error) {
	responseChan := make(chan interface{})

	select {
	case process.ProgramTaskChan <- ProgramTaskRootActionWithResponse{
		ProgramTaskRootAction: ProgramTaskRootAction{
			TaskBase: TaskBase{
				Action: ProgramTaskActionGetConfig,
			},
		},

		ResponseChan: responseChan,
	}:
	case <-process.Context.Done():
		return ProgramConfiguration{}, ErrChannelClosed
	}

	select {
	case res := <-responseChan:
		config := res.(ProgramConfiguration)

		return config, nil
	case <-process.Context.Done():
		return ProgramConfiguration{}, ErrChannelClosed
	}
}

func (process *Process) Wait() {
	if process.DeadCh != nil {
		<-*process.DeadCh
	}
}

func (process *Process) CloseFileDescriptors() error {
	if err := process.stdoutClose(); err != nil {
		return err
	}

	if err := process.stderrClose(); err != nil {
		return err
	}

	return nil
}
