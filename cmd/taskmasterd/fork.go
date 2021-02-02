package main

import (
	"os"
	"os/exec"
	"syscall"
)

func fork(args Args) (int, error) {
	logFile := logGetFile(args)

	forkArgs := []string{
		"-d",
	}
	for _, parent := range os.Args[1:] {
		forkArgs = append(forkArgs, parent)
	}

	cmd := exec.Command(os.Args[0], forkArgs...)
	cmd.Env = os.Environ()
	cmd.Stdin = nil
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.ExtraFiles = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Setsid is used to detach the process from the parent (normally a shell)
		//
		// The disowning of a child process is accomplished by executing the system call
		// setpgrp() or setsid(), (both of which have the same functionality) as soon as
		// the child is forked. These calls create a new process session group, make the
		// child process the session leader, and set the process group ID to the process
		// ID of the child. https://bsdmag.org/unix-kernel-system-calls/
		Setsid: true,
	}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	return cmd.Process.Pid, nil
}
