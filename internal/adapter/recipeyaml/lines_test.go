package recipeyaml

import (
	"strings"
	"testing"

	"github.com/goccy/go-yaml/parser"
)

// Every shape of place a problem can name is reported at the line it was
// written on.
//
// 🔴 This is the guard for two spellings of one rule. The reader names a place
// as parent.key for a mapping, parent[index] for a sequence entry and a bare
// key at the top of the file, and the walk that finds lines has to spell them
// the same way. They are in one package and that does not make them one rule,
// so this closes them against each other on a recipe carrying all four shapes.
//
// A line that drifts is worse than no line, because a window drawing it points
// somebody at a place that is not the one their problem is about.
func TestEveryShapeOfPlaceIsReportedAtTheLineItWasWrittenOn(t *testing.T) {
	// Written with the line numbers in the comments, because a reader checking
	// this test should not have to count.
	source := strings.Join([]string{
		`version: "0.1"`, //  1
		`entities:`,      //  2
		`  customer:`,    //  3
		`    count: -5`,  //  4
		`    nosuchkey: 1`,
		`    fields:`,
		`      id:`,
		`        max_length: -3`,
		`    consistency:`,
		`      - pesel matches: [birth_date]`,
	}, "\n") + "\n"

	lines := map[string]int{}
	for _, problem := range Decode([]byte(source)).Problems {
		lines[problem.Where] = problem.Line
	}

	for _, wanted := range []struct {
		where string
		line  int
		shape string
	}{
		{"entities.customer.count", 4, "a key of an entity"},
		{"entities.customer.nosuchkey", 5, "a key of an entity nobody knows"},
		{"entities.customer.fields.id.max_length", 8, "a key of a field"},
		{"entities.customer.consistency[0]", 10, "an entry of a sequence"},
	} {
		line, reported := lines[wanted.where]
		if !reported {
			t.Errorf("nothing was reported at %s, which is %s, so this case checks nothing",
				wanted.where, wanted.shape)
			continue
		}
		if line != wanted.line {
			t.Errorf("%s was reported at line %d and it was written on line %d. A line that "+
				"is wrong sends somebody to a place their problem is not about",
				wanted.where, line, wanted.line)
		}
	}
}

// A place nobody wrote is reported without a line rather than with a wrong one.
//
// A recipe missing its version is the plainest case: the problem is about a key
// that is not in the file, so there is no line it could point at. Zero is the
// ordinary answer here and whoever draws it draws nothing.
func TestAPlaceNobodyWroteIsReportedWithoutALine(t *testing.T) {
	source := "entities:\n  customer:\n    count: 1\n    fields:\n      id: { type: uuid }\n"

	found := false
	for _, problem := range Decode([]byte(source)).Problems {
		if problem.Where != "version" {
			continue
		}
		found = true
		if problem.Line != 0 {
			t.Errorf("the missing version was reported at line %d, and it was never written, "+
				"so that line belongs to something else", problem.Line)
		}
	}
	if !found {
		t.Fatal("a recipe with no version raised nothing about the version, so this guard " +
			"is checking something other than what it says")
	}
}

// A path two places claim is left without a line rather than settled by
// whichever was walked last.
//
// 🔴 An entity, a field and a rule may be named anything YAML takes as a key,
// dots included, so entities.a.count is both the row count of an entity named a
// and an entity named a.count. That ambiguity belongs to the path and was there
// before lines were, which is why this leaves the place unresolved instead of
// pretending to pick. Reached through the walk rather than through a problem,
// because nothing reports a problem at a path this shape today and a guard that
// waited for one would be a guard that never ran.
func TestAPathTwoPlacesClaimIsLeftWithoutALine(t *testing.T) {
	source := strings.Join([]string{
		`version: "0.1"`,
		`entities:`,
		`  a:`,
		`    count: 1`,
		`  a.count:`,
		`    count: 1`,
	}, "\n") + "\n"

	file, err := parser.ParseBytes([]byte(source), 0)
	if err != nil {
		t.Fatalf("the sample does not parse as YAML: %v", err)
	}
	lines := linesIn(file)

	if line, resolved := lines["entities.a.count"]; resolved {
		t.Errorf("entities.a.count resolved to line %d, and two different places in this "+
			"recipe are written under that path, so one of them is being pointed at as "+
			"though it were the other", line)
	}
	// The half that keeps this from passing by finding nothing at all.
	if _, resolved := lines["entities.a"]; !resolved {
		t.Error("the walk found no line for entities.a, so it is not walking this recipe " +
			"and the check above is empty")
	}
}
