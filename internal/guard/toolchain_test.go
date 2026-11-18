package guard

import (
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
)

// The window is built with cgo, so the C compiler's linker puts code of its own
// into it: start up code and a runtime that come with the compiler on the
// runner, not from this tree, and that no build record names (DEC-327). The
// notices guard reads what go list says is compiled and cannot see them.
//
// Three places describe that code, and each is read by something else: the pin,
// which window_toolchain.sh holds a build of the window to by the linker's own
// map, the notices, which a person reads, and ci.yml, whose test job builds the
// window with that map and runs the reader on all three systems. A library
// added to the pin alone would ship code whose licence nobody read, and a
// reader no build runs would hold nothing to anything.
//
// Adapted on 2026-11-18 from the same guard in Cert Doctor, a project of the
// same owner under the same licence.
const (
	toolchainPin    = ".github/window-toolchain"
	toolchainReader = ".github/scripts/window_toolchain.sh"
	windowBuilder   = ".github/workflows/ci.yml"
	windowNotices   = "THIRD-PARTY-NOTICES-GUI.md"
)

// A toolchainLibrary is one library the pin may name, with the section of the
// window's notices that answers for it, or with why none has to.
type toolchainLibrary struct {
	heading string
	why     string
}

// toolchainLibraries are the libraries the pin may name. A library this map
// does not know is refused, so the first build that links code of a new one
// asks somebody to read its licence before it can say whose it is.
var toolchainLibraries = map[string]toolchainLibrary{
	// MinGW-w64 wrote COPYING.MinGW-w64-runtime.txt for a program statically
	// linked against its runtime, and asks that the notices in it travel with
	// it (the first lines of that file at version 13.0.0).
	"mingw-w64": {heading: "MinGW-w64 runtime"},
	// Two of the eight files the start files are built from are
	// LGPL-2.1-or-later without the permission to link them anywhere that the
	// other six give, and section 6 of that licence asks for a notice with a
	// copy linked into another program.
	"glibc": {heading: "GNU C Library"},
	// The GCC Runtime Library Exception, version 3.1, section 1: a combination
	// of GCC's runtime with independent modules may be conveyed "under terms of
	// your choice". Read on 2026-11-18 in COPYING.RUNTIME of GCC 15.2.0 and in
	// the copyright file of GCC 15 in Ubuntu 26.04. The opening of the notices
	// says so and quotes nothing.
	"gcc": {why: "the GCC Runtime Library Exception lets a program carry GCC's runtime under terms of its own"},
}

// toolchainHeadings are the sections of the window's notices that answer for
// the compiler's own code, sorted.
func toolchainHeadings() []string {
	var out []string
	for _, lib := range toolchainLibraries {
		if lib.heading != "" {
			out = append(out, lib.heading)
		}
	}
	sort.Strings(out)
	return out
}

// pinnedToolchain is what the pin says, for each target the window is built for.
type pinnedToolchain struct {
	links    map[string]map[string]string // target to file to library
	versions map[string]map[string]string // target to library to version
	nothing  map[string]bool
}

// toolchainLineFields is how many fields each kind of line of the pin has.
var toolchainLineFields = map[string]int{"link": 4, "version": 4, "nothing": 2}

func parseToolchainPin(rel, body string) (pinnedToolchain, []string) {
	pin := pinnedToolchain{links: map[string]map[string]string{}, versions: map[string]map[string]string{}, nothing: map[string]bool{}}
	var problems []string
	at := func(line int, format string, args ...any) {
		problems = append(problems, fmt.Sprintf("%s:%d: %s", rel, line, fmt.Sprintf(format, args...)))
	}
	for i, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		want := toolchainLineFields[fields[0]]
		switch {
		case want == 0:
			at(i+1, "has a %s line, and a line is link, version or nothing", fields[0])
		case len(fields) != want:
			at(i+1, "has %d fields where a %s line has %d", len(fields), fields[0], want)
		case !isWindowTarget(fields[1]):
			at(i+1, "names %s, which is not a target the window is built for", fields[1])
		default:
			if why := pin.take(fields); why != "" {
				at(i+1, "%s", why)
			}
		}
	}
	for _, target := range windowTargets {
		for _, why := range pin.gaps(target[0] + "/" + target[1]) {
			at(0, "%s", why)
		}
	}
	return pin, problems
}

// take records one line of the pin, or says why it cannot.
func (p pinnedToolchain) take(fields []string) string {
	target := fields[1]
	switch fields[0] {
	case "nothing":
		if p.nothing[target] {
			return "says twice that " + target + " links nothing of the compiler's"
		}
		p.nothing[target] = true
	case "link":
		file, lib := fields[2], fields[3]
		if _, known := toolchainLibraries[lib]; !known {
			return fmt.Sprintf("says %s is of %s, a library nobody has read the licence of. Add it to toolchainLibraries first", file, lib)
		}
		return setOnce(p.links, target, file, lib)
	case "version":
		lib, version := fields[2], fields[3]
		if toolchainLibraries[lib].heading == "" {
			return fmt.Sprintf("names a version of %s, and no section of the notices quotes its texts", lib)
		}
		return setOnce(p.versions, target, lib, version)
	}
	return ""
}

