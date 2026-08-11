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

// sampleOf builds what the reader would have gathered from a column.
func sampleOf(field string, values []string) dataset.Sample {
	return dataset.Sample{Field: field, Values: values, NonEmpty: len(values)}
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
// used to build the case where two schemes fit equally well. It is the only way
// to reach that path, because no two real schemes accept the same forty values.
type acceptsAnything struct{ id string }

func (a *acceptsAnything) ID() string { return a.id }
func (a *acceptsAnything) Capabilities() []capability.Name {
	return []capability.Name{capability.IdentifierCheckDigit}
}
func (a *acceptsAnything) Validate(string) provider.Verdict { return provider.Valid() }
func (a *acceptsAnything) Generate(provider.Request) (string, error) {
	return "", nil
}
