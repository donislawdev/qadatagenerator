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
// # Why the second entry is here
//
// The toolkit draws on one thread, and a button is pressed on it. A run that
// damaged somebody else's file on that thread would hold it for as long as the
// file takes: the window would stop repainting, stop answering the keyboard and
// stop offering the way to stop the run. The three things a screen owes a person
// during a run are exactly the three that thread has to be free for, so this one
// is not a choice about speed.
//
// The file listed keeps every concurrent thing about that in one place, the way
// the first one does for signals. It starts the goroutine, hands the answer back
// through the toolkit's own call for it, and turns a person pressing a button
// into an answer the run asks for between records. The screens themselves stay
// single threaded and nothing else in the window knows a goroutine exists.
//
// Two entries, one per surface, and each one is the same shape: work that cannot
// be done where it was asked for, isolated in a file with nothing else in it. A
// third deserves the same paragraph, and if the paragraph is hard to write the
// concurrency is probably in the wrong place.
//
// 🔴 That second entry moved on 2026-09-01, from internal/gui/takeview to a
// package of its own, and the move is the point rather than a tidying. Two more
// things in the window turned out to need the same treatment and neither of them
// is on the screen that runs: reading somebody's file, which held the drawing
// thread for the length of the file on every screen that opens one, and noticing
// that the system has asked this program to stop, which the window did not do at
// all. Filing each beside the screen that wants it would have grown this list by
// one entry per screen, and a list nobody can read in one sitting is a list that
// stops being read. The entry is still one file, still one surface, and the
// paragraph above is still the whole of what it allows.
//
// # Why the third entry is here, and why it is not that shape
//
// 🔴 It starts no work at all, and saying otherwise would be the first step
// towards a list that grows by resemblance. It shares something, which is the
// other half of what this guard watches, so it is written out rather than filed
// beside the two above.
//
// A word list becomes a value by drawing an index inside its length, so a run
// asks for the whole list once per cell and used to read and split the file
// every time. Measured on 2026-08-26: one hundred thousand rows of four columns
// took 11.7 seconds with three word lists and 0.28 seconds without any, and a
// single surname cost 3695 string allocations. The lists are compiled into the
// binary and never change while it runs, so reading one twice is work with no
// answer attached.
//
// What is shared is a map from a locale and a name to a list. It is mutable in
// that entries are added to it, and its contents are not: every writer for a
// key stores an equal list, because the bytes come from the same embedded file
// and the parse is a pure function of them. So the rule the architecture states
// holds in the sense that matters here, which is that no output can depend on
// which goroutine got there first. That is the claim, and it is a claim about
// this code rather than a property the type gives for free.
//
// 🔴 The lists handed out are shared, so no caller may write through one or
// append to one without copying first. The one caller that filters a list does
// it through a zero capacity reslice, which forces the append to allocate. A
// test stands in front of that, because the change that would break it is one
// character long and nothing else would notice.
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
	"internal/cli/signals.go":                    true,
	"internal/gui/background/background.go":      true,
	"internal/adapter/provider/corpus/corpus.go": true,
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
