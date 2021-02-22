package main

import (
	"io"
	"log"
	"os"
)

const configDefaultPath = "./taskmaster.yaml"

func configFileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return !os.IsNotExist(err)
}

func configCheckPath(path string) bool {
	if !configFileExists(path) {
		log.Printf("Could not find config file: %s", path)
		if path == configDefaultPath {
			log.Print("Use -c option to specify your config file location")
		}
		return false
	}
	return true
}

func configGetFileReader(path string) (io.ReadCloser, error) {
	if !configCheckPath(path) {
		os.Exit(1)
	}
	return os.Open(path)
}

func configParse(r io.Reader) (ProgramsYaml, ProgramsConfigurations, error) {
	parsedPrograms, err := yamlParse(r)
	if err != nil {
		return ProgramsYaml{}, nil, err
	}

	programsConfigurations, err := parsedPrograms.Validate()

	return parsedPrograms, programsConfigurations, err
}
