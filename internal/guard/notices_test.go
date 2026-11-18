package guard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/license"
)

// noticedProgram is a binary this project ships and the file of notices that
// travels with it (DEC-327).
//
// # Why a guard and not a careful hand
//
// MIT and BSD ask for their notice in every copy of the code, a binary copy
// included, and the notice that matters is the one of the code that is
// actually linked. A dependency raised, a package newly imported or a new Go
// release each move that set, and none of them touches the notices. The same
// guard, in the same shape, has stood in Cert Doctor since 2026-10-28, and this
// file is adapted from it.
type noticedProgram struct {
	pkg     string      // the main package, as go list is asked for it
	notices string      // the file beside LICENSE that answers for it
	targets [][2]string // the builds that ship
	cgo     bool        // whether those builds are made with a C compiler

	// landmarks are files that one road of the gathering alone arrives at, each
	// as a module path and a path below that module. Text quoted before keeps
	// this guard green after the road to it is lost, so a road that matters
	// names a file it has to reach.
	landmarks [][2]string

	// byCompiler are the sections for code the C compiler's linker puts in of
	// its own. It comes with the compiler, not from this tree, so no walk
	// arrives at it, and toolchain_test.go holds these sections to the
	// linker's map through the pin.
	byCompiler []string
}

// noticedPrograms are the binaries and their notices. The command line answers
// with THIRD-PARTY-NOTICES.md, which its release archive carries beside LICENSE
// (DEC-324 point 5, DEC-327). The window answers with a file of its own,
// because it links the toolkit and what that brings, a C library among it, and
// the command line links two of those modules and none of the rest.
var noticedPrograms = []noticedProgram{
	{pkg: "./cmd/qadg", notices: "THIRD-PARTY-NOTICES.md", targets: releaseTargets},
	{pkg: "./cmd/qadg-gui", notices: windowNotices, targets: windowTargets, cgo: true,
		landmarks: [][2]string{
			{"github.com/go-gl/glfw/v3.4/glfw", "glfw/src/win32_init.c"}, // only an #include reaches it
			{"github.com/go-gl/glfw/v3.4/glfw", "glfw/LICENSE.md"},       // only from beside the files an #include reaches
			{"fyne.io/fyne/v2", "theme/font/LICENSE_Inter.txt"},          // only from beside the fonts go:embed takes in
			{"github.com/go-gl/gl", "LICENSE"},                           // only through the replace in go.mod, in this tree
		},
		byCompiler: toolchainHeadings()},
}

// stdHeading is the one section every notices file has that is not named after
// a module. The Go runtime and standard library are linked into every Go
// program and are not a module, so go list never names them the way it names
// the rest.
const stdHeading = "Go standard library and runtime"

// releaseTargets are the five builds DEC-324 point 1 ships the command line
// for. .github/scripts/build_release.sh builds its own list, and
// TestTheReleaseBuildsTheTargetsTheNoticesAreCheckedFor holds that list, and
// the window job's matrix, to this one and to windowTargets.
//
// All five are asked, not one, because what is linked depends on the target.
// A file the runtime builds for amd64 alone carries a notice the arm64 builds
// do not have.
var releaseTargets = [][2]string{
	{"windows", "amd64"}, {"windows", "arm64"},
	{"linux", "amd64"}, {"linux", "arm64"},
	{"darwin", "arm64"},
}

// windowTargets are the three builds DEC-324 point 1 ships the window for. It
// needs a C compiler for each, so it is not built for every target the command
// line is.
var windowTargets = [][2]string{
	{"windows", "amd64"},
	{"linux", "amd64"},
	{"darwin", "arm64"},
}

// carriedWorks are works of somebody else's that this module keeps in its own
// tree, each with the heading its notice stands under. A package of this
// module is ours and asks for no notice, so its licence files are looked for
// below the module root only, where LICENSE and LICENSE-spec are this
// project's own, and one found that this map does not name is refused:
// somebody's terms with nowhere to stand.
var carriedWorks = map[string]string{
	"internal/adapter/provider/corpus/LICENSE-faker-MIT.txt": "joke2k/faker",
	"internal/gui/assets/fonts/LICENSE-IBM-Plex-OFL.txt":     "IBM/plex",
	"internal/gui/assets/icons/LICENSE-lucide-ISC.txt":       "lucide-icons/lucide",
}

// assetPackage is the package that embeds the typefaces and icons
// license.Embedded describes. A program that links it carries them, so its
// notices name the project of each with the version the register states, and
// the heading a carried licence file stands under is that project.
const assetPackage = "github.com/donislawdev/qadatagenerator/internal/gui/assets"

// dataPackage is the package that embeds the reference lists license.Data
// describes. A program that links it carries every one of those lists, so its
// notices name every project they came from, the ones whose terms ask for no
// notice included. The sentence that a list is in the public domain is the
// answer to the question a reader of the notices is asking.
const dataPackage = "github.com/donislawdev/qadatagenerator/internal/adapter/provider/corpus"

// licenceName is a file that carries terms rather than code. Matched by name,
// because that is how a module publishes one: LICENSE beside the code, and
// sometimes a second one beside the package that came from elsewhere.
var licenceName = regexp.MustCompile(`(?i)^(licen[cs]e|copying|notice|attrib|patents|copyright|unlicense)([._-].*)?$`)

