package main

import (
	"errors"
	"io"
	"os"
	"regexp"
	"strconv"
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

var (
	ValidationIssueEmptyField         = errors.New("field is required but empty")
	ValidationIssueValueOutsideBounds = errors.New("value is outside bounds")
	ValidationIssueUnexpectedMapKey   = errors.New("unexpected map key")
	ValidationIssueUnexpectedValue    = errors.New("unexpected value")
	ValidationIssueUnexpectedType     = errors.New("unexpected type")
	ValidationIssueInvalidPath        = errors.New("invalid path")
	ValidationIssueNullChar           = errors.New("string cannot contains null char")
)

type ErrProgramsYamlValidation struct {
	Field string
	Issue error
}

func (err *ErrProgramsYamlValidation) Error() string {
	return "validation error for field " + err.Field + " : " + err.Issue.Error()
}

func (err *ErrProgramsYamlValidation) Unwrap() error {
	return err.Issue
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
	Stdout       string            `json:"stdout"`
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

func (program *ProgramYaml) NormalizedExitcodes() ([]int, error) {
	if program.Exitcodes == nil {
		return []int{0}, nil
	}

	switch exitcodes := program.Exitcodes.(type) {
	case int:
		return []int{exitcodes}, nil
	case []interface{}:
		exitcodesSlice := make([]int, len(exitcodes))

		for index, exitcode := range exitcodes {
			switch convertedExitcode := exitcode.(type) {
			case float64:
				exitcodesSlice[index] = int(convertedExitcode)
			case int:
				exitcodesSlice[index] = convertedExitcode
			default:
				return nil, ValidationIssueUnexpectedType
			}
		}

		return exitcodesSlice, nil
	default:
		return nil, ValidationIssueUnexpectedType
	}
}

type ProgramYamlValidateArgs struct {
	PickProgramName bool
}

func (program *ProgramYaml) Validate(args ProgramYamlValidateArgs) (ProgramConfiguration, error) {
	const HourInSeconds = 60 * 60

	var config ProgramConfiguration

	normalizedExitcodes, err := program.NormalizedExitcodes()
	if err != nil {
		return config, &ErrProgramsYamlValidation{
			Field: "Exitcodes",
			Issue: err,
		}
	}
	for _, exitcode := range normalizedExitcodes {
		if !(0 <= exitcode && exitcode <= 255) {
			return config, &ErrProgramsYamlValidation{
				Field: "Exitcodes",
				Issue: ValidationIssueValueOutsideBounds,
			}
		}
	}
	config.Exitcodes = normalizedExitcodes

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

		if hasNullChar(programName) {
			return config, &ErrProgramsYamlValidation{
				Field: "Name",
				Issue: ValidationIssueNullChar,
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
	if hasNullChar(*program.Cmd) {
		return config, &ErrProgramsYamlValidation{
			Field: "Cmd",
			Issue: ValidationIssueNullChar,
		}
	}

	config.Cmd = *program.Cmd

	if program.Numprocs == nil {
		config.Numprocs = 1
	} else if *program.Numprocs < 1 || *program.Numprocs > 100 {
		return config, &ErrProgramsYamlValidation{
			Field: "Numprocs",
			Issue: ValidationIssueValueOutsideBounds,
		}
	} else {
		config.Numprocs = *program.Numprocs
	}

	if program.Umask != nil {
		umask := *program.Umask

		if umask != "" {
			// Try to parse octal string to an int.
			// If we fail, the umask property must be rejected.
			if umaskAsInt, err := strconv.ParseInt(umask, 8, 64); err != nil {
				return config, &ErrProgramsYamlValidation{
					Field: "Umask",
					Issue: ValidationIssueUnexpectedValue,
				}
			} else if umaskAsInt < 0 {
				return config, &ErrProgramsYamlValidation{
					Field: "Umask",
					Issue: ValidationIssueValueOutsideBounds,
				}
			}
		}

		config.Umask = umask
	}

	if program.Workingdir != nil {
		if hasNullChar(*program.Workingdir) {
			return config, &ErrProgramsYamlValidation{
				Field: "Workingdir",
				Issue: ValidationIssueNullChar,
			}
		}
		config.Workingdir = *program.Workingdir
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
		if hasNullChar(*program.Stdout) {
			return config, &ErrProgramsYamlValidation{
				Field: "Stdout",
				Issue: ValidationIssueNullChar,
			}
		}
		config.Stdout = *program.Stdout
	}

	if program.Stderr == nil {
		config.Stderr = string(StdTypeAuto)
	} else {
		if hasNullChar(*program.Stderr) {
			return config, &ErrProgramsYamlValidation{
				Field: "Stderr",
				Issue: ValidationIssueNullChar,
			}
		}
		config.Stderr = *program.Stderr
	}

	if program.Env == nil {
		config.Env = nil
	} else {
		for key := range program.Env {
			if !isValidEnvironementVariableName(key) {
				return config, &ErrProgramsYamlValidation{
					Field: "Env",
					Issue: ValidationIssueUnexpectedMapKey,
				}
			}
		}

		config.Env = program.Env
	}

	return config, nil
}

func hasNullChar(s string) bool {
	for _, c := range s {
		if c == rune(0) {
			return true
		}
	}
	return false
}

func isValidEnvironementVariableName(s string) bool {
	re := regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_]*$")

	return re.MatchString(s)
}

func yamlParse(r io.Reader) (ProgramsYaml, error) {
	var programsYaml ProgramsYaml

	decoder := yaml.NewDecoder(r)
	if err := decoder.Decode(&programsYaml); err != nil {
		return ProgramsYaml{}, err
	}
	return programsYaml, nil
}
