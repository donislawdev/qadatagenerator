package quickview

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
	"github.com/donislawdev/qadatagenerator/internal/quick"
)

// The shape of this screen, drawn out of the primitives rather than out of the
// stock widgets, which is what DEC-228 settled and DEC-232 built the materials
// for.
//
// # What changed on 2026-09-09 and why the old shape was wrong
//
// The screen was a sentence, a form and a bordered box, in two equal columns.
// Seen at the size the owner keeps the window, that arrangement spent the whole
// lower half on nothing and drew the settings and the result as two frames of
// the same weight, so nothing on it said which one was the tool and which one
// was the thing the screen is for.
//
// What it is now follows the drawing that was accepted: a band of chrome
// carrying the settings, ONE card carrying the values, and a band at the foot
// carrying the line that does the same thing in a terminal. The card is the
// only framed thing, so it is the thing the eye lands on.

// shape assembles the whole screen out of the three parts above.
//
// 🔴 The objects are handed to the layout in the order the keyboard should walk
// them, which is not the order container.NewBorder would produce. That helper
// appends the CENTRE first and the edges after it, so a screen built with it
// would have the keyboard reach the values and their buttons before the
// settings standing above them. The layout finds its edges by comparing the
// objects it was given, so passing them in reading order costs nothing and is
// the whole difference between a walk that follows the screen and one that
// jumps.
//
// This project has paid for that once already, on the screen next door, where
// the button at the top was the last stop of the keyboard.
func (v *View) shape(settings, body, command fyne.CanvasObject) fyne.CanvasObject {
	return container.New(layout.NewBorderLayout(settings, command, nil, nil),
		settings, body, command)
}

// settingsBand is the strip of chrome every value on this screen is asked for
// with.
//
// # Why the promise stands inside the band rather than above it
//
// It is one sentence about what this screen does NOT do, and the drawing put it
// in a band of its own under the title. Two bands stacked draw two rules across
// the window a few pixels apart, which reads as a mistake rather than as a
// separation. One surface carrying both is one rule, and the sentence belongs
// to the settings anyway: it says what pressing them will not cost you.
func (v *View) settingsBand(fields fyne.CanvasObject) fyne.CanvasObject {
	return paint.Band(paint.BandBelow, palette.NameStepGroup, palette.NameStepMajor,
		paint.Column(palette.NameStepGroup,
			paint.Text(v.text.T(i18n.QuickNothingWritten), palette.TypeBody, palette.TextSecondary),
			fields))
}

// commandBand is the line that would ask a terminal for what is on screen.
//
// # Why a band at the foot rather than a panel in the column
//
// It answers a question somebody asks AFTER they have what they came for, so it
// must not compete with the values for the middle of the screen. A band at the
// foot is also symmetrical with the settings at the head, which is what makes
// the two read as chrome and the card between them as content.
func (v *View) commandBand() fyne.CanvasObject {
	return paint.Band(paint.BandAbove, palette.NameStepRelated, palette.NameStepMajor,
		container.NewBorder(nil, nil,
			paint.Middle(paint.Mono(commandPrompt, palette.TypeBody, palette.TextAccent)),
			paint.Middle(v.copyCommand.OnScreen()),
			paint.Middle(v.command)))
}

// commandPrompt is the mark a terminal writes before a line somebody types.
//
// It is not translated and it is not a word: it is the one character that says
// the monospace line beside it is a command rather than a value, which is the
// only thing on this screen that could be mistaken for one.
const commandPrompt = "$"

// valuesCard is the one framed thing on this screen.
//
// # Why the rows share the height rather than each taking its own
//
// Measured while the drawing was made: ten values pinned to their smallest
// height left a few hundred pixels of gap under the card at the size this window
// is kept, and a card stretched to swallow that gap left the same gap INSIDE
// itself, which reads as a fault rather than as room. Rows that grow spend the
// space on the thing the screen is for, and the ceiling in the theme is what
// stops four values in a tall window becoming four paragraphs.
func (v *View) valuesCard(values []string) fyne.CanvasObject {
	rows := make([]fyne.CanvasObject, 0, len(values))
	// The width of the widest number there will be, so that a one and a ten
	// line up under each other. Derived rather than written down: the face is
	// monospace, so padding to the digits of the LAST index is exactly the
	// column every index needs, and a list of a hundred widens itself.
	digits := len(strconv.Itoa(len(values)))
	// Toggled rather than tested for an even index, and not only to keep the
	// guard on values quiet. The guard is right that it cannot tell a two that
	// means a distance from a two that means every other one, and a flag says
	// what this is in a word instead of in arithmetic a reader has to run.
	banded := false
	for at, value := range values {
		rows = append(rows, valueRow(at+1, digits, value, banded))
		banded = !banded
	}
	card := paint.Card(palette.NameStepTight, paint.EvenRows(rows...))
	// 🔴 Scrolled, because this screen offers up to fifty values and a window is
	// not always tall enough for them.
	//
	// Without it the card asked for fifty rows at their smallest, which is over
	// two thousand pixels, and was handed that much whatever the window had.
	// Fyne does not clip children, so it drew over the band of chrome under it
	// and off the bottom of the window. The layout next door refuses to overflow
	// now, but on its own that only trades an overflow for rows squashed thinner
	// than a line of text, which is the same collision by another route.
	//
	// The two together are the answer, and the toolkit does the arithmetic:
	// scrollContainerRenderer.Layout resizes its content to the LARGER of the
	// content's minimum and its own size. So when the window has room the card
	// still grows into it, and when it has not the rows keep their floor and
	// the list scrolls. Neither behaviour is written here.
	//
	// KeyScroll rather than the toolkit's own, for the reason DEC-187 settled:
	// container.Scroll carries no keyboard method at all, so a person without a
	// mouse could reach the values on the About screen only by not needing to.
	// 73.6% of that screen was unreachable before the same repair.
	return parts.NewKeyScroll(paint.FitHeight(len(values), palette.NameStepTight, card))
}

