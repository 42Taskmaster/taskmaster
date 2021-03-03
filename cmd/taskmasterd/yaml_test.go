package main

import (
	"errors"
	"testing"
)

func Equal(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func MapStringKeyStringValueEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}

	for key, valueInA := range a {
		valueInB, ok := b[key]
		if !ok {
			return false
		}
		if valueInA != valueInB {
			return false
		}
	}

	return true
}

func strToPointer(str string) *string {
	return &str
}

func intToPointer(nb int) *int {
	return &nb
}

func boolToPointer(b bool) *bool {
	return &b
}

func autorestartTypeToPointer(t AutorestartType) *AutorestartType {
	return &t
}

func stopSignalToPointer(signal StopSignal) *StopSignal {
	return &signal
}

func TestCmdIsRequired(t *testing.T) {
	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd: nil,
			},
		},
	}

	_, err := programs.Validate()
	if err == nil {
		t.Errorf("Validate should have returned an error")
		return
	}

	var validationError *ErrProgramsYamlValidation
	if errors.As(err, &validationError) {
		if !(validationError.Field == "Programs[taskmaster].Cmd" && validationError.Issue == ValidationIssueEmptyField) {
			t.Errorf(
				"Incorrect error: (%s, %s); expected (%s, %s)",
				validationError.Field,
				validationError.Issue,
				"Programs[taskmaster].Cmd",
				ValidationIssueEmptyField,
			)
			return
		}
		return
	}

	t.Errorf("Returned invalid error")
}

func TestRejectsInvalidTypeValuesForExitcodes(t *testing.T) {
	var (
		invalidStringExitcodes          interface{} = "0"
		invalidSliceWithNotNumberValues interface{} = []interface{}{
			0,
			"0",
		}

		invalidValuesToTest = []interface{}{
			invalidStringExitcodes,
			invalidSliceWithNotNumberValues,
		}
	)

	for _, invalidValue := range invalidValuesToTest {
		programs := ProgramsYaml{
			Programs: map[string]ProgramYaml{
				"taskmaster": {
					Cmd:       strToPointer("cmd"),
					Exitcodes: invalidValue,
				},
			},
		}

		_, err := programs.Validate()
		if err == nil {
			t.Errorf("Validate should have returned an error")
			return
		}

		var validationError *ErrProgramsYamlValidation
		if errors.As(err, &validationError) {
			if validationError.Field == "Programs[taskmaster].Exitcodes" && errors.Is(validationError, ValidationIssueUnexpectedType) {
				continue
			}

			t.Errorf(
				"Incorrect error: (%s, %s); expected (%s, %s)",
				validationError.Field,
				validationError.Issue,
				"Programs[taskmaster].Exitcodes",
				ValidationIssueUnexpectedType,
			)
			return
		}

		t.Errorf("Returned invalid error")
		return
	}
}

func TestExitcodesAcceptIntAndFloat(t *testing.T) {
	var (
		programRawExitcodes = []interface{}{
			0,
			1.0,
		}
		programIntExitcodes = []int{
			0,
			1,
		}
	)

	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd:       strToPointer("cmd"),
				Exitcodes: programRawExitcodes,
			},
		},
	}

	config, _ := programs.Validate()

	if exitcodes := config["taskmaster"].Exitcodes; !Equal(exitcodes, programIntExitcodes) {
		t.Errorf(
			"Numprocs not set to correct default value: %v; expected %v",
			exitcodes,
			programIntExitcodes,
		)
	}
}

func TestExitcodesAreNotOutsideLowerBounds(t *testing.T) {
	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd: strToPointer("cmd"),
				Exitcodes: []interface{}{
					0,
					-1,
				},
			},
		},
	}

	_, err := programs.Validate()
	if err == nil {
		t.Errorf("Validate should have returned an error")
		return
	}

	var validationError *ErrProgramsYamlValidation
	if errors.As(err, &validationError) {
		if !(validationError.Field == "Programs[taskmaster].Exitcodes" && errors.Is(validationError, ValidationIssueValueOutsideBounds)) {
			t.Errorf(
				"Incorrect error: (%s, %s); expected (%s, %s)",
				validationError.Field,
				validationError.Issue,
				"Programs[taskmaster].Exitcodes",
				ValidationIssueValueOutsideBounds,
			)
			return
		}
		return
	}

	t.Errorf("Returned invalid error")
}

