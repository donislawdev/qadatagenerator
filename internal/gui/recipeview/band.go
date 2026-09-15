package recipeview

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// The band along the top: where a run writes, whether it may replace what is
// there, and the one action of the screen.
//
// # Why along the top rather than under the answer
//
// The answer grows with the recipe. A table of a wide entity, eleven problems,
// two lists of what will not happen: all of them push whatever is under them
// further down, and the one thing a person came to press would be the thing
// furthest away. That is where it was until 2026-09-15, and at the size the
// owner keeps the window it was below the fold (B-280). The band is where the
// Quick screen keeps what a run needs and its action, and the take screen the
// file and its action, so the three screens that produce read the same way.

// buildBand makes the controls once.
//
// Once rather than on each draw, because a control rebuilt under somebody is a
// control that loses what they typed and where their keyboard was standing.
func (v *View) buildBand() {
	v.out = parts.NewFieldEntry()
	// 🔴 A short hint rather than the sentence about the directory. Rendering
	// showed the two side by side saying the same thing twice, and a
	// placeholder is meant to be an example of what goes in the field rather
	// than a second copy of the instruction beside it.
	v.out.SetPlaceHolder(v.text.T(i18n.RecipeDirHint))
	v.out.OnChanged = func(string) { v.settingChanged() }
	v.choose = parts.NewButton(v.text.T(i18n.RecipeChooseDir), v.chooseDirectory)
	v.choose.ShowIcon(paint.IconFolder)
	// Replacing what is already there is off until somebody says otherwise,
	// and that is the same answer the command line gives. A run that quietly
	// wrote over a file in somebody's working directory is the failure the
	// writer of files exists to prevent, and a window is where an accidental
	// second press is easiest.
	v.replace = parts.NewCheck(v.text.T(i18n.RecipeReplace), func(bool) { v.settingChanged() })
	// The one action this screen is for, and the only filled button on it. Its
	// fill stands 4.99 against the window ground, so unlike every outlined
	// button beside it, it needs nothing added to be found.
	v.generate = parts.NewPrimaryButton(v.text.T(i18n.RecipeGenerate), v.Start)
	v.stop = parts.NewButton(v.text.T(i18n.RecipeStopRun), v.Stop)
	// The button that chooses a directory is the one control of the band
	// always in the tree, which is what the keyboard needs to find the canvas
	// through and to fall back onto.
	v.keys = parts.Handover{Anchor: v.choose}
}

// settingsBand is the band: what the screen is for, then the directory with
// the two controls beside it, and the one action at the right, dropped by the
// height of the label it does not have so that it stands level with the field.
//
// The field takes what the controls beside it leave, because a path is the one
// value here nobody can retype from memory. Everything else takes the width it
// asks for.
func (v *View) settingsBand() fyne.CanvasObject {
	drop := paint.Space(0, palette.TypeSize(palette.TypeLabel)+palette.StepOf(palette.NameStepHairline))
	out := labelled(v.text.T(i18n.RecipeWriteTo), parts.Standing(parts.Field(v.out)))
	beside := paint.PadLeft(palette.NameStepRelated, paint.Row(palette.NameStepRelated,
		container.NewVBox(drop, v.choose.OnScreen()),
		container.NewVBox(drop, paint.AtLeast(0, palette.ControlHeight, v.replace.OnScreen()))))
	settings := container.New(layout.NewBorderLayout(nil, nil, nil, beside), out, beside)
	action := paint.PadLeft(palette.NameStepGroup, container.NewVBox(drop, v.action))
	// The sentence wraps, because it is the longest line on this screen and a
	// piece of canvas text would otherwise decide how narrow the window may be.
	return paint.Band(paint.BandBelow, palette.NameStepGroup, palette.NameStepMajor,
		paint.Column(palette.NameStepGroup,
			v.paragraph(v.text.T(i18n.PurposeRecipe)),
			container.New(layout.NewBorderLayout(nil, nil, nil, action), settings, action)))
}

// labelled names a control, above it rather than beside it.
//
// It is one call to the drawing next door, named here so that the band reads
// as a list of settings rather than as layout instructions.
func labelled(name string, control fyne.CanvasObject) fyne.CanvasObject {
	return paint.FieldLabel(name, control)
}

// settingChanged redraws what depends on the directory and on whether files
// may be replaced: the card, the button and the terminal line.
//
// 🔴 What the last run did is let go of the moment a setting changes, the way
// the take screen lets go of it (DEC-245, DEC-246). The card says one thing:
// what pressing the button now would do, or what the last press did, whichever
// is newer, and a result kept beside settings it was not made with would
// belong to neither.
func (v *View) settingChanged() {
	if v.cardBox == nil {
		// Asked while the controls are still being built, before there is a
		// card to draw into.
		return
	}
	if !v.busy() {
		v.forgetLastRun()
	}
	v.draw()
}

// chooseDirectory asks for a directory the way the system asks for one.
//
// A field alone would be enough for somebody who knows the path they want, and
// is a poor answer for the person this screen was built for. The field stays,
// because a path pasted from somewhere else is the fastest way in for the other
// half of the people who use this.
func (v *View) chooseDirectory() {
	if v.window == nil || v.busy() {
		return
	}
	dialog.ShowFolderOpen(func(chosen fyne.ListableURI, err error) {
		if err != nil || chosen == nil {
			return
		}
		v.out.SetText(chosen.Path())
	}, v.window)
}

// SetOut puts a directory in the field, without anybody typing it.
//
// Exported for the reason SetRecipe is: the guard holding what every screen
// draws and the probe that photographs each one both have to reach the state
// where the button can be pressed, and reaching it through the field a person
// types into is what keeps those pictures pictures of a state somebody can get
// to.
func (v *View) SetOut(into string) { v.out.SetText(into) }
