// The states of the screen a recipe is written on.
//
// They live beside the list rather than in it because that file reached the
// hard limit on the length of a file on 2026-09-02, when the preview added
// three answers and one place for the keyboard to stand. A file per screen is
// the seam that costs nothing: nobody reading about one screen has to walk
// past another.
package screens

import (
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/damage"
	"github.com/donislawdev/qadatagenerator/internal/gui/recipeview"
	"github.com/donislawdev/qadatagenerator/internal/produce"
)

// RecipeRun is what a run of the recipe editor left, for the states that show
// one.
type RecipeRun int

const (
	// RecipeRanNothing is every state where nobody pressed the button.
	RecipeRanNothing RecipeRun = iota
	// RecipeRanDamaged is a run of a recipe with a policy that carried out part
	// of its order, which is the answer that has the most to say: the files,
	// the manifest among them, and every count including the shortfall.
	RecipeRanDamaged
)

// What a sample run of a recipe with a policy did, as numbers rather than as a
// run, for the reason the take screen's samples give: which rows a share of
// them lands on is a draw, and a stored tree should be a statement about the
// screen rather than about the draw. Three records damaged one defect each, of
// which one kind this build carries out.
const (
	sampleRecipeRows    = 3
	sampleRecipeOrdered = 3
	sampleRecipeApplied = 1
)

// recipeRunFor is the way to write for a state, which never writes.
func recipeRunFor(ran RecipeRun) recipeview.Write {
	if ran == RecipeRanNothing {
		return nil
	}
	return func([]byte, produce.Settings, generate.Halt) (produce.Written, error) {
		return produce.Written{
			Files:   produce.Writes([]string{"customer.csv"}, true, ""),
			Rows:    sampleRecipeRows,
			Damaged: true,
			Counts:  damage.Counts{Requested: sampleRecipeOrdered, Applied: sampleRecipeApplied},
		}, nil
	}
}

// producedHere carries a run out where it is asked for, so that the screen has
// finished answering by the time the picture is taken.
func producedHere(work func() (produce.Written, error), done func(produce.Written, error)) {
	done(work())
}

// settleRecipe takes the recipe editor the rest of the way, through the calls a
// person makes: the form picked from its list, then the button.
func settleRecipe(screen *recipeview.View, state State) {
	if state.Form != "" {
		screen.ChooseForm(state.Form)
	}
	if state.Wrote != RecipeRanNothing {
		screen.Start()
	}
}

// recipeStates are the states of the screen a recipe is written on.
//
// Split for the reason the screen next door was: the list outgrew the hard
// limit on the length of a function, twice, and both seams cost nothing. The
// first is between what the screen SAYS and where the keyboard STANDS. The
// second, on 2026-09-03, is between what it says about a RECIPE and what it
// says about the BUILD, which is the same picture whatever anybody types.
func recipeStates() []State {
	states := append(recipeStatesOfTheAnswer(), recipeStatesOfTheHints()...)
	return append(states, recipeStatesOfTheKeyboard()...)
}

// recipeStatesOfTheAnswer are the answers this screen can give.
//
// Six, and each is a different sentence rather than a different arrangement of
// one: nothing written, something wrong, something usable, something being
// judged, something usable that will not run, and something usable that
// produces nothing. The last two arrived with the preview, and neither of them
// had anywhere to be drawn before it.
func recipeStatesOfTheAnswer() []State {
	return append(recipeStatesOfTheReading(), recipeStatesOfTheWriting()...)
}

// recipeStatesOfTheReading are the answers about what is written: nothing,
// something wrong, something usable, something being judged, and the rows a
// usable recipe would produce.
//
// Apart from the answers about writing below because the list of both passed
// the limit on the length of a function. Joined in this order, which is the
// order the list had as one.
func recipeStatesOfTheReading() []State {
	return []State{
		{
			Name:    "recipe-empty",
			Purpose: "nothing written yet, a sentence saying what to do rather than an empty rectangle",
			Screen:  screenRecipe,
		},
		{
			Name:    "recipe-problems",
			Purpose: "a recipe that cannot be used, each problem at the line it was written on",
			Screen:  screenRecipe,
			Recipe:  BrokenRecipe,
		},
		{
			Name:    "recipe-clean",
			Purpose: "a recipe that can be used, with what was read of it, so approval is not bare",
			Screen:  screenRecipe,
			Recipe:  CleanRecipe,
		},
		{
			Name:     "recipe-checking",
			Purpose:  "a recipe being judged, which for a large one means the screen says so and stays answerable",
			Screen:   screenRecipe,
			Recipe:   CleanRecipe,
			Checking: true,
		},
		{
			Name: "recipe-preview-related",
			Purpose: "the rows a recipe with a relation would produce, cut to the sample, " +
				"with the column naming the parent that comes from the writer rather than the recipe",
			Screen: screenRecipe,
			Recipe: RelatedRecipe,
		},
	}
}

