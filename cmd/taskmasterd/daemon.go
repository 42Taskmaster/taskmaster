package main

import (
	"log"
	"os"
)

func isDaemon() bool {
	return daemonArg
}

func daemonRun() int {
	pid, err := fork()
	if err != nil {
		log.Panic(err)
	}
	return pid
}

func daemonInit() {
	if !isDaemon() {
		logLogo()
		log.Print("Starting daemon...")
		if lockFileExists() {
			log.Fatal("Daemon lockfile exists: is daemon already running ?")
		}
		pid := daemonRun()
		if pid > 0 {
			log.Printf("Log file location: %s", logPathArg)
			log.Fatalf("Daemon launched with PID %d", pid)
			os.Exit(0)
		}
	}
}
