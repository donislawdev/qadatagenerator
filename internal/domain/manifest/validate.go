package manifest

import (
	"fmt"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
)

// Problems reports what is missing from the envelope.
//
// It is exported because a manifest written entry by entry cannot be held
// against the model at the end: by then the entries are already on a disk. A
// streaming writer asks this before it writes anything, so a document the model
// would refuse never reaches somebody who would read it as a promise.
//
// Every one of these is a field somebody reading the manifest a year later
// needs in order to know what they are looking at. A manifest without a seed
// cannot be reproduced, and one without a locale cannot explain why a mutator
// is absent, because availability at L3 and L4 follows from the provider the
// locale chose.
func (r Run) Problems() []string {
	required := []struct {
		name  string
		value string
		why   string
	}{
		{"generated_at", r.GeneratedAt, "nothing says when this ran"},
		{"seed", r.Seed, "the run cannot be reproduced"},
		{"locale", r.Locale, "nothing explains which mutators were even available"},
		{"catalog_version", r.CatalogVersion, "nothing says which catalog these identifiers came from"},
	}
	var problems []string
	for _, field := range required {
		if strings.TrimSpace(field.value) == "" {
			problems = append(problems, fmt.Sprintf("the envelope has no %s, so %s", field.name, field.why))
		}
	}
	if r.RowsTotal < 0 || r.RowsMutated < 0 {
		problems = append(problems, "the envelope counts a negative number of rows")
	}
	if r.RowsMutated > r.RowsTotal {
		problems = append(problems, fmt.Sprintf(
			"the envelope says %d of %d rows were mutated", r.RowsMutated, r.RowsTotal))
	}
	return append(problems, r.Inference.problems()...)
}

// problems reports what would make an inference unreadable to whoever has to
// judge it.
//
// # Why an inference with no stated limits is refused outright
//
// The other three rules here are ordinary: a guess about no column, or one with
// no evidence, tells nobody anything. The fourth is the one worth defending. An
// inference that lists what it recognised and stops reads as a complete account
// of the file, so a reader learns nothing about the columns where the damage
// they ordered quietly had nothing to work with. Making the model refuse it is
// the same move as making a defect that did not happen still be an entry, one
// level up: the silence that costs the most is the one that looks like an
// answer.
func (i *Inference) problems() []string {
	if i == nil {
		return nil
	}
	var problems []string
	if strings.TrimSpace(i.Method) == "" {
		problems = append(problems, "the envelope infers a schema without saying how, so nobody "+
			"reading it can tell one way of looking from another")
	}
	if len(i.Columns) == 0 {
		problems = append(problems, "the envelope infers a schema and names no column it decided "+
			"anything about")
	}
	for _, column := range i.Columns {
		if strings.TrimSpace(column.Field) == "" {
			problems = append(problems, "an inferred column has no name, so nothing says which "+
				"column was looked at")
			continue
		}
		if strings.TrimSpace(column.Basis) == "" {
			problems = append(problems, fmt.Sprintf(
				"the inference about %s carries no basis, so it is a conclusion nobody can "+
					"argue with", column.Field))
		}
	}
	if len(i.NotInferred) == 0 {
		problems = append(problems, "the envelope infers a schema without saying what it does "+
			"not infer, which would read as a complete account of the file")
	}
	return problems
}

// defectProblems collects what is wrong across the entries, including the one
// thing no single entry can see: two of them sharing an identifier.
func defectProblems(defects []Defect) []string {
	var problems []string
	seen := make(map[string]bool, len(defects))
	for index, defect := range defects {
		name := defect.ID
		if strings.TrimSpace(name) == "" {
			problems = append(problems, fmt.Sprintf(
				"the entry at position %d has no identifier, so no test can refer to it", index+1))
			continue
		}
		if seen[name] {
			problems = append(problems, fmt.Sprintf("%s: two entries carry this identifier", name))
		}
		seen[name] = true
		problems = append(problems, defect.Problems()...)
	}
	return problems
}

// Problems reports everything wrong with one entry.
//
// Exported for the same reason as the one on the envelope: an entry is written
// as soon as it happens, so it has to be checkable one at a time.
func (d Defect) Problems() []string {
	var problems []string
	problems = append(problems, d.vocabularyProblems()...)
	problems = append(problems, d.outcomeProblems()...)
	problems = append(problems, d.Location.problemsAt(d.Layer, d.ID)...)
	return problems
}

