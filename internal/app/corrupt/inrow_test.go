package corrupt_test

import (
	"fmt"
	"strings"
	"testing"
	"unicode"

	"github.com/donislawdev/qadatagenerator/internal/app/corrupt"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

const letteredID catalog.ID = "UNI.letters.only"

// lettered is damage made of letters, which finds a value holding none unfit,
// the way the emoji and the raised capitals do. It also takes an empty value,
// so that a column the row does not have would look like one it could damage.
type lettered struct{}

func (lettered) ID() catalog.ID { return letteredID }

func (lettered) Damage(target mutator.Target) mutator.Outcome {
	if target.Value != "" && strings.IndexFunc(target.Value, unicode.IsLetter) < 0 {
		return mutator.Unfit("the value holds no letter")
	}
	if target.Value == "Ewa" {
		// A decline that came from trying the damage, the way a swap the
		// scheme still accepts does, which is a finding about this row.
		return mutator.Skipped("the scheme still accepts the damaged value")
	}
	return mutator.Done(target.Value + "!")
}

// aRow answers for the columns of one record, and counts what it was asked.
type aRow struct {
	values map[string]string
	asked  []string
}

func (r *aRow) cell(field string) (corrupt.Cell, bool) {
	r.asked = append(r.asked, field)
	value, held := r.values[field]
	return corrupt.Cell{Field: recipe.FieldInfo{Name: field, Type: "string"}, Value: value}, held
}

func letteredEngine(t *testing.T) corrupt.Engine {
	t.Helper()
	return corrupt.New(aCatalog(t, letteredID), registry{letteredID: lettered{}}, noSchemes{},
		capability.NewSet(), seed.New("in a row"))
}

func letteredOrder(row int, field string, instead ...string) corrupt.Order {
	return corrupt.Order{ID: "d1", Mutator: letteredID, Entity: "person", Row: row,
		Field: field, Instead: instead}
}

// 🔴 Damage the value in the drawn column cannot carry lands on another column
// of the row that takes it, and the entry says where it landed.
func TestADeclinedValueMovesTheDamageToAColumnThatTakesIt(t *testing.T) {
	row := &aRow{values: map[string]string{"ssn": "123456789", "code": "4711", "name": "Anna"}}
	entry, change, wrote := letteredEngine(t).ApplyInRow(letteredOrder(1, "ssn", "code", "name"), row.cell)

	if !entry.Applied || !wrote {
		t.Fatalf("a row with a column of letters left the damage refused: %s", entry.FailureReason)
	}
	if change.Field != "name" || change.Value != "Anna!" {
		t.Errorf("the change is %+v, and only name holds letters", change)
	}
	if entry.Location.Field != "name" || entry.ValueAfter != "Anna!" {
		t.Errorf("the entry points at %q with %q after, not where the damage landed",
			entry.Location.Field, entry.ValueAfter)
	}
}

// When no column takes the damage the entry is the refusal from the column the
// order was drawn for, and nothing is written.
func TestEveryColumnDecliningLeavesTheRefusalOfTheFirst(t *testing.T) {
	row := &aRow{values: map[string]string{"ssn": "123456789", "code": "4711"}}
	entry, _, wrote := letteredEngine(t).ApplyInRow(letteredOrder(1, "ssn", "code"), row.cell)

	if entry.Applied || wrote {
		t.Fatal("damage made of letters was applied to a row holding none")
	}
	if entry.FailureKind != manifest.DeclinedByMutator || entry.Location.Field != "ssn" {
		t.Errorf("the refusal is %s at %q, want the decline at ssn", entry.FailureKind, entry.Location.Field)
	}
	if len(row.asked) != 2 {
		t.Errorf("the engine asked for %v, and both columns should have been tried", row.asked)
	}
}

// 🔴 A refusal about the damage or the run does not move on. It reads the same
// in every column, so trying another would only repeat it.
func TestARefusalThatIsNotADeclineStaysWhereTheOrderWasAimed(t *testing.T) {
	unimplemented := corrupt.New(aCatalog(t, letteredID), registry{}, noSchemes{},
		capability.NewSet(), seed.New("in a row"))
	row := &aRow{values: map[string]string{"ssn": "123456789", "name": "Anna"}}
	entry, _, wrote := unimplemented.ApplyInRow(letteredOrder(1, "ssn", "name"), row.cell)

	if entry.Applied || wrote || entry.FailureKind != manifest.Unimplemented {
		t.Fatalf("an unimplemented damage came back as %+v", entry)
	}
	if len(row.asked) != 1 || entry.Location.Field != "ssn" {
		t.Errorf("the engine asked for %v and points at %q, after a refusal no column changes",
			row.asked, entry.Location.Field)
	}
}

// 🔴 A decline that is a finding about the row stays where it was found.
//
// Moving the damage on until some column takes it would erase the finding,
// which is the search the lesson on searching forbids (DEC-221 is the case it
// came from). Both halves: the finding in the drawn column is not moved, and a
// finding in another column ends the search there rather than passing over it.
func TestAFindingAboutTheRowStaysWhereItWasFound(t *testing.T) {
	engine := letteredEngine(t)

	drawn := &aRow{values: map[string]string{"first": "Ewa", "second": "Nowak"}}
	entry, _, wrote := engine.ApplyInRow(letteredOrder(1, "first", "second"), drawn.cell)
	if entry.Applied || wrote || entry.Location.Field != "first" {
		t.Errorf("a finding in the drawn column moved the damage to %q (applied %v)",
			entry.Location.Field, entry.Applied)
	}

	further := &aRow{values: map[string]string{"code": "4711", "first": "Ewa"}}
	entry, _, _ = engine.ApplyInRow(letteredOrder(1, "code", "first"), further.cell)
	if entry.Location.Field != "first" || !strings.Contains(entry.FailureReason, "still accepts") {
		t.Errorf("a finding in another column was passed over, and the entry is %q at %q",
			entry.FailureReason, entry.Location.Field)
	}
}

// A column the row does not have is never where the damage lands, even when
// its empty value would have been taken.
func TestAColumnTheRowDoesNotHaveIsPassedOver(t *testing.T) {
	row := &aRow{values: map[string]string{"ssn": "123456789"}}
	entry, _, wrote := letteredEngine(t).ApplyInRow(letteredOrder(1, "ssn", "ghost"), row.cell)

	if entry.Applied || wrote {
		t.Fatalf("the damage landed on %q, a column the row does not have", entry.Location.Field)
	}
}

// 🔴 The column that takes the damage is drawn evenly from those that would.
//
// Walking on from the drawn column in schema order would hand every row to the
// first column of letters after the run of digits, and the other would never be
// damaged. Four hundred rows split between two columns leave each well inside a
// third to two thirds.
func TestTheColumnThatTakesTheDamageIsDrawnEvenly(t *testing.T) {
	engine := letteredEngine(t)
	landed := map[string]int{}
	const rows = 400
	for at := 1; at <= rows; at++ {
		row := &aRow{values: map[string]string{
			"d1": "1", "d2": "2", "d3": "3", "first": "Anna", "second": "Nowak",
		}}
		entry, change, wrote := engine.ApplyInRow(letteredOrder(at, "d1", "d2", "d3", "first", "second"), row.cell)
		if !entry.Applied || !wrote {
			t.Fatalf("row %d left the damage refused with two columns of letters in it", at)
		}
		landed[change.Field]++
	}
	if problem := unevenBetween(landed, rows, "first", "second"); problem != "" {
		t.Error(problem)
	}
	// The canary is the shape of the defect: every row to the first in turn.
	if unevenBetween(map[string]int{"first": rows}, rows, "first", "second") == "" {
		t.Fatal("the check does not notice one column taking every row")
	}
}

// unevenBetween says what is wrong when two columns did not share the rows
// between a third and two thirds each, and nothing when they did.
func unevenBetween(landed map[string]int, rows int, columns ...string) string {
	for _, column := range columns {
		if landed[column] < rows/3 || landed[column] > rows*2/3 {
			return fmt.Sprintf("%s took %d of %d rows: %v", column, landed[column], rows, landed)
		}
	}
	return ""
}
