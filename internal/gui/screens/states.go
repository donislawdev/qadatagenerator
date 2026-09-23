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

// The screens in the order the column shows them, named so that a state says
// WHICH screen it wants instead of a position that moves under it.
//
// 🔴 WHY THESE EXIST, AND IT IS MEASURED RATHER THAN TIDINESS. Until
// 2026-09-08 every state carried a bare number. Reordering the column that day
// repointed all forty-five of them at different screens, and NOT ONE TEST WENT
// RED: the stored trees regenerated cleanly, so the state named for the recipe
// editor happily recorded the screen that produces one value. What caught it
// was reading the diff, where a hundred and fifty-three strings of recipe text
// had quietly stopped appearing anywhere.
//
// A number pointing into a list somebody may reorder is a mine, and the guard
// that stops it going off is not this comment: it is the test asserting that
// each name below sits where its name says, which fails on a reorder instead
// of recording the wrong window.
const (
	screenQuick = iota
	screenTake
	screenRecipe
	screenManifest
	screenAbout
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
	states = append(states, quickStatesOfTheKeyboard()...)
	states = append(states, quickStatesOfThePanel()...)
	return append(states, quickStatesOfThePointer()...)
}

// quickStates are the states of the screen that produces one value.
//
// Every one of them names a seed. That screen draws a new seed whenever it is
// asked without one, which is what it is for and what makes a picture of it
// worthless as a guard unless the seed is fixed here.
func quickStates() []State {
	return []State{
		{
			// 🔴 The state named quick-empty stood here until 2026-09-08 and is
			// gone rather than renamed, because there is no longer a state it
			// could photograph: this screen chooses a kind while it is being
			// built, so it opens on values. A stored picture of a state nobody
			// can reach is a guard over something that cannot break, and it
			// would have been an exact copy of the state below it.
			Name: "quick-opens",
			Purpose: "what this screen opens as, which is the first thing anybody sees " +
				"of this product and the one state no drawing ever showed",
			Screen: screenQuick,
			// Only a seed, and no kind. The kind is the one the screen chooses
			// for itself, so naming it here would photograph a state somebody
			// reached rather than the state it opens in. The seed has to be
			// named because the real one is drawn at random, and a picture of it
			// would differ every time it was taken.
			Quick: Quick{Seed: "for-the-picture"},
		},
		{
			Name: "quick-value",
			Purpose: "a value produced by choosing alone, with the two fields it takes " +
				"account of shown beside it",
			Screen: screenQuick,
			Quick:  Quick{Generator: "pesel", Seed: "for-the-picture"},
		},
		{
			Name: "quick-many",
			Purpose: "ten values at once, which is what somebody filling a form over and " +
				"over actually needs, and the limit that comes with them",
			Screen: screenQuick,
			Quick:  Quick{Generator: "pesel", Seed: "for-the-picture", HowMany: "10"},
		},
		{
			Name: "quick-agreed",
			Purpose: "the same value bound to a gender and a date of birth, which the " +
				"first six digits and the tenth one carry",
			Screen: screenQuick,
			Quick: Quick{Generator: "pesel", Gender: "female", Born: "1980-05-05",
				Seed: "for-the-picture"},
		},
		{
			Name: "quick-plain",
			Purpose: "a value that takes account of neither field, where neither field " +
				"is on screen rather than sitting there doing nothing",
			Screen: screenQuick,
			// An American identifier rather than a city, and the render is
			// what settled it: this build carries no cities for en-US, so the
			// state meant to show a plain value showed a refusal instead.
			Quick: Quick{Generator: "ssn", Seed: "for-the-picture"},
		},
		{
			Name: "quick-unavailable",
			Purpose: "a value this build carries no data for in the locale chosen, which " +
				"is the refusal that comes from the engine rather than from the screen",
			Screen: screenQuick,
			Quick:  Quick{Generator: "city", Seed: "for-the-picture"},
		},
		{
			Name: "quick-refused",
			Purpose: "a date of birth nobody can read, which is the one refusal this " +
				"screen can produce",
			Screen: screenQuick,
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
			Name: "quick-focus-list",
			// 🔴 The list of KINDS, which is the first control here, and the
			// wording said "the list of values" until 2026-09-09.
			//
			// That was a name describing something else on the same screen, and
			// it went unnoticed because the pattern records whatever happened
			// rather than what the name promised. It mattered from the day the
			// card of values gained a keyboard stop of its own: there are now
			// two things somebody could mean by the list, and this state is the
			// other one. The stop for the values is fourteen, from
			// tools/klawiatura, and nothing photographs it yet.
			Purpose:  "the keyboard on the list of kinds, which is the first control here",
			Screen:   screenQuick,
			Quick:    Quick{Generator: "pesel", Seed: "for-the-picture"},
			Keyboard: Keyboard{Tabs: 6},
		},
		{
			Name:    "quick-focus-copy",
			Purpose: "the keyboard on the button that puts the value in the clipboard",
			Screen:  screenQuick,
			Quick:   Quick{Generator: "pesel", Seed: "for-the-picture"},
			// 🔴 Twelve rather than eleven since the count arrived, because the
			// list of how many stands second and every stop after it moved by
			// one. Taken from tools/klawiatura rather than counted here: at
			// eleven the keyboard landed on the seed field, which marks focus
			// with a border rather than a fill, so the stored tree simply lost
			// its focus fill and the pattern would have blessed that as the
			// picture of a focused button.
			//
			// 🔴 Thirteen rather than twelve since DEC-235, because Generate
			// stands after the seed and before the buttons over the card, so
			// every stop after it moved by one. Taken from tools/klawiatura
			// again rather than counted here, for the reason above: at twelve
			// the keyboard now lands on Generate, and the pattern would have
			// blessed a picture of the wrong button under this name.
			Keyboard: Keyboard{Tabs: 13},
		},
		{
			Name: "quick-focus-generate",
			// The one action this screen is for gets a state of its own, for
			// the reason every other primary action on this window has one: a
			// filled button marks focus with a ring drawn INSIDE it, and that
			// ring is a thing only a picture can check.
			Purpose:  "the keyboard on the button that produces, which is the one action this screen is for",
			Screen:   screenQuick,
			Quick:    Quick{Generator: "pesel", Seed: "for-the-picture"},
			Keyboard: Keyboard{Tabs: 12},
		},
	}
}

