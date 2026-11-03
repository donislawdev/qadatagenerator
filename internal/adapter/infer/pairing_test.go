package infer

import (
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
)

// The schemes here read a gender and a date out of values written as a letter
// and three numbers, F|1990|5|17. A real scheme never hands the pairing a value
// with no gender in it, a date it cannot read or a year of five digits, and
// those are exactly the records that end a relation or take the long way
// through writes.

type plainScheme struct{}

func (plainScheme) ID() string                                { return "plain" }
func (plainScheme) Capabilities() []capability.Name           { return nil }
func (plainScheme) Generate(provider.Request) (string, error) { return "", nil }
func (plainScheme) Validate(string) provider.Verdict          { return provider.Verdict{Valid: true} }

type genderScheme struct{ plainScheme }

func (genderScheme) EncodedGender(value string) (provider.Gender, bool) { return genderIn(value) }
func (genderScheme) WithGender(string, provider.Gender) (string, error) { return "", nil }

type dateScheme struct{ plainScheme }

func (dateScheme) EncodedDate(value string) (provider.Date, bool) { return dateIn(value) }
func (dateScheme) WithDate(string, provider.Date) (string, error) { return "", nil }

type bothScheme struct{ plainScheme }

func (bothScheme) EncodedGender(value string) (provider.Gender, bool) { return genderIn(value) }
func (bothScheme) WithGender(string, provider.Gender) (string, error) { return "", nil }
func (bothScheme) EncodedDate(value string) (provider.Date, bool)     { return dateIn(value) }
func (bothScheme) WithDate(string, provider.Date) (string, error)     { return "", nil }

// genderIn reads F and M, says Unknown for U, and cannot read anything else.
func genderIn(value string) (provider.Gender, bool) {
	switch {
	case strings.HasPrefix(value, "F"):
		return provider.Female, true
	case strings.HasPrefix(value, "M"):
		return provider.Male, true
	case strings.HasPrefix(value, "U"):
		return provider.Unknown, true
	}
	return "", false
}

func dateIn(value string) (provider.Date, bool) {
	parts := strings.Split(value, "|")
	if len(parts) != 4 {
		return provider.Date{}, false
	}
	var numbers [3]int
	for at := range numbers {
		number, err := strconv.Atoi(parts[at+1])
		if err != nil {
			return provider.Date{}, false
		}
		numbers[at] = number
	}
	return provider.Date{Year: numbers[0], Month: numbers[1], Day: numbers[2]}, true
}

// 🔴 writes answers what writing the date out and comparing would answer.
//
// It is asked instead of asWritten on every record two columns share, so a
// difference between the two is a relation found or missed. Every layout is put
// to dates that fit it and dates that do not, and to values that are the date,
// the date one character longer or shorter, the date with one character
// changed, and the date in the other layouts. Changing the lowest bit of a
// character turns a slash into a full stop and a hyphen into a comma, so the
// changed values include the other layouts' separators.
func TestWritesAnswersWhatWritingTheDateOutWould(t *testing.T) {
	matched := 0
	for _, year := range []int{0, 7, 99, 1999, 2024, 9999, 10000, 12345, -1, -42} {
		for _, month := range []int{0, 1, 9, 10, 12, 99, 100, -1} {
			for _, day := range []int{0, 1, 9, 10, 31, 99, 100, -5} {
				date := provider.Date{Year: year, Month: month, Day: day}
				written := make([]string, len(dateLayouts))
				var values []string
				for at, layout := range dateLayouts {
					written[at] = asWritten(layout, date)
					values = append(values, near(written[at])...)
				}
				for at, layout := range dateLayouts {
					for _, value := range values {
						want := written[at] == value
						if writes(layout, date, value) != want {
							t.Errorf("writes(%q, %v, %q) said %v, and writing the date out says %v",
								layout, date, value, !want, want)
						}
						if want {
							matched++
						}
					}
				}
			}
		}
	}
	if matched == 0 {
		t.Fatal("no value was ever the date, so only refusals were checked")
	}
}

