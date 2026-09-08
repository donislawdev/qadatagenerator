package cli

import (
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/commandline"
)

// The window writes command lines out of names, and these are those names.
//
// # Why this guard is here rather than beside them
//
// internal/commandline has no dependencies on purpose, so it cannot check
// itself against the register it has to agree with. This register is the one
// the help text is built from, and it lives here. A test may look at both, and
// this is the only place that can.
//
// # What goes wrong without it
//
// The window prints a line for a person to paste. A flag renamed here and not
// there gives them a line that looks right, reads right, and is refused by the
// tool it names. That is worse than no line at all, because it is the window
// telling somebody something untrue about its own product.
func TestEveryNameTheWindowPrintsIsOneTheCommandLineTakes(t *testing.T) {
	verbs := make(map[string]bool, len(commandDocs))
	flags := make(map[string]bool)
	for _, doc := range commandDocs {
		verbs[doc.name] = true
		for _, flag := range doc.flags {
			for _, name := range flag.names {
				flags[name] = true
			}
		}
	}
	for _, flag := range flagDocs {
		for _, name := range flag.names {
			flags[name] = true
		}
	}

	for _, verb := range commandline.Verbs {
		if !verbs[verb] {
			t.Errorf("the window writes lines beginning %q and this tool has no such verb, so a "+
				"person pasting one is refused by the thing the window named", verb)
		}
	}
	for _, flag := range commandline.Flags {
		if !flags[flag] {
			t.Errorf("the window writes %q into the lines it shows and no verb takes it, so the "+
				"line looks right and is refused", flag)
		}
	}
}
