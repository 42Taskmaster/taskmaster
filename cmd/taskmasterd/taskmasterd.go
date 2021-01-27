package main

import (
	"context"
	"log"
	"sync"
)

type Taskmasterd struct {
	ProgramManager  *ProgramManager
	Umask           int
	UmaskLock       sync.Mutex
	ProgramTaskChan chan ProgramTask

	Context context.Context
	Cancel  context.CancelFunc
}

func NewTaskmasterd() *Taskmasterd {
	context, cancel := context.WithCancel(context.Background())

	taskmasterd := &Taskmasterd{
		Umask:   -1,
		Context: context,
		Cancel:  cancel,
	}

	go taskmasterd.ProgramsGoroutine()

	return taskmasterd
}

func (taskmasterd *Taskmasterd) ProgramsGoroutine() {
	programs := make(map[string]Program)

	for task := range taskmasterd.ProgramTaskChan {
		switch task.Action {
		case ProgramTaskActionStart:
			program, ok := programs[task.ProgramID]
			if !ok {
				log.Printf("program '%s' not found", task.ProgramID)
			} else {
				program.Start()
			}
		case ProgramTaskActionStop:
		case ProgramTaskActionRestart:
		case ProgramTaskActionKill:

		}
	}
}
