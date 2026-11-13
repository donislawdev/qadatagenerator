package paint

import (
	"image/color"
	"strconv"

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
//	Card  - the result. A ground one step off the window, rounded, no edge.
//	Band  - chrome. At the top of a screen it is the header and has no ground
//	        of its own, at the foot it is the ground of the sidebar.
//	Pill  - a badge on data. Rounded to a capsule, in a colour that MEANS.
//
// 🔴 Fewer lines, by the owner's choice on 2026-10-21 (DEC-261). Every card,
// every field, every button and every section heading had an edge, and the
// window read as a drawing of boxes rather than as a tool. A card is now told
// apart from the window by its ground alone, and an edge is kept for what a
// person presses or types into.

// Card is the one raised surface of a screen.
//
// The padding is named by the caller because a card of rows and a card of
// paragraphs want different room, and neither is more correct than the other.
func Card(padding string, content fyne.CanvasObject) fyne.CanvasObject {
	return On(CardGround(), PadAll(padding, content))
}

// CardGround is the ground of a card without what stands on it, for a card
// whose rows something else places: the list of values on the quick screen
// builds only the rows in view and puts them on this itself.
func CardGround() *canvas.Rectangle { return Fill(theme.BgSurface, theme.RadiusCard) }

// Band is a strip of chrome touching the left and right edges of the window.
//
// # Why the top one has no ground and the bottom one has no line
//
// Until 2026-10-21 both had the ground of a card and a line along the content.
// The band at the top then sat under the title of the screen as a second,
// lighter strip, so the name of a screen hung above the words explaining it
// rather than standing with them, and the window had three greys stacked in
// stripes (DEC-261). The band at the top is now the rest of the header: the
// same ground as the title, with the room under it saying where the content
// starts. The band at the bottom keeps its ground, the same as the sidebar,
// and the ground alone says it is chrome.
func Band(edge string, padding, sides string, content fyne.CanvasObject) fyne.CanvasObject {
	inside := Pad(padding, padding, sides, sides, content)
	if edge == BandBelow {
		return inside
	}
	return On(Fill(theme.BgSurface, 0), inside)
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

// SectionHeading names one part of a screen, with room at the far end for the
// actions belonging to that part.
//
// # Why no line any more
//
// A line ran from the words to the far edge until 2026-10-21, to say where one
// section ends. With a card under most headings and an edge on every control,
// it was one more line in a window the owner found chaotic (DEC-261). The room
// above a heading says where a section starts, and that is what the gap of a
// group is for.
func SectionHeading(words string, actions fyne.CanvasObject) fyne.CanvasObject {
	return container.NewBorder(nil, nil,
		Middle(Strong(words, theme.TypeSection, theme.TextPrimary)), actions, Spacer())
}

// StepHeading names one step of a screen that is read from the top down, with
// the number of the step in a disc before the words.
//
// # Why numbers, added on 2026-10-21 (DEC-261)
//
// The owner looked at the window as somebody seeing it for the first time and
// could not tell where to start: the action of a screen stood in its top right
// corner and the settings were one flat list. A number says the order before a
// word is read, and the last step is where the button is.
func StepHeading(number int, words string) fyne.CanvasObject {
	return Row(theme.NameStepTight, Middle(StepMark(number)),
		Middle(Strong(words, theme.TypeSection, theme.TextPrimary)))
}

// Step is one numbered step of a screen: the words of its heading and what
// stands under them.
type Step struct {
	Words string
	Body  fyne.CanvasObject
}

// Steps stands numbered steps one under another, the gap between them named by
// the caller.
//
// Each is numbered by its place in the list rather than by a number written
// beside it, so a step added, removed or moved renumbers the ones after it, and
// a view writes no number at all.
func Steps(gap string, steps ...Step) *fyne.Container {
	column := make([]fyne.CanvasObject, 0, len(steps))
	for at, step := range steps {
		column = append(column, Column(theme.NameStepRelated, StepHeading(at+1, step.Words), step.Body))
	}
	return Column(gap, column...)
}

// StepHeadings is the headings of numbered steps that do not stand one under
// another, numbered by their place in the list for the reason Steps gives.
func StepHeadings(words ...string) []fyne.CanvasObject {
	headings := make([]fyne.CanvasObject, 0, len(words))
	for at, said := range words {
		headings = append(headings, StepHeading(at+1, said))
	}
	return headings
}

// StepMark is the disc with the number of a step in it, on its own, for a step
// whose first control says what it is better than a heading would: the two
// buttons opening the two files of the screen that checks a result.
func StepMark(number int) fyne.CanvasObject {
	digit := container.NewCenter(Strong(strconv.Itoa(number), theme.TypeLabel, theme.TextAccent))
	return AtLeast(theme.StepDisc, theme.StepDisc,
		container.NewStack(Fill(theme.BgCurrent, theme.RadiusPill), digit))
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
