package recipeview_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/app/validate"
	"github.com/donislawdev/qadatagenerator/internal/gui/recipeview"
	"github.com/donislawdev/qadatagenerator/internal/gui/screens"
	"github.com/donislawdev/qadatagenerator/internal/preview"
)

// The rows on screen are the rows the run would write.
//
// 🔴 It is held against the sample rather than against a list of expected
// values, because a list written here would be a second statement of what this
// build produces and would go on agreeing with itself after the engine changed.
// What makes the sample the run's own rows is guarded in internal/preview, and
// this is the other half: that the screen draws them rather than something else.
func TestTheRowsOnScreenAreTheRowsTheSampleCarries(t *testing.T) {
	view := here(t, real(t))
	view.SetRecipe(screens.RelatedRecipe)
	drawn := rendered(t, view)

	sample := sampleOf(t, screens.RelatedRecipe)
	if len(sample.Tables) != 2 {
		t.Fatalf("this recipe has two tables and the sample carries %d", len(sample.Tables))
	}
	for _, table := range sample.Tables {
		if !strings.Contains(drawn, ">"+table.Entity+"<") {
			t.Errorf("the screen does not name the table %q", table.Entity)
		}
		for _, column := range table.Columns {
			if !strings.Contains(drawn, ">"+column+"<") {
				t.Errorf("%s: the column %q is not on screen", table.Entity, column)
			}
		}
		// The first row is enough to say the values are drawn. Every row of a
		// table is drawn by one call, so a screen that draws one draws them all,
		// and asserting on all of them would be asserting on the toolkit.
		for _, value := range table.Rows[0] {
			if value != "" && !strings.Contains(drawn, ">"+value+"<") {
				t.Errorf("%s: the value %q is not on screen", table.Entity, value)
			}
		}
	}
}

// 🔴 A value cut although its column had room to hold it.
//
// Found by rendering, with every test green: an eleven digit number came out as
// nine and an ellipsis. A cell reports the width at which it exactly fits, the
// table takes its own spacing out of the column before handing the rest to the
// cell, and a column sized to the reported minimum is therefore a hair short of
// what it was sized for.
//
// The recipe here carries the longest values this build produces for a person:
// an identifier of eleven digits and a date of ten characters.
func TestAValueIsNotCutWhenItsColumnHasRoomForIt(t *testing.T) {
	view := here(t, real(t))
	view.SetRecipe(screens.RelatedRecipe)
	drawn := rendered(t, view)

	if strings.Contains(drawn, "…") {
		for _, line := range strings.Split(drawn, "\n") {
			if strings.Contains(line, "…") {
				t.Errorf("a value is cut although its column was sized for it: %s",
					strings.TrimSpace(line))
			}
		}
	}
}

// A recipe that produces nothing says so.
//
// Point eight of the GUI checklist, and the one drawing where a heading with an
// empty rectangle under it would read as the tool having failed rather than as
// the recipe asking for nothing.
func TestATableWithNoRowsIsASentenceRatherThanAnEmptyTable(t *testing.T) {
	view := here(t, real(t))
	view.SetRecipe(screens.EmptyRecipe)
	drawn := rendered(t, view)

	if !strings.Contains(drawn, i18n.English.T(i18n.RecipeNoRows)) {
		t.Error("a table with no rows does not say so")
	}
	if strings.Contains(drawn, "parts.KeyTable") {
		t.Error("a table with no rows was drawn as a table anyway")
	}
}

// 🔴 A recipe the validator approves and the plan refuses.
//
// The refusal takes the place of the approval rather than standing under it. A
// screen showing both would be the two answers this project measured as B-205,
// drawn one above the other on one screen instead of spread across two verbs.
func TestARecipeThePlanRefusesIsDrawnAsARefusalAndNotAsAnApproval(t *testing.T) {
	view := here(t, real(t))
	view.SetRecipe(screens.RefusedRecipe)
	drawn := rendered(t, view)

	if !strings.Contains(drawn, i18n.English.T(i18n.RecipeRefused)) {
		t.Error("a recipe the plan refuses is not drawn as refused")
	}
	if strings.Contains(drawn, i18n.English.T(i18n.RecipeNoObjections)) {
		t.Error("the screen approves a recipe it cannot run")
	}
}