func setOnce(into map[string]map[string]string, target, key, value string) string {
	if into[target] == nil {
		into[target] = map[string]string{}
	}
	if _, twice := into[target][key]; twice {
		return fmt.Sprintf("names %s for %s twice", key, target)
	}
	into[target][key] = value
	return ""
}

// gaps are what the pin leaves unsaid about one target. It has to say what the
// compiler links there or that it links nothing, and name the version of every
// library with notices of its own that it links, and no version of one it does
// not.
func (p pinnedToolchain) gaps(target string) []string {
	var out []string
	linked := map[string]bool{}
	for _, lib := range p.links[target] {
		linked[lib] = true
	}
	switch {
	case len(linked) > 0 && p.nothing[target]:
		out = append(out, "says both that "+target+" links files of the compiler's and that it links nothing")
	case len(linked) == 0 && !p.nothing[target]:
		out = append(out, "says nothing about "+target+", and window_toolchain.sh refuses a build it was not written for")
	}
	for lib := range linked {
		if toolchainLibraries[lib].heading != "" && p.versions[target][lib] == "" {
			out = append(out, fmt.Sprintf("links %s into %s and names no version of it, and its notices quote the texts of one", lib, target))
		}
	}
	for lib := range p.versions[target] {
		if !linked[lib] {
			out = append(out, fmt.Sprintf("names a version of %s for %s and links nothing of it there", lib, target))
		}
	}
	sort.Strings(out)
	return out
}

func isWindowTarget(target string) bool {
	for _, t := range windowTargets {
		if t[0]+"/"+t[1] == target {
			return true
		}
	}
	return false
}

// checkToolchainNotices holds the pin to the window's notices: every library
// a build links that has notices of its own has its section, at the version the
// pin names, and no such section stands for a library no build links.
func checkToolchainNotices(pin pinnedToolchain, sections map[string]string) []string {
	var problems []string
	at := func(format string, args ...any) {
		problems = append(problems, windowNotices+": "+fmt.Sprintf(format, args...))
	}
	pinned := map[string]string{} // heading to the version the pin names
	for _, target := range slices.Sorted(maps.Keys(pin.versions)) {
		for lib, version := range pin.versions[target] {
			heading := toolchainLibraries[lib].heading
			if earlier, ok := pinned[heading]; ok && earlier != version {
				at("%s is pinned at %s on one target and %s on another, and one section quotes one version", lib, earlier, version)
			}
			pinned[heading] = version
		}
	}
	for _, heading := range toolchainHeadings() {
		version, isPinned := pinned[heading]
		section, present := sections[heading]
		switch {
		case isPinned && !present:
			at("a build links code the section \"## %s\" answers for, and there is no such section", heading)
		case !isPinned && present:
			at("has the section \"## %s\", and %s says no build links that code", heading, toolchainPin)
		case isPinned && versionIn(section) != version:
			at("the section %q states %s, and %s pins %s", heading, versionIn(section), toolchainPin, version)
		}
	}
	return problems
}

// checkToolchainWiring holds the build to the pin: the build of the window asks
// the linker for its map on both linkers and runs the reader on it, the reader
// reads the pin, and no other file under .github copies the list.
func checkToolchainWiring(builder, reader string, pin pinnedToolchain, others map[string]string) []string {
	var problems []string
	for _, want := range []string{"bash " + toolchainReader + " ", "-extldflags=-Wl,-Map=", "-extldflags=-Wl,-map,"} {
		if !strings.Contains(builder, want) {
			problems = append(problems, fmt.Sprintf("%s: does not carry %q, so the linker's map of the window is not read the way the pin needs", windowBuilder, want))
		}
	}
	if !strings.Contains(reader, `pin="`+toolchainPin+`"`) {
		problems = append(problems, fmt.Sprintf("%s: does not read %s, so it holds the build to a list of its own", toolchainReader, toolchainPin))
	}
	for _, rel := range slices.Sorted(maps.Keys(others)) {
		for _, target := range slices.Sorted(maps.Keys(pin.links)) {
			for _, file := range slices.Sorted(maps.Keys(pin.links[target])) {
				if strings.HasSuffix(file, ".a") && strings.Contains(others[rel], file) {
					problems = append(problems, fmt.Sprintf("%s: names %s, a second copy of %s that can drift from it", rel, file, toolchainPin))
				}
			}
		}
	}
	return problems
}

