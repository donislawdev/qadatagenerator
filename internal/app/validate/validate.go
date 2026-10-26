// Package validate answers one question: can this recipe be used.
//
// It is a use case, so it takes what it needs as parameters and returns a
// result. It does not know that recipes are YAML, it does not open files, and
// it does not print. Those belong to the surface that called it, which is what
// lets the window ask the same question later and get the same answer.
package validate

import (
	"fmt"
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
	// Lines says which line each place of the recipe was written on, so that
	// the problems raised here can be pointed at as precisely as the ones the
	// decoder raised.
	//
	// 🔴 The reason it has to travel is that the spelling of a place is written
	// down in more than one file. The decoder names a place while it walks, and
	// the checks below build the same name from the model. Those two agreeing
	// is what makes a line correct, and a guard holds them together rather than
	// a hope.
	Lines map[string]int
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
	// CanProduce reports whether this build can fill a field with that
	// generator in a locale, and says why not when it cannot.
	//
	// The locale comes from the recipe being read rather than from the source
	// this verb is given, which carries none.
	CanProduce(generator, locale string) (bool, string)
}

// Catalog is the damage this build knows about, for checking a corruption
// policy.
//
// # Why this port exists
//
// A policy names layers, and whether a layer has damage behind it is a fact
// about the catalog, which lives in an adapter. Without this port validate would
// approve a policy that corrupt then does nothing about, which is the same
// defect the generator registry caused: not a gap in a check but the opposite of
// the truth, from the one verb whose entire purpose is that sentence. It landed
// with the policy key rather than after it, so that the two verbs have never
// disagreed about a policy.
type Catalog interface {
	// Layers returns every layer this build carries damage in, sorted.
	Layers() []string
	// Unimplemented is how much damage in a layer is described and cannot yet
	// be done.
	Unimplemented(layer string) int
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
func Run(decode Decoder, source Source, generators Generators, described Catalog) (Report, error) {
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
	report.Problems = append(report.Problems, collidingParentColumns(decoded.Recipe)...)
	report.Problems = append(report.Problems, unusableLayers(decoded.Recipe, described)...)
	// The problems raised above name their places the way the decoder does, so
	// they can be given the same lines. Done here, once, rather than at each
	// place that builds a problem, because the join is one rule.
	placeProblems(report.Problems, decoded.Lines)
	report.Unchecked = append(report.Unchecked, whatThePolicyWillNotDo(decoded.Recipe, described)...)

	if !report.Usable() {
		return report, fault.New(fault.Config, plural(len(report.Problems), "problem")+" in "+source.Name)
	}
	return report, nil
}

// placeProblems gives every problem that has no line the one its place was
// written on.
//
// A problem the decoder already placed keeps what it was given. A place nobody
// can resolve to exactly one line keeps zero, which is what whoever draws it
// reads as no line at all.
func placeProblems(problems []recipe.Problem, lines map[string]int) {
	for at := range problems {
		if problems[at].Line != 0 {
			continue
		}
		problems[at].Line = lines[problems[at].Where]
	}
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
			if field.Generator == "" {
				continue
			}
			where := "entities." + entity.Name + ".fields." + field.Name
			if !generators.Known(field.Generator) {
				found = append(found, recipe.Problem{
					Where:   where,
					What:    "this build has no generator called " + strconv.Quote(field.Generator),
					Allowed: strings.Join(generators.Names(), ", "),
					Hint: "pick one of the names above, or drop the generator and give the " +
						"field a type",
				})
				continue
			}
			// 🔴 The name is carried and the locale may still not be. generate
			// refuses such a recipe on the first row, so calling it usable here
			// is the two verbs disagreeing about one file, which is what this
			// check exists to end.
			if fine, why := generators.CanProduce(field.Generator, rec.Locale); !fine {
				found = append(found, recipe.Problem{
					Where: where, What: why,
					Hint: "generate this recipe for a locale this build carries, or drop the field",
				})
			}
		}
	}
	return found
}

