package recipeyaml

import (
	"slices"
	"sort"
	"strings"
	"unicode"
)

// How Suggest reads a recipe that is still being typed: every line by its
// indentation, and the caret's own line by its punctuation as well.

// recipeLine is one line of a recipe as its indentation reads.
type recipeLine struct {
	// start is where the line begins in the text, in characters.
	start int
	// runes is the line without its line break.
	runes  []rune
	indent int
	// item is a line that begins an item of a list with a dash.
	item bool
	// key is the key the line writes, after the dash of an item, or nothing.
	key string
	// opens is a key with nothing after its colon, so a block follows.
	opens bool
	// blank is a line of nothing but spaces and a comment.
	blank bool
}

type recipeLines []recipeLine

// readLines splits the text at its line breaks, a carriage return before one
// included, and reads every line. An empty text is one empty line.
func readLines(text string) recipeLines {
	runes := []rune(text)
	var lines recipeLines
	start := 0
	for at := 0; at <= len(runes); at++ {
		if at < len(runes) && runes[at] != '\n' {
			continue
		}
		end := at
		if end > start && runes[end-1] == '\r' {
			end--
		}
		lines = append(lines, readLine(runes[start:end], start))
		start = at + 1
	}
	return lines
}

// readLine reads one line by its indentation, its dash and its key.
func readLine(runes []rune, start int) recipeLine {
	line := recipeLine{start: start, runes: runes, indent: leadingSpaces(runes)}
	content := uncommented(runes)[line.indent:]
	if isBlank(content) {
		line.blank = true
		return line
	}
	content, line.item = afterDash(content)
	if key, rest, found := splitKey(content); found {
		line.key = key
		line.opens = isBlank(rest)
	}
	return line
}

// locate is the line the caret stands on and how far into it, a caret outside
// the text held at its nearer end.
func (lines recipeLines) locate(caret int) (int, int) {
	index := max(sort.Search(len(lines), func(i int) bool { return lines[i].start > caret })-1, 0)
	column := min(max(caret-lines[index].start, 0), len(lines[index].runes))
	return index, column
}

// spot is where the caret stands, as far as offering goes.
type spot struct {
	// mapping is the keys from the top of the recipe down to the mapping the
	// caret writes in, the names of entities and fields among them.
	mapping []string
	// key is the key whose value the caret writes, or nothing while it writes
	// a key. listed says the value is one item of a list.
	key    string
	listed bool
	// rule is the field a rule of consistency is about, once a space follows
	// its name, so that what is typed now is the way it follows the others.
	rule string
	// keyed is a key being typed that already has its colon after it.
	keyed    bool
	typed    string
	from, to int
	// taken is what the mapping or the list holds besides the word at the caret.
	taken []string
}

// spotAt is where the caret stands, or false where no offer belongs: in a
// comment, in quotes, past a collection that has closed, or in a shape this
// does not read.
func (lines recipeLines) spotAt(index, column int) (spot, bool) {
	line := lines[index]
	head := line.runes[:column]
	if len(uncommented(head)) < len(head) {
		return spot{}, false
	}
	// A line of nothing but spaces up to the caret stands at the caret's
	// indent, which is where the next key will begin.
	indent := line.indent
	if isBlank(head) {
		indent = column
	}
	content, item := afterDash(head[indent:])
	parents, parent, ok := lines.parentsOf(index, indent, item)
	if !ok {
		return spot{}, false
	}
	frames, ok := scanFlow(content, false)
	if !ok {
		return spot{}, false
	}
	at, ok := placeIn(parents, frames, item)
	if !ok || !at.word(line, column, content, frames[len(frames)-1]) {
		return spot{}, false
	}
	// What is already written is left out of keys and of lists. A single
	// value has nothing beside it to leave out.
	switch {
	case at.key != "" && !at.listed:
	case len(frames) > 1:
		whole, _ := afterDash(uncommented(line.runes)[indent:])
		at.taken = entriesBeside(whole, frames[len(frames)-1])
	default:
		at.taken = lines.siblings(index, parent, indent, item)
	}
	return at, true
}

