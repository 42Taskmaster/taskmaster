package main

import (
	"log"
	"os"
)

func main() {
	argsParse()

	configReader, err := configGetFileReader(configPathArg)
	if err != nil {
		log.Panic(err)
	}

	programsConfigurations, err := configParse(configReader)
	if err != nil {
		log.Fatalf("Error parsing configuration file: %s: %v\n", configPathArg, err)
		os.Exit(1)
	}

	daemonInit()

	// Daemon only code

	logLogo()
	log.Printf("Started as daemon with PID %d", os.Getpid())

	lockFileCreate()
	defer lockFileRemove()

	taskmasterd := NewTaskmasterd()
	taskmasterd.SignalsSetup()
	taskmasterd.LoadProgramsConfigurations(programsConfigurations)

	httpSetup(taskmasterd)
	httpListenAndServe()
}
