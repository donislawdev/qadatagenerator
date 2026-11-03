package cli

import (
	"flag"
	"strings"
)

// parseAnywhere reads the flags of one verb wherever they stand among its
// arguments, and returns the arguments that are not flags in the order typed.
//
// # Why the flag package is not simply handed the arguments
//
// It stops at the first argument that is not a flag. Every verb but one parsed
// that way until 2026-11-03, so qadg generate recipe.yaml --out ./data was read
// as two recipes, the second named --out, and refused with a suggestion to type
// the command that had just been typed (B-97, DEC-272). Nothing after the path
// was read as a flag at all, so a misspelt flag or --help written there was
// never answered either. The one verb that worked took its first argument off
// the front by hand, which fixed that verb and left the other four as they were.
//
// # Why the split asks the flag set rather than the shape of the text
//
// A flag that takes a value takes the next argument whatever it looks like, so
// qadg generate --out data recipe.yaml has one path and not two. Only the set
// knows which of its flags are switches, so the split asks it, the way the flag
// package itself does. A flag the set does not define stays with the flags and
// is refused by the parse below, before anything is said about the paths, so a
// misspelt flag is reported as one rather than as a path too many.
//
// # Why the paths are not handed to the parser after a "--"
//
// A flag that takes a value and is written last, with nothing after it, would
// take that "--" as its value. --out with no directory would then write into a
// directory named "--" rather than being refused as a flag missing its value.
//
// "--" still ends the flags, as the flag package documents, so a file whose
// name begins with a dash is reached by writing "--" in front of it.
func parseAnywhere(set *flag.FlagSet, arguments []string) ([]string, error) {
	var flags, paths []string
	for at := 0; at < len(arguments); at++ {
		argument := arguments[at]
		if argument == "--" {
			paths = append(paths, arguments[at+1:]...)
			break
		}
		if !looksLikeFlag(argument) {
			paths = append(paths, argument)
			continue
		}
		flags = append(flags, argument)
		if takesNextArgument(set, argument) && at+1 < len(arguments) {
			at++
			flags = append(flags, arguments[at])
		}
	}
	return paths, set.Parse(flags)
}

// looksLikeFlag is the test the flag package applies: a dash and something
// after it. A dash on its own is an argument, as it is there.
func looksLikeFlag(argument string) bool {
	return len(argument) > 1 && argument[0] == '-'
}

// takesNextArgument says whether a flag takes the argument after it as its
// value. A switch takes none, and a flag the set does not define is left for
// the parse to refuse. One written with an equals sign is not looked up apart
// from its value, so no flag is found under that name and it takes nothing
// either, which is right, because it carries its value already.
func takesNextArgument(set *flag.FlagSet, argument string) bool {
	defined := set.Lookup(strings.TrimPrefix(argument[1:], "-"))
	if defined == nil {
		return false
	}
	switched, isSwitch := defined.Value.(interface{ IsBoolFlag() bool })
	return !isSwitch || !switched.IsBoolFlag()
}
