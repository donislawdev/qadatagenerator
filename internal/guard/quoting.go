package guard

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// quotingVerb is the formatting verb that puts a value into a sentence with its
// quotes around it, and it is the one a scheme may not use.
const quotingVerb = "%q"

// RawQuotedValues reports every message in the numbering schemes that puts a
// value into a sentence with the quoting verb.
//
// # Why a whole guard for one verb
//
// The values a scheme is handed come from a recipe or from somebody else's
// file, so they are untrusted and of unbounded length, and a message built with
// the quoting verb is as large as the value it was given. Measured on
// 2026-08-02 on the other road this class takes: two hundred kilobytes in, two
// hundred kilobytes out, into somebody's build log and, with the machine
// readable flag, onto the channel a pipeline parses.
//
// Thirteen places across four schemes did this, and the reason there is a guard
// rather than thirteen repairs is that a fourteenth scheme would have written a
// fourteenth. The note left with the first repair of this class asked how many
// roads lead to the channel, and the answer keeps growing because nothing was
// watching. What a scheme uses instead is provider.Show, which shortens the
// value and then quotes it.
//
// # What this deliberately does not do
//
// It reads the syntax tree rather than the text, so the verb inside a comment
// is not a finding. A guard that fires on prose gets switched off, and one
// switched off is worse than absent because nobody notices the day it stopped
// looking.
//
// It says nothing about the other formatting verbs. A value written with the
// plain string verb is just as unbounded, and catching that would need to know
// which argument is a value and which is a count. This catches the shape that
// actually happened thirteen times, and the narrowness is recorded rather than
// hidden: the sentence a scheme writes is still its own responsibility.
func RawQuotedValues(root string) ([]string, error) {
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
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}

		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		found, err := quotedValuesIn(path, filepath.ToSlash(relative))
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

// quotedValuesIn returns the string literals of one file that carry the quoting
// verb.
func quotedValuesIn(path, name string) ([]string, error) {
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	var findings []string
	ast.Inspect(parsed, func(node ast.Node) bool {
		literal, isLiteral := node.(*ast.BasicLit)
		if !isLiteral || literal.Kind != token.STRING {
			return true
		}
		// An unquotable literal is a raw string, which carries its text as
		// written, so the fallback keeps looking at something real.
		text, unquoteErr := strconv.Unquote(literal.Value)
		if unquoteErr != nil {
			text = literal.Value
		}
		if !strings.Contains(text, quotingVerb) {
			return true
		}
		line := fileSet.Position(literal.Pos()).Line
		findings = append(findings, name+":"+strconv.Itoa(line)+
			" writes a value into a message with "+quotingVerb+
			", which is unbounded. Use provider.Show, which shortens and then quotes")
		return true
	})
	return findings, nil
}
