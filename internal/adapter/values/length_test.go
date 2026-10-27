package values_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// numeric is a recipe of one field described by type alone, with a limit.
func numeric(kind string, limit, rows int) recipe.Recipe {
	return person(rows, []recipe.Field{
		{Name: "n", Type: kind, MaxLength: limit, HasMaxLength: true},
	})
}

// A field described by type alone that cannot hold the shortest value of its
// type is refused by the plan, before anything is written.
//
// Reproduced by an outside review on 2026-10-22: a decimal limited to three
// characters passed validate and generate, and every row held 0.44 or the like,
// four characters each.
func TestALimitTooShortForTheTypeIsRefusedBeforeTheRun(t *testing.T) {
	for _, given := range []struct {
		kind  string
		limit int
	}{
		{values.TypeDecimal, 3}, {values.TypeDecimal, 1}, {values.TypeInteger, 0},
	} {
		rec := numeric(given.kind, given.limit, 30)
		_, err := generate.Prepare(rec, values.New(rec.Locale, rec.Seed), generate.NeverStops{})
		if err == nil {
			t.Errorf("a %s limited to %d was planned, and no value of it fits", given.kind, given.limit)
			continue
		}
		if fault.KindOf(err) != fault.Config || !strings.Contains(err.Error(), "entities.person.fields.n") {
			t.Errorf("a %s limited to %d was refused as %v with %q", given.kind, given.limit,
				fault.KindOf(err), err)
		}
	}
}

// Every value of a field described by type alone stays inside its limit, at
// the shortest limit the plan accepts and above it.
func TestEveryValueOfATypeStaysInsideItsLimit(t *testing.T) {
	for _, given := range []struct {
		kind  string
		limit int
	}{
		{values.TypeDecimal, 4}, {values.TypeDecimal, 5}, {values.TypeInteger, 1},
		{values.TypeInteger, 2},
	} {
		_, rows := produced(t, numeric(given.kind, given.limit, 500))
		for _, row := range rows {
			if length := utf8.RuneCountInString(row["n"]); length > given.limit || length == 0 {
				t.Errorf("a %s limited to %d holds %q", given.kind, given.limit, row["n"])
				break
			}
		}
	}
}
