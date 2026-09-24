package fynetheme

import "fyne.io/fyne/v2"

// Current is the theme this window is drawn with today.
//
// # Why one function rather than each place naming a theme
//
// Many places install a theme: the product, the probe that renders every screen
// to a picture, the probe that walks the keyboard, the mock, and the tests of
// several packages. Every one of them is asking the same question, which is
// what the WINDOW looks like, and every one of them answered it by naming a
// type. A transition between two themes then has all of them to change and no
// way to tell that one was missed, because a probe left on the old theme renders a
// picture that simply looks like the old window. That is the shape of defect
// this project keeps finding: a register kept in more than one copy, where the
// copies stay green while they drift.
//
// # Why it stays now that there is one theme
//
// It was written for the move from Dark to Visual (DEC-233), with a note that
// the last screen moving would delete it. Dark went with DEC-253 and this
// stayed, because the argument above does not depend on how many themes exist:
// every place naming Visual is a place a later change of theme has to find. A test OF a theme still names that theme, since it asks what that type
// answers rather than what the window wears.
func Current() fyne.Theme { return Visual{} }
