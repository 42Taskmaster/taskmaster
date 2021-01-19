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
	cmds  []*exec.Cmd
	yaml  ProgramYaml
	state ProgramState
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

func (program *Program) GetExitcodes() []int {
	switch exitcodes := program.yaml.Exitcodes.(type) {
	case int:
		return []int{exitcodes}
	case []int:
		return exitcodes
	default:
		return []int{0}
	}
}

func (program *Program) Start() {

}

func (program *Program) Stop() {

}

func (program *Program) Restart() {

}

func programParse(programYaml ProgramYaml) *Program {
	var stdoutWrite, stderrWrite io.Writer

	if len(*programYaml.Stdout) > 0 {
		stdoutFile, err := os.OpenFile(*programYaml.Stdout, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		stdoutWrite = bufio.NewWriter(stdoutFile)
	}
	if len(*programYaml.Stderr) > 0 {
		stderrFile, err := os.OpenFile(*programYaml.Stderr, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		stderrWrite = bufio.NewWriter(stderrFile)
	}

	env := []string{}
	for name, value := range programYaml.Env {
		env = append(env, name+"="+value)
	}

	cmds := []*exec.Cmd{}

	for i := 0; i < *programYaml.Numprocs; i++ {
		cmd := exec.Command(*programYaml.Cmd)
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
		cmds: cmds,
		yaml: programYaml,
	}
}

func programsParse(programsYaml ProgramsYaml) Programs {
	parsedPrograms := make(Programs)

	for name, programYaml := range programsYaml.Programs {
		parsedPrograms[name] = programParse(programYaml)
	}

	return parsedPrograms
}
