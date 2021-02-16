package parser

import (
	"fmt"
	"unicode"

	"github.com/42Taskmaster/taskmaster/machine"
)

const (
	parserStateWhitespace         machine.StateType = "WHITESPACE"
	parserStateInWord             machine.StateType = "IN_WORD"
	parserStateInWordDoubleQuoted machine.StateType = "IN_WORD_DOUBLE_QUOTED"
	parserStateInWordSingleQuoted machine.StateType = "IN_WORD_SINGLE_QUOTED"
	parserStateWaiting            machine.StateType = "WAITING"
	parserStateEnd                machine.StateType = "END"
)

const (
	parserEventWhitespace machine.EventType = "whitespace"
	parserEventCharacter  machine.EventType = "in-word"

	parserEventDoubleQuote machine.EventType = "double-quote"

	parserEventSingleQuote machine.EventType = "single-quote"
	parserEventEnd         machine.EventType = "end"
)

// ErrParsing holds the identifier that caused a problem
// and an error describing the issue.
type ErrParsing struct {
	Identifier string
	Err        error
}

func (err *ErrParsing) Error() string {
	return fmt.Sprintf(
		"parsing error for character: %s: %s",
		err.Identifier,
		err.Err.Error(),
	)
}

func (err *ErrParsing) Unwrap() error {
	return err.Err
}

type parserContext struct {
	In          string
	CurrentChar *rune

	ProcessedChunks *[]string
	CurrentChunk    string
}

// ParsedCommand holds the command to execute and the arguments to
// pass to it.
type ParsedCommand struct {
	Cmd  string
	Args []string
}

// ParseCommand takes a raw command line and returns what is
// the command to execute and what are the arguments to pass
// to the command.
func ParseCommand(cmd string) (ParsedCommand, error) {
	var (
		chunks []string
		char   rune
	)

	parserMachine := machine.Machine{
		Context: &parserContext{
			In:          cmd,
			CurrentChar: &char,

			ProcessedChunks: &chunks,
		},

		Initial: parserStateWhitespace,

		StateNodes: machine.StateNodes{
			parserStateWhitespace: machine.StateNode{
				Actions: []machine.Action{
					parserStateWhitespaceAction,
				},

				On: machine.Events{
					parserEventWhitespace:  parserStateWhitespace,
					parserEventCharacter:   parserStateInWord,
					parserEventDoubleQuote: parserStateInWordDoubleQuoted,
					parserEventSingleQuote: parserStateInWordSingleQuoted,
					parserEventEnd:         parserStateEnd,
				},
			},

			parserStateInWord: machine.StateNode{
				Actions: []machine.Action{
					parserStateInWordAction,
				},

				On: machine.Events{
					parserEventWhitespace:  parserStateWhitespace,
					parserEventCharacter:   parserStateInWord,
					parserEventDoubleQuote: parserStateInWordDoubleQuoted,
					parserEventSingleQuote: parserStateInWordSingleQuoted,
					parserEventEnd:         parserStateEnd,
				},
			},

			parserStateInWordDoubleQuoted: machine.StateNode{
				Actions: []machine.Action{
					parserStateInWordDoubleQuotedAction,
				},

				On: machine.Events{
					parserEventWhitespace:  parserStateInWordDoubleQuoted,
					parserEventCharacter:   parserStateInWordDoubleQuoted,
					parserEventSingleQuote: parserStateInWordDoubleQuoted,

					parserEventDoubleQuote: parserStateWaiting,
				},
			},

			parserStateInWordSingleQuoted: machine.StateNode{
				Actions: []machine.Action{
					parserStateInWordSingleQuotedAction,
				},

				On: machine.Events{
					parserEventWhitespace:  parserStateInWordSingleQuoted,
					parserEventCharacter:   parserStateInWordSingleQuoted,
					parserEventDoubleQuote: parserStateInWordSingleQuoted,

					parserEventSingleQuote: parserStateWaiting,
				},
			},

			parserStateWaiting: machine.StateNode{
				On: machine.Events{
					parserEventWhitespace:  parserStateWhitespace,
					parserEventCharacter:   parserStateInWord,
					parserEventDoubleQuote: parserStateInWordDoubleQuoted,
					parserEventSingleQuote: parserStateInWordSingleQuoted,
					parserEventEnd:         parserStateEnd,
				},
			},

			parserStateEnd: machine.StateNode{
				Actions: []machine.Action{
					parserStateEndAction,
				},
			},
		},
	}
	parserMachine.Init()

	for _, char = range cmd {
		var event machine.EventType

		switch {
		case unicode.IsSpace(char):
			event = parserEventWhitespace
		case char == '"':
			event = parserEventDoubleQuote
		case char == '\'':
			event = parserEventSingleQuote
		case unicode.IsPrint(char):
			event = parserEventCharacter
		default:
			continue
		}

		_, err := parserMachine.Send(event)
		if err != nil {
			return ParsedCommand{}, &ErrParsing{
				Identifier: string(char),
				Err:        err,
			}
		}
	}
	_, err := parserMachine.Send(parserEventEnd)
	if err != nil {
		return ParsedCommand{}, &ErrParsing{
			Identifier: "EOF",
			Err:        err,
		}
	}

	if len(chunks) == 0 {
		return ParsedCommand{}, nil
	}

	return ParsedCommand{
		Cmd:  chunks[0],
		Args: chunks[1:],
	}, nil
}

func parserStateWhitespaceAction(stateMachine *machine.Machine, context machine.Context) (machine.EventType, error) {
	ctx := context.(*parserContext)

	if ctx.CurrentChunk == "" {
		return machine.NoopEvent, nil
	}

	*ctx.ProcessedChunks = append(*ctx.ProcessedChunks, ctx.CurrentChunk)
	ctx.CurrentChunk = ""

	return machine.NoopEvent, nil
}

func parserStateInWordAction(stateMachine *machine.Machine, context machine.Context) (machine.EventType, error) {
	ctx := context.(*parserContext)

	ctx.CurrentChunk += string(*ctx.CurrentChar)

	return machine.NoopEvent, nil
}

func parserStateInWordDoubleQuotedAction(stateMachine *machine.Machine, context machine.Context) (machine.EventType, error) {
	ctx := context.(*parserContext)

	if *ctx.CurrentChar == '"' {
		return machine.NoopEvent, nil
	}

	ctx.CurrentChunk += string(*ctx.CurrentChar)

	return machine.NoopEvent, nil
}

func parserStateInWordSingleQuotedAction(stateMachine *machine.Machine, context machine.Context) (machine.EventType, error) {
	ctx := context.(*parserContext)

	if *ctx.CurrentChar == '\'' {
		return machine.NoopEvent, nil
	}

	ctx.CurrentChunk += string(*ctx.CurrentChar)

	return machine.NoopEvent, nil
}

func parserStateEndAction(stateMachine *machine.Machine, context machine.Context) (machine.EventType, error) {
	ctx := context.(*parserContext)

	if ctx.CurrentChunk == "" {
		return machine.NoopEvent, nil
	}

	*ctx.ProcessedChunks = append(*ctx.ProcessedChunks, ctx.CurrentChunk)
	ctx.CurrentChunk = ""

	return machine.NoopEvent, nil
}