// 🔴 Both forms of a counted word, because one of them plus a letter is wrong
// in English before it is wrong anywhere else.
//
// Found by rendering on 2026-09-02: four fields of two entities came out as
// "2 entitys". The singular reads correctly, which is why a guard that checks
// one recipe of one entity is a guard this walks past.
func TestTheCountsOnScreenUseTheRightFormOfEachWord(t *testing.T) {
	view := here(t, real(t))

	view.SetRecipe(screens.RelatedRecipe)
	if drawn := rendered(t, view); !strings.Contains(drawn, "2 entities, 4 fields") {
		t.Errorf("more than one of each is not written in the plural: %s", counts(drawn))
	}

	view.SetRecipe(screens.CleanRecipe)
	if drawn := rendered(t, view); !strings.Contains(drawn, "1 entity, 1 field") {
		t.Errorf("one of each is not written in the singular: %s", counts(drawn))
	}
}

// 🔴 A parent this sample does not reach is named rather than left blank.
//
// The writer looks a parent up among the rows it has written, and a sample
// holds ten of them, so a child pointing further gets an empty key back. An
// empty value is an ordinary thing for generated data to be, so drawn as
// nothing the two would be one cell, and one of them would be the tool failing
// to answer without saying so.
//
// The answer is handed in rather than produced, because the shares that reach
// past a sample depend on a draw and a test that waits for one is a test that
// passes for a reason nobody chose.
func TestAParentOutsideTheSampleIsNamedRatherThanLeftBlank(t *testing.T) {
	view := here(t, func(string, []byte) recipeview.Answer {
		return recipeview.Answer{
			Report: validate.Report{Entities: 1, Fields: 1},
			Sample: preview.Sample{Tables: []preview.Table{{
				Entity:       "order",
				Columns:      []string{"placed_at", "customer"},
				Rows:         [][]string{{"2004-01-25", ""}},
				Total:        1,
				ParentColumn: 1,
			}}},
		}
	})
	view.SetRecipe(screens.CleanRecipe)

	drawn := rendered(t, view)
	words := i18n.English.T(i18n.RecipeParentOutside)
	// In full, because the first drawing said it and said it cut short: the
	// column had been sized for the empty value the words replaced.
	if !strings.Contains(drawn, ">"+words+"<") {
		t.Errorf("a parent outside the sample is not named in full: %q", cell(drawn, words))
	}
}

// sampleOf is what the real check produces for a recipe, so that a test can
// hold the screen against it.
func sampleOf(t *testing.T, recipe string) preview.Sample {
	t.Helper()
	answer := real(t)(i18n.English.T(i18n.RecipeName), []byte(recipe))
	if answer.Refused != nil {
		t.Fatalf("this recipe cannot be run: %v", answer.Refused)
	}
	if !answer.Report.Usable() {
		t.Fatalf("this recipe cannot be used: %v", answer.Report.Problems)
	}
	return answer.Sample
}

// counts is the line of the drawing that carries the two numbers, for a failure
// that says what was on screen rather than only that it was wrong.
func counts(drawn string) string {
	for _, line := range strings.Split(drawn, "\n") {
		if strings.Contains(line, "entit") && strings.Contains(line, "field") {
			return strings.TrimSpace(line)
		}
	}
	return "no line of the drawing carries both counts"
}

// cell is the line of the drawing that begins with some words, so that a
// failure about text cut short can show how much of it arrived.
func cell(drawn, beginning string) string {
	for _, line := range strings.Split(drawn, "\n") {
		if strings.Contains(line, ">"+beginning[:len(beginning)/2]) {
			return strings.TrimSpace(line)
		}
	}
	return "those words are nowhere in the drawing"
}
