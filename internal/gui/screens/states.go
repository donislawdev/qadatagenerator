package screens

// The list of states, apart from the types and the assembly beside them.
//
// # Why this is a file of its own
//
// screens.go reached the hard ceiling on the length of a file when the fifth
// screen arrived, and the ceiling may be lowered and never raised. A list of
// states is not ordinary code in that respect: it grows with the number of
// things it describes, so the only move available is to cut it, and the seam
// that costs nothing is between what a state IS and which states there are.
// Nothing outside this package names either file, so the cut is invisible from
// anywhere else.

import (
	"fyne.io/fyne/v2"
)

// All is every state this window can be put into, for both screens.
//
// One list for both screens on purpose. A list written for one screen stays a
// list of one screen forever, and nobody notices, because it is green.
func All() []State {
	states := recipeStates()
	states = append(states, takeStates()...)
	states = append(states, browserStates()...)
	states = append(states, aboutStates()...)
	states = append(states, quickStates()...)
	return append(states, quickStatesOfTheKeyboard()...)
}

// quickStates are the states of the screen that produces one value.
//
// Every one of them names a seed. That screen draws a new seed whenever it is
// asked without one, which is what it is for and what makes a picture of it
// worthless as a guard unless the seed is fixed here.
func quickStates() []State {
	return []State{
		{
			Name:    "quick-empty",
			Purpose: "the screen before anything is chosen, which is what it opens as",
			Screen:  4,
		},
		{
			Name: "quick-value",
			Purpose: "a value produced by choosing alone, with the two fields it takes " +
				"account of shown beside it",
			Screen: 4,
			Quick:  Quick{Generator: "pesel", Seed: "for-the-picture"},
		},
		{
			Name: "quick-agreed",
			Purpose: "the same value bound to a gender and a date of birth, which the " +
				"first six digits and the tenth one carry",
			Screen: 4,
			Quick: Quick{Generator: "pesel", Gender: "female", Born: "1980-05-05",
				Seed: "for-the-picture"},
		},
		{
			Name: "quick-plain",
			Purpose: "a value that takes account of neither field, where neither field " +
				"is on screen rather than sitting there doing nothing",
			Screen: 4,
			// An American identifier rather than a city, and the render is
			// what settled it: this build carries no cities for en-US, so the
			// state meant to show a plain value showed a refusal instead.
			Quick: Quick{Generator: "ssn", Seed: "for-the-picture"},
		},
		{
			Name: "quick-unavailable",
			Purpose: "a value this build carries no data for in the locale chosen, which " +
				"is the refusal that comes from the engine rather than from the screen",
			Screen: 4,
			Quick:  Quick{Generator: "city", Seed: "for-the-picture"},
		},
		{
			Name: "quick-refused",
			Purpose: "a date of birth nobody can read, which is the one refusal this " +
				"screen can produce",
			Screen: 4,
			Quick:  Quick{Generator: "pesel", Born: "05-05-1980", Seed: "for-the-picture"},
		},
	}
}

// quickStatesOfTheKeyboard are the same screen with the keyboard somewhere on
// it, kept apart for the reason the screen next door keeps its own apart: they
// answer a different question and the list of them grows on its own.
func quickStatesOfTheKeyboard() []State {
	return []State{
		{
			Name:     "quick-focus-list",
			Purpose:  "the keyboard on the list of values, which is the first control here",
			Screen:   4,
			Quick:    Quick{Generator: "pesel", Seed: "for-the-picture"},
			Keyboard: Keyboard{Tabs: 6},
		},
		{
			Name:     "quick-focus-copy",
			Purpose:  "the keyboard on the button that puts the value in the clipboard",
			Screen:   4,
			Quick:    Quick{Generator: "pesel", Seed: "for-the-picture"},
			Keyboard: Keyboard{Tabs: 11},
		},
	}
}

// takeStates are the states of the screen that damages somebody else's file,
// which is the screen with the most of them.
func takeStates() []State {
	states := append(takeStatesBeforeARun(), takeStatesOfTheKeyboard()...)
	states = append(states, takeStatesOfARun()...)
	return append(states, takeStatesOfWhatARunLeft()...)
}

// takeStatesBeforeARun are the states in which nothing has been written.
func takeStatesBeforeARun() []State {
	return []State{

		{
			Name:    "take-empty",
			Purpose: "no file chosen, a sentence rather than an empty rectangle",
			Screen:  1,
		},
		{
			Name:        "take-refused",
			Purpose:     "a file read and no kind of damage chosen, so the screen says why it would not run",
			Screen:      1,
			NeedsSource: true,
		},
		{
			Name:        "take-plan",
			Purpose:     "a file read and two kinds of damage chosen, so the screen says what a run would do",
			Screen:      1,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
		},
	}
}

