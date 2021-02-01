package main

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
)

type ErrProcessNotFound struct {
	ProcessID string
}

func (err *ErrProcessNotFound) Error() string {
	return fmt.Sprintf(
		"process not found: %s",
		err.ProcessID,
	)
}

type Program struct {
	ProcessTaskChan chan Tasker

	Context context.Context

	processes     map[string]Process
	configuration ProgramConfiguration

	Valid bool
}

type ProgramState string

const (
	ProgramStateStarting ProgramState = "STARTING"
	ProgramStateBackoff  ProgramState = "BACKOFF"
	ProgramStateRunning  ProgramState = "RUNNING"
	ProgramStateStopping ProgramState = "STOPPING"
	ProgramStateStopped  ProgramState = "STOPPED"
	ProgramStateExited   ProgramState = "EXITED"
	ProgramStateFatal    ProgramState = "FATAL"
	ProgramStateUnknown  ProgramState = "UNKNOWN"
)

type NewProgramArgs struct {
	Context       context.Context
	Configuration ProgramConfiguration
}

func NewProgram(args NewProgramArgs) Program {
	program := Program{
		ProcessTaskChan: make(chan Tasker),

		Context: args.Context,

		processes:     make(map[string]Process),
		configuration: args.Configuration,

		Valid: true,
	}

	for index := 1; index <= program.configuration.Numprocs; index++ {
		id := strconv.Itoa(index)
		process := NewProcess(NewProcessArgs{
			ID:              id,
			Context:         program.Context,
			ProgramTaskChan: program.ProcessTaskChan,
		})
		program.processes[id] = process
	}

	go program.Monitor()

	return program
}

func (program *Program) getProcessByID(id string) (Process, error) {
	process, ok := program.processes[id]
	if !ok {
		return Process{}, &ErrProcessNotFound{
			ProcessID: id,
		}
	}

	return process, nil
}

func (program *Program) getProcesses(task Tasker) error {
	programTaskWithResponse := task.(ProgramTaskRootActionWithResponse)

	select {
	case programTaskWithResponse.ResponseChan <- program.processes:
	case <-program.Context.Done():
		return ErrChannelClosed
	}

	return nil
}

func (program *Program) getProcessFromTasker(task Tasker) (Process, error) {
	processTask := task.(ProcessTask)
	processID := processTask.ProcessID
	process, err := program.getProcessByID(processID)
	if err != nil {
		return Process{}, err
	}

	return process, nil
}

func (program *Program) startSingleProcess(task Tasker) error {
	process, err := program.getProcessFromTasker(task)
	if err != nil {
		return err
	}

	process.Start()
	return nil
}

func (program *Program) stopSingleProcess(task Tasker) error {
	process, err := program.getProcessFromTasker(task)
	if err != nil {
		return err
	}

	process.Stop()
	return nil
}

func (program *Program) restartSingleProcess(task Tasker) error {
	process, err := program.getProcessFromTasker(task)
	if err != nil {
		return err
	}

	process.Restart()
	return nil
}

func (program *Program) killSingleProcess(task Tasker) error {
	process, err := program.getProcessFromTasker(task)
	if err != nil {
		return err
	}

	process.Kill()
	return nil
}

func (program *Program) startAllProcesses(task Tasker) error {
	for _, process := range program.processes {
		process.Start()
	}
	return nil
}

func (program *Program) stopAllProcesses(task Tasker) error {
	for _, process := range program.processes {
		process.Stop()
	}
	return nil
}

func (program *Program) restartAllProcesses(task Tasker) error {
	for _, process := range program.processes {
		process.Restart()
	}
	return nil
}

