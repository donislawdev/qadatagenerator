package guard

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// SyscallAllowed names the files that may write syscall, together with the
// identifiers each of them may take from it.
//
// # Why an exception exists at all
//
// The network guard refuses the syscall package everywhere, and the reason is
// written beside that refusal: it can do whatever the operating system can,
// sockets included, so importing it would route around every other name on that
// list. That is still true. What it costs is that a program cannot name the
// termination signal, because the standard library exposes it nowhere else.
//
// # Why the exception is a name and not a package
//
// A file level exception would let anything from syscall into that file, and a
// year from now nobody would remember that the file is an exception. The list
// below is therefore read as a pair: which file, and which identifiers. Writing
// syscall.Socket in the allowed file is a finding exactly like writing syscall
// anywhere else.
//
// # What it does not do
//
// It reads the tree as written. A name reached indirectly, through a variable
// holding a function value, is invisible here, as it is to the concurrency
// guard beside it. Both answer the question a reviewer would ask by reading,
// which is a smaller question than what the program can do.
var SyscallAllowed = map[string][]string{
	// The termination signal, so that a run stopped by a system rather than by
	// a person still removes what it was in the middle of writing.
	"internal/cli/signals.go": {"SIGTERM"},
}

// syscallPackage is the import being controlled.
const syscallPackage = "syscall"

// packageMayNameSyscall reports whether a package holds a file allowed to write
// syscall.
//
// It is derived from the list above rather than being a second list of its own.
// Two lists agreeing today is two lists disagreeing the first time somebody
// edits one of them, which is the failure this project has already had between
// a document and the code beside it.
func packageMayNameSyscall(pkg string) bool {
	for file := range SyscallAllowed {
		if path.Dir(file) == pkg {
			return true
		}
	}
	return false
}

// SyscallUse reports every place that writes syscall outside what the list
// allows, in both directions.
//
// The three findings it can make are different failures. A file nobody allowed
// is somebody reaching for the operating system. A name outside the allowance
// is the exception widening quietly inside a file that already had one. An
// entry naming a file that no longer writes syscall is the list rotting, which
// matters because a stale allowance is an open door nobody is looking at.
func SyscallUse(root string, allowed map[string][]string) ([]string, error) {
	var findings []string
	seen := map[string]bool{}

	walkErr := filepath.WalkDir(root, func(p string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if p != root && (skippedDirs[name] || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")) {
				return filepath.SkipDir
			}
			return nil
		}
		// A test may name syscall freely. The promise being kept is about the
		// product, and the guard that reads the whole build graph in CI answers
		// the other half.
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}

		relative, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(relative)
		found, imports, err := syscallIn(p, name, allowed[name])
		if err != nil {
			return err
		}
		if imports {
			seen[name] = true
		}
		findings = append(findings, found...)
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	for file := range allowed {
		if !seen[file] {
			findings = append(findings, file+" is allowed to name "+syscallPackage+
				" and does not, so the allowance is stale")
		}
	}
	sort.Strings(findings)
	return findings, nil
}

// syscallIn reports what one file does with syscall, and whether it imports it.
func syscallIn(path, name string, permitted []string) ([]string, bool, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, false, err
	}

	imported := false
	for _, spec := range parsed.Imports {
		unquoted, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, false, err
		}
		if unquoted == syscallPackage {
			imported = true
		}
	}
	if !imported {
		return nil, false, nil
	}
	if permitted == nil {
		return []string{name + " imports " + syscallPackage +
			", and only the files in guard.SyscallAllowed may"}, true, nil
	}
	return namesTakenFrom(parsed, name, permitted), true, nil
}

// namesTakenFrom reports every identifier a file takes from syscall that its
// allowance does not cover.
func namesTakenFrom(parsed *ast.File, name string, permitted []string) []string {
	allowed := make(map[string]bool, len(permitted))
	for _, one := range permitted {
		allowed[one] = true
	}

	var findings []string
	ast.Inspect(parsed, func(node ast.Node) bool {
		selector, isSelector := node.(*ast.SelectorExpr)
		if !isSelector {
			return true
		}
		pkg, isIdent := selector.X.(*ast.Ident)
		if !isIdent || pkg.Name != syscallPackage || allowed[selector.Sel.Name] {
			return true
		}
		findings = append(findings, fmt.Sprintf("%s uses %s.%s, and it may only use %s",
			name, syscallPackage, selector.Sel.Name, strings.Join(permitted, ", ")))
		return true
	})
	return findings
}
