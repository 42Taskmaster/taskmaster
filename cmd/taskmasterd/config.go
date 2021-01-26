package main

import (
	"io"
	"log"
	"os"
)

const configDefaultPath = "./taskmaster.yaml"

func configFileExists(path string) bool {
	_, err := os.Stat(configPathArg)
	if err == nil {
		return true
	}
	return !os.IsNotExist(err)
}

func configCheckPath(path string) bool {
	if !configFileExists(path) {
		log.Printf("Could not find config file: %s", configPathArg)
		if configPathArg == configDefaultPath {
			log.Print("Use -c option to specify your config file location")
		}
		return false
	}
	return true
}

func configGetFileReader(path string) (io.Reader, error) {
	if !configCheckPath(path) {
		os.Exit(1)
	}
	return os.Open(path)
}

func configParse(r io.Reader) (ProgramsConfiguration, error) {
	parsedPrograms, err := yamlParse(r)
	if err != nil {
		return nil, err
	}

	programsConfiguration, err := parsedPrograms.Validate()

	return programsConfiguration, err
}