func (program *Program) setConfig(task Tasker) error {
	programTaskWithPayload := task.(ProgramTaskRootActionWithPayload)

	newConfig := programTaskWithPayload.Payload.(ProgramConfiguration)
	program.configuration = newConfig

	numProcesses := len(program.processes)
	if numProcesses > program.configuration.Numprocs {
		for index := numProcesses; index > program.configuration.Numprocs; index-- {
			processID := strconv.Itoa(index)

			process, err := program.getProcessByID(processID)
			if err != nil {
				return err
			}

			err = process.Stop()
			if err != nil {
				return err
			}

			go func() {
				log.Print("waiting for death")
				<-process.DeadCh

				log.Print("process died")

				program.ProcessTaskChan <- ProcessTask{
					TaskBase: TaskBase{
						Action: ProgramTaskActionRemove,
					},
					ProcessID: process.ID,
				}
			}()
		}
	} else if numProcesses < program.configuration.Numprocs {
		for index := numProcesses; index <= program.configuration.Numprocs; index++ {
			id := strconv.Itoa(index)

			process := NewProcess(NewProcessArgs{
				ID:              id,
				Context:         program.Context,
				ProgramTaskChan: program.ProcessTaskChan,
			})
			program.processes[id] = process

			if program.configuration.Autostart {
				err := process.Start()
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (program *Program) removeSingleProcess(task Tasker) error {
	processTask := task.(ProcessTask)

	delete(program.processes, processTask.ProcessID)
	return nil
}

func (program *Program) getConfig(task Tasker) error {
	programTaskWithResponse := task.(ProgramTaskRootActionWithResponse)

	config := program.configuration

	select {
	case programTaskWithResponse.ResponseChan <- config:
	case <-program.Context.Done():
		return ErrChannelClosed
	}

	return nil
}

func (program *Program) Monitor() {
	var handlers = map[TaskAction]func(*Program, Tasker) error{
		ProgramTaskActionGetAll: (*Program).getProcesses,

		ProgramTaskActionStart:    (*Program).startSingleProcess,
		ProgramTaskActionStartAll: (*Program).startAllProcesses,

		ProgramTaskActionStop:    (*Program).stopSingleProcess,
		ProgramTaskActionStopAll: (*Program).stopAllProcesses,

		ProgramTaskActionRestart:    (*Program).restartSingleProcess,
		ProgramTaskActionRestartAll: (*Program).restartAllProcesses,

		ProgramTaskActionKill: (*Program).killSingleProcess,

		ProgramTaskActionRemove: (*Program).removeSingleProcess,

		ProgramTaskActionSetConfig: (*Program).setConfig,
		ProgramTaskActionGetConfig: (*Program).getConfig,
	}

	for {
		select {
		case <-program.Context.Done():
			return

		case task := <-program.ProcessTaskChan:
			fn, ok := handlers[task.GetAction()]
			if !ok {
				log.Fatalf("not implemented task: %v", task)
			}

			fn(program, task)
		}
	}
}

func (program *Program) Start() {
	select {
	case program.ProcessTaskChan <- ProgramTaskRootAction{
		TaskBase: TaskBase{
			Action: ProgramTaskActionStartAll,
		},
	}:
	case <-program.Context.Done():
	}
}

func (program *Program) Stop() {
	select {
	case program.ProcessTaskChan <- ProgramTaskRootAction{
		TaskBase: TaskBase{
			Action: ProgramTaskActionStopAll,
		},
	}:
	case <-program.Context.Done():
	}
}

func (program *Program) Restart() {
	select {
	case program.ProcessTaskChan <- ProgramTaskRootAction{
		TaskBase: TaskBase{
			Action: ProgramTaskActionRestart,
		},
	}:
	case <-program.Context.Done():
	}
}

func (program *Program) GetProcesses() (map[string]Process, error) {
	responseChan := make(chan interface{})

	select {
	case program.ProcessTaskChan <- ProgramTaskRootActionWithResponse{
		ProgramTaskRootAction: ProgramTaskRootAction{
			TaskBase: TaskBase{
				Action: ProgramTaskActionGetAll,
			},
		},

		ResponseChan: responseChan,
	}:
	case <-program.Context.Done():
		return nil, ErrChannelClosed
	}

	select {
	case resp := <-responseChan:
		processes := resp.(map[string]Process)
		return processes, nil
	case <-program.Context.Done():
		return nil, ErrChannelClosed
	}
}

func (program *Program) GetSortedProcesses() ([]Process, error) {
	ids := []string{}

	processes, err := program.GetProcesses()
	if err != nil {
		return nil, err
	}

	for id := range processes {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	sortedProcesses := []Process{}
	for _, id := range ids {
		process, ok := processes[id]
		if ok {
			sortedProcesses = append(sortedProcesses, process)
		}
	}

	return sortedProcesses, nil
}

func GetProgramState(processes []Process) ProgramState {
	starting := 0
	running := 0
	backoff := 0
	stopping := 0
	stopped := 0
	exited := 0
	fatal := 0
	unknown := 0

	for _, process := range processes {
		switch process.Machine.Current() {
		case ProcessStateStarting:
			starting++
		case ProcessStateRunning:
			running++
		case ProcessStateBackoff:
			backoff++
		case ProcessStateStopping:
			stopping++
		case ProcessStateStopped:
			stopped++
		case ProcessStateExited:
			exited++
		case ProcessStateFatal:
			fatal++
		default:
			unknown++
		}
	}

	if unknown > 0 {
		return ProgramStateUnknown
	}
	if fatal > 0 {
		return ProgramStateFatal
	}
	if starting > 0 {
		return ProgramStateStarting
	}
	if stopping > 0 {
		return ProgramStateStopping
	}
	if backoff > 0 {
		return ProgramStateBackoff
	}
	if stopped == len(processes) {
		return ProgramStateStopped
	}
	if exited == len(processes) {
		return ProgramStateExited
	}
	if running > 0 {
		return ProgramStateRunning
	}
	return ProgramStateUnknown
}
