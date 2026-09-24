package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"
)

// wrongInvocation is the exit code of a flag this binary does not know, the
// same number the command line gives a wrong invocation (docs/09).
const wrongInvocation = 2

// launch is what the command line asked the window for.
type launch struct {
	// catalogue opens the catalogue of controls instead of the window
	// proper (DEC-253).
	catalogue bool
	// help is a request for the list of flags, which is written out and
	// answered with no window.
	help bool
}

// arguments reads the command line of the window.
//
// # 🔴 Why a flag it does not know is refused rather than ignored (DEC-253, 7)
//
// The window has one flag, and the likeliest way to get it wrong is a
// spelling: -catalog for -catalogue. Ignored, that opens the window proper
// without a word, and a person who asked for the catalogue looks at the
// screens and concludes the flag does nothing. Refused, it says what went
// wrong on the error channel and opens nothing, the way the command line
// answers the same mistake. A word left over after the flags is refused for
// the same reason, since this window takes no file on its command line.
//
// The list of what is allowed is read from the flags themselves, so a flag
// added later is named in the refusal without anybody writing it twice.
//
// # 🔴 Why -psn_ is not let through, measured rather than remembered
//
// Older macOS handed a program started from the Finder an argument -psn_ with
// the number of its process, and a window refusing unknown flags would then
// refuse to open from the Finder. DEC-253 made the refusal wait for a
// measurement. Taken on 2026-09-25 on macOS 27.0 (arm64): a bundle started
// through LaunchServices, with open and with open -n, was handed no argument
// at all, three launches out of three, exactly like a start from a shell. What
// was NOT measured: a double click in the Finder itself, and the first start
// of a download still carrying the quarantine attribute, which is where the
// argument was last reported. That is B-316, before the first build for macOS
// goes to anybody.
func arguments(args []string, out io.Writer) (launch, error) {
	var asked launch
	flags := flag.NewFlagSet("qadg-gui", flag.ContinueOnError)
	// The toolkit of flags writes its own complaint and its own usage when a
	// parse fails. Both are replaced by the refusal below, which says the four
	// things a refusal here says.
	flags.SetOutput(io.Discard)
	flags.BoolVar(&asked.catalogue, "catalogue", false,
		"open the catalogue of controls, every control of the window in every state, instead of the window")
	err := flags.Parse(args)
	switch {
	case errors.Is(err, flag.ErrHelp):
		asked.help = true
		fmt.Fprintln(out, "Usage: qadg-gui [flags]")
		fmt.Fprintln(out, "Opens the window. The flags are:")
		flags.SetOutput(out)
		flags.PrintDefaults()
		return asked, nil
	case err != nil:
		return asked, refusal(err.Error(), "this build of the window does not define that flag", flags)
	case flags.NArg() > 0:
		return asked, refusal("unexpected argument: "+flags.Arg(0),
			"the window takes flags and nothing else on its command line", flags)
	}
	return asked, nil
}

// refusal says what went wrong, why, what is allowed and what to do instead,
// in the shape the command line gives the same four things.
func refusal(what, why string, flags *flag.FlagSet) error {
	var allowed []string
	flags.VisitAll(func(each *flag.Flag) { allowed = append(allowed, "-"+each.Name) })
	allowed = append(allowed, "-h")
	sort.Strings(allowed)
	return errors.New(strings.Join([]string{
		"error: " + what,
		"       " + why,
		"       allowed: " + strings.Join(allowed, ", "),
		"       try: qadg-gui -h lists every flag the window accepts",
	}, "\n"))
}