// takeStatesOfTheKeyboard are the places on this screen the keyboard can stand,
// which are worth a picture each because what marks them is a colour and a
// shape rather than a word.
func takeStatesOfTheKeyboard() []State {
	return []State{
		{
			Name: "take-focus-file",
			Purpose: "the keyboard on the button that chooses a file, which is the first " +
				"thing on this screen it reaches and was for a while the last",
			Screen:      1,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Keyboard:    Keyboard{Tabs: 6},
		},
		{
			Name: "take-focus-setting",
			Purpose: "the keyboard in a text setting, which is the one control the toolkit " +
				"marks with the chosen colour rather than the focus colour",
			Screen:      1,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Keyboard:    Keyboard{Tabs: 17},
		},
		{
			Name: "take-focus-choice",
			Purpose: "the keyboard on a chooser, which the toolkit floods with the focus " +
				"colour exactly as it floods a button, and which no state here " +
				"reached until that flooding was measured",
			Screen:      1,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Keyboard:    Keyboard{Tabs: 18},
		},
		{
			Name: "take-focus-pane",
			Purpose: "the keyboard on the pane that says what the file holds, which the " +
				"toolkit's own scrolling area never lets it reach",
			Screen:      1,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Keyboard:    Keyboard{Tabs: 7},
		},
		{
			Name: "take-focus-damage",
			Purpose: "the keyboard at the far end of the screen, on the button that damages " +
				"the file, having passed every setting of the run on the way",
			Screen:      1,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Keyboard:    Keyboard{Tabs: 26},
		},
		{
			Name: "take-reading",
			Purpose: "a file being read, which for a file this mode is for means walking " +
				"all of it while the window stays answerable",
			Screen:      1,
			NeedsSource: true,
			Reading:     true,
		},
	}
}

// takeStatesOfARun are the states a run puts the screen into.
//
// Every one of them ticks kinds of damage first, because a state pressing the
// button without them would be a picture of the screen refusing the run rather
// than of the run.
func takeStatesOfARun() []State {
	return []State{
		{
			Name:        "take-running",
			Purpose:     "a run going on, with the settings out of reach and the way to stop it in reach",
			Screen:      1,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Ran:         RanInFlight,
		},
		{
			Name: "take-focus-stop",
			Purpose: "the keyboard on the way to stop a run, which is the only thing on " +
				"this screen it can reach while one is going on",
			Screen:      1,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Ran:         RanInFlight,
			Keyboard:    Keyboard{Tabs: 8},
		},
	}
}

// takeStatesOfWhatARunLeft are the states of a run that is over.
//
// Split from the two above at the seam the screen itself has: while a run goes
// on the settings are out of reach and the button stops it, and afterwards the
// panel is a report of what happened. Splitting on that rather than on a line
// count keeps each list about one thing.
func takeStatesOfWhatARunLeft() []State {
	return []State{
		{
			Name:        "take-written",
			Purpose:     "a run that finished and carried out everything it ordered",
			Screen:      1,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Ran:         RanWritten,
		},
		{
			Name: "take-partial",
			Purpose: "a run that wrote its files with some of the ordered damage unapplied, " +
				"which is the state a count of what was applied hides",
			Screen:      1,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Ran:         RanPartial,
		},
		{
			Name:        "take-stopped",
			Purpose:     "a run somebody ended, which leaves nothing behind and says so",
			Screen:      1,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Ran:         RanStopped,
		},
		{
			Name:        "take-failed",
			Purpose:     "a run that ended on a failure, with what failed and what that left",
			Screen:      1,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Ran:         RanFailed,
		},
		{
			Name: "take-oversized",
			Purpose: "a run that succeeded and left a manifest too large to read back, which " +
				"nothing else about the screen shows",
			Screen:      1,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Ran:         RanOversized,
		},
	}
}

// browserStates are the states of the manifest browser.
func browserStates() []State {
	return []State{
		{
			Name:    "browser-empty",
			Purpose: "no manifest open, a sentence rather than an empty rectangle",
			Screen:  2,
		},
		{
			Name:          "browser-promised",
			Purpose:       "a manifest open, which is what a run promised about a dataset",
			Screen:        2,
			NeedsManifest: true,
		},
		{
			Name:          "browser-compared",
			Purpose:       "the same manifest beside what an application did about it",
			Screen:        2,
			NeedsManifest: true,
			NeedsResponse: true,
		},
		{
			Name:          "browser-focus-button",
			Purpose:       "the keyboard on the button that opens a manifest",
			Screen:        2,
			NeedsManifest: true,
			Keyboard:      Keyboard{Tabs: 6},
		},
		{
			Name:          "browser-focus-cell",
			Purpose:       "the keyboard inside the table, three rows down and one column across",
			Screen:        2,
			NeedsManifest: true,
			Keyboard: Keyboard{
				Tabs: 8,
				Keys: []fyne.KeyName{fyne.KeyDown, fyne.KeyDown, fyne.KeyDown, fyne.KeyRight},
			},
		},
	}
}

// aboutStates are the states of the screen naming what this build is.
func aboutStates() []State {
	return []State{
		{
			Name:    "about",
			Purpose: "versions, licences and every third party component",
			Screen:  3,
		},
		{
			Name:     "about-focus-name",
			Purpose:  "the keyboard on the name of the screen being read",
			Screen:   3,
			Keyboard: Keyboard{Tabs: 4},
		},
		{
			Name: "about-focus-body",
			Purpose: "the keyboard in the text of the screen, which the toolkit's own " +
				"scrolling area never allows it to reach",
			Screen:   3,
			Keyboard: Keyboard{Tabs: 6},
		},
		{
			Name: "about-scrolled",
			Purpose: "the same screen two pages further down, reached with the keyboard " +
				"alone, which is three quarters of it that a mouse used to be needed for",
			Screen: 3,
			Keyboard: Keyboard{
				Tabs: 6,
				Keys: []fyne.KeyName{fyne.KeyPageDown, fyne.KeyPageDown},
			},
		},
	}
}