// copyrightLine is a statement of copyright in a source comment: the word, and
// then a year or a (c) or the copyright sign. Either is what tells a statement
// from a sentence ABOUT copyright, which every licence is full of.
var copyrightLine = regexp.MustCompile(`(?i)\bcopyright\s*(?:(?:\(c\)|\x{00A9})\s*\S|\d{4})`)

// gnuLicenceLine is a line naming one of the GNU licences. Code under one of
// them is what a GPL-3.0 program has to look at twice, and the notice of such
// a file may state no year, so its copyright line alone would not bring it up.
var gnuLicenceLine = regexp.MustCompile(`(?i)\bGeneral\s+Public\b`)

// commentMarker is what precedes the text of a comment line in Go, C and the
// assembler.
var commentMarker = regexp.MustCompile(`^[\s/*]*`)

// includeLine is a line that pulls another file into C, Objective-C or a cgo
// preamble, with the bracket it opens and the name. A preamble written as line
// comments carries the directive after the slashes. A name a macro builds is
// not followed.
var includeLine = regexp.MustCompile(`^\s*(?://\s*)?#\s*(?:include|import)\s*([<"])([^>"]+)[>"]`)

// statedVersion is how a section opens: the version as go list spells it,
// between backticks.
var statedVersion = regexp.MustCompile("(?m)^Version `([^`]+)`\\.")

// termsOfAnUnlinkedPart are licence files that sit in the tree of a linked
// package and cover a part of it that no release build links. Each names the
// package carrying that part, and the exception holds only while that package
// stays out of every build. The moment one links it, its licence is asked for
// like any other.
//
// One entry, measured in Cert Doctor on 2026-10-28 and here again on
// 2026-11-18. Every build links crypto/internal/boring as stubs, and its
// LICENSE says so in its first lines: the Go licence covers the Go code there,
// and the BoringSSL licence below applies only with GOEXPERIMENT=boringcrypto,
// to an object file that crypto/internal/boring/syso carries.
var termsOfAnUnlinkedPart = map[string]string{
	"src/crypto/internal/boring/LICENSE": "crypto/internal/boring/syso",
}

// linkedCode is what the builds of one program link, gathered from go list.
type linkedCode struct {
	modules  map[string]string // module path to version, this module left out
	dirs     map[string]string // module path to its directory, for the landmarks
	headings map[string]string // sections that name no module and must stand, to the version they state or ""
	licences []ownedText       // licence files in the trees of linked code
	sources  map[string]string // source file to the heading that answers for it
	strays   []string          // licence files in this module's tree no carried work names
	skipped  []string          // licence files left out, with the reason
}

// ownedText is a piece of somebody else's text and the section it belongs in.
type ownedText struct {
	path  string
	owner string
	text  string
}

// walkUp is where the search for licence files from one directory stops, and
// whose terms it finds there.
type walkUp struct {
	top   string
	owner string
	ours  bool
}

// listedPackage is the part of "go list -json" this guard reads.
type listedPackage struct {
	ImportPath  string
	Dir         string
	Standard    bool
	GoFiles     []string
	CgoFiles    []string
	CFiles      []string
	CXXFiles    []string
	MFiles      []string
	HFiles      []string
	SFiles      []string
	EmbedFiles  []string
	CgoCFLAGS   []string
	CgoCPPFLAGS []string
	CgoCXXFLAGS []string
	Module      *struct {
		Path    string
		Version string
		Dir     string
		Main    bool
	}
	Error *struct{ Err string }
}

// TestTheNoticesCoverWhatTheBinaryLinks is the guard. For each program one
// gathering and three questions: which modules and versions, which licence
// files, which copyright statements.
//
// It reads what go list says the builds compile, Go, assembler, C and
// Objective-C, and the files an #include reaches, and the licence files beside
// all of that and beside what go:embed takes in. What a C toolchain's linker
// adds of its own is not in this tree, so it is not asked here.
func TestTheNoticesCoverWhatTheBinaryLinks(t *testing.T) {
	root := rootOrFail(t)
	goroot := goEnv(t, root, "GOROOT")
	// The first word only: a toolchain built with an experiment adds it after
	// a space, and the release is not built with one.
	running := strings.Fields(goEnv(t, root, "GOVERSION"))[0]
	goModVersion := goModGo(t, root)
	reached := map[string]bool{}
	for _, program := range noticedPrograms {
		linked := gatherLinked(t, root, goroot, program, reached)
		t.Run(program.notices, func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join(root, program.notices))
			if err != nil {
				t.Fatalf("%s travels with every copy of %s, and it cannot be read: %v", program.notices, program.pkg, err)
			}
			sections, parseProblems := noticeSections(program.notices, string(body))
			requireNone(t, parseProblems, "the notices file cannot be read as sections")
			t.Logf("%s: %d module(s), %d licence file(s) and %d source file(s) across %d targets",
				program.pkg, len(linked.modules), len(linked.licences), len(linked.sources), len(program.targets))
			for _, reason := range linked.skipped {
				t.Logf("not required: %s", reason)
			}

			t.Run("modules and versions", func(t *testing.T) {
				requireNone(t, checkNoticedModules(program.notices, sections, linked.modules, linked.headings), "the modules in the notices")
				requireNone(t, checkGoVersion(program.notices, sections, goModVersion, running), "the Go version in the notices")
			})
			t.Run("licence files", func(t *testing.T) {
				requireNone(t, linked.strays, "a licence file in this module's own tree")
				requireNone(t, checkLicenceTexts(program.notices, sections, linked.licences), "a licence file of linked code")
			})
			t.Run("copyright statements", func(t *testing.T) {
				statements, err := copyrightStatements(linked.sources)
				if err != nil {
					t.Fatal(err)
				}
				requireNone(t, checkLicenceTexts(program.notices, sections, statements), "a copyright statement in linked code")
			})
		})
	}
	for _, exceptions := range []map[string]string{termsOfAnUnlinkedPart, carriedWorks} {
		for rel := range exceptions {
			if !reached[rel] {
				t.Errorf("%s is named as an exception and no walk of any program reached it, and an exception for nothing reads as protection", rel)
			}
		}
	}
	if !reached[dataPackage] {
		t.Errorf("no program links %s, so the reference lists are asked for nowhere and the name of that package here is out of date", dataPackage)
	}
	if !reached[assetPackage] {
		t.Errorf("no program links %s, so the typefaces and icons are asked for nowhere and the name of that package here is out of date", assetPackage)
	}
}

