package infer_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/infer"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// peselScheme is the numbering scheme these tests read gender and date out of.
func peselScheme(t *testing.T) provider.Provider {
	t.Helper()
	for _, scheme := range schemes() {
		if scheme.ID() == "PL.pesel" {
			return scheme
		}
	}
	t.Fatal("this build carries no PL.pesel, so there is nothing to read a relation off")
	return nil
}

// neighbours builds the two columns that agree with a column of identifiers,
// by reading what each identifier encodes rather than by writing values down.
//
// The words for the genders are deliberately not the ones anybody would guess.
// A column agreeing one for one is agreeing whatever its values say, and a test
// using "male" and "female" would pass just as well against an implementation
// that recognised those two words and nothing else.
func neighbours(t *testing.T, identifiers []string) (genders []string, dates []string) {
	t.Helper()
	scheme := peselScheme(t)
	readsGender, ok := scheme.(provider.GenderEncoder)
	if !ok {
		t.Fatal("PL.pesel no longer reads a gender, so this relation cannot be proved")
	}
	readsDate, ok := scheme.(provider.DateEncoder)
	if !ok {
		t.Fatal("PL.pesel no longer reads a date, so this relation cannot be proved")
	}
	for _, value := range identifiers {
		gender, ok := readsGender.EncodedGender(value)
		if !ok {
			t.Fatalf("cannot read the gender out of %s", value)
		}
		// Two words from nobody's vocabulary, so that what is proved is the
		// mapping and not the spelling.
		if gender == provider.Female {
			genders = append(genders, "aleph")
		} else {
			genders = append(genders, "bet")
		}
		date, ok := readsDate.EncodedDate(value)
		if !ok {
			t.Fatalf("cannot read the date out of %s", value)
		}
		dates = append(dates, pad(date.Year, 4)+"-"+pad(date.Month, 2)+"-"+pad(date.Day, 2))
	}
	return genders, dates
}

func pad(number, width int) string {
	written := strconv.Itoa(number)
	for len(written) < width {
		written = "0" + written
	}
	return written
}

func relationBetween(shape infer.Schema, field, with string) (infer.Relation, bool) {
	for _, relation := range shape.Relations() {
		if relation.Field == field && relation.With == with {
			return relation, true
		}
	}
	return infer.Relation{}, false
}

// 🔴 A column standing one for one against the gender a scheme encodes is tied
// to it, whatever its values say.
//
// What a gender column looks like in somebody else's file cannot be known: it
// is M and F, or K and M, or two words in a language this build does not read.
// So what is asked is not what the values mean but whether they map one for one
// onto what the scheme encodes, every record the same way.
func TestAColumnStandingForTheEncodedGenderIsTiedToIt(t *testing.T) {
	identifiers := produced(t, "pesel", 60)
	genders, dates := neighbours(t, identifiers)

	shape := infer.From(tableOf("pesel", "sex", "born"), []dataset.Sample{
		sampleOf("pesel", identifiers),
		sampleOf("sex", genders),
		sampleOf("born", dates),
	}, schemes())

	found, has := relationBetween(shape, "pesel", "sex")
	if !has {
		t.Fatalf("the gender column was not tied to the identifier. Relations: %v", shape.Relations())
	}
	if found.Kind != infer.GenderRelation {
		t.Errorf("the relation is of kind %q, want %q", found.Kind, infer.GenderRelation)
	}
	if found.Basis == "" {
		t.Error("the relation carries no basis, so it is a conclusion nobody can argue with")
	}
	if _, has := relationBetween(shape, "pesel", "born"); !has {
		t.Error("the date column was not tied to the identifier")
	}
	// The rule is what the engine reads, and it names both neighbours.
	rules := shape.RulesOf("records")
	if len(rules) != 1 || rules[0].Field != "pesel" || rules[0].Kind != recipe.Matches {
		t.Fatalf("the schema does not offer one matches rule for pesel: %v", rules)
	}
	if len(rules[0].With) != 2 {
		t.Errorf("the rule ties pesel to %v, want both neighbours", rules[0].With)
	}
}

// 🔴 The other direction, and it is the whole value of the threshold.
//
// A column that does not agree is not tied, however ordinary it looks. The
// cases here are the three ways an agreement can be worth nothing: it does not
// hold, it holds over one gender only, or it holds over too few ways of being
// dealt out to mean anything.
func TestAnAgreementWorthNothingIsRefused(t *testing.T) {
	identifiers := produced(t, "pesel", 60)
	genders, dates := neighbours(t, identifiers)

	shifted := append([]string{}, genders...)
	copy(shifted, genders[1:])
	shifted[len(shifted)-1] = genders[0]

	// One gender for every record. Every column holding a single repeated value
	// agrees with this perfectly, and the agreement says nothing at all.
	oneGender := make([]string, len(genders))
	for at := range oneGender {
		oneGender[at] = "aleph"
	}

	for _, given := range []struct {
		what    string
		column  []string
		records int
	}{
		{"a column that does not agree", shifted, 60},
		{"a column agreeing over one gender only", oneGender, 60},
		{"a column agreeing over too few records", genders, 8},
	} {
		t.Run(given.what, func(t *testing.T) {
			shape := infer.From(tableOf("pesel", "sex"), []dataset.Sample{
				sampleOf("pesel", identifiers[:given.records]),
				sampleOf("sex", given.column[:given.records]),
			}, schemes())
			if found, has := relationBetween(shape, "pesel", "sex"); has {
				t.Errorf("%s was tied to the identifier anyway: %s", given.what, found.Basis)
			}
			if rules := shape.RulesOf("records"); len(rules) != 0 {
				t.Errorf("%s produced a rule for the engine: %v", given.what, rules)
			}
		})
	}
	// The date column of the same file is untouched by all of that, so the
	// refusals above are about the column they name and not about the file.
	shape := infer.From(tableOf("pesel", "born"), []dataset.Sample{
		sampleOf("pesel", identifiers), sampleOf("born", dates),
	}, schemes())
	if _, has := relationBetween(shape, "pesel", "born"); !has {
		t.Error("the date column was refused as well, so the refusals are too broad")
	}
}

