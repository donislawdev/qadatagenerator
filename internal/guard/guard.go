// Package guard holds the checks that keep the architecture honest.
//
// It is imported by tests only. The rules it protects are the ones whose
// breach costs a rewrite rather than a fix: the dependency rule between the
// layers, the ban on network traffic, and the one import edge that would stop
// the command line binary from cross compiling.
//
// The checks take a package graph as their input instead of reading the disk
// themselves. That is what lets a test feed them a graph that breaks the rule
// and prove that they notice. A guard that has never failed is a guess.
package guard

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Package is one directory of Go source, described the way the rules need it.
type Package struct {
	// Path is the directory relative to the module root, with forward
	// slashes, for example internal/cli.
	Path string
	// Imports are the imports of the ordinary source files.
	Imports []string
	// TestImports are the imports of the test files, kept apart because a
	// test is allowed to reach places the product code is not.
	TestImports []string
}

// skipped directories hold no product source. Three of them are outside the
// repository altogether and one holds bytes that are deliberately not valid
// anything.
var skippedDirs = map[string]bool{
	".git":     true,
	".claude":  true,
	"docs":     true,
	"tools":    true,
	"spec":     true,
	"testdata": true,
	"vendor":   true,
}

// ModuleRoot returns the directory holding go.mod, starting from the working
// directory and walking upwards.
//
// Tests run with their own package directory as the working directory, so
// anything that reads a file of the repository needs this.
func ModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

// Scan reads the package graph of the module rooted at root.
func Scan(root string) ([]Package, error) {
	var found []Package
	walkErr := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		name := entry.Name()
		if path != root && (skippedDirs[name] || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")) {
			return filepath.SkipDir
		}
		pkg, hasSource, readErr := readDir(root, path)
		if readErr != nil {
			return readErr
		}
		if hasSource {
			found = append(found, pkg)
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return found, nil
}

func readDir(root, dir string) (Package, bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return Package{}, false, err
	}
	relative, err := filepath.Rel(root, dir)
	if err != nil {
		return Package{}, false, err
	}
	pkg := Package{Path: filepath.ToSlash(relative)}
	hasSource := false
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		hasSource = true
		imports, err := importsOf(filepath.Join(dir, entry.Name()))
		if err != nil {
			return Package{}, false, err
		}
		if strings.HasSuffix(entry.Name(), "_test.go") {
			pkg.TestImports = append(pkg.TestImports, imports...)
			continue
		}
		pkg.Imports = append(pkg.Imports, imports...)
	}
	return pkg, hasSource, nil
}

func importsOf(file string) ([]string, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, spec := range parsed.Imports {
		unquoted, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, err
		}
		paths = append(paths, unquoted)
	}
	return paths, nil
}
