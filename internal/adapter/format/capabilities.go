package format

import (
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
)

// CapabilitiesOf is what a writer brings into a run.
//
// # Why the byte level ones are not here
//
// The writers own the bytes, so it would be easy to claim every output
// capability for them. They do not have them. Writing a different character
// set, a different line ending, a byte at a chosen offset or a stray byte order
// mark are four things a writer would have to offer on purpose, and none of
// them exists yet: the CSV writer chooses its own line ending precisely so that
// the bytes are a promise rather than a platform detail.
//
// Claiming them would turn a manifest entry from "this run cannot control the
// line endings" into "this build has no implementation", and the first is the
// true one. The names are here in a comment rather than in the list so that
// whoever adds the ability adds the capability in the same commit.
func CapabilitiesOf(format string) []capability.Name {
	switch format {
	case NameJSON:
		return []capability.Name{capability.FormatJSON}
	case NameSQL:
		// The SQL form brings nothing an entry of the catalog asks for. Damage
		// to the statements themselves, such as a quote left open, has no entry
		// yet, and the catalog defines only the capabilities its entries need,
		// so a name for this one would be a definition nothing uses.
		return nil
	default:
		// The comma separated form is what a recipe gets when it asks for
		// nothing, so it answers for the empty name too.
		return []capability.Name{capability.FormatCSV}
	}
}

// The names a recipe uses for the forms this build writes.
//
// They are the names the reader of recipes accepts, taken from the one list
// both read (dataset.Forms) rather than spelled again here. A name is how a
// user asks for one of these writers, and a second spelling is a copy that
// parts company the first time a form is added.
const (
	NameCSV  = dataset.FormCSV
	NameJSON = dataset.FormJSON
	NameSQL  = dataset.FormSQL
)
