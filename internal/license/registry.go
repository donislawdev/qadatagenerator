// Package license answers the question that the --license flag asks: what is
// inside this binary and under what terms.
//
// The answer has to come from the binary rather than from the repository. A
// license of the LGPL family binds whoever receives the executable, not
// whoever clones the source, and the legal department that blocks adoption
// reads neither. It will run a flag if somebody shows it one.
package license

// Code is the license of everything this project compiles.
//
// Only version 3, without the usual "or any later version" clause. The clause
// hands a future version of the license authority over this code, which is a
// choice the owner has to make rather than a default to inherit.
const Code = "GPL-3.0-only"

// Spec is the license of spec/ and of the corruption catalog entries carried
// inside this binary.
//
// It differs from the code license on purpose. A specification meant to be
// implemented by other tools cannot sit under copyleft, and this one is also a
// database, which the code license says nothing about.
const Spec = "CC-BY-4.0"

// SpecScope says which part of the product the specification license covers.
// Without it the two names above look like a contradiction.
const SpecScope = "Applies to spec/ and to the corruption catalog entries built into this binary."

// GeneratedDataNotice is the sentence that removes the most common reason to
// reject this tool: the belief that a copyleft generator makes its output
// copyleft too.
const GeneratedDataNotice = "Data generated with this tool belongs to you. It is not a derived work, and the license of this tool does not cover it."

// SyntheticDataNotice says what everything this tool produces is, and what it
// is not.
//
// # Why this ships in the binary rather than only in the README
//
// A national identifier or a bank routing number with a correct check digit is
// indistinguishable from a real one, because that is the whole point of
// producing it. Several of the schemes here have no block that is never issued,
// so a correct value cannot be shown not to be somebody's. The defence this
// project chose is not avoidance, which is impossible, but saying so plainly
// wherever somebody might look.
//
// A README does not travel with a file of data. A flag on the binary does, and
// so does the notice a dataset carries in its own metadata. This constant is the
// one sentence all of those say.
const SyntheticDataNotice = "Every value this tool produces is synthetic and is generated for testing. " +
	"Names, addresses, identifiers and account numbers do not describe anybody. " +
	"Where a value resembles a real person, company or institution, that is a coincidence " +
	"of arithmetic rather than a record of anything, and it must not be treated as one."

// ExampleDataNotice covers the values written into this repository rather than
// produced by it: the examples in the corruption catalog and the values in
// tests.
//
// They are held to the same rule for the same reason, and to one extra rule that
// only applies to a repository. A file in a public repository is published, so
// the fewest possible correct identifiers are written into one, and those that
// are have a reason recorded beside them.
const ExampleDataNotice = "Example values in the catalog and in the tests are illustrative and synthetic. " +
	"They are chosen to show the shape of a defect, not to describe any real subject."

// DataSource is one collection of reference data compiled into this binary.
//
// It is kept apart from Component because the two bind for different reasons.
// A module is code this binary links. A data source is a collection somebody
// else assembled, which was read and written down here, and keeping their
// notice with it is the condition under which that was allowed. Reading a list
// and keeping it takes a substantial part of a collection whether it arrived
// through a function call or through a file, and neither route makes the data
// this project's own work.
type DataSource struct {
	// Corpus is the list it produced, as locale and name.
	Corpus string
	// Project is where it was read from.
	Project string
	// Version is the exact tag or vintage it was read at, so the snapshot can
	// be reproduced and so nobody has to guess which release it matches.
	Version string
	// License is the SPDX identifier of that project, or a plain statement of
	// the terms where no SPDX identifier fits.
	//
	// The second case is not a loophole. A work of the United States Government
	// carries no copyright at all by statute, and there is no SPDX identifier
	// that says so without also claiming a grant somebody made. Writing one of
	// those in would be inventing a licence the source never issued, which is
	// the opposite of what this field is for.
	License string
	// Source is where the terms and the original can be read.
	Source string
	// Checked is the day somebody read those terms at that address, written
	// as YYYY-MM-DD.
	//
	// It is here because a license is not a property of a project, it is a
	// property of a version of a project, and projects have moved between
	// permissive and copyleft terms between releases. "This one is MIT" is the
	// kind of sentence that reads most convincingly exactly when it has gone
	// out of date, so the entry carries the day it was true rather than
	// leaving the next session to guess how fresh it is.
	Checked string
}