// placeIn is the mapping and the key of the place, from the keys above the
// caret's line and the collections on it before the caret.
func placeIn(parents []string, frames []flowFrame, item bool) (spot, bool) {
	at := spot{mapping: parents}
	// A rule of consistency is always an item, and the only item that is a
	// mapping. Every other item is one word of a list of names.
	ruled := sectionOf(parents) == inRules
	switch {
	case ruled && !item:
		return spot{}, false
	case item && !ruled:
		if len(parents) == 0 || len(frames) > 1 || frames[0].key != "" {
			return spot{}, false
		}
		last := len(parents) - 1
		at.mapping, at.key, at.listed = parents[:last], parents[last], true
		return at, true
	}
	for depth := 1; depth < len(frames); depth++ {
		frame := frames[depth]
		if frame.owner == "" {
			return spot{}, false
		}
		if frame.list {
			if depth != len(frames)-1 {
				return spot{}, false
			}
			at.key, at.listed = frame.owner, true
			return at, true
		}
		at.mapping = append(slices.Clone(at.mapping), frame.owner)
	}
	at.key = frames[len(frames)-1].key
	return at, true
}

// word reads the word being typed at the caret, and where it begins and ends,
// and false when what is typed there is not one word.
func (at *spot) word(line recipeLine, column int, content []rune, frame flowFrame) bool {
	begin := frame.begin
	if frame.value >= 0 {
		begin = frame.value
	}
	typed := strings.TrimLeft(string(content[begin:]), " ")
	if sectionOf(at.mapping) == inRules && at.key == "" {
		// The name of the field, then a space, then the way it follows.
		if field, kind, spaced := strings.Cut(typed, " "); spaced {
			at.rule, typed = field, strings.TrimLeft(kind, " ")
		}
	}
	if !isWord(typed) {
		return false
	}
	after := line.runes[column:]
	length := wordLength(after)
	at.typed = typed
	at.from = line.start + column - len([]rune(typed))
	at.to = line.start + column + length
	rest := strings.TrimLeft(string(after[length:]), " ")
	at.keyed = at.key == "" && strings.HasPrefix(rest, ":")
	return true
}

// parentsOf is the keys of the blocks the caret's line stands in, from the top
// down, and the line of the nearest one, or -1 at the top. It is false when a
// line above holds the caret's line in something other than a mapping.
//
// A list item may stand at the indent of the key it hangs under, which YAML
// allows, so the nearest line at the item's own indent can be its key.
func (lines recipeLines) parentsOf(index, indent int, item bool) ([]string, int, bool) {
	var path []string
	parent := -1
	limit, hangs := indent, item
	for at := index - 1; at >= 0 && (limit > 0 || hangs); at-- {
		line := lines[at]
		if line.blank || line.indent > limit {
			continue
		}
		hanging := hangs && line.indent == limit && !line.item
		if line.indent == limit && !hanging {
			continue
		}
		hangs = false
		if !line.opens || line.item {
			return nil, -1, false
		}
		if parent < 0 {
			parent = at
		}
		path = append(path, line.key)
		limit = line.indent
	}
	slices.Reverse(path)
	return path, parent, true
}

// siblings is what the block of the caret's line already holds at its indent:
// the keys, or for an item the items, every line but the caret's own.
func (lines recipeLines) siblings(index, parent, indent int, items bool) []string {
	var found []string
	for at := parent + 1; at < len(lines); at++ {
		line := lines[at]
		if line.blank || at == index {
			continue
		}
		if parent >= 0 && lines.endsBlock(parent, line, items) {
			break
		}
		if line.indent != indent || line.item != items {
			continue
		}
		if items {
			content, _ := afterDash(uncommented(line.runes)[line.indent:])
			found = append(found, strings.TrimSpace(string(content)))
		} else if line.key != "" {
			found = append(found, line.key)
		}
	}
	return found
}

// endsBlock says whether a line ends the block a parent line opens: it stands
// left of the parent, or at its indent and is not an item hanging under it.
func (lines recipeLines) endsBlock(parent int, line recipeLine, items bool) bool {
	floor := lines[parent].indent
	return line.indent < floor || (line.indent == floor && !(items && line.item))
}