func TestExitcodesAreNotOutsideUpperBounds(t *testing.T) {
	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd: strToPointer("cmd"),
				Exitcodes: []interface{}{
					0,
					256,
				},
			},
		},
	}

	_, err := programs.Validate()
	if err == nil {
		t.Errorf("Validate should have returned an error")
		return
	}

	var validationError *ErrProgramsYamlValidation
	if errors.As(err, &validationError) {
		if !(validationError.Field == "Programs[taskmaster].Exitcodes" && errors.Is(validationError, ValidationIssueValueOutsideBounds)) {
			t.Errorf(
				"Incorrect error: (%s, %s); expected (%s, %s)",
				validationError.Field,
				validationError.Issue,
				"Programs[taskmaster].Exitcodes",
				ValidationIssueValueOutsideBounds,
			)
			return
		}
		return
	}

	t.Errorf("Returned invalid error")
}

func TestNumprocsSetToDefaultValue(t *testing.T) {
	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd:      strToPointer("cmd"),
				Numprocs: nil,
			},
		},
	}

	config, _ := programs.Validate()

	if numprocs := config["taskmaster"].Numprocs; numprocs != 1 {
		t.Errorf(
			"Numprocs not set to correct default value: %v; expected %v",
			numprocs,
			1,
		)
	}
}

func TestNumprocsIsNotOutsideLowerBounds(t *testing.T) {
	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd:      strToPointer("cmd"),
				Numprocs: intToPointer(-1),
			},
		},
	}

	_, err := programs.Validate()
	if err == nil {
		t.Errorf("Validate should have returned an error")
		return
	}

	var validationError *ErrProgramsYamlValidation
	if errors.As(err, &validationError) {
		if !(validationError.Field == "Programs[taskmaster].Numprocs" && validationError.Issue == ValidationIssueValueOutsideBounds) {
			t.Errorf(
				"Incorrect error: (%s, %s); expected (%s, %s)",
				validationError.Field,
				validationError.Issue,
				"Programs[taskmaster].Numprocs",
				ValidationIssueValueOutsideBounds,
			)
			return
		}
		return
	}

	t.Errorf("Returned invalid error")
}

func TestNumprocsIsNotOutsideUpperBounds(t *testing.T) {
	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd:      strToPointer("cmd"),
				Numprocs: intToPointer(200),
			},
		},
	}

	_, err := programs.Validate()
	if err == nil {
		t.Errorf("Validate should have returned an error")
		return
	}

	var validationError *ErrProgramsYamlValidation
	if errors.As(err, &validationError) {
		if !(validationError.Field == "Programs[taskmaster].Numprocs" && validationError.Issue == ValidationIssueValueOutsideBounds) {
			t.Errorf(
				"Incorrect error: (%s, %s); expected (%s, %s)",
				validationError.Field,
				validationError.Issue,
				"Programs[taskmaster].Numprocs",
				ValidationIssueValueOutsideBounds,
			)
			return
		}
		return
	}

	t.Errorf("Returned invalid error")
}

func TestUmaskIsValidValue(t *testing.T) {
	const umask = "755"

	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd:   strToPointer("cmd"),
				Umask: strToPointer(umask),
			},
		},
	}

	config, err := programs.Validate()
	if err != nil {
		t.Errorf("Expected no error for umask; received %v", err)
		return
	}

	if configUmask := config["taskmaster"].Umask; configUmask != umask {
		t.Errorf(
			"Umask value has not been provided, received: %v; expected %v",
			configUmask,
			umask,
		)
	}
}

