package main

import (
	"context"
	"io"
	"os/exec"
	"syscall"
	"time"

	"github.com/42Taskmaster/taskmaster/machine"
)

type ProcessSerialized struct {
	ID                 string
	State              machine.StateType
	StartedAt, EndedAt time.Time
}

type Processer interface {
	GetConfig() (ProgramConfiguration, error)
	GetContext() context.Context
	GetStateMachineCurrentState() machine.StateType
	GetCmd() *exec.Cmd
	SetCmd(*exec.Cmd)
	SetStdoutStderrCloser(stdout, stderr io.WriteCloser)
	StartChronometer()
	StopChronometer()
	Start()
	Stop()
	Restart()
	Kill()
	Wait()
	GetDeadChannel() chan struct{}
	CreateNewDeadChannel() chan struct{}
	Serialize() ProcessSerialized
}

type Process struct {
	id string

	context               context.Context
	programMonitorChannel chan<- Tasker

	externalMonitorChannel, internalMonitorChannel chan Tasker
	cmd                                            *exec.Cmd
	stdoutClose, stderrClose                       func() error
	machine                                        *machine.Machine
	startedAt, endedAt                             time.Time

	deadCh chan struct{}
}

type NewProcessArgs struct {
	ID              string
	Context         context.Context
	ProgramTaskChan chan<- Tasker
}

func NewProcess(args NewProcessArgs) *Process {
	process := &Process{
		id:                    args.ID,
		context:               args.Context,
		programMonitorChannel: args.ProgramTaskChan,

		externalMonitorChannel: make(chan Tasker),
		internalMonitorChannel: make(chan Tasker),
	}

	process.machine = NewProcessMachine(process)

	go process.monitor()

	return process
}

func (process *Process) monitor() {
	for {
		select {
		case <-process.context.Done():
			return

		case task := <-process.externalMonitorChannel:
			switch action := task.GetAction(); action {
			case ProcessTaskActionStart:
				go process.machine.Send(ProcessEventStart)
			case ProcessTaskActionStop:
				go process.machine.Send(ProcessEventStop)
			case ProcessTaskActionRestart:
				go func() {
					process.machine.Send(ProcessEventStop)
					<-process.deadCh
					process.machine.Send(ProcessEventStart)
				}()
			case ProcessTaskActionKill:
				go process.cmd.Process.Signal(syscall.SIGKILL)
			}

		case task := <-process.internalMonitorChannel:
			switch action := task.GetAction(); action {
			case ProcessTaskActionStartChronometer:
				process.startedAt = time.Now()
				process.endedAt = time.Time{}
			case ProcessTaskActionStopChronometer:
				process.endedAt = time.Now()
			case ProcessTaskActionGetProgramConfig:
				taskWithResponse := task.(ProcessInternalTaskWithResponse)
				responseChan := taskWithResponse.ResponseChan
				programResponseChan := make(chan interface{})

				select {
				case process.programMonitorChannel <- ProgramTaskRootActionWithResponse{
					ProgramTaskRootAction: ProgramTaskRootAction{
						TaskBase: TaskBase{
							Action: ProgramTaskActionGetConfig,
						},
					},

					ResponseChan: programResponseChan,
				}:
				case <-process.context.Done():
					responseChan <- nil
					break
				}

				select {
				case res := <-programResponseChan:
					config := res.(ProgramConfiguration)

					responseChan <- config

					close(responseChan)
				case <-process.context.Done():
					responseChan <- nil
				}
			case ProcessTaskActionSerialize:
				taskWithResponse := task.(ProcessInternalTaskWithResponse)
				responseChan := taskWithResponse.ResponseChan

				responseChan <- ProcessSerialized{
					ID:        process.id,
					State:     process.machine.UnsafeCurrent(),
					StartedAt: process.startedAt,
					EndedAt:   process.endedAt,
				}

				close(responseChan)
			case ProcessTaskActionCreateNewDeadChannel:
				taskWithResponse := task.(ProcessInternalTaskWithResponse)
				responseChan := taskWithResponse.ResponseChan

				newDeadChannel := make(chan struct{})

				process.deadCh = newDeadChannel

				responseChan <- newDeadChannel

				close(responseChan)
			case ProcessTaskActionGetCmd:
				taskWithResponse := task.(ProcessInternalTaskWithResponse)
				responseChan := taskWithResponse.ResponseChan

				responseChan <- process.cmd

				close(responseChan)
			case ProcessTaskActionGetDeadChannel:
				taskWithResponse := task.(ProcessInternalTaskWithResponse)
				responseChan := taskWithResponse.ResponseChan

				responseChan <- process.deadCh

				close(responseChan)
			case ProcessTaskActionGetStateMachineCurrentState:
				taskWithResponse := task.(ProcessInternalTaskWithResponse)
				responseChan := taskWithResponse.ResponseChan

				responseChan <- process.machine.UnsafeCurrent()

				close(responseChan)
			case ProcessTaskActionSetCmd:
				taskWithResponse := task.(ProcessInternalTaskWithPayload)
				cmd := taskWithResponse.Payload.(*exec.Cmd)

				process.cmd = cmd
			case ProcessTaskActionSetStdoutStderrCloser:
				taskWithResponse := task.(ProcessInternalTaskWithPayload)
				closers := taskWithResponse.Payload.([]func() error)
				stdoutClose := closers[0]
				stderrClose := closers[1]

				process.stdoutClose = stdoutClose
				process.stderrClose = stderrClose
			}
		}
	}
}

func (process *Process) GetContext() context.Context {
	return process.context
}

func (process *Process) StartChronometer() {
	select {
	case process.internalMonitorChannel <- ProcessTaskActionStartChronometer:
		return
	case <-process.context.Done():
		return
	}
}

