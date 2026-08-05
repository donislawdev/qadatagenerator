// Package catalogyaml reads the corruption catalog this build ships with.
//
// It is the only place that knows the catalog is YAML at all, the same way the
// recipe reader is the only place that knows a recipe is. The model it produces
// carries no trace of the file it came from.
//
// # Why there is a copy of the file here
//
// The canonical catalog is spec/corruption-catalog.yaml, published under its own
// licence, edited by hand and reviewed in pull requests. The compiler cannot
// reach it from here: an embed pattern may not walk up out of its own package,
// and measured on 2026-08-05 a pattern of ../../spec/corruption-catalog.yaml is
// refused outright with invalid pattern syntax.
//
// Three ways around that were weighed and two were dropped. A package at the
// repository root could embed spec directly, but such a package is public API,
// and one module with everything under internal exists precisely so that GPL-3.0
// code is not published with an invitation to link against it. Generating Go
// source from the file would let the compiler check the data, but the generator
// would live outside the repository, so a contributor editing the catalog would
// have no way to regenerate anything, and growth by contribution is the reason
// the catalog is a data file in the first place.
//
// What is left is a copy, and the copy is only safe because a test compares it
// with the canonical file byte for byte and names the command that fixes it. The
// cost is stated plainly: two identical files of the same fifty four kilobytes,
// and one copy step whenever the catalog changes.
//
// # Nothing is read at startup
//
// Loading is an ordinary call rather than package initialisation, so a run that
// never corrupts anything never pays for parsing the catalog, and nothing here
// needs a lock or a goroutine to stay correct.
package catalogyaml

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"

	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// FileName is the catalog file, named once so that the loader, the test that
// compares the copy with the canonical file and any message about it cannot
// disagree.
const FileName = "corruption-catalog.yaml"

//go:embed corruption-catalog.yaml
var embedded []byte

// Embedded returns the bytes compiled into this binary.
//
// It exists for the test that holds the copy to the canonical file. Nothing in
// the product reads it, because everything in the product wants the model.
//
// The copy is so that a caller cannot write through the slice into the one
// thing every run reads.
func Embedded() []byte {
	bytes := make([]byte, len(embedded))
	copy(bytes, embedded)
	return bytes
}

// Load returns the catalog this build carries.
//
// The failure is a tool failure rather than a configuration one. A recipe or a
// policy that will not do is the user's file and their problem to fix, while a
// catalog that will not parse is a defect in this binary, and giving the two the
// same kind would send somebody to look at their own file for our mistake.
func Load() (catalog.Catalog, error) {
	return parse(embedded)
}

// parse is the whole of Load with the bytes handed in, so that a test can feed
// it a broken catalog and prove the refusals fire. A guard that has never been
// seen to fail is a guess.
func parse(data []byte) (catalog.Catalog, error) {
	var document rawCatalog
	if err := yaml.Unmarshal(data, &document); err != nil {
		return catalog.Catalog{}, fault.Wrap(fault.Runtime, err, refusal("it is not readable as YAML"))
	}

	entries := make([]catalog.Entry, 0, len(document.Mutators))
	var problems []string
	for index, raw := range document.Mutators {
		entry, entryProblems := raw.toEntry(index)
		problems = append(problems, entryProblems...)
		entries = append(entries, entry)
	}

	built, modelProblems := catalog.New(document.Version, document.Status, entries)
	problems = append(problems, modelProblems...)
	if len(problems) > 0 {
		return catalog.Catalog{}, fault.New(fault.Runtime, refusal(strings.Join(problems, "\n  ")))
	}
	return built, nil
}

// refusal builds the message, which carries the four parts every refusal in
// this tool carries: what happened, why, what a usable catalog looks like, and
// what to do about it.
func refusal(detail string) string {
	return "the corruption catalog built into this tool could not be read:\n  " + detail +
		"\n  A usable catalog carries a version and entries that each declare a layer, a summary," +
		" an expectation, a confidence, what they apply to and a before and after example." +
		"\n  This is a defect in this build rather than in anything you wrote. Report it with the" +
		" version printed by qadg --version."
}

// rawCatalog is the file as written, kept apart from the model so that the
// shape of the file can change without the model noticing.
type rawCatalog struct {
	Version  string       `yaml:"catalog_version"`
	Status   string       `yaml:"status"`
	Mutators []rawMutator `yaml:"mutators"`
	// Capabilities is read although the model does not carry it. The names
	// live in the code registry, which is the half that says who supplies
	// each one, and the two are held together by a separate check. Reading
	// it here would give the file a second opinion on that.
	Capabilities map[string]rawCapability `yaml:"capabilities"`
}

type rawCapability struct {
	SuppliedBy  string `yaml:"supplied_by"`
	Description string `yaml:"description"`
}

