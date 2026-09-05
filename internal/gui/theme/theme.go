// Package theme carries every colour and every spacing step the window is
// allowed to use.
//
// # Why the values live here and nowhere else
//
// A colour woven into thirty controls costs more to change than a constant,
// which is the reason docs/17 wrote the palette down before any window
// existed. That document is measured rather than judged: tools/kontrast.py
// computes the contrast of every entry with the WCAG formula and gates three
// thresholds. Until this package existed the script had only one side to read,
// and it said so about itself. Now it has two, and a colour that differs
// between the document and this file is a finding in both directions.
//
// The guard in internal/guard refuses a colour literal and a bare number
// anywhere in internal/gui outside this package. The two mechanisms answer
// different halves of one question: this file is where a value may live, and
// that guard is why no second place can quietly appear.
//
// # Why no graphical toolkit is imported here
//
// image/color is in the standard library, so the palette compiles without the
// C toolchain and the bridge closes before the toolkit is added. A package
// that needed the toolkit to state its own colours would tie the day the
// palette becomes checkable to the day the first window is built, and those
// are two different days on purpose.
//
// # What this package does not answer
//
// Whether a colour carries a shape or a word beside it, which is WCAG 1.4.1
// and the reason docs/17 states a measured number for it. Whether the window
// is legible. Both are read by a person against the checklist in docs/17, and
// neither has a mechanism.
package theme

import (
	"fmt"
	"image/color"
)

// Role says which contrast threshold an entry answers to.
//
// The third one is ours rather than WCAG's. WCAG says nothing about a stripe
// that only helps the eye stay inside a row, because such a stripe carries no
// information, so the threshold for it had to come from somewhere else and is
// cited as ours wherever it appears.
type Role string

const (
	// RoleText is read as text and answers to WCAG 1.4.3, 4.5 to 1.
	RoleText Role = "text"
	// RoleSurface is a control or a border and answers to WCAG 1.4.11,
	// 3 to 1.
	RoleSurface Role = "surface"
	// RoleAuxiliary only helps the eye follow a row. The threshold is ours.
	RoleAuxiliary Role = "auxiliary"
	// RoleOnFill is read as text and answers to the same 4.5 to 1 of WCAG
	// 1.4.3, but it is measured against the fill it is read on rather than
	// against the window. MeasuredAgainst below names that fill.
	//
	// The role exists because the toolkit floods a whole control with one
	// colour while the keyboard is on it, so a label there is never read
	// against the window at all. Measuring it there would report a number
	// nobody can see, and it would report it as a pass.
	RoleOnFill Role = "on-fill"
	// RoleBackground is what other colours are measured against, so it
	// answers to no threshold of its own.
	//
	// A panel is meant to sit barely apart from the window behind it, which
	// is why the measured 1.11 of the panel is reported rather than gated.
	// Gating it would demand that a panel shout, and a threshold invented to
	// be met is not a threshold.
	RoleBackground Role = "background"
)

// Color is one entry of the palette.
//
// The channels are stored as bytes rather than as a string, so nothing parses
// a colour while the window is running and no entry can fail late. The hex
// form is computed for reports and for the bridge, which keeps one source for
// the value instead of a pair that can drift apart.
type Color struct {
	// Name is the key shared with the table in docs/17.
	Name string
	// R, G and B are the channels.
	R, G, B uint8
	// Role picks the threshold this entry is measured against.
	Role Role
	// Purpose is what stops being expressible if this entry disappears.
	Purpose string
}

// NRGBA is the colour in the form the toolkit takes. Every entry is fully
// opaque, because a translucent value would make the measured contrast a
// statement about a blend that the measurement never saw.
func (c Color) NRGBA() color.NRGBA {
	return color.NRGBA{R: c.R, G: c.G, B: c.B, A: 0xFF}
}

// Hex is the entry written the way docs/17 writes it.
func (c Color) Hex() string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}

// Background is the colour every other entry is measured against.
//
// It is named separately rather than kept in the list, because it is the
// reference of the measurement and not one of its subjects. An entry measured
// against itself would report a contrast of one and read like a failure.
var Background = Color{
	Name:    "background",
	R:       0x1B,
	G:       0x1B,
	B:       0x1B,
	Role:    RoleBackground,
	Purpose: "the window itself, and the reference every contrast is measured against",
}