func (process *Process) StopChronometer() {
	select {
	case process.internalMonitorChannel <- ProcessTaskActionStopChronometer:
		return
	case <-process.context.Done():
		return
	}
}

func (process *Process) Start() {
	go func() {
		select {
		case process.externalMonitorChannel <- ProcessTaskActionStart:
			return
		case <-process.context.Done():
			return
		}
	}()
}

func (process *Process) Stop() {
	go func() {
		select {
		case process.externalMonitorChannel <- ProcessTaskActionStop:
			return
		case <-process.context.Done():
			return
		}
	}()
}

func (process *Process) Restart() {
	go func() {
		select {
		case process.externalMonitorChannel <- ProcessTaskActionRestart:
			return
		case <-process.context.Done():
			return
		}
	}()
}

func (process *Process) Kill() {
	go func() {
		select {
		case process.externalMonitorChannel <- ProcessTaskActionKill:
			return
		case <-process.context.Done():
			return
		}
	}()
}

func (process *Process) GetConfig() (ProgramConfiguration, error) {
	responseChan := make(chan interface{})

	select {
	case process.internalMonitorChannel <- ProcessInternalTaskWithResponse{
		TaskBase: TaskBase{
			Action: ProcessTaskActionGetProgramConfig,
		},
		ResponseChan: responseChan,
	}:
	case <-process.context.Done():
		return ProgramConfiguration{}, ErrChannelClosed
	}

	select {
	case resp := <-responseChan:
		if resp == nil {
			return ProgramConfiguration{}, ErrChannelClosed
		}

		config := resp.(ProgramConfiguration)
		return config, nil
	case <-process.context.Done():
		return ProgramConfiguration{}, ErrChannelClosed
	}
}

func (process *Process) Serialize() ProcessSerialized {
	responseChan := make(chan interface{})

	select {
	case process.internalMonitorChannel <- ProcessInternalTaskWithResponse{
		TaskBase: TaskBase{
			Action: ProcessTaskActionSerialize,
		},
		ResponseChan: responseChan,
	}:
	case <-process.context.Done():
		return ProcessSerialized{}
	}

	select {
	case resp := <-responseChan:
		serializedProcess := resp.(ProcessSerialized)
		return serializedProcess
	case <-process.context.Done():
		return ProcessSerialized{}
	}
}

func (process *Process) CreateNewDeadChannel() chan struct{} {
	responseChan := make(chan interface{})

	select {
	case process.internalMonitorChannel <- ProcessInternalTaskWithResponse{
		TaskBase: TaskBase{
			Action: ProcessTaskActionCreateNewDeadChannel,
		},
		ResponseChan: responseChan,
	}:
	case <-process.context.Done():
		return nil
	}

	select {
	case resp := <-responseChan:
		serializedProcess := resp.(chan struct{})
		return serializedProcess
	case <-process.context.Done():
		return nil
	}
}

func (process *Process) GetCmd() *exec.Cmd {
	responseChan := make(chan interface{})

	select {
	case process.internalMonitorChannel <- ProcessInternalTaskWithResponse{
		TaskBase: TaskBase{
			Action: ProcessTaskActionGetCmd,
		},
		ResponseChan: responseChan,
	}:
	case <-process.context.Done():
		return nil
	}

	select {
	case resp := <-responseChan:
		cmd := resp.(*exec.Cmd)
		return cmd
	case <-process.context.Done():
		return nil
	}
}

func (process *Process) GetDeadChannel() chan struct{} {
	responseChan := make(chan interface{})

	select {
	case process.internalMonitorChannel <- ProcessInternalTaskWithResponse{
		TaskBase: TaskBase{
			Action: ProcessTaskActionGetDeadChannel,
		},
		ResponseChan: responseChan,
	}:
	case <-process.context.Done():
		return nil
	}

	select {
	case resp := <-responseChan:
		deadCh := resp.(chan struct{})
		return deadCh
	case <-process.context.Done():
		return nil
	}
}

func (process *Process) GetStateMachineCurrentState() machine.StateType {
	responseChan := make(chan interface{})

	select {
	case process.internalMonitorChannel <- ProcessInternalTaskWithResponse{
		TaskBase: TaskBase{
			Action: ProcessTaskActionGetStateMachineCurrentState,
		},
		ResponseChan: responseChan,
	}:
	case <-process.context.Done():
		return machine.NoopState
	}

	select {
	case resp := <-responseChan:
		stateMachine := resp.(machine.StateType)
		return stateMachine
	case <-process.context.Done():
		return machine.NoopState
	}
}

func (process *Process) SetCmd(cmd *exec.Cmd) {
	go func() {
		select {
		case process.internalMonitorChannel <- ProcessInternalTaskWithPayload{
			TaskBase: TaskBase{
				Action: ProcessTaskActionSetCmd,
			},
			Payload: cmd,
		}:
		case <-process.context.Done():
			return
		}
	}()
}

func (process *Process) SetStdoutStderrCloser(stdout, stderr io.WriteCloser) {
	var stdoutClose, stderrClose func() error
	if stdout != nil {
		stdoutClose = stdout.Close
	}
	if stderr != nil {
		stderrClose = stderr.Close
	}
	go func() {
		select {
		case process.internalMonitorChannel <- ProcessInternalTaskWithPayload{
			TaskBase: TaskBase{
				Action: ProcessTaskActionSetStdoutStderrCloser,
			},
			Payload: []func() error{
				stdoutClose,
				stderrClose,
			},
		}:
		case <-process.context.Done():
			return
		}
	}()
}

func (process *Process) Wait() {
	if deadCh := process.GetDeadChannel(); deadCh != nil {
		<-deadCh
	}
}