func TestUmaskFailsOnInvalidOctal(t *testing.T) {
	// `9` is not a valid number in octal
	const umask = "759"

	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd:   strToPointer("cmd"),
				Umask: strToPointer(umask),
			},
		},
	}

	_, err := programs.Validate()
	if err == nil {
		t.Errorf("Validate should have returned an error")
		return
	}

	var validationError *ErrProgramsYamlValidation
	if errors.As(err, &validationError) {
		if !(validationError.Field == "Programs[taskmaster].Umask" && errors.Is(validationError, ValidationIssueUnexpectedValue)) {
			t.Errorf(
				"Incorrect error: (%s, %s); expected (%s, %s)",
				validationError.Field,
				validationError.Issue,
				"Programs[taskmaster].Umask",
				ValidationIssueUnexpectedValue,
			)
			return
		}
		return
	}

	t.Errorf("Returned invalid error")
}

func TestUmaskFailsOnNegativeOctal(t *testing.T) {
	// `9` is not a valid number in octal
	const umask = "-022"

	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd:   strToPointer("cmd"),
				Umask: strToPointer(umask),
			},
		},
	}

	_, err := programs.Validate()
	if err == nil {
		t.Errorf("Validate should have returned an error")
		return
	}

	var validationError *ErrProgramsYamlValidation
	if errors.As(err, &validationError) {
		if !(validationError.Field == "Programs[taskmaster].Umask" && errors.Is(validationError, ValidationIssueValueOutsideBounds)) {
			t.Errorf(
				"Incorrect error: (%s, %s); expected (%s, %s)",
				validationError.Field,
				validationError.Issue,
				"Programs[taskmaster].Umask",
				ValidationIssueValueOutsideBounds,
			)
			return
		}
		return
	}

	t.Errorf("Returned invalid error")
}

func TestUmaskAcceptsEmptyStrings(t *testing.T) {
	const umask = ""

	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd:   strToPointer("cmd"),
				Umask: strToPointer(umask),
			},
		},
	}

	config, err := programs.Validate()
	if err != nil {
		t.Errorf("Expected no error for umask; received %v", err)
		return
	}

	if configUmask := config["taskmaster"].Umask; configUmask != umask {
		t.Errorf(
			"Umask value has not been provided, received: %v; expected %v",
			configUmask,
			umask,
		)
	}
}

func TestProvidesWorkingdir(t *testing.T) {
	const workingdir = "/bin"

	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd:        strToPointer("cmd"),
				Workingdir: strToPointer(workingdir),
			},
		},
	}

	config, _ := programs.Validate()

	if configUmask := config["taskmaster"].Workingdir; configUmask != workingdir {
		t.Errorf(
			"Workingdir value has not been provided, received: %v; expected %v",
			configUmask,
			workingdir,
		)
	}
}

func TestAutostartSetToDefaultValue(t *testing.T) {
	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd:       strToPointer("cmd"),
				Numprocs:  intToPointer(10),
				Autostart: nil,
			},
		},
	}

	config, _ := programs.Validate()

	if autostart := config["taskmaster"].Autostart; autostart != true {
		t.Errorf(
			"Autostart not set to correct default value: %v; expected %v",
			autostart,
			1,
		)
	}
}

func TestAutorestartSetToDefaultValue(t *testing.T) {
	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd:         strToPointer("cmd"),
				Numprocs:    intToPointer(10),
				Autostart:   boolToPointer(true),
				Autorestart: nil,
			},
		},
	}

	config, _ := programs.Validate()

	if autorestart := config["taskmaster"].Autorestart; autorestart != AutorestartUnexpected {
		t.Errorf(
			"Autorestart not set to correct default value: %v; expected %v",
			autorestart,
			AutorestartUnexpected,
		)
	}
}

func TestAutorestartIsValidValue(t *testing.T) {
	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd:         strToPointer("cmd"),
				Numprocs:    intToPointer(10),
				Autostart:   boolToPointer(true),
				Autorestart: autorestartTypeToPointer(AutorestartOn),
			},
		},
	}

	_, err := programs.Validate()
	if err != nil {
		t.Errorf("Expected no error for autorestart = %s", AutorestartOn)
		return
	}

	*programs.Programs["taskmaster"].Autorestart = AutorestartOff

	_, err = programs.Validate()
	if err != nil {
		t.Errorf("Expected no error for autorestart = %s", AutorestartOff)
		return
	}

	*programs.Programs["taskmaster"].Autorestart = AutorestartUnexpected

	_, err = programs.Validate()
	if err != nil {
		t.Errorf("Expected no error for autorestart = %s", AutorestartUnexpected)
		return
	}

	*programs.Programs["taskmaster"].Autorestart = "Invalid value"

	_, err = programs.Validate()
	var validationError *ErrProgramsYamlValidation
	if errors.As(err, &validationError) {
		if !(validationError.Field == "Programs[taskmaster].Autorestart" && validationError.Issue == ValidationIssueUnexpectedValue) {
			t.Errorf(
				"Incorrect error: (%s, %s); expected (%s, %s)",
				validationError.Field,
				validationError.Issue,
				"Programs[taskmaster].Autorestart",
				ValidationIssueUnexpectedValue,
			)
			return
		}
		return
	}

	t.Errorf("Returned invalid error")
}

