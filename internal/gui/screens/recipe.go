// The states of the screen a recipe is written on.
//
// They live beside the list rather than in it because that file reached the
// hard limit on the length of a file on 2026-09-02, when the preview added
// three answers and one place for the keyboard to stand. A file per screen is
// the seam that costs nothing: nobody reading about one screen has to walk
// past another.
package screens

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
				"question an empty screen used to leave unanswered, and the three " +
				"generators that cannot be filled in a locale say why rather than saying no",
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
			Keyboard: Keyboard{Tabs: 8},
		},
		{
			Name: "recipe-focus-button",
			Purpose: "the keyboard on the button that writes, which is the last stop of the " +
				"walk because the bar is read last",
			Screen:   screenRecipe,
			Recipe:   CleanRecipe,
			Into:     "a-directory",
			Keyboard: Keyboard{Tabs: 13},
		},
		{
			Name: "recipe-focus-example",
			Purpose: "the keyboard on the offer of a recipe that runs, which is on the empty " +
				"screen and nowhere else",
			Screen:   screenRecipe,
			Keyboard: Keyboard{Tabs: 7},
		},
		{
			Name: "recipe-focus-generators",
			Purpose: "the keyboard on the list of what this build can generate, which is a " +
				"heading somebody opens rather than a wall of names nobody asked for",
			Screen:   screenRecipe,
			Keyboard: Keyboard{Tabs: 9},
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
			Purpose:  "the keyboard on the pane holding the answer, which the toolkit's own scrolling area never lets it reach",
			Screen:   screenRecipe,
			Recipe:   BrokenRecipe,
			Keyboard: Keyboard{Tabs: 7},
		},
	}
}