// near is a written date and the values one step away from it.
func near(written string) []string {
	values := []string{written, written + "0", written[1:], "0" + written}
	for at := range written {
		values = append(values, written[:at]+string(written[at]^1)+written[at+1:])
	}
	return values
}

// 🔴 The pairing answers what comparing every record the plain way answers.
//
// The walk keeps hypotheses and stops early, reads each identifier once and
// compares a date without writing it out, and each of those is a way to answer
// differently from the rule. referenceRelations is the rule written plainly,
// the way the pairing was written before B-344, and thousands of tables are put
// to both: gaps in either column, a value on a record the identifiers leave
// empty, identifiers with no gender or no readable date, one gender, a few
// dates repeated, years of five digits, a neighbour wrong on one record, and
// schemes that read a gender only, a date only, or neither.
func TestThePairingAnswersWhatComparingEveryRecordWouldAnswer(t *testing.T) {
	draw := rand.New(rand.NewPCG(344, 355))
	schemes := []provider.Provider{bothScheme{}, bothScheme{}, genderScheme{}, dateScheme{},
		plainScheme{}}
	found := map[RelationKind]int{}
	pruned := 0
	for trial := range 3000 {
		scheme := schemes[draw.IntN(len(schemes))]
		names, byName := drawnTable(draw)
		around := neighbourhoodOf(names, byName)
		got := relationsOf("id", scheme, around, byName)
		want := referenceRelations("id", scheme, names, byName)
		if !slices.Equal(got, want) {
			t.Fatalf("table %d: the pairing found %v and the plain comparison %v", trial, got, want)
		}
		for _, relation := range got {
			found[relation.Kind]++
		}
		readsGender, gendered := scheme.(provider.GenderEncoder)
		readsDate, dated := scheme.(provider.DateEncoder)
		if gendered || dated {
			ids := encodedIn(byName["id"], readsGender, readsDate)
			if len(around.candidatesFor(ids, gendered, dated)) < len(names) {
				pruned++
			}
		}
	}
	// Agreeing on refusals alone would prove little, so the tables have to give
	// both kinds of relation often enough to be worth agreeing on, and the index
	// of neighbours has to have left columns out often enough for the agreement
	// to be about it too.
	t.Logf("relations found in 3000 tables: %d of gender, %d of date, and %d tables "+
		"walked fewer columns than they have", found[GenderRelation], found[DateRelation], pruned)
	if pruned < 300 {
		t.Errorf("the index of neighbours left a column out in only %d tables of 3000", pruned)
	}
	for _, kind := range []RelationKind{GenderRelation, DateRelation} {
		if found[kind] < 100 {
			t.Errorf("only %d %s relations were found in 3000 tables, so the comparison "+
				"rests on refusals", found[kind], kind)
		}
	}
}

