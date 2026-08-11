package values

import (
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// CapabilitiesOf is what the values of one recipe bring into a run.
//
// # Why it depends on the recipe rather than on the build
//
// Two of the five suppliers meet here: the identifier providers and the word
// lists. Both are locale bound and both are only present if something actually
// uses them. A recipe with no national identifier in it does not supply
// identifier.check-digit, so a mutator that breaks a check digit has no digit
// to break, and the manifest should say that rather than claim the run could
// have done it.
//
// Answering from the build instead would be the mistake DEC-89 named: reading
// availability off the wrong supplier leaves mutators permanently unapplied, or
// permanently claimed.
func CapabilitiesOf(rec recipe.Recipe, source Source) []capability.Name {
	found := map[capability.Name]bool{}
	for _, entity := range rec.Entities {
		for _, field := range entity.Fields {
			for _, name := range capabilitiesOfGenerator(field.Generator) {
				found[name] = true
			}
		}
	}
	names := make([]capability.Name, 0, len(found))
	for name := range found {
		names = append(names, name)
	}
	return names
}

// CapabilitiesOfView is the same question asked of a dataset nobody described.
//
// # Why it cannot be the function above
//
// That one walks the generators a recipe named, and in the mode that reads
// somebody else's file there are no generators: nothing produced those values,
// they were already there. What a column follows was worked out by offering its
// values to every scheme, and the answer is on the column rather than on a
// generator.
//
// The capabilities themselves come from the same place in both cases, which is
// the provider itself. So the two functions differ in how they find the schemes
// present in a run and agree on everything after that, and a run that recognised
// a column of national identifiers can break a check digit whether a recipe said
// so or whether the values did.
func CapabilitiesOfView(view recipe.SchemaView, source Source) []capability.Name {
	found := map[capability.Name]bool{}
	for _, entity := range view.EntityNames() {
		for _, field := range view.FieldsOf(entity) {
			if field.Provider == "" {
				continue
			}
			scheme, carried := source.ProviderNamed(field.Provider)
			if !carried {
				continue
			}
			for _, name := range scheme.Capabilities() {
				found[name] = true
			}
		}
	}
	names := make([]capability.Name, 0, len(found))
	for name := range found {
		names = append(names, name)
	}
	return names
}

// capabilitiesOfGenerator asks what stands behind a generator what it brings.
//
// The answer comes from the provider itself rather than from a table here,
// because a provider declaring a capability it does not implement already fails
// the build, and a second list would be a place for the two to disagree.
func capabilitiesOfGenerator(name string) []capability.Name {
	known, found := producers[name]
	if !found {
		return nil
	}
	if known.behind != nil {
		return known.behind.Capabilities()
	}
	if known.list != "" {
		// A word list, which is the fourth supplier. What it brings is the kind
		// of value it holds rather than anything computed, and today that is the
		// international characters a name or a place carries. The other three
		// corpus capabilities describe lists this build does not ship: values
		// that coerce, sentinels, and text chosen to stress Unicode.
		return []capability.Name{capability.CorpusNamesInternational}
	}
	return nil
}
