package infer

import (
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

// aligned is the values of two columns for the records where both are present.
type aligned struct {
	rows  []int
	left  []string
	right []string
}

// alignBy walks two samples in record order and keeps the records both hold.
//
// 🔴 This is the whole reason a sample carries its record numbers. Empty cells
// are dropped when a column is sampled, so the third value of one column and
// the third of another come from the same record only when neither has a gap
// above it. Reading them by position would compare one person's identifier
// against another person's gender and call the disagreement evidence.
func alignBy(left, right dataset.Sample) aligned {
	var both aligned
	at, to := 0, 0
	for at < len(left.Rows) && to < len(right.Rows) {
		switch {
		case left.Rows[at] < right.Rows[to]:
			at++
		case left.Rows[at] > right.Rows[to]:
			to++
		default:
			both.rows = append(both.rows, left.Rows[at])
			both.left = append(both.left, left.Values[at])
			both.right = append(both.right, right.Values[to])
			at++
			to++
		}
	}
	return both
}

// relationsOf works out which columns a column of identifiers agrees with.
func relationsOf(field string, scheme provider.Provider, columns []string,
	byName map[string]dataset.Sample) []Relation {
	var found []Relation
	carrier, has := byName[field]
	if !has {
		return nil
	}
	for _, other := range columns {
		if other == field {
			continue
		}
		neighbour, has := byName[other]
		if !has {
			continue
		}
		both := alignBy(carrier, neighbour)
		if reads, ok := scheme.(provider.GenderEncoder); ok {
			if relation, ok := genderAgreement(field, other, reads, both); ok {
				found = append(found, relation)
			}
		}
		if reads, ok := scheme.(provider.DateEncoder); ok {
			if relation, ok := dateAgreement(field, other, reads, both); ok {
				found = append(found, relation)
			}
		}
	}
	return found
}

// genderAgreement answers whether a column stands one for one against the
// gender the identifiers encode.
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
func genderAgreement(field, other string, reads provider.GenderEncoder,
	both aligned) (Relation, bool) {
	if len(both.rows) == 0 {
		return Relation{}, false
	}
	means := map[provider.Gender]string{}
	saidBy := map[string]provider.Gender{}
	counts := map[provider.Gender]int{}
	for at, value := range both.left {
		encoded, ok := reads.EncodedGender(value)
		if !ok || encoded == provider.Unknown {
			return Relation{}, false
		}
		stands := both.right[at]
		if seen, has := means[encoded]; has && seen != stands {
			return Relation{}, false
		}
		if seen, has := saidBy[stands]; has && seen != encoded {
			return Relation{}, false
		}
		means[encoded] = stands
		saidBy[stands] = encoded
		counts[encoded]++
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
	ways := arrangements(sizesOf(counts))
	if ways < Coincidences {
		return Relation{}, false
	}
	return Relation{Field: field, With: other, Kind: GenderRelation,
		Basis: "every one of " + counted(len(both.rows), "record") + " holding both columns has " +
			"the gender encoded in " + field + " standing for the same value of " + other +
			", both genders appear, and the values could have been dealt out " +
			roughly(ways) + " ways"}, true
}

// dateAgreement answers whether a column holds the date the identifiers encode.
func dateAgreement(field, other string, reads provider.DateEncoder,
	both aligned) (Relation, bool) {
	if len(both.rows) == 0 {
		return Relation{}, false
	}
	encoded := make([]provider.Date, len(both.left))
	for at, value := range both.left {
		read, ok := reads.EncodedDate(value)
		if !ok {
			return Relation{}, false
		}
		encoded[at] = read
	}
	agreeing := 0
	for _, layout := range dateLayouts {
		if writtenAs(layout, encoded, both.right) {
			agreeing++
		}
	}
	if agreeing == 0 {
		return Relation{}, false
	}
	ways := arrangements(sizesOf(tally(encoded)))
	if ways < Coincidences {
		return Relation{}, false
	}
	return Relation{Field: field, With: other, Kind: DateRelation,
		Basis: "every one of " + counted(len(both.rows), "record") + " holding both columns has " +
			"the date encoded in " + field + " written in " + other +
			", and the dates could have been dealt out " + roughly(ways) + " ways"}, true
}

// writtenAs reports whether every value is the date beside it in one layout.
func writtenAs(layout string, encoded []provider.Date, written []string) bool {
	for at, value := range written {
		if asWritten(layout, encoded[at]) != value {
			return false
		}
	}
	return true
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