// Palette is every colour the window may show.
//
// The order matches the table in docs/17 so that a person comparing the two by
// eye is not also comparing two orderings.
var Palette = []Color{
	{"surface", 0x24, 0x24, 0x24, RoleBackground, "a panel and the header of a table, which sits barely apart from the window on purpose"},
	{"text", 0xE8, 0xE8, 0xE8, RoleText, "the text a person reads first"},
	{"text.subdued", 0xA0, 0xA0, 0xA0, RoleText, "metadata and captions"},
	{"expect.rejection", 0xF1, 0x70, 0x7A, RoleText, "an expectation of rejection, the single most important thing a manifest says"},
	{"expect.acceptance", 0x6C, 0xCB, 0x5F, RoleText, "an expectation of acceptance"},
	{"expect.warning", 0xFF, 0xB9, 0x00, RoleText, "an expectation of a warning"},
	{"expect.not-applied", 0xC7, 0x92, 0xEA, RoleText, "a corruption that could not be applied, which is the ban on silence made visible"},
	{"corrupted", 0xFF, 0x9E, 0x64, RoleText, "a corrupted value beside a correct one"},
	{"focus", 0x4E, 0xC9, 0xF5, RoleSurface, "the focus indicator, WCAG 2.1.1"},
	{"text.on-focus", 0x1B, 0x1B, 0x1B, RoleOnFill, "the label of a control the toolkit floods with the focus colour, where the ordinary text colour measures 1.56"},
	{"control.chosen", 0x9E, 0xD8, 0xF0, RoleSurface, "a control that has been chosen, and the border of a text field the keyboard is in"},
	{"border.active", 0x6E, 0x6E, 0x6E, RoleSurface, "the border of the active control"},
	{"row.divider", 0x3A, 0x3A, 0x3A, RoleAuxiliary, "the line between rows, which stands in for the zebra stripe that was measured not to work here"},
	{"row.hover", 0x30, 0x30, 0x30, RoleAuxiliary, "the row under the pointer"},
	{"row.selected", 0x38, 0x50, 0x6B, RoleAuxiliary, "the selected row"},
}

// MeasuredAgainst names, for every entry that is read on top of another entry,
// the entry it is read on.
//
// # Why a register beside the palette rather than a field in it
//
// Almost every colour here is measured against the window, and a field would
// put an empty value on thirteen entries to say so. A register says the same
// thing by being short, and it can be closed from both sides by a test: an
// entry carrying RoleOnFill with no row here is a colour whose threshold
// nobody can compute, and a row naming an entry that does not exist is a rule
// about nothing.
//
// # Why this exists at all
//
// Measured on 2026-09-05 through the pixels of the render: the label of a
// focused button arrived as #E8E8E8 on #4EC9F5, which is 1.56 where WCAG 1.4.3
// asks for 4.5. The toolkit floods the whole control with the focus colour
// rather than tinting it, because the blend it performs collapses to the
// overlay when the overlay is opaque, and every entry here is opaque by
// decision. So the label is read on the focus colour and has to be measured
// there.
var MeasuredAgainst = map[string]string{
	NameTextOnFocus: NameFocus,
}

// The names a view refers to its values by.
//
// They exist so that a screen never types a name as a bare string, where a
// typo becomes a colour that silently resolves to nothing. Every one of these
// is checked against the palette from both sides by a test, so a constant
// without an entry and an entry without a constant are both failures.
const (
	NameSurface       = "surface"
	NameText          = "text"
	NameTextSubdued   = "text.subdued"
	NameRejection     = "expect.rejection"
	NameAcceptance    = "expect.acceptance"
	NameWarning       = "expect.warning"
	NameNotApplied    = "expect.not-applied"
	NameCorrupted     = "corrupted"
	NameFocus         = "focus"
	NameTextOnFocus   = "text.on-focus"
	NameControlChosen = "control.chosen"
	NameBorderActive  = "border.active"
	NameRowDivider    = "row.divider"
	NameRowHover      = "row.hover"
	NameRowSelected   = "row.selected"
	NameStepHairline  = "hairline"
	NameStepTight     = "tight"
	NameStepRelated   = "related"
	NameStepGroup     = "group"
	NameStepSection   = "section"
	NameStepMajor     = "major"
)

// ColorNames is every colour constant above, so a test can close the register
// from the side the constants are on.
var ColorNames = []string{
	NameSurface, NameText, NameTextSubdued, NameRejection, NameAcceptance,
	NameWarning, NameNotApplied, NameCorrupted, NameFocus, NameTextOnFocus,
	NameControlChosen, NameBorderActive,
	NameRowDivider, NameRowHover, NameRowSelected,
}

// StepNames is every spacing constant above, for the same reason.
var StepNames = []string{
	NameStepHairline, NameStepTight, NameStepRelated,
	NameStepGroup, NameStepSection, NameStepMajor,
}

var colorsByName = func() map[string]Color {
	byName := make(map[string]Color, len(Palette)+1)
	for _, entry := range Palette {
		byName[entry.Name] = entry
	}
	byName[Background.Name] = Background
	return byName
}()

