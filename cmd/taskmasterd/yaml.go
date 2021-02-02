package main

import (
	"io"
	"os"

	"gopkg.in/yaml.v2"
)

type AutorestartType string

const (
	AutorestartOn         AutorestartType = "true"
	AutorestartOff        AutorestartType = "false"
	AutorestartUnexpected AutorestartType = "unexpected"
)

type StdType string

const (
	StdTypeAuto StdType = "AUTO"
	StdTypeNone StdType = "NONE"
)

type ValidationIssue string

const (
	ValidationIssueEmptyField         ValidationIssue = "field is required but empty"
	ValidationIssueValueOutsideBounds ValidationIssue = "value is outside bounds"
	ValidationIssueUnexpectedValue    ValidationIssue = "unexpected value"
)

type ErrProgramsYamlValidation struct {
	Field string
	Issue ValidationIssue
}

func (err *ErrProgramsYamlValidation) Error() string {
	return "validation error for field " + err.Field + " : " + string(err.Issue)
}

type ProgramsConfigurations map[string]ProgramConfiguration

type ProgramsYaml struct {
	Programs map[string]ProgramYaml `yaml:"programs,omitempty"`
}

func (programs *ProgramsYaml) Validate() (ProgramsConfigurations, error) {
	programsConfigurations := make(ProgramsConfigurations)

	if programs.Programs == nil {
		return nil, &ErrProgramsYamlValidation{
			Field: "Programs",
			Issue: ValidationIssueEmptyField,
		}
	}

	for programName, programConfiguration := range programs.Programs {
		err, parsedConfiguration := programConfiguration.Validate()
		if err == nil {
			parsedConfiguration.Name = programName
			programsConfigurations[programName] = parsedConfiguration
			continue
		}

		err.Field = "Programs[" + programName + "]." + err.Field
		return nil, err
	}

	return programsConfigurations, nil
}

type ProgramConfiguration struct {
	Name         string
	Cmd          string
	Numprocs     int
	Umask        string
	Workingdir   string
	Autostart    bool
	Autorestart  AutorestartType
	Exitcodes    []int
	Startretries int
	Starttime    int
	Stopsignal   StopSignal
	Stoptime     int
	Stdout       string
	Stderr       string
	Env          map[string]string
}

func (config *ProgramConfiguration) CreateCmdEnvironment() []string {
	env := os.Environ()
	for name, value := range config.Env {
		concatenatedKeyValue := name + "=" + value

		env = append(env, concatenatedKeyValue)
	}
	return env
}