// TestTheNoticesGuardCanFail is the canary, one planted gap per question. A
// guard over a list it never managed to fill looks exactly like a guard over a
// list that is complete.
func TestTheNoticesGuardCanFail(t *testing.T) {
	const file = "PLANTED.md"
	sections, problems := noticeSections(file, "# Notices\n\n## "+stdHeading+"\n\nVersion `go1.0`. Some text.\n\n"+
		"## example.com/kept\n\nVersion `v1.0.0`. MIT.\n\n```\nCopyright (c) 2001 Somebody\n```\n\n"+
		"## example.com/gone\n\nVersion `v2.0.0`. MIT.\n\n"+
		"## A carried work\n\nSome terms.\n\n"+
		"## A data project\n\nVersion `2020 vintage`. Public domain.\n")
	requireNone(t, problems, "the planted notices")

	linked := map[string]string{"example.com/kept": "v1.1.0", "example.com/new": "v1.0.0"}
	required := map[string]string{stdHeading: "", "A carried work": "", "A data project": "2020 vintage"}
	modules := checkNoticedModules(file, sections, linked, required)
	if len(modules) != 3 {
		t.Errorf("want exactly the new module, the gone one and the wrong version, got %v", modules)
	}
	requireFinding(t, modules, "example.com/new is linked")
	requireFinding(t, modules, "example.com/gone and nothing links it")
	requireFinding(t, modules, "states v1.0.0 and the build links v1.1.0")
	requireFinding(t, checkNoticedModules(file, sections, linked, map[string]string{stdHeading: ""}), "A carried work and nothing links it")
	requireFinding(t, checkNoticedModules(file, sections, linked, map[string]string{stdHeading: "", "A missing project": ""}),
		"A missing project has to stand here")
	requireFinding(t, checkNoticedModules(file, sections, linked, map[string]string{stdHeading: "", "A data project": "2021 vintage"}),
		"states 2020 vintage and the build carries 2021 vintage")

	requireFinding(t, checkGoVersion(file, sections, "1.27.4", "go1.27.4"), "states go1.0")
	requireFinding(t, checkGoVersion(file, sections, "1.27.4", "go1.28.0"), "runs go1.28.0")

	texts := checkLicenceTexts(file, sections, []ownedText{
		{path: "kept/LICENSE", owner: "example.com/kept", text: "Copyright (c) 2001 Somebody\n"},
		{path: "kept/ATTRIB", owner: "example.com/kept", text: "Copyright (c) 2002 Somebody Else\n"},
		{path: "std/LICENSE", owner: stdHeading, text: "Copyright (c) 2001 Somebody\n"},
	})
	if len(texts) != 2 {
		t.Errorf("want exactly the missing ATTRIB and the text quoted under the wrong heading, got %v", texts)
	}
	requireFinding(t, texts, "kept/ATTRIB")
	requireFinding(t, texts, "std/LICENSE")

	boring := "src/crypto/internal/boring/LICENSE"
	if why := notRequired(boring, "BoringSSL terms", "grant", map[string]bool{"crypto/internal/boring/syso": true}); why != "" {
		t.Errorf("the BoringSSL terms were let off with their object file linked: %s", why)
	}
	if notRequired(boring, "BoringSSL terms", "grant", map[string]bool{}) == "" {
		t.Error("the BoringSSL terms were asked for with their object file linked nowhere")
	}
	if notRequired("", "conditions of somebody's own", "grant", nil) != "" {
		t.Error("a file was let off that is neither the grant nor an unlinked part")
	}

	if heading, strays := carriedHeading(file, "internal/adapter/provider/corpus/LICENSE-faker-MIT.txt"); heading != "joke2k/faker" || strays != nil {
		t.Errorf("the faker licence stands under %q with %v, want under joke2k/faker", heading, strays)
	}
	_, strays := carriedHeading(file, "internal/planted/LICENSE")
	requireFinding(t, strays, "carriedWorks names no heading for")

	face := []license.Asset{{Asset: "A face", Project: "someone/face", Version: "1.0", Notice: "LICENSE-face.txt"}}
	if headings, problems := embeddedHeadings(face, map[string]string{"x/LICENSE-face.txt": "someone/face"}); headings["someone/face"] != "1.0" || problems != nil {
		t.Errorf("an asset whose notice stands under its project gave %v and %v, want version 1.0 and no finding", headings, problems)
	}
	_, elsewhere := embeddedHeadings(face, map[string]string{"x/LICENSE-face.txt": "somebody else"})
	requireFinding(t, elsewhere, "rather than under someone/face alone")
	_, nowhere := embeddedHeadings(face, map[string]string{})
	requireFinding(t, nowhere, "names the notice LICENSE-face.txt")

	_, dup := noticeSections(file, "## a\n\n## a\n")
	requireFinding(t, dup, "two sections named a")
	_, fenced := noticeSections(file, "## a\n\n```\n## not a heading\n")
	requireFinding(t, fenced, "a code block that never closes")
}