func TestStarttimeSetToDefaultValue(t *testing.T) {
	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd:         strToPointer("cmd"),
				Numprocs:    intToPointer(10),
				Autostart:   boolToPointer(true),
				Autorestart: autorestartTypeToPointer(AutorestartOn),
				Starttime:   nil,
			},
		},
	}

	config, _ := programs.Validate()

	if starttime := config["taskmaster"].Starttime; starttime != 5 {
		t.Errorf(
			"Starttime not set to correct default value: %v; expected %v",
			starttime,
			5,
		)
	}
}

func TestStarttimeIsNotOutsideLowerBounds(t *testing.T) {
	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd:         strToPointer("cmd"),
				Numprocs:    intToPointer(10),
				Autostart:   boolToPointer(true),
				Autorestart: autorestartTypeToPointer(AutorestartOn),
				Starttime:   intToPointer(-1),
			},
		},
	}

	_, err := programs.Validate()
	if err == nil {
		t.Errorf("Validate should have returned an error")
		return
	}

	var validationError *ErrProgramsYamlValidation
	if errors.As(err, &validationError) {
		if !(validationError.Field == "Programs[taskmaster].Starttime" && validationError.Issue == ValidationIssueValueOutsideBounds) {
			t.Errorf(
				"Incorrect error: (%s, %s); expected (%s, %s)",
				validationError.Field,
				validationError.Issue,
				"Programs[taskmaster].Starttime",
				ValidationIssueValueOutsideBounds,
			)
			return
		}
		return
	}

	t.Errorf("Returned invalid error")
}

func TestStarttimeIsNotOutsideUpperBounds(t *testing.T) {
	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd:         strToPointer("cmd"),
				Numprocs:    intToPointer(10),
				Autostart:   boolToPointer(true),
				Autorestart: autorestartTypeToPointer(AutorestartOn),
				Starttime:   intToPointer(100000),
			},
		},
	}

	_, err := programs.Validate()
	if err == nil {
		t.Errorf("Validate should have returned an error")
		return
	}

	var validationError *ErrProgramsYamlValidation
	if errors.As(err, &validationError) {
		if !(validationError.Field == "Programs[taskmaster].Starttime" && validationError.Issue == ValidationIssueValueOutsideBounds) {
			t.Errorf(
				"Incorrect error: (%s, %s); expected (%s, %s)",
				validationError.Field,
				validationError.Issue,
				"Programs[taskmaster].Starttime",
				ValidationIssueValueOutsideBounds,
			)
			return
		}
		return
	}

	t.Errorf("Returned invalid error")
}

func TestStartretriesSetToDefaultValue(t *testing.T) {
	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd:          strToPointer("cmd"),
				Numprocs:     intToPointer(10),
				Autostart:    boolToPointer(true),
				Autorestart:  autorestartTypeToPointer(AutorestartOn),
				Starttime:    intToPointer(5),
				Startretries: nil,
			},
		},
	}

	config, _ := programs.Validate()

	if startretries := config["taskmaster"].Startretries; startretries != 3 {
		t.Errorf(
			"Startretries not set to correct default value: %v; expected %v",
			startretries,
			3,
		)
	}
}

