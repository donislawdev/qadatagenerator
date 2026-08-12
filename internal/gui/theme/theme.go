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
	{"border.active", 0x6E, 0x6E, 0x6E, RoleSurface, "the border of the active control"},
	{"row.divider", 0x3A, 0x3A, 0x3A, RoleAuxiliary, "the line between rows, which stands in for the zebra stripe that was measured not to work here"},
	{"row.hover", 0x30, 0x30, 0x30, RoleAuxiliary, "the row under the pointer"},
	{"row.selected", 0x38, 0x50, 0x6B, RoleAuxiliary, "the selected row"},
}

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