// TestAnIncludeIsFollowedTheWayTheCompilerLooks pins the road the window's C
// library is reached by. It is compiled by a cgo preamble that includes its .c
// files, and those include the rest, so the notices of all of it are only seen
// if every #include is followed to where the compiler would find it.
func TestAnIncludeIsFollowedTheWayTheCompilerLooks(t *testing.T) {
	base := t.TempDir()
	top := filepath.Join(base, "module")
	pkg := filepath.Join(top, "pkg")
	for name, text := range map[string]string{
		"pkg/a.go":         "package a\n\n/*\n#include \"src/one.c\"\n#include <stdlib.h>\n*/\nimport \"C\"\n",
		"pkg/b.go":         "package a\n\n// #include \"src/four.h\"\nimport \"C\"\n",
		"pkg/src/one.c":    "#include \"one.h\"\n#include <two.h>\n#include <beside.h>\n#include \"../../../outside.h\"\n",
		"pkg/src/one.h":    "\n",
		"pkg/src/beside.h": "\n",
		"pkg/src/four.h":   "\n",
		"pkg/inc/two.h":    "  #  import \"three.h\"\n",
		"pkg/inc/three.h":  "#include \"one.h\"\n",
		"../outside.h":     "\n",
	} {
		path := filepath.Join(top, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	dirs := includeDirs(pkg, []string{"-D_X", "-Iinc"}, []string{"-I", filepath.Join(pkg, "missing")})
	if want := []string{filepath.Join(pkg, "inc"), filepath.Join(pkg, "missing")}; fmt.Sprint(dirs) != fmt.Sprint(want) {
		t.Errorf("read the include directories %v out of the flags, want %v", dirs, want)
	}
	got, err := includedFiles([]string{filepath.Join(pkg, "a.go"), filepath.Join(pkg, "b.go")}, dirs, top)
	if err != nil {
		t.Fatal(err)
	}
	var rel []string
	for _, path := range got {
		rel = append(rel, relativeTo(top, path))
	}
	// beside.h is next to one.c but asked for in angle brackets, which do not
	// look there. outside.h exists and is not this module's.
	want := []string{"pkg/inc/three.h", "pkg/inc/two.h", "pkg/src/four.h", "pkg/src/one.c", "pkg/src/one.h"}
	if fmt.Sprint(rel) != fmt.Sprint(want) {
		t.Errorf("followed the includes to %v, want %v", rel, want)
	}
}

// copyrightSign is the sign as UTF-8 bytes, spelled so that this file stays
// ASCII. Inferno's notice in the runtime is written with it.
const copyrightSign = "\xc2\xa9"

// TestACopyrightStatementIsReadOffAComment pins what the third question picks
// up, because a statement it fails to see is a notice nobody is asked for.
func TestACopyrightStatementIsReadOffAComment(t *testing.T) {
	seen := map[string]bool{}
	for _, line := range []string{
		"// Copyright (C) 1993 by Sun Microsystems, Inc. All rights reserved.",
		" * Copyright (c) 2001 David E. O'Brien",
		"//         Revisions Copyright " + copyrightSign + " 2000-2007 Vita Nuova Holdings Limited.  All rights reserved.",
		" * Copyright (C) the Wine project",
		" * modify it under the terms of the GNU Lesser General Public",
	} {
		text, ok := copyrightStatement(line)
		seen[text] = ok
	}
	for _, want := range []string{
		"Copyright (C) 1993 by Sun Microsystems, Inc. All rights reserved.",
		"Copyright (c) 2001 David E. O'Brien",
		"Revisions Copyright " + copyrightSign + " 2000-2007 Vita Nuova Holdings Limited. All rights reserved.",
		"Copyright (C) the Wine project",
		"modify it under the terms of the GNU Lesser General Public",
	} {
		if !seen[want] {
			t.Errorf("did not read %q off its comment, read %v", want, seen)
		}
	}
	for _, line := range []string{
		"// Copyright 2009 The Go Authors. All rights reserved.",
		"// Redistributions of source code must retain the above copyright",
		"var copyright = 1",
	} {
		if text, ok := copyrightStatement(line); ok {
			t.Errorf("read %q as a statement somebody else's notice has to carry", text)
		}
	}
}

// noticeSections cuts a notices file at its second-level headings. A heading
// inside a code block is licence text, not a heading, so fences are followed.
func noticeSections(file, body string) (map[string]string, []string) {
	sections := map[string]string{}
	var problems []string
	var current string
	var text strings.Builder
	inFence, open := false, false
	flush := func() {
		if open {
			sections[current] = text.String()
		}
		text.Reset()
	}
	for i, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
		}
		if heading, ok := strings.CutPrefix(line, "## "); ok && !inFence {
			flush()
			current, open = strings.TrimSpace(heading), true
			if _, dup := sections[current]; dup {
				problems = append(problems, fmt.Sprintf("%s:%d: two sections named %s", file, i+1, current))
			}
			continue
		}
		text.WriteString(line + "\n")
	}
	flush()
	if inFence {
		problems = append(problems, file+": the file ends inside a code block that never closes")
	}
	return sections, problems
}