// drawnTable draws one column of identifiers and the neighbours this test puts
// against it.
func drawnTable(draw *rand.Rand) ([]string, map[string]dataset.Sample) {
	records := draw.IntN(80)
	gap := []float64{0, 0, 0.1, 0.4}[draw.IntN(4)]
	pool := drawnDates(draw)
	oneGender := draw.IntN(5) == 0
	noisy := draw.IntN(3) == 0

	genders := make([]byte, records+1)
	dates := make([]provider.Date, records+1)
	ids := dataset.Sample{Field: "id"}
	for record := 1; record <= records; record++ {
		genders[record] = "FM"[draw.IntN(2)]
		if oneGender {
			genders[record] = 'F'
		}
		if noisy && draw.IntN(40) == 0 {
			genders[record] = "UX"[draw.IntN(2)]
		}
		dates[record] = pool[draw.IntN(len(pool))]
		value := string(genders[record]) + "|" + strconv.Itoa(dates[record].Year) + "|" +
			strconv.Itoa(dates[record].Month) + "|" + strconv.Itoa(dates[record].Day)
		if noisy && draw.IntN(40) == 0 {
			// The neighbours write the zero date beside it, which is what a
			// walk would compare against if it forgot that this date was
			// never read.
			value = string(genders[record]) + "|no date"
			dates[record] = provider.Date{}
		}
		if draw.Float64() >= gap {
			ids.Rows = append(ids.Rows, record)
			ids.Values = append(ids.Values, value)
		}
	}
	held := map[int]bool{}
	for _, record := range ids.Rows {
		held[record] = true
	}

	words := [][2]string{{"a", "b"}, {"K", "M"}, {"1", "2"}}[draw.IntN(3)]
	layout := dateLayouts[draw.IntN(len(dateLayouts))]
	// A list rather than a map, because the draws are taken in this order and a
	// table that changed from one run to the next could not be drawn again.
	neighbours := []struct {
		name    string
		valueAt func(record int) string
	}{
		{"sex", func(record int) string {
			if !held[record] && draw.IntN(4) == 0 {
				return "z"
			}
			if genders[record] == 'M' {
				return words[1]
			}
			return words[0]
		}},
		{"born", func(record int) string {
			if !held[record] && draw.IntN(4) == 0 {
				return "never"
			}
			return asWritten(layout, dates[record])
		}},
		{"mixed", func(record int) string {
			return asWritten(dateLayouts[draw.IntN(len(dateLayouts))], dates[record])
		}},
		{"same", func(int) string { return "c" }},
		{"noise", func(int) string { return strconv.Itoa(draw.IntN(3)) }},
	}
	names := []string{"id", "sex", "born", "mixed", "same", "noise", "id2", "unsampled"}
	byName := map[string]dataset.Sample{"id": ids, "id2": ids}
	for _, neighbour := range neighbours {
		name, valueAt := neighbour.name, neighbour.valueAt
		sample := dataset.Sample{Field: name}
		// One record written wrong, drawn apart from the noise in the
		// identifiers, so that a table can carry either without the other.
		wrong := 0
		if draw.IntN(3) == 0 {
			wrong = 1 + draw.IntN(max(records, 1))
		}
		for record := 1; record <= records; record++ {
			if draw.Float64() < gap {
				continue
			}
			value := valueAt(record)
			if record == wrong {
				value += "x"
			}
			sample.Rows = append(sample.Rows, record)
			sample.Values = append(sample.Values, value)
		}
		byName[name] = sample
	}
	return names, byName
}

// drawnDates is the dates one table draws from: many, so that dates can make a
// relation, three, so that they cannot, or a few that do not fit a layout.
func drawnDates(draw *rand.Rand) []provider.Date {
	switch draw.IntN(4) {
	case 0:
		return []provider.Date{{Year: 1990, Month: 1, Day: 2}, {Year: 1991, Month: 3, Day: 4},
			{Year: 2001, Month: 12, Day: 31}}
	case 1:
		var odd []provider.Date
		for _, year := range []int{12345, -7, 2000, 99999, 1999, 10000} {
			for month := 1; month <= 3; month++ {
				odd = append(odd, provider.Date{Year: year, Month: month, Day: 100 * (month % 2)})
			}
		}
		return odd
	}
	many := make([]provider.Date, 200)
	for at := range many {
		many[at] = provider.Date{Year: 1900 + draw.IntN(121), Month: 1 + draw.IntN(12),
			Day: 1 + draw.IntN(28)}
	}
	return many
}

// referenceRelations is the pairing as it was written before B-344: every pair
// lines the two columns up, reads every identifier again, writes every date out
// in every layout and only then decides. It is slow and plain, which is what it
// is for. The records are matched through a map of the neighbour rather than
// by walking both columns, so it shares no step with what it checks.
func referenceRelations(field string, scheme provider.Provider, columns []string,
	byName map[string]dataset.Sample) []Relation {
	carrier, has := byName[field]
	if !has {
		return nil
	}
	var found []Relation
	for _, other := range columns {
		if other == field {
			continue
		}
		neighbour, has := byName[other]
		if !has {
			continue
		}
		byRecord := map[int]string{}
		for at, record := range neighbour.Rows {
			byRecord[record] = neighbour.Values[at]
		}
		var left, right []string
		for at, record := range carrier.Rows {
			if value, has := byRecord[record]; has {
				left = append(left, carrier.Values[at])
				right = append(right, value)
			}
		}
		if reads, ok := scheme.(provider.GenderEncoder); ok {
			if relation, ok := referenceGender(field, other, reads, left, right); ok {
				found = append(found, relation)
			}
		}
		if reads, ok := scheme.(provider.DateEncoder); ok {
			if relation, ok := referenceDate(field, other, reads, left, right); ok {
				found = append(found, relation)
			}
		}
	}
	return found
}

