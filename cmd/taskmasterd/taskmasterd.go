package main

import (
	"log"
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
	//taskmasterd.SignalChldSetup()
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

func (taskmasterd *Taskmasterd) SignalChldSetup() {
	sigs := make(chan os.Signal, 100)
	signal.Notify(sigs, syscall.SIGCHLD)
	go func() {
		for range sigs {
			log.Print("SIGCHLD received")
			taskmasterd.ProgramManager.ExitedProgramsProcesses()
		}
	}()
}