// valueRow is one produced value, numbered.
//
// # Why the number is there at all
//
// Somebody comparing what this screen shows against what their application
// stored needs to say WHICH one, and "the third" is how a person says it. It is
// also the address the mode itself uses, so the number on screen and the number
// in the command line mean the same thing.
//
// # Why every other row carries a ground
//
// Ten values of nine digits each are the hardest thing on this screen to read
// across, and a band under every second one is what stops the eye sliding
// between rows. It is one step of the ladder, which is as loud as this may be
// without becoming a list of alternating cards.
func valueRow(number, digits int, value string, banded bool) fyne.CanvasObject {
	var ground fyne.CanvasObject = paint.Clear()
	if banded {
		ground = paint.Fill(palette.BgControl, 0)
	}

	index := paint.Mono(fmt.Sprintf("%*d", digits, number), palette.TypeLabel, palette.TextSecondary)
	index.Alignment = fyne.TextAlignTrailing

	inner := paint.Row(palette.NameStepGroup,
		paint.Middle(index),
		paint.Middle(paint.Mono(value, palette.TypeValue, palette.TextPrimary)))

	return paint.On(ground, paint.Pad(palette.NameStepHairline, palette.NameStepHairline,
		palette.NameStepGroup, palette.NameStepGroup, inner))
}

// heading names the card and carries the two things a person does with it.
//
// The count is in the words rather than beside them, because "10 values" is
// what somebody would say out loud and "Values (10)" is what a form would say.
func (v *View) heading(count int) fyne.CanvasObject {
	return paint.SectionHeading(v.countedValues(count),
		paint.Row(palette.NameStepTight, v.copy, v.again.OnScreen()))
}

// countedValues is the heading of the card, which says how many are under it.
//
// # Why the number is put into the sentence here rather than in the catalogue
//
// The catalogue carries one form of the sentence and a language that inflects
// the noun after a number would need several. That is a real problem and it is
// not this one: until a second language exists there is nothing to inflect, and
// the shape that solves it, a count aware catalogue, is a change to the
// translation layer rather than to this screen. What matters now is that the
// number is not written into the English words here.
func (v *View) countedValues(count int) string {
	return strconv.Itoa(count) + " " + v.text.T(i18n.QuickValuesCounted)
}

// dated is the field a date of birth is typed into, given room for the shape it
// expects.
//
// # Why this one field is measured and none of the others are
//
// A list sizes itself to its longest option and a button to its label, so every
// other control in that row asks for what it needs. A text field asks for
// almost nothing, because the toolkit cannot know what will be typed into it,
// and the placeholder does not count. In a row that gives each child what it
// asks for, this one collapsed to a sliver and its label ran into the label
// beside it, which is what the render showed as "BornSeed".
//
// 🔴 The width is MEASURED from the shape the field expects rather than written
// down. A number here would be a guess that goes stale the day the type ladder
// moves, and the guard on literals would refuse it anyway. The shape comes from
// the mode, so a date written some other way widens its own field.
func dated(field fyne.CanvasObject) fyne.CanvasObject {
	room := fyne.MeasureText(quick.DateShape, palette.TypeSize(palette.TypeBody), fyne.TextStyle{})
	// One margin either side, taken from the scale. Added rather than doubled
	// because it is two margins and not one margin twice, which is the same
	// reason the card of values states its own chrome that way.
	margin := palette.StepOf(palette.NameStepRelated)
	return paint.AtLeast(room.Width+margin+margin, palette.ControlHeight, field)
}

// answer is the middle of the screen: the values, or what is said instead.
//
// 🔴 A border rather than a column, and the render is what settled it.
//
// A column of the toolkit gives every child its MINIMUM height, so the card
// inside one was told it had room for ten rows at their smallest and no more.
// The card grows with the window on purpose, which is what spends spare space
// on the thing the screen is for, and in a column that ability was simply never
// reached: the rows stayed at their floor and the surplus sat below them as
// several hundred pixels of nothing.
//
// A border hands its centre everything the edges did not take, which is the
// only layout here that passes a height DOWN rather than asking for one up. The
// heading and the sentence are the edges because neither of them grows.
//
// The objects go in reading order for the reason the whole screen does, and the
// heading carries the two buttons, so it has to be the first stop.
func (v *View) answer() fyne.CanvasObject {
	if len(v.values) == 0 {
		// Nothing was produced, so there is no card to grow and the only thing
		// here is what the screen has to say about that. Given the border it
		// would be stretched down the window with a sentence floating in the
		// middle of it.
		return container.NewVBox(v.told)
	}

	// 🔴 What the screen SAYS goes above the card rather than below it, which
	// is where the render put it and where it was wrong.
	//
	// Under the card it became the bottom edge of the border, and the border
	// puts its bottom edge at the bottom of the WINDOW, so a sentence about ten
	// values stood a few hundred pixels away from them with nothing in between.
	// Above them it sits under the heading, beside the button that produced it,
	// which is where a person is looking when it appears.
	top := []fyne.CanvasObject{v.heading(len(v.values))}
	if v.told.Text != "" {
		top = append(top, v.told)
	}
	heading := paint.Pad(palette.NameStepHairline, palette.NameStepRelated,
		palette.NameStepHairline, palette.NameStepHairline,
		paint.Column(palette.NameStepTight, top...))

	return container.New(layout.NewBorderLayout(heading, nil, nil, nil),
		heading, v.valuesCard(v.values))
}
