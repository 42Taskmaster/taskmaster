package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
)

type Taskmasterd struct {
	ProgramManager *ProgramManager
	Umask          int
	UmaskLock      sync.Mutex
}

func NewTaskmasterd() *Taskmasterd {
	return &Taskmasterd{
		Umask: -1,
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

func (taskmasterd *Taskmasterd) SetUmask(umask string) {
	if len(umask) == 0 {
		return
	}

	taskmasterd.UmaskLock.Lock()
	defer taskmasterd.UmaskLock.Unlock()

	log.Print("Setting umask: ", umask)
	octal, err := strconv.ParseInt(umask, 8, 64)
	if err != nil {
		log.Panic(err)
	}

	taskmasterd.Umask = syscall.Umask(int(octal))
}

func (taskmasterd *Taskmasterd) ResetUmask() {
	if taskmasterd.Umask != -1 {
		syscall.Umask(taskmasterd.Umask)
	}
}
