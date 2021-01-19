package main

import (
	"log"
	"os"
)

func main() {
	argsParse()

	logLogo()

	programsConfiguration, err := configParse()
	if err != nil {
		log.Fatalf("Error parsing configuration file: %s: %v\n", configPathArg, err)
	}

	daemonInit()

	// Daemon only code

	logSetup()
	logLogo()
	log.Printf("Started as daemon with PID %d", os.Getpid())

	lockFileCreate()
	defer lockFileRemove()

	signalsSetup()

	programManager := NewProgramManager()
	programManager.programs = programsParse(programsConfiguration)

	httpSetup(programManager)
	httpListenAndServe()
}
