// Package license answers the question that the --license flag asks: what is
// inside this binary and under what terms.
//
// The answer has to come from the binary rather than from the repository. A
// license of the LGPL family binds whoever receives the executable, not
// whoever clones the source, and the legal department that blocks adoption
// reads neither. It will run a flag if somebody shows it one.
package license

import "github.com/donislawdev/qadatagenerator/internal/version"

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
// The JSON names on this type and on Component are read by compliance
// scanners, so they are a public surface and follow the same rule as the
// manifest: fields may be added, an existing name never changes meaning.
type DataSource struct {
	// Corpus is the list it produced, as locale and name.
	Corpus string `json:"corpus"`
	// Project is where it was read from.
	Project string `json:"project"`
	// Version is the exact tag or vintage it was read at, so the snapshot can
	// be reproduced and so nobody has to guess which release it matches.
	Version string `json:"version"`
	// License is the SPDX identifier of that project, or a plain statement of
	// the terms where no SPDX identifier fits.
	//
	// The second case is not a loophole. A work of the United States Government
	// carries no copyright at all by statute, and there is no SPDX identifier
	// that says so without also claiming a grant somebody made. Writing one of
	// those in would be inventing a licence the source never issued, which is
	// the opposite of what this field is for.
	License string `json:"license"`
	// Source is where the terms and the original can be read.
	Source string `json:"source"`
	// Checked is the day somebody read those terms at that address, written
	// as YYYY-MM-DD.
	//
	// It is here because a license is not a property of a project, it is a
	// property of a version of a project, and projects have moved between
	// permissive and copyleft terms between releases. "This one is MIT" is the
	// kind of sentence that reads most convincingly exactly when it has gone
	// out of date, so the entry carries the day it was true rather than
	// leaving the next session to guess how fresh it is.
	//
	// The JSON name says what was checked rather than repeating the field name,
	// because a bare "checked" in a compliance document invites the reading
	// that somebody checked the data.
	Checked string `json:"license_checked"`
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
	Module string `json:"module"`
	// Version is the exact version linked, not a range.
	Version string `json:"version"`
	// License is the SPDX identifier.
	License string `json:"license"`
	// Source is where the module and its license text live.
	Source string `json:"source"`
	// Checked is the day the license file of this exact version was read, as
	// YYYY-MM-DD. See the note on the same field of DataSource.
	Checked string `json:"license_checked"`
}

