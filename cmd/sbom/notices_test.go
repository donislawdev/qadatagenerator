package main

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/license"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

// shippedTree is the tree this command is compiled with.
var shippedTree = tree{notices: license.CommandLineNotices, windowNotices: license.WindowNotices, version: version.Tool}

// TestTheNoticesInThisTreeDescribeABuild reads the notices this tree compiles
// into every binary, so a section edited into a shape this command cannot read
// fails here, on the push, rather than on the day of a release.
//
// It asks its check for both answers. The shipped notices with one section
// written back to the words it had before 2026-11-18, and with the icons left
// without a section, have to be reported, and the shipped notices as they are
// have to pass.
func TestTheNoticesInThisTreeDescribeABuild(t *testing.T) {
	for name, planted := range map[string]tree{
		"a licence in words": {notices: shippedTree.notices, version: shippedTree.version, windowNotices: strings.Replace(
			shippedTree.windowNotices, "`2020 vintage`. LicenseRef-US-Government-Work.", "`2020 vintage`. Public domain.", 1)},
		"a work without its section": {notices: shippedTree.notices, version: shippedTree.version, windowNotices: strings.Replace(
			shippedTree.windowNotices, "## lucide-icons/lucide", "## lucide-icons", 1)},
	} {
		if planted.windowNotices == shippedTree.windowNotices {
			t.Fatalf("%s: the planted fault is not in the shipped notices, so the case would read them as they are", name)
		}
		if len(shippedProblems(planted)) == 0 {
			t.Errorf("%s: the check found nothing wrong with notices planted with a fault", name)
		}
	}
	for _, p := range shippedProblems(shippedTree) {
		t.Error(p)
	}
}

// shippedProblems is everything wrong with notices that should describe the
// builds of this tree.
func shippedProblems(from tree) []string {
	notes, err := parseTree(from)
	if err != nil {
		return []string{err.Error()}
	}
	var problems []string
	n, w := notes[programPath], notes[windowPath]
	if !goRelease.MatchString(n.std.version) {
		problems = append(problems, fmt.Sprintf("the standard library is at %q, which is not a Go release", n.std.version))
	}
	if w.std.version != n.std.version {
		problems = append(problems, fmt.Sprintf("the window's notices name %s and the command line's %s, and one commit "+
			"builds both with one toolchain", w.std.version, n.std.version))
	}
	if len(n.modules) == 0 || len(w.modules) == 0 {
		problems = append(problems, "a notices file names no module, and every build of either program links some")
	}
	// A module whose files state no copyright at all is described with
	// NOASSERTION, and is named here with the day that was read, so that a
	// section which merely lost its copyright lines is not taken for one.
	stateNone := map[string]string{
		"github.com/rymdport/portal": "v0.4.2 has the Apache-2.0 text alone and no file of it says copyright (read 2026-11-18)",
	}
	for _, program := range []string{programPath, windowPath} {
		for _, c := range append([]component{notes[program].std}, notes[program].modulesInOrder()...) {
			if _, known := stateNone[c.path]; len(c.copyrights) == 0 && !known {
				problems = append(problems, fmt.Sprintf("%s: %q quotes no copyright line", programNamed(program).file, c.path))
			}
		}
	}
	for program, want := range map[string][]string{
		programPath: {"LicenseRef-Sun-fdlibm-2004", "LicenseRef-Cephes", "LicenseRef-US-Government-Work"},
		windowPath: {"LicenseRef-Sun-fdlibm-2004", "LicenseRef-Cephes", "LicenseRef-Arev", "LicenseRef-Public-Domain",
			"LicenseRef-SGI-FreeB", "LicenseRef-MinGW-w64-Cephes", "LicenseRef-US-Government-Work"},
	} {
		var ids []string
		for _, e := range notes[program].extracted {
			ids = append(ids, e.id)
		}
		if !reflect.DeepEqual(ids, want) {
			problems = append(problems, fmt.Sprintf("the texts without an SPDX identifier in the notices of %s are %v, want %v",
				program, ids, want))
		}
	}
	for program, want := range map[string][]string{
		programPath: {"U.S. Census Bureau", "joke2k/faker"},
		windowPath:  {"GNU C Library", "IBM/plex", "MinGW-w64 runtime", "U.S. Census Bureau", "joke2k/faker", "lucide-icons/lucide"},
	} {
		if got := sortedKeys(notes[program].works); !reflect.DeepEqual(got, want) {
			problems = append(problems, fmt.Sprintf("the works in the notices of %s are %v, want %v", program, got, want))
		}
	}
	if w.works["MinGW-w64 runtime"].version == "" || w.works["GNU C Library"].version == "" {
		problems = append(problems, fmt.Sprintf("the C compiler's code is read as %+v and %+v",
			w.works["MinGW-w64 runtime"], w.works["GNU C Library"]))
	}
	return problems
}

