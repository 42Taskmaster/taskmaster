package main

import (
	"log"
	"os"
)

func daemonRun(args Args) int {
	pid, err := fork(args)
	if err != nil {
		log.Panic(err)
	}
	return pid
}

func daemonInit(args Args) {
	if !args.DaemonArg {
		log.Print("Starting daemon...")
		if lockFileExists() {
			log.Fatal("Daemon lockfile exists: is daemon already running ?")
		}
		pid := daemonRun(args)
		if pid > 0 {
			log.Printf("Log file location: %s", args.LogPathArg)
			log.Fatalf("Daemon launched with PID %d", pid)
			os.Exit(0)
		}
	}
}
