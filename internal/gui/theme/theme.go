// Package theme carries every colour, every spacing step and every dimension
// the window is allowed to use.
//
// # Why the values live here and nowhere else
//
// A colour woven into thirty controls costs more to change than a constant,
// which is the reason docs/17 wrote the values down before any window existed.
// The colours are the ladders and the token register in visual.go. The test
// beside that file measures every token against every ground it stands on, and
// tools/kontrast.py measures the same pairs a second time without sharing a
// line of code with the test. This file holds the closed spacing scale and the
// sizes of the window, and the same script closes the scale against its table
// in docs/17 in both directions.
//
// The guard in internal/guard refuses a colour literal and a bare number
// anywhere in internal/gui outside this package. The two mechanisms answer
// different halves of one question: this package is where a value may live,
// and that guard is why no second place can quietly appear.
//
// # Where the first palette went
//
// Until DEC-253 this file also held the palette measured for the first window,
// a near black of 0x1B1B1B with eighteen named colours, and the fynetheme
// package held the theme that wore it. The window stopped drawing with them at
// DEC-232 and they were deleted once their only readers were their own tests.
// docs/17 section 5 keeps that table and its reasoning as history.
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

// The names a view refers to the steps of the spacing scale by.
//
// They exist so that a screen never types a name as a bare string, where a
// typo becomes a distance that silently resolves to the fallback. Every one of
// these is checked against the scale from both sides by a test, so a constant
// without a step and a step without a constant are both failures.
const (
	NameStepHairline = "hairline"
	NameStepTight    = "tight"
	NameStepRelated  = "related"
	NameStepGroup    = "group"
	NameStepSection  = "section"
	NameStepMajor    = "major"
)

// StepNames is every spacing constant above, so a test can close the register
// from the side the constants are on.
var StepNames = []string{
	NameStepHairline, NameStepTight, NameStepRelated,
	NameStepGroup, NameStepSection, NameStepMajor,
}

var stepsByName = func() map[string]Space {
	byName := make(map[string]Space, len(Scale))
	for _, step := range Scale {
		byName[step.Name] = step
	}
	return byName
}()

// StepOf is the distance a view asks for by name.
//
// A name with no step returns the tight step rather than nought, because a
// distance of nought glues two things together and nobody sees why. A view
// cannot stop, so the wrong answer is at least a visible one, and the pair of
// tests closing the register in both directions keeps this branch unreachable.
func StepOf(name string) float32 {
	if step, ok := stepsByName[name]; ok {
		return step.Step
	}
	return stepsByName[NameStepTight].Step
}

// BorderThickness is how thick an outline is: the one the toolkit draws
// around a text field, and the one this window draws around a button.
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

// OutlineInset is how far inside its own bounds the edge of a button is drawn.
//
// # Why an inset is needed at all
//
// The toolkit centres a stroke on the edge of the rectangle it is given, so a
// rectangle the size of the button would have half of its stroke outside the
// button and half over the button's own fill. That is the same half pixel trap
// BorderThickness above is about, and it has the same answer: the toolkit
// insets its own outline of a text field by half the stroke and shrinks it by
// the whole stroke, which is what the stored tree of the recipe editor shows
// at pos 1,1 with a stroke of 2. Doing the same here puts the whole stroke
// inside the button, on whole pixels, in the colour it was given.
//
// It is derived rather than typed, because the two are one decision: a
// thickness changed without the inset changing would put the stroke back on
// the boundary.
const OutlineInset float32 = BorderThickness / 2

