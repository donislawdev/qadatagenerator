// Package recipeyaml reads a recipe written in YAML.
//
// It is the only place that knows the recipe is YAML at all. The model it
// produces knows nothing about the file, and the surfaces that show problems
// know nothing about keys, which is what lets the format grow without the rest
// of the program noticing.
//
// Two habits run through this file and both are deliberate.
//
// It reports everything it found rather than stopping at the first problem. A
// person fixing a recipe wants the list, not one line at a time, and a tool
// that hands them out one by one turns a five minute fix into five rounds.
//
// It says out loud what it did not check. Generator settings cannot be
// verified before the generators exist, so they are carried and reported as
// unchecked. Silence there would read as approval.
package recipeyaml

import (
	"fmt"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"

	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

// Known keys, listed once so that the messages and the walk cannot disagree.
var (
	topLevelKeys = []string{"version", "seed", "locale", "name", "tag", "entities", "output"}
	entityKeys   = []string{"count", "primary_key", "fields", "consistency", "belongs_to"}
	fieldKeys    = []string{"type", "generator", "expression", "unique", "required", "max_length", "counted_in", "empty", "values"}
	outputKeys   = []string{"format", "dir", "file_per_entity", "encoding", "delimiter"}

	// How a maximum length can be counted, taken from the model so that the
	// list a user is shown cannot drift from the list the model accepts.
	lengthUnits = []string{string(recipe.Characters), string(recipe.Bytes), string(recipe.Graphemes)}

	// The locales the catalog and the providers are built around.
	knownLocales = []string{"en-US", "pl-PL"}

	// Formats this build can write. The others are named in the
	// documentation and refused with a reason rather than accepted and
	// quietly ignored at the end of a long run.
	writableFormats = []string{"csv", "json"}
)

// Result is everything reading a recipe produced.
type Result struct {
	Recipe    recipe.Recipe
	Problems  []recipe.Problem
	Unchecked []recipe.Unchecked
}

// Valid reports whether the recipe can be used.
func (r Result) Valid() bool { return len(r.Problems) == 0 }

// Decode turns the bytes of a recipe into a model, a list of problems and a
// list of what could not be checked.
func Decode(data []byte) Result {
	reader := &reader{}

	var root any
	if err := yaml.UnmarshalWithOptions(data, &root, yaml.UseOrderedMap()); err != nil {
		reader.report("", "the file is not valid YAML: "+firstLine(err.Error()), "",
			"check the indentation and that every key ends with a colon")
		return reader.result(recipe.Recipe{})
	}
	if root == nil {
		reader.report("", "the recipe is empty", "",
			"a recipe needs at least a version and one entity")
		return reader.result(recipe.Recipe{})
	}

	top, ok := root.(yaml.MapSlice)
	if !ok {
		reader.report("", "the recipe has to be a mapping of keys", "",
			"start the file with a line such as version: \"0.1\"")
		return reader.result(recipe.Recipe{})
	}

	built := reader.recipe(top)
	return reader.result(built)
}

type reader struct {
	problems  []recipe.Problem
	unchecked []recipe.Unchecked
}

func (r *reader) report(where, what, allowed, hint string) {
	r.problems = append(r.problems, recipe.Problem{Where: where, What: what, Allowed: allowed, Hint: hint})
}

func (r *reader) note(where, what, reason string) {
	r.unchecked = append(r.unchecked, recipe.Unchecked{Where: where, What: what, Reason: reason})
}

func (r *reader) result(built recipe.Recipe) Result {
	recipe.SortProblems(r.problems)
	sort.SliceStable(r.unchecked, func(i, j int) bool {
		if r.unchecked[i].Where != r.unchecked[j].Where {
			return r.unchecked[i].Where < r.unchecked[j].Where
		}
		return r.unchecked[i].What < r.unchecked[j].What
	})
	return Result{Recipe: built, Problems: r.problems, Unchecked: r.unchecked}
}

func (r *reader) recipe(top yaml.MapSlice) recipe.Recipe {
	var built recipe.Recipe
	seen := map[string]bool{}

	for _, item := range top {
		key, ok := r.key("", item)
		if !ok {
			continue
		}
		seen[key] = true

		switch key {
		case "version":
			built.Version, _ = r.text(key, item.Value)
		case "seed":
			built.Seed, _ = r.text(key, item.Value)
		case "locale":
			built.Locale = r.oneOf(key, item.Value, knownLocales)
		case "name":
			built.Name, _ = r.text(key, item.Value)
		case "tag":
			built.Tag, _ = r.text(key, item.Value)
		case "entities":
			built.Entities = r.entities(item.Value)
		case "output":
			built.Output = r.output(item.Value)
		default:
			r.report(key, "the recipe has no such key", strings.Join(topLevelKeys, ", "),
				"remove the line or check the spelling")
		}
	}

	r.requireVersion(seen, built.Version)
	if !seen["entities"] {
		r.report("entities", "the recipe declares no entities", "",
			"add an entities section with at least one entity")
	}
	r.checkReferences(built)
	return built
}

// requireVersion refuses a recipe written for a schema this build does not
// speak, rather than reading it as far as it can and producing something the
// user did not ask for.
func (r *reader) requireVersion(seen map[string]bool, declared string) {
	if !seen["version"] {
		r.report("version", "the recipe does not say which schema version it is written for",
			version.Recipe, "add version: \""+version.Recipe+"\" as the first line")
		return
	}
	if declared != "" && declared != version.Recipe {
		r.report("version", "this build does not speak recipe schema "+declared,
			version.Recipe, "regenerate the recipe or install a build that speaks it")
	}
}

func (r *reader) entities(node any) []recipe.Entity {
	items, ok := r.mapping("entities", node)
	if !ok {
		return nil
	}
	if len(items) == 0 {
		r.report("entities", "the entities section is empty", "",
			"declare at least one entity with its fields")
		return nil
	}

	// Nothing here looks for a name declared twice. Measured 2026-08-01: the
	// parser refuses a repeated key at every level and says where both are,
	// which is better than anything this walk could report, because it knows
	// the line numbers and this walk does not.
	var entities []recipe.Entity
	for _, item := range items {
		name, ok := r.key("entities", item)
		if !ok {
			continue
		}
		entities = append(entities, r.entity(name, "entities."+name, item.Value))
	}
	return entities
}

func (r *reader) entity(name, where string, node any) recipe.Entity {
	built := recipe.Entity{Name: name}
	items, ok := r.mapping(where, node)
	if !ok {
		return built
	}

	// The place each rule was written, kept beside the rules themselves. A
	// rule that could not be read is left out of the model, so counting
	// positions in the model afterwards would point at the wrong line.
	var ruleWheres []string
	seen := map[string]bool{}
	for _, item := range items {
		key, ok := r.key(where, item)
		if !ok {
			continue
		}
		seen[key] = true
		switch key {
		case "count":
			built.Count = r.count(where+".count", item.Value)
		case "primary_key":
			built.PrimaryKey = r.listOrOne(where+".primary_key", item.Value)
		case "fields":
			built.Fields = r.fields(where+".fields", item.Value)
		case "consistency":
			built.Consistency, ruleWheres = r.consistency(where+".consistency", item.Value)
		case "belongs_to":
			built.Parents = r.parents(where+".belongs_to", item.Value)
		default:
			r.report(where+"."+key, "an entity has no such key", strings.Join(entityKeys, ", "),
				"remove the line or check the spelling")
		}
	}
	if !seen["fields"] {
		r.report(where+".fields", "the entity declares no fields", "",
			"an entity without fields produces empty records")
	}
	// Checked here rather than in the pass over the whole recipe, because
	// this is where both halves are known and where the places the rules
	// were written are still at hand.
	r.checkRules(built, ruleWheres)
	r.checkPrimaryKey(built, where)
	return built
}

// checkPrimaryKey asks whether the key names fields that exist and names each
// of them once.
//
// A key naming a field twice is worth reporting rather than tidying away. It
// usually means somebody meant two different fields, and silently collapsing
// it would leave a one column key where the recipe asked for two.
func (r *reader) checkPrimaryKey(entity recipe.Entity, where string) {
	if len(entity.PrimaryKey) == 0 {
		return
	}
	fields := map[string]bool{}
	for _, field := range entity.Fields {
		fields[field.Name] = true
	}
	seen := map[string]bool{}
	for _, name := range entity.PrimaryKey {
		at := where + ".primary_key"
		if !fields[name] {
			r.report(at, "the entity has no field "+name,
				strings.Join(entity.FieldNames(), ", "), "check the spelling")
			continue
		}
		if seen[name] {
			r.report(at, "the key names "+name+" twice", "",
				"name each field once, or name the second field you meant")
		}
		seen[name] = true
	}
}

// checkRules asks whether every field a rule names exists in the entity.
func (r *reader) checkRules(entity recipe.Entity, wheres []string) {
	fields := map[string]bool{}
	for _, field := range entity.Fields {
		fields[field.Name] = true
	}
	for index, rule := range entity.Consistency {
		at := wheres[index]
		for _, name := range append([]string{rule.Field}, rule.With...) {
			if !fields[name] {
				r.report(at, "the entity has no field "+name,
					strings.Join(entity.FieldNames(), ", "), "check the spelling")
			}
		}
	}
}

func (r *reader) fields(where string, node any) []recipe.Field {
	items, ok := r.mapping(where, node)
	if !ok {
		return nil
	}
	if len(items) == 0 {
		r.report(where, "the entity declares no fields", "",
			"an entity without fields produces empty records")
		return nil
	}

	var fields []recipe.Field
	for _, item := range items {
		name, ok := r.key(where, item)
		if !ok {
			continue
		}
		fields = append(fields, r.field(name, where+"."+name, item.Value))
	}
	return fields
}

func (r *reader) field(name, where string, node any) recipe.Field {
	built := recipe.Field{Name: name}
	items, ok := r.mapping(where, node)
	if !ok {
		return built
	}

	for _, item := range items {
		key, ok := r.key(where, item)
		if !ok {
			continue
		}
		r.fieldKey(&built, where, key, item.Value)
	}

	r.checkField(built, where)
	return built
}

// fieldKey reads one property of a field.
func (r *reader) fieldKey(built *recipe.Field, where, key string, value any) {
	switch key {
	case "type":
		built.Type, _ = r.text(where+".type", value)
	case "generator":
		built.Generator, _ = r.text(where+".generator", value)
	case "expression":
		built.Expression, _ = r.text(where+".expression", value)
	case "unique":
		built.Unique = r.flag(where+".unique", value)
	case "required":
		built.Required = r.flag(where+".required", value)
	case "max_length":
		built.MaxLength, built.HasMaxLength = r.maxLength(where+".max_length", value)
	case "counted_in":
		built.LengthUnit = recipe.LengthUnit(r.oneOf(where+".counted_in", value, lengthUnits))
	case "empty":
		built.EmptyShare, built.HasEmptyShare = r.share(where+".empty", value)
	case "values":
		built.Values = r.list(where+".values", value)
	default:
		// A generator setting, such as the age range of a date of birth. It is
		// kept rather than refused, because the format documents this shape,
		// and it is reported rather than accepted in silence, because a
		// misspelled generic key would otherwise land here and turn itself
		// off.
		built.Parameters = append(built.Parameters, recipe.Parameter{
			Name:  key,
			Value: fmt.Sprint(value),
		})
		r.note(where+"."+key, "the setting "+key,
			"generator settings are checked by the generator, and generators arrive with the next slice. "+
				"A misspelling of "+strings.Join(fieldKeys, ", ")+" also lands here")
	}
}

// checkField asks the questions that need the whole field rather than one key.
func (r *reader) checkField(built recipe.Field, where string) {
	// A field that must always carry a value and is also asked to be empty
	// part of the time. Both halves are legal on their own, so nothing else
	// would catch this, and whichever half won at generation time would be
	// the tool deciding which one the user meant.
	if built.Required && built.HasEmptyShare && built.EmptyShare > 0 {
		r.report(where, "the field is required and is also asked to be empty in some rows",
			"either required: true or a share under empty, not both",
			"drop required if some rows should be empty, or drop empty if none should be")
	}

	// A unit with nothing to count. It reads as though a limit was set, which
	// is exactly why it is worth a line rather than being ignored.
	if built.LengthUnit != "" && !built.HasMaxLength {
		r.report(where+".counted_in", "the field says how to count a length it does not declare",
			"", "add max_length, or remove counted_in")
	}

	sources := 0
	for _, source := range []string{built.Type, built.Generator, built.Expression} {
		if source != "" {
			sources++
		}
	}
	switch sources {
	case 0:
		r.report(where, "the field does not say where its value comes from",
			"type, generator, expression", "give the field one of them")
	case 1:
	default:
		r.report(where, "the field says twice where its value comes from",
			"exactly one of type, generator, expression", "keep the one you meant")
	}
}

// consistency returns the rules it could read and, beside them, the place each
// one was written. The two travel together because a rule that could not be
// read leaves no entry, and counting positions afterwards would then name a
// line the user did not write.
func (r *reader) consistency(where string, node any) ([]recipe.Rule, []string) {
	entries, ok := r.sequence(where, node)
	if !ok {
		return nil, nil
	}

	var rules []recipe.Rule
	var wheres []string
	for index, entry := range entries {
		at := fmt.Sprintf("%s[%d]", where, index)
		items, ok := r.mapping(at, entry)
		if !ok {
			continue
		}
		if len(items) != 1 {
			r.report(at, "a rule has to be one line saying which field follows which",
				"", "write one rule per line, such as pesel matches: [birth_date, gender]")
			continue
		}
		key, ok := r.key(at, items[0])
		if !ok {
			continue
		}
		parts := strings.Fields(key)
		if len(parts) != 2 {
			r.report(at, "a rule needs a field and a way it follows the others",
				string(recipe.Matches)+", "+string(recipe.DerivedFrom),
				"write it as pesel matches: [birth_date, gender]")
			continue
		}
		kind := recipe.RuleKind(parts[1])
		if kind != recipe.Matches && kind != recipe.DerivedFrom {
			r.report(at, "there is no such way of following a field",
				string(recipe.Matches)+", "+string(recipe.DerivedFrom), "")
			continue
		}
		rules = append(rules, recipe.Rule{
			Field: parts[0],
			Kind:  kind,
			With:  r.listOrOne(at, items[0].Value),
		})
		wheres = append(wheres, at)
	}
	return rules, wheres
}

func (r *reader) parents(where string, node any) []recipe.Parent {
	items, ok := r.mapping(where, node)
	if !ok {
		return nil
	}
	var parents []recipe.Parent
	for _, item := range items {
		name, ok := r.key(where, item)
		if !ok {
			continue
		}
		at := where + "." + name
		text, ok := r.text(at, item.Value)
		if !ok {
			continue
		}
		// The relation is recorded even when the range is unreadable, so
		// that the pass over the whole recipe still asks whether the parent
		// exists. Reporting one problem per line and hiding the second
		// behind it is how a fix turns into three rounds.
		low, high, ok := parseSpan(text)
		switch {
		case !ok:
			r.report(at, "the number of children per parent is not a range", "a range such as 0-5",
				"write how many children one parent has, such as 0-5")
		case low > high:
			r.report(at, "the range runs backwards", "", "write the smaller number first")
		}
		parents = append(parents, recipe.Parent{Entity: name, Min: low, Max: high})
	}
	return parents
}

func (r *reader) output(node any) recipe.Output {
	var built recipe.Output
	items, ok := r.mapping("output", node)
	if !ok {
		return built
	}
	for _, item := range items {
		key, ok := r.key("output", item)
		if !ok {
			continue
		}
		where := "output." + key
		switch key {
		case "format":
			built.Format = r.oneOf(where, item.Value, writableFormats)
		case "dir":
			built.Dir, _ = r.text(where, item.Value)
		case "file_per_entity":
			built.FilePerEntity = r.flag(where, item.Value)
		case "encoding":
			built.Encoding, _ = r.text(where, item.Value)
		case "delimiter":
			built.Delimiter, _ = r.text(where, item.Value)
		default:
			r.report(where, "the output section has no such key", strings.Join(outputKeys, ", "),
				"remove the line or check the spelling")
		}
	}
	return built
}

// checkReferences answers the questions that need the whole recipe: whether
// every name a rule or an expression mentions is a field that exists.
func (r *reader) checkReferences(built recipe.Recipe) {
	declared := map[string]bool{}
	for _, entity := range built.Entities {
		declared[entity.Name] = true
	}

	for _, entity := range built.Entities {
		where := "entities." + entity.Name
		fields := map[string]bool{}
		for _, field := range entity.Fields {
			fields[field.Name] = true
		}
		parents := map[string]bool{}
		for _, parent := range entity.Parents {
			parents[parent.Entity] = true
			if !declared[parent.Entity] {
				r.report(where+".belongs_to."+parent.Entity, "the recipe has no such entity",
					strings.Join(built.EntityNames(), ", "), "check the spelling or declare it")
			}
		}

		for _, field := range entity.Fields {
			if field.Expression == "" {
				continue
			}
			at := where + ".fields." + field.Name + ".expression"
			for _, reference := range referencesIn(field.Expression) {
				owner, name, isParent := strings.Cut(reference, ".")
				switch {
				case isParent && !parents[owner]:
					r.report(at, entity.Name+" does not belong to "+owner,
						"", "declare the relation under belongs_to or use a field of this entity")
				case isParent:
					r.checkParentField(built, at, owner, name)
				case fields[reference]:
				default:
					// It may be a value the tool supplies, such as a
					// random domain. Those arrive with the generators,
					// so a name that is not a field cannot be told from
					// a typo yet.
					r.note(at, "the reference to "+reference,
						"it is not a field of this entity, so it is either a value the tool supplies or a misspelling, "+
							"and telling those apart needs the generators")
				}
			}
		}
	}
}

func (r *reader) checkParentField(built recipe.Recipe, at, parent, field string) {
	owner, ok := built.Entity(parent)
	if !ok {
		return
	}
	if _, ok := owner.Field(field); !ok {
		r.report(at, parent+" has no field "+field,
			strings.Join(owner.FieldNames(), ", "), "check the spelling")
	}
}
