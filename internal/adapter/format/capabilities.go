package format

import "github.com/donislawdev/qadatagenerator/internal/domain/capability"

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
	default:
		// The comma separated form is what a recipe gets when it asks for
		// nothing, so it answers for the empty name too.
		return []capability.Name{capability.FormatCSV}
	}
}

// The names a recipe uses for the two forms this build writes.
//
// They live here, beside the writers, rather than in the surface that reads a
// recipe. A name is how a user asks for one of these writers, and a second copy
// in another package is a copy that parts company the first time a format is
// added.
const (
	NameCSV  = "csv"
	NameJSON = "json"
)