// InnerRingInset is how far inside a control a second edge is drawn, so that
// it stands just inside the first one and touches it without covering it.
//
// # Why a control ever has two edges
//
// A text field holding what its screen would refuse wears its edge in
// state.danger, and focus is shown on a field by the colour of that same edge.
// Somebody typing a refused number holds both at once, so the mark of the
// keyboard moves one edge inward rather than being lost under the red (B-363).
//
// Derived for the reason OutlineInset is: the first edge ends one whole
// thickness inside its own inset, and a second one placed by a number of its
// own would overlap it or leave a gap the first time the thickness moved.
const InnerRingInset float32 = OutlineInset + BorderThickness

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
// They stood at 900 by 700 from the first window until 2026-09-08, chosen so
// that the About screen fits a module path beside its version without
// wrapping. The comment there recorded what would change them: the first
// screen whose natural content is wider. That screen arrived, so they changed.
//
// # Why 1120 by 780, which is arithmetic rather than taste
//
// A settings column puts a label beside the control it names. At 900, with a
// sidebar of 220 and a margin of 24 on each side, the content is 616 wide, two
// columns are 292 each, and a label of 140 plus a gap of 16 leaves 136 for the
// control itself. A directory path does not fit in 136. Asking instead for a
// control of 220 puts the column at 376, the content at 784 and the window at
// 1084, so 1120 is the first round number above it. The height follows the
// same screen: Take and corrupt shows the kinds of damage, the settings and
// what the run would do, and 780 is where the primary action stops falling
// below the fold.
//
// Two ways out were measured and rejected in DEC-225. Putting the label above
// the control frees the full 292 at 900, and doubles the height of every row
// on a screen that already does not fit vertically, so it moves the problem
// rather than solving it. A layout that folds two columns into one below a
// threshold is the best answer for a person on a small display and the most
// expensive to build, and it waits for someone who actually has one.
//
// What would change them again: a measurement on a display where this is too
// large to open comfortably, which is the case the folding layout is for.
const (
	StartWidth  float32 = 1120
	StartHeight float32 = 780
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

// AsideWidth is how wide a column of explanation standing beside content is.
//
// # Why a width at all, when everything else here measures itself
//
// A paragraph has a readable line length and a window does not. Left to fill
// whatever is free, the sentences explaining a value would run the full width
// of a large display, and a line a reader has to track back across is the one
// thing typography has measured for a century. This is the only place in the
// window where text is the content rather than the label on something.
//
// # Why this number
//
// It holds roughly fifty to sixty characters at the body rung, which is inside
// the range every source on the subject gives for comfortable reading, and it
// leaves the card beside it the larger share at the size this window opens at.
// A judgement taken from the accepted drawing rather than a measurement, and
// stated as one.
//
// What would change it: a rung of the type ladder taller than the body used for
// a paragraph, or a second aside on one screen, which would make the pair of
// them the thing to measure rather than either alone.
const AsideWidth float32 = 330

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
// columns need on the short ones: "row 4" and a phrase of five words do not
// want the same width. The widths below come from what each column holds, and
// since 2026-09-15 they are MEASURED on the face the table draws with rather
// than estimated: an identifier of the form LAYER.area.variant, a place, one of
// three phrases with a mark in front of the soft ones, and one of a closed set
// of outcomes.
//
// 🔴 Four columns rather than six, and the reason is arithmetic (DEC-247).
// The six columns of the first drawing added up to 1740, and the window at the
// size the owner keeps it hands the table 828: the sidebar takes 222 of 1113
// and the body a margin on each side. So the column saying whether damage
// happened at all, and the one carrying the reason it did not, were never on
// the screen without scrolling sideways, on the one screen where rule 4.4 asks
// that an unapplied entry stand out. The certainty joined the expectation as a
// mark in front of it, and the sentence went under the table into a card of
// its own, where it has room to wrap. The outcome column says, once a response
// record is open, what the application did with the row, because a row that
// was not applied has no answer and an applied one said only "applied".
//
// # What each width is, measured with tools/miary on 2026-09-15
//
// The first column is a FLOOR rather than a width: the table hands it whatever
// the other three leave, since an identifier is the one thing that gains from
// a wide window. The floor holds most identifiers whole (the longest in the
// catalog, L4.coherence.national_id_vs_birthdate, renders 236 and is cut with
// an ellipsis at the floor, where the card under the table carries it whole).
// The second holds "customer   row 2   last_name" at 176 and cuts a five digit
// row number. The third holds "~ should be accepted and marked" at 205, the
// widest phrase of its closed set. The fourth holds "no column of that kind" at
// 138, the widest outcome. Each is its widest value plus one tight step, so
// that text never touches the edge of its cell, which reads as text that was
// cut. A guard renders every value each fixed column can hold and asks whether
// it fits, because the paragraph above the previous widths had the proportion
// right in words while the numbers under it had it reversed.
var EntryColumnWidths = []float32{200, 184, 216, 148}

// EntryStretchColumn is the column that takes whatever width the others leave,
// never less than its entry in EntryColumnWidths.
const EntryStretchColumn = 0

// EntryColumns is how many columns one entry is laid out in.
func EntryColumns() int { return len(EntryColumnWidths) }

// EntryRowsAtLeast is the fewest rows the entries table asks the window for,
// the heading not counted.
//
// 🔴 Measured before it was written down: with the envelope open in the
// compared state at 1670 by 1210, a table with no floor was handed a negative
// height and the render died in the rasteriser, where a real window would have
// grown without a word. With a floor the window knows what it needs and the
// render says outright that the content does not fit.
const EntryRowsAtLeast = 4

// EntryDetailHeight is how tall the card under the entries table is, in the
// units the toolkit scales for the user.
//
// Fixed rather than hugging its content, because the card is refilled with
// whichever entry the keyboard or the pointer is on, and a card that changed
// height with every entry would move the foot of the table on every arrow key.
// Four lines of body type with a hairline between them and the card's own
// padding above and below: 112 cut the fourth line on the render of the
// compared state, and this is the height at which the same render showed all
// four.
const EntryDetailHeight float32 = 132

// CatalogueAreaRows is how many rows of a control's height an area of the
// catalogue of controls is given: a table and a scrolling area (DEC-253).
//
// A choice with a reason rather than a measurement. Enough rows to show what
// the area looks like with something in it, and fewer than the samples hold,
// so the same picture also shows that it scrolls. What would change it: a page
// of the catalogue that no longer fits the window it opens in.
const CatalogueAreaRows = 4

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
// Three since DEC-247, and two before it. The envelope carries eight to eleven
// pairs and stands in a section a person opens over the table, so every row of
// it is a row the table loses while the section is open: three columns hold
// eleven pairs in four rows where two held them in six.
const EnvelopeColumns = 3

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
