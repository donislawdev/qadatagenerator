// Package version carries the four version numbers this build speaks.
//
// They move independently on purpose. The tool changes faster than the
// formats, and a user whose recipe gives a different result six months later
// needs to see which of the four moved. The log of a CI run is often the only
// record left by then.
package version

// Tool is the version of this binary. Only the owner raises it.
//
// Set to 0.1.0 by the owner on 2026-08-02. Nothing has been released yet, and
// this number does not claim otherwise: it says which version is being built,
// not which version somebody can download. A release is a separate act and
// needs its own decision.
//
// A major version here is a statement about mutator identifiers and the
// manifest format, which is to say about files sitting in somebody else's test
// repository. Raising it is never housekeeping, which is why the next session
// does not get to do it either.
const Tool = "0.1.0"

// Catalog is the version of the corruption catalog this build carries. It
// mirrors catalog_version in spec/corruption-catalog.yaml, which is the
// canonical form. A test reads that file and fails when the two drift apart.
const Catalog = "0.1"

// Manifest is the version of the expectation manifest format this build
// writes. It mirrors spec_version in spec/expectation-manifest.md, checked by
// the same test.
const Manifest = "0.1"

// Recipe is the version of the recipe schema this build reads.
//
// Unlike the two above it has no specification file yet, so this constant is
// its only source. A test fails once such a file appears, so that the missing
// link gets wired instead of being discovered later.
const Recipe = "0.1"

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
