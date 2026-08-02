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
	// Version is the exact tag it was read at, so the snapshot can be
	// reproduced and so nobody has to guess which release it matches.
	Version string
	// License is the SPDX identifier of that project.
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
