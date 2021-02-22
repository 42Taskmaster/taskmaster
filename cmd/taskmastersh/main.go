package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/42Taskmaster/taskmaster/parser"
)

type Taskmastersh struct{}

type Command func(parser.ParsedCommand, Taskmastersh) error

var Commands = map[string]Command{
	"status":   StatusCommand,
	"start":    StartCommand,
	"stop":     StopCommand,
	"restart":  RestartCommand,
	"shutdown": ShutdownCommand,
	"quit":     QuitCommand,
}

func StatusCommand(parser.ParsedCommand, Taskmastersh) error {
	return nil
}

func StartCommand(parser.ParsedCommand, Taskmastersh) error {
	return nil
}

func StopCommand(parser.ParsedCommand, Taskmastersh) error {
	return nil
}

func RestartCommand(parser.ParsedCommand, Taskmastersh) error {
	return nil
}

func ShutdownCommand(parser.ParsedCommand, Taskmastersh) error {
	return nil
}

func QuitCommand(parser.ParsedCommand, Taskmastersh) error {
	return io.EOF
}

func printPrompt() {
	fmt.Printf("$> ")
}

func quitLabel() {
	fmt.Printf("Thank you, bye!")
}

func main() {
	fmt.Println("Welcome to taskmastersh")

	taskmastersh := Taskmastersh{}

	reader := bufio.NewReader(os.Stdin)

	for {
		printPrompt()

		input, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				quitLabel()
				return
			}

			log.Println(err)
			return
		}

		cmd, err := parser.ParseCommand(input)
		if err != nil {
			log.Println(err)
			return
		}

		commandToExecute, ok := Commands[cmd.Cmd]
		if !ok {
			fmt.Printf("command not found: %s\n", cmd.Cmd)
			continue
		}

		if err := commandToExecute(cmd, taskmastersh); err != nil {
			if errors.Is(err, io.EOF) {
				quitLabel()
				return
			}
		}
	}
}
