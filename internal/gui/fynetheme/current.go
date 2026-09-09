package fynetheme

import "fyne.io/fyne/v2"

// Current is the theme this window is drawn with today.
//
// # Why one function rather than each place naming a theme
//
// Eight places install a theme: the product, the probe that renders every
// screen to a picture, the probe that walks the keyboard, the mock, and four
// package tests. Every one of them is asking the same question, which is what
// the WINDOW looks like, and every one of them answered it by naming a type.
// A transition between two themes then has eight places to change and no way to
// tell that one was missed, because a probe left on the old theme renders a
// picture that simply looks like the old window. That is the shape of defect
// this project keeps finding: a register kept in more than one copy, where the
// copies stay green while they drift.
//
// # What this is NOT, and the distinction is the reason two themes still exist
//
// A test OF a theme names that theme. TestTheThemeAnswersForEveryColour asks
// what Dark answers for the palette it was built from, so it says Dark and has
// to keep saying it. This function answers a different question, which is which
// of them the window wears.
//
// # Why it answers Visual as of 2026-09-08
//
// DEC-233. The window changes theme at the FIRST screen rebuilt on the new
// foundation rather than after the last one. Two themes at once would mean that
// moving between screens changes the look of the program, which reads as a
// fault or as two programs stitched together, and that is worse than one screen
// in a new palette with its old layout. The transition is visible either way,
// so the only question was whether it shows as an inconsistency of LOOK or of
// LAYOUT, and layout is the cheaper of the two.
//
// # What deletes this function
//
// The last screen moving. After that there is one theme, Dark goes, and naming
// it here stops being a choice worth recording.
func Current() fyne.Theme { return Visual{} }
