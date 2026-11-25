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
// of the file left as a filled in example until there is a file, and the copy
// beside it cannot be pressed until then. Turned down was hiding the line until
// there is a path, which takes the bridge away from exactly the person who is
// learning what a recipe is, and turned down was writing a path this window
// guessed, which hands somebody a command naming a file that may not exist.
//
// Since 2026-11-25 the screen opens and saves recipe files (file.go), so the
// line names the real file once there is one and the copy can be pressed. The
// sentence under it says which of the three it is: an example, the file as it
// is on the screen, or the file without the changes not saved yet.
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
	v.copyCommand = parts.NewButton(v.text.T(i18n.CommandCopy), v.putCommandInClipboard)
	v.copyCommand.ShowIcon(paint.IconCopy)
	v.copyCommand.Confirmed = v.text.T(i18n.CommandCopied)
	v.copyCommand.Disable()
	v.commandNote = paint.Text("", palette.TypeLabel, palette.TextSecondary)
}

// CopyCommandTo is where the copy button puts the line, handed in after the
// screen is built, like the layers it offers, so that the many callers that
// build this screen to look at it need not name a clipboard they never use.
func (v *View) CopyCommandTo(clipboard Clipboard) { v.clipboard = clipboard }

// putCommandInClipboard copies the line and says that it did, as the take
// screen does: what is in a clipboard cannot be seen, so the button says it.
func (v *View) putCommandInClipboard() {
	if v.clipboard == nil || v.kept.Path == "" {
		return
	}
	v.clipboard.SetContent(v.commandText.Text)
	v.copyCommand.Confirm()
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
		paint.Column(palette.NameStepHairline, line, v.commandNote))
}

// commandPrompt is the mark a terminal writes before a line, so that the
// monospace line beside it reads as a command rather than a value.
const commandPrompt = "$"

// drawCommand writes the terminal line out again.
//
// The settings that ARE real here are written as they stand: the directory
// somebody typed, whether they ticked replace, and the extra form of the
// manifest they picked, which is --manifest in a terminal and is left out when
// they picked none. The file is the one the recipe came from or was saved to,
// and an example name while there is none. It moves with the settings, so it
// is written on every draw rather than once when the screen is built - a bridge
// showing the directory somebody typed two edits ago is worse than none.
func (v *View) drawCommand() {
	file, note := v.text.T(i18n.RecipeCommandFile), i18n.RecipeCommandExample
	switch {
	case v.kept.Path != "" && v.unsaved():
		file, note = v.kept.Path, i18n.RecipeCommandUnsaved
	case v.kept.Path != "":
		file, note = v.kept.Path, i18n.RecipeCommandSaved
	}
	v.commandText.SetText(commandline.New(commandline.Generate).
		Flag(commandline.Out, v.out.Text).
		Switch(commandline.Overwrite, v.replace.Checked).
		Flag(commandline.Manifest, v.form).
		Argument(file).
		String())
	v.commandNote.Text = v.text.T(note)
	v.commandNote.Refresh()
	// Alive once there is a file, whether or not a clipboard was handed in, so
	// a picture of the screen shows the button a person sees.
	if v.kept.Path != "" {
		v.copyCommand.Enable()
	} else {
		v.copyCommand.Disable()
	}
}