func TestStartretriesIsNotOutsideLowerBounds(t *testing.T) {
	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd:          strToPointer("cmd"),
				Numprocs:     intToPointer(10),
				Autostart:    boolToPointer(true),
				Autorestart:  autorestartTypeToPointer(AutorestartOn),
				Starttime:    intToPointer(5),
				Startretries: intToPointer(-1),
			},
		},
	}

	_, err := programs.Validate()
	if err == nil {
		t.Errorf("Validate should have returned an error")
		return
	}

	var validationError *ErrProgramsYamlValidation
	if errors.As(err, &validationError) {
		if !(validationError.Field == "Programs[taskmaster].Startretries" && validationError.Issue == ValidationIssueValueOutsideBounds) {
			t.Errorf(
				"Incorrect error: (%s, %s); expected (%s, %s)",
				validationError.Field,
				validationError.Issue,
				"Programs[taskmaster].Startretries",
				ValidationIssueValueOutsideBounds,
			)
			return
		}
		return
	}

	t.Errorf("Returned invalid error")
}

func TestStartretriesIsNotOutsideUpperBounds(t *testing.T) {
	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd:          strToPointer("cmd"),
				Numprocs:     intToPointer(10),
				Autostart:    boolToPointer(true),
				Autorestart:  autorestartTypeToPointer(AutorestartOn),
				Starttime:    intToPointer(5),
				Startretries: intToPointer(50),
			},
		},
	}

	_, err := programs.Validate()
	if err == nil {
		t.Errorf("Validate should have returned an error")
		return
	}

	var validationError *ErrProgramsYamlValidation
	if errors.As(err, &validationError) {
		if !(validationError.Field == "Programs[taskmaster].Startretries" && validationError.Issue == ValidationIssueValueOutsideBounds) {
			t.Errorf(
				"Incorrect error: (%s, %s); expected (%s, %s)",
				validationError.Field,
				validationError.Issue,
				"Programs[taskmaster].Startretries",
				ValidationIssueValueOutsideBounds,
			)
			return
		}
		return
	}

	t.Errorf("Returned invalid error")
}

func TestStopsignalSetToDefaultValue(t *testing.T) {
	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd:          strToPointer("cmd"),
				Numprocs:     intToPointer(10),
				Autostart:    boolToPointer(true),
				Autorestart:  autorestartTypeToPointer(AutorestartOn),
				Starttime:    intToPointer(5),
				Startretries: intToPointer(10),
				Stopsignal:   nil,
			},
		},
	}

	config, _ := programs.Validate()

	if stopsignal := config["taskmaster"].Stopsignal; stopsignal != StopSignalTerm {
		t.Errorf(
			"Stopsignal not set to correct default value: %v; expected %v",
			stopsignal,
			StopSignalTerm,
		)
	}
}

func TestStopsignalIsValidValue(t *testing.T) {
	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd:          strToPointer("cmd"),
				Numprocs:     intToPointer(10),
				Autostart:    boolToPointer(true),
				Autorestart:  autorestartTypeToPointer(AutorestartOn),
				Starttime:    intToPointer(5),
				Startretries: intToPointer(10),
				Stopsignal:   stopSignalToPointer(StopSignalTerm),
			},
		},
	}

	_, err := programs.Validate()
	if err != nil {
		t.Errorf("Expected no error for Stopsignal = %s", StopSignalTerm)
	}

	*programs.Programs["taskmaster"].Stopsignal = StopSignalHup

	_, err = programs.Validate()
	if err != nil {
		t.Errorf("Expected no error for Stopsignal = %s", StopSignalHup)
	}

	*programs.Programs["taskmaster"].Stopsignal = StopSignalInt

	_, err = programs.Validate()
	if err != nil {
		t.Errorf("Expected no error for Stopsignal = %s", StopSignalInt)
	}

	*programs.Programs["taskmaster"].Stopsignal = StopSignalQuit

	_, err = programs.Validate()
	if err != nil {
		t.Errorf("Expected no error for Stopsignal = %s", StopSignalQuit)
	}

	*programs.Programs["taskmaster"].Stopsignal = StopSignalKill

	_, err = programs.Validate()
	if err != nil {
		t.Errorf("Expected no error for Stopsignal = %s", StopSignalKill)
	}

	*programs.Programs["taskmaster"].Stopsignal = StopSignalUsr1

	_, err = programs.Validate()
	if err != nil {
		t.Errorf("Expected no error for Stopsignal = %s", StopSignalUsr1)
	}

	*programs.Programs["taskmaster"].Stopsignal = StopSignalUsr2

	_, err = programs.Validate()
	if err != nil {
		t.Errorf("Expected no error for Stopsignal = %s", StopSignalUsr2)
	}

	*programs.Programs["taskmaster"].Stopsignal = "Invalid value"

	_, err = programs.Validate()
	var validationError *ErrProgramsYamlValidation
	if errors.As(err, &validationError) {
		if !(validationError.Field == "Programs[taskmaster].Stopsignal" && validationError.Issue == ValidationIssueUnexpectedValue) {
			t.Errorf(
				"Incorrect error: (%s, %s); expected (%s, %s)",
				validationError.Field,
				validationError.Issue,
				"Programs[taskmaster].Stopsignal",
				ValidationIssueUnexpectedValue,
			)
			return
		}
		return
	}

	t.Errorf("Returned invalid error")
}

