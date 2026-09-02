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
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"

	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

// Known keys, listed once so that the messages and the walk cannot disagree.
var (
	topLevelKeys = []string{"version", "seed", "locale", "name", "tag", "entities", "output", "policy"}
	policyKeys   = []string{"layers", "rows", "per_row", "exclude_fields", "only_when"}

	// How many defects one row may carry. The two are what the format
	// promises: exactly one isolates a cause, several at once tests how
	// failures cascade.
	perRowChoices = []string{"exactly_one", "any"}
	entityKeys    = []string{"count", "primary_key", "fields", "consistency", "belongs_to"}
	fieldKeys     = []string{"type", "generator", "expression", "unique", "required", "max_length", "counted_in", "empty", "values"}
	outputKeys    = []string{"format", "dir", "file_per_entity", "encoding", "delimiter"}

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

// MaxSize is the largest recipe this build will read.
//
// # Why there is a limit at all
//
// A recipe comes from somebody else's repository. In a pipeline that means it
// can arrive through a pull request, so it is untrusted input in the same
// sense a downloaded file is, however ordinary it usually looks.
//
// Measured on 2026-08-02: reading a deliberately nested document costs time
// that grows with the square of its size. A hundred thousand levels of nesting
// is a hundred and ninety five kilobytes and takes close to four seconds, and
// doubling the input quadruples that. The cost is entirely inside the YAML
// library, measured separately, so nothing this package does with the parsed
// tree can bring it down. The only lever before parsing is the size of the
// input, which is why the limit is here rather than on the depth.
//
// # Why this number
//
// The first design principle of the format is that a recipe is readable and
// writable by hand and reviewed in a pull request. The documented example is
// about one and a half kilobytes, so this leaves room for a recipe a hundred
// and seventy times larger, while holding the worst case at a few seconds
// rather than minutes.
//
// The refusal carries the configuration exit code rather than one of its own.
// A recipe this build will not read is a recipe that cannot be used, which is
// what that code already means, and adding a code after a release is a
// breaking change for everybody reading them.
const MaxSize = 256 * 1024

// MaxNestingDepth is the deepest a recipe may nest the bracketed shapes, which
// are the flow sequence written with square brackets and the flow mapping
// written with braces. Both count towards the same depth.
//
// # Why a second limit when there is already one on size
//
// The size limit was set from a measurement of time and it holds time down. It
// did nothing about memory, because memory was never measured, and the note
// beside it said the only lever available before parsing was the size of the
// input. That was wrong. There is a second lever, it is cheap, and it is the
// one that matters.
//
// Measured on 2026-08-02, on square brackets:
//
//	8 000 open brackets    8 kB    46 ms    101 MB
//	16 000 open brackets   16 kB   171 ms   406 MB
//
// Memory grows faster than the input. A recipe of sixteen kilobytes,
// comfortably inside the size limit, already costs four hundred megabytes, and
// one at the size limit costs tens of gigabytes. That is not a slow build, it
// is a machine killed by the operating system, and a killed build says nothing
// about why it died.
//
// # Why braces are here too, which they were not until 2026-08-03
//
// The note above used to clear them, on one measurement reading seven megabytes
// at sixteen kilobytes. That measurement is not wrong and the conclusion drawn
// from it was: sixteen kilobytes is six percent of the largest recipe this
// build accepts, and a shape cleared at six percent of a range was never
// cleared across it. Measured again, on braces, at three sizes rather than one:
//
//	4 000 levels    20 kB    27 MB
//	8 000 levels    40 kB   194 MB
//	16 000 levels   80 kB   597 MB
//
// Six hundred megabytes at eighty kilobytes, with the limit at two hundred and
// fifty six, is the same machine killed by the operating system through the
// other spelling of the same idea. The measurement was deliberately not taken
// at the size limit itself, because a payload built to be expensive is a weapon
// aimed at whoever runs it.
//
// Ordinary indented nesting stays cleared, at nineteen megabytes for three
// hundred and fifty kilobytes, which is past the size limit already.
//
// # Why sixty four
//
// It is far above anything written by hand. The format's own deepest use of a
// bracket is a list of field names in a consistency rule, which is one level,
// and a field written inline as a mapping, which is another. Sixty four leaves
// room for a shape nobody has thought of while refusing everything a bomb
// needs.
const MaxNestingDepth = 64

// MaxDeclarableLength is the largest maximum length a field may declare.
//
// # Why there is a ceiling on a number that only describes a column
//
// The number is not only described, it is built. Damage that goes past a
// declared limit produces a value of that length plus the overshoot, so a limit
// is an instruction to put that many characters in one cell of one row of the
// output. Until this ceiling existed the format accepted any number the machine
// could hold, and a recipe is untrusted input that arrives in a pull request.
//
// # Why a million
//
// Measured on this project rather than reasoned about. The largest limit
// declared anywhere in this repository is sixty four, so a million leaves room
// for a column fifteen thousand times wider than anything here has needed.
// Producing one such value costs 3,8 ms and about a megabyte, so the ceiling is
// not defending against the cost of one value: it is defending against a
// dataset where every row carries one, which at a thousand rows is a gigabyte
// of output nobody asked a test dataset for.
//
// The refusal carries the configuration exit code, for the same reason the size
// limit above does.
const MaxDeclarableLength = 1_000_000

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
//
// # One parse, where there used to be two
//
// Counting the documents in the file needs the parser, and until 2026-08-03 it
// ran its own, inside the block whose whole stated purpose was to answer before
// the parser was handed anything. So every recipe was parsed twice and the
// block promised a protection its third check was busy spending.
//
// The count now comes from the same syntax tree the model is built from. What
// still runs before any parsing are the two checks that read bytes alone, the
// size and the nesting depth, and those are what actually hold the cost down.
func Decode(data []byte) Result {
	reader := &reader{}

	// The mark Notepad writes is removed before anything else looks at the
	// bytes, so every check below sees the file the way its author wrote it.
	data = stripBOM(data)

	if refusedBeforeParsing(reader, data) {
		return reader.result(recipe.Recipe{})
	}

	file, err := parser.ParseBytes(data, 0)
	if err != nil {
		reader.report("", "the file is not valid YAML: "+firstLine(err.Error()), "",
			"check the indentation and that every key ends with a colon")
		return reader.result(recipe.Recipe{})
	}
	// Where each place was written, taken from the tree this same parse
	// produced. Nothing below has to ask for it, because every problem goes
	// through one call and that call looks it up.
	reader.lines = linesIn(file)

	body, ok := reader.singleDocument(file)
	if !ok {
		return reader.result(recipe.Recipe{})
	}

	// Numbers are checked against how they were WRITTEN, which is only possible
	// here: one line further down the spellings have already become values and
	// 010 is indistinguishable from 8.
	if refuseCleverNumberSpellings(reader, body) {
		return reader.result(recipe.Recipe{})
	}

	var root any
	if err := yaml.NodeToValue(body, &root, yaml.UseOrderedMap()); err != nil {
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

// singleDocument returns the one document carrying content, or refuses the
// file and reports why.
//
// # Documents carrying content, not raw documents
//
// The difference was measured here rather than taken from a neighbouring
// project that hit the same question. On this version of the parser a recipe
// followed by a trailing separator reads as two raw documents and one carrying
// content, so counting the raw ones would refuse a file somebody had every
// right to write. The neighbouring project measured a different shape catching
// them out, which is the reason for measuring rather than copying.
//
// Verified on six layouts: a bare recipe, a leading separator, a leading
// comment before a separator, a trailing separator, a trailing comment, and two
// recipes. Only the last counts as two.
//
// A file holding a second recipe is refused rather than half read. Taking the
// first and dropping the rest without a word is the shape of silence this
// project argues against: the caller gets some of the entities they asked for
// and an exit code saying everything went well.
func (r *reader) singleDocument(file *ast.File) (ast.Node, bool) {
	var carrying []ast.Node
	for _, document := range file.Docs {
		if document.Body != nil {
			carrying = append(carrying, document.Body)
		}
	}

	switch {
	case len(carrying) > 1:
		r.report("", fmt.Sprintf("the file holds %d recipes and this build reads one", len(carrying)),
			"one recipe per file",
			"split them into separate files, or remove the --- separator between them")
		return nil, false
	case len(carrying) == 0:
		r.report("", "the recipe is empty", "",
			"a recipe needs at least a version and one entity")
		return nil, false
	}
	return carrying[0], true
}

type reader struct {
	problems []recipe.Problem
	// lines says which line each place was written on, or nothing before the
	// file has been parsed. A place that is not in here has no line, which is
	// an ordinary answer.
	lines     map[string]int
	unchecked []recipe.Unchecked
}

// report records one problem.
//
// Every part goes through shorten, and this is the only place it has to. The
// names of entities, fields and keys all come from the recipe and all end up
// inside one of these sentences, so bounding the sentence here bounds every
// one of them without anybody having to remember at the dozen places that
// build one.
func (r *reader) report(where, what, allowed, hint string) {
	r.problems = append(r.problems, recipe.Problem{
		Where:   shorten(where),
		What:    shorten(what),
		Allowed: shorten(allowed),
		Hint:    shorten(hint),
		// Resolved here rather than by whoever shows the problem, because this
		// is the one place a place gets its name. A caller joining paths to
		// lines afterwards would be spelling those paths a second time.
		Line: r.lines[where],
	})
}

func (r *reader) note(where, what, reason string) {
	r.unchecked = append(r.unchecked, recipe.Unchecked{
		Where:  shorten(where),
		What:   shorten(what),
		Reason: shorten(reason),
	})
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
		case "policy":
			built.Policy = r.policy(item.Value)
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
			built.HasCount = true
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
		reportedBefore := len(r.problems)
		with := r.listOrOne(at, items[0].Value)
		if !r.ruleSaysSomething(at, parts[0], with, reportedBefore) {
			continue
		}
		rules = append(rules, recipe.Rule{
			Field: parts[0],
			Kind:  kind,
			With:  with,
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

// policy reads what a run should break.
//
// Nothing here checks that a layer has damage behind it in this build. That
// question belongs to whoever holds the catalog, and this package holds only
// the file. Asking it here would put a copy of the catalog's vocabulary in the
// reader, and the reader would then answer differently from the verb that runs
// the damage.
func (r *reader) policy(node any) recipe.Policy {
	var built recipe.Policy
	items, ok := r.mapping("policy", node)
	if !ok {
		return built
	}
	for _, item := range items {
		key, ok := r.key("policy", item)
		if !ok {
			continue
		}
		where := "policy." + key
		switch key {
		case "layers":
			built.Layers = r.listOrOne(where, item.Value)
		case "rows":
			built.Rows, _ = r.text(where, item.Value)
		case "per_row":
			built.PerRow = r.oneOf(where, item.Value, perRowChoices)
		case "exclude_fields":
			built.ExcludeFields = r.listOrOne(where, item.Value)
		case "only_when":
			built.OnlyWhen, _ = r.text(where, item.Value)
		default:
			r.report(where, "the policy section has no such key", strings.Join(policyKeys, ", "),
				"remove the line or check the spelling")
		}
	}
	if len(built.Layers) == 0 {
		r.report("policy.layers", "the policy names no layers, so it orders no damage",
			"one or more of L0, L1, L2, L3, L4, L5, SEC, UNI, REV",
			"name the lines of defence to attack, or remove the policy section")
	}
	return built
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

// The checks that need more than one key are in checks.go, beside each other.