// checkNoticedModules compares the sections with what is linked, both ways,
// and the version each section states with the one the build links. required
// are the headings that name no module and must stand all the same, the
// standard library, a carried work the walk reached and a project the
// reference lists came from, each with the version its section states, or
// nothing where the section states none that this check reads.
func checkNoticedModules(file string, sections map[string]string, modules, required map[string]string) []string {
	var problems []string
	for path, version := range modules {
		section, ok := sections[path]
		if !ok {
			problems = append(problems, fmt.Sprintf(
				"%s: %s is linked into the binary and has no section here. Read its licence files in the module "+
					"cache and add \"## %s\", \"Version `%s`.\" and each file in full", file, path, path, version))
			continue
		}
		if stated := versionIn(section); stated != version {
			problems = append(problems, fmt.Sprintf(
				"%s: the section for %s states %s and the build links %s, and a notice belongs to the code that ships",
				file, path, stated, version))
		}
	}
	for heading, version := range required {
		section, ok := sections[heading]
		if !ok {
			problems = append(problems, fmt.Sprintf("%s: %s has to stand here, because the binary carries it, and there is no \"## %s\"", file, heading, heading))
			continue
		}
		if stated := versionIn(section); version != "" && stated != version {
			problems = append(problems, fmt.Sprintf("%s: the section for %s states %s and the build carries %s", file, heading, stated, version))
		}
	}
	for heading := range sections {
		_, linked := modules[heading]
		_, carried := required[heading]
		if !linked && !carried {
			problems = append(problems, fmt.Sprintf(
				"%s: the notices name %s and nothing links it, which is a section for code that does not ship", file, heading))
		}
	}
	sort.Strings(problems)
	return problems
}

// checkGoVersion asks the standard library section for the Go that builds the
// release, the go line of go.mod, which CI installs, and refuses to judge it
// with a different Go, whose standard library may carry other notices.
func checkGoVersion(file string, sections map[string]string, goModGo, running string) []string {
	want := "go" + goModGo
	if running != want {
		return []string{fmt.Sprintf(
			"%s: this guard runs %s and the release is built with %s from go.mod. The notices of one say "+
				"nothing about the other, so run it with that one", file, running, want)}
	}
	section, ok := sections[stdHeading]
	if !ok {
		return []string{fmt.Sprintf("%s: there is no section \"## %s\", and every build links it", file, stdHeading)}
	}
	if stated := versionIn(section); stated != want {
		return []string{fmt.Sprintf("%s: the standard library section states %s and the release is built with %s", file, stated, want)}
	}
	return nil
}

// checkLicenceTexts asks that every piece of somebody else's text stands, in
// full, in the section of the code it came with. Under another heading it does
// not count: the reader looks for a notice where its code is named.
//
// Whitespace runs are folded on both sides. A licence file in the module cache
// may end its lines differently than the notices do, and a comment loses its
// indentation when it is quoted. The words and their order are what is asked.
func checkLicenceTexts(file string, sections map[string]string, texts []ownedText) []string {
	folded := map[string]string{}
	for heading, section := range sections {
		folded[heading] = fold(section)
	}
	var problems []string
	for _, piece := range texts {
		if strings.Contains(folded[piece.owner], fold(piece.text)) {
			continue
		}
		problems = append(problems, fmt.Sprintf(
			"%s: %s is not quoted in full under \"## %s\". Copy it there inside a code block, as it stands", file, piece.path, piece.owner))
	}
	sort.Strings(problems)
	return problems
}

// carriedHeading names the section a licence file in this module's own tree
// stands under, and refuses one that no carried work names.
func carriedHeading(file, rel string) (string, []string) {
	if heading, ok := carriedWorks[rel]; ok {
		return heading, nil
	}
	return "", []string{fmt.Sprintf("%s: a licence file in this module's own tree, and carriedWorks names no heading for it. "+
		"Those are somebody's terms that %s leaves out", rel, file)}
}

func fold(s string) string { return strings.Join(strings.Fields(s), " ") }

func versionIn(section string) string {
	if m := statedVersion.FindStringSubmatch(section); m != nil {
		return m[1]
	}
	return "no version"
}

