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
		errorMsg := configPathArg
		if configPathArg == configDefaultPath {
			errorMsg += "\nUse -c option to specify your config file location"
		}
		log.Fatalf("Could not find config file: %s", errorMsg)
	}
}

func configParse() ProgramsYaml {
	configCheckPath()

	yamlData, err := ioutil.ReadFile(configPathArg)
	if err != nil {
		log.Panic(err)
	}
	return yamlParse(yamlData)
}