// quickStatesOfThePanel are the panel a kind is picked from, in each state a
// person can put it in from the keyboard (DEC-244).
//
// # Why five pictures of one panel
//
// Each shows something none of the others can. The panel as it opens, with the
// chosen kind marked and the keyboard on it. The keyboard moved onto a kind
// that is not chosen, which is the one state where the mark and the outline
// stand on different rows. A filter that leaves three rows under one heading
// and hides the other headings. A filter that leaves nothing, which is the
// state a list quietly goes blank in unless a sentence is put there. And the
// flat panel the locale chooser opens, which has no field and no foot.
//
// Every one of them opens the panel with Enter on a Tab count from
// tools/klawiatura, for the reason the states above do: the count is the only
// thing that names a control, and the walk is what says when it moves.
func quickStatesOfThePanel() []State {
	pesel := Quick{Generator: "pesel", Seed: "for-the-picture"}
	return []State{
		{
			Name:     "quick-kind-open",
			Purpose:  "the panel of kinds as it opens, with the chosen kind marked and the keyboard on it",
			Screen:   screenQuick,
			Quick:    pesel,
			Keyboard: Keyboard{Tabs: 6, Keys: []fyne.KeyName{fyne.KeyReturn}},
		},
		{
			Name:     "quick-kind-walked",
			Purpose:  "the keyboard moved two kinds down from the chosen one, so the mark and the outline stand apart",
			Screen:   screenQuick,
			Quick:    pesel,
			Keyboard: Keyboard{Tabs: 6, Keys: []fyne.KeyName{fyne.KeyReturn, fyne.KeyDown, fyne.KeyDown}},
		},
		{
			Name:     "quick-kind-filtered",
			Purpose:  "a filter typed into the panel, leaving the Polish kinds under their heading and nothing else",
			Screen:   screenQuick,
			Quick:    pesel,
			Keyboard: Keyboard{Tabs: 6, Keys: []fyne.KeyName{fyne.KeyReturn}, Typed: "pol"},
		},
		{
			Name:     "quick-kind-nothing",
			Purpose:  "a filter nothing matches, which the panel says in a sentence rather than by going blank",
			Screen:   screenQuick,
			Quick:    pesel,
			Keyboard: Keyboard{Tabs: 6, Keys: []fyne.KeyName{fyne.KeyReturn}, Typed: "zzz"},
		},
		{
			Name:     "quick-locale-open",
			Purpose:  "the flat panel a short list opens, with no field and no foot, so the two shapes can be compared",
			Screen:   screenQuick,
			Quick:    pesel,
			Keyboard: Keyboard{Tabs: 8, Keys: []fyne.KeyName{fyne.KeyReturn}},
		},
	}
}