// gatherLinked asks go list what one program links on every target it ships
// for. reached collects the exceptions the walks arrived at, across programs.
func gatherLinked(t *testing.T, root, goroot string, program noticedProgram, reached map[string]bool) linkedCode {
	t.Helper()
	code := linkedCode{modules: map[string]string{}, dirs: map[string]string{},
		headings: map[string]string{stdHeading: ""}, sources: map[string]string{}}
	walks, imported := code.gatherSources(t, root, goroot, program)
	found := code.gatherLicences(t, root, goroot, program, walks, imported, reached)
	if !found[filepath.Join(goroot, "LICENSE")] {
		t.Fatalf("the walk for %s never reached %s, so it proves nothing about the files above any package", program.pkg, filepath.Join(goroot, "LICENSE"))
	}
	if imported[dataPackage] {
		reached[dataPackage] = true
		code.requireDataProjects(t)
	}
	if imported[assetPackage] {
		reached[assetPackage] = true
		headings, problems := embeddedHeadings(license.Embedded, carriedWorks)
		requireNone(t, problems, "the register of embedded assets against the carried works")
		for heading, version := range headings {
			code.headings[heading] = version
		}
	}
	// No version here: the pin states it, and toolchain_test.go holds the
	// section to the pin.
	for _, heading := range program.byCompiler {
		code.headings[heading] = ""
	}
	for _, mark := range program.landmarks {
		dir, linked := code.dirs[mark[0]]
		path := filepath.Join(dir, filepath.FromSlash(mark[1]))
		if _, read := code.sources[path]; !linked || !(read || found[path]) {
			t.Errorf("the gathering for %s never reached %s in %s. The road to it is lost, and what was quoted before passes for proof",
				program.pkg, mark[1], mark[0])
		}
	}
	sort.Slice(code.licences, func(i, j int) bool { return code.licences[i].path < code.licences[j].path })
	sort.Strings(code.skipped)
	return code
}

// requireDataProjects asks for a section for every project license.Data names,
// with the version the lists were read at. Two lists of one project read at two
// versions cannot share one section, so that is refused here rather than left
// for the section to state one of them.
func (code *linkedCode) requireDataProjects(t *testing.T) {
	t.Helper()
	for _, source := range license.Data {
		if stated, ok := code.headings[source.Project]; ok && stated != "" && stated != source.Version {
			t.Errorf("license.Data reads %s at %s and at %s, and one section of the notices cannot state both", source.Project, stated, source.Version)
		}
		code.headings[source.Project] = source.Version
	}
}

// embeddedHeadings answers with the section every embedded asset has to stand
// under, its project, with the version the register states. The carried work
// whose licence file is the asset's notice has to name the same project, or
// the licence would be asked for under one heading and the version under
// another.
func embeddedHeadings(assets []license.Asset, carried map[string]string) (map[string]string, []string) {
	headings := map[string]string{}
	var problems []string
	for _, asset := range assets {
		headings[asset.Project] = asset.Version
		var named []string
		for rel, heading := range carried {
			if filepath.Base(filepath.FromSlash(rel)) == asset.Notice {
				named = append(named, heading)
			}
		}
		if len(named) != 1 || named[0] != asset.Project {
			problems = append(problems, fmt.Sprintf("%s names the notice %s, and the carried works put that file under %v rather than under %s alone",
				asset.Asset, asset.Notice, named, asset.Project))
		}
	}
	sort.Strings(problems)
	return headings, problems
}

// gatherSources reads the packages of every target into code, and answers with
// the directories to look for licence files from and the packages imported.
func (code *linkedCode) gatherSources(t *testing.T, root, goroot string, program noticedProgram) (map[string]walkUp, map[string]bool) {
	t.Helper()
	walks := map[string]walkUp{} // directory to where its walk stops
	imported := map[string]bool{}
	for _, target := range program.targets {
		for _, p := range listPackages(t, root, program, target) {
			imported[p.ImportPath] = true
			embedded := dirsOf(joined(p.Dir, p.EmbedFiles))
			if !p.Standard && p.Module == nil {
				t.Fatalf("%s is neither in the standard library nor in a module", p.ImportPath)
			}
			if !p.Standard && p.Module.Main {
				for _, dir := range append([]string{p.Dir}, embedded...) {
					walks[dir] = walkUp{top: root, ours: true}
				}
				continue
			}
			owner, top := stdHeading, goroot
			if !p.Standard {
				code.modules[p.Module.Path] = p.Module.Version
				code.dirs[p.Module.Path] = p.Module.Dir
				owner, top = p.Module.Path, p.Module.Dir
			}
			compiled := joined(p.Dir, p.CgoFiles, p.CFiles, p.CXXFiles, p.MFiles, p.HFiles)
			included, err := includedFiles(compiled, includeDirs(p.Dir, p.CgoCFLAGS, p.CgoCPPFLAGS, p.CgoCXXFLAGS), top)
			if err != nil {
				t.Fatalf("following the includes of %s: %v", p.ImportPath, err)
			}
			for _, path := range append(append(joined(p.Dir, p.GoFiles, p.SFiles), compiled...), included...) {
				code.sources[path] = owner
			}
			for _, dir := range append(append([]string{p.Dir}, dirsOf(included)...), embedded...) {
				walks[dir] = walkUp{top: top, owner: owner}
			}
		}
	}
	return walks, imported
}

