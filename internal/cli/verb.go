package cli

import (
	"flag"
	"io"
)

// verb is one command of the tool, as the dispatcher runs it and as a guard
// asks it what it accepts.
//
// # Why one table rather than six openings (B-360)
//
// Every verb used to open with the same lines: a set of flags, the flags of the
// tool declared again so that they mean the same after the verb as before it,
// its own flags, --help, and the parse. A guard asking which flags a verb takes
// could only build the verbs it had been told about, from a list written by
// hand, and on 2026-11-03 that list had left out quick. A flag of a new name
// given to a verb off the list still passed every test after that. The set a
// verb parses is now built in one place, and the guard asks this table, which
// is the table the dispatcher runs: a flag declared anywhere else is never
// parsed, so it cannot work without the guard seeing it.
type verb struct {
	// run reads the arguments after the name of the verb and does the work.
	run func(out *streams, arguments []string, global crossCutting) int
	// flags builds the set run parses, for a caller asking what the verb
	// accepts without running it.
	flags func() *flag.FlagSet
}

// verbs is every command this build has, by the name typed to run it.
var verbs = map[string]verb{
	commandValidate: verbOf(commandValidate, noOwnFlags, validateWith, nil),
	commandGenerate: verbOf(commandGenerate, declareGenerate, generateWith, nil),
	commandCorrupt:  verbOf(commandCorrupt, declareCorrupt, corruptWith, nil),
	commandVerify:   verbOf(commandVerify, noOwnFlags, verifyWith, nil),
	commandQuick:    verbOf(commandQuick, declareQuick, quickWith, nil),
	commandHelp:     verbOf(commandHelp, noOwnFlags, helpWith, helpInsteadOf),
}

// parsed is what a verb is handed once its arguments have been read.
type parsed[F any] struct {
	// set is the set the arguments were read into, for a verb that asks which
	// flags were given rather than only what they hold.
	set *flag.FlagSet
	// rest is the arguments that were not flags, in the order they were given.
	rest []string
	// own is the flags of the verb's own.
	own F
	// flags is the flags of the tool, given before the verb or after it.
	flags crossCutting
}

// noFlags is what a verb taking only the flags of the tool declares.
type noFlags struct{}

func noOwnFlags(*flag.FlagSet) noFlags { return noFlags{} }

// verbOf makes a verb out of the declaration of its own flags and the work it
// does with them.
//
// The flags of the tool are declared again in every set, so that they mean the
// same after the verb as before it, and --help after any verb asks about that
// verb. A flag the set does not define is refused in the four parts, and
// instead says what to type rather than that flag. Left nil it points at the
// help of the verb, which is what every verb but help says.
func verbOf[F any](name string, declare func(*flag.FlagSet) F,
	do func(*streams, parsed[F]) int, instead func(*flag.FlagSet, []string) string) verb {
	build := func() (*flag.FlagSet, crossCutting, F, func() bool) {
		set := flag.NewFlagSet(name, flag.ContinueOnError)
		set.SetOutput(io.Discard)
		local := declareCrossCutting(set)
		own := declare(set)
		askedForHelp := declareVerbHelp(set)
		return set, local, own, askedForHelp
	}
	return verb{
		flags: func() *flag.FlagSet {
			set, _, _, _ := build()
			return set
		},
		run: func(out *streams, arguments []string, global crossCutting) int {
			set, local, own, askedForHelp := build()
			rest, err := parseAnywhere(set, arguments)
			if err != nil {
				typeThis := problemHelpSubjectDo + " " + name
				if instead != nil {
					typeThis = instead(set, arguments)
				}
				return usage(out, flagFailure(err, acceptedFlagsOf(name), typeThis))
			}
			if askedForHelp() {
				return printCommandHelp(out, name)
			}
			flags := global.merge(local)
			if code, ok := applyVerbosity(out, flags); !ok {
				return code
			}
			return do(out, parsed[F]{set: set, rest: rest, own: own, flags: flags})
		},
	}
}
