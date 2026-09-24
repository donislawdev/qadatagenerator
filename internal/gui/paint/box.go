package paint

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"

	theme "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// The three surfaces this window is made of, and the rule that keeps them
// apart.
//
// 🔴 There is ONE card on a screen and it is the result. That was a correction
// made on the pixels rather than a preference: while the strip of settings was
// also drawn as a card, the screen had two identical frames and nothing said
// which one was the tool and which one was the content. Turning the settings
// into a band of chrome leaves exactly one framed thing, and it is the thing
// the screen is for.
//
//	Card  - the result. A ground one step off the window, an edge, rounded.
//	Band  - chrome. The same ground, touching the edges, square, a line under.
//	Pill  - a badge on data. Rounded to a capsule, in a colour that MEANS.

// Card is the one framed surface of a screen.
//
// The padding is named by the caller because a card of rows and a card of
// paragraphs want different room, and neither is more correct than the other.
func Card(padding string, content fyne.CanvasObject) fyne.CanvasObject {
	ground := Bordered(theme.BgSurface, theme.LineQuiet, theme.RadiusCard)
	return On(ground, PadAll(padding, content))
}

// Band is a strip of chrome touching the left and right edges of the window.
//
// # Why it is square and has a line rather than a frame
//
// A band is not content, so it must not look like content. Rounding it would
// leave a sliver of window showing in each corner, which reads as a shape that
// failed to fit, and framing it would make a second card. A single line along
// the edge it meets the content on is the whole separation, and it is the
// treatment every application with a toolbar uses.
//
// The line goes under a band at the top of a screen and over one at the bottom,
// which is what edge names.
func Band(edge string, padding, sides string, content fyne.CanvasObject) fyne.CanvasObject {
	body := On(Fill(theme.BgSurface, 0), Pad(padding, padding, sides, sides, content))
	if edge == BandBelow {
		return container.NewBorder(nil, Hairline(theme.LineQuiet), nil, nil, body)
	}
	return container.NewBorder(Hairline(theme.LineQuiet), nil, nil, nil, body)
}

// Which side of a band the line separating it from the content is drawn on.
const (
	// BandBelow is a band at the top of a screen, with content under it.
	BandBelow = "below"
	// BandAbove is a band at the bottom of a screen, with content over it.
	BandAbove = "above"
)

// Pill is a badge that carries a verdict about the data beside it.
//
// # Why a badge exists at all, when a sentence would say the same
//
// A person who has never seen this program cannot tell a value that is safe to
// paste into a real system from one that is not, and that difference is the
// single most important thing this screen knows. A sentence says it to somebody
// already reading. A badge says it to somebody scanning.
//
// # Why the ground is the token colour made translucent
//
// The three verdict colours are measured as TEXT on the window. Filling a badge
// with one of them and writing on it would be a different measurement that
// nothing here has made. A wash of the same hue under text of the full colour
// keeps the measured pair intact, because the text is still read against a
// ground barely off the window.
func Pill(words, token, icon string) fyne.CanvasObject {
	ground := canvas.NewRectangle(wash(token))
	ground.CornerRadius = theme.RadiusPill

	row := []fyne.CanvasObject{}
	if icon != "" {
		row = append(row, Middle(Icon(icon, token, theme.IconSmall)))
	}
	row = append(row, Middle(Strong(words, theme.TypeLabel, token)))

	return On(ground, Pad(theme.NameStepHairline, theme.NameStepHairline,
		theme.NameStepTight, theme.NameStepTight,
		Row(theme.NameStepHairline, row...)))
}

// SectionHeading names one part of a screen, with a line running from the words
// to the far edge and room at the end for the actions belonging to that part.
//
// # Why the line rather than size alone
//
// The heading and the rows under it are two rungs apart on the type ladder, and
// two rungs is not enough on its own to say where one section ends and the next
// begins. The line does that in a way no font size can, and it costs nothing
// but a pixel.
func SectionHeading(words string, actions fyne.CanvasObject) fyne.CanvasObject {
	rule := Pad(theme.NameStepHairline, theme.NameStepHairline,
		theme.NameStepRelated, theme.NameStepRelated,
		Middle(Hairline(theme.LineSection)))
	return container.NewBorder(nil, nil,
		Middle(Strong(words, theme.TypeSection, theme.TextPrimary)), actions, rule)
}

// FieldLabel names a control, above it rather than beside it.
//
// # Why above
//
// The guidance of the IntelliJ Platform says it plainly for the case this
// window has: controls whose labels are of similar length go on separate lines
// with their input boxes aligned on the left. This screen has four labels of
// four to eight characters, and beside a control they were right aligned, which
// opens a gap between a label and the thing it names and is the single detail
// that dates an interface to the nineteen nineties.
func FieldLabel(words string, control fyne.CanvasObject) fyne.CanvasObject {
	return Column(theme.NameStepHairline, fieldLabel(words), control)
}

// Unlabelled stands a control that has no label of its own level with the
// controls beside it that FieldLabel put under theirs.
//
// # Why the room is measured on the label rather than read off the ladder
//
// It was read off the ladder until 2026-09-24: the size of the label's type
// plus the hairline step, written out in three copies on three screens. A line
// of text is taller than the size of its type, because it carries the ascent
// and the descent, so every control without a label stood four units higher
// than the field beside it. The live window showed it and no test did (B-310).
// The room is now the height of the very object FieldLabel draws, in the same
// column with the same gap, so the two cannot drift apart whatever the ladder
// or the font does next.
func Unlabelled(control fyne.CanvasObject) fyne.CanvasObject {
	return Column(theme.NameStepHairline,
		Space(0, fieldLabel("").MinSize().Height), control)
}

// fieldLabel is the words above a control, shared so that the label and the
// room left where a label is missing are one object measured one way.
func fieldLabel(words string) *canvas.Text {
	return Text(words, theme.TypeLabel, theme.TextSecondary)
}

// Spacer takes the room left over on the axis it is laid out along.
//
// It is the toolkit's own, named here so that a screen composing a row does not
// have to import a second layout package for one call.
func Spacer() fyne.CanvasObject { return layout.NewSpacer() }

// wash is a token colour at the alpha a badge ground is drawn with.
//
// # Why an alpha at all, when every colour in the theme is opaque
//
// The theme holds colours and every one is fully opaque by decision, because a
// translucent entry would make its measured contrast a statement about a blend
// nobody measured. This is not another colour: it is one of them drawn faintly
// UNDER text of the full colour, so the pair that was measured, the text and
// the window, is still the pair on screen.
//
// What would change that: a wash dark or light enough to become the ground the
// text is read on. At the alpha the theme states, over these grounds, it is not.
func wash(token string) color.NRGBA {
	ink := theme.Ink(token)
	ink.A = theme.BadgeWashAlpha
	return ink
}