var stepsByName = func() map[string]Space {
	byName := make(map[string]Space, len(Scale))
	for _, step := range Scale {
		byName[step.Name] = step
	}
	return byName
}()

// Named is the colour a view asks for by name.
//
// A name with no entry returns the foreground text colour rather than an empty
// value, because a zero colour is transparent black and would render as a hole
// nobody can see. Loud is not available here, since a view cannot stop, so the
// wrong answer is at least a visible one. The pair of tests that close the
// register in both directions is what keeps this branch unreachable.
func Named(name string) Color {
	if entry, ok := colorsByName[name]; ok {
		return entry
	}
	return colorsByName[NameText]
}

// Lookup is the colour of that name, and whether this palette has one.
//
// # Why this exists beside Named
//
// Named always answers, which is what a view wants: a control has to be painted
// something. This one reports whether the name is ours at all, which is what the
// bridge to the toolkit needs, since it is asked for colours by names that
// belong to the toolkit and has to tell those apart from ours without a second
// list of what ours are.
func Lookup(name string) (Color, bool) {
	entry, ok := colorsByName[name]
	return entry, ok
}

// StepOf is the distance a view asks for by name. The same reasoning as Named
// applies to the fallback.
func StepOf(name string) float32 {
	if step, ok := stepsByName[name]; ok {
		return step.Step
	}
	return stepsByName[NameStepTight].Step
}

// BorderThickness is how thick the toolkit draws the outline of a text field.
//
// # Why this is not a step of the spacing scale
//
// The scale measures distances BETWEEN things and its smallest step is 4. This
// is the width of a line, which is not a distance between anything, and 4 draws
// a text field like a picture frame. So it is a value of the view with a reason
// beside it, the way the size the window opens at is.
//
// # Why 2 and not the toolkit's 1
//
// 🔴 Measured on 2026-08-24, and it is the difference between a colour that
// exists and a colour anybody can see. The toolkit draws this outline offset by
// half of its width, so at a width of 1 the line falls between two rows of
// pixels and every pixel of it is a half blend with what is behind. Measured
// through the pixels: the outline asked for #9ED8F0 and arrived as #617E8A,
// against a border of #6E6E6E that had itself arrived as #484848, which is a
// contrast of 2.12 where the rule asks for 3.0. No colour repairs that, because
// the blend is what caps it: pure white measured 2.78.
//
// At a width of 2 the line lands on whole pixels and arrives as the colour it
// was given, so the same #9ED8F0 measures 3.29, which is what the palette says
// it should be. Wider than 2 changes nothing but the number of pixels covered.
//
// What would change it: a toolkit that stops offsetting the outline by half its
// width, or a display scale where 2 stops landing on whole pixels.
const BorderThickness float32 = 2

// StartWidth and StartHeight are the size a window opens at.
//
// # Why this is here rather than in the screen that opens the window
//
// It is a value of the view, and the guard refuses a bare number anywhere in
// the window outside this package. That refusal is what produced this entry:
// the first screen opened 128 pixels tall, showing one heading and nothing
// else, because a scrolling container asks for almost no height of its own and
// a window with no size of its own shrinks to what its content demands. Every
// test was green, since they read the content rather than how much of it is
// visible.
//
// # Why these two numbers
//
// They are a choice with a reason rather than a measurement, and are written
// as such. The width fits the longest line the About screen has, which is a
// module path beside its version, without wrapping it. The height shows two
// sections at once, so that the first thing a person sees is that the screen
// continues rather than a single heading that looks like the whole window.
//
// What would change them: the first screen whose natural content is wider, or
// a measurement on a display where this is too large to open comfortably.
const (
	StartWidth  float32 = 900
	StartHeight float32 = 700
)

// SidebarWidth is how wide the column naming the screens is.
//
// # Why the window has such a column at all
//
// docs/08 names seven screens. Tabs across the top narrow as they are added,
// so the name of the screen a person is on stops being fully readable exactly
// when there are enough screens for that to matter, and a list down the side
// does not have that property. The owner chose it on 2026-08-17, recorded with
// what was turned down.
//
// # Why this number
//
// A choice with a reason rather than a measurement, and written as such. It
// holds the longest screen name in the catalogue without wrapping it, and
// leaves the content the greater part of a window opened at StartWidth. What
// would change it: the first screen name that wraps here, or a language whose
// names are longer than the English ones.
const SidebarWidth float32 = 220

