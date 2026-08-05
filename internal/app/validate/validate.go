// Package validate answers one question: can this recipe be used.
//
// It is a use case, so it takes what it needs as parameters and returns a
// result. It does not know that recipes are YAML, it does not open files, and
// it does not print. Those belong to the surface that called it, which is what
// lets the window ask the same question later and get the same answer.
package validate

import (
	"strconv"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// Decoded is what reading a recipe produced.
type Decoded struct {
	Recipe    recipe.Recipe
	Problems  []recipe.Problem
	Unchecked []recipe.Unchecked
}

// Decoder turns the bytes of a recipe into a model and a list of problems.
//
// It is a function rather than an interface because there is one method and
// nothing to keep between calls. The point of the port is the direction of the
// dependency, not the ceremony.
type Decoder func(data []byte) Decoded

// Generators is the registry of names a recipe may draw from.
//
// # Why this port exists, and why it did not until 2026-08-05
//
// Without it this use case answered `can be used` about a recipe that generate
// then refused, because the registry lives in an adapter and validate could not
// see it. That is the worst answer a validator can give: it is not a gap, it is
// the opposite of the truth, and the whole point of the verb is that sentence.
//
// Found by running the binary, not by reading the code. Every test was green,
// because both halves were right about their own half.
type Generators interface {
	// Known reports whether this build has a generator of that name.
	Known(generator string) bool
	// Names returns every generator this build has, sorted, for the refusal.
	Names() []string
}

// Source is a recipe as it arrived, with the name to put in messages.
type Source struct {
	Name string
	Data []byte
}

// Report is what a person needs to see, whether or not the recipe is usable.
type Report struct {
	Name string
	// Problems is why the recipe cannot be used. Empty means it can.
	Problems []recipe.Problem
	// Unchecked is what this build did not look at. It is filled in whether
	// or not there were problems, because a clean report that hides its own
	// blind spots invites trust it has not earned.
	Unchecked []recipe.Unchecked
	// Entities and Fields count what was understood, so that a person can
	// see the recipe was read rather than skimmed.
	Entities int
	Fields   int
}

// Usable reports whether the recipe can be used as it stands.
func (r Report) Usable() bool { return len(r.Problems) == 0 }

// Run reads a recipe and reports what it found.
//
// The failure it returns carries the kind, so the surface that called it does
// not have to work out which ending this was. A recipe that cannot be used is
// a configuration failure and never a tool failure, which matters because the
// two mean different days of work for whoever reads the exit code.
func Run(decode Decoder, source Source, generators Generators) (Report, error) {
	decoded := decode(source.Data)

	report := Report{
		Name:      source.Name,
		Problems:  decoded.Problems,
		Unchecked: decoded.Unchecked,
		Entities:  len(decoded.Recipe.Entities),
	}
	for _, entity := range decoded.Recipe.Entities {
		report.Fields += len(entity.Fields)
	}
	report.Problems = append(report.Problems, unknownGenerators(decoded.Recipe, generators)...)

	if !report.Usable() {
		return report, fault.New(fault.Config, plural(len(report.Problems), "problem")+" in "+source.Name)
	}
	return report, nil
}

// unknownGenerators reports every field asking for a name this build lacks.
//
// The answer is a problem rather than a note, because generate refuses such a
// recipe outright. A note would leave the two verbs disagreeing about the same
// file, which is the thing this check exists to end.
func unknownGenerators(rec recipe.Recipe, generators Generators) []recipe.Problem {
	if generators == nil {
		return nil
	}
	var found []recipe.Problem
	for _, entity := range rec.Entities {
		for _, field := range entity.Fields {
			if field.Generator == "" || generators.Known(field.Generator) {
				continue
			}
			found = append(found, recipe.Problem{
				Where:   "entities." + entity.Name + ".fields." + field.Name,
				What:    "this build has no generator called " + strconv.Quote(field.Generator),
				Allowed: strings.Join(generators.Names(), ", "),
				Hint:    "pick one of the names above, or drop the generator and give the field a type",
			})
		}
	}
	return found
}

func plural(count int, word string) string {
	if count == 1 {
		return "1 " + word
	}
	return strconv.Itoa(count) + " " + word + "s"
}