// The document names this project's copyright and licence in words of its
// own, and both are the ones the notices and the register say.
func TestTheDocumentNamesOurselvesAsTheNoticesDo(t *testing.T) {
	for _, p := range programs {
		if !strings.Contains(strings.Join(strings.Fields(p.notes(shippedTree)), " "), "is "+ourCopyright+" ") {
			t.Errorf("%s does not say %q, which the document names this project's copyright as", p.file, ourCopyright)
		}
	}
	if !listedLicences[ourLicence] {
		t.Errorf("this project is under %s, which licences.go has not read on the SPDX License List", ourLicence)
	}
}

// The window's notices name a choice in brackets, a licence once for each of
// two texts and an exception, and each is read for what it is.
func TestTheWindowsFixtureNoticesAreRead(t *testing.T) {
	w := fixtureNotes(t)[windowPath]
	for path, want := range map[string][]string{
		"example.com/toolkit": {"Unlicense OR BSD-3-Clause", "MIT", "LicenseRef-Arev"},
		"github.com/go-gl/gl": {"MIT", "LicenseRef-SGI-FreeB"},
	} {
		if got := w.modules[path].licences; !reflect.DeepEqual(got, want) {
			t.Errorf("%s: %q, want %q", path, got, want)
		}
	}
	if glibc := w.works["GNU C Library"]; !reflect.DeepEqual(glibc.licences,
		[]string{"LGPL-2.1-or-later WITH GCC-exception-2.0-note", "LGPL-2.1-or-later"}) {
		t.Errorf("GNU C Library: %q", glibc.licences)
	}
	if plex := w.works["IBM/plex"]; plex.version != "1.1.0" || plex.files != nil ||
		!reflect.DeepEqual(plex.copyrights, []string{`Copyright © 2017 IBM Corp. with Reserved Font Name "Plex"`}) {
		t.Errorf("IBM/plex is read as %+v", plex)
	}
	if census := w.works["U.S. Census Bureau"]; len(census.copyrights) != 0 {
		t.Errorf("the law the Census section quotes is read as the copyright lines %q", census.copyrights)
	}
}

// A work is in one place: compiled into a build or beside it, never both, and
// every entry of the table says one of the two at most and names a program.
func TestAWorkIsEitherCompiledInOrBesideABuild(t *testing.T) {
	if err := (work{beside: "windows/amd64", in: "linux/amd64"}).check(component{path: "Both"}); err == nil ||
		!strings.Contains(err.Error(), `"Both" is both compiled into linux/amd64 and beside windows/amd64`) {
		t.Errorf("a work both compiled in and beside was let through: %v", err)
	}
	for heading, w := range works {
		if w.beside != "" && w.in != "" {
			t.Errorf("programs.go says %q is both compiled into %s and beside %s", heading, w.in, w.beside)
		}
		if len(w.programs) == 0 {
			t.Errorf("programs.go names no program for %q", heading)
		}
		for _, p := range w.programs {
			if !slices.ContainsFunc(programs, func(q program) bool { return q.path == p }) {
				t.Errorf("programs.go gives %q to %s, which no release builds", heading, p)
			}
		}
		if w.renderer && w.beside == "" {
			t.Errorf("programs.go reads %q from -renderer and says nothing it travels beside", heading)
		}
	}
}