// EntryColumnWidths is how wide each column of the manifest table is.
//
// # Why a row is columns rather than lines
//
// The first drawing put every fact of an entry on a line of its own, four lines
// to a row. Running it showed what the practice on data tables says outright:
// multi line rows destroy the vertical rhythm that makes a list scannable, so a
// reader compares by reading rather than by looking, and an entry with nothing
// to say on one line leaves a hole that pushes its last line against the
// following entry.
//
// # Why the widths differ rather than being equal
//
// Equal columns were the second attempt and they waste the room the long
// columns need on the short ones: "row 4" and a sentence explaining why damage
// could not be applied do not want the same width. The proportions below come
// from what each column holds: an identifier of the form LAYER.area.variant, a
// place, one of three phrases, one of two, one of two words, and a sentence.
//
// The last one is deliberately the widest, because it is the one column allowed
// to carry prose. Practice on tables reserves full wrapping for exactly one
// column and holds every other to a single line, which is the same rhythm the
// paragraph above is about.
//
// Every value is a multiple of the spacing module for the reason recorded with
// that module: a desktop scales the interface by a factor the user picks, and
// these land on whole numbers at the common ones.
var EntryColumnWidths = []float32{280, 240, 220, 340, 140, 720}

// EntryColumns is how many columns one entry is laid out in.
func EntryColumns() int { return len(EntryColumnWidths) }

// PreviewColumnMin and PreviewColumnMax bound a column of the recipe preview.
//
// # Why bounds rather than a list of widths
//
// The manifest table above knows its six columns and can be told what each one
// holds. A preview cannot: its columns are whatever somebody's recipe declares,
// named by them and filled by generators this file has never heard of. So the
// width of each is measured from what it actually holds, by the toolkit, and
// only the two ends of the range are decided here.
//
// # Why both ends are needed
//
// Without the lower bound a column of a one character value would be a sliver
// its own heading does not fit in. Without the upper one a single long value
// takes the width of the pane and pushes every column after it out of sight,
// which is the fault this project measured on the screen next door: a layout
// that fits the sample and grows with the input. Between them a value that does
// not fit truncates with an ellipsis rather than wrapping, because a preview
// row is one line by rule.
//
// Both are multiples of the spacing module, for the reason recorded with that
// module: a desktop scales the interface by a factor the user picks, and these
// land on whole numbers at the common ones.
const (
	PreviewColumnMin float32 = 96
	PreviewColumnMax float32 = 320
)

// EnvelopeColumns is how many columns of label and value pairs the envelope of
// a run is split into.
//
// It lives here rather than beside the screen for the reason every other value
// of the view does: the guard refuses a bare number anywhere in the window, and
// it draws no line between a count and a distance. That refusal is worth more
// than the exception would be, since "this one is only a count" is the sentence
// every value in a view could be introduced with.
//
// Two, because the envelope carries about a dozen pairs and one column of them
// is a tall block of a screen whose subject is the table under it.
const EnvelopeColumns = 2

// Space is one step of the spacing scale.
type Space struct {
	// Name is the key shared with the table in docs/17.
	Name string
	// Step is the distance in the units the toolkit scales for the user.
	Step float32
	// Purpose is the one thing this step is for. A step whose purpose
	// cannot be written in a line is a step somebody added to fit a screen.
	Purpose string
}

// SpacingModule is the number every step is a multiple of.
//
// This is arithmetic rather than a measurement and is stated as such. A
// desktop scales its interface by factors a user picks, and 1.25, 1.5 and 2.0
// are the common ones. Every step below times any of those lands on a whole
// number, so a gap never rounds to a different distance than its neighbour of
// the same name. A scale built on 6 breaks that at the first factor, because
// six and a quarter is not whole. The test beside this file does that
// arithmetic rather than trusting this paragraph.
const SpacingModule = 4

// Scale is every spacing step the window may use, and it is closed.
//
// # Why closed rather than a table of values
//
// A list that grows whenever a screen needs one more gap is a register of
// excuses rather than a scale, and it stops being able to say that two
// distances are deliberately the same. This scale was closed before the first
// screen existed, so a screen that does not fit it has to argue rather than
// quietly add a seventh entry.
//
// Six steps rather than four: four cannot separate a gap inside a control from
// a gap between two controls without one of them being wrong by a factor of
// two. Six rather than eight: two neighbouring steps a person cannot tell
// apart are two names for one distance.
var Scale = []Space{
	{"hairline", 4, "inside a single control, between an icon and its label"},
	{"tight", 8, "between parts of one thing, such as a label and the field it names"},
	{"related", 12, "between rows of a list"},
	{"group", 16, "between groups of controls"},
	{"section", 24, "the margin of the window, and between sections of a screen"},
	{"major", 32, "between the parts of a screen that are read separately"},
}
