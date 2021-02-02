package main

import (
	"log"
	"os"
)

func rootCheck(args Args) {
	if os.Geteuid() == 0 && !args.BypassRootArg {
		log.Print("Taskmasterd should not be launched as root. Please use a non-root user.")
		log.Print("Use -r argument to launch as root anyway.")
		os.Exit(1)
	}
}

func main() {
	var args Args
	args.Parse()

	logLogo()

	rootCheck(args)

	configReader, err := configGetFileReader(args.ConfigPathArg)
	if err != nil {
		log.Panic(err)
	}

	programsConfigurations, err := configParse(configReader)
	if err != nil {
		log.Fatalf("Error parsing configuration file: %s: %v\n", args.ConfigPathArg, err)
		os.Exit(1)
	}

	daemonInit(args)

	// Daemon only code

	log.Printf("Started as daemon with PID %d", os.Getpid())

	lockFileCreate()
	defer lockFileRemove()

	taskmasterd := NewTaskmasterd()
	taskmasterd.SignalsSetup()
	taskmasterd.LoadProgramsConfigurations(programsConfigurations)

	httpSetup(taskmasterd)
	httpListenAndServe(args)
}