func (config *ProgramConfiguration) CreateCmdStdout(processID string) (io.Writer, error) {
	if len(config.Stdout) == 0 || config.Stdout == "NONE" {
		return nil, nil
	}

	path := config.Stdout
	if config.Stdout == "AUTO" {
		path = joinTempDir("taskmasterd-" + config.Name + "-" + processID + ".stdout")
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (config *ProgramConfiguration) CreateCmdStderr(processID string) (io.Writer, error) {
	if len(config.Stderr) == 0 || config.Stderr == "NONE" {
		return nil, nil
	}

	path := config.Stderr
	if config.Stderr == "AUTO" {
		path = joinTempDir("taskmasterd-" + config.Name + "-" + processID + ".stderr")
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return file, nil
}

type ProgramYaml struct {
	Name         string
	Cmd          *string          `yaml:"cmd,omitempty"`
	Numprocs     *int             `yaml:"numprocs,omitempty"`
	Umask        *string          `yaml:"umask,omitempty"`
	Workingdir   *string          `yaml:"workingdir,omitempty"`
	Autostart    *bool            `yaml:"autostart,omitempty"`
	Autorestart  *AutorestartType `yaml:"autorestart,omitempty"`
	Exitcodes    interface{}      `yaml:"exitcodes,omitempty"`
	Startretries *int             `yaml:"startretries,omitempty"`
	Starttime    *int             `yaml:"starttime,omitempty"`
	Stopsignal   *StopSignal      `yaml:"stopsignal,omitempty"`
	Stoptime     *int             `yaml:"stoptime,omitempty"`
	Stdout       *string          `yaml:"stdout,omitempty"`
	Stderr       *string          `yaml:"stderr,omitempty"`
	Env          map[string]string
}

func (program *ProgramYaml) NormalizedExitcodes() []int {
	switch exitcodes := program.Exitcodes.(type) {
	case int:
		return []int{exitcodes}
	case []interface{}:
		var exitcodesSlice []int
		for _, exitcode := range exitcodes {
			exitcodesSlice = append(exitcodesSlice, exitcode.(int))
		}
		return exitcodesSlice
	default:
		return []int{0}
	}
}

func (program *ProgramYaml) Validate() (*ErrProgramsYamlValidation, ProgramConfiguration) {
	const HourInSeconds = 60 * 60

	config := ProgramConfiguration{
		Exitcodes: program.NormalizedExitcodes(),
		Env:       program.Env,
	}

	if program.Cmd == nil {
		return &ErrProgramsYamlValidation{
			Field: "Cmd",
			Issue: ValidationIssueEmptyField,
		}, config
	}

	config.Cmd = *program.Cmd

	if program.Numprocs == nil {
		config.Numprocs = 1
	} else if *program.Numprocs < 0 || *program.Numprocs > 100 {
		return &ErrProgramsYamlValidation{
			Field: "Numprocs",
			Issue: ValidationIssueValueOutsideBounds,
		}, config
	} else {
		config.Numprocs = *program.Numprocs
	}

	if program.Autostart == nil {
		config.Autostart = true
	} else {
		config.Autostart = *program.Autostart
	}

	if program.Autorestart == nil {
		config.Autorestart = AutorestartUnexpected
	} else if !(*program.Autorestart == AutorestartOn ||
		*program.Autorestart == AutorestartOff ||
		*program.Autorestart == AutorestartUnexpected) {
		return &ErrProgramsYamlValidation{
			Field: "Autorestart",
			Issue: ValidationIssueUnexpectedValue,
		}, config
	} else {
		config.Autorestart = *program.Autorestart
	}

	if program.Starttime == nil {
		config.Starttime = 5
	} else if *program.Starttime < 0 || *program.Starttime > HourInSeconds {
		return &ErrProgramsYamlValidation{
			Field: "Starttime",
			Issue: ValidationIssueValueOutsideBounds,
		}, config
	} else {
		config.Starttime = *program.Starttime
	}

	if program.Startretries == nil {
		config.Startretries = 3
	} else if *program.Startretries < 0 || *program.Startretries > 20 {
		return &ErrProgramsYamlValidation{
			Field: "Startretries",
			Issue: ValidationIssueValueOutsideBounds,
		}, config
	} else {
		config.Startretries = *program.Startretries
	}

	if program.Stopsignal == nil {
		config.Stopsignal = StopSignalTerm
	} else if !program.Stopsignal.Valid() {
		return &ErrProgramsYamlValidation{
			Field: "Stopsignal",
			Issue: ValidationIssueUnexpectedValue,
		}, config
	} else {
		config.Stopsignal = *program.Stopsignal
	}

	if program.Stoptime == nil {
		config.Stoptime = 10
	} else if *program.Stoptime < 0 || *program.Stoptime > HourInSeconds {
		return &ErrProgramsYamlValidation{
			Field: "Stoptime",
			Issue: ValidationIssueValueOutsideBounds,
		}, config
	} else {
		config.Stoptime = *program.Stoptime
	}

	if program.Stdout == nil {
		config.Stdout = string(StdTypeAuto)
	} else {
		config.Stdout = *program.Stdout
	}

	if program.Stderr == nil {
		config.Stderr = string(StdTypeAuto)
	} else {
		config.Stderr = *program.Stderr
	}

	return nil, config
}

func yamlParse(r io.Reader) (ProgramsYaml, error) {
	var programsYaml ProgramsYaml

	decoder := yaml.NewDecoder(r)
	if err := decoder.Decode(&programsYaml); err != nil {
		return ProgramsYaml{}, err
	}
	return programsYaml, nil
}
