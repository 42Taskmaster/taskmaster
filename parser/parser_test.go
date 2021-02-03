package parser_test

import (
	"errors"
	"testing"

	"github.com/42Taskmaster/taskmaster/machine"
	"github.com/42Taskmaster/taskmaster/parser"
)

func StringSlicesEqual(a, b []string) bool {
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}

	return true
}

func TestParsesEmptyString(t *testing.T) {
	const in = ""

	cmd, err := parser.ParseCommand(in)
	if err != nil {
		t.Fatalf(
			"expected no error to be returned; received %v",
			err,
		)
	}

	var (
		expectedCmd  = ""
		expectedArgs = []string{}
	)
	if cmd.Cmd != expectedCmd {
		t.Fatalf(
			"parsed cmd is incorrect %v; expected %v",
			cmd.Cmd,
			expectedCmd,
		)
	}
	if !StringSlicesEqual(cmd.Args, expectedArgs) {
		t.Fatalf(
			"parsed args are incorrect %v; expected %v",
			cmd.Args,
			expectedArgs,
		)
	}
}

func TestParsesCommandWithOnlyWhitespaces(t *testing.T) {
	const in = "         \t \n    \n   \t   "

	cmd, err := parser.ParseCommand(in)
	if err != nil {
		t.Fatalf(
			"expected no error to be returned; received %v",
			err,
		)
	}

	var (
		expectedCmd  = ""
		expectedArgs = []string{}
	)
	if cmd.Cmd != expectedCmd {
		t.Fatalf(
			"parsed cmd is incorrect %v; expected %v",
			cmd.Cmd,
			expectedCmd,
		)
	}
	if !StringSlicesEqual(cmd.Args, expectedArgs) {
		t.Fatalf(
			"parsed args are incorrect %v; expected %v",
			cmd.Args,
			expectedArgs,
		)
	}
}

func TestParsesSingleWord(t *testing.T) {
	const in = "ls"

	cmd, err := parser.ParseCommand(in)
	if err != nil {
		t.Fatalf(
			"expected no error to be returned; received %v",
			err,
		)
	}

	var (
		expectedCmd  = "ls"
		expectedArgs = []string{}
	)
	if cmd.Cmd != expectedCmd {
		t.Fatalf(
			"parsed cmd is incorrect %v; expected %v",
			cmd.Cmd,
			expectedCmd,
		)
	}
	if !StringSlicesEqual(cmd.Args, expectedArgs) {
		t.Fatalf(
			"parsed args are incorrect %v; expected %v",
			cmd.Args,
			expectedArgs,
		)
	}
}

func TestParsesTwoWords(t *testing.T) {
	const in = "ls aa"

	cmd, err := parser.ParseCommand(in)
	if err != nil {
		t.Fatalf(
			"expected no error to be returned; received %v",
			err,
		)
	}

	var (
		expectedCmd  = "ls"
		expectedArgs = []string{
			"aa",
		}
	)
	if cmd.Cmd != expectedCmd {
		t.Fatalf(
			"parsed cmd is incorrect %v; expected %v",
			cmd.Cmd,
			expectedCmd,
		)
	}
	if !StringSlicesEqual(cmd.Args, expectedArgs) {
		t.Fatalf(
			"parsed args are incorrect %v; expected %v",
			cmd.Args,
			expectedArgs,
		)
	}
}

func TestParsesTwoWordsWithSeveralSeparatingWhitespaces(t *testing.T) {
	const in = "ls       \t       aa"

	cmd, err := parser.ParseCommand(in)
	if err != nil {
		t.Fatalf(
			"expected no error to be returned; received %v",
			err,
		)
	}

	var (
		expectedCmd  = "ls"
		expectedArgs = []string{
			"aa",
		}
	)
	if cmd.Cmd != expectedCmd {
		t.Fatalf(
			"parsed cmd is incorrect %v; expected %v",
			cmd.Cmd,
			expectedCmd,
		)
	}
	if !StringSlicesEqual(cmd.Args, expectedArgs) {
		t.Fatalf(
			"parsed args are incorrect %v; expected %v",
			cmd.Args,
			expectedArgs,
		)
	}
}

func TestParsesWordsWithLeadingAndEndingWhitespaces(t *testing.T) {
	const in = "    ls  a  \t"

	cmd, err := parser.ParseCommand(in)
	if err != nil {
		t.Fatalf(
			"expected no error to be returned; received %v",
			err,
		)
	}

	var (
		expectedCmd  = "ls"
		expectedArgs = []string{
			"a",
		}
	)
	if cmd.Cmd != expectedCmd {
		t.Fatalf(
			"parsed cmd is incorrect %v; expected %v",
			cmd.Cmd,
			expectedCmd,
		)
	}
	if !StringSlicesEqual(cmd.Args, expectedArgs) {
		t.Fatalf(
			"parsed args are incorrect %v; expected %v",
			cmd.Args,
			expectedArgs,
		)
	}
}

func TestParsesSingleQuotes(t *testing.T) {
	const in = `ls 'salut toi !'`

	cmd, err := parser.ParseCommand(in)
	if err != nil {
		t.Fatalf(
			"expected no error to be returned; received %v",
			err,
		)
	}

	var (
		expectedCmd  = "ls"
		expectedArgs = []string{
			"salut toi !",
		}
	)
	if cmd.Cmd != expectedCmd {
		t.Fatalf(
			"parsed cmd is incorrect %v; expected %v",
			cmd.Cmd,
			expectedCmd,
		)
	}
	if !StringSlicesEqual(cmd.Args, expectedArgs) {
		t.Fatalf(
			"parsed args are incorrect %v; expected %v",
			cmd.Args,
			expectedArgs,
		)
	}
}