// collidingParentColumns reports every child carrying a field named after the
// entity it belongs to.
//
// # Why this is a problem and not a note
//
// The relation puts a column of that name beside the fields, so the two would
// share one column and the writer refuses rather than guessing which of them
// the reader wanted. generate therefore ends on it, and until 2026-09-02 this
// verb answered that the same recipe can be used. That is not a gap in a check
// but the opposite of the truth, from the one verb whose whole purpose is this
// sentence, and it is the same shape as the unknown generator above: found by
// running the binary, with every test green.
//
// 🔴 The refusal generate ends on made it worse rather than softer. It says
// `try: qadg validate <recipe> reports every problem the recipe has at once`,
// so it sent whoever hit it to the command that had just approved the file.
func collidingParentColumns(rec recipe.Recipe) []recipe.Problem {
	var found []recipe.Problem
	for _, entity := range rec.Entities {
		name, collides := entity.ParentColumnCollision()
		if !collides {
			continue
		}
		found = append(found, recipe.Problem{
			Where: "entities." + entity.Name + ".fields." + name,
			What: "this entity belongs to " + strconv.Quote(name) + " and also has a field of " +
				"that name, so the relation and the field would need the same column",
			Hint: "rename the field. The column carrying the relation takes the name of the " +
				"entity it points at",
		})
	}
	return found
}

// unusableLayers reports every layer a policy names that this build has no
// damage in.
//
// A problem rather than a note, because damage nobody described can never
// happen, so a policy naming only such layers orders a run that produces
// nothing while looking as though it produced everything.
func unusableLayers(rec recipe.Recipe, described Catalog) []recipe.Problem {
	if described == nil || !rec.Policy.Asks() {
		return nil
	}
	carries := map[string]bool{}
	for _, layer := range described.Layers() {
		carries[layer] = true
	}
	var found []recipe.Problem
	for _, named := range rec.Policy.Layers {
		if carries[named] {
			continue
		}
		found = append(found, recipe.Problem{
			Where:   "policy.layers",
			What:    "this build describes no damage in " + strconv.Quote(named),
			Allowed: strings.Join(described.Layers(), ", "),
			Hint:    "name a layer from the list above, or remove it from the policy",
		})
	}
	return found
}

// whatThePolicyWillNotDo names the damage this build describes and cannot yet
// carry out, and the condition it will not evaluate.
//
// Both are reported as unchecked rather than refused. A layer waiting for its
// implementations is the ordinary state of a catalog that grows faster than the
// engine, and a condition nobody evaluates does not stop a run. Silence about
// either would leave somebody believing their policy did more than it did.
func whatThePolicyWillNotDo(rec recipe.Recipe, described Catalog) []recipe.Unchecked {
	if !rec.Policy.Asks() {
		return nil
	}
	var notes []recipe.Unchecked
	if described != nil {
		for _, named := range rec.Policy.Layers {
			if waiting := described.Unimplemented(named); waiting > 0 {
				notes = append(notes, recipe.Unchecked{
					Where:  "policy.layers",
					What:   fmt.Sprintf("%s in %s", kindsOfDamage(waiting), named),
					Reason: "described in the catalog and not implemented in this build yet, so the manifest will record every one as not applied",
				})
			}
		}
	}
	// A condition on which rows to damage is not noted here since 2026-10-26:
	// the reader of recipes refuses it while expressions are not evaluated
	// (DEC-266), so a recipe carrying one never reaches this.
	return notes
}

// kindsOfDamage counts damage in words.
//
// It is written out rather than handed to the general pluraliser beside it,
// which appends to the end of what it is given and would say "kind of damages".
// The message is read by somebody deciding whether their policy will do what
// they asked, and prose that stumbles there costs more than the four lines.
func kindsOfDamage(count int) string {
	if count == 1 {
		return "one kind of damage"
	}
	return fmt.Sprintf("%d kinds of damage", count)
}

func plural(count int, word string) string {
	if count == 1 {
		return "1 " + word
	}
	return strconv.Itoa(count) + " " + word + "s"
}
