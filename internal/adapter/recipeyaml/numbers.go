package recipeyaml

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml/ast"
)

// plainDecimal is how a whole number may be written in a recipe.
//
// An optional sign, then either a single nought or a digit string that does not
// start with one. Everything else is refused rather than interpreted.
var plainDecimal = regexp.MustCompile(`^[+-]?(0|[1-9][0-9]*)$`)

// refuseCleverNumberSpellings stops a recipe whose numbers YAML would read as
// something other than what they look like.
//
// # The measurement this exists because of
//
// YAML types a bare scalar by its own rules, and those rules are not the ones a
// person reading a recipe applies. Measured on this project on 2026-08-12,
// through the built binary: count: 010 produced eight rows. Not a refusal, not
// a warning, exit code nought and a run that looks entirely successful. The
// author asked for ten, wrote the nought the way a person writes a padded
// number, and received eight.
//
// That is the failure this whole product exists to attack, appearing in our own
// input path: a quantity silently different from the one that was requested,
// with nothing in the output saying so. It was reported to us by the sibling
// project, which hit the same thing in their own recipes, and confirmed here by
// running our own binary rather than by trusting the report.
//
// # Why refusing beats interpreting
//
// The obvious alternative is to read 010 as ten, on the grounds that nobody
// means octal. That trades a silent wrong number for a different silent wrong
// number, because whoever does mean octal now gets ten. A recipe lives in
// somebody's repository and is read in pull requests, so the only honest answer
// is that a number which reads two ways is not written well enough, and to say
// which spelling is wanted.
//
// # What is refused, and each is a real YAML spelling
//
// A leading nought (octal), 0o and 0x and 0b prefixes, and underscores as digit
// separators. All of them parse today and all of them can produce a number the
// line does not appear to say.
//
// # What this does not cover
//
// A number written with a decimal point, which the reader of whole numbers
// already refuses where a whole number is required, and quoted strings that
// merely look numeric, which never reach a number at all.
func refuseCleverNumberSpellings(r *reader, body ast.Node) bool {
	before := len(r.problems)

	ast.Walk(&numberVisitor{reader: r}, body)

	return len(r.problems) > before
}

type numberVisitor struct {
	reader *reader
}

func (v *numberVisitor) Visit(node ast.Node) ast.Visitor {
	integer, ok := node.(*ast.IntegerNode)
	if !ok {
		return v
	}
	token := integer.GetToken()
	if token == nil {
		return v
	}
	written := strings.TrimSpace(token.Value)
	if plainDecimal.MatchString(written) {
		return v
	}

	// The value YAML arrived at is in the message beside the spelling, because
	// the pair is the whole argument. "010 is not plain decimal" is a rule to
	// look up. "010 is read as 8" is the reason, and an existing test for the
	// seed already insisted on showing the number nobody wrote.
	v.reader.report(where(node),
		"the number written as "+written+" is read as "+fmt.Sprintf("%v", integer.Value)+
			", because YAML types a bare scalar by its own rules rather than by how it looks",
		written,
		"write it in plain decimal digits, for example "+suggestion(written))
	return v
}

// where turns the path YAML tracks into the one this reader reports with.
//
// The parser writes paths as $.entities.customer.count, and every other message
// from this package names the same place without the leading marker.
func where(node ast.Node) string {
	path := node.GetPath()
	path = strings.TrimPrefix(path, "$")
	return strings.TrimPrefix(path, ".")
}

// suggestion offers the plain form of what was written, so the message ends
// with something to type rather than with a rule to interpret.
func suggestion(written string) string {
	trimmed := strings.TrimLeft(strings.ReplaceAll(written, "_", ""), "+")
	switch {
	case strings.HasPrefix(strings.ToLower(trimmed), "0x"),
		strings.HasPrefix(strings.ToLower(trimmed), "0o"),
		strings.HasPrefix(strings.ToLower(trimmed), "0b"):
		return "the same value in ordinary digits"
	}
	if plain := strings.TrimLeft(trimmed, "0"); plain != "" {
		return plain
	}
	return "0"
}
