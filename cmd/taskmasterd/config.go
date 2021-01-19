package main

import (
	"io/ioutil"
	"log"
	"os"
)

const configDefaultPath = "./taskmaster.yaml"

func configFileExists() bool {
	_, err := os.Stat(configPathArg)
	if err == nil {
		return true
	}
	return !os.IsNotExist(err)
}

func configCheckPath() {
	if !configFileExists() {
		log.Printf("Could not find config file: %s", configPathArg)
		if configPathArg == configDefaultPath {
			log.Print("Use -c option to specify your config file location")
		}
		os.Exit(1)
	}
}

func configParse() (ProgramsConfiguration, error) {
	configCheckPath()

	yamlData, err := ioutil.ReadFile(configPathArg)
	if err != nil {
		return ProgramsConfiguration{}, err
	}

	parsedPrograms := yamlParse(yamlData)

	programsConfiguration, err := parsedPrograms.Validate()

	return programsConfiguration, err
}
