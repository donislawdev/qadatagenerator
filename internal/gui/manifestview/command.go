package manifestview

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
	"github.com/donislawdev/qadatagenerator/internal/commandline"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// buildCommand makes the line that would do this in a terminal, once.
//
// # Why this screen carries one at all
//
// docs/08 asks for it on every screen with an equivalent at the other surface,
// and gives the reason in the same breath: without it the window and the
// command line stop being one product. This screen does not produce anything,
// but its second state IS a verb: a manifest beside a response record is what
// qadg verify reads, and the person who has just seen where the two disagree
// is exactly the person who then has to put that comparison into a pipeline.
//
// The line is hidden while there is no manifest, the way the take screen hides
// its line while there is no file: a command naming no file is a command that
// refuses. With a manifest and no response the response is an example, the
// copy cannot be pressed, and the sentence under the line says which part is
// the example, which is what the owner settled for the recipe screen on
// 2026-09-08 and for the same reason: a command carrying an example name fails
// when pasted, and saying which part is the example turns a line nobody can
// run into a line somebody can finish.
func (v *View) buildCommand() {
	v.command = widget.NewLabelWithStyle("", fyne.TextAlignLeading,
		fyne.TextStyle{Monospace: true})
	// Cut with an ellipsis rather than wrapped or left to run. A command is one
	// long line by nature, and a label reports the width of its whole text as
	// the least it can be, so an uncut one drags the band holding it along.
	// Nothing here asks anybody to READ the line: the button beside it carries
	// the whole of it to the clipboard.
	v.command.Wrapping = fyne.TextWrapOff
	v.command.Truncation = fyne.TextTruncateEllipsis
	v.copyCommand = parts.NewButton(v.text.T(i18n.CommandCopy), v.putCommandInClipboard)
	v.copyCommand.ShowIcon(paint.IconCopy)
	v.copyCommand.Confirmed = v.text.T(i18n.CommandCopied)
	v.commandNote = paint.Text(v.text.T(i18n.ManifestCommandExample), palette.TypeLabel, palette.TextSecondary)
}

// commandBand is the band at the foot holding the line and the button that
// copies it, the same band the other screens carry: chrome touching the edge
// of the window, symmetrical to the band at the top.
func (v *View) commandBand() fyne.CanvasObject {
	line := container.NewBorder(nil, nil,
		paint.Middle(paint.Mono(commandPrompt, palette.TypeBody, palette.TextAccent)),
		paint.Middle(v.copyCommand.OnScreen()),
		paint.Middle(v.command))
	return paint.Band(paint.BandAbove, palette.NameStepRelated, palette.NameStepMajor,
		paint.Column(palette.NameStepHairline, line, v.commandNote))
}

// commandPrompt is what a terminal shows before a line, so that the line reads
// as one and is not mistaken for a sentence about it.
const commandPrompt = "$"

// drawCommand writes the line out again, and hides the band while there is
// no manifest to write it about.
//
// The manifest is named by its path when it came from one, so that the line
// can be pasted as it stands, and by the name a run gives the file when it
// arrived already parsed, which is what the pictures of this screen are of.
func (v *View) drawCommand() {
	if v.terminal == nil {
		return
	}
	if !v.hasPromise {
		v.command.SetText("")
		v.terminal.Hide()
		return
	}
	manifestName := v.manifestPath
	if manifestName == "" {
		manifestName = manifestjson.Name
	}
	responseName := v.responsePath
	if !v.hasResponse || responseName == "" {
		responseName = v.text.T(i18n.ManifestCommandResponse)
	}
	v.command.SetText(commandline.New(commandline.Verify).
		Argument(manifestName).
		Argument(responseName).
		String())
	if v.hasResponse && v.responsePath != "" {
		v.copyCommand.Enable()
		v.commandNote.Hide()
	} else {
		v.copyCommand.Disable()
		v.commandNote.Show()
	}
	v.terminal.Show()
}

// putCommandInClipboard copies the line and says that it did.
//
// Saying so is not decoration: a clipboard cannot be seen, so a button that
// silently succeeded and one that silently did nothing look the same.
func (v *View) putCommandInClipboard() {
	if v.clipboard == nil || v.command.Text == "" || !v.hasResponse {
		return
	}
	v.clipboard.SetContent(v.command.Text)
	v.copyCommand.Confirm()
}