// vocabularyProblems checks the values copied out of the catalog entry.
//
// A manifest naming a layer or an expectation this build has never heard of is
// as wrong as a catalog entry doing it, and the reader of a manifest has no
// catalog beside them to check against.
func (d Defect) vocabularyProblems() []string {
	var problems []string
	if strings.TrimSpace(string(d.Mutator)) == "" {
		problems = append(problems, fmt.Sprintf("%s: no mutator identifier, so nothing says what was done", d.ID))
	}
	if !catalog.KnownLayer(d.Layer) {
		problems = append(problems, fmt.Sprintf("%s: layer %q is not one this build knows", d.ID, d.Layer))
	}
	if !catalog.KnownExpectation(d.Expectation) {
		problems = append(problems, fmt.Sprintf(
			"%s: expectation %q is not one of rejection, acceptance or warning", d.ID, d.Expectation))
	}
	if !catalog.KnownConfidence(d.Confidence) {
		problems = append(problems, fmt.Sprintf(
			"%s: confidence %q is neither hard nor soft", d.ID, d.Confidence))
	}
	if strings.TrimSpace(d.Rationale) == "" {
		problems = append(problems, fmt.Sprintf(
			"%s: no rationale, and the entry exists to be read by somebody who was not here", d.ID))
	}
	return problems
}

// outcomeProblems is the rule the whole product rests on.
//
// Damage that could not be applied is still an entry, and an entry that says
// so without saying why is the silence this tool sells as a defect in other
// people's software. The other half matters too: an entry that did not happen
// cannot carry values, because carrying them would describe a change nobody
// made.
func (d Defect) outcomeProblems() []string {
	if d.Applied {
		if strings.TrimSpace(d.FailureReason) != "" {
			return []string{fmt.Sprintf(
				"%s: applied and carrying a failure reason, so the entry says both things at once", d.ID)}
		}
		if d.FailureKind != "" {
			return []string{fmt.Sprintf(
				"%s: applied and carrying a failure kind, so the entry says both things at once", d.ID)}
		}
		return nil
	}
	var problems []string
	if strings.TrimSpace(d.FailureReason) == "" {
		problems = append(problems, fmt.Sprintf(
			"%s: not applied and no reason given, which is the silence this format exists to prevent", d.ID))
	}
	// The same rule in the dimension a machine reads. An entry carrying only
	// the sentence would leave anything downstream matching on prose to tell one
	// kind of failure from another, which is the pattern this project has a
	// measured defect from.
	if d.FailureKind == "" {
		problems = append(problems, fmt.Sprintf(
			"%s: not applied and no failure kind given, so nothing but the prose says what happened", d.ID))
	} else if !d.FailureKind.Known() {
		problems = append(problems, fmt.Sprintf(
			"%s: %q is not a failure kind this format defines", d.ID, d.FailureKind))
	}
	if d.HasValueBefore || d.HasValueAfter {
		problems = append(problems, fmt.Sprintf(
			"%s: not applied and carrying values, so it describes a change nobody made", d.ID))
	}
	return problems
}

// problemsAt checks that a location says enough for its layer.
//
// The layers damage different things, so they need different places named. A
// byte offset means nothing for a business rule spanning two records, and an
// entity and a field mean nothing for a file whose bytes were rewritten.
func (l Location) problemsAt(layer catalog.Layer, id string) []string {
	switch layer {
	case catalog.Bytes:
		// Damage to the whole file names no place, which is why an empty
		// location is legal here and nowhere else.
		return nil
	case catalog.Container:
		if l.Row > 0 || l.Path != "" {
			return nil
		}
		return []string{fmt.Sprintf("%s: a container defect names neither a row nor a path", id)}
	case catalog.SetLevel:
		return l.setLevelProblems(id)
	default:
		// Everything else damages a value, including the cross cutting
		// modules. They cut across the layers rather than replacing them, so
		// they name a place the same way.
		if l.Path != "" || (l.Entity != "" && l.Row > 0 && l.Field != "") {
			return nil
		}
		return []string{fmt.Sprintf(
			"%s: a defect in a value needs an entity, a row and a field, or a path", id)}
	}
}

// setLevelProblems covers L5, where naming one place is never enough.
func (l Location) setLevelProblems(id string) []string {
	var problems []string
	if l.Empty() {
		problems = append(problems, fmt.Sprintf("%s: a set integrity defect names no place at all", id))
	}
	if len(l.Related) == 0 {
		problems = append(problems, fmt.Sprintf(
			"%s: a set integrity defect spans more than one record, and naming one of them is useless"+
				" to whoever has to look", id))
	}
	return problems
}