// A section names a licence once for each text it quotes, so a licence named
// twice is two texts under one licence, and is one term of the expression.
func TestALicenceNamedTwiceIsOneTerm(t *testing.T) {
	n, err := parseNotices(edited("MIT AND Apache-2.0", "MIT AND Apache-2.0 AND MIT"))
	if err != nil {
		t.Fatal(err)
	}
	if got := n.modules["example.com/alpha"].licences; !reflect.DeepEqual(got, []string{"MIT", "Apache-2.0"}) {
		t.Errorf("read %q", got)
	}
}

func (n notices) modulesInOrder() []component {
	var out []component
	for _, path := range sortedKeys(n.modules) {
		out = append(out, n.modules[path])
	}
	return out
}

func TestTheFixtureNoticesAreRead(t *testing.T) {
	n, err := parseNotices(fixtureNotices)
	if err != nil {
		t.Fatal(err)
	}
	want := notices{
		std: component{
			version:  "go1.27.4",
			licences: []string{"BSD-3-Clause", "LicenseRef-Sun-fdlibm-2004", "LicenseRef-Cephes"},
			copyrights: []string{
				"Copyright 2009 The Go Authors.",
				"Copyright (C) 2004 by Sun Microsystems, Inc. All rights reserved.",
				"Copyright 1984 by Stephen L. Moshier",
			},
		},
		modules: map[string]component{
			"example.com/alpha": {path: "example.com/alpha", version: "v1.2.3", licences: []string{"MIT", "Apache-2.0"},
				copyrights: []string{"Copyright (c) 2020 Alpha Authors"}},
			"example.com/beta/v2": {path: "example.com/beta/v2", version: "v2.0.0+incompatible", licences: []string{"BSD-2-Clause"},
				copyrights: []string{"Copyright (c) 2016 Beta & <Contributors>"}},
		},
		works: map[string]component{
			"joke2k/faker": {path: "joke2k/faker", version: "v40.36.0", licences: []string{"MIT"},
				copyrights: []string{"Copyright (c) 2012 Daniele Faraglia"}},
			"U.S. Census Bureau": {path: "U.S. Census Bureau", version: "2020 vintage", licences: []string{"LicenseRef-US-Government-Work"}},
		},
		extracted: []extractedLicence{
			{id: "LicenseRef-Sun-fdlibm-2004", name: "Sun Microsystems fdlibm notice, 2004 form",
				text: "Copyright (C) 2004 by Sun Microsystems, Inc. All rights reserved.\n\nPermission to use, copy, modify, and distribute this\n" +
					"software is freely granted, provided that this notice\nis preserved."},
			{id: "LicenseRef-Cephes", name: "Cephes Math Library notice",
				text: "Cephes Math Library Release 2.8:  June, 2000\nCopyright 1984 by Stephen L. Moshier\n## Not a heading, because it is quoted."},
			{id: "LicenseRef-US-Government-Work", name: "17 U.S.C. 105(a), no copyright in a work of the United States Government",
				text: "Copyright protection under this title is not available for any work of the\nUnited States Government."},
		},
	}
	if !reflect.DeepEqual(n, want) {
		t.Errorf("read\n%+v\nwant\n%+v", n, want)
	}
}

// edited is the fixture with one piece of it replaced, for the refusals below.
func edited(old, replacement string) string {
	return strings.Replace(fixtureNotices, old, replacement, 1)
}

