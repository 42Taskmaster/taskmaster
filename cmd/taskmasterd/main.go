package main

import (
	"log"
	"os"
)

func main() {
	argsParse()

	programsConfiguration, err := configParse()
	if err != nil {
		log.Fatalf("Error parsing configuration file: %s: %v\n", configPathArg, err)
	}

	daemonInit()

	// Daemon only code

	logLogo()
	log.Printf("Started as daemon with PID %d", os.Getpid())

	lockFileCreate()
	defer lockFileRemove()

	taskmasterd := NewTaskmasterd()
	taskmasterd.SignalsSetup()

	programManager := NewProgramManager(taskmasterd)
	programManager.Programs = programsParse(programManager, programsConfiguration)

	taskmasterd.ProgramManager = programManager

	httpSetup(programManager)
	httpListenAndServe()
}