// quickStatesOfThePointer are the states a MOUSE puts this screen into.
//
// # 🔴 Why there were none until 2026-09-09, and what that hid
//
// Because the controls had nothing to show. They stopped answering the pointer
// the day they took their drawing off the toolkit, and no picture in this
// repository had a mouse in it, so neither absence could reveal the other.
// Measured on the real binary: a window with the pointer over a sidebar item
// was identical, to the last of 1,965,600 pixels, to one with the pointer in
// empty space. The owner saw it by using the program.
//
// # Why six, and why these six
//
// One per thing that cannot be seen in any of the others, rather than one per
// control. The rule for the ordinary controls is a single function - the ground
// a pointer gives a control and the edge that ground needs - so photographing
// every button under a pointer would be photographing one rule five times.
//
// What differs between them is what each control has to say it WITH:
//
//   - the sidebar item has no edge in any state, so its ground is the only
//     signal it owns, and the step from hovered to pressed is the smallest
//     measurement in this whole system at 1.159.
//   - the quiet button has an edge, and pressing is the one state where that
//     edge moves, because the resting one measures 2.80 on the ground pressing
//     gives it.
//   - the chooser has a ground of its own at rest, so its step under a pointer
//     is smaller than a button's: 1.237 where a button on the window gets 1.451.
//   - the filled button is a different rule entirely. It is the one control
//     whose INK moves with its ground, because no violet in this ladder above
//     accent.9 carries a white label, and it needs both of its touched states
//     because they are two different fills with two different arithmetics.
//
// So three of the six are a state held down and three are a pointer resting,
// and the pairing is not symmetric on purpose: a control is photographed
// pressed only where pressing shows something hovering does not.
//
// Every count of tabs below comes from tools/klawiatura on this exact state,
// never from counting the controls here. Every one of them will move the next
// time this screen gains or loses a control, and the walk is the only thing
// that says so.
func quickStatesOfThePointer() []State {
	return []State{
		{
			Name: "quick-pointer-sidebar",
			Purpose: "the pointer on the column of screens, which draws no edge in any state " +
				"and so has only its ground to answer with",
			Screen: screenQuick,
			Quick:  Quick{Seed: "for-the-picture"},
			// 🔴 The SECOND item and not the first, which was chosen after
			// looking at the first. Item one is the screen already open, so it
			// rests on bg.current rather than on nothing, and the step a
			// pointer gives it is 1.131 where the same step on any other item
			// is 1.357. That is a real difference and it is left alone: the
			// item a person hovers in order to click is one of the others, and
			// clicking the open one does nothing at all, so a quieter answer
			// there is honest rather than broken.
			//
			// What it costs is stated: nothing photographs the open item under
			// a pointer, so nothing would notice if that 1.131 became 1.000.
			Pointer: Pointer{Tabs: 2},
		},
		{
			Name: "quick-pointer-sidebar-pressed",
			Purpose: "that same item held down, which is the smallest difference this system asks " +
				"anybody to see and therefore the one worth a picture",
			Screen:  screenQuick,
			Quick:   Quick{Seed: "for-the-picture"},
			Pointer: Pointer{Tabs: 2, Pressed: true},
		},
		{
			Name: "quick-pointer-chooser",
			Purpose: "the pointer on a chooser, which already has a ground at rest and so moves " +
				"less under one than a button does",
			Screen:  screenQuick,
			Quick:   Quick{Seed: "for-the-picture"},
			Pointer: Pointer{Tabs: 6},
		},
		{
			Name: "quick-pointer-quiet-pressed",
			Purpose: "a quiet button held down, which is the only state where the edge of a control " +
				"moves rather than its ground",
			Screen:  screenQuick,
			Quick:   Quick{Seed: "for-the-picture"},
			Pointer: Pointer{Tabs: 11, Pressed: true},
		},
		{
			Name: "quick-pointer-primary",
			Purpose: "the pointer on the one action this screen is for, which is the only control " +
				"here whose label changes colour with its fill",
			Screen:  screenQuick,
			Quick:   Quick{Seed: "for-the-picture"},
			Pointer: Pointer{Tabs: 10},
		},
		{
			Name: "quick-pointer-primary-pressed",
			Purpose: "that same action held down, on the third and last fill the accent ladder " +
				"offers a button that has to carry a label",
			Screen:  screenQuick,
			Quick:   Quick{Seed: "for-the-picture"},
			Pointer: Pointer{Tabs: 10, Pressed: true},
		},
	}
}

