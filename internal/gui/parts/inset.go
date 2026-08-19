package parts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"

	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// Inset keeps content off the edges of the window by the margin of the scale.
//
// # Why this is shared rather than copied into each screen
//
// It began inside the manifest browser, where the first drawing had no margin
// at all and every word began at the frame. The About screen then wanted the
// same thing and could not have it, which is how a probe rendering both screens
// found them looking different for no reason anybody chose.
//
// A third screen is what settled where it lives. Two copies are a coincidence
// and three are a habit, and the habit here would be each screen deciding its
// own margin, which is precisely the drift the scale of spacings exists to
// prevent.
//
// # Why it is applied once around the whole screen
//
// Around each part instead, two of them stacked read as a gap somebody forgot
// to close. The margin separates the window from the thing inside it, and there
// is one of each.
func Inset(content fyne.CanvasObject) fyne.CanvasObject {
	margin := palette.StepOf(palette.NameStepSection)
	return container.New(layout.NewCustomPaddedLayout(margin, margin, margin, margin), content)
}
