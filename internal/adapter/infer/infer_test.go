package infer_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/infer"
	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/pl"
	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/us"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// schemes are every numbering scheme this build carries, taken from the registry
// rather than listed here, so that a scheme added tomorrow is offered to the
// inference by this test without anybody remembering to add it.
func schemes() []provider.Provider { return values.New("pl-PL", "infer").Schemes() }

// produced returns count values of one generator, built by the generator rather
// than written down. A correct identifier typed into a test is a correct
// identifier nobody checked.
func produced(t *testing.T, generator string, count int) []string {
	t.Helper()
	source := values.New("pl-PL", "infer-corpus")
	built := make([]string, 0, count)
	for row := 0; row < count; row++ {
		value, err := source.Produce(generate.Request{
			Generator: generator,
			At:        seed.Coordinate{Entity: "records", Row: int64(row), Field: generator, Purpose: "value"},
		})
		if err != nil {
			t.Fatalf("cannot produce a %s: %v", generator, err)
		}
		built = append(built, value)
	}
	return built
}

func tableOf(columns ...string) dataset.Table {
	return dataset.Table{Entity: "records", Columns: columns, Rows: 40}
}

// sampleOf builds what the reader would have gathered from a column with no
// gaps in it, so the values sit on records one upwards.
func sampleOf(field string, values []string) dataset.Sample {
	rows := make([]int, len(values))
	for at := range values {
		rows[at] = at + 1
	}
	return dataset.Sample{Field: field, Values: values, NonEmpty: len(values), Rows: rows}
}

// sampleAtRows builds what the reader would have gathered from a column with
// gaps in it, naming the record each value came from.
func sampleAtRows(field string, values []string, rows []int) dataset.Sample {
	return dataset.Sample{Field: field, Values: values, NonEmpty: len(values), Rows: rows}
}

func findingFor(t *testing.T, shape infer.Schema, field string) infer.Finding {
	t.Helper()
	for _, finding := range shape.Findings() {
		if finding.Field == field {
			return finding
		}
	}
	t.Fatalf("nothing was decided about %s at all", field)
	return infer.Finding{}
}

func TestAColumnOfNationalIdentifiersIsRecognisedFromItsValues(t *testing.T) {
	shape := infer.From(
		tableOf("pesel"),
		[]dataset.Sample{sampleOf("pesel", produced(t, values.Pesel, 40))},
		schemes(),
	)
	finding := findingFor(t, shape, "pesel")
	if finding.Scheme != "PL.pesel" {
		t.Errorf("a column of PESEL values was taken for %q, and the reason given was %q",
			finding.Scheme, finding.Basis)
	}
	if finding.Type != "identifier" {
		t.Errorf("the column was typed %q rather than as an identifier", finding.Type)
	}
	if !strings.Contains(finding.Basis, "40") || !strings.Contains(finding.Basis, "PL.pesel") {
		t.Errorf("the basis has to say how much was looked at and what accepted it, and it says %q",
			finding.Basis)
	}
}

// 🔴 Every scheme with a check digit is recognised in a column of its own
// values and in no other way, and it is asked of every one of them rather than
// of the newest. The payment card is why: an equipment identity is fifteen
// digits under the same formula as a card number, so a card validator that
// approved it would leave a column of IMEI values accepted by two schemes and
// recognised as neither. Walking every scheme catches the next pair like that
// on the day it arrives.
//
// The book number is accepted by the article number validator as well, because
// an ISBN-13 is a GTIN-13 carrying the prefix 978 or 979. It is recognised as
// itself all the same, because the book number declares the nesting and the
// narrower scheme of a declared pair is chosen (DEC-269). Until then this test
// held it as a named exception, which is what B-354 recorded.
func TestEverySchemeIsRecognisedInAColumnOfItsOwnValues(t *testing.T) {
	source := values.New("pl-PL", "infer")
	for _, generator := range source.Names() {
		scheme, behind := source.ProviderOf(generator)
		if !behind {
			continue
		}
		numbering, _ := source.ProviderNamed(scheme)
		checked := false
		for _, brought := range numbering.Capabilities() {
			checked = checked || brought == capability.IdentifierCheckDigit
		}
		if !checked {
			continue
		}
		finding := findingFor(t, infer.From(
			tableOf(generator),
			[]dataset.Sample{sampleOf(generator, produced(t, generator, 40))},
			schemes(),
		), generator)
		if finding.Scheme != scheme {
			t.Errorf("a column of %s values was taken for %q: %s", scheme, finding.Scheme, finding.Basis)
		}
	}
}

