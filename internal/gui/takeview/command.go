package takeview

import (
	"strings"

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
// # Why this screen carries one at all
//
// docs/08 asks for it on every screen that produces something, and gives the
// reason in the same breath: without it the window and the command line stop
// being one product. Measured on 2026-09-08, nothing in internal/gui contained
// the words, so the rule had been written down for as long as the window had
// existed and nothing had ever held it.
//
// This screen is the one where it matters most. It carries ten settings, and
// somebody who has just worked out the ten they want is exactly the person who
// then has to put them into a pipeline.
func (v *View) buildCommand() {
	v.command = widget.NewLabelWithStyle("", fyne.TextAlignLeading,
		fyne.TextStyle{Monospace: true})
	// Cut with an ellipsis rather than wrapped or left to run. A command is one
	// long line by nature, and a label reports the width of its whole text as
	// the least it can be, so an uncut one drags the band holding it along.
	//
	// A cut line is honest here in a way it would not be elsewhere, because
	// nothing on this screen asks anybody to READ this line. The button beside
	// it carries the whole of it to the clipboard, and what stays visible is
	// the verb and the first flags, which is what tells a person whether the
	// line is the one they wanted.
	v.command.Wrapping = fyne.TextWrapOff
	v.command.Truncation = fyne.TextTruncateEllipsis
	v.copyCommand = parts.NewButton(v.text.T(i18n.CommandCopy), v.putCommandInClipboard)
	v.copyCommand.ShowIcon(paint.IconCopy)
	v.copyCommand.Confirmed = v.text.T(i18n.CommandCopied)
}

// commandBand is the band at the foot of the screen holding the line and the
// button that copies it, the same band the Quick screen has and for the same
// reason: the command is not a box floating over nothing, it is chrome
// touching the edge of the window, symmetrical to the band at the top.
func (v *View) commandBand() fyne.CanvasObject {
	return paint.Band(paint.BandAbove, palette.NameStepRelated, palette.NameStepMajor,
		container.NewBorder(nil, nil,
			paint.Middle(paint.Mono(commandPrompt, palette.TypeBody, palette.TextAccent)),
			paint.Middle(v.copyCommand.OnScreen()),
			paint.Middle(v.command)))
}

// commandPrompt is what a terminal shows before a line, so that the line reads
// as one and is not mistaken for a sentence about it.
const commandPrompt = "$"

// drawCommand writes the line out again, and hides the band while there is
// no file to write it about.
//
// A command with no file is a command that refuses, and a band offering one
// to copy would be offering exactly the thing the mode turns down.
func (v *View) drawCommand() {
	if v.terminal == nil {
		// Asked while the controls are still being built: choosing the
		// default of a list runs its callback, and the band does not exist
		// until every control does. Nothing to draw into yet.
		return
	}
	if v.path == "" {
		v.command.SetText("")
		v.terminal.Hide()
		return
	}
	v.command.SetText(v.commandLine())
	v.terminal.Show()
}

// commandLine writes out what a terminal would be asked for.
//
// # What is written and what is left out
//
// A setting the person did not touch is left out, because a flag carrying an
// empty value is not the same request as no flag: the first is refused and the
// second is what this screen actually did.
//
// The seed is the exception worth naming. It is written whenever there is one,
// because without it a run draws a fresh seed, and a line copied from here
// would then damage different rows from the ones this screen said it would.
// That is the one way this bridge could quietly lie.
func (v *View) commandLine() string {
	line := commandline.New(commandline.Corrupt).
		Flag(commandline.Layers, strings.Join(v.chosen.Layers, ",")).
		Flag(commandline.Rows, v.chosen.Rows).
		Flag(commandline.PerRow, v.chosen.PerRow).
		Flag(commandline.ExcludeFields, strings.Join(v.chosen.ExcludeFields, ",")).
		Flag(commandline.Delimiter, v.chosen.Delimiter).
		Flag(commandline.Locale, v.chosen.Locale).
		Flag(commandline.Seed, v.chosen.Seed).
		Flag(commandline.Out, v.chosen.Out).
		Flag(commandline.Manifest, v.chosen.Manifest).
		Switch(commandline.Overwrite, v.chosen.Overwrite)
	return line.Argument(v.path).String()
}

// putCommandInClipboard copies the line and says that it did.
//
// Saying so is not decoration, for the reason the screen next door gives: a
// clipboard cannot be seen, so a button that silently succeeded and one that
// silently did nothing look the same.
func (v *View) putCommandInClipboard() {
	if v.clipboard == nil || v.command.Text == "" {
		return
	}
	v.clipboard.SetContent(v.command.Text)
	v.copyCommand.Confirm()
}
