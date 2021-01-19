package main

func TestProgramsRequired (t *testing.T) {
	programs := ProgramsYaml{
	}

	err := programs.Validate()
	if err != nil {
		t.Errorf("Validate should have returned an error")
	}
	
	if validationError, ok := err.(*ErrProgramsYamlValidation); ok {
		if !(validationError.Field == "Programs" && validationError.Issue == ValidationIssueEmptyField) {
			t.Errorf(
				"Incorrect error: (%s, %s); expected (%s, %s)",
				validationError.Field,
				validationError.Issue,
				"Programs",
				ValidationIssueEmptyField
			)
			return
		}
		return
	}

	t.Errorf("Returned invalid error")
}

func TestCmdIsRequired (t *testing.T) {
	programs := ProgramsYaml{
		Programs: {
			"taskmaster": {
				Cmd: nil,
			}
		},
	}

	err := programs.Validate()
	if err != nil {
		t.Errorf("Validate should have returned an error")
	}
	
	if validationError, ok := err.(*ErrProgramsYamlValidation); ok {
		if !(validationError.Field == "Programs[taskmaster].Cmd" && validationError.Issue == ValidationIssueEmptyField) {
			t.Errorf(
				"Incorrect error: (%s, %s); expected (%s, %s)",
				validationError.Field,
				validationError.Issue,
				"Programs[taskmaster].Cmd",
				ValidationIssueEmptyField
			)
			return
		}
		return
	}

	t.Errorf("Returned invalid error")
}

func TestNumprocsSetToDefaultValue (t *testing.T) {
	programs := ProgramsYaml{
		Programs: {
			"taskmaster": {
				Cmd: "cmd",
				Numprocs: nil
			}
		},
	}

	programs.Validate()

	if numprocs := programs.Programs["taskmaster"].Numprocs; numprocs != 1 {
		t.Errorf(
			"Numprocs not set to correct default value: %v; expected %v",
			numprocs,
			1,
		)
	}
}

func TestNumprocsIsNotOutsideLowerBounds (t *testing.T) {
	programs := ProgramsYaml{
		Programs: {
			"taskmaster": {
				Cmd: "cmd",
				Numprocs: -1
			}
		},
	}

	err := programs.Validate()
	if err != nil {
		t.Errorf("Validate should have returned an error")
	}
	
	if validationError, ok := err.(*ErrProgramsYamlValidation); ok {
		if !(validationError.Field == "Programs[taskmaster].Numprocs" && validationError.Issue == ValidationIssueValueOutsideBounds) {
			t.Errorf(
				"Incorrect error: (%s, %s); expected (%s, %s)",
				validationError.Field,
				validationError.Issue,
				"Programs[taskmaster].Numprocs",
				ValidationIssueValueOutsideBounds
			)
			return
		}
		return
	}

	t.Errorf("Returned invalid error")
}

func TestNumprocsIsNotOutsideUpperBounds (t *testing.T) {
	programs := ProgramsYaml{
		Programs: {
			"taskmaster": {
				Cmd: "cmd",
				Numprocs: 200
			}
		},
	}

	err := programs.Validate()
	if err != nil {
		t.Errorf("Validate should have returned an error")
	}
	
	if validationError, ok := err.(*ErrProgramsYamlValidation); ok {
		if !(validationError.Field == "Programs[taskmaster].Numprocs" && validationError.Issue == ValidationIssueValueOutsideBounds) {
			t.Errorf(
				"Incorrect error: (%s, %s); expected (%s, %s)",
				validationError.Field,
				validationError.Issue,
				"Programs[taskmaster].Numprocs",
				ValidationIssueValueOutsideBounds
			)
			return
		}
		return
	}

	t.Errorf("Returned invalid error")
}

func TestAutostartSetToDefaultValue (t *testing.T) {
	programs := ProgramsYaml{
		Programs: {
			"taskmaster": {
				Cmd: "cmd",
				Numprocs: 10,
				Autostart: nil,
			}
		},
	}

	programs.Validate()

	if autostart := programs.Programs["taskmaster"].Autostart; autostart != 1 {
		t.Errorf(
			"Autostart not set to correct default value: %v; expected %v",
			autostart,
			1,
		)
	}
}

func TestAutorestartSetToDefaultValue (t *testing.T) {
	programs := ProgramsYaml{
		Programs: {
			"taskmaster": {
				Cmd: "cmd",
				Numprocs: 10,
				Autostart: true,
				Autorestart: nil,
			}
		},
	}

	programs.Validate()

	if autorestart := programs.Programs["taskmaster"].Autorestart; autorestart != AutorestartUnexpected {
		t.Errorf(
			"Autorestart not set to correct default value: %v; expected %v",
			autorestart,
			AutorestartUnexpected,
		)
	}
}