// ThirdParty lists every external module inside this build, reached directly
// or through another one.
//
// A test compares this list against go.mod in both directions, so a dependency
// nobody described fails the build instead of shipping unlisted, and an entry
// for a module this build no longer uses fails it too. Every entry was read
// from the module's own licence file at the pinned version rather than from
// anybody's memory of what it is licensed under.
//
// # Why the indirect ones are here as well
//
// A module pulled in through another sits in the binary just the same, and the
// licence of code inside a binary binds whoever received the binary. Listing
// one module too many costs a line. Listing one too few is a claim about
// somebody else's terms that nobody checked.
//
// # How this list is kept, since it stopped being hand sized on 2026-08-12
//
// tools/licencje.py reads the licence file of every module from the module
// cache, where go.sum has already checked that the bytes are the ones the
// author released under that version. One dependency was a hand written line.
// Forty is forty chances to mistype one, and a register of licences with a
// typo in it is worse than none because it reads as though somebody checked.
//
// The generator rewrites the whole list, so anything worth keeping is written
// here rather than beside an entry. A note between the braces below survives
// exactly until the next regeneration.
//
// # The one entry that deserves a note, and it is fyne.io/fyne/v2
//
// It is the only module here that reaches one of the two binaries rather than
// both. The command line binary never imports it, which a guard checks
// transitively, because that import would pull in the C toolchain and stop it
// cross compiling from one machine.
//
// Measured on 2026-08-12: a window built on it closes over 328 packages
// against 152 for the command line binary, and its root package imports
// net/http directly to serve resources over it. The rule about outgoing
// connections is about what the product does, so a separate guard checks that
// nothing this project wrote asks it to.
//
// Adding it also brought three vulnerabilities in golang.org/x/image that the
// scanner traced into our own call paths, fixed by pinning that module forward
// rather than by waiting for the toolkit to do it. A dependency arrives with
// its own dependencies and their history, which is what sieve question (c) is
// about.
var ThirdParty = []Component{
	{
		Module:  "fyne.io/fyne/v2",
		Version: "v2.8.1",
		License: "BSD-3-Clause",
		Source:  "https://fyne.io/fyne/v2",
		Checked: "2026-09-02",
	},
	{
		Module:  "fyne.io/systray",
		Version: "v1.12.3-0.20260810170012-af4e8e793ec4",
		License: "Apache-2.0",
		Source:  "https://fyne.io/systray",
		Checked: "2026-09-02",
	},
	{
		Module:  "github.com/BurntSushi/toml",
		Version: "v1.6.0",
		License: "MIT",
		Source:  "https://github.com/BurntSushi/toml",
		Checked: "2026-08-12",
	},
	{
		Module:  "github.com/FyshOS/fancyfs",
		Version: "v0.0.1",
		License: "BSD-3-Clause",
		Source:  "https://github.com/FyshOS/fancyfs",
		Checked: "2026-08-17",
	},
	{
		Module:  "github.com/anthonynsimon/bild",
		Version: "v0.14.0",
		License: "MIT",
		Source:  "https://github.com/anthonynsimon/bild",
		Checked: "2026-08-12",
	},
	{
		Module:  "github.com/clipperhouse/uax29/v2",
		Version: "v2.2.0",
		License: "MIT",
		Source:  "https://github.com/clipperhouse/uax29/v2",
		Checked: "2026-08-12",
	},
	{
		Module:  "github.com/davecgh/go-spew",
		Version: "v1.1.1",
		License: "ISC",
		Source:  "https://github.com/davecgh/go-spew",
		Checked: "2026-08-12",
	},
	{
		Module:  "github.com/fsnotify/fsnotify",
		Version: "v1.9.0",
		License: "BSD-3-Clause",
		Source:  "https://github.com/fsnotify/fsnotify",
		Checked: "2026-08-12",
	},
	{
		Module:  "github.com/fyne-io/gl-js",
		Version: "v0.2.1-0.20260315212741-029c47fd27e8",
		License: "BSD-3-Clause",
		Source:  "https://github.com/fyne-io/gl-js",
		Checked: "2026-08-12",
	},
	{
		Module:  "github.com/fyne-io/glfw-js",
		Version: "v0.4.0",
		License: "MIT",
		Source:  "https://github.com/fyne-io/glfw-js",
		Checked: "2026-08-12",
	},
	{
		Module:  "github.com/fyne-io/image",
		Version: "v0.1.1",
		License: "BSD-3-Clause",
		Source:  "https://github.com/fyne-io/image",
		Checked: "2026-08-12",
	},
	{
		Module:  "github.com/fyne-io/oksvg",
		Version: "v0.2.0",
		License: "BSD-3-Clause",
		Source:  "https://github.com/fyne-io/oksvg",
		Checked: "2026-08-12",
	},
	{
		Module:  "github.com/go-gl/gl",
		Version: "v0.0.0-20260331235117-4566fea9a276",
		License: "MIT",
		Source:  "https://github.com/go-gl/gl",
		Checked: "2026-08-12",
	},
	{
		Module:  "github.com/go-gl/glfw/v3.4/glfw",
		Version: "v0.1.0-pre.1.0.20260707082822-2a407d02d01a",
		License: "BSD-3-Clause",
		Source:  "https://github.com/go-gl/glfw/v3.4/glfw",
		Checked: "2026-08-12",
	},
	{
		Module:  "github.com/go-text/render",
		Version: "v0.2.1",
		License: "BSD-3-Clause",
		Source:  "https://github.com/go-text/render",
		Checked: "2026-08-12",
	},
	{
		Module:  "github.com/go-text/typesetting",
		Version: "v0.3.4",
		License: "BSD-3-Clause",
		Source:  "https://github.com/go-text/typesetting",
		Checked: "2026-08-12",
	},
	{
		Module:  "github.com/goccy/go-yaml",
		Version: "v1.19.2",
		License: "MIT",
		Source:  "https://github.com/goccy/go-yaml",
		Checked: "2026-08-01",
	},
	{
		Module:  "github.com/godbus/dbus/v5",
		Version: "v5.2.2",
		License: "BSD-2-Clause",
		Source:  "https://github.com/godbus/dbus/v5",
		Checked: "2026-08-12",
	},
	{
		Module:  "github.com/hack-pad/go-indexeddb",
		Version: "v0.3.2",
		License: "Apache-2.0",
		Source:  "https://github.com/hack-pad/go-indexeddb",
		Checked: "2026-08-12",
	},
	{
		Module:  "github.com/hack-pad/safejs",
		Version: "v0.1.0",
		License: "Apache-2.0",
		Source:  "https://github.com/hack-pad/safejs",
		Checked: "2026-08-12",
	},
	{
		Module:  "github.com/jeandeaual/go-locale",
		Version: "v0.0.0-20250612000132-0ef82f21eade",
		License: "MIT",
		Source:  "https://github.com/jeandeaual/go-locale",
		Checked: "2026-08-12",
	},
	{
		Module:  "github.com/jsummers/gobmp",
		Version: "v0.0.0-20230614200233-a9de23ed2e25",
		License: "MIT",
		Source:  "https://github.com/jsummers/gobmp",
		Checked: "2026-08-12",
	},
	{
		Module:  "github.com/mattn/go-runewidth",
		Version: "v0.0.24",
		License: "MIT",
		Source:  "https://github.com/mattn/go-runewidth",
		Checked: "2026-08-12",
	},
	{
		Module:  "github.com/nfnt/resize",
		Version: "v0.0.0-20180221191011-83c6a9932646",
		License: "ISC",
		Source:  "https://github.com/nfnt/resize",
		Checked: "2026-08-12",
	},
	{
		Module:  "github.com/nicksnyder/go-i18n/v2",
		Version: "v2.5.1",
		License: "MIT",
		Source:  "https://github.com/nicksnyder/go-i18n/v2",
		Checked: "2026-08-12",
	},
	{
		Module:  "github.com/pmezard/go-difflib",
		Version: "v1.0.0",
		License: "BSD-2-Clause",
		Source:  "https://github.com/pmezard/go-difflib",
		Checked: "2026-08-12",
	},
	{
		Module:  "github.com/rymdport/portal",
		Version: "v0.4.2",
		License: "Apache-2.0",
		Source:  "https://github.com/rymdport/portal",
		Checked: "2026-08-12",
	},
	{
		Module:  "github.com/srwiley/oksvg",
		Version: "v0.0.0-20221011165216-be6e8873101c",
		License: "BSD-3-Clause",
		Source:  "https://github.com/srwiley/oksvg",
		Checked: "2026-08-12",
	},
	{
		Module:  "github.com/srwiley/rasterx",
		Version: "v0.0.0-20220730225603-2ab79fcdd4ef",
		License: "BSD-3-Clause",
		Source:  "https://github.com/srwiley/rasterx",
		Checked: "2026-08-12",
	},
	{
		Module:  "github.com/stretchr/testify",
		Version: "v1.11.1",
		License: "MIT",
		Source:  "https://github.com/stretchr/testify",
		Checked: "2026-08-12",
	},
	{
		Module:  "github.com/yuin/goldmark",
		Version: "v1.8.2",
		License: "MIT",
		Source:  "https://github.com/yuin/goldmark",
		Checked: "2026-08-12",
	},
	{
		Module:  "golang.org/x/image",
		Version: "v0.43.0",
		License: "BSD-3-Clause",
		Source:  "https://golang.org/x/image",
		Checked: "2026-08-12",
	},
	{
		Module:  "golang.org/x/net",
		Version: "v0.55.0",
		License: "BSD-3-Clause",
		Source:  "https://golang.org/x/net",
		Checked: "2026-08-17",
	},
	{
		Module:  "golang.org/x/sys",
		Version: "v0.45.0",
		License: "BSD-3-Clause",
		Source:  "https://golang.org/x/sys",
		Checked: "2026-08-17",
	},
	{
		Module:  "golang.org/x/text",
		Version: "v0.38.0",
		License: "BSD-3-Clause",
		Source:  "https://golang.org/x/text",
		Checked: "2026-08-12",
	},
	{
		Module:  "gopkg.in/yaml.v3",
		Version: "v3.0.1",
		License: "Apache-2.0",
		Source:  "https://gopkg.in/yaml.v3",
		Checked: "2026-08-12",
	},
}