// A column of book numbers is accepted by two validators and recognised as the
// narrower, and the basis says both halves: what else accepted it, and that
// the definition of the scheme is what settled the choice. A reader handed
// only "isbn" would not know that an article number reading was open to them.
func TestABookNumberColumnIsRecognisedAsTheNarrowerOfTwoSchemes(t *testing.T) {
	finding := findingFor(t, infer.From(
		tableOf("book"),
		[]dataset.Sample{sampleOf("book", produced(t, "isbn", 40))},
		schemes(),
	), "book")
	if finding.Scheme != "isbn" {
		t.Fatalf("a column of book numbers was taken for %q: %s", finding.Scheme, finding.Basis)
	}
	for _, said := range []string{"by ean", "narrower isbn", "definition"} {
		if !strings.Contains(finding.Basis, said) {
			t.Errorf("the basis has to say %q, and it says %q", said, finding.Basis)
		}
	}
}

// 🔴 The pair DEC-269 does NOT cover. A column of American Express test numbers
// is accepted by the payment card and by the equipment identity, because both
// are fifteen digits under the same formula, and neither definition puts one
// inside the other. So nobody declares a nesting, and the column stays the
// refusal it was. A rule reading containment off the column would have chosen
// one of them here.
func TestAColumnTwoUnrelatedSchemesAcceptIsStillARefusal(t *testing.T) {
	amex := []string{
		"378282246310005", "371449635398431", "378282246310005", "371449635398431",
		"378282246310005", "371449635398431", "378282246310005", "371449635398431",
	}
	finding := findingFor(t, infer.From(
		tableOf("paid with"),
		[]dataset.Sample{sampleOf("paid with", amex)},
		schemes(),
	), "paid with")
	if finding.Scheme != "" {
		t.Fatalf("a column two unrelated schemes accept was given %q: %s", finding.Scheme, finding.Basis)
	}
	for _, named := range []string{"card", "imei", "alike"} {
		if !strings.Contains(finding.Basis, named) {
			t.Errorf("the refusal has to name %q, and it says %q", named, finding.Basis)
		}
	}
}

// The rule on its own, away from any real scheme. A narrower scheme naming the
// one beside it is chosen, a third scheme accepting the column and named by
// nobody keeps the refusal, and two schemes naming each other are no choice
// at all, because neither is the narrower.
func TestTheNarrowerSchemeIsChosenOnlyWhenItNamesEveryOther(t *testing.T) {
	wide := &acceptsAnything{id: "BB.wide"}
	narrow := &nestedInside{acceptsAnything: acceptsAnything{id: "AA.narrow"}, within: []provider.Provider{wide}}
	beside := &acceptsAnything{id: "CC.beside"}
	looped := &nestedInside{acceptsAnything: acceptsAnything{id: "DD.looped"}}
	loopedBack := &nestedInside{acceptsAnything: acceptsAnything{id: "EE.looped"}, within: []provider.Provider{looped}}
	looped.within = []provider.Provider{loopedBack}
	codes := []string{"1", "2", "3", "4", "5", "6", "7", "8"}
	for _, one := range []struct {
		name    string
		schemes []provider.Provider
		want    string
	}{
		{"a narrower scheme naming the wider one", []provider.Provider{wide, narrow}, "AA.narrow"},
		{"a third scheme nobody named", []provider.Provider{wide, narrow, beside}, ""},
		{"two schemes naming each other", []provider.Provider{looped, loopedBack}, ""},
	} {
		finding := findingFor(t, infer.From(
			tableOf("code"), []dataset.Sample{sampleOf("code", codes)}, one.schemes,
		), "code")
		if finding.Scheme != one.want {
			t.Errorf("%s: the column was given %q where %q was wanted: %s",
				one.name, finding.Scheme, one.want, finding.Basis)
		}
	}
}

// 🔴 The decision this defends is DEC-156, and it is the expensive half of it.
//
// A social security number validator answers about shape, because it has to
// accept the values this tool itself produces. Recognising a column from it
// would mean announcing telephone numbers and invoice numbers as social
// security numbers, and every defect aimed at one would carry a hard
// expectation that nobody is going to meet. So a column of REAL ones is not
// recognised either, and that is the price rather than an oversight.
func TestAColumnOfRealSocialSecurityNumbersIsStillNotRecognised(t *testing.T) {
	real := produced(t, values.SSN, 40)
	for _, value := range real {
		if verdict := (us.SSN{}).Validate(value); !verdict.Valid {
			t.Fatalf("this test is built on values its own provider rejects: %q", value)
		}
	}
	finding := findingFor(t,
		infer.From(tableOf("ssn"), []dataset.Sample{sampleOf("ssn", real)}, schemes()), "ssn")
	if finding.Scheme != "" {
		t.Errorf("a scheme with no check digit was read off values: %q", finding.Scheme)
	}
	if finding.Type != "string" {
		t.Errorf("an unrecognised column has to be left as text and was typed %q", finding.Type)
	}
}