func TestStoptimeSetToDefaultValue(t *testing.T) {
	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd:          strToPointer("cmd"),
				Numprocs:     intToPointer(10),
				Autostart:    boolToPointer(true),
				Autorestart:  autorestartTypeToPointer(AutorestartOn),
				Starttime:    intToPointer(5),
				Startretries: intToPointer(10),
				Stopsignal:   stopSignalToPointer(StopSignalTerm),
				Stoptime:     nil,
			},
		},
	}

	config, _ := programs.Validate()

	if stoptime := config["taskmaster"].Stoptime; stoptime != 10 {
		t.Errorf(
			"Stoptime not set to correct default value: %v; expected %v",
			stoptime,
			intToPointer(10),
		)
	}
}

func TestStoptimeIsNotOutsideLowerBounds(t *testing.T) {
	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd:          strToPointer("cmd"),
				Numprocs:     intToPointer(10),
				Autostart:    boolToPointer(true),
				Autorestart:  autorestartTypeToPointer(AutorestartOn),
				Starttime:    intToPointer(5),
				Startretries: intToPointer(10),
				Stopsignal:   stopSignalToPointer(StopSignalTerm),
				Stoptime:     intToPointer(-1),
			},
		},
	}

	_, err := programs.Validate()
	if err == nil {
		t.Errorf("Validate should have returned an error")
		return
	}

	var validationError *ErrProgramsYamlValidation
	if errors.As(err, &validationError) {
		if !(validationError.Field == "Programs[taskmaster].Stoptime" && validationError.Issue == ValidationIssueValueOutsideBounds) {
			t.Errorf(
				"Incorrect error: (%s, %s); expected (%s, %s)",
				validationError.Field,
				validationError.Issue,
				"Programs[taskmaster].Stoptime",
				ValidationIssueValueOutsideBounds,
			)
			return
		}
		return
	}

	t.Errorf("Returned invalid error")
}

func TestStoptimeIsNotOutsideUpperBounds(t *testing.T) {
	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd:          strToPointer("cmd"),
				Numprocs:     intToPointer(10),
				Autostart:    boolToPointer(true),
				Autorestart:  autorestartTypeToPointer(AutorestartOn),
				Starttime:    intToPointer(5),
				Startretries: intToPointer(10),
				Stopsignal:   stopSignalToPointer(StopSignalTerm),
				Stoptime:     intToPointer(5000),
			},
		},
	}

	_, err := programs.Validate()
	if err == nil {
		t.Errorf("Validate should have returned an error")
		return
	}

	var validationError *ErrProgramsYamlValidation
	if errors.As(err, &validationError) {
		if !(validationError.Field == "Programs[taskmaster].Stoptime" && validationError.Issue == ValidationIssueValueOutsideBounds) {
			t.Errorf(
				"Incorrect error: (%s, %s); expected (%s, %s)",
				validationError.Field,
				validationError.Issue,
				"Programs[taskmaster].Stoptime",
				ValidationIssueValueOutsideBounds,
			)
			return
		}
		return
	}

	t.Errorf("Returned invalid error")
}

