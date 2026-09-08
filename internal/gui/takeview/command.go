package takeview

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/commandline"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
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
	// the least it can be, so an uncut one drags the pane holding it along.
	// Measured on 2026-09-08 once the pane was no longer being stretched by
	// something else: cut, the line renders 513 pixels wide inside a pane of
	// 522, which is what it should do.
	//
	// A cut line is honest here in a way it would not be elsewhere, because
	// nothing on this screen asks anybody to READ this line. The button beside
	// it carries the whole of it to the clipboard, and what stays visible is
	// the verb and the first flags, which is what tells a person whether the
	// line is the one they wanted.
	//
	// 🔴 A note for whoever measures this next, because it cost an hour. When
	// the pane was 1998 wide, this label rendered 1515 wide and cutting it
	// looked like it did nothing. It was doing exactly what it says: there was
	// simply room. The pane was being stretched by the row of tick boxes above,
	// which is written up where that row is built. A remedy tried against the
	// wrong cause looks like a remedy that does not work.
	v.command.Wrapping = fyne.TextWrapOff
	v.command.Truncation = fyne.TextTruncateEllipsis
	v.copyCommand = parts.NewButton(v.text.T(i18n.CommandCopy), v.putCommandInClipboard)
}

// commandContent is that line with the button that copies it.
func (v *View) commandContent() fyne.CanvasObject {
	v.command.SetText(v.commandLine())
	// 🔴 The line sits in the middle of a border and nowhere else. Measured on
	// 2026-09-08 with it in a plain column: the text rendered as one unbroken
	// run 1515 pixels wide and pushed the pane holding it to 1998 inside a
	// window of 1342, so the description of the file beside it lost a third of
	// its width and every paragraph in it rewrapped. Wrapping was set on the
	// label and did nothing, which is the trap written up at beside() below:
	// a wrapping label asks for the width of its whole sentence, and only the
	// middle of a border is given what is left rather than what it asks for.
	return container.NewVBox(
		v.command,
		container.NewHBox(v.copyCommand.OnScreen(), layout.NewSpacer()),
	)
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
}
