package main

import (
	"context"
	"fmt"
	"log"
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

	Context         context.Context

	processes map[string]Process
	configuration ProgramConfiguration
}

type NewProgramArgs struct {
	Context         context.Context
	Configuration ProgramConfiguration
}

func NewProgram(args NewProgramArgs) *Program {
	program := &Program{
		Context:         args.Context,
		
		processes: make(map[string]Process)
		configuration: args.Configuration,
	}

	go program.Monitor()

	return program
}


func (program  *Program) GetProcessByID(id string) (Process, error) {
	process, ok := program.process[id]
	if !ok {
		return Process{}, &ErrProcessNotFound{
			ProcessID: id,
		}
	}

	return process, nil
}

func (program *Program) getProcessFromTasker(task Tasker) (Process, error) {
	processTask := task.(ProcessTask)
	processId := processTask.ProcessID
	process, err := program.GetProcessByID(processId)
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
}

func (program *Program) stopSingleProcess(task Tasker) error {
	process, err := program.getProcessFromTasker(task)
	if err != nil {
		return err
	}

	process.Stop()
}

func (program *Program) restartSingleProcess(task Tasker) error {
	process, err := program.getProcessFromTasker(task)
	if err != nil {
		return err
	}

	process.Restart()
}

func (program *Program) killSingleProcess(task Tasker) error {
	process, err := program.getProcessFromTasker(task)
	if err != nil {
		return err
	}

	process.Kill()
}

func (program *Program) startAllProcesses(task Tasker) error {
	task.(ProgramTask)

	for _, process := range program.processes {
		process.Start()
	}
}

func (program *Program) stopAllProcesses(task Tasker) error {
	task.(ProgramTask)

	for _, process := range program.processes {
		process.Stop()
	}
}

func (program *Program) restartAllProcesses(task Tasker) error {
	task.(ProgramTask)

	for _, process := range program.processes {
		process.Restart()
	}
}

func (program *Program) getConfig(task Tasker) error {
	programTaskWithResponse := task.(ProgramTaskWithResponse)

	config := program.configuration
	programTaskWithResponse.ResponseChan <- config
}

func (program *Program) Monitor() {
	var handlers = map[ProcessTaskAction]func (*Program, Tasker) error{
		ProcessTaskActionStart: (*Program).startSingleProcess,
		ProcessTaskActionStop: (*Program).stopSingleProcess,
		ProcessTaskActionRestart: (*Program).restartSingleProcess,
		ProcessTaskActionKill: (*Program).killSingleProcess,

		ProgramTaskActionStart: (*Program).startAllProcesses,
		ProgramTaskActionStop: (*Program).stopAllProcesses,
		ProgramTaskActionRestart: (*Program).restartAllProcesses,

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

			fn(&program, task)
		}
	}
}
