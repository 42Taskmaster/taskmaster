package main

import (
	"log"

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

type StopSignal string

const (
	StopSignalTerm StopSignal = "TERM"
	StopSignalHup  StopSignal = "HUP"
	StopSignalInt  StopSignal = "INT"
	StopSignalQuit StopSignal = "QUIT"
	StopSignalKill StopSignal = "KILL"
	StopSignalUsr1 StopSignal = "USR1"
	StopSignalUsr2 StopSignal = "USR2"
)

var StopSignalAvailable = [...]StopSignal{
	StopSignalTerm,
	StopSignalHup,
	StopSignalInt,
	StopSignalQuit,
	StopSignalKill,
	StopSignalUsr1,
	StopSignalUsr2,
}

func (signal StopSignal) Valid() bool {
	for _, availableStopSignal := range StopSignalAvailable {
		if availableStopSignal == signal {
			return true
		}
	}

	return false
}

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

type ProgramsYaml struct {
	Programs map[string]ProgramYaml `yaml:"omitempty"`
}

func (programs *ProgramsYaml) Validate() error {
	if programs.Programs == nil {
		return &ErrProgramsYamlValidation{
			Field: "Programs",
			Issue: ValidationIssueEmptyField,
		}
	}

	for programName, programConfiguration := range programs.Programs {
		err := validateProgram(&programConfiguration)
		if err == nil {
			continue
		}

		err.Field = "Programs[" + programName + "]." + err.Field
		return err
	}

	return nil
}

func validateProgram(program *ProgramYaml) *ErrProgramsYamlValidation {
	const HourInSeconds = 60 * 60

	if program.Cmd == nil {
		return &ErrProgramsYamlValidation{
			Field: "Cmd",
			Issue: ValidationIssueEmptyField,
		}
	}

	if program.Numprocs == nil {
		*program.Numprocs = 1
	}
	if *program.Numprocs < 0 || *program.Numprocs > 100 {
		return &ErrProgramsYamlValidation{
			Field: "Numprocs",
			Issue: ValidationIssueValueOutsideBounds,
		}
	}

	if program.Autostart == nil {
		*program.Autostart = true
	}

	if program.Autorestart == nil {
		*program.Autorestart = AutorestartUnexpected
	}
	if !(*program.Autorestart == AutorestartOn ||
		*program.Autorestart == AutorestartOff ||
		*program.Autorestart == AutorestartUnexpected) {
		return &ErrProgramsYamlValidation{
			Field: "Autorestart",
			Issue: ValidationIssueUnexpectedValue,
		}
	}

	if program.Starttime == nil {
		*program.Starttime = 1
	}
	if *program.Starttime < 0 || *program.Starttime > HourInSeconds {
		return &ErrProgramsYamlValidation{
			Field: "Starttime",
			Issue: ValidationIssueValueOutsideBounds,
		}
	}

	if program.Startretries == nil {
		*program.Startretries = 3
	}
	if *program.Startretries < 0 || *program.Startretries > 20 {
		return &ErrProgramsYamlValidation{
			Field: "Starttime",
			Issue: ValidationIssueValueOutsideBounds,
		}
	}

	if program.Stopsignal == nil {
		*program.Stopsignal = StopSignalTerm
	}
	if !program.Stopsignal.Valid() {
		return &ErrProgramsYamlValidation{
			Field: "Stopsignal",
			Issue: ValidationIssueUnexpectedValue,
		}
	}

	if program.Stoptime == nil {
		*program.Stoptime = 10
	}
	if *program.Stoptime < 0 || *program.Stoptime > HourInSeconds {
		return &ErrProgramsYamlValidation{
			Field: "Stoptime",
			Issue: ValidationIssueValueOutsideBounds,
		}
	}

	if program.Stdout == nil {
		*program.Stdout = string(StdTypeAuto)
	}

	if program.Stderr == nil {
		*program.Stderr = string(StdTypeAuto)
	}

	return nil
}

type ProgramYaml struct {
	Name         string
	Cmd          *string          `yaml:"omitempty"`
	Numprocs     *int             `yaml:"omitempty"`
	Umask        *string          `yaml:"omitempty"`
	Workingdir   *string          `yaml:"omitempty"`
	Autostart    *bool            `yaml:"omitempty"`
	Autorestart  *AutorestartType `yaml:"omitempty"`
	Exitcodes    interface{}      `yaml:"omitempty"`
	Startretries *int             `yaml:"omitempty"`
	Starttime    *int             `yaml:"omitempty"`
	Stopsignal   *StopSignal      `yaml:"omitempty"`
	Stoptime     *int             `yaml:"omitempty"`
	Stdout       *string          `yaml:"omitempty"`
	Stderr       *string          `yaml:"omitempty"`
	Env          map[string]string
}

func yamlParse(yamlData []byte) ProgramsYaml {
	var programsYaml ProgramsYaml

	err := yaml.Unmarshal(yamlData, &programsYaml)
	if err != nil {
		log.Fatalf("Error parsing config file: %s", err)
	}

	return programsYaml
}
