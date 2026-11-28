// Package version carries the four version numbers this build speaks.
//
// They move independently on purpose. The tool changes faster than the
// formats, and a user whose recipe gives a different result six months later
// needs to see which of the four moved. The log of a CI run is often the only
// record left by then.
package version

// Tool is the version of this binary. Only the owner raises it.
//
// Set to 0.2.0 by the owner on 2026-11-28 for the first release. It stays until
// the owner raises it again, so a build from a later commit says 0.2.0 as well.
// A release is a separate act and needs its own decision.
//
// A major version here is a statement about mutator identifiers and the
// manifest format, which is to say about files sitting in somebody else's test
// repository. Raising it is never housekeeping, which is why the next session
// does not get to do it either.
//
// Two copies of it live in the Windows resource scripts, cmd/qadg/qadg.rc and
// cmd/qadg-gui/qadg-gui.rc, because Windows shows a version in the properties
// of each .exe and a resource script cannot read this constant. Raising it
// means writing the new number in both scripts and compiling them again with
// the windres command each one carries. A test in internal/guard fails until
// that is done.
const Tool = "0.2.0"

// Catalog is the version of the corruption catalog this build carries. It
// mirrors catalog_version in spec/corruption-catalog.yaml, which is the
// canonical form. A test reads that file and fails when the two drift apart.
const Catalog = "0.1"

// Manifest is the version of the expectation manifest format this build
// writes. It mirrors spec_version in spec/expectation-manifest.md, checked by
// the same test.
//
// 0.2 added the list of entities with the rows each one had, without which a
// response record about two tables could not be held to either of them
// (DEC-266). A manifest written to 0.1 is still read.
const Manifest = "0.2"

// Recipe is the version of the recipe schema this build reads.
//
// Unlike the two above it has no specification file yet, so this constant is
// its only source. A test fails once such a file appears, so that the missing
// link gets wired instead of being discovered later.
const Recipe = "0.1"

// Response is the version of the application response format this build reads.
//
// It is the one format this tool only ever reads, so this constant is not part
// of Set: Set answers what this binary produces, and adding a member to it
// changes what every pipeline parses out of --version. The number is still
// needed here, because a reader that refuses a version it does not know has to
// know which one it does.
//
// 0.2 added the entity a row belongs to (DEC-266), and a record written to 0.1
// is read by the same rules, since leaving the entity out is allowed in both.
const Response = "0.2"

// Set is the whole answer to the question of what this binary speaks.
//
// The JSON names are what a pipeline parses, so they are part of the surface
// that scripts depend on.
type Set struct {
	Tool     string `json:"tool"`
	Catalog  string `json:"catalog"`
	Manifest string `json:"manifest_format"`
	Recipe   string `json:"recipe_schema"`
}

// Current returns the versions compiled into this build.
func Current() Set {
	return Set{
		Tool:     Tool,
		Catalog:  Catalog,
		Manifest: Manifest,
		Recipe:   Recipe,
	}
}
