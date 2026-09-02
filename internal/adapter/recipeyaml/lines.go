package recipeyaml

import (
	"strconv"

	"github.com/goccy/go-yaml/ast"
)

// linesIn maps every place in a recipe onto the line it was written on.
//
// # Why this exists
//
// A problem names a place, such as entities.customer.fields.email, and until
// 2026-09-01 that was all it named. On the command line a path is enough,
// because the person reading it has the file open beside the output. In a
// window the recipe is the screen, and docs/08 promises that a problem is shown
// at the place it belongs to. A path with no line makes that promise into a
// list of paths.
//
// # Why the walk is here rather than beside whoever wants a line
//
// The spelling of a place is decided by the reader in this package: a key under
// a mapping is parent.key, an entry of a sequence is parent[index], and a key
// at the top of the file is bare. A second walk elsewhere would be a second
// copy of that rule, and the two would part company at the first place that
// gains a level. Keeping both in one package does not make them one rule, so a
// guard closes them against each other on the documented recipe, in both
// directions.
//
// # What it cannot answer, and it is the pre-existing limit of a path
//
// An entity, a field or a rule may be named anything YAML accepts as a key,
// dots included. A name carrying a dot makes the path ambiguous, and it was
// ambiguous before this walk existed: entities.a.b.fields.x reads as an entity
// named a.b and as a field b of an entity a. Nothing here guesses between them.
// A place this walk cannot resolve to exactly one line is left without one, and
// a problem without a line is drawn without a line rather than beside the wrong
// one.
func linesIn(file *ast.File) map[string]int {
	found := map[string]int{}
	if file == nil {
		return found
	}
	// Ambiguous is what a path resolves to when two places share it. The value
	// is kept out of found rather than fought over, so the caller sees the same
	// thing as for a place nobody wrote: no line.
	ambiguous := map[string]bool{}
	for _, document := range file.Docs {
		if document != nil {
			walkLines(document.Body, "", found, ambiguous)
		}
	}
	for path := range ambiguous {
		delete(found, path)
	}
	return found
}

// walkLines records where each place under node was written.
//
// The two shapes of a mapping are both handled on purpose. The parser gives a
// mapping of several pairs as one node and a mapping of exactly one pair as the
// pair itself, and a recipe is full of one pair mappings: every rule of a
// consistency section is one.
func walkLines(node ast.Node, at string, found map[string]int, ambiguous map[string]bool) {
	switch typed := node.(type) {
	case *ast.MappingNode:
		for _, pair := range typed.Values {
			walkPair(pair, at, found, ambiguous)
		}
	case *ast.MappingValueNode:
		walkPair(typed, at, found, ambiguous)
	case *ast.SequenceNode:
		for index, entry := range typed.Values {
			under := at + "[" + strconv.Itoa(index) + "]"
			// The entry itself is a place, not only the keys inside it. A
			// consistency rule is reported at entities.x.consistency[0] and
			// nothing under that path is ever named, so without this line the
			// rules of a recipe are exactly the problems drawn without one.
			if entry != nil {
				if token := entry.GetToken(); token != nil {
					record(under, token.Position.Line, found, ambiguous)
				}
			}
			walkLines(entry, under, found, ambiguous)
		}
	}
}

// walkPair records one key and then whatever hangs under it.
func walkPair(pair *ast.MappingValueNode, at string, found map[string]int, ambiguous map[string]bool) {
	if pair == nil || pair.Key == nil {
		return
	}
	key := pair.Key.GetToken()
	if key == nil {
		return
	}
	path := key.Value
	if at != "" {
		path = at + "." + key.Value
	}
	record(path, key.Position.Line, found, ambiguous)
	walkLines(pair.Value, path, found, ambiguous)
}

// record keeps where a place was written, and remembers a path claimed twice.
//
// Two places under one path is what a name carrying a dot produces. Neither
// answer is better than the other, so the path is marked and dropped rather
// than settled by whichever was walked last.
func record(path string, line int, found map[string]int, ambiguous map[string]bool) {
	if _, taken := found[path]; taken {
		ambiguous[path] = true
	}
	found[path] = line
}