// 🔴 An agreement is a relation only when it could have come out a hundred
// million other ways, and the split of the records decides that, however many
// of them agree.
//
// The count used to stop as soon as a partial product passed the bar, and the
// partial products of the last group rise past the final answer before they
// come back down to it. Twenty nine records split ten to nineteen can be dealt
// out about twenty million ways, thirty records of one gender in exactly one,
// and both were announced (B-355). Fifteen to fifteen can be dealt out about a
// hundred and fifty five million ways, and it is the control: the two refusals
// have to come from the count and not from thirty records being too few.
func TestASplitUnderTheBarIsRefusedHoweverManyRecordsAgree(t *testing.T) {
	reads, ok := peselScheme(t).(provider.GenderEncoder)
	if !ok {
		t.Fatal("PL.pesel no longer reads a gender, so this relation cannot be proved")
	}
	var women, men []string
	for _, value := range produced(t, "pesel", 400) {
		if gender, _ := reads.EncodedGender(value); gender == provider.Female {
			women = append(women, value)
		} else {
			men = append(men, value)
		}
	}
	for _, given := range []struct {
		what       string
		women, men int
		related    bool
	}{
		{"ten women and nineteen men", 10, 19, false},
		{"thirty women and no men", 30, 0, false},
		{"fifteen women and fifteen men", 15, 15, true},
	} {
		t.Run(given.what, func(t *testing.T) {
			if len(women) < given.women || len(men) < given.men {
				t.Fatalf("400 numbers gave %d women and %d men, too few for this split",
					len(women), len(men))
			}
			identifiers := append(append([]string{}, women[:given.women]...), men[:given.men]...)
			genders, _ := neighbours(t, identifiers)
			shape := infer.From(tableOf("pesel", "sex"), []dataset.Sample{
				sampleOf("pesel", identifiers), sampleOf("sex", genders),
			}, schemes())
			found, has := relationBetween(shape, "pesel", "sex")
			switch {
			case has && !given.related:
				t.Errorf("%s were tied to the identifier: %s", given.what, found.Basis)
			case !has && given.related:
				t.Errorf("%s were not tied to the identifier. Relations: %v", given.what,
					shape.Relations())
			}
		})
	}
}

// 🔴 Two columns are compared record by record, not by position.
//
// Empty cells are dropped when a column is sampled, so the third value of one
// column and the third of another come from the same record only when neither
// has a gap above it. Reading them by position compares one person's identifier
// against another person's gender, and here that would turn a column that does
// agree into one that does not.
func TestColumnsAreLinedUpByRecordAndNotByPosition(t *testing.T) {
	identifiers := produced(t, "pesel", 60)
	genders, _ := neighbours(t, identifiers)

	// The neighbour is missing its first value, as a column with one blank in
	// it would be. Read by position every pair after it is wrong.
	rows := make([]int, 0, len(genders)-1)
	for at := 2; at <= len(genders); at++ {
		rows = append(rows, at)
	}

	shape := infer.From(tableOf("pesel", "sex"), []dataset.Sample{
		sampleOf("pesel", identifiers),
		sampleAtRows("sex", genders[1:], rows),
	}, schemes())

	found, has := relationBetween(shape, "pesel", "sex")
	if !has {
		t.Fatalf("a column with one blank above it was not tied to the identifier, "+
			"so the two were read by position. Relations: %v", shape.Relations())
	}
	if want := "59 records"; !strings.Contains(found.Basis, want) {
		t.Errorf("the basis says %q and should count only the records both columns hold (%s)",
			found.Basis, want)
	}
}

// A file nobody can read a scheme out of has no relations either, because a
// relation here hangs off a scheme.
func TestAFileWithNoSchemeInItHasNoRelations(t *testing.T) {
	shape := infer.From(tableOf("left", "right"), []dataset.Sample{
		sampleOf("left", []string{"1", "2", "3", "4", "5", "6", "7", "8"}),
		sampleOf("right", []string{"a", "b", "a", "b", "a", "b", "a", "b"}),
	}, schemes())
	if len(shape.Relations()) != 0 {
		t.Errorf("two columns of nothing in particular were tied together: %v", shape.Relations())
	}
}