func TestStdoutSetToDefaultValue(t *testing.T) {
	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd:          strToPointer("cmd"),
				Numprocs:     intToPointer(10),
				Autostart:    boolToPointer(true),
				Autorestart:  autorestartTypeToPointer(AutorestartOn),
				Starttime:    intToPointer(5),
				Startretries: intToPointer(10),
				Stopsignal:   stopSignalToPointer(StopSignalTerm),
				Stoptime:     intToPointer(60),
				Stdout:       nil,
			},
		},
	}

	config, _ := programs.Validate()

	if stdout := config["taskmaster"].Stdout; stdout != string(StdTypeAuto) {
		t.Errorf(
			"Stdout not set to correct default value: %v; expected %v",
			stdout,
			string(StdTypeAuto),
		)
	}
}

func TestStderrSetToDefaultValue(t *testing.T) {
	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd:          strToPointer("cmd"),
				Numprocs:     intToPointer(10),
				Autostart:    boolToPointer(true),
				Autorestart:  autorestartTypeToPointer(AutorestartOn),
				Starttime:    intToPointer(5),
				Startretries: intToPointer(10),
				Stopsignal:   stopSignalToPointer(StopSignalTerm),
				Stoptime:     intToPointer(60),
				Stderr:       nil,
			},
		},
	}

	config, _ := programs.Validate()

	if stderr := config["taskmaster"].Stderr; stderr != string(StdTypeAuto) {
		t.Errorf(
			"Stderr not set to correct default value: %v; expected %v",
			stderr,
			string(StdTypeAuto),
		)
	}
}

func TestEnvFailsForInvalidKeys(t *testing.T) {
	var env = map[string]string{
		"NODE_ENV": "production",
		" yolo 👹 ": "http://localhost:8080/configuration/fake",
	}

	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd: strToPointer("cmd"),
				Env: env,
			},
		},
	}

	_, err := programs.Validate()
	if err == nil {
		t.Errorf("Validate should have returned an error")
		return
	}

	var validationError *ErrProgramsYamlValidation
	if errors.As(err, &validationError) {
		if !(validationError.Field == "Programs[taskmaster].Env" && errors.Is(err, ValidationIssueUnexpectedMapKey)) {
			t.Errorf(
				"Incorrect error: (%s, %s); expected (%s, %s)",
				validationError.Field,
				validationError.Issue,
				"Programs[taskmaster].Env",
				ValidationIssueUnexpectedMapKey,
			)
			return
		}
		return
	}

	t.Errorf("Returned invalid error")
}

func TestEnvIsValidValue(t *testing.T) {
	var env = map[string]string{
		"NODE_ENV":        "production",
		"TASKMASTERD_URL": "http://localhost:8080/configuration/fake",
		"TEST":            "",
	}

	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd: strToPointer("cmd"),
				Env: env,
			},
		},
	}

	config, _ := programs.Validate()

	if configEnv := config["taskmaster"].Env; !MapStringKeyStringValueEqual(configEnv, env) {
		t.Errorf(
			"Env value has not been provided, received: %v; expected %v",
			configEnv,
			env,
		)
	}
}

func TestParsesValidFullConfiguration(t *testing.T) {
	exitcodes := []interface{}{0}

	programs := ProgramsYaml{
		Programs: map[string]ProgramYaml{
			"taskmaster": {
				Cmd:          strToPointer("echo"),
				Numprocs:     intToPointer(1),
				Umask:        strToPointer("066"),
				Workingdir:   strToPointer("/dir"),
				Autostart:    boolToPointer(true),
				Autorestart:  autorestartTypeToPointer(AutorestartOn),
				Exitcodes:    exitcodes,
				Startretries: intToPointer(3),
				Starttime:    intToPointer(10),
				Stopsignal:   stopSignalToPointer(StopSignalTerm),
				Stoptime:     intToPointer(10),
				Stdout:       strToPointer("/dev/stdout"),
				Stderr:       strToPointer("/dev/stderr"),
				Env: map[string]string{
					"TERM": "DUMB",
				},
			},
		},
	}

	_, err := programs.Validate()
	if err != nil {
		t.Errorf("Validation error on valid configuration: %v", err)
	}
}