func TestAutorestartIsValidValue (t *testing.T) {
	programs := ProgramsYaml{
		Programs: {
			"taskmaster": {
				Cmd: "cmd",
				Numprocs: 10,
				Autostart: true,
				Autorestart: AutorestartOn,
			}
		},
	}

	err := programs.Validate()
	if err != nil {
		t.Errorf("Expected no error for autorestart = %s", AutorestartOn)
	}

	programs.Programs["taskmaster"].Autorestart = AutorestartOff

	err = programs.Validate()
	if err != nil {
		t.Errorf("Expected no error for autorestart = %s", AutorestartOff)
	}

	programs.Programs["taskmaster"].Autorestart = AutorestartUnexpected

	err = programs.Validate()
	if err != nil {
		t.Errorf("Expected no error for autorestart = %s", AutorestartUnexpected)
	}

	programs.Programs["taskmaster"].Autorestart = "Invalid value"

	err = programs.Validate()
	if validationError, ok := err.(*ErrProgramsYamlValidation); ok {
		if !(validationError.Field == "Programs[taskmaster].Autorestart" && validationError.Issue == ValidationIssueUnexpectedValue
			t.Errorf(
				"Incorrect error: (%s, %s); expected (%s, %s)",
				validationError.Field,
				validationError.Issue,
				"Programs[taskmaster].Autorestart",
				ValidationIssueUnexpectedValue
			)
			return
		}
		return
	}

	t.Errorf("Returned invalid error")
}

func TestStarttimeSetToDefaultValue (t *testing.T) {
	programs := ProgramsYaml{
		Programs: {
			"taskmaster": {
				Cmd: "cmd",
				Numprocs: 10,
				Autostart: true,
				Autorestart: AutorestartOn,
				Starttime: nil,
			}
		},
	}

	programs.Validate()

	if starttime := programs.Programs["taskmaster"].Starttime; starttime != 1 {
		t.Errorf(
			"Starttime not set to correct default value: %v; expected %v",
			starttime,
			1,
		)
	}
}

func TestStarttimeIsNotOutsideLowerBounds (t *testing.T) {
	programs := ProgramsYaml{
		Programs: {
			"taskmaster": {
				Cmd: "cmd",
				Numprocs: 10,
				Autostart: true,
				Autorestart: AutorestartOn,
				Starttime: -1,
			}
		},
	}

	err := programs.Validate()
	if err != nil {
		t.Errorf("Validate should have returned an error")
	}

	if validationError, ok := err.(*ErrProgramsYamlValidation); ok {
		if !(validationError.Field == "Programs[taskmaster].Starttime" && validationError.Issue == ValidationIssueEmptyField) {
			t.Errorf(
				"Incorrect error: (%s, %s); expected (%s, %s)",
				validationError.Field,
				validationError.Issue,
				"Programs[taskmaster].Starttime",
				ValidationIssueValueOutsideBounds
			)
			return
		}
		return
	}

	t.Errorf("Returned invalid error")
}

func TestStarttimeIsNotOutsideUpperBounds (t *testing.T) {
	programs := ProgramsYaml{
		Programs: {
			"taskmaster": {
				Cmd: "cmd",
				Numprocs: 10,
				Autostart: true,
				Autorestart: AutorestartOn,
				Starttime: 100000,
			}
		},
	}

	err := programs.Validate()
	if err != nil {
		t.Errorf("Validate should have returned an error")
	}

	if validationError, ok := err.(*ErrProgramsYamlValidation); ok {
		if !(validationError.Field == "Programs[taskmaster].Starttime" && validationError.Issue == ValidationIssueEmptyField) {
			t.Errorf(
				"Incorrect error: (%s, %s); expected (%s, %s)",
				validationError.Field,
				validationError.Issue,
				"Programs[taskmaster].Starttime",
				ValidationIssueValueOutsideBounds
			)
			return
		}
		return
	}

	t.Errorf("Returned invalid error")
}

func TestProgramYamlValidation(t *testing.T) {
	programs := ProgramsYaml{
		Programs: {
			{
				Cmd: nil,
				Numprocs: ,
				Umask: ,
				Workingdir: ,
				Autostart: ,
				Autorestart: ,
				Exitcodes: ,
				Startretries: ,
				Starttime: ,
				Stopsignal: ,
				Stoptime: ,
				Stdout: ,
				Stderr: ,
				Env: ,
			}
		}
	}
}