// children is the lines of the keys directly inside the block a line opens, or
// at the top for -1, in the order written.
func (lines recipeLines) children(parent int) []int {
	floor := -1
	if parent >= 0 {
		floor = lines[parent].indent
	}
	child := -1
	var found []int
	for at := parent + 1; at < len(lines); at++ {
		line := lines[at]
		if line.blank {
			continue
		}
		if line.indent <= floor {
			break
		}
		if child < 0 {
			child = line.indent
		}
		if line.indent == child && !line.item && line.key != "" {
			found = append(found, at)
		}
	}
	return found
}

// keysUnder is the keys directly inside the block the path names, or nothing
// when the recipe has no such block.
func (lines recipeLines) keysUnder(path ...string) []string {
	parent := -1
	for _, key := range path {
		next := -1
		for _, at := range lines.children(parent) {
			if lines[at].key == key {
				next = at
				break
			}
		}
		if next < 0 {
			return nil
		}
		parent = next
	}
	var keys []string
	for _, at := range lines.children(parent) {
		keys = append(keys, lines[at].key)
	}
	return keys
}

// entityNames is the entities the recipe declares.
func (lines recipeLines) entityNames() []string { return lines.keysUnder("entities") }

// fieldNames is the fields one entity declares.
func (lines recipeLines) fieldNames(entity string) []string {
	return lines.keysUnder("entities", entity, "fields")
}

// allFieldNames is the fields of every entity, each name once, in the order
// written, which is what a policy leaving columns alone names.
func (lines recipeLines) allFieldNames() []string {
	var names []string
	for _, entity := range lines.entityNames() {
		for _, field := range lines.fieldNames(entity) {
			if !slices.Contains(names, field) {
				names = append(names, field)
			}
		}
	}
	return names
}

// flowFrame is the line itself, or one collection in braces or brackets on it,
// with the entry of it the scan has reached.
type flowFrame struct {
	list bool
	// owner is the key the collection is the value of, and open is where it
	// opens, -1 for the line itself.
	owner string
	open  int
	// begin is where the entry the scan has reached begins, and value is where
	// its value begins, after the key's colon, or -1 before it.
	begin, value int
	// key is that entry's key, once its colon is passed.
	key string
	// closed is a collection inside that entry which has already ended.
	closed bool
	// entries are the entries the scan has passed, with where each began.
	entries []flowEntry
}

// flowEntry is one entry of a collection: its key in a mapping, what it says
// in a list.
type flowEntry struct {
	begin int
	text  string
}

// flowScan walks the punctuation of one line.
type flowScan struct {
	content []rune
	frames  []flowFrame
	quote   rune
	// endCounts says a colon at the very end is a key's colon, which holds for
	// a whole line and not for the part of one before the caret.
	endCounts bool
	// watched is where the collection whose entries are wanted opens, and
	// seen is its entries once it has ended.
	watched int
	seen    []flowEntry
}

// scanFlow walks the content and says what is open at its end, and false when
// it ends in quotes or after a collection that has closed.
func scanFlow(content []rune, endCounts bool) ([]flowFrame, bool) {
	scan := flowScan{content: content, endCounts: endCounts, watched: -2,
		frames: []flowFrame{{open: -1, value: -1}}}
	for at := range content {
		scan.step(at)
	}
	last := scan.frames[len(scan.frames)-1]
	return scan.frames, scan.quote == 0 && !last.closed
}

// step reads one character.
func (s *flowScan) step(at int) {
	r := s.content[at]
	top := &s.frames[len(s.frames)-1]
	switch {
	case s.quote != 0:
		if r == s.quote {
			s.quote = 0
		}
	case r == '"' || r == '\'':
		s.quote = r
	case r == ':' && !top.list && top.value < 0 && s.colonEnds(at):
		top.key = unquoted(strings.TrimSpace(string(s.content[top.begin:at])))
		top.value = at + 1
	case r == '{' || r == '[':
		s.frames = append(s.frames, flowFrame{list: r == '[', owner: top.key, open: at, begin: at + 1, value: -1})
	case r == ',' && len(s.frames) > 1:
		top.finish(s.content, at)
		top.begin, top.value, top.key, top.closed = at+1, -1, "", false
	case (r == '}' || r == ']') && len(s.frames) > 1:
		top.finish(s.content, at)
		if top.open == s.watched {
			s.seen = top.entries
		}
		s.frames = s.frames[:len(s.frames)-1]
		s.frames[len(s.frames)-1].closed = true
	}
}

