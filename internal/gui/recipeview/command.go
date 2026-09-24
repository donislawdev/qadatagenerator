package recipeview

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/commandline"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// buildCommand makes the line that would do this in a terminal, once.
//
// # Why this screen carries one at all, and the way it differs from the others
//
// docs/08 asks for it on every screen that produces something. The other two
// produce from what is ON them: the quick screen has a generator and a locale,
// the take screen has a file it was handed. This one produces from a FILE, and
// the file is the thing it does not have. What a person types here is text in a
// field, and the command line takes a path.
//
// The owner settled it on 2026-09-08: the line is always there, with the name
// of the file left as a filled in example, and the copy beside it cannot be
// pressed. Turned down was hiding the line until there is a path, which takes
// the bridge away from exactly the person who is learning what a recipe is, and
// turned down was writing a path this window guessed, which hands somebody a
// command naming a file that may not exist.
//
// 🔴 Worth knowing before changing this: nothing in this build writes a recipe
// file, so there is no state this screen can reach where the copy becomes
// pressable. It is disabled in all of them, and the sentence beside it says so
// in words. The day this screen learns to save a recipe is the day the button
// has something to become.
func (v *View) buildCommand() {
	v.commandText = widget.NewLabelWithStyle("", fyne.TextAlignLeading,
		fyne.TextStyle{Monospace: true})
	// Cut rather than wrapped, for the reason the take screen measured: a label
	// reports the width of its whole text as the least it can be, so an uncut
	// command drags the band holding it wider than the window. Nothing here
	// asks anybody to read the line to its end - the button carries the whole of
	// it and what stays visible is the verb and the first flags.
	v.commandText.Wrapping = fyne.TextWrapOff
	v.commandText.Truncation = fyne.TextTruncateEllipsis
	// No clipboard is taken for this screen, because a control that cannot be
	// pressed cannot copy. Adding the port now would be a wire to nowhere, and
	// the screens that do copy show exactly what to add when it is needed.
	v.copyCommand = parts.NewButton(v.text.T(i18n.CommandCopy), func() {})
	v.copyCommand.ShowIcon(paint.IconCopy)
	v.copyCommand.Disable()
}

// commandBand is the band at the foot: the mark of a terminal, the line, the
// button that would copy it, and under them the sentence saying why the button
// is the way it is.
//
// It is the same band the Quick and take screens carry, so the three read as
// one program: chrome touching the edge of the window, symmetrical to the band
// at the top. The sentence is not an apology. A command carrying an example
// file name is a command that fails when pasted, and saying which part is the
// example is what turns a line nobody can run into a line somebody can finish.
func (v *View) commandBand() fyne.CanvasObject {
	line := container.NewBorder(nil, nil,
		paint.Middle(paint.Mono(commandPrompt, palette.TypeBody, palette.TextAccent)),
		paint.Middle(v.copyCommand.OnScreen()),
		paint.Middle(v.commandText))
	return paint.Band(paint.BandAbove, palette.NameStepRelated, palette.NameStepMajor,
		paint.Column(palette.NameStepHairline,
			line,
			paint.Text(v.text.T(i18n.RecipeCommandExample), palette.TypeLabel, palette.TextSecondary)))
}

// commandPrompt is the mark a terminal writes before a line, so that the
// monospace line beside it reads as a command rather than a value.
const commandPrompt = "$"

// drawCommand writes the terminal line out again.
//
// The settings that ARE real here are written as they stand: the directory
// somebody typed, whether they ticked replace, and the extra form of the
// manifest they picked, which is --manifest in a terminal and is left out when
// they picked none. Only the file is an example,
// which is what keeps the rest of the line worth reading. It moves with the
// settings, so it is written on every draw rather than once when the screen is
// built - a bridge showing the directory somebody typed two edits ago is worse
// than none.
func (v *View) drawCommand() {
	v.commandText.SetText(commandline.New(commandline.Generate).
		Flag(commandline.Out, v.out.Text).
		Switch(commandline.Overwrite, v.replace.Checked).
		Flag(commandline.Manifest, v.form).
		Argument(v.text.T(i18n.RecipeCommandFile)).
		String())
}
