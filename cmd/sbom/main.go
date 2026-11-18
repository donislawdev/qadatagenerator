// Command sbom writes the bill of materials of a release of QA Data
// Generator, as an SPDX 2.3 document on standard output.
//
// It is a tool of the release and never ships. A release builds ./cmd/qadg and
// ./cmd/qadg-gui by name, and a main package cannot be imported, so nothing in
// here reaches an archive that a person downloads (DEC-328).
//
// WHAT IT READS. Standard input is what "go version -m" prints for the built
// binaries, so the document describes the binaries rather than the tree they
// came from: the toolchain, the target, the commit and every module with the
// version the compiler recorded in each one. What a build cannot know, the
// licence of each part and its copyright lines, comes from the notices of its
// program: THIRD-PARTY-NOTICES.md for the command line and
// THIRD-PARTY-NOTICES-GUI.md for the window, both compiled into this command
// from internal/license of the same tree. A build that links a module or a
// version the notices of its program do not name is refused rather than
// described. A bill of materials that leaves out what it cannot see still
// claims to be complete.
//
// THE SOFTWARE RENDERER. The archive of the window for Windows carries Mesa in
// a folder called opengl, beside the binary and not in it, and its notices live
// in .github/opengl rather than in the window's (DEC-323). They are read from
// the folder -renderer names. A document of a build of the window for Windows
// without them is refused, and so is the folder given with no build it travels
// beside, because either would describe an archive nobody built.
//
// IT WRITES NO FILE. The document leaves on standard output and the summary on
// standard error.
//
// Usage:
//
//	go version -m dist/* | go run ./cmd/sbom -renderer .github/opengl > qadg.spdx.json
//
// Exit codes: 0 when the document was written, 1 when the builds and the
// notices disagree or the notices cannot describe a build, 2 when the command
// line or standard input is not what this command reads.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/donislawdev/qadatagenerator/internal/license"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

// inputCeiling is how much of standard input is read. A listing of the five
// builds of the command line and the three of the window is under twenty
// kilobytes, and anything near a megabyte is something other than a listing.
const inputCeiling = 1 << 20

// A tree is what this command knows about the source the builds came from: the
// notices of each program and the version, all compiled in.
type tree struct {
	notices       string // THIRD-PARTY-NOTICES.md, the command line's
	windowNotices string // THIRD-PARTY-NOTICES-GUI.md, the window's
	version       string
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr,
		tree{notices: license.CommandLineNotices, windowNotices: license.WindowNotices, version: version.Tool}))
}

const usage = "usage: go version -m <binary>... | sbom [-renderer <folder of the software renderer's notices>] > qadg.spdx.json"

// run is the command without the process around it, so that a test can call it
// with notices of its own.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer, from tree) int {
	// say writes one line to standard error. A line that cannot be written
	// there has nowhere else to go, and the exit code still says what
	// happened.
	say := func(format string, a ...any) {
		_, _ = fmt.Fprintf(stderr, format+"\n", a...)
	}
	flags := flag.NewFlagSet("sbom", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	rendererDir := flags.String("renderer", "", "")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		say("%s", usage)
		return 2
	}
	input, err := io.ReadAll(io.LimitReader(stdin, inputCeiling+1))
	if err != nil {
		say("sbom: reading standard input: %v", err)
		return 2
	}
	if len(input) > inputCeiling {
		say("sbom: standard input is over %d bytes, which no listing of builds is", inputCeiling)
		return 2
	}
	builds, err := parseBuilds(string(input))
	if err != nil {
		say("sbom: %v", err)
		return 2
	}
	notes, err := parseTree(from)
	if err != nil {
		say("sbom: %v", err)
		return 1
	}
	if *rendererDir != "" {
		if err := addRenderer(notes, os.DirFS(*rendererDir), *rendererDir); err != nil {
			say("sbom: %v", err)
			return 1
		}
	}
	if problems := reconcile(builds, notes); len(problems) > 0 {
		say("sbom: the builds and their notices disagree, so no document was written:")
		for _, p := range problems {
			say("  %s", p)
		}
		return 1
	}
	document, err := render(from.version, builds, notes)
	if err != nil {
		say("sbom: %v", err)
		return 1
	}
	if _, err := stdout.Write(document); err != nil {
		say("sbom: writing the document: %v", err)
		return 1
	}
	say("sbom: %s", summary(from.version, builds, notes))
	return 0
}
