package cli

import (
	"flag"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// Everything about answering the question "how do I use this".
//
// Split out of run.go on 2026-08-05, when that file went over the hard ceiling
// on file length. The ceiling may only be lowered, so the answer is to divide
// rather than to raise it, and help is the part that divides cleanly: it prints
// and decides nothing.

// runHelp answers the most common thing somebody types.
//
// With nothing after it, it is --help. With a command after it, it is the help
// of that command, which is the same text that command prints for --help. One
// renderer rather than two, because two would answer the same question
// differently the first time somebody edited one of them.
func runHelp(out *streams, arguments []string) int {
	switch {
	case len(arguments) == 0:
		printHelp(out)
		return finish(out, codeOK)
	case len(arguments) > 1:
		return usage(out, problem{
			what:    problemHelpArguments,
			why:     problemHelpArgumentsWhy,
			allowed: problemHelpArgumentsAllow,
			instead: problemHelpSubjectDo + " " + arguments[0],
		})
	}

	doc, found := commandDocFor(arguments[0])
	if !found {
		refusal := unknownCommand(arguments[0])
		refusal.what = problemHelpSubject + arguments[0]
		refusal.why = problemHelpSubjectWhy
		return usage(out, refusal)
	}
	out.emit(renderCommandHelp(doc))
	return finish(out, codeOK)
}

// declareVerbHelp gives a command the two spellings of the help flag and
// returns the question of whether either was given.
//
// A verb has to declare them itself. The parser of the tool stops at the first
// argument that is not a flag, so anything after the verb belongs to the verb,
// and without this qadg generate --help would be refused as an unknown flag.
func declareVerbHelp(set *flag.FlagSet) func() bool {
	long := set.Bool("help", false, "")
	short := set.Bool("h", false, "")
	return func() bool { return *long || *short }
}

// printCommandHelp prints what one verb says about itself.
func printCommandHelp(out *streams, name string) int {
	doc, found := commandDocFor(name)
	if !found {
		// Unreachable while every verb the dispatcher accepts is in the
		// register, which a test asserts. Answering rather than printing
		// nothing keeps a future verb from failing silently.
		return fail(out, fault.New(fault.Runtime, "no help for the command "+name))
	}
	out.emit(renderCommandHelp(doc))
	return finish(out, codeOK)
}

func printHelp(out *streams) {
	blocks := []string{
		summaryLine,
		examplesBlock,
		usageBlock,
		renderCommands(),
		renderFlags(),
		channelsBlock,
		exitCodesHeading + "\n" + renderExitCodes(),
	}
	out.emit(strings.Join(blocks, "\n\n"))
}
