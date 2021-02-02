package main

import (
	"context"
	"errors"
	"log"
	"sort"
)

type Taskmasterd struct {
	Args Args

	ProgramTaskChan chan Tasker

	Context context.Context
	Cancel  context.CancelFunc
}

func NewTaskmasterd(args Args) *Taskmasterd {
	context, cancel := context.WithCancel(context.Background())

	taskmasterd := &Taskmasterd{
		Args:            args,
		Context:         context,
		Cancel:          cancel,
		ProgramTaskChan: make(chan Tasker),
	}

	go taskmasterd.Monitor()

	return taskmasterd
}

func copyProgramMap(programs map[string]Program) map[string]Program {
	programsCopy := make(map[string]Program)

	for programID, program := range programs {
		programsCopy[programID] = program
	}

	return programsCopy
}

func (taskmasterd *Taskmasterd) Monitor() {
	programs := make(map[string]Program)

	for task := range taskmasterd.ProgramTaskChan {
		switch task.GetAction() {
		case TaskmasterdTaskActionGet:
			taskmasterdTask := task.(TaskmasterdTaskGet)

			program, ok := programs[taskmasterdTask.ProgramID]
			if !ok {
				close(taskmasterdTask.ResponseChan)
				break
			}

			select {
			case taskmasterdTask.ResponseChan <- program:
			case <-taskmasterd.Context.Done():
			}
		case TaskmasterdTaskActionGetAll:
			taskmasterdTask := task.(TaskmasterdTaskGetAll)

			select {
			case taskmasterdTask.ResponseChan <- copyProgramMap(programs):
			case <-taskmasterd.Context.Done():
			}
		case TaskmasterdTaskActionAdd:
			taskmasterdTask := task.(TaskmasterdTaskAdd)

			program := taskmasterdTask.Program
			programs[program.configuration.Name] = program
			if program.configuration.Autostart {
				program.Start()
			}
		case TaskmasterdTaskActionRemove:
			taskmasterdTask := task.(TaskmasterdTask)

			program := programs[taskmasterdTask.ProgramID]
			delete(programs, taskmasterdTask.ProgramID)
			program.Stop()
		}
	}
}

func (taskmasterd *Taskmasterd) LoadProgramConfiguration(config ProgramConfiguration) error {
	log.Printf("Loading program '%s' configuration...", config.Name)
	program, err := taskmasterd.GetProgramById(config.Name)
	if err != nil {
		program := NewProgram(NewProgramArgs{
			Context:       taskmasterd.Context,
			Configuration: config,
		})

		select {
		case taskmasterd.ProgramTaskChan <- TaskmasterdTaskAdd{
			TaskBase: TaskBase{
				Action: TaskmasterdTaskActionAdd,
			},
			Program: program,
		}:
		case <-taskmasterd.Context.Done():
			return ErrChannelClosed
		}

		return nil
	}

	program.ProcessTaskChan <- ProgramTaskRootActionWithPayload{
		ProgramTaskRootAction: ProgramTaskRootAction{
			TaskBase{
				Action: ProgramTaskActionSetConfig,
			},
		},
		Payload: config,
	}

	return nil
}

func (taskmasterd *Taskmasterd) LoadProgramsConfigurations(configs ProgramsConfigurations) error {
	log.Printf("Loading %d program(s) configuration(s)...", len(configs))
	programs, err := taskmasterd.GetPrograms()
	if err != nil {
		return err
	}

	for _, config := range configs {
		err := taskmasterd.LoadProgramConfiguration(config)
		if err != nil {
			return err
		}
	}

	for programID := range programs {
		_, isProgramInNewConfig := configs[programID]
		if isProgramInNewConfig {
			continue
		}

		select {
		case taskmasterd.ProgramTaskChan <- TaskmasterdTask{
			TaskBase: TaskBase{
				Action: TaskmasterdTaskActionRemove,
			},
			ProgramID: programID,
		}:
		case <-taskmasterd.Context.Done():
			return ErrChannelClosed
		}
	}
	return nil
}

var ErrProgramNotFound = errors.New("program not found")

func (taskmasterd *Taskmasterd) GetProgramById(id string) (Program, error) {
	responseChan := make(chan Program)

	select {
	case taskmasterd.ProgramTaskChan <- TaskmasterdTaskGet{
		TaskBase: TaskBase{
			Action: TaskmasterdTaskActionGet,
		},

		ProgramID:    id,
		ResponseChan: responseChan,
	}:
	case <-taskmasterd.Context.Done():
		return Program{}, ErrChannelClosed
	}

	select {
	case program := <-responseChan:
		if !program.Valid {
			return Program{}, ErrProgramNotFound
		}
		return program, nil
	case <-taskmasterd.Context.Done():
		return Program{}, ErrChannelClosed
	}
}

func (taskmasterd *Taskmasterd) GetPrograms() (map[string]Program, error) {
	responseChan := make(chan map[string]Program)

	select {
	case taskmasterd.ProgramTaskChan <- TaskmasterdTaskGetAll{
		TaskBase: TaskBase{
			Action: TaskmasterdTaskActionGetAll,
		},
		ResponseChan: responseChan,
	}:
	case <-taskmasterd.Context.Done():
		return nil, ErrChannelClosed
	}

	select {
	case programs := <-responseChan:
		return programs, nil
	case <-taskmasterd.Context.Done():
		return nil, ErrChannelClosed
	}
}

func (taskmasterd *Taskmasterd) GetSortedPrograms() ([]Program, error) {
	ids := []string{}

	programs, err := taskmasterd.GetPrograms()
	if err != nil {
		return nil, err
	}

	for id := range programs {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	sortedPrograms := []Program{}
	for _, id := range ids {
		program, ok := programs[id]
		if ok {
			sortedPrograms = append(sortedPrograms, program)
		}
	}

	return sortedPrograms, nil
}
