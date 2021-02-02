package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

type StopSignal string

const (
	StopSignalTerm StopSignal = "TERM"
	StopSignalHup  StopSignal = "HUP"
	StopSignalInt  StopSignal = "INT"
	StopSignalQuit StopSignal = "QUIT"
	StopSignalKill StopSignal = "KILL"
	StopSignalUsr1 StopSignal = "USR1"
	StopSignalUsr2 StopSignal = "USR2"
)

var StopSignalAvailable = [...]StopSignal{
	StopSignalTerm,
	StopSignalHup,
	StopSignalInt,
	StopSignalQuit,
	StopSignalKill,
	StopSignalUsr1,
	StopSignalUsr2,
}

func (signal StopSignal) String() string {
	return string(signal)
}

func (signal StopSignal) Valid() bool {
	for _, availableStopSignal := range StopSignalAvailable {
		if availableStopSignal == signal {
			return true
		}
	}

	return false
}

func (signal StopSignal) ToOsSignal() os.Signal {
	switch signal {
	case StopSignalTerm:
		return syscall.SIGTERM
	case StopSignalHup:
		return syscall.SIGHUP
	case StopSignalInt:
		return syscall.SIGINT
	case StopSignalQuit:
		return syscall.SIGQUIT
	case StopSignalKill:
		return syscall.SIGKILL
	case StopSignalUsr1:
		return syscall.SIGUSR1
	case StopSignalUsr2:
		return syscall.SIGUSR2
	default:
		log.Panicf("unexpected signal: %s\n", signal)
		return nil
	}
}
func (taskmasterd *Taskmasterd) SignalsSetup() {
	taskmasterd.SignalsExitSetup()
	taskmasterd.SignalSighupSetup()
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

func (taskmasterd *Taskmasterd) SignalSighupSetup() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP)
	go func() {
		for {
			<-sigs
			log.Print("SIGHUP received, reloading configuration file")
			configReader, err := configGetFileReader(configPathArg)
			if err != nil {
				log.Panic(err)
			}

			programsConfigurations, err := configParse(configReader)
			if err != nil {
				log.Print("Error parsing configuration file: %s: %v\n", configPathArg, err)
			} else {
				taskmasterd.LoadProgramsConfigurations(programsConfigurations)
			}
		}
	}()
}

func (taskmasterd *Taskmasterd) SetUmask(umask string) {
	if len(umask) == 0 {
		return
	}

	taskmasterd.UmaskLock.Lock()
	defer taskmasterd.UmaskLock.Unlock()

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
