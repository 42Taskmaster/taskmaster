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

	GlobalContext context.Context

	LocalContext       context.Context
	CancelLocalContext context.CancelFunc

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
	localContext, localCancel := context.WithCancel(context.Background())

	program := Program{
		ProcessTaskChan: make(chan Tasker),

		GlobalContext: args.Context,

		LocalContext:       localContext,
		CancelLocalContext: localCancel,

		processes:     make(map[string]Process),
		configuration: args.Configuration,

		Valid: true,
	}

	for index := 1; index <= program.configuration.Numprocs; index++ {
		id := strconv.Itoa(index)
		process := NewProcess(NewProcessArgs{
			ID:              id,
			Context:         localContext,
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

	programTaskWithResponse.ResponseChan <- program.processes

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
	log.Printf("Starting program '%s' with %d process(es)...", program.configuration.Name, program.configuration.Numprocs)
	for _, process := range program.processes {
		process.Start()
	}
	return nil
}

func (program *Program) stopAllProcesses(task Tasker) error {
	log.Printf("Stopping program '%s' with %d process(es)...", program.configuration.Name, program.configuration.Numprocs)
	for _, process := range program.processes {
		process.Stop()
	}
	return nil
}

func (program *Program) stopAllProcessesAndWait(task Tasker) error {
	go func() {
		programTaskWithResponse := task.(ProgramTaskRootActionWithResponse)

		for _, process := range program.processes {
			process.Stop()
		}

		for _, process := range program.processes {
			process.Wait()
		}

		program.CancelLocalContext()
		close(programTaskWithResponse.ResponseChan)
	}()

	return nil
}

func (program *Program) restartAllProcesses(task Tasker) error {
	log.Printf("Restarting program '%s' with %d process(es)...", program.configuration.Name, program.configuration.Numprocs)
	for _, process := range program.processes {
		process.Restart()
	}
	return nil
}

func (program *Program) setConfig(task Tasker) error {
	programTaskWithPayload := task.(ProgramTaskRootActionWithPayload)

	newConfig := programTaskWithPayload.Payload.(ProgramConfiguration)

	program.configuration = newConfig

	oldNumProcess := len(program.processes)
	newNumProcesses := newConfig.Numprocs
	delta := newNumProcesses - oldNumProcess

	if delta < 0 {
		for index := newNumProcesses + 1; index <= oldNumProcess; index++ {
			processID := strconv.Itoa(index)

			process, err := program.getProcessByID(processID)
			if err != nil {
				return err
			}

			go func() {
				process.Wait()

				program.ProcessTaskChan <- ProcessTask{
					TaskBase: TaskBase{
						Action: ProgramTaskActionRemove,
					},
					ProcessID: process.ID,
				}
			}()

			err = process.Stop()
			if err != nil {
				return err
			}
		}
	} else if delta > 0 {
		for index := oldNumProcess + 1; index <= newNumProcesses; index++ {
			processID := strconv.Itoa(index)

			process := NewProcess(NewProcessArgs{
				ID:              processID,
				Context:         program.LocalContext,
				ProgramTaskChan: program.ProcessTaskChan,
			})
			program.processes[processID] = process

			if program.configuration.Autostart {
				err := process.Start()
				if err != nil {
					return err
				}
			}
		}
	}

	// TODO: reload the configuration of all the others processes if necessary

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
	programTaskWithResponse.ResponseChan <- config

	return nil
}

func (program *Program) Monitor() {
	var handlers = map[TaskAction]func(*Program, Tasker) error{
		ProgramTaskActionGetAll: (*Program).getProcesses,

		ProgramTaskActionStart:    (*Program).startSingleProcess,
		ProgramTaskActionStartAll: (*Program).startAllProcesses,

		ProgramTaskActionStop:           (*Program).stopSingleProcess,
		ProgramTaskActionStopAll:        (*Program).stopAllProcesses,
		ProgramTaskActionStopAllAndWait: (*Program).stopAllProcessesAndWait,

		ProgramTaskActionRestart:    (*Program).restartSingleProcess,
		ProgramTaskActionRestartAll: (*Program).restartAllProcesses,

		ProgramTaskActionKill: (*Program).killSingleProcess,

		ProgramTaskActionRemove: (*Program).removeSingleProcess,

		ProgramTaskActionSetConfig: (*Program).setConfig,
		ProgramTaskActionGetConfig: (*Program).getConfig,
	}

	for {
		select {
		case <-program.LocalContext.Done():
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
	case <-program.GlobalContext.Done():
	}
}

func (program *Program) Stop() {
	select {
	case program.ProcessTaskChan <- ProgramTaskRootAction{
		TaskBase: TaskBase{
			Action: ProgramTaskActionStopAll,
		},
	}:
	case <-program.GlobalContext.Done():
	}
}

func (program *Program) StopAndWait() chan interface{} {
	processesExited := make(chan interface{})

	program.ProcessTaskChan <- ProgramTaskRootActionWithResponse{
		ProgramTaskRootAction: ProgramTaskRootAction{
			TaskBase: TaskBase{
				Action: ProgramTaskActionStopAllAndWait,
			},
		},

		ResponseChan: processesExited,
	}

	return processesExited
}

func (program *Program) Restart() {
	select {
	case program.ProcessTaskChan <- ProgramTaskRootAction{
		TaskBase: TaskBase{
			Action: ProgramTaskActionRestart,
		},
	}:
	case <-program.GlobalContext.Done():
	}
}

func (program *Program) GetProcesses() (map[string]Process, error) {
	responseChan := make(chan interface{})

	program.ProcessTaskChan <- ProgramTaskRootActionWithResponse{
		ProgramTaskRootAction: ProgramTaskRootAction{
			TaskBase: TaskBase{
				Action: ProgramTaskActionGetAll,
			},
		},

		ResponseChan: responseChan,
	}

	resp := <-responseChan
	processes := resp.(map[string]Process)
	return processes, nil
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