// 🔴 The measurement behind the threshold in DEC-156, done rather than argued.
//
// Unrelated digits must not be announced as anything. The arithmetic says the
// weakest check digit lets one value through about one time in ten, so eight of
// them in a row is about one in a hundred million. This runs two hundred
// unrelated columns and asks for none.
func TestUnrelatedDigitsAreNotAnnouncedAsANumberingScheme(t *testing.T) {
	draws := seed.New("unrelated-digits")
	recognised := 0
	for column := 0; column < 200; column++ {
		var digits []string
		for row := 0; row < infer.MinimumEvidence; row++ {
			at := seed.Coordinate{Entity: "noise", Row: int64(column*100 + row), Purpose: "digits"}
			digits = append(digits, strconv.FormatInt(100000000+draws.IntN(at, 899999999), 10))
		}
		finding := findingFor(t,
			infer.From(tableOf("code"), []dataset.Sample{sampleOf("code", digits)}, schemes()), "code")
		if finding.Scheme != "" {
			recognised++
			t.Logf("column %d of unrelated digits was taken for %s: %v", column, finding.Scheme, digits)
		}
	}
	if recognised > 0 {
		t.Errorf("%d of 200 columns of unrelated digits were announced as a numbering scheme", recognised)
	}
}

// A column shorter than the threshold is left alone however well it fits, and
// the basis says which of the two reasons it was.
func TestTooFewValuesLeaveAColumnUnrecognised(t *testing.T) {
	short := produced(t, values.Pesel, infer.MinimumEvidence-1)
	finding := findingFor(t,
		infer.From(tableOf("pesel"), []dataset.Sample{sampleOf("pesel", short)}, schemes()), "pesel")
	if finding.Scheme != "" {
		t.Fatalf("%d values were enough to announce %s, and the threshold is %d",
			len(short), finding.Scheme, infer.MinimumEvidence)
	}
	if !strings.Contains(finding.Basis, strconv.Itoa(infer.MinimumEvidence)) {
		t.Errorf("the basis has to say how many values would have been needed, and it says %q",
			finding.Basis)
	}
}

// One value that does not fit is enough to refuse the whole column. A column of
// identifiers with typos in it is missed, which is the cheaper mistake.
//
// The value that does not fit is built by breaking a correct one and then asking
// the provider whether it is now broken. Typing one out is how this test failed
// the first time it ran: the number chosen to be wrong was a valid PESEL.
func TestOneValueThatDoesNotFitRefusesTheColumn(t *testing.T) {
	withATypo := produced(t, values.Pesel, 40)
	broken := brokenLastDigit(withATypo[17])
	if verdict := (pl.PESEL{}).Validate(broken); verdict.Valid {
		t.Fatalf("this test needs a value the validator refuses and %q is accepted", broken)
	}
	withATypo[17] = broken
	finding := findingFor(t,
		infer.From(tableOf("pesel"), []dataset.Sample{sampleOf("pesel", withATypo)}, schemes()), "pesel")
	if finding.Scheme != "" {
		t.Errorf("a column carrying a value the validator refuses was still announced as %s",
			finding.Scheme)
	}
}

// brokenLastDigit moves the final digit of a value by one, wrapping at nine.
func brokenLastDigit(value string) string {
	last := value[len(value)-1]
	moved := byte('0' + (last-'0'+1)%10)
	return value[:len(value)-1] + string(moved)
}

// Two schemes accepting the whole sample is a refusal rather than a choice,
// and the basis names both so that whoever reads it can settle it themselves.
func TestTwoSchemesAcceptingEverythingLeaveTheColumnUnrecognised(t *testing.T) {
	both := &acceptsAnything{id: "AA.first"}
	other := &acceptsAnything{id: "BB.second"}
	finding := findingFor(t, infer.From(
		tableOf("code"),
		[]dataset.Sample{sampleOf("code", []string{"1", "2", "3", "4", "5", "6", "7", "8"})},
		[]provider.Provider{both, other},
	), "code")
	if finding.Scheme != "" {
		t.Fatalf("one of two equally good schemes was chosen: %s", finding.Scheme)
	}
	for _, named := range []string{"AA.first", "BB.second"} {
		if !strings.Contains(finding.Basis, named) {
			t.Errorf("the basis has to name both schemes that fitted, and it says %q", finding.Basis)
		}
	}
}

// A column of blanks is an answer rather than a gap, and it says so.
func TestAColumnWithNoValuesSaysThatIsWhyNothingWasDecided(t *testing.T) {
	finding := findingFor(t, infer.From(
		tableOf("spare"),
		[]dataset.Sample{{Field: "spare", Empty: 40}},
		schemes(),
	), "spare")
	if finding.Basis == "" {
		t.Fatal("a column nothing was decided about still needs a basis, or it reads as overlooked")
	}
	if finding.Type != "string" {
		t.Errorf("a column of blanks was typed %q", finding.Type)
	}
}