type rawMutator struct {
	ID        string       `yaml:"id"`
	Layer     string       `yaml:"layer"`
	Summary   string       `yaml:"summary"`
	AppliesTo rawAppliesTo `yaml:"applies_to"`
	// Parameters and Example are read as plain maps rather than structs, and
	// the reason is measured rather than stylistic.
	//
	// Both blocks have a field where an absent key and a written one mean
	// different things. A parameter default written as null is the value
	// L1.json.duplicate_key writes under a repeated key, and the after value
	// of L2.string.empty is an empty string because emptying the field is
	// the whole mutation. Read through a struct, both arrive as the zero
	// value and are indistinguishable from a key nobody wrote.
	//
	// The obvious way out does not work. A custom unmarshaler on the field
	// is never called for an explicit null: measured on 2026-08-05 with the
	// library this build uses, default: null and no default at all both
	// leave it untouched. Decoding the block as a map shows the key in both
	// cases, which is why these two are maps and the rest are not.
	Parameters             []map[string]any `yaml:"parameters"`
	Expectation            string           `yaml:"expectation"`
	Confidence             string           `yaml:"confidence"`
	Reversible             bool             `yaml:"reversible"`
	Requires               []string         `yaml:"requires"`
	Example                map[string]any   `yaml:"example"`
	ApplicabilityCondition string           `yaml:"applicability_condition"`
	Limitation             string           `yaml:"limitation"`
}

type rawAppliesTo struct {
	FieldTypes []string `yaml:"field_types"`
	Formats    []string `yaml:"formats"`
	Providers  []string `yaml:"providers"`
}

// toEntry turns one raw mutator into a model entry, reporting what only this
// step can see.
//
// Everything a well formed entry has to satisfy is checked by the model, which
// is where those rules belong. What is left here is what the file can get wrong
// on its own: a parameter with no default at all, and an index to name an entry
// that has no identifier to be named by.
func (m rawMutator) toEntry(index int) (catalog.Entry, []string) {
	var problems []string
	name := m.ID
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("the entry at position %d", index+1)
	}

	parameters, parameterProblems := m.parameters(name)
	problems = append(problems, parameterProblems...)
	example, exampleProblems := m.example(name)
	problems = append(problems, exampleProblems...)

	required := make([]capability.Name, 0, len(m.Requires))
	for _, capabilityName := range m.Requires {
		required = append(required, capability.Name(capabilityName))
	}

	return catalog.Entry{
		ID:                     catalog.ID(m.ID),
		Layer:                  catalog.Layer(m.Layer),
		Summary:                strings.TrimSpace(m.Summary),
		AppliesTo:              catalog.AppliesTo(m.AppliesTo),
		Parameters:             parameters,
		Expectation:            catalog.Expectation(m.Expectation),
		Confidence:             catalog.Confidence(m.Confidence),
		Reversible:             m.Reversible,
		Requires:               required,
		Example:                example,
		ApplicabilityCondition: strings.TrimSpace(m.ApplicabilityCondition),
		Limitation:             strings.TrimSpace(m.Limitation),
	}, problems
}

// parameters reads the knobs, refusing one that has no default at all.
//
// A parameter with no default is a policy that leaves it out having no value to
// use, and the alternative to refusing is inventing one, which would be this
// tool quietly deciding how much damage to do.
func (m rawMutator) parameters(name string) ([]catalog.Parameter, []string) {
	var problems []string
	parameters := make([]catalog.Parameter, 0, len(m.Parameters))
	for index, raw := range m.Parameters {
		parameterName, _ := text(raw, "name")
		// The stand in is for the message only. Putting it in the model
		// would hand the model a parameter that looks named, and the check
		// asking whether a parameter has a name would then never fire on
		// the file that lost one.
		called := parameterName
		if called == "" {
			called = fmt.Sprintf("the parameter at position %d", index+1)
		}
		value, present := raw["default"]
		if !present {
			problems = append(problems, fmt.Sprintf(
				"%s: parameter %q has no default, so a policy that leaves it out has no value to use",
				name, called))
		}
		description, _ := text(raw, "description")
		parameters = append(parameters, catalog.Parameter{
			Name:        parameterName,
			Description: description,
			Default:     value,
		})
	}
	return parameters, problems
}

// example reads the pair that becomes a test case.
//
// Both values have to be written as text. A YAML scalar left unquoted is read
// as whatever it resembles, so an example of 010 would arrive as eight and one
// of 2025-02-28 as a date, and the published example would then differ from
// what the engine is tested against without anybody touching either. This is
// the same trap that made the seed a string in the recipe.
func (m rawMutator) example(name string) (catalog.Example, []string) {
	var problems []string
	values := map[string]string{}
	for _, key := range []string{"before", "after"} {
		raw, present := m.Example[key]
		if !present {
			problems = append(problems, fmt.Sprintf("%s: the example has no %s value", name, key))
			continue
		}
		written, ok := raw.(string)
		if !ok {
			problems = append(problems, fmt.Sprintf(
				"%s: the %s value is %T rather than text, so quote it", name, key, raw))
			continue
		}
		values[key] = written
	}
	note, _ := text(m.Example, "note")
	return catalog.Example{Before: values["before"], After: values["after"], Note: note}, problems
}

// text reads one string out of a decoded block, answering whether it was there
// and readable as text at all.
func text(block map[string]any, key string) (string, bool) {
	written, ok := block[key].(string)
	return strings.TrimSpace(written), ok
}