// Report is the whole answer to what is inside this binary, in the shape a
// machine reads it.
//
// # Why a machine readable form exists at all
//
// The reader this flag was built for is a compliance scanner, and the person
// who runs it is the one who can block adoption. Without this shape they parse
// the form written for a human, which this project has already been bitten by
// once: an oracle recognised a class of routing number with strings.Contains
// over a fragment of a description, so rewording a sentence silently moved a
// number from one class to another.
//
// # What is deliberately different from the printed form
//
// The printed form groups the reference lists under the project they came from,
// because a person wants the project named once. This one keeps a row per list,
// unflattened, because that is what the registry holds and what a gate compares
// against the embedded files. Grouping is a decision about presentation, and
// making a parser undo it would be a second place where the same fact is
// arranged.
//
// # This shape is a public surface and grows the way the manifest does
//
// Fields may be added. The meaning of an existing field never changes, and a
// consumer is expected to ignore what it does not know. Without that rule the
// flag could not have been built before the registry settled, which is what let
// it exist now rather than after the release.
type Report struct {
	Tool       string       `json:"tool_version"`
	Code       Terms        `json:"code"`
	Spec       Terms        `json:"spec"`
	Components []Component  `json:"components"`
	Data       []DataSource `json:"data"`
	Notices    Notices      `json:"notices"`
}

