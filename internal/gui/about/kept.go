package about

import (
	"fyne.io/fyne/v2"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// Kept is what the window keeps on this computer between runs, as this screen
// needs it: where it is, and a way to forget it (DEC-325).
//
// It is handed in like the Saver, and it may be nil. A screen rendered into a
// picture or a test has no window keeping anything, and the section is then
// left out rather than drawn over a folder that is not there.
type Kept interface {
	Folder() string
	Forget()
}

// keptSection says what the window keeps and where, with the one button that
// forgets it.
//
// # Why on this screen
//
// It is the screen a person opens to ask what this program is and what it does
// with their machine, and the only other thing this window writes is what a
// person asks it to write, into a folder they chose. A window with a screen of
// settings would put this there. This one has no settings, so it stands beside
// the sentences about the person's data.
//
// # Why the button greys out rather than going away
//
// Pressed, it has done all it can for this run: the values are gone and the
// size will not be written as the window closes. A button that went away would
// move everything under it, and the sentence under it changes instead, so the
// screen says what happened in the place the person is looking.
func keptSection(text i18n.Catalogue, kept Kept) fyne.CanvasObject {
	line := quiet(text.T(i18n.AboutForgetWhat))
	var forget *parts.Button
	forget = parts.NewButton(text.T(i18n.AboutForget), func() {
		kept.Forget()
		line.SetText(text.T(i18n.AboutForgotten))
		forget.Disable()
	})
	forget.ShowIcon(paint.IconAgain)
	return section(text.T(i18n.AboutHeadingKept), "", paint.Column(palette.NameStepRelated,
		parts.Sentence(text.T(i18n.AboutKeptWhat)),
		form([]pair{{text.T(i18n.AboutKeptIn), kept.Folder()}}),
		paint.Row(palette.NameStepRelated, forget.OnScreen()),
		line))
}
