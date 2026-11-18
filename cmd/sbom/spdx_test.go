package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// update writes the documents the fixtures render to testdata, in place of
// comparing them. A document written this way is not accepted until the SPDX
// project's own validator has read it (docs/21, tools/noty/spdx).
var update = flag.Bool("update", false, "write the golden documents instead of comparing them")

// golden is the document the fixture of the command line renders to, and
// goldenWindow the one of both programs with the software renderer. Each is
// the shape the SPDX project's own validator accepted: pyspdxtools from
// spdx-tools 0.8.5, on 2026-11-18. A change to the renderer shows up here as a
// failure that prints the new document, and that document is what to run the
// validator on before it replaces the file.
var (
	golden       = filepath.Join("testdata", "fixture.spdx.json")
	goldenWindow = filepath.Join("testdata", "fixture-window.spdx.json")
)

func renderFixture(t *testing.T, listing string, notes map[string]notices) []byte {
	t.Helper()
	document, err := render(fixtureTree.version, readBuilds(t, listing), notes)
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func TestTheDocumentIsTheOneTheValidatorAccepted(t *testing.T) {
	for file, document := range map[string][]byte{
		golden:       renderFixture(t, fixtureListing(), fixtureNotes(t)),
		goldenWindow: renderFixture(t, fixtureListing()+fixtureWindowListing(), fixtureNotesWithRenderer(t)),
	} {
		if *update {
			if err := os.WriteFile(file, document, 0o600); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(document, want) {
			t.Errorf("the document is not %s. If the change is meant, validate this one and put it there:\n%s", file, document)
		}
	}
}

// packagesOf is the document's packages by name, each once.
func packagesOf(t *testing.T, document []byte) (map[string]spdxPackage, map[string]bool) {
	t.Helper()
	var doc spdxDocument
	if err := json.Unmarshal(document, &doc); err != nil {
		t.Fatal(err)
	}
	pkgs := map[string]spdxPackage{}
	for _, p := range doc.Packages {
		if _, twice := pkgs[p.Name]; twice {
			t.Errorf("%s is listed twice", p.Name)
		}
		pkgs[p.Name] = p
	}
	relations := map[string]bool{}
	for _, r := range doc.Relationships {
		relations[r.SPDXElementID+" "+r.RelationshipType+" "+r.RelatedSPDXElement] = true
	}
	return pkgs, relations
}

// What the document of both programs says about its parts, read back rather
// than trusted from the golden file: the parts both programs link are one
// package each, the binding says it was built from a copy, the portal says it
// is in one build, the software renderer is a dependency of the window that
// the window does not contain, and the list from the Census Bureau has no
// copyright at all.
func TestThePartsAreDescribedForWhatTheyAre(t *testing.T) {
	pkgs, relations := packagesOf(t, renderFixture(t, fixtureListing()+fixtureWindowListing(), fixtureNotesWithRenderer(t)))
	for name, want := range map[string]string{
		"github.com/go-gl/gl": "Linked from a copy kept in third_party/go-gl-gl of this repository",
		"example.com/portal":  "Linked into the build(s) of qadg-gui for linux/amd64 and into no other.",
		"Mesa 3D":             "opengl/opengl32.dll (139264 bytes, SHA-256 c8511461ba71a0288750467e3be2f5c1c2bbbceaaf0ab5d8c496b0ba7302daa8).",
		"qadg-gui":            "and one linked into some of them alone says which",
		"qadg":                "Every one of these builds links each part this package contains",
		"stdlib":              "linked into every build of every program",
	} {
		if !strings.Contains(pkgs[name].Comment, want) {
			t.Errorf("%s says %q, which does not say %q", name, pkgs[name].Comment, want)
		}
	}
	for name, want := range map[string]string{
		"stdlib":              "BSD-3-Clause AND LicenseRef-Sun-fdlibm-2004 AND LicenseRef-Cephes AND MIT",
		"example.com/toolkit": "(Unlicense OR BSD-3-Clause) AND MIT AND LicenseRef-Arev",
		"Mesa 3D":             "MIT AND Apache-2.0 WITH LLVM-exception AND NCSA",
		"MinGW-w64 runtime":   "ZPL-2.1 AND LicenseRef-MinGW-w64-Cephes",
		"GNU C Library":       "LGPL-2.1-or-later WITH GCC-exception-2.0-note AND LGPL-2.1-or-later",
		"U.S. Census Bureau":  "LicenseRef-US-Government-Work",
	} {
		if got := pkgs[name].LicenseDeclared; got != want {
			t.Errorf("%s is under %q, want %q", name, got, want)
		}
	}
	for name, want := range map[string]string{
		"example.com/beta/v2": "Copyright (c) 2016 Beta & <Contributors>\nCopyright (c) 2017 A Second Beta Author",
		"U.S. Census Bureau":  none,
		"Mesa 3D":             "Copyright (C) 1999-2007  Brian Paul   All Rights Reserved.\nCopyright (c) 2003-2019 University of Illinois at Urbana-Champaign.",
		"qadg":                ourCopyright,
	} {
		if got := pkgs[name].CopyrightText; got != want {
			t.Errorf("%s carries the copyright %q, want %q", name, got, want)
		}
	}
	for _, want := range []string{
		"SPDXRef-DOCUMENT DESCRIBES SPDXRef-Package-qadg",
		"SPDXRef-DOCUMENT DESCRIBES SPDXRef-Package-qadg-gui",
		"SPDXRef-Package-qadg CONTAINS SPDXRef-Package-example.com-beta-v2",
		"SPDXRef-Package-qadg-gui CONTAINS SPDXRef-Package-example.com-beta-v2",
		"SPDXRef-Package-qadg CONTAINS SPDXRef-Package-joke2k-faker",
		"SPDXRef-Package-qadg-gui CONTAINS SPDXRef-Package-joke2k-faker",
		"SPDXRef-Package-qadg-gui CONTAINS SPDXRef-Package-IBM-plex",
		"SPDXRef-Package-qadg-gui CONTAINS SPDXRef-Package-MinGW-w64-runtime",
		"SPDXRef-Package-qadg-gui CONTAINS SPDXRef-Package-GNU-C-Library",
		"SPDXRef-Package-Mesa-3D OPTIONAL_DEPENDENCY_OF SPDXRef-Package-qadg-gui",
	} {
		if !relations[want] {
			t.Errorf("no relationship %q", want)
		}
	}
	for _, unwanted := range []string{
		"SPDXRef-Package-qadg CONTAINS SPDXRef-Package-IBM-plex",
		"SPDXRef-Package-qadg CONTAINS SPDXRef-Package-MinGW-w64-runtime",
		"SPDXRef-Package-qadg CONTAINS SPDXRef-Package-GNU-C-Library",
		"SPDXRef-Package-qadg-gui CONTAINS SPDXRef-Package-Mesa-3D",
		"SPDXRef-Package-qadg CONTAINS SPDXRef-Package-Mesa-3D",
		"SPDXRef-Package-qadg-gui CONTAINS SPDXRef-Package-example.com-alpha",
	} {
		if relations[unwanted] {
			t.Errorf("a relationship %q", unwanted)
		}
	}
}

// The MinGW-w64 runtime is linked into the build for Windows alone, so a
// document of builds without that one does not name it, and names the GNU C
// Library, which is in the build for Linux.
func TestTheCompilersCodeIsNamedOnlyWithItsBuild(t *testing.T) {
	listing := fixtureListing() + windowListingFor("dist/qadg-gui", "linux/amd64", windowDeps(fixturePortal))
	pkgs, _ := packagesOf(t, renderFixture(t, listing, fixtureNotes(t)))
	if _, named := pkgs["MinGW-w64 runtime"]; named {
		t.Error("a document with no build for windows/amd64 names the runtime only that build links")
	}
	if _, named := pkgs["GNU C Library"]; !named {
		t.Error("a document with the build for linux/amd64 does not name the start files linked into it")
	}
	if got := summary("0.1.0", readBuilds(t, listing), fixtureNotes(t)); !strings.HasSuffix(got, ", 12 components beside the programs") {
		t.Errorf("the summary counts what a build that is not there carries: %s", got)
	}
}

// One document names one version of a thing, and the notices of the two
// programs naming two is refused, whether a module or the toolchain.
func TestTwoVersionsOfOnePartAreRefused(t *testing.T) {
	builds := readBuilds(t, fixtureListing()+fixtureWindowListing())
	for name, edit := range map[string]func(n *notices){
		"example.com/beta/v2 is v2.0.0+incompatible in the notices read before and v2.0.1 in THIRD-PARTY-NOTICES-GUI.md": func(n *notices) {
			c := n.modules["example.com/beta/v2"]
			c.version = "v2.0.1"
			n.modules["example.com/beta/v2"] = c
		},
		"Go standard library and runtime is go1.27.4 in the notices read before and go1.27.5": func(n *notices) {
			n.std.version = "go1.27.5"
		},
		"joke2k/faker is v40.36.0 in the notices read before and v41.0.0": func(n *notices) {
			c := n.works["joke2k/faker"]
			c.version = "v41.0.0"
			n.works["joke2k/faker"] = c
		},
	} {
		notes := fixtureNotesWithRenderer(t)
		w := notes[windowPath]
		edit(&w)
		notes[windowPath] = w
		if _, err := render("0.1.0", builds, notes); err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("err = %v, want one saying %q", err, name)
		}
	}
}

// The same builds give the same bytes, in whatever order they arrive, because a
// document that differs every time it is generated cannot be compared with the
// one published.
func TestTheSameBuildsGiveTheSameBytes(t *testing.T) {
	reversed := listingFor("dist/qadg-windows-arm64.exe", "windows/arm64", "go1.27.4", fixtureDeps) +
		listingFor("dist/qadg-linux-amd64", "linux/amd64", "go1.27.4", fixtureDeps)
	if !bytes.Equal(renderFixture(t, fixtureListing(), fixtureNotes(t)), renderFixture(t, reversed, fixtureNotes(t))) {
		t.Error("two orders of the same builds rendered two documents")
	}
	windowFirst := fixtureWindowListing() + reversed
	if !bytes.Equal(renderFixture(t, fixtureListing()+fixtureWindowListing(), fixtureNotesWithRenderer(t)),
		renderFixture(t, windowFirst, fixtureNotesWithRenderer(t))) {
		t.Error("two orders of the builds of both programs rendered two documents")
	}
}

// Another set of targets is another document, and a namespace names exactly one.
func TestAnotherSetOfTargetsIsAnotherNamespace(t *testing.T) {
	one := listingFor("dist/qadg-linux-amd64", "linux/amd64", "go1.27.4", fixtureDeps)
	var a, b spdxDocument
	if err := json.Unmarshal(renderFixture(t, one, fixtureNotes(t)), &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(renderFixture(t, fixtureListing(), fixtureNotes(t)), &b); err != nil {
		t.Fatal(err)
	}
	if a.DocumentNamespace == b.DocumentNamespace {
		t.Errorf("one build and two builds share the namespace %s", a.DocumentNamespace)
	}
}

func TestAToolchainThatIsNotAReleaseIsRefused(t *testing.T) {
	if _, err := stdlibPackage(component{version: "go1.28rc1"}); err == nil || !strings.Contains(err.Error(), "not a Go release") {
		t.Errorf("err = %v", err)
	}
	notes := fixtureNotes(t)
	n := notes[programPath]
	n.std.version = "go1.28rc1"
	notes[programPath] = n
	if _, err := render("0.1.0", readBuilds(t, fixtureListing()), notes); err == nil || !strings.Contains(err.Error(), "not a Go release") {
		t.Errorf("err = %v", err)
	}
}

func TestTwoComponentsCannotShareAnIdentifier(t *testing.T) {
	notes := fixtureNotes(t)
	notes[programPath].modules["example.com-alpha"] = component{path: "example.com-alpha", version: "v1.0.0", licences: []string{"MIT"}}
	if _, err := render("0.1.0", readBuilds(t, fixtureListing()), notes); err == nil ||
		!strings.Contains(err.Error(), "both become SPDXRef-Package-example.com-alpha") {
		t.Errorf("err = %v", err)
	}
}

func TestAPackageURLIsPercentEncoded(t *testing.T) {
	got := packageURL("example.com/a b/c~d", "v2.0.0+incompatible").ReferenceLocator
	if want := "pkg:golang/example.com/a%20b/c~d@v2.0.0%2Bincompatible"; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestAComponentWithoutACopyrightLineAssertsNothing(t *testing.T) {
	if got := copyrightText(component{}); got != noAssertion {
		t.Errorf("got %q", got)
	}
}
