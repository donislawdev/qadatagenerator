package guard

import (
	"fmt"
	"go/parser"
	"go/token"
	"sort"
	"strings"
	"unicode"
)

// CompilerDirectives are the compiler directives our own files may write, the
// //go: lines.
//
// # Why the rest are refused
//
// A directive tells the compiler or the linker something no import shows. The
// directive go:linkname binds a function declared here to one of another
// package, so a file can call into syscall without importing it. Measured on
// 2026-11-17 with Go 1.27.4: a program importing only os and unsafe linked
// syscall.syscall for darwin/arm64, a raw system call, while the linker
// refused syscall.socket for linux/amd64, which the standard library does not
// mark for linking from outside. The directive go:cgo_import_dynamic, which
// the compiler accepts outside cgo, names a function in a system library, and
// the directive go:wasmimport takes one from the host. A directive a later Go
// adds is refused before anybody has read it, which is why this is a list of
// what may stand and not of what may not.
//
// Only line comments are directives. A block comment holding go:linkname is
// not one, measured the same day: the function it named had no body.
var CompilerDirectives = map[string]bool{
	// Which system a file is built for.
	"go:build": true,
	// Files carried inside the binary as data: the catalogue, the word lists,
	// the translations, the fonts and the icons.
	"go:embed": true,
}

// DirectiveUse reports every compiler directive in our own files that allowed
// does not name. Tests are left out, as by every guard on the product.
func DirectiveUse(root string, allowed map[string]bool) ([]string, error) {
	var findings []string
	err := eachProductFile(root, ".", func(path, name string) error {
		files := token.NewFileSet()
		parsed, err := parser.ParseFile(files, path, nil, parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		for _, group := range parsed.Comments {
			for _, comment := range group.List {
				text, isDirective := strings.CutPrefix(comment.Text, "//go:")
				if !isDirective {
					continue
				}
				end := strings.IndexFunc(text, unicode.IsSpace)
				if end < 0 {
					end = len(text)
				}
				verb := "go:" + text[:end]
				if !allowed[verb] {
					findings = append(findings, fmt.Sprintf("%s:%d writes //%s, and only the directives in "+
						"guard.CompilerDirectives may stand in our files", name, files.Position(comment.Pos()).Line, verb))
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(findings)
	return findings, nil
}
