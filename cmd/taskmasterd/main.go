package main

import (
	"context"
	"log"
	"os"
)

func rootCheck(bypass bool) {
	if os.Geteuid() == 0 && !bypass {
		log.Print("Taskmasterd should not be launched as root. Please use a non-root user.")
		log.Print("Use -r argument to launch as root anyway.")
		os.Exit(1)
	}
}

func main() {
	var args Args
	args.Parse()

	logLogo()

	rootCheck(args.BypassRootArg)

	configReader, err := configGetFileReader(args.ConfigPathArg)
	if err != nil {
		log.Panic(err)
	}

	programsYamlConfiguration, programsConfigurations, err := configParse(configReader)
	if err != nil {
		log.Fatalf("Error parsing configuration file: %s: %v\n", args.ConfigPathArg, err)
		os.Exit(1)
	}
	configReader.Close()

	daemonInit(args)

	// Daemon only code

	log.Printf("Started as daemon with PID %d", os.Getpid())

	lockFileCreate()
	defer lockFileRemove()

	context, cancel := context.WithCancel(context.Background())

	taskmasterd := NewTaskmasterd(NewTaskmasterdArgs{
		Args:                  args,
		ProgramsConfiguration: programsYamlConfiguration,
		Context:               context,
		Cancel:                cancel,
	})
	taskmasterd.SignalsSetup()
	go taskmasterd.LoadProgramsConfigurations(programsConfigurations)

	httpSetup(taskmasterd)
	<-httpListenAndServe(context, args.PortArg)
	<-taskmasterd.Closed

	log.Println("Exited gracefully, bye!")
}