// gatherLicences walks up from every directory for licence files, sorts each
// into quoted, let off or refused, and answers with every file it found.
func (code *linkedCode) gatherLicences(t *testing.T, root, goroot string, program noticedProgram,
	walks map[string]walkUp, imported, reached map[string]bool) map[string]bool {
	t.Helper()
	grant := readText(t, filepath.Join(goroot, "PATENTS"))
	found := map[string]bool{}
	for dir, walk := range walks {
		for _, path := range licenceFilesUp(t, dir, walk.top) {
			if found[path] {
				continue
			}
			found[path] = true
			if walk.ours {
				code.ourLicence(t, root, program.notices, path, reached)
				continue
			}
			rel := relativeTo(goroot, path)
			if _, ok := termsOfAnUnlinkedPart[rel]; ok {
				reached[rel] = true
			}
			text := readText(t, path)
			if why := notRequired(rel, text, grant, imported); why != "" {
				code.skipped = append(code.skipped, path+" "+why)
				continue
			}
			code.licences = append(code.licences, ownedText{path: path, owner: walk.owner, text: text})
		}
	}
	return found
}

// ourLicence sorts a licence file found in this module's own tree: this
// project's own licences at the root, a carried work's under its heading, and
// anything else refused.
func (code *linkedCode) ourLicence(t *testing.T, root, notices, path string, reached map[string]bool) {
	t.Helper()
	if filepath.Dir(path) == filepath.Clean(root) {
		return // LICENSE and LICENSE-spec, this project's own
	}
	if filepath.Ext(path) == ".go" {
		return // a file of ours named after what it is about
	}
	rel := relativeTo(root, path)
	heading, strays := carriedHeading(notices, rel)
	if strays != nil {
		code.strays = append(code.strays, strays...)
		return
	}
	reached[rel] = true
	if _, ok := code.headings[heading]; !ok {
		code.headings[heading] = ""
	}
	code.licences = append(code.licences, ownedText{path: path, owner: heading, text: readText(t, path)})
}

// goListAnswers keeps what go list answered, by program and build, for the run
// of this package's tests. Each answer costs a second or so, and a later guard
// asking the same question gets it from here.
var goListAnswers = struct {
	sync.Mutex
	answers map[string][]listedPackage
}{answers: map[string][]listedPackage{}}

func listPackages(t *testing.T, root string, program noticedProgram, target [2]string) []listedPackage {
	t.Helper()
	key := strings.Join([]string{root, program.pkg, target[0], target[1], fmt.Sprint(program.cgo)}, "|")
	goListAnswers.Lock()
	defer goListAnswers.Unlock()
	if answer, ok := goListAnswers.answers[key]; ok {
		return answer
	}
	answer := askGoList(t, root, program, target)
	goListAnswers.answers[key] = answer
	return answer
}

// askGoList is go list asked once, for one program and one build.
func askGoList(t *testing.T, root string, program noticedProgram, target [2]string) []listedPackage {
	t.Helper()
	cgo := "0"
	if program.cgo {
		cgo = "1"
	}
	cmd := exec.Command("go", "list", "-deps", "-json", program.pkg)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOOS="+target[0], "GOARCH="+target[1], "CGO_ENABLED="+cgo)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list %s for %s/%s: %v\n%s", program.pkg, target[0], target[1], err, stderr.String())
	}
	var packages []listedPackage
	for dec := json.NewDecoder(bytes.NewReader(out)); dec.More(); {
		var p listedPackage
		if err := dec.Decode(&p); err != nil {
			t.Fatalf("reading go list %s for %s/%s: %v", program.pkg, target[0], target[1], err)
		}
		if p.Error != nil {
			t.Fatalf("go list could not load %s for %s/%s: %s", p.ImportPath, target[0], target[1], p.Error.Err)
		}
		packages = append(packages, p)
	}
	if len(packages) == 0 {
		t.Fatalf("go list named no package of %s for %s/%s", program.pkg, target[0], target[1])
	}
	return packages
}

// includeDirs reads the -I directories out of a package's cgo flags. go list
// gives them with ${SRCDIR} spelled out, and one written relative to the
// package is relative to its directory, where cgo runs the compiler.
func includeDirs(pkgDir string, flags ...[]string) []string {
	var dirs []string
	for _, list := range flags {
		for i, flag := range list {
			dir, ok := strings.CutPrefix(flag, "-I")
			if !ok {
				continue
			}
			if dir == "" && i+1 < len(list) {
				dir = list[i+1]
			}
			if dir == "" {
				continue
			}
			if !filepath.IsAbs(dir) {
				dir = filepath.Join(pkgDir, dir)
			}
			dirs = append(dirs, dir)
		}
	}
	return dirs
}

