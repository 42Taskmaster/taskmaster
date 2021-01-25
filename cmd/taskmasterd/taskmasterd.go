package main

import (
	"os"
	"os/signal"
	"syscall"
)

type Taskmasterd struct {
	ProgramManager *ProgramManager
}

func NewTaskmasterd(programManager *ProgramManager) *Taskmasterd {
	return &Taskmasterd{
		ProgramManager: programManager,
	}
}

func (taskmasterd *Taskmasterd) SignalsSetup() {
	taskmasterd.SignalsExitSetup()
}

func (taskmasterd *Taskmasterd) SignalsExitSetup() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT, syscall.SIGQUIT)
	go func() {
		<-sigs
		lockFileRemove()
		os.Exit(0)
	}()
}
