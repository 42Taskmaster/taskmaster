package main

import (
	"bufio"
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"
)

type Programs map[string]*Program

type Program struct {
	Cmds   []*exec.Cmd
	Config ProgramConfig
	State  ProgramState
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

func (program *Program) Start() {
	program.State = ProgramStateStarting
}

func (program *Program) Stop() {
	program.State = ProgramStateStopping
}

func (program *Program) Restart() {
	program.Stop()
	program.Start()
}

func programParse(config ProgramConfig) *Program {
	var stdoutWrite, stderrWrite io.Writer

	if len(config.Stdout) > 0 {
		stdoutFile, err := os.OpenFile(config.Stdout, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		stdoutWrite = bufio.NewWriter(stdoutFile)
	}
	if len(config.Stderr) > 0 {
		stderrFile, err := os.OpenFile(config.Stderr, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		stderrWrite = bufio.NewWriter(stderrFile)
	}

	env := []string{}
	for name, value := range config.Env {
		env = append(env, name+"="+value)
	}

	cmds := []*exec.Cmd{}

	for i := 0; i < config.Numprocs; i++ {
		cmd := exec.Command(config.Cmd)
		cmd.Env = append(os.Environ(), env...)
		cmd.Stdin = nil
		cmd.Stdout = stdoutWrite
		cmd.Stderr = stderrWrite
		cmd.ExtraFiles = nil
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setsid: true,
		}
		cmds = append(cmds, cmd)
	}

	return &Program{
		Cmds:   cmds,
		Config: config,
	}
}

func programsParse(config ProgramsConfiguration) Programs {
	parsedPrograms := make(Programs)

	for name, program := range config {
		parsedPrograms[name] = programParse(program)
	}

	return parsedPrograms
}