// includedFiles follows #include and #import from the given files to every
// file of the module they reach, looking where the compiler looks: a quoted
// name beside the file that asks for it first, then each -I directory, and a
// name in angle brackets in those directories alone. A name found nowhere in
// the module is a system header, which comes with the platform and not with
// this program.
//
// There is no preprocessor here, so an #include inside an #if for another
// platform is followed too. That asks for more notices than a build links,
// never for fewer, and a notices file answers for all its targets at once.
func includedFiles(start, dirs []string, top string) ([]string, error) {
	seen := map[string]bool{}
	for _, path := range start {
		seen[path] = true
	}
	queue := append([]string{}, start...)
	var reached []string
	for len(queue) > 0 {
		file := queue[0]
		queue = queue[1:]
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", file, err)
		}
		for _, line := range strings.Split(string(content), "\n") {
			m := includeLine.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			look := dirs
			if m[1] == `"` {
				look = append([]string{filepath.Dir(file)}, dirs...)
			}
			path, ok := findIn(look, m[2], top)
			if !ok || seen[path] {
				continue
			}
			seen[path] = true
			reached = append(reached, path)
			queue = append(queue, path)
		}
	}
	sort.Strings(reached)
	return reached, nil
}

// findIn answers with the first of dirs that holds name as a file inside top.
func findIn(dirs []string, name, top string) (string, bool) {
	for _, dir := range dirs {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if relativeTo(top, path) == "" {
			continue
		}
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			return path, true
		}
	}
	return "", false
}

// joined spells the names of files in dir as paths.
func joined(dir string, lists ...[]string) []string {
	var paths []string
	for _, names := range lists {
		for _, name := range names {
			paths = append(paths, filepath.Join(dir, filepath.FromSlash(name)))
		}
	}
	return paths
}

// dirsOf lists the directories the given files sit in, each once.
func dirsOf(paths []string) []string {
	seen := map[string]bool{}
	var dirs []string
	for _, path := range paths {
		if dir := filepath.Dir(path); !seen[dir] {
			seen[dir] = true
			dirs = append(dirs, dir)
		}
	}
	return dirs
}

// licenceFilesUp lists the licence files in a directory and in every directory
// above it, up to and including top. A licence beside the module's go.mod
// covers every package below it, and one beside a package covers that package.
func licenceFilesUp(t *testing.T, dir, top string) []string {
	t.Helper()
	var found []string
	for d := filepath.Clean(dir); ; {
		entries, err := os.ReadDir(d)
		if err != nil {
			t.Fatalf("listing %s: %v", d, err)
		}
		for _, e := range entries {
			if e.Type().IsRegular() && licenceName.MatchString(e.Name()) {
				found = append(found, filepath.Join(d, e.Name()))
			}
		}
		if d == filepath.Clean(top) {
			return found
		}
		parent := filepath.Dir(d)
		if parent == d {
			t.Fatalf("%s is not below %s, so the walk for its licence files had nowhere to stop", dir, top)
		}
		d = parent
	}
}

// copyrightStatements reads every statement of copyright out of the linked
// source files, other than the Go Authors' own, whose licence is quoted whole.
func copyrightStatements(sources map[string]string) ([]ownedText, error) {
	var statements []ownedText
	for path, owner := range sources {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		for i, line := range strings.Split(string(content), "\n") {
			if text, ok := copyrightStatement(line); ok {
				statements = append(statements, ownedText{path: fmt.Sprintf("%s:%d (%q)", path, i+1, text), owner: owner, text: text})
			}
		}
	}
	sort.Slice(statements, func(i, j int) bool { return statements[i].path < statements[j].path })
	return statements, nil
}

// copyrightStatement reads one line, and says whether it states somebody
// else's copyright or names a GNU licence, with the comment markers taken off.
func copyrightStatement(line string) (string, bool) {
	if !(copyrightLine.MatchString(line) || gnuLicenceLine.MatchString(line)) || strings.Contains(line, "The Go Authors") {
		return "", false
	}
	return fold(commentMarker.ReplaceAllString(line, "")), true
}

// notRequired says why a licence file found in the tree of linked code need not
// be quoted, and answers with nothing when it must be. rel is the path below
// the Go distribution, empty for a file outside it.
//
// The grant is matched by its text and not by the name PATENTS, because a file
// of that name in somebody else's module may set conditions of its own.
func notRequired(rel, text, grant string, imported map[string]bool) string {
	if text == grant {
		return "is Go's patent grant, which sets no condition on passing it on"
	}
	if part, ok := termsOfAnUnlinkedPart[rel]; ok && !imported[part] {
		return "covers " + part + ", which no release build links"
	}
	return ""
}

// relativeTo spells a path below root the way termsOfAnUnlinkedPart does, and
// answers with nothing for a path outside it.
func relativeTo(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return ""
	}
	return filepath.ToSlash(rel)
}

func readText(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return strings.ReplaceAll(string(content), "\r\n", "\n")
}

func goEnv(t *testing.T, root, name string) string {
	t.Helper()
	cmd := exec.Command("go", "env", name)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go env %s: %v", name, err)
	}
	value := strings.TrimSpace(string(out))
	if value == "" {
		t.Fatalf("go env %s answered nothing", name)
	}
	return value
}

func goModGo(t *testing.T, root string) string {
	t.Helper()
	m := regexp.MustCompile(`(?m)^go (\S+)\s*$`).FindStringSubmatch(readText(t, filepath.Join(root, "go.mod")))
	if m == nil {
		t.Fatal("go.mod has no go line")
	}
	return m[1]
}
