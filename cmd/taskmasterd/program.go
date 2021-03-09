package main

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"sort"
	"strconv"
	"strings"
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

	processes     map[string]Processer
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

		processes:     make(map[string]Processer),
		configuration: args.Configuration,

		Valid: true,
	}

	for index := 1; index <= program.configuration.Numprocs; index++ {
		id := createProcessName(program.configuration.Name, index)

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

func createProcessName(programName string, id int) string {
	return strings.ReplaceAll(programName, " ", "-") + "_" + strconv.Itoa(id)
}

func (program *Program) getProcessByID(id string) (Processer, error) {
	process, ok := program.processes[id]
	if !ok {
		return nil, &ErrProcessNotFound{
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

func (program *Program) getProcessFromTasker(task Tasker) (Processer, error) {
	processTask := task.(ProcessTask)
	processID := processTask.ProcessID
	process, err := program.getProcessByID(processID)
	if err != nil {
		return nil, err
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
	log.Printf(
		"Restarting program '%s' with %d process(es)...",
		program.configuration.Name,
		program.configuration.Numprocs,
	)
	for _, process := range program.processes {
		process.Restart()
	}
	return nil
}

func (program *Program) setConfig(task Tasker) error {
	programTaskWithPayload := task.(ProgramTaskRootActionWithPayload)

	newConfig := programTaskWithPayload.Payload.(ProgramConfiguration)

	restartProcesses := false
	if newConfig.Cmd != program.configuration.Cmd ||
		!reflect.DeepEqual(newConfig.Env, program.configuration.Env) ||
		newConfig.Umask != program.configuration.Umask ||
		newConfig.Stdout != program.configuration.Stdout ||
		newConfig.Stderr != program.configuration.Stderr ||
		newConfig.Workingdir != program.configuration.Workingdir {
		restartProcesses = true
	}

	program.configuration = newConfig

	oldNumProcess := len(program.processes)
	newNumProcesses := newConfig.Numprocs
	delta := newNumProcesses - oldNumProcess

	if delta < 0 {
		for index := 1; index <= oldNumProcess; index++ {
			processID := createProcessName(program.configuration.Name, index)

			process, err := program.getProcessByID(processID)
			if err != nil {
				return err
			}

			if index > newNumProcesses {
				serializedProcess := process.Serialize()

				go func() {
					process.Wait()

					program.ProcessTaskChan <- ProcessTask{
						TaskBase: TaskBase{
							Action: ProgramTaskActionRemove,
						},
						ProcessID: serializedProcess.ID,
					}
				}()

				process.Stop()
			} else if restartProcesses {
				process.Restart()
			}
		}
	} else if delta > 0 {
		for index := 1; index <= newNumProcesses; index++ {
			if index > oldNumProcess {
				processID := createProcessName(program.configuration.Name, index)

				process := NewProcess(NewProcessArgs{
					ID:              processID,
					Context:         program.LocalContext,
					ProgramTaskChan: program.ProcessTaskChan,
				})
				program.processes[processID] = process

				if program.configuration.Autostart {
					process.Start()
				}
			} else if restartProcesses {
				processID := createProcessName(program.configuration.Name, index)

				process, err := program.getProcessByID(processID)
				if err != nil {
					return err
				}

				process.Restart()
			}
		}
	} else if restartProcesses {
		program.restartAllProcesses(nil)
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
			Action: ProgramTaskActionRestartAll,
		},
	}:
	case <-program.GlobalContext.Done():
	}
}

func (program *Program) GetProcesses() (map[string]Processer, error) {
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
	processes := resp.(map[string]Processer)
	return processes, nil
}

func (program *Program) GetSortedProcesses() ([]Processer, error) {
	ids := []string{}

	processes, err := program.GetProcesses()
	if err != nil {
		return nil, err
	}

	for id := range processes {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	sortedProcesses := make([]Processer, 0, len(ids))
	for _, id := range ids {
		process, ok := processes[id]
		if ok {
			sortedProcesses = append(sortedProcesses, process)
		}
	}

	return sortedProcesses, nil
}

func (program *Program) GetConfig() (ProgramConfiguration, error) {
	responseChan := make(chan interface{})

	select {
	case program.ProcessTaskChan <- ProgramTaskRootActionWithResponse{
		ProgramTaskRootAction: ProgramTaskRootAction{
			TaskBase: TaskBase{
				Action: ProgramTaskActionGetConfig,
			},
		},

		ResponseChan: responseChan,
	}:
	case <-program.LocalContext.Done():
		return ProgramConfiguration{}, ErrChannelClosed
	}

	select {
	case res := <-responseChan:
		config := res.(ProgramConfiguration)

		return config, nil
	case <-program.LocalContext.Done():
		return ProgramConfiguration{}, ErrChannelClosed
	}
}

func GetProgramState(processes []Processer) ProgramState {
	starting := 0
	running := 0
	backoff := 0
	stopping := 0
	stopped := 0
	exited := 0
	fatal := 0
	unknown := 0

	for _, process := range processes {
		switch process.GetStateMachineCurrentState() {
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