func TestNoticesThatCannotDescribeABuildAreRefused(t *testing.T) {
	sum := strings.Repeat("0", 64)
	for _, c := range []struct{ name, text, want string }{
		{"quoted before any section", "```\nCopyright 2020 Nobody\n```\n" + fixtureNotices, "comes before any section"},
		{"never closed", fixtureNotices + "\n```\nCopyright 2020 Nobody\n", "never closed"},
		{"no opening line", edited("Version `v1.2.3`. MIT AND Apache-2.0.", "MIT."), `"example.com/alpha" does not open with`},
		{"words for a licence", edited("Version `v1.2.3`. MIT AND Apache-2.0.", "Version `v1.2.3`. Public domain."),
			`"example.com/alpha" does not open with`},
		{"subheading first", edited("## example.com/beta/v2\n\n", "## example.com/beta/v2\n\n### Aside\n\n"),
			`"example.com/beta/v2" does not open with`},
		{"bracket never closed", edited("MIT AND Apache-2.0.", "(MIT AND Apache-2.0."), "a bracket is opened and never closed"},
		{"bracket closed twice", edited("MIT AND Apache-2.0.", "MIT AND Apache-2.0)."), `")" follows a complete expression`},
		{"operator for a licence", edited("MIT AND Apache-2.0.", "MIT AND (OR)."), `"OR" stands where a licence should`},
		{"WITH and no exception", edited("MIT AND Apache-2.0.", "MIT AND Apache-2.0 WITH (LLVM-exception)."),
			"WITH after Apache-2.0 is not followed by an exception"},
		{"exception nobody read", edited("MIT AND Apache-2.0.", "MIT AND Apache-2.0 WITH Classpath-exception-2.0."),
			"names the exception Classpath-exception-2.0, which is not"},
		{"identifier nobody read", edited("BSD-2-Clause.", "BSD-2-Clause-Patent."), "names BSD-2-Clause-Patent, which is not"},
		{"identifier nobody read in a choice", edited("BSD-2-Clause.", "(BSD-2-Clause OR 0BSD)."), "names 0BSD, which is not"},
		{"unknown reference", edited("MIT AND Apache-2.0", "MIT AND LicenseRef-Mine"), "names LicenseRef-Mine, which is not"},
		{"reference without its text", edited("Cephes Math Library Release 2.8:  June, 2000", "Cephes Math Library"),
			"0 of its quoted texts open with"},
		{"reference text twice", edited("#### Cephes\n", "#### Cephes\n\n```\nCephes Math Library Release 2.8:  June, 2000\n```\n"),
			"2 of its quoted texts open with"},
		{"the law without its text", edited("Copyright protection under this title", "Protection under this title"),
			`"U.S. Census Bureau" names LicenseRef-US-Government-Work, and 0 of its quoted texts`},
		{"files beside a binary, none listed", strings.Replace(fixtureNotices, "## joke2k/faker", "## Mesa 3D\n\nVersion `1`. MIT.\n\n## joke2k/faker", 1),
			`"Mesa 3D" is of files beside a binary, and lists none`},
		{"a work compiled in, with files", fixtureNotices + "\n## IBM/plex\n\nVersion `1.1.0`. OFL-1.1.\n\n- `x.ttf`, 1 bytes, SHA-256 `" + sum + "`\n",
			`"IBM/plex" lists files, and it is a work compiled into a binary`},
		{"work twice", fixtureNotices + "\n## joke2k/faker\n\nVersion `v40.36.0`. MIT.\n",
			`"joke2k/faker" appears twice`},
		{"no standard library", edited("## Go standard library and runtime", "## example.com/gamma"),
			"there is no section"},
		{"standard library twice", fixtureNotices + "\n## Go standard library and runtime\n\nVersion `go1.27.4`. MIT.\n",
			"appears twice"},
		{"module twice", fixtureNotices + "\n## example.com/alpha\n\nVersion `v1.2.3`. MIT.\n", `"example.com/alpha" appears twice`},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := parseNotices(c.text)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("err = %v, want one saying %q", err, c.want)
			}
		})
	}
}

