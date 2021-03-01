package main

import (
	"errors"
	"io"
	"os"
	"strings"

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
	Programs map[string]ProgramYaml `yaml:"programs"`
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
		parsedConfiguration, err := programConfiguration.Validate(ProgramYamlValidateArgs{})
		if err == nil {
			parsedConfiguration.Name = programName
			programsConfigurations[programName] = parsedConfiguration
			continue
		}

		var validationErr *ErrProgramsYamlValidation
		if errors.As(err, &validationErr) {
			validationErr.Field = "Programs[" + programName + "]." + validationErr.Field
			return nil, validationErr
		}

		return nil, err
	}

	return programsConfigurations, nil
}

type ProgramConfiguration struct {
	Name         string            `json:"name"`
	Cmd          string            `json:"cmd"`
	Numprocs     int               `json:"numprocs"`
	Umask        string            `json:"umask"`
	Workingdir   string            `json:"workingdir"`
	Autostart    bool              `json:"autostart"`
	Autorestart  AutorestartType   `json:"autorestart"`
	Exitcodes    []int             `json:"exitcodes"`
	Startretries int               `json:"startretries"`
	Starttime    int               `json:"starttime"`
	Stopsignal   StopSignal        `json:"stopsignal"`
	Stoptime     int               `json:"stoptime"`
	Stdout       string            `json:"stout"`
	Stderr       string            `json:"stderr"`
	Env          map[string]string `json:"env"`
}

func (config *ProgramConfiguration) CreateCmdEnvironment() []string {
	env := os.Environ()
	for name, value := range config.Env {
		concatenatedKeyValue := name + "=" + value

		env = append(env, concatenatedKeyValue)
	}
	return env
}

func (config *ProgramConfiguration) CreateCmdStdout(processID string) (io.WriteCloser, error) {
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

func (config *ProgramConfiguration) CreateCmdStderr(processID string) (io.WriteCloser, error) {
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
	Name         *string           `yaml:"-" json:"name,omitempty"`
	Cmd          *string           `yaml:"cmd,omitempty" json:"cmd,omitempty"`
	Numprocs     *int              `yaml:"numprocs,omitempty" json:"numprocs,omitempty"`
	Umask        *string           `yaml:"umask,omitempty" json:"umask,omitempty"`
	Workingdir   *string           `yaml:"workingdir,omitempty" json:"workingdir,omitempty"`
	Autostart    *bool             `yaml:"autostart,omitempty" json:"autostart,omitempty"`
	Autorestart  *AutorestartType  `yaml:"autorestart,omitempty" json:"autorestart,omitempty"`
	Exitcodes    interface{}       `yaml:"exitcodes,omitempty" json:"exitcodes,omitempty"`
	Startretries *int              `yaml:"startretries,omitempty" json:"startretries,omitempty"`
	Starttime    *int              `yaml:"starttime,omitempty" json:"starttime,omitempty"`
	Stopsignal   *StopSignal       `yaml:"stopsignal,omitempty" json:"stopsignal,omitempty"`
	Stoptime     *int              `yaml:"stoptime,omitempty" json:"stoptime,omitempty"`
	Stdout       *string           `yaml:"stdout,omitempty" json:"stdout,omitempty"`
	Stderr       *string           `yaml:"stderr,omitempty" json:"stderr,omitempty"`
	Env          map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
}

func (program *ProgramYaml) NormalizedExitcodes() []int {
	switch exitcodes := program.Exitcodes.(type) {
	case int:
		return []int{exitcodes}
	case []interface{}:
		var exitcodesSlice []int
		for _, exitcode := range exitcodes {
			float, ok := exitcode.(float64)
			if ok {
				exitcodesSlice = append(exitcodesSlice, int(float))
			} else {
				exitcodesSlice = append(exitcodesSlice, exitcode.(int))
			}
		}
		return exitcodesSlice
	default:
		return []int{0}
	}
}

type ProgramYamlValidateArgs struct {
	PickProgramName bool
}

func (program *ProgramYaml) Validate(args ProgramYamlValidateArgs) (ProgramConfiguration, error) {
	const HourInSeconds = 60 * 60

	config := ProgramConfiguration{
		Exitcodes: program.NormalizedExitcodes(),
		Env:       program.Env,
	}

	if args.PickProgramName {
		if program.Name == nil {
			return config, &ErrProgramsYamlValidation{
				Field: "Name",
				Issue: ValidationIssueEmptyField,
			}
		}

		programName := strings.TrimSpace(*program.Name)

		if programName == "" {
			return config, &ErrProgramsYamlValidation{
				Field: "Name",
				Issue: ValidationIssueEmptyField,
			}
		}

		config.Name = programName
	}

	if program.Cmd == nil {
		return config, &ErrProgramsYamlValidation{
			Field: "Cmd",
			Issue: ValidationIssueEmptyField,
		}
	}

	config.Cmd = *program.Cmd

	if program.Numprocs == nil {
		config.Numprocs = 1
	} else if *program.Numprocs < 0 || *program.Numprocs > 100 {
		return config, &ErrProgramsYamlValidation{
			Field: "Numprocs",
			Issue: ValidationIssueValueOutsideBounds,
		}
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
		return config, &ErrProgramsYamlValidation{
			Field: "Autorestart",
			Issue: ValidationIssueUnexpectedValue,
		}
	} else {
		config.Autorestart = *program.Autorestart
	}

	if program.Starttime == nil {
		config.Starttime = 5
	} else if *program.Starttime < 0 || *program.Starttime > HourInSeconds {
		return config, &ErrProgramsYamlValidation{
			Field: "Starttime",
			Issue: ValidationIssueValueOutsideBounds,
		}
	} else {
		config.Starttime = *program.Starttime
	}

	if program.Startretries == nil {
		config.Startretries = 3
	} else if *program.Startretries < 0 || *program.Startretries > 20 {
		return config, &ErrProgramsYamlValidation{
			Field: "Startretries",
			Issue: ValidationIssueValueOutsideBounds,
		}
	} else {
		config.Startretries = *program.Startretries
	}

	if program.Stopsignal == nil {
		config.Stopsignal = StopSignalTerm
	} else if !program.Stopsignal.Valid() {
		return config, &ErrProgramsYamlValidation{
			Field: "Stopsignal",
			Issue: ValidationIssueUnexpectedValue,
		}
	} else {
		config.Stopsignal = *program.Stopsignal
	}

	if program.Stoptime == nil {
		config.Stoptime = 10
	} else if *program.Stoptime < 0 || *program.Stoptime > HourInSeconds {
		return config, &ErrProgramsYamlValidation{
			Field: "Stoptime",
			Issue: ValidationIssueValueOutsideBounds,
		}
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

	return config, nil
}

func yamlParse(r io.Reader) (ProgramsYaml, error) {
	var programsYaml ProgramsYaml

	decoder := yaml.NewDecoder(r)
	if err := decoder.Decode(&programsYaml); err != nil {
		return ProgramsYaml{}, err
	}
	return programsYaml, nil
}
