package fynetheme

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"fyne.io/fyne/v2/widget"
)

// leftOut are the words the toolkit asks for that this window does not hand
// over, each with the reason it may stay English on every desktop.
var leftOut = map[string]string{
	"Advanced": "the colour picker, which this window never opens",
	"Confirm":  "the colour picker, which this window never opens",
}

// unreadable are the files of the toolkit that ask for a word by a key their
// source does not spell out, so no reading of the source can say which word.
var unreadable = map[string]string{
	"widget/calendar.go": "the names of the days, built from a date, for a calendar this window never opens",
}

// searched are the directories of the toolkit that draw what this window shows.
// The lang package itself is not among them, because it is where the words are
// looked up and not where they are asked for.
var searched = []string{"app", "container", "dialog", "internal", "widget"}

// askers are the functions of the lang package that take a word, by the names
// the package gives them.
var askers = map[string]bool{
	"L": true, "X": true, "N": true, "XN": true,
	"Localize": true, "LocalizeKey": true, "LocalizePlural": true, "LocalizePluralKey": true,
}

// 🔴 The list handed to the toolkit and the words the toolkit asks for are read
// from one source, and that source is the toolkit's own code.
//
// Until 2026-10-14 the list was made by reading the toolkit by hand, twice. The
// first reading missed the words of an empty chooser, so a desktop set to
// Polish logged a failure for every chooser built (B-322). A mutation removing
// an entry from the list then passed every test, because nothing compared the
// list with anything. This reads the source this test binary was compiled
// from, so a new release of the toolkit that asks for a new word fails here on
// the day it is taken in, rather than in somebody's terminal.
//
// It holds both directions. A word the toolkit asks for is handed over or left
// out with a reason, and a word handed over or left out is one the toolkit still
// asks for, or the list carries a translation nobody will ever see.
func TestTheWordsHandedToTheToolkitAreTheWordsItAsksFor(t *testing.T) {
	asked, dynamic := wordsTheToolkitAsksFor(t, toolkitSource(t))

	for _, word := range sortedKeys(asked) {
		_, handed := toolkitWords[word]
		_, excused := leftOut[word]
		if !handed && !excused {
			t.Errorf("the toolkit asks for %q at %s, and this window neither hands it over nor "+
				"says why not", word, strings.Join(asked[word], ", "))
		}
	}
	for word := range toolkitWords {
		if _, ok := asked[word]; !ok {
			t.Errorf("%q is handed to the toolkit, which no longer asks for it", word)
		}
	}
	for word := range leftOut {
		if _, ok := asked[word]; !ok {
			t.Errorf("%q is left out on purpose, and the toolkit no longer asks for it", word)
		}
	}
	for _, file := range sortedKeys(dynamic) {
		if _, ok := unreadable[file]; !ok {
			t.Errorf("%s asks for a word by a key its source does not spell out, at %s, so "+
				"this test cannot say which word", file, strings.Join(dynamic[file], ", "))
		}
	}
	for file := range unreadable {
		if _, ok := dynamic[file]; !ok {
			t.Errorf("%s is excused for asking by a key nobody can read, and it no longer does", file)
		}
	}
}

// toolkitSource is the directory of the toolkit's source, taken from where this
// binary says one of its functions was compiled from.
//
// Asking the binary rather than go.mod or the module cache means the test reads
// exactly the code it runs, whatever replaced or redirected it.
func toolkitSource(t *testing.T) string {
	t.Helper()
	entry := reflect.ValueOf(widget.NewSelect).Pointer()
	file, _ := runtime.FuncForPC(entry).FileLine(entry)
	root := filepath.Dir(filepath.Dir(file))
	if info, err := os.Stat(filepath.Join(root, "widget")); err != nil || !info.IsDir() {
		t.Fatalf("the toolkit was compiled from %s and its source is not there, which is what "+
			"a build with -trimpath leaves. Run this test without it", file)
	}
	return root
}

// wordsTheToolkitAsksFor reads every call of the lang package in the searched
// directories, test files aside. A word written out goes to asked, with where it
// was asked for. A call whose word is not written out goes to dynamic, by file.
func wordsTheToolkitAsksFor(t *testing.T, root string) (asked, dynamic map[string][]string) {
	t.Helper()
	asked, dynamic = map[string][]string{}, map[string][]string{}
	files := token.NewFileSet()
	for _, dir := range searched {
		err := filepath.WalkDir(filepath.Join(root, dir), func(name string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(name, ".go") ||
				strings.HasSuffix(name, "_test.go") {
				return err
			}
			parsed, err := parser.ParseFile(files, name, nil, parser.SkipObjectResolution)
			if err != nil {
				return err
			}
			relative, _ := filepath.Rel(root, name)
			collect(files, parsed, filepath.ToSlash(relative), asked, dynamic)
			return nil
		})
		if err != nil {
			t.Fatalf("cannot read the toolkit under %s: %v", dir, err)
		}
	}
	if len(asked) == 0 {
		t.Fatalf("no word asked for anywhere under %s, so this read the wrong place", root)
	}
	return asked, dynamic
}

// collect adds the calls of one file of the toolkit to the two maps.
func collect(files *token.FileSet, parsed *ast.File, relative string, asked, dynamic map[string][]string) {
	local := langName(parsed)
	if local == "" {
		return
	}
	ast.Inspect(parsed, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 {
			return true
		}
		chosen, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !askers[chosen.Sel.Name] {
			return true
		}
		if pkg, ok := chosen.X.(*ast.Ident); !ok || pkg.Name != local {
			return true
		}
		where := relative + ":" + strconv.Itoa(files.Position(call.Pos()).Line)
		if literal, ok := call.Args[0].(*ast.BasicLit); ok && literal.Kind == token.STRING {
			if word, err := strconv.Unquote(literal.Value); err == nil {
				asked[word] = append(asked[word], where)
				return true
			}
		}
		dynamic[relative] = append(dynamic[relative], where)
		return true
	})
}

// langName is the name a file of the toolkit gives the lang package, or empty
// when it does not import it.
func langName(parsed *ast.File) string {
	for _, spec := range parsed.Imports {
		if imported, err := strconv.Unquote(spec.Path.Value); err == nil && imported == "fyne.io/fyne/v2/lang" {
			if spec.Name != nil {
				return spec.Name.Name
			}
			return path.Base(imported)
		}
	}
	return ""
}

// sortedKeys gives the keys of a map in order, so that a failure lists its
// findings the same way on every run.
func sortedKeys(of map[string][]string) []string {
	keys := make([]string, 0, len(of))
	for key := range of {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