// Terms is one license and what it covers.
type Terms struct {
	License string `json:"license"`
	// Scope is empty where the license covers everything this binary compiles,
	// and filled where it does not.
	Scope string `json:"scope,omitempty"`
}

// Notices are the sentences that answer the questions asked most often about a
// generator under copyleft.
type Notices struct {
	GeneratedData string `json:"generated_data"`
	SyntheticData string `json:"synthetic_data"`
	ExampleData   string `json:"example_data"`
}

// Current returns what this build carries.
//
// The slices are copied rather than handed out, so that a caller writing the
// report cannot change the registry that a gate is holding to the embedded
// files.
func Current() Report {
	components := make([]Component, len(ThirdParty))
	copy(components, ThirdParty)
	data := make([]DataSource, len(Data))
	copy(data, Data)
	return Report{
		// The version is here because a compliance answer is about one build,
		// not about a project. Reading this package is allowed to reach the
		// version numbers: they are written to be read by every layer and know
		// none of them.
		Tool:       version.Tool,
		Code:       Terms{License: Code},
		Spec:       Terms{License: Spec, Scope: SpecScope},
		Components: components,
		Data:       data,
		Notices: Notices{
			GeneratedData: GeneratedDataNotice,
			SyntheticData: SyntheticDataNotice,
			ExampleData:   ExampleDataNotice,
		},
	}
}