// takeStates are the states of the screen that damages somebody else's file,
// which is the screen with the most of them.
func takeStates() []State {
	states := append(takeStatesBeforeARun(), takeStateOfThePointer()...)
	states = append(states, takeStatesOfTheKeyboard()...)
	states = append(states, takeStatesOfARun()...)
	return append(states, takeStatesOfWhatARunLeft()...)
}

// takeStatesBeforeARun are the states in which nothing has been written.
func takeStatesBeforeARun() []State {
	return []State{

		{
			Name:    "take-empty",
			Purpose: "no file chosen, a sentence rather than an empty rectangle",
			Screen:  screenTake,
		},
		{
			Name:        "take-refused",
			Purpose:     "a file read and no kind of damage chosen, so the screen says why it would not run",
			Screen:      screenTake,
			NeedsSource: true,
		},
		{
			Name:        "take-plan",
			Purpose:     "a file read and two kinds of damage chosen, so the screen says what a run would do",
			Screen:      screenTake,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
		},
	}
}

// takeStateOfThePointer is one picture on a SECOND screen, and it exists to
// answer a question the six on the first one cannot.
//
// 🔴 Every touched state photographed elsewhere lives on the screen the states
// of the pointer were built on, and a mechanism proved on one screen is a
// mechanism nobody has run on the others. This project has paid for that
// distinction: twenty three stored trees stood green for three weeks and failed
// at the first run on a system they had never been run on.
//
// One rather than six, because what differs between screens is not the rule -
// the controls are the same four types everywhere - but whether the screen
// USES them. This build has no stock button or chooser on any screen, which is
// what TestNoScreenHoldsAControlTheToolkitFloods already holds, so one picture
// here turns that from an argument into a photograph.
func takeStateOfThePointer() []State {
	return []State{
		{
			Name: "take-pointer-choose",
			Purpose: "the pointer on a control of a screen the states of the pointer were not " +
				"built on, which is the only thing saying they are not a property of one screen",
			Screen: screenTake,
			// Six, from tools/klawiatura on this state: five names in the
			// column and then the one control this screen has before a file
			// has been chosen.
			Pointer: Pointer{Tabs: 6},
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
			Screen:      screenTake,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Keyboard:    Keyboard{Tabs: 6},
		},
		// 🔴 Every count below is from tools/klawiatura on 2026-09-15, the day
		// the screen moved onto paint (DEC-245) and its walk changed shape: the
		// band holds the file, how it is read and the act, the column under it
		// the settings of the damage, and the panel comes last. Taken from the
		// walk rather than counted here, for the reason the quick screen's
		// states give: a count from reading lands on the neighbour.
		{
			Name: "take-focus-setting",
			Purpose: "the keyboard in a text setting, the share of records, which is the " +
				"first setting of the damage under the band",
			Screen:      screenTake,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Keyboard:    Keyboard{Tabs: 18},
		},
		{
			Name: "take-focus-choice",
			Purpose: "the keyboard on a chooser, the one for defects per record, which " +
				"draws its own ring like every other control here",
			Screen:      screenTake,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Keyboard:    Keyboard{Tabs: 19},
		},
		{
			Name: "take-focus-pane",
			Purpose: "the keyboard on the panel that says what the file holds, which the " +
				"toolkit's own scrolling area never lets it reach, and which is the " +
				"last stop of the screen before the command",
			Screen:      screenTake,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Keyboard:    Keyboard{Tabs: 25},
		},
		{
			Name: "take-focus-damage",
			Purpose: "the keyboard on the button that damages the file, which stands in the " +
				"band beside the file it is about, before the settings of the damage",
			Screen:      screenTake,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Keyboard:    Keyboard{Tabs: 9},
		},
		{
			Name: "take-reading",
			Purpose: "a file being read, which for a file this mode is for means walking " +
				"all of it while the window stays answerable",
			Screen:      screenTake,
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
			Screen:      screenTake,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Ran:         RanInFlight,
		},
		{
			Name: "take-focus-stop",
			Purpose: "the keyboard on the way to stop a run, which stands where the button " +
				"that started it stood and is the first thing on this screen it can " +
				"reach while one is going on",
			Screen:      screenTake,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Ran:         RanInFlight,
			Keyboard:    Keyboard{Tabs: 6},
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
			Screen:      screenTake,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Ran:         RanWritten,
		},
		{
			Name: "take-partial",
			Purpose: "a run that wrote its files with some of the ordered damage unapplied, " +
				"which is the state a count of what was applied hides",
			Screen:      screenTake,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Ran:         RanPartial,
		},
		{
			Name:        "take-stopped",
			Purpose:     "a run somebody ended, which leaves nothing behind and says so",
			Screen:      screenTake,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Ran:         RanStopped,
		},
		{
			Name:        "take-failed",
			Purpose:     "a run that ended on a failure, with what failed and what that left",
			Screen:      screenTake,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Ran:         RanFailed,
		},
		{
			Name: "take-oversized",
			Purpose: "a run that succeeded and left a manifest too large to read back, which " +
				"nothing else about the screen shows",
			Screen:      screenTake,
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
			Screen:  screenManifest,
		},
		{
			Name:          "browser-promised",
			Purpose:       "a manifest open, which is what a run promised about a dataset",
			Screen:        screenManifest,
			NeedsManifest: true,
		},
		{
			Name:          "browser-compared",
			Purpose:       "the same manifest beside what an application did about it",
			Screen:        screenManifest,
			NeedsManifest: true,
			NeedsResponse: true,
		},
		{
			Name:          "browser-envelope",
			Purpose:       "the envelope of the run in the card under the table, in place of an entry",
			Screen:        screenManifest,
			NeedsManifest: true,
			Envelope:      true,
		},
		{
			Name:          "browser-focus-button",
			Purpose:       "the keyboard on the button that opens a manifest",
			Screen:        screenManifest,
			NeedsManifest: true,
			Keyboard:      Keyboard{Tabs: 6},
		},
		{
			// Tabs derived with tools/klawiatura on 2026-09-23: five screens,
			// the two buttons of the band, then the table. The button that
			// puts the run in the card stood over the table until that day
			// and was the eighth stop. It moved under the table with the
			// envelope (DEC-249), and at 9 this state photographed it. The
			// card under the table reads out the entry the keyboard lands on.
			Name:          "browser-focus-cell",
			Purpose:       "the keyboard inside the table, three rows down and one column across, and the card reading that entry out",
			Screen:        screenManifest,
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
			Screen:  screenAbout,
		},
		// Tabs derived with tools/klawiatura on 2026-09-23: five screens, the two
		// buttons of the band, then the page. 🔴 The name stood at 4 until then,
		// which is the manifest browser: the stored picture of "the keyboard on
		// the name of this screen" had its ring on another screen's name, and it
		// was green, because a picture records what happened rather than what
		// the name of the state promises.
		{
			Name:     "about-focus-name",
			Purpose:  "the keyboard on the name of the screen being read",
			Screen:   screenAbout,
			Keyboard: Keyboard{Tabs: 5},
		},
		{
			Name: "about-focus-body",
			Purpose: "the keyboard in the text of the screen, which the toolkit's own " +
				"scrolling area never allows it to reach",
			Screen:   screenAbout,
			Keyboard: Keyboard{Tabs: 8},
		},
		{
			Name: "about-scrolled",
			Purpose: "the same screen two pages further down, reached with the keyboard " +
				"alone, which is three quarters of it that a mouse used to be needed for",
			Screen: screenAbout,
			Keyboard: Keyboard{
				Tabs: 8,
				Keys: []fyne.KeyName{fyne.KeyPageDown, fyne.KeyPageDown},
			},
		},
	}
}