func TestParsesThrowsOnUncompleteSingleQuoting(t *testing.T) {
	const in = `ls 'hi !`

	_, err := parser.ParseCommand(in)
	if err == nil {
		t.Fatalf(
			"expected an error to be returned; no error returned",
		)
	}

	if !errors.Is(err, machine.ErrInvalidTransitionNotImplemented) {
		t.Fatalf(
			"expected a state machine error caused by a not implemented transition; received %v",
			err,
		)
	}
}

func TestParsesSuccessiveSingleQuotes(t *testing.T) {
	const in = `ls 'salut toi !''et oui toi !'`

	cmd, err := parser.ParseCommand(in)
	if err != nil {
		t.Fatalf(
			"expected no error to be returned; received %v",
			err,
		)
	}

	var (
		expectedCmd  = "ls"
		expectedArgs = []string{
			"salut toi !et oui toi !",
		}
	)
	if cmd.Cmd != expectedCmd {
		t.Fatalf(
			"parsed cmd is incorrect %v; expected %v",
			cmd.Cmd,
			expectedCmd,
		)
	}
	if !StringSlicesEqual(cmd.Args, expectedArgs) {
		t.Fatalf(
			"parsed args are incorrect %v; expected %v",
			cmd.Args,
			expectedArgs,
		)
	}
}

func TestParsesDoubleQuotes(t *testing.T) {
	const in = `ls "salut toi !"`

	cmd, err := parser.ParseCommand(in)
	if err != nil {
		t.Fatalf(
			"expected no error to be returned; received %v",
			err,
		)
	}

	var (
		expectedCmd  = "ls"
		expectedArgs = []string{
			"salut toi !",
		}
	)
	if cmd.Cmd != expectedCmd {
		t.Fatalf(
			"parsed cmd is incorrect %v; expected %v",
			cmd.Cmd,
			expectedCmd,
		)
	}
	if !StringSlicesEqual(cmd.Args, expectedArgs) {
		t.Fatalf(
			"parsed args are incorrect %v; expected %v",
			cmd.Args,
			expectedArgs,
		)
	}
}

func TestParsesThrowsOnUncompleteDoubleQuoting(t *testing.T) {
	const in = `ls "hi !`

	_, err := parser.ParseCommand(in)
	if err == nil {
		t.Fatalf(
			"expected an error to be returned; no error returned",
		)
	}

	if !errors.Is(err, machine.ErrInvalidTransitionNotImplemented) {
		t.Fatalf(
			"expected a state machine error caused by a not implemented transition; received %v",
			err,
		)
	}
}

func TestIgnoresNotPrintableCharacters(t *testing.T) {
	const in = "ls \x02\x03\x04\x7f"

	cmd, err := parser.ParseCommand(in)
	if err != nil {
		t.Fatalf(
			"expected no error to be returned; received %v",
			err,
		)
	}

	var (
		expectedCmd  = "ls"
		expectedArgs = []string{}
	)
	if cmd.Cmd != expectedCmd {
		t.Fatalf(
			"parsed cmd is incorrect %v; expected %v",
			cmd.Cmd,
			expectedCmd,
		)
	}
	if !StringSlicesEqual(cmd.Args, expectedArgs) {
		t.Fatalf(
			"parsed args are incorrect %v; expected %v",
			cmd.Args,
			expectedArgs,
		)
	}
}

func TestParsesWtfSuccessiveDoubleAndSingleQuoting(t *testing.T) {
	const in = `ls "salut toi !"'yolo   yolo"lol"'"et oui toi !"`

	cmd, err := parser.ParseCommand(in)
	if err != nil {
		t.Fatalf(
			"expected no error to be returned; received %v",
			err,
		)
	}

	var (
		expectedCmd  = "ls"
		expectedArgs = []string{
			`salut toi !yolo   yolo"lol"et oui toi !`,
		}
	)
	if cmd.Cmd != expectedCmd {
		t.Fatalf(
			"parsed cmd is incorrect %v; expected %v",
			cmd.Cmd,
			expectedCmd,
		)
	}
	if !StringSlicesEqual(cmd.Args, expectedArgs) {
		t.Fatalf(
			"parsed args are incorrect %v; expected %v",
			cmd.Args,
			expectedArgs,
		)
	}
}

func TestParsesSuccessiveDoubleQuotes(t *testing.T) {
	const in = `ls "salut toi !""et oui toi !"`

	cmd, err := parser.ParseCommand(in)
	if err != nil {
		t.Fatalf(
			"expected no error to be returned; received %v",
			err,
		)
	}

	var (
		expectedCmd  = "ls"
		expectedArgs = []string{
			"salut toi !et oui toi !",
		}
	)
	if cmd.Cmd != expectedCmd {
		t.Fatalf(
			"parsed cmd is incorrect %v; expected %v",
			cmd.Cmd,
			expectedCmd,
		)
	}
	if !StringSlicesEqual(cmd.Args, expectedArgs) {
		t.Fatalf(
			"parsed args are incorrect %v; expected %v",
			cmd.Args,
			expectedArgs,
		)
	}
}