// githubFiles are the files under .github other than the pin, by path from the
// root with forward slashes. A workflow or script that spelled out the list
// would be a second copy of it.
func githubFiles(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	walkErr := filepath.WalkDir(filepath.Join(root, ".github"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		rel := relativeTo(root, path)
		if rel == toolchainPin {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[rel] = string(body)
		return nil
	})
	if walkErr != nil {
		t.Fatalf("cannot read the files under .github: %v", walkErr)
	}
	if _, ok := out[windowBuilder]; !ok {
		t.Fatalf("the walk of .github never reached %s, so it proves nothing about the files it read", windowBuilder)
	}
	return out
}

// TestTheCompilersOwnCodeInTheWindowIsTheCodeItsNoticesAnswerFor holds the
// pin to the notices and the build to the pin.
func TestTheCompilersOwnCodeInTheWindowIsTheCodeItsNoticesAnswerFor(t *testing.T) {
	root := rootOrFail(t)
	pin, problems := parseToolchainPin(toolchainPin, readRepoFile(t, root, toolchainPin))
	requireNone(t, problems, "the pin of the C compiler's own code in the window")

	sections, parse := noticeSections(windowNotices, readRepoFile(t, root, windowNotices))
	requireNone(t, parse, "the notices file")
	requireNone(t, checkToolchainNotices(pin, sections), "the notices of the C compiler's own code")

	requireNone(t, checkToolchainWiring(readRepoFile(t, root, windowBuilder), readRepoFile(t, root, toolchainReader), pin, githubFiles(t, root)),
		"the build that reads the linker's map")
}

// TestTheToolchainGuardCanFail is the canary, one planted gap per question.
func TestTheToolchainGuardCanFail(t *testing.T) {
	good := "# a comment\nversion windows/amd64 mingw-w64 1.0.0\nlink windows/amd64 crt2.o mingw-w64\nlink windows/amd64 libgcc.a gcc\n" +
		"version linux/amd64 glibc 2.39\nlink linux/amd64 crt1.o glibc\nnothing darwin/arm64\n"
	pin, problems := parseToolchainPin("fixture", good)
	requireNone(t, problems, "a well formed pin")

	_, problems = parseToolchainPin("fixture", good+"link windows/amd64 crt2.o mingw-w64\nlink linux/amd64 libssp.a ssp\n"+
		"version darwin/arm64 gcc 13\nlink freebsd/amd64 crt1.o glibc\nlink linux/amd64 crt1.o\ncolour linux/amd64\nnothing darwin/arm64\n")
	requireFinding(t, problems, "names crt2.o for windows/amd64 twice")
	requireFinding(t, problems, "a library nobody has read the licence of")
	requireFinding(t, problems, "names a version of gcc, and no section of the notices quotes its texts")
	requireFinding(t, problems, "names freebsd/amd64, which is not a target")
	requireFinding(t, problems, "has 3 fields where a link line has 4")
	requireFinding(t, problems, "has a colour line")
	requireFinding(t, problems, "says twice that darwin/arm64 links nothing")
	_, problems = parseToolchainPin("fixture", "link windows/amd64 crt2.o mingw-w64\nnothing windows/amd64\nversion linux/amd64 glibc 2.39\n")
	requireFinding(t, problems, "says both that windows/amd64 links files of the compiler's and that it links nothing")
	requireFinding(t, problems, "links mingw-w64 into windows/amd64 and names no version of it")
	requireFinding(t, problems, "names a version of glibc for linux/amd64 and links nothing of it there")
	requireFinding(t, problems, "says nothing about darwin/arm64")
	_, problems = parseToolchainPin("fixture", "")
	requireFinding(t, problems, "says nothing about windows/amd64")

	sections := map[string]string{"MinGW-w64 runtime": "\nVersion `1.0.0`. ZPL-2.1.\n", "GNU C Library": "\nVersion `2.39`. LGPL-2.1-or-later.\n"}
	requireNone(t, checkToolchainNotices(pin, sections), "notices that agree with the pin")
	requireFinding(t, checkToolchainNotices(pin, map[string]string{"GNU C Library": sections["GNU C Library"]}), "no such section")
	requireFinding(t, checkToolchainNotices(pin, map[string]string{"MinGW-w64 runtime": "\nVersion `0.9.0`. ZPL-2.1.\n",
		"GNU C Library": sections["GNU C Library"]}), "states 0.9.0")
	noGlibc, _ := parseToolchainPin("fixture", "version windows/amd64 mingw-w64 1.0.0\nlink windows/amd64 crt2.o mingw-w64\n"+
		"nothing linux/amd64\nnothing darwin/arm64\n")
	requireFinding(t, checkToolchainNotices(noGlibc, sections), `has the section "## GNU C Library"`)

	builder := "bash " + toolchainReader + " \"$target\"\n-extldflags=-Wl,-Map=\n-extldflags=-Wl,-map,\n"
	reader := `pin="` + toolchainPin + `"`
	requireNone(t, checkToolchainWiring(builder, reader, pin, map[string]string{"x.yml": "go build"}), "a build that reads the pin")
	requireFinding(t, checkToolchainWiring("go build\n", reader, pin, nil), "does not carry \"bash .github/scripts/window_toolchain.sh \"")
	requireFinding(t, checkToolchainWiring(strings.Replace(builder, "-map,", "-Map,", 1), reader, pin, nil), `"-extldflags=-Wl,-map,"`)
	requireFinding(t, checkToolchainWiring(builder, `pin=".github/other"`, pin, nil), "holds the build to a list of its own")
	requireFinding(t, checkToolchainWiring(builder, reader, pin, map[string]string{"x.yml": "allowed: libgcc.a"}), "names libgcc.a, a second copy")
}
