package recipe

import "github.com/donislawdev/qadatagenerator/internal/domain/capability"

// CapabilitiesOf is what the shape of the data brings into a run.
//
// # Why this is worked out rather than declared
//
// The schema is one of the five suppliers of capabilities, and unlike the other
// four it has nothing to declare: what it supplies depends on what a particular
// recipe says. A recipe that declares no maximum length anywhere does not
// supply schema.max-length, and a mutator that overruns a declared limit has
// nothing to overrun. Claiming it anyway would let such a mutator be attempted
// and then refused for the wrong reason, or worse, appear to have worked.
//
// # What the answer is used for
//
// The engine asks whether a run can do what an entry requires, and says in the
// manifest when it cannot. This list is therefore the difference between an
// entry reading "this run cannot declare a maximum length" and an entry that
// never appears at all.
//
// The order is not fixed here. Every caller that shows these to a person sorts
// them, and capability.Set does its own sorting for the same reason.
func CapabilitiesOf(view SchemaView) []capability.Name {
	found := map[capability.Name]bool{}
	for _, entity := range view.EntityNames() {
		if len(view.PrimaryKeyOf(entity)) > 0 {
			found[capability.KeyPrimary] = true
		}
		if len(view.ParentsOf(entity)) > 0 {
			found[capability.RelationForeignKey] = true
		}
		if len(view.RulesOf(entity)) > 0 {
			found[capability.SchemaFieldRelations] = true
		}
		for _, field := range view.FieldsOf(entity) {
			gatherField(field, found)
		}
	}
	names := make([]capability.Name, 0, len(found))
	for name := range found {
		names = append(names, name)
	}
	return names
}

// CapabilitiesOfField is what ONE column brings, which is a different question
// from what a run brings.
//
// A run carries a capability when anything in it supplies one, and for a
// handful of them that is the wrong question: a maximum length belongs to a
// column, so a run carrying schema.max-length says only that some column
// declares a limit. Damage aimed at a column without one has to be refused
// there, and this is what lets the engine ask.
func CapabilitiesOfField(field FieldInfo) []capability.Name {
	found := map[capability.Name]bool{}
	gatherField(field, found)
	names := make([]capability.Name, 0, len(found))
	for name := range found {
		names = append(names, name)
	}
	return names
}

// gatherField records what one field supplies.
func gatherField(field FieldInfo, found map[capability.Name]bool) {
	if field.Type != "" {
		found[capability.SchemaFieldType] = true
	}
	if field.HasMaxLength {
		found[capability.SchemaMaxLength] = true
	}
	if len(field.Values) > 0 {
		found[capability.SchemaEnumValues] = true
	}
	if field.Required {
		found[capability.SchemaRequiredFields] = true
	}
	if field.Unique {
		found[capability.KeyUnique] = true
	}
}
