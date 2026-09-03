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

// adviceVerbs are the words that begin a next move, as they were actually
// found in the messages of this project.
//
// # Why a list of words rather than a rule about grammar
//
// A next move is an imperative, and nothing tells an imperative from a noun of
// the same spelling without understanding the sentence. What can be done is to
// carry the vocabulary this project has used for advice and refuse it inside a
// message. The list is closed and named, so a message using a word that is not
// here passes, and that limit is stated below rather than hidden.
//
// Every word here was taken out of a message on 2026-09-03. None of them was
// hypothetical.
var adviceVerbs = []string{
	"Add", "Change", "Check", "Choose", "Compare", "Damage", "Delete", "Generate",
	"Give", "Leave", "Pass", "Remove", "Rename", "Report", "Run", "Split",
	"Try", "Use", "Wait", "Write",
}

// AdviceInsideMessages reports every refusal whose message carries its own next
// move instead of handing it over.
//
// # What goes wrong without this
//
// A refusal reaches a person in four parts, and the fourth is what to do about
// it. A message carrying advice of its own gets the sentence for its kind added
// underneath, so the reader is given two next moves, and the second one is
// chosen by the kind rather than by the failure. Measured on 2026-09-03 across
// seven places: one told the reader to split the run and then, on the next
// line, to change the flags describing the file. Another named a flag, which is
// worse than useless on the surface that has no flags.
//
// The family was closed once, on the two places somebody had looked at. It was
// not counted. Five more were standing in the tree the next day, which is why
// this is a guard and not a sixth repair.
//
// # What this deliberately does not do
//
// It reads the syntax tree rather than the text, so a word in a comment is
// never a finding. A guard that fires on prose gets switched off, and one
// switched off is worse than absent.
//
// 🔴 It sees the words it knows. A message advising with a verb outside
// adviceVerbs passes, and so does one advising without an imperative at all.
// What it does catch is the shape that happened seven times, and the narrowness
// is written down rather than left to be discovered.
//
// It says nothing about whether a next move handed over properly is a good one.
// Whether "rename the entity" helps is a question for a person.
func AdviceInsideMessages(root string) ([]string, error) {
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
		found, err := adviceIn(path, filepath.ToSlash(relative))
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

// adviceIn returns the refusals of one file whose message advises.
func adviceIn(path, name string) ([]string, error) {
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	var findings []string
	ast.Inspect(parsed, func(node ast.Node) bool {
		call, isCall := node.(*ast.CallExpr)
		if !isCall || !refusalCall(call) {
			return true
		}
		said := messageOf(call)
		verb := adviceVerbIn(said)
		if verb == "" {
			return true
		}
		findings = append(findings, name+":"+
			strconv.Itoa(fileSet.Position(call.Pos()).Line)+" the message advises with "+
			strconv.Quote(verb)+", so the reader gets this and the sentence for its kind. "+
			"Hand the move over with DoInstead instead")
		return true
	})
	return findings, nil
}

// refusalCall says whether a call builds a refusal.
func refusalCall(call *ast.CallExpr) bool {
	selector, isSelector := call.Fun.(*ast.SelectorExpr)
	if !isSelector {
		return false
	}
	pkg, isName := selector.X.(*ast.Ident)
	if !isName || pkg.Name != "fault" {
		return false
	}
	return selector.Sel.Name == "New" || selector.Sel.Name == "Wrap"
}

// messageOf joins the string literals of a refusal into the sentence a person
// reads, so that advice split across concatenated lines is still seen.
func messageOf(call *ast.CallExpr) string {
	var said strings.Builder
	for _, argument := range call.Args {
		ast.Inspect(argument, func(node ast.Node) bool {
			literal, isLiteral := node.(*ast.BasicLit)
			if !isLiteral || literal.Kind != token.STRING {
				return true
			}
			text, unquoteErr := strconv.Unquote(literal.Value)
			if unquoteErr != nil {
				text = literal.Value
			}
			said.WriteString(text)
			return true
		})
	}
	return said.String()
}

// adviceVerbIn returns the advice verb a message begins a sentence with, or an
// empty string.
//
// A sentence boundary is what makes it advice rather than a word inside a
// statement: "changed while it was being read" is a description and "Change the
// file" is a move. The first word of a message is not looked at either, because
// a message starting with a capital is a sentence about the failure and nothing
// precedes it to make it an instruction.
func adviceVerbIn(said string) string {
	for _, verb := range adviceVerbs {
		for _, boundary := range []string{". ", "? ", ": "} {
			if strings.Contains(said, boundary+verb+" ") {
				return verb
			}
		}
	}
	return ""
}