// Data lists every reference collection inside this build.
//
// A test compares it with the files that are actually embedded, in both
// directions, so a list added without its attribution fails the build and an
// attribution left behind after a list was dropped fails it too.
var Data = []DataSource{
	// The postal registry is the one collection here that is not a word list
	// from a fake data project, and it is the largest by an order of magnitude.
	// The terms are the reason it could be taken whole: a work of the United
	// States Government has no copyright, and the United States has no database
	// right for a compilation of facts to fall under. Both halves had to be true,
	// because a licence granted by whoever republishes a collection answers
	// nothing about a collection they did not make.
	{"en-US/postal-places", "U.S. Census Bureau", "2020 vintage", "Public domain, 17 U.S.C. 105", "https://www.census.gov/geographies/reference-files/time-series/geo/relationship-files.html", "2026-08-04"},
	{"en-US/first-names-female", "joke2k/faker", "v40.36.0", "MIT", "https://github.com/joke2k/faker", "2026-08-02"},
	{"en-US/first-names-male", "joke2k/faker", "v40.36.0", "MIT", "https://github.com/joke2k/faker", "2026-08-02"},
	{"en-US/last-names", "joke2k/faker", "v40.36.0", "MIT", "https://github.com/joke2k/faker", "2026-08-02"},
	{"pl-PL/cities", "joke2k/faker", "v40.36.0", "MIT", "https://github.com/joke2k/faker", "2026-08-02"},
	{"pl-PL/first-names-female", "joke2k/faker", "v40.36.0", "MIT", "https://github.com/joke2k/faker", "2026-08-02"},
	{"pl-PL/first-names-male", "joke2k/faker", "v40.36.0", "MIT", "https://github.com/joke2k/faker", "2026-08-02"},
	{"pl-PL/last-names-male", "joke2k/faker", "v40.36.0", "MIT", "https://github.com/joke2k/faker", "2026-08-02"},
	{"pl-PL/last-names-unisex", "joke2k/faker", "v40.36.0", "MIT", "https://github.com/joke2k/faker", "2026-08-02"},
	{"pl-PL/regions", "joke2k/faker", "v40.36.0", "MIT", "https://github.com/joke2k/faker", "2026-08-02"},
}

// Component is one external module compiled into this binary.
//
// This list is the strictest of the three ways outside code reaches this
// project, and the difference is worth knowing because it decides how much
// work each one costs. A module here is linked into what somebody receives,
// so its terms bind them and the full sieve applies. A module needed only to
// build the tests is never distributed, so it takes a lighter check but still
// a pinned version. A program run over our source, such as a linter, makes
// nothing of ours a derived work at all, the same way a compiler does not.
//
// Only the first kind belongs in this list, because the flag that prints it
// answers one question: what is inside this binary.
type Component struct {
	// Module is the import path, which is what go.mod calls it.
	Module string
	// Version is the exact version linked, not a range.
	Version string
	// License is the SPDX identifier.
	License string
	// Source is where the module and its license text live.
	Source string
	// Checked is the day the license file of this exact version was read, as
	// YYYY-MM-DD. See the note on the same field of DataSource.
	Checked string
}

// ThirdParty lists every external module inside this build.
//
// A test compares this list against the direct requirements in go.mod, so a
// dependency that nobody described here fails the build instead of shipping
// unlisted. Every entry was read from the module itself rather than from
// anybody's memory of what it is licensed under.
var ThirdParty = []Component{
	{
		Module:  "github.com/goccy/go-yaml",
		Version: "v1.19.2",
		License: "MIT",
		Source:  "https://github.com/goccy/go-yaml",
		Checked: "2026-08-01",
	},
}
