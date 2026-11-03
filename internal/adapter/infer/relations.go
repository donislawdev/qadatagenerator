package infer

import (
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// Coincidences is how many ways an agreement between two columns has to be
// able to have come out differently before it counts as a relation.
//
// # One bar, computed per relation, and the same bar the schemes answer to
//
// A numbering scheme is believed when a coincidence would take about one in a
// hundred million, which is what eight values accepted by the weakest check
// digit come to. A relation earns the same bar and gets there by a different
// count: how many ways the values of the other column could have been dealt
// out across these records, of which the observed agreement is one.
//
// 🔴 The number of records is NOT that count, and using it would be the
// mistake this whole file exists to avoid. A column agreeing with a gender
// split of seven to one has eight ways of lining up, so it agrees by accident
// one time in four. The same eight records split four and four have seventy
// ways. What decides is the balance, not the size, and a rule stated in
// records would announce a relation on the thinnest evidence in the file.
const Coincidences = 1e8

// RelationKind is the closed set of relations this build reads off values.
type RelationKind string

const (
	// GenderRelation is a column whose values stand one for one against the
	// gender a numbering scheme encodes.
	GenderRelation RelationKind = "gender"
	// DateRelation is a column holding the date a numbering scheme encodes.
	DateRelation RelationKind = "date"
)

// Relation is one agreement between two columns and the evidence for it.
type Relation struct {
	// Field is the column carrying the numbering scheme, and With is the
	// column it agrees with. The direction matters: the identifier is what a
	// mutator rebuilds, and the neighbour is what it then contradicts.
	Field string
	With  string
	Kind  RelationKind
	Basis string
}

// dateLayouts are the ways a date is written that this build recognises.
//
// A closed list rather than a guess. Two layouts that both parse the whole
// column and agree on every record are not a problem, because the answer is
// the same either way. Two that disagree end the same way as two numbering
// schemes accepting one column, which is a refusal rather than a choice.
var dateLayouts = []string{
	"2006-01-02", "2006/01/02", "02.01.2006", "02-01-2006", "02/01/2006", "20060102",
}

// column is one column of the table and what was sampled of it, looked up once
// for the file rather than once for every pair the column is part of.
type column struct {
	name    string
	sample  dataset.Sample
	sampled bool
}

// columnsOf lines the samples up with the columns of the table.
func columnsOf(names []string, byName map[string]dataset.Sample) []column {
	columns := make([]column, len(names))
	for at, name := range names {
		sample, sampled := byName[name]
		columns[at] = column{name: name, sample: sample, sampled: sampled}
	}
	return columns
}

// indexedRecords is how many records at the top of a file the index of
// neighbours reads.
//
// # Why eight
//
// Two columns that both hold the first eight records share exactly those as
// their first eight, so whatever ends a hypothesis there ends it for the pair
// without a walk. Eight is enough for a column of two values to match the
// genders of eight records by chance one time in 128, and few enough that a
// blank near the top of a column is the exception rather than the rule.
const indexedRecords = 8

// neighbourhood is the columns of a table as the pairing reads them, with an
// index of which of them can agree with a column of identifiers at all.
//
// 🔴 The index answers from the first records only, and it never leaves out a
// neighbour the walk would keep. For two columns that both hold the first
// indexedRecords records, the first steps of the walk are known in advance:
// the gender lasts through them only if the values of the neighbour repeat
// exactly where the genders repeat, and the date lasts through the first only
// if the first value is that date in one of the layouts. A column with a blank
// among its first records is outside the index and is walked by everybody, and
// an identifier column with such a blank walks everybody. This is the sorting
// of columns once that B-344 asked for, made exact. Sorting a column by what all
// of its values look like would have dropped a gender column holding a third
// value on the records that carry no identifier, and such a column agrees.
type neighbourhood struct {
	columns []column
	// everyone is every column, for an identifier column the index cannot
	// speak for, and ragged is every column the index holds nothing about.
	everyone []int
	ragged   []int
	// byPattern holds the columns that hold the first records, by which of
	// their first values repeat which, and byFirst by their first value.
	byPattern map[string][]int
	byFirst   map[string][]int
}

// neighbourhoodOf reads every column of the table once, for all the columns
// of identifiers that will be paired with it.
func neighbourhoodOf(names []string, byName map[string]dataset.Sample) neighbourhood {
	around := neighbourhood{columns: columnsOf(names, byName),
		byPattern: map[string][]int{}, byFirst: map[string][]int{}}
	for at, other := range around.columns {
		around.everyone = append(around.everyone, at)
		switch {
		case !other.sampled:
		case !fromTheTop(other.sample.Rows):
			around.ragged = append(around.ragged, at)
		default:
			pattern := patternOf(other.sample.Values[:indexedRecords])
			around.byPattern[pattern] = append(around.byPattern[pattern], at)
			first := other.sample.Values[0]
			around.byFirst[first] = append(around.byFirst[first], at)
		}
	}
	return around
}

// candidatesFor names, in the order of the table, the columns a column of
// identifiers has to be walked against. Every other column is one the walk
// would have ended on within the first records.
func (n neighbourhood) candidatesFor(ids encoded, gendered, dated bool) []int {
	if !fromTheTop(ids.rows) {
		return n.everyone
	}
	lists := [][]int{n.ragged}
	if gendered && !slices.Contains(ids.genders[:indexedRecords], provider.Unknown) {
		lists = append(lists, n.byPattern[patternOf(ids.genders[:indexedRecords])])
	}
	if dated && ids.dated[0] {
		for _, layout := range dateLayouts {
			lists = append(lists, n.byFirst[asWritten(layout, ids.dates[0])])
		}
	}
	total := 0
	for _, list := range lists {
		total += len(list)
	}
	candidates := make([]int, 0, total)
	for _, list := range lists {
		candidates = append(candidates, list...)
	}
	slices.Sort(candidates)
	return slices.Compact(candidates)
}

// fromTheTop reports whether a column holds every one of the first
// indexedRecords records. The records of a sample only go up, so the first
// standing at one and the last of them at indexedRecords leave no room for a
// gap between.
func fromTheTop(rows []int) bool {
	return len(rows) >= indexedRecords && rows[0] == 1 && rows[indexedRecords-1] == indexedRecords
}

// patternOf writes down which of the values repeat which: for every position,
// the position where its value was first seen. Two columns of values, or a
// column of genders and one of values, have the same pattern exactly when the
// one stands for the other one for one.
func patternOf[T comparable](values []T) string {
	pattern := make([]byte, len(values))
	for at := range values {
		pattern[at] = byte(at)
		for before := range at {
			if values[before] == values[at] {
				pattern[at] = pattern[before]
				break
			}
		}
	}
	return string(pattern)
}

// encoded is what the identifiers of one column carry, read once for the
// column.
//
// 🔴 Reading it once is most of B-344. It used to be read again for every other
// column of the file, and every date written out again in six layouts, so one
// pair cost as much as decoding the whole sample and the time grew with the
// square of the width at that price. A value that carries no gender holds
// Unknown here, and one whose date cannot be read is not dated, because either
// ends the relation as soon as a record both columns hold lands on it.
type encoded struct {
	rows    []int
	genders []provider.Gender
	dates   []provider.Date
	dated   []bool
}

// encodedIn reads every identifier of a column once. A scheme that encodes no
// gender, or no date, is passed as nil for it.
func encodedIn(carrier dataset.Sample, readsGender provider.GenderEncoder,
	readsDate provider.DateEncoder) encoded {
	ids := encoded{
		rows:    carrier.Rows,
		genders: make([]provider.Gender, len(carrier.Values)),
		dates:   make([]provider.Date, len(carrier.Values)),
		dated:   make([]bool, len(carrier.Values)),
	}
	for at, value := range carrier.Values {
		ids.genders[at] = provider.Unknown
		if readsGender != nil {
			if gender, ok := readsGender.EncodedGender(value); ok {
				ids.genders[at] = gender
			}
		}
		if readsDate != nil {
			ids.dates[at], ids.dated[at] = readsDate.EncodedDate(value)
		}
	}
	return ids
}

// relationsOf works out which columns a column of identifiers agrees with.
func relationsOf(field string, scheme provider.Provider, around neighbourhood,
	byName map[string]dataset.Sample) []Relation {
	carrier, has := byName[field]
	if !has {
		return nil
	}
	readsGender, gendered := scheme.(provider.GenderEncoder)
	readsDate, dated := scheme.(provider.DateEncoder)
	if !gendered && !dated {
		return nil
	}
	ids := encodedIn(carrier, readsGender, readsDate)
	var found []Relation
	var walk pairing
	for _, at := range around.candidatesFor(ids, gendered, dated) {
		other := &around.columns[at]
		if other.name == field || !other.sampled {
			continue
		}
		shared := walk.over(ids, other.sample, gendered, dated)
		if relation, ok := walk.genderAgreement(field, other.name, shared); ok {
			found = append(found, relation)
		}
		if relation, ok := walk.dateAgreement(field, other.name, ids, other.sample, shared); ok {
			found = append(found, relation)
		}
	}
	return found
}

// pairing walks the records a column of identifiers shares with one neighbour
// and keeps what they have not yet ruled out: that the neighbour stands for the
// gender, and that it holds the date in each layout.
//
// One walk serves every neighbour of a column of identifiers and is emptied
// before each, so a pair allocates nothing. It stops at the record that rules
// out the last hypothesis, which for two columns with nothing to do with each
// other is the first or the second record they share, instead of lining every
// record up first and asking afterwards.
type pairing struct {
	gender bool
	// layouts has bit i set while the neighbour can still be dateLayouts[i].
	layouts uint
	// genders, stands and counts say which value of the neighbour each gender
	// stands for and on how many records, for as long as the gender holds.
	genders []provider.Gender
	stands  []string
	counts  []int
}

// over walks the records both columns hold, in record order, and returns how
// many there are. The count is whole only when a hypothesis lasted to the end,
// which is the only time it is read.
//
// 🔴 This is the whole reason a sample carries its record numbers. Empty cells
// are dropped when a column is sampled, so the third value of one column and
// the third of another come from the same record only when neither has a gap
// above it. Reading them by position would compare one person's identifier
// against another person's gender and call the disagreement evidence.
func (p *pairing) over(ids encoded, neighbour dataset.Sample, gendered, dated bool) int {
	p.gender, p.layouts = gendered, 0
	if dated {
		p.layouts = 1<<len(dateLayouts) - 1
	}
	p.genders, p.stands, p.counts = p.genders[:0], p.stands[:0], p.counts[:0]
	shared := 0
	at, to := 0, 0
	for at < len(ids.rows) && to < len(neighbour.Rows) && (p.gender || p.layouts != 0) {
		switch {
		case ids.rows[at] < neighbour.Rows[to]:
			at++
		case ids.rows[at] > neighbour.Rows[to]:
			to++
		default:
			value := neighbour.Values[to]
			if p.gender {
				p.gender = p.standsFor(ids.genders[at], value)
			}
			if !ids.dated[at] {
				p.layouts = 0
			}
			for layout := range dateLayouts {
				if p.layouts&(1<<layout) != 0 && !writes(dateLayouts[layout], ids.dates[at], value) {
					p.layouts &^= 1 << layout
				}
			}
			shared++
			at++
			to++
		}
	}
	return shared
}

// standsFor keeps the gender through one more record. Every gender has to
// stand for one value of the neighbour and every value for one gender, so a
// gender met with a second value, or a value already standing for the other
// gender, ends it.
func (p *pairing) standsFor(gender provider.Gender, value string) bool {
	if gender == provider.Unknown {
		return false
	}
	for at := range p.genders {
		sameGender, sameValue := p.genders[at] == gender, p.stands[at] == value
		if sameGender != sameValue {
			return false
		}
		if sameGender {
			p.counts[at]++
			return true
		}
	}
	p.genders = append(p.genders, gender)
	p.stands = append(p.stands, value)
	p.counts = append(p.counts, 1)
	return true
}

// genderAgreement answers, once the walk is over, whether a column stands one
// for one against the gender the identifiers encode.
//
// # Why the vocabulary of the column never comes into it
//
// What a gender column looks like in somebody else's file cannot be known: it
// is M and F, or K and M, or 0 and 1, or two words in a language this build
// does not read. So the question asked here is not what the values mean but
// whether they map one for one onto what the scheme encodes, every record the
// same way. A column that does is tied to the identifier whatever its values
// say, and one that does not is not, however familiar its values look.
//
// 🔴 Both genders have to appear, and that is not a formality. With one gender
// in the sample every column holding a single repeated value agrees perfectly,
// and the agreement says nothing at all.
func (p *pairing) genderAgreement(field, other string, shared int) (Relation, bool) {
	if !p.gender || shared == 0 {
		return Relation{}, false
	}
	// 🔴 There is no separate check that both genders appeared, and its absence
	// is the finding rather than an oversight. One was written, and a mutation
	// removing it changed nothing: a sample carrying one gender is one group,
	// one group can be dealt out exactly one way, and one way is far under the
	// bar. The degenerate case where every column of a single repeated value
	// agrees perfectly is therefore refused by the count itself, and
	// TestASplitUnderTheBarIsRefusedHoweverManyRecordsAgree holds the count to
	// that. A condition nobody can break is a condition nobody can be sure
	// still works.
	ways := arrangements(ascending(p.counts))
	if ways < Coincidences {
		return Relation{}, false
	}
	return genderRelation(field, other, shared, ways), true
}

// genderRelation is the relation a gender agreement proves, in the words the
// manifest carries.
func genderRelation(field, other string, shared int, ways float64) Relation {
	return Relation{Field: field, With: other, Kind: GenderRelation,
		Basis: "every one of " + counted(shared, "record") + " holding both columns has " +
			"the gender encoded in " + field + " standing for the same value of " + other +
			", both genders appear, and the values could have been dealt out " +
			roughly(ways) + " ways"}
}

// dateAgreement answers, once the walk is over, whether a column holds the date
// the identifiers encode.
//
// Two layouts both lasting to the end are no conflict, because they agree on
// every record and the answer is the same either way.
func (p *pairing) dateAgreement(field, other string, ids encoded, neighbour dataset.Sample,
	shared int) (Relation, bool) {
	if p.layouts == 0 || shared == 0 {
		return Relation{}, false
	}
	ways := arrangements(sizesOf(tally(sharedDates(ids, neighbour))))
	if ways < Coincidences {
		return Relation{}, false
	}
	return dateRelation(field, other, shared, ways), true
}

// dateRelation is the relation a date agreement proves, in the words the
// manifest carries.
func dateRelation(field, other string, shared int, ways float64) Relation {
	return Relation{Field: field, With: other, Kind: DateRelation,
		Basis: "every one of " + counted(shared, "record") + " holding both columns has " +
			"the date encoded in " + field + " written in " + other +
			", and the dates could have been dealt out " + roughly(ways) + " ways"}
}

// sharedDates are the dates encoded on the records both columns hold.
//
// It is a second walk over the same records, taken only by a neighbour that
// held the date on every one of them, so it is paid for by relations rather
// than by pairs.
func sharedDates(ids encoded, neighbour dataset.Sample) []provider.Date {
	var dates []provider.Date
	at, to := 0, 0
	for at < len(ids.rows) && to < len(neighbour.Rows) {
		switch {
		case ids.rows[at] < neighbour.Rows[to]:
			at++
		case ids.rows[at] > neighbour.Rows[to]:
			to++
		default:
			dates = append(dates, ids.dates[at])
			at++
			to++
		}
	}
	return dates
}

// writes reports whether value is the date written in the layout, which is
// what asWritten(layout, date) == value answers, without writing the date out.
//
// It is asked for every layout still standing on every record two columns
// share, so it is the innermost step of the pairing and allocates nothing. The
// layout is read the way asWritten replaces it, a year, a month or a day where
// one is named and the character itself everywhere else. A date with a part
// wider than its place is written out in full, because only then can the
// written date be longer than the layout.
func writes(layout string, date provider.Date, value string) bool {
	if !fits(date) {
		return asWritten(layout, date) == value
	}
	if len(value) != len(layout) {
		return false
	}
	for at := 0; at < len(layout); {
		switch {
		case strings.HasPrefix(layout[at:], "2006"):
			if !digits(value[at:at+4], date.Year) {
				return false
			}
			at += 4
		case strings.HasPrefix(layout[at:], "01"):
			if !digits(value[at:at+2], date.Month) {
				return false
			}
			at += 2
		case strings.HasPrefix(layout[at:], "02"):
			if !digits(value[at:at+2], date.Day) {
				return false
			}
			at += 2
		default:
			if value[at] != layout[at] {
				return false
			}
			at++
		}
	}
	return true
}

// fits reports whether every part of the date takes exactly the width its
// place in a layout gives it, so that the written date is as long as the
// layout.
func fits(date provider.Date) bool {
	return date.Year >= 0 && date.Year <= 9999 && date.Month >= 0 && date.Month <= 99 &&
		date.Day >= 0 && date.Day <= 99
}

// digits reports whether written is the number in exactly that many decimal
// digits, with zeros in front.
func digits(written string, number int) bool {
	for at := len(written) - 1; at >= 0; at-- {
		if written[at] != byte('0'+number%10) {
			return false
		}
		number /= 10
	}
	return number == 0
}

// asWritten renders a date in a layout, without a time package, because the
// layouts here are fixed widths of digits and a calendar is not needed to
// write one out.
func asWritten(layout string, date provider.Date) string {
	written := strings.NewReplacer(
		"2006", pad(date.Year, 4),
		"01", pad(date.Month, 2),
		"02", pad(date.Day, 2),
	)
	return written.Replace(layout)
}

func pad(number, width int) string {
	written := strconv.Itoa(number)
	for len(written) < width {
		written = "0" + written
	}
	return written
}

// tally counts how many records carry each distinct date.
func tally(dates []provider.Date) map[provider.Date]int {
	counted := map[provider.Date]int{}
	for _, date := range dates {
		counted[date]++
	}
	return counted
}

func sizesOf[K comparable](counts map[K]int) []int {
	sizes := make([]int, 0, len(counts))
	for _, count := range counts {
		sizes = append(sizes, count)
	}
	sort.Ints(sizes)
	return sizes
}

// ascending is a copy of the counts in order, from the smallest, the order
// sizesOf gives.
func ascending(counts []int) []int {
	sizes := slices.Clone(counts)
	sort.Ints(sizes)
	return sizes
}

// arrangements is how many ways records carrying these groups could have been
// dealt out, which is the multinomial coefficient of the group sizes.
//
// It stops counting once the answer is past the bar, because the numbers grow
// beyond what a float can hold and the only question asked of them is whether
// they are large enough.
//
// 🔴 Stopping early is honest only while no partial product is larger than the
// final one, and counting a group up to its own size breaks that. The last
// group of n records can be chosen one way, but counted a record at a time it
// passes n choose n/2 on the way there, so thirty records of one gender reached
// the bar halfway and were announced as a relation that can be dealt out in a
// single way (B-355). Each group is counted up to the smaller of its size and
// what is left beside it instead. That is the same binomial, and it climbs the
// whole way.
func arrangements(sizes []int) float64 {
	left := 0
	for _, size := range sizes {
		left += size
	}
	ways := 1.0
	for _, size := range sizes {
		for taken := 1; taken <= min(size, left-size); taken++ {
			ways = ways * float64(left-taken+1) / float64(taken)
			if ways >= Coincidences {
				return Coincidences
			}
		}
		left -= size
	}
	return ways
}

// roughly writes a count of arrangements the way a sentence can carry it.
func roughly(ways float64) string {
	if ways >= Coincidences {
		return "more than a hundred million"
	}
	return strconv.FormatFloat(ways, 'f', 0, 64)
}

// rulesOf turns the relations found into what the engine reads.
//
// One rule per identifier column, listing every column it agrees with, because
// that is the shape a recipe writes by hand and the engine already knows.
func rulesOf(found []Relation) []recipe.Rule {
	byField := map[string][]string{}
	var order []string
	for _, relation := range found {
		if _, has := byField[relation.Field]; !has {
			order = append(order, relation.Field)
		}
		byField[relation.Field] = append(byField[relation.Field], relation.With)
	}
	var rules []recipe.Rule
	for _, field := range order {
		with := byField[field]
		sort.Strings(with)
		rules = append(rules, recipe.Rule{Field: field, Kind: recipe.Matches, With: with})
	}
	return rules
}
