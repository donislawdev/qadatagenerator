package parts

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	toolkit "fyne.io/fyne/v2/theme"

	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// Flooded dresses a control the toolkit fills entirely with the focus colour,
// so that the label on it stays readable while the keyboard is there.
//
// # What the toolkit does, and why that is not enough here
//
// A focused button is answered at widget/button.go:379 with backgroundBlend
// set to ColorNameFocus, which button.go:341 then blends over ColorNameButton.
// The blend itself is at button.go:453 and reads src + dst - dst*srcA/0xFFFF,
// which for an opaque overlay is src. So the whole control arrives in the
// focus colour rather than tinted by it. The toolkit expects a tint: its own
// focus blue is #006CFF at an alpha of 0x2A, about a sixth of a coat, while
// every entry of our palette is opaque by the decision recorded beside it.
//
// The same colour is used the same way by widget/select.go:432 and
// widget/menu_item.go:295.
//
// Measured through the pixels of the render on 2026-09-05: the label arrived
// as #E8E8E8 on #4EC9F5, a contrast of 1.56 where WCAG 1.4.3 asks for 4.5.
// Eight states across the four screens reported that same number.
//
// # Why the colour of the label rather than the colour of the fill
//
// Because no fill answers both thresholds, and that is arithmetic rather than
// taste. To stand 3 to 1 against the resting button a fill needs a luminance
// of at least 0.1670, and #E8E8E8 can only be read on a luminance of at most
// 0.1404. The two do not meet. A darker fill would need a luminance of
// -0.0259, which is less than black has.
//
// Softening the focus colour with an alpha fails for a second and independent
// reason. The toolkit uses this one colour for three different jobs: a fill
// under text, a mark beside text at widget/check.go:380 and
// widget/radio_item.go:201, and an outline at widget/hyperlink.go:92. There is
// one alpha for all three. An alpha of 0x70 or less keeps a label readable and
// leaves the mark of a check box at 2.67 against the 3.0 it has to reach, and
// an alpha of 0x80 or more does the reverse.
//
// # Why a container rather than a renderer of our own
//
// A widget resolves a colour through the theme of the scope it is in, and the
// scope is what this changes. container.NewThemeOverride is the only way in
// from outside the toolkit, since cache.OverrideTheme lives in an internal
// package that nothing else can import. The toolkit does exactly this to a
// button of its own at container/innerwindow.go:385.
//
// Nothing here draws anything, so there is no renderer to go stale when the
// toolkit moves, and the control keeps its own drawing, hover, tap animation
// and icon.
func Flooded(control fyne.CanvasObject, focused func() bool) *container.ThemeOverride {
	return container.NewThemeOverride(control, readable{focused: focused})
}

// readable answers every colour the way the window's own theme does, except
// the one a flooded control paints its label with.
//
// The label is told a colour NAME rather than a colour: button.go:355 sets
// Style.ColorName on the text segment and leaves the segment to resolve it.
// So the answer has to come from the scope the label is in, which is why this
// is a theme and not a value written onto the control.
type readable struct {
	focused func() bool
}

var _ fyne.Theme = readable{}

func (r readable) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == toolkit.ColorNameForeground && r.focused != nil && r.focused() {
		// 🔴 The token rather than the old palette entry, changed 2026-09-09.
		// Both name the ink read on the toolkit's focus flood, but the flood
		// itself became mark.current when the window took the new theme, so
		// the entry describing it had to move to the ladder as well or it
		// would go on measuring a pair that is no longer on screen. It
		// measures 7.55 there, where a label needs 4.5.
		return palette.Ink(palette.TextOnMark)
	}
	return beneath().Color(name, variant)
}

func (readable) Font(style fyne.TextStyle) fyne.Resource { return beneath().Font(style) }

func (readable) Icon(name fyne.ThemeIconName) fyne.Resource { return beneath().Icon(name) }

func (readable) Size(name fyne.ThemeSizeName) float32 { return beneath().Size(name) }

// beneath is the theme this one changes a single answer of.
//
// 🔴 The application is asked for its theme directly rather than through
// theme.Current(). Current() answers with whatever theme the toolkit has
// pushed while it is rendering, and inside this container that is this theme,
// so asking it from here would call the method that asked, without end.
func beneath() fyne.Theme {
	if app := fyne.CurrentApp(); app != nil {
		if th := app.Settings().Theme(); th != nil {
			return th
		}
	}
	// A screen can be built with no application behind it, which is what a
	// screen built to be read rather than shown does. Nothing is drawn on that
	// path, so the answer goes unused, and the toolkit's own is the one that
	// cannot be wrong about itself.
	return toolkit.DefaultTheme()
}
