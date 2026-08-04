package guard

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ConcurrencyAllowed lists the files permitted to start or coordinate
// concurrent work.
//
// It stood empty until the first command that writes files existed, which is
// what this guard was installed to make visible. The day it happened, putting
// the file here was a deliberate act rather than a line that slipped in, and
// the reason is written below rather than left to whoever finds the entry.
//
// # Why the one entry is here
//
// A signal is delivered to a program on a goroutine the runtime owns, so
// noticing one cannot be done without concurrency. The file listed keeps every
// concurrent thing about that in one place: it turns a signal into an answer to
// a question, and the run itself decides where to stop and removes its own
// temporary files on its own goroutine. Nothing else in the product knows a
// signal exists.
//
// One entry is a number worth keeping. A second one deserves the same
// paragraph, and if the paragraph is hard to write the concurrency is probably
// in the wrong place.
//
// # Why a guard rather than a rule in a document
//
// The architecture already says the result is assembled by row number and
// never by whichever goroutine finished first, and that nothing mutable is
// shared. Those sentences have no enforcement, and a rule with no enforcement
// is a sentence that outlives the decade unchanged.
//
// The race detector catches a race that a test actually triggers. This catches
// the code that could have one before any test has to be lucky. The two answer
// different halves of the same question and neither replaces the other.
//
// # What it does not cover
//
// Adding a file here says a person thought about it. It does not say they were
// right. The rules the architecture states, that the output is ordered by row
// and that nothing mutable is shared, are still on whoever writes the code.
var ConcurrencyAllowed = map[string]bool{
	"internal/cli/signals.go": true,
}

// concurrencyPackages are imports whose whole purpose is coordinating work
// between goroutines.
//
// context is deliberately absent. Cancellation travels through it, but a
// context on its own starts nothing and shares nothing, and refusing it would
// push somebody towards a worse way of passing deadlines around.
//
// os/signal is here for a reason found while writing the first entry in the
// list above. It hands a program values from a goroutine of its own, and one of
// its functions builds the whole arrangement without the caller writing go,
// chan or select anywhere. Concurrency would then enter through a file that
// looks entirely ordinary, which is the one thing this guard exists to stop.
var concurrencyPackages = map[string]bool{
	"sync":        true,
	"sync/atomic": true,
	"os/signal":   true,
}

// Concurrency reports every place that starts or coordinates concurrent work
// outside the files allowed to.
//
// It reads the syntax tree rather than the text, so the word chan in a comment
// and the phrase go func inside a message are not findings. A guard that fires
// on prose gets switched off, and one switched off is worse than absent
// because nobody notices the day it stops looking.
func Concurrency(root string, allowed map[string]bool) ([]string, error) {
	var findings []string

	walkErr := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if path != root && (skippedDirs[name] || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}

		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(relative)
		// A test may start a goroutine, because proving that something is
		// safe under concurrency needs concurrency to prove it with.
		if allowed[name] || strings.HasSuffix(name, "_test.go") {
			return nil
		}

		found, err := concurrencyIn(path, name)
		if err != nil {
			return err
		}
		findings = append(findings, found...)
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	sort.Strings(findings)
	return findings, nil
}

func concurrencyIn(path, name string) ([]string, error) {
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	var findings []string
	at := func(node ast.Node, what string) {
		findings = append(findings, fmt.Sprintf("%s:%d %s", name, fileSet.Position(node.Pos()).Line, what))
	}

	for _, spec := range parsed.Imports {
		unquoted, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, err
		}
		if concurrencyPackages[unquoted] {
			at(spec, "imports "+unquoted)
		}
	}

	ast.Inspect(parsed, func(node ast.Node) bool {
		switch node.(type) {
		case *ast.GoStmt:
			at(node, "starts a goroutine")
		case *ast.ChanType:
			at(node, "declares a channel")
		case *ast.SelectStmt:
			at(node, "waits on several channels")
		}
		return true
	})
	return findings, nil
}