func referenceGender(field, other string, reads provider.GenderEncoder,
	left, right []string) (Relation, bool) {
	if len(left) == 0 {
		return Relation{}, false
	}
	means := map[provider.Gender]string{}
	saidBy := map[string]provider.Gender{}
	counts := map[provider.Gender]int{}
	for at, value := range left {
		encoded, ok := reads.EncodedGender(value)
		if !ok || encoded == provider.Unknown {
			return Relation{}, false
		}
		stands := right[at]
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
	ways := arrangements(sizesOf(counts))
	if ways < Coincidences {
		return Relation{}, false
	}
	return genderRelation(field, other, len(left), ways), true
}

func referenceDate(field, other string, reads provider.DateEncoder,
	left, right []string) (Relation, bool) {
	if len(left) == 0 {
		return Relation{}, false
	}
	encoded := make([]provider.Date, len(left))
	for at, value := range left {
		read, ok := reads.EncodedDate(value)
		if !ok {
			return Relation{}, false
		}
		encoded[at] = read
	}
	agreeing := 0
	for _, layout := range dateLayouts {
		all := true
		for at, value := range right {
			if asWritten(layout, encoded[at]) != value {
				all = false
				break
			}
		}
		if all {
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
	return dateRelation(field, other, len(left), ways), true
}

// 🔴 The index of neighbours leaves out exactly the columns the walk would end
// on within the first records, and keeps every other.
//
// A column standing for the genders, one holding the dates, one with a blank
// near its top and one going wrong only after the indexed records all have to
// be walked. Fifty columns of other values, a column of one value and a gender
// column wrong on its fifth record do not. The comparison test shows the
// relations come out the same either way, so this is the test that shows the
// index is doing anything at all.
func TestOnlyColumnsThatCanAgreeAreWalked(t *testing.T) {
	values := map[string][]string{}
	names := []string{"id", "sex", "born", "late", "wrongAt5", "wrongAt20", "same"}
	for record := 1; record <= 40; record++ {
		gender, word := "M", "man"
		if record%3 == 0 {
			gender, word = "F", "woman"
		}
		date := provider.Date{Year: 1950 + record, Month: 1 + record%12, Day: 1 + record%28}
		values["id"] = append(values["id"], gender+"|"+strconv.Itoa(date.Year)+"|"+
			strconv.Itoa(date.Month)+"|"+strconv.Itoa(date.Day))
		values["sex"] = append(values["sex"], word)
		values["born"] = append(values["born"], asWritten("02.01.2006", date))
		values["late"] = append(values["late"], asWritten("02.01.2006", date))
		values["wrongAt5"] = append(values["wrongAt5"], word)
		values["wrongAt20"] = append(values["wrongAt20"], word)
		values["same"] = append(values["same"], "c")
	}
	values["wrongAt5"][4] = "nobody"
	values["wrongAt20"][19] = "nobody"
	for other := range 50 {
		name := "other" + strconv.Itoa(other)
		names = append(names, name)
		for record := 1; record <= 40; record++ {
			values[name] = append(values[name], name+"-"+strconv.Itoa(record))
		}
	}
	byName := map[string]dataset.Sample{}
	for _, name := range names {
		sample := dataset.Sample{Field: name}
		for at, value := range values[name] {
			if name == "late" && at == 2 {
				continue
			}
			sample.Rows = append(sample.Rows, at+1)
			sample.Values = append(sample.Values, value)
		}
		byName[name] = sample
	}
	around := neighbourhoodOf(names, byName)

	var walked []string
	ids := encodedIn(byName["id"], bothScheme{}, bothScheme{})
	for _, at := range around.candidatesFor(ids, true, true) {
		if name := around.columns[at].name; name != "id" {
			walked = append(walked, name)
		}
	}
	if want := []string{"sex", "born", "late", "wrongAt20"}; !slices.Equal(walked, want) {
		t.Errorf("the identifiers are walked against %v, want %v", walked, want)
	}

	// Identifiers with a blank near the top share their first records with no
	// column in particular, so the index cannot speak for them.
	ragged := byName["id"]
	ragged.Rows, ragged.Values = ragged.Rows[1:], ragged.Values[1:]
	if candidates := around.candidatesFor(encodedIn(ragged, bothScheme{}, bothScheme{}),
		true, true); len(candidates) != len(names) {
		t.Errorf("identifiers missing their first record are walked against %d columns of %d",
			len(candidates), len(names))
	}
}

// BenchmarkPairingOfAWideFile is one column of identifiers against 4096 others
// of 29 values each, which is what a file at the widest this build reads keeps
// of every column, so one iteration is the work of one row of the square.
func BenchmarkPairingOfAWideFile(b *testing.B) {
	draw := rand.New(rand.NewPCG(1, 2))
	names := []string{}
	byName := map[string]dataset.Sample{}
	for at := range 4097 {
		name := "c" + strconv.Itoa(at)
		sample := dataset.Sample{Field: name}
		for record := 1; record <= 29; record++ {
			sample.Rows = append(sample.Rows, record)
			sample.Values = append(sample.Values, string("FM"[draw.IntN(2)])+"|"+
				strconv.Itoa(1900+draw.IntN(120))+"|"+strconv.Itoa(1+draw.IntN(12))+"|"+
				strconv.Itoa(1+draw.IntN(28)))
		}
		names = append(names, name)
		byName[name] = sample
	}
	around := neighbourhoodOf(names, byName)
	b.ResetTimer()
	for range b.N {
		relationsOf("c0", bothScheme{}, around, byName)
	}
}

// 🔴 A pair of columns allocates nothing, which is what B-344 bought.
//
// The walk over two columns used to line them up into three new slices and read
// every identifier again, so a wide file paid the whole sample for every pair.
// A count of allocations is a measure no machine makes noisy: one column of
// identifiers against ten neighbours and against four hundred has to allocate
// the same, because everything a pair needs was allocated for the column. The
// identifiers miss their second record, which keeps them outside the index of
// neighbours, so every pair here is walked.
func TestAPairOfColumnsAllocatesNothing(t *testing.T) {
	ids := dataset.Sample{Field: "id"}
	for record := 1; record <= 40; record++ {
		if record == 2 {
			continue
		}
		gender := "F"
		if record > 20 {
			gender = "M"
		}
		ids.Rows = append(ids.Rows, record)
		ids.Values = append(ids.Values, gender+"|"+strconv.Itoa(1950+record)+"|"+
			strconv.Itoa(1+record%12)+"|"+strconv.Itoa(1+record%28))
	}
	// Each neighbour lasts a record or two: two values taking turns, one date
	// for everybody, and noise. None of them is a relation, so nothing is kept.
	shapes := []func(record int) string{
		func(record int) string { return "ab"[record%2 : record%2+1] },
		func(int) string { return "2000-01-01" },
		func(record int) string { return "n" + strconv.Itoa(record) },
	}
	allocations := func(width int) float64 {
		names := []string{"id"}
		byName := map[string]dataset.Sample{"id": ids}
		for at := range width {
			name := "n" + strconv.Itoa(at)
			sample := dataset.Sample{Field: name}
			for record := 1; record <= 40; record++ {
				sample.Rows = append(sample.Rows, record)
				sample.Values = append(sample.Values, shapes[at%len(shapes)](record))
			}
			names = append(names, name)
			byName[name] = sample
		}
		around := neighbourhoodOf(names, byName)
		return testing.AllocsPerRun(20, func() {
			if found := relationsOf("id", bothScheme{}, around, byName); len(found) != 0 {
				t.Fatalf("columns made to agree with nothing gave %v", found)
			}
		})
	}
	narrow, wide := allocations(10), allocations(400)
	t.Logf("allocations of one column of identifiers: %v against 10 neighbours, %v against 400",
		narrow, wide)
	if wide != narrow {
		t.Errorf("one column against 10 neighbours allocates %v times and against 400 %v times, "+
			"so a pair costs an allocation again", narrow, wide)
	}
}