// Every column of the table gets a finding, including the ones nothing was
// concluded about. A column with no entry at all would leave a reader unable to
// tell one that was examined from one that was skipped.
func TestEveryColumnGetsAFindingWithABasis(t *testing.T) {
	shape := infer.From(
		tableOf("id", "pesel", "note"),
		[]dataset.Sample{
			sampleOf("id", []string{"1", "2", "3"}),
			sampleOf("pesel", produced(t, values.Pesel, 40)),
			sampleOf("note", []string{"a", "b"}),
		},
		schemes(),
	)
	if len(shape.Findings()) != 3 {
		t.Fatalf("3 columns went in and %d findings came out", len(shape.Findings()))
	}
	for _, finding := range shape.Findings() {
		if finding.Field == "" || finding.Basis == "" || finding.Type == "" {
			t.Errorf("a finding is incomplete: %+v", finding)
		}
	}
}

// 🔴 What a file cannot say, this must not claim. DEC-158 in one test: every
// one of these would be a property of a contract rather than of the data, and
// claiming one lets a mutator break a limit this tool invented.
func TestNothingBeyondTypeAndSchemeIsClaimed(t *testing.T) {
	shape := infer.From(
		tableOf("pesel", "note"),
		[]dataset.Sample{
			sampleOf("pesel", produced(t, values.Pesel, 40)),
			sampleOf("note", []string{"a", "bb", "ccc"}),
		},
		schemes(),
	)
	for _, field := range shape.FieldsOf("records") {
		switch {
		case field.HasMaxLength:
			t.Errorf("%s was given a maximum length nobody declared", field.Name)
		case field.Required:
			t.Errorf("%s was called required and no file says that", field.Name)
		case field.Unique:
			t.Errorf("%s was called unique on the strength of values not repeating", field.Name)
		case len(field.Values) > 0:
			t.Errorf("%s was given a set of allowed values a file cannot carry", field.Name)
		}
	}
	if got := shape.PrimaryKeyOf("records"); len(got) > 0 {
		t.Errorf("a primary key was inferred: %v", got)
	}
	if got := shape.RulesOf("records"); len(got) > 0 {
		t.Errorf("%d relations between columns were inferred", len(got))
	}
	if got := shape.ParentsOf("records"); len(got) > 0 {
		t.Errorf("%d relations between tables were inferred from one file", len(got))
	}
}

// The list of what is not inferred travels with the answer. An empty one would
// make the manifest read as a complete account of the file, and the manifest
// model refuses that anyway.
func TestTheLimitsOfThisWayOfLookingAreStated(t *testing.T) {
	stated := infer.NotInferred()
	if len(stated) == 0 {
		t.Fatal("nothing says what this does not read, so it reads as reading everything")
	}
	for _, expected := range []string{"maximum length", "required", "unique", "relation"} {
		found := false
		for _, line := range stated {
			if strings.Contains(line, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("nothing in the stated limits mentions %q: %v", expected, stated)
		}
	}
}

// A view of one file answers about one entity and about nothing else.
func TestTheViewAnswersAboutTheOneTableItRead(t *testing.T) {
	shape := infer.From(tableOf("a"), []dataset.Sample{sampleOf("a", []string{"x"})}, schemes())
	if names := shape.EntityNames(); len(names) != 1 || names[0] != "records" {
		t.Fatalf("one file is one table and the view names %v", names)
	}
	if fields := shape.FieldsOf("somebody else"); fields != nil {
		t.Errorf("the view answered about an entity it never read: %v", fields)
	}
}

// acceptsAnything is a scheme that carries a check digit and takes every value,
// used to build the case where two schemes fit equally well. Real schemes reach
// that path too, the book number inside the article number and an American
// Express column under the card and the equipment identity, and each has its
// own test above. This one isolates the rule from any of them.
type acceptsAnything struct{ id string }

func (a *acceptsAnything) ID() string { return a.id }
func (a *acceptsAnything) Capabilities() []capability.Name {
	return []capability.Name{capability.IdentifierCheckDigit}
}
func (a *acceptsAnything) Validate(string) provider.Verdict { return provider.Valid() }
func (a *acceptsAnything) Generate(provider.Request) (string, error) {
	return "", nil
}

// nestedInside takes every value and declares that it lies inside the schemes
// given, which is how the narrower half of a pair is built without a real one.
type nestedInside struct {
	acceptsAnything
	within []provider.Provider
}

func (n *nestedInside) Within() []provider.Provider { return n.within }