// recipeStatesOfTheWriting are the answers about writing a usable recipe: ready
// to write, nothing to write, refused by the plan, an order of damage, a form
// with nothing to export, and what a run left.
func recipeStatesOfTheWriting() []State {
	return []State{
		{
			Name: "recipe-ready",
			Purpose: "a recipe that can be used with somewhere to write it, which is the " +
				"one state where the button is not the thing saying what is missing",
			Screen: screenRecipe,
			Recipe: CleanRecipe,
			Into:   "a-directory",
		},
		{
			Name: "recipe-no-rows",
			Purpose: "a recipe that can be used and produces nothing, which is a sentence " +
				"rather than a heading with an empty rectangle under it",
			Screen: screenRecipe,
			Recipe: EmptyRecipe,
		},
		{
			Name: "recipe-refused",
			Purpose: "a recipe that reads correctly and cannot be run, which the validator " +
				"cannot see and only the plan can say",
			Screen: screenRecipe,
			Recipe: RefusedRecipe,
		},
		{
			Name: "recipe-policy",
			Purpose: "a recipe that orders damage, before anything is written: the manifest " +
				"among the files, how much of each kind this build carries out, and a " +
				"sentence saying the rows shown are the rows before the damage (DEC-252)",
			Screen: screenRecipe,
			Recipe: PolicyRecipe,
			Into:   "a-directory",
			Form:   "csv",
		},
		{
			Name: "recipe-form-unused",
			Purpose: "an extra form of the manifest picked for a recipe that orders no damage, " +
				"which says that nothing will be exported rather than going grey",
			Screen: screenRecipe,
			Recipe: CleanRecipe,
			Into:   "a-directory",
			Form:   "csv",
		},
		{
			Name: "recipe-ran-damaged",
			Purpose: "what a run of a recipe with a policy left: the files with the manifest " +
				"and the note, then every count, the shortfall marked rather than left to " +
				"whoever compares two numbers",
			Screen: screenRecipe,
			Recipe: PolicyRecipe,
			Into:   "a-directory",
			Wrote:  RecipeRanDamaged,
		},
	}
}

// recipeStatesOfTheHints are what this screen says to somebody who has never
// seen a recipe.
//
// Separate from the answers above because it is not one. Every state there is
// this screen judging what somebody wrote, and this one is the screen
// describing the build: the same picture whatever is in the field, and the only
// one that has to be opened before it shows anything.
func recipeStatesOfTheHints() []State {
	return []State{
		{
			Name: "recipe-generators",
			Purpose: "the vocabulary this build carries, opened. It is the answer to the " +
				"question an empty screen used to leave unanswered, and a generator that " +
				"cannot be filled in a locale says why rather than saying no",
			Screen:     screenRecipe,
			Generators: true,
		},
	}
}

// recipeStatesOfTheKeyboard are the places the keyboard can stand on it.
//
// 🔴 The counts of Tab presses move whenever a screen is added to the window or
// a control is added to this one, because the column of screen names is walked
// first and every screen is one stop. They are checked by rendering rather than
// by counting in anybody's head.
func recipeStatesOfTheKeyboard() []State {
	return []State{
		{
			Name:     "recipe-focus-table",
			Purpose:  "the keyboard on the preview itself, which is where the arrow keys move a cell at a time",
			Screen:   screenRecipe,
			Recipe:   CleanRecipe,
			Keyboard: Keyboard{Tabs: 12},
		},
		{
			Name: "recipe-focus-button",
			Purpose: "the keyboard on the button that writes, which since 2026-10-21 " +
				"closes the second step under the editor, after every setting it uses",
			Screen:   screenRecipe,
			Recipe:   CleanRecipe,
			Into:     "a-directory",
			Keyboard: Keyboard{Tabs: 11},
		},
		{
			Name: "recipe-focus-example",
			Purpose: "the keyboard on the offer of a recipe that runs, which is in the card " +
				"on the empty screen and nowhere else",
			Screen:   screenRecipe,
			Keyboard: Keyboard{Tabs: 12},
		},
		{
			Name: "recipe-focus-generators",
			Purpose: "the keyboard on the list of what this build can generate, which is a " +
				"heading somebody opens rather than a wall of names nobody asked for",
			Screen:   screenRecipe,
			Keyboard: Keyboard{Tabs: 13},
		},
		{
			Name:     "recipe-focus-editor",
			Purpose:  "the keyboard in the field the recipe is written in, which it can also leave",
			Screen:   screenRecipe,
			Recipe:   BrokenRecipe,
			Keyboard: Keyboard{Tabs: 6},
		},
		{
			Name:     "recipe-focus-answer",
			Purpose:  "the keyboard on the card holding the answer, which the toolkit's own scrolling area never lets it reach",
			Screen:   screenRecipe,
			Recipe:   BrokenRecipe,
			Keyboard: Keyboard{Tabs: 11},
		},
	}
}