// colonEnds says whether the colon at that place ends a key: a blank follows
// it, or the end does where the end counts.
func (s *flowScan) colonEnds(at int) bool {
	if at+1 == len(s.content) {
		return s.endCounts
	}
	return s.content[at+1] == ' ' || s.content[at+1] == '\t'
}

// finish records the entry the scan stands in, which ends at that place.
func (f *flowFrame) finish(content []rune, end int) {
	text := f.key
	if f.list || text == "" {
		text = strings.TrimSpace(string(content[f.begin:end]))
	}
	f.entries = append(f.entries, flowEntry{begin: f.begin, text: text})
}

// entriesBeside is what the collection the caret is in holds on the whole
// line, besides the entry the caret is in.
func entriesBeside(whole []rune, frame flowFrame) []string {
	scan := flowScan{content: whole, endCounts: true, watched: frame.open,
		frames: []flowFrame{{open: -1, value: -1}}}
	for at := range whole {
		scan.step(at)
	}
	entries := scan.seen
	for _, open := range scan.frames {
		if open.open == frame.open {
			// The line ends before the collection does, which is how a
			// collection being typed looks.
			open.finish(whole, len(whole))
			entries = open.entries
		}
	}
	var taken []string
	for _, entry := range entries {
		if entry.begin != frame.begin && entry.text != "" {
			taken = append(taken, entry.text)
		}
	}
	return taken
}

// splitKey is the key a line's content begins with and what follows its colon,
// and false when the content writes no key, a collection or a plain value
// among them.
func splitKey(content []rune) (string, []rune, bool) {
	var quote rune
	for at, r := range content {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			}
		case r == '"' || r == '\'':
			quote = r
		case r == '{' || r == '[':
			return "", nil, false
		case r == ':' && (at+1 == len(content) || content[at+1] == ' ' || content[at+1] == '\t'):
			return unquoted(strings.TrimSpace(string(content[:at]))), content[at+1:], true
		}
	}
	return "", nil, false
}

// uncommented is a line without its comment: from a hash that begins the line
// or follows a blank, outside quotes.
func uncommented(runes []rune) []rune {
	var quote rune
	for at, r := range runes {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			}
		case r == '"' || r == '\'':
			quote = r
		case r == '#' && (at == 0 || runes[at-1] == ' ' || runes[at-1] == '\t'):
			return runes[:at]
		}
	}
	return runes
}

// afterDash is the content of a list item after its dash and the blanks after
// it, and whether there was one. A dash with nothing after it yet is not one.
func afterDash(content []rune) ([]rune, bool) {
	if len(content) < 2 || content[0] != '-' || (content[1] != ' ' && content[1] != '\t') {
		return content, false
	}
	rest := content[1:]
	for len(rest) > 0 && (rest[0] == ' ' || rest[0] == '\t') {
		rest = rest[1:]
	}
	return rest, true
}

// unquoted is a key without the quotes around it.
func unquoted(key string) string {
	if len(key) >= 2 && (key[0] == '"' || key[0] == '\'') && key[len(key)-1] == key[0] {
		return key[1 : len(key)-1]
	}
	return key
}

func leadingSpaces(runes []rune) int {
	count := 0
	for count < len(runes) && runes[count] == ' ' {
		count++
	}
	return count
}

func isBlank(runes []rune) bool {
	for _, r := range runes {
		if r != ' ' && r != '\t' {
			return false
		}
	}
	return true
}

// isWordRune is a character a key, a value or a name offered here can hold.
func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.'
}

// isWord says whether the text is one word, the empty one included.
func isWord(text string) bool {
	for _, r := range text {
		if !isWordRune(r) {
			return false
		}
	}
	return true
}

// wordLength is how many characters of a word the runes begin with.
func wordLength(runes []rune) int {
	length := 0
	for length < len(runes) && isWordRune(runes[length]) {
		length++
	}
	return length
}