// The two files are held to each other, and each refusal names the file it is
// about.
func TestTheNoticesOfTheTwoProgramsAreHeldToEachOther(t *testing.T) {
	cephes := "Cephes Math Library Release 2.8:  June, 2000"
	window := func(old, replacement string) tree {
		return tree{notices: fixtureNotices, windowNotices: strings.Replace(fixtureWindowNotices, old, replacement, 1), version: "0.1.0"}
	}
	// The window's standard library naming Cephes, and quoting another text
	// for it than the command line's does.
	twoTexts := window("BSD-3-Clause AND MIT. Every build", "BSD-3-Clause AND LicenseRef-Cephes. Every build")
	twoTexts.windowNotices = strings.Replace(twoTexts.windowNotices, "Copyright (c) 2010 Somebody Else",
		cephes+"\nAnother text altogether.", 1)
	cliWithout := func(heading string) string {
		start := strings.Index(fixtureNotices, "\n## "+heading)
		end := strings.Index(fixtureNotices[start+1:], "\n## ")
		if end < 0 {
			return fixtureNotices[:start+1]
		}
		return fixtureNotices[:start+1] + fixtureNotices[start+1+end+1:]
	}
	for _, c := range []struct {
		name string
		from tree
		want string
	}{
		{"the window's unreadable", tree{notices: fixtureNotices, windowNotices: "no sections"},
			"THIRD-PARTY-NOTICES-GUI.md cannot describe a build"},
		{"the command line's unreadable", tree{notices: "no sections", windowNotices: fixtureWindowNotices},
			"THIRD-PARTY-NOTICES.md cannot describe a build"},
		{"a work in the other program's notices", tree{notices: fixtureNotices + "\n## IBM/plex\n\nVersion `1.1.0`. OFL-1.1.\n",
			windowNotices: fixtureWindowNotices}, `THIRD-PARTY-NOTICES.md has the section "IBM/plex", which is a work of qadg-gui`},
		{"the renderer in a program's notices", window("## IBM/plex\n", "## Mesa 3D\n\nVersion `1`. MIT.\n\n- `opengl/x.dll`, 1 bytes, SHA-256 `"+
			strings.Repeat("0", 64)+"`\n\n## IBM/plex\n"), `has the section "Mesa 3D", whose notices are the software renderer's own`},
		{"a work nobody wrote a section for", window("## IBM/plex\n", "## example.com/plex\n"),
			`names the work "IBM/plex" for qadg-gui, and THIRD-PARTY-NOTICES-GUI.md has no section for it`},
		{"a work of both programs in one of them alone", tree{notices: cliWithout("joke2k/faker"), windowNotices: fixtureWindowNotices},
			`names the work "joke2k/faker" for qadg, and THIRD-PARTY-NOTICES.md has no section for it`},
		{"one identifier, two texts", twoTexts, "THIRD-PARTY-NOTICES-GUI.md quotes another text for LicenseRef-Cephes"},
		{"a reference nobody quotes", tree{notices: strings.Replace(fixtureNotices, " AND\nLicenseRef-Cephes.", ".", 1),
			windowNotices: fixtureWindowNotices}, "LicenseRef-Cephes is in the list of unlisted texts and no section names it"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := parseTree(c.from)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("err = %v, want one saying %q", err, c.want)
			}
		})
	}
}

// A fence holding nothing but blank lines opens with nothing, and is never the
// text an unlisted identifier stands for.
func TestAnEmptyQuotedTextOpensWithNothing(t *testing.T) {
	if got := firstLine([]string{"", "  "}); got != "" {
		t.Errorf("an empty block opens with %q", got)
	}
	s := section{heading: "x", blocks: [][]string{{""}, {"Cephes Math Library Release 2.8:  June, 2000", ""}}}
	e, err := extract("LicenseRef-Cephes", s)
	if err != nil || e.text != "Cephes Math Library Release 2.8:  June, 2000" {
		t.Errorf("extracted %+v, %v", e, err)
	}
}
