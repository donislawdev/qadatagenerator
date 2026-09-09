package theme

import "image/color"

// The visual system the window is drawn with: two ladders of twelve shades, a
// ladder of type sizes, and a register of tokens naming what each shade is for.
//
// # Why a second set of values beside the palette above
//
// The palette above answers a different question and answers it well: which
// colour carries which MEANING. An expectation of rejection is red because the
// manifest says rejection, and that entry is measured against the window.
//
// This file answers the question that made the owner call the window generic
// on 2026-09-08: which colour carries which PLACE. A card apart from the
// window, a control apart from the card, an edge apart from a divider. The
// palette above has two shades of ground and one of text, so there was nothing
// to say any of that with and every screen came out flat. Counting the shades
// is what settled it rather than looking at them, and the count is the
// diagnosis in docs/NAUKA-GUI.md section 6.1: this is a shortage of steps, not
// a shortage of contrast in the steps that exist.
//
// # Why twelve steps rather than the four a screen would use today
//
// The method is Radix Colors and its point is that a step has a JOB rather
// than a lightness. Steps 1 to 2 are grounds, 3 to 5 are the states of a
// control, 6 to 8 are edges, 9 to 10 are fills, 11 to 12 are text. A ladder
// cut down to the steps in use today stops being a ladder: the worth of step 7
// is that it sits between 6 and 8, so an edge is made louder by moving one
// step rather than by inventing a colour. What would make a step wrong here is
// one that no token names and no neighbouring token could ever move onto.
//
// # Why a token register rather than screens naming shades
//
// A screen asking for neutral.11 has decided two things: what it is drawing
// and how light that should be. The second decision is then repeated at every
// call site, which is how a colour ends up woven into thirty controls. A
// screen asking for text.secondary has decided only the first, so the shade
// behind it moves in one line here. That is the owner's requirement of
// 2026-09-08, build it so it can be extended or reworked later, written as
// something a compiler can hold rather than as an intention.
//
// # What is measured, and what a test does about it
//
// Every token carries the threshold it answers to AND the grounds it actually
// stands on. The second half is what the first measurement of this ladder got
// wrong: read against every ground in the window, no control passed, because
// the probe measured a button against the row highlight and no button stands
// there. A ground set is part of a contrast measurement rather than a footnote
// to it, so it lives in the register and the test beside this file walks it.

// Shade is one step of one ladder.
//
// It carries no threshold of its own. A shade is a colour in a sequence, and
// whether it is legible depends entirely on what it is used FOR, which is what
// the token register below says and what the test measures.
type Shade struct {
	// Name is the key a token refers to this step by.
	Name string
	// R, G and B are the channels.
	R, G, B uint8
	// Job is what this step is for, in the words of the method rather than
	// in the words of the one screen that happens to use it first.
	Job string
}

// NRGBA is the shade in the form the toolkit takes.
func (s Shade) NRGBA() color.NRGBA {
	return color.NRGBA{R: s.R, G: s.G, B: s.B, A: 0xFF}
}

// Neutral is the ladder every ground, edge and piece of text is drawn from.
//
// # Why it is warm rather than grey
//
// A neutral built from equal channels is the ground of the toolkit's own dark
// theme, and a window drawn on it looks like the toolkit. Bending the ladder
// warm costs nothing, is invisible on any single element, and is the first
// thing that makes the window look chosen rather than defaulted. A judgement
// rather than a measurement, and written as one.
var Neutral = []Shade{
	{"neutral.1", 0x10, 0x0F, 0x0E, "the window itself"},
	{"neutral.2", 0x18, 0x17, 0x16, "a ground set apart from the window: a card, the sidebar, a band of chrome"},
	{"neutral.3", 0x20, 0x20, 0x1E, "the inside of a control at rest, and every other row of a list"},
	{"neutral.4", 0x28, 0x27, 0x25, "the sidebar item naming the screen that is open"},
	{"neutral.5", 0x31, 0x30, 0x2D, "the row under the pointer"},
	{"neutral.6", 0x3B, 0x3A, 0x36, "the quiet line: between two rows, and around a card"},
	{"neutral.7", 0x4A, 0x48, 0x42, "one step louder than the quiet line, for an edge that is found rather than followed"},
	{"neutral.8", 0x61, 0x5F, 0x58, "the loudest shade that is still an edge rather than a mark"},
	{"neutral.9", 0x6F, 0x6D, 0x65, "a filled shape carrying no text of its own"},
	{"neutral.10", 0x80, 0x7E, 0x76, "the edge of a control, which has to be seen rather than followed"},
	{"neutral.11", 0xB4, 0xB1, 0xA9, "text read second: a caption, a group heading, the sentence beside a value"},
	{"neutral.12", 0xED, 0xEC, 0xE9, "text read first"},
}

// Accent is the ladder the one action of a screen is drawn from.
//
// # Why violet, and it is an argument rather than a taste
//
// This product already spends green, amber and red on things that MEAN
// something: an expectation of acceptance, of a warning, of rejection. An
// accent taken from any of those hues would put a colour saying press this
// beside the same hue saying this was rejected. Violet is the hue furthest
// from all three, so it can never be read as a verdict.
//
// # Why almost none of it is spent
//
// One filled shape per screen, and the mark on the sidebar item the window is
// on. That is the entire budget. An accent spent on every control is the same
// as no accent, which is the state the window was in before this file existed,
// only louder.
var Accent = []Shade{
	{"accent.1", 0x14, 0x11, 0x1E, "the window, tinted, for a screen that is entirely about one action"},
	{"accent.2", 0x1B, 0x16, 0x2B, "a ground set apart, tinted"},
	{"accent.3", 0x24, 0x1C, 0x3C, "the inside of a chosen control at rest"},
	{"accent.4", 0x2C, 0x22, 0x50, "that ground under the pointer"},
	{"accent.5", 0x35, 0x2A, 0x60, "that ground while pressed"},
	{"accent.6", 0x41, 0x34, 0x6F, "the quiet edge of a chosen control"},
	{"accent.7", 0x51, 0x41, 0x86, "one step louder"},
	{"accent.8", 0x65, 0x53, 0xA6, "the edge of the field a search is typed into"},
	{"accent.9", 0x70, 0x59, 0xD6, "the fill of the one action a screen is for"},
	{"accent.10", 0x7E, 0x68, 0xE2, "that fill under the pointer"},
	{"accent.11", 0xA5, 0x94, 0xFF, "the accent where it has to be READ: an icon, a prompt, the mark on the current item"},
	{"accent.12", 0xE0, 0xDA, 0xFF, "text on a tinted ground"},
}

// Semantic is the three colours that carry a verdict rather than a place.
//
// They are stated here rather than borrowed from the palette above because a
// contrast is a statement about a PAIR, and these are read on the grounds of
// this ladder while the palette above is measured against a window of another
// colour. One entry serving both would report one number about two pairs.
var Semantic = []Shade{
	{"semantic.success", 0x3D, 0xA7, 0x66, "a value inside the range it belongs to, and a safe harbour a provider can claim"},
	{"semantic.warning", 0xE8, 0xA3, 0x3D, "something to look at before trusting it"},
	{"semantic.danger", 0xE0, 0x55, 0x61, "a refusal, and a value outside what the schema allows"},
}

// White is the one colour here that belongs to no ladder.
//
// It exists because the label on the accent fill is read ON that fill, and the
// top of the neutral ladder is not light enough for it: neutral.12 on accent.9
// measures 4.35 where WCAG 1.4.3 asks for 4.5. White measures 5.14. A ladder
// bent so that its own top passes would be a ladder bent to fit one pair,
// which is the same fault as a threshold lowered to fit a control.
var White = Shade{"white", 0xFF, 0xFF, 0xFF, "the label on the accent fill, which the neutral ladder cannot carry"}

// The names a view refers to a token by.
//
// A token says WHAT is being drawn and never how light it should be, so the
// shade behind it moves here in one line rather than at every call site.
const (
	BgWindow  = "bg.window"
	BgSurface = "bg.surface"
	BgControl = "bg.control"
	BgCurrent = "bg.current"
	BgHover   = "bg.hover"

	LineQuiet     = "line.quiet"
	BorderControl = "border.control"

	FillPrimary = "fill.primary"
	MarkCurrent = "mark.current"
	MarkActive  = "mark.active"

	TextPrimary   = "text.primary"
	TextSecondary = "text.secondary"
	TextAccent    = "text.accent"
	TextOnFill    = "text.on-fill"
	TextOnMark    = "text.on-mark"

	StateSuccess = "state.success"
	StateWarning = "state.warning"
	StateDanger  = "state.danger"
)

// TokenNames is every token constant above, so a test can close the register
// from the side the constants are on. The same reason ColorNames exists.
var TokenNames = []string{
	BgWindow, BgSurface, BgControl, BgCurrent, BgHover,
	LineQuiet, BorderControl,
	FillPrimary, MarkCurrent, MarkActive,
	TextPrimary, TextSecondary, TextAccent, TextOnFill, TextOnMark,
	StateSuccess, StateWarning, StateDanger,
}

// Token is one thing a screen can ask to be drawn in.
type Token struct {
	// Name is what a screen asks for.
	Name string
	// Shade is which step of which ladder answers.
	Shade string
	// Threshold is the contrast this token has to reach against every
	// ground in On.
	Threshold float64
	// On is every ground this token is actually drawn on.
	//
	// 🔴 This is the half a first measurement of this ladder got wrong, and
	// it is the half that decides the answer. Read against every ground in
	// the window, no control passed, because the probe measured a button
	// against the row highlight and no button stands there. Read against
	// the grounds it stands on, one token failed and it was a different one.
	On []string
	// Purpose is what stops being expressible if this token disappears.
	Purpose string
}

// The thresholds. The first two are WCAG. The third is ours and says so.
const (
	// ThresholdText is WCAG 1.4.3 for text below the large size, which is
	// every piece of text in this window: the tallest rung of the ladder
	// below is 21, and the rule turns at 24, or at 19 when the text is bold.
	ThresholdText = 4.5
	// ThresholdControl is WCAG 1.4.11, the visual boundary of a control.
	ThresholdControl = 3.0
	// ThresholdGuide is ours and is stated as ours. A line between two rows
	// helps the eye stay on one of them and carries nothing a person reads,
	// so no WCAG threshold applies. Leaving it unmeasured instead would
	// mean a line drawn so faintly nobody can follow it is not a failure.
	ThresholdGuide = 1.4
	// ThresholdGround is nought and marks a token that is itself a ground.
	// It is measured by whatever stands on it. Named rather than written as
	// a bare nought so a reader does not have to guess what nought means.
	ThresholdGround = 0
)

// Tokens is every token, the shade behind it, and what it is measured against.
//
// # The five entries that differ from the drawing the owner accepted
//
// Measured on 2026-09-08, after the drawing was accepted and before any of it
// reached the product. Acceptance is of a DIRECTION and a contrast is a fact
// about a pair, so the two do not overlap and neither overrules the other:
//
//   - a quiet button drew its edge in neutral.7, measuring 1.44 against the
//     row highlight it stands on. It is neutral.10 here, measuring 3.25.
//   - group headings and captions were written in neutral.9, measuring 2.88.
//     They are neutral.11 here, and their quietness moves off lightness and
//     onto size and capitals, which the ladder of type sizes below carries.
//   - the sentence beside a value was neutral.10, measuring 3.25 on the row
//     under the pointer. It is neutral.11 here.
//   - the mark on the current sidebar item was accent.9 on neutral.4, which
//     measures 2.91 against a threshold of 3.0. It is accent.11, measuring
//     5.88.
//   - the label on the accent fill was already white and stays white, which
//     is the one of the five that needed no change.
//
// None of the five changes the direction: violet stays violet and quiet stays
// quieter than loud. What changed is which rung of the ladder each one takes.
var Tokens = []Token{
	{BgWindow, "neutral.1", ThresholdGround, nil,
		"the window, which everything else is measured against"},
	{BgSurface, "neutral.2", ThresholdGround, nil,
		"a card, the sidebar and a band of chrome, set apart from the window by one step rather than by a shadow"},
	{BgControl, "neutral.3", ThresholdGround, nil,
		"the inside of a field or a chooser, and every other row of a list"},
	{BgCurrent, "neutral.4", ThresholdGround, nil,
		"the sidebar item naming the screen that is open"},
	{BgHover, "neutral.5", ThresholdGround, nil,
		"the row under the pointer, which is where the action belonging to that row appears"},

	{LineQuiet, "neutral.6", ThresholdGuide, []string{BgWindow, BgSurface},
		"the line between two rows and the edge of a card, which the eye follows rather than reads"},
	{BorderControl, "neutral.10", ThresholdControl, []string{BgWindow, BgSurface, BgControl, BgHover},
		"the edge saying a shape is a control rather than a word, which DEC-211 settled has to be drawn"},

	{FillPrimary, "accent.9", ThresholdControl, []string{BgSurface},
		"the fill of the one action a screen is for, which is the only thing carrying hierarchy here"},
	{MarkCurrent, "accent.11", ThresholdControl, []string{BgSurface, BgCurrent},
		"the bar beside the sidebar item the window is on, which says where you are without a second word"},
	// 🔴 The one token measured against another MARK rather than against a
	// ground, and the reason is the whole entry.
	//
	// The toolkit paints this on a stock control that is chosen or holds the
	// keyboard: the border of a focused field, the caret inside it, and the
	// glyph of a ticked box. On the field it REPLACES the resting border, so
	// the pair a person actually sees is this against border.control, and a
	// measurement against the ground behind them would pass while the change
	// stayed invisible. That is not a hypothesis: with the accent fill here,
	// the border went from 0x807E76 to 0x7059D6 and measured 1.26 where the
	// rule asks for 3.0, which is one saturated mid tone replacing another.
	//
	// # Why the accent ladder cannot answer this
	//
	// Measured across it: accent.9 gives 1.26, accent.10 gives 1.04, accent.11
	// gives 1.60. Only accent.12 clears the threshold, at 3.03, and its stated
	// job is text on a tinted ground. So the mark rides the top of the neutral
	// ladder, where it clears at 3.44 with room to spare.
	//
	// What this costs, said plainly: a ticked box and a focused field are not
	// violet. The accent is spent on the one action of a screen instead, which
	// is the budget this system already set, and a focus mark that cannot be
	// seen is worse than one that is not tinted.
	{MarkActive, "neutral.12", ThresholdControl, []string{BorderControl, BgControl, BgSurface},
		"what the toolkit paints on a stock control that is chosen or holds the keyboard, which has to differ from the resting edge it replaces"},

	{TextPrimary, "neutral.12", ThresholdText, []string{BgWindow, BgSurface, BgControl, BgCurrent, BgHover},
		"the value, the label and every word read first"},
	{TextSecondary, "neutral.11", ThresholdText, []string{BgWindow, BgSurface, BgControl, BgCurrent, BgHover},
		"a caption, a group heading and the sentence explaining a value"},
	{TextAccent, "accent.11", ThresholdText, []string{BgWindow, BgSurface, BgCurrent},
		"the prompt of a command line, and the icon of the screen that is open"},
	{TextOnFill, "white", ThresholdText, []string{FillPrimary},
		"the label on the accent fill, measured on that fill rather than on the window"},
	// 🔴 The second token measured on a MARK rather than on a ground, and it
	// exists because the toolkit FLOODS rather than tints.
	//
	// A stock button or chooser holding the keyboard is painted entirely in
	// the focus colour - widget/button.go:379 and widget/select.go:432 - so
	// everything drawn on it is read on mark.current and not on the ground the
	// control stands on. Measured on 2026-09-05, before any of this: the
	// ordinary text ink on that flood gave 1.56 where a label needs 4.5, which
	// is what parts.Flooded was written to repair.
	//
	// # Why it is a token now and was a palette entry before
	//
	// Flooded asked the OLD palette for this ink, which is the same debt the
	// filled button carried until this morning: a control still reading from
	// the design that preceded the ladder. It also left an icon on such a
	// control with nothing to repaint itself to, and an icon is exactly what
	// the sidebar of the accepted drawing puts beside every name.
	//
	// Measured 2026-09-09: neutral.1 on mark.current gives 7.55 where a label
	// needs 4.5 and a mark needs 3.0, so one entry serves both the words and
	// the icons drawn on a flooded control.
	{TextOnMark, "neutral.1", ThresholdText, []string{MarkCurrent},
		"the label and the icon on a control the toolkit has flooded with the focus colour, which is read on that flood rather than on the ground behind it"},

	{StateSuccess, "semantic.success", ThresholdText, []string{BgWindow, BgSurface},
		"a safe harbour, and a value inside the range it belongs to"},
	{StateWarning, "semantic.warning", ThresholdText, []string{BgWindow, BgSurface},
		"something to look at before trusting it"},
	{StateDanger, "semantic.danger", ThresholdText, []string{BgWindow, BgSurface},
		"a refusal, and a value outside what the schema allows"},
}

// TextStep is one rung of the ladder of type sizes.
type TextStep struct {
	// Name is what a view asks for.
	Name string
	// Size is the height of the type.
	Size float32
	// Purpose is the one thing this rung is for. A rung whose purpose does
	// not fit on a line is a rung somebody added to fit one screen.
	Purpose string
}

// The names a view refers to a type size by.
const (
	TypeTitle   = "type.title"
	TypeValue   = "type.value"
	TypeSection = "type.section"
	TypeBody    = "type.body"
	TypeLabel   = "type.label"
)

// TypeNames is every type constant above, for the same reason as TokenNames.
var TypeNames = []string{TypeTitle, TypeValue, TypeSection, TypeBody, TypeLabel}

// TypeScale is every type size the window may use, and it is closed.
//
// # Why five rungs
//
// Four cannot separate the name of a screen from the heading of a section
// inside it without one of the two being wrong. Six puts two rungs a person
// cannot tell apart into one window, which is two names for one size.
//
// # Why a value has a rung of its own rather than taking the body size
//
// A generated value is what this screen exists for and it is read character by
// character: somebody compares 666-43-6935 against what their application
// stored. At the body size that comparison is harder than it needs to be, and
// the rung costs nothing, because no other text here is read that way.
//
// # Why the ladder leaves the toolkit's base size alone
//
// Turned down when it was first proposed and still turned down: the base size
// is what every control of the toolkit measures itself from, so moving it
// moves the height of every field and button at once. A label takes a size by
// NAME since Fyne 2.6, and that is what lets this ladder exist without
// touching the base.
var TypeScale = []TextStep{
	{TypeTitle, 21, "the name of the screen a person is looking at"},
	{TypeValue, 16, "a generated value or a command, read character by character"},
	{TypeSection, 14, "the heading of one part of a screen"},
	{TypeBody, 13, "a sentence, a label, and the text inside a control"},
	{TypeLabel, 11, "the name of a field, a caption, and a group heading"},
}

// How far a corner is rounded.
//
// # Why three rather than one
//
// A card and a button rounded by the same amount read as the same kind of
// thing, and the whole point of a card is that it is not a control.
//
// # Why a band of chrome is not on this list
//
// A band touching the edge of the window is not rounded at all, and that is a
// decision rather than an omission: a rounded band leaves a sliver of window
// showing in its corners, which reads as a shape that failed to fit.
const (
	// RadiusCard rounds a card and a floating panel.
	RadiusCard float32 = 10
	// RadiusControl rounds a button, a field and a chooser.
	RadiusControl float32 = 6
	// RadiusPill rounds a badge into a capsule.
	//
	// Far larger than any badge, because the toolkit clamps a corner radius
	// to half the shorter side. Asking for half the height instead would
	// mean measuring that height first, and the height of a badge is
	// whatever its label needs.
	RadiusPill float32 = 999
)

// MarkWidth is how wide the bar beside the current item of the sidebar is.
//
// # Why it is its own number and not the border thickness
//
// They are two different things that happen to be small. A border says a shape
// is a control and runs all the way round it, so it is as thin as it can be
// while still arriving in the colour it asked for, which was measured at 2. A
// bar says WHICH of five places a person is in, read at a glance down a column,
// and is the only thing on that edge - so it is wide enough to register as a
// bar rather than as a line somebody drew by accident.
//
// Tying it to the border thickness would mean a change made for the reason
// borders exist silently changing what the sidebar says.
const MarkWidth float32 = 3

// IconStrokeTarget is how thick the line of an icon should look on screen.
//
// # Why a target rather than the number written in the file
//
// Every icon of the set is drawn with a stroke of 2 on a grid of 24. Scaled
// down to 13 pixels that stroke arrives just over one pixel wide, and at 26 it
// arrives at two, so a small icon beside a large one looks like it came from
// another set. Holding the width ON SCREEN steady means computing the stroke
// from the size the icon is drawn at, which is what the icon loader does.
//
// 1.4 rather than 1: a stroke of exactly one pixel thins to nothing at the
// display scales a desktop applies, and this set is drawn entirely in stroke.
const IconStrokeTarget float32 = 1.4

// IconStrokeFloor and IconStrokeCeiling bound the computed stroke.
//
// Without them a very small icon asks for a stroke thicker than the shape it
// draws, and a very large one asks for a hairline. Both ends are reached by
// the sizes this window actually uses, which is why they are values here
// rather than something left to the arithmetic.
const (
	IconStrokeFloor   float32 = 1.6
	IconStrokeCeiling float32 = 2.6
)

// The shadow under a panel that floats over the content, and the only shadow
// this window draws.
//
// # Why a shadow is used once and nowhere else
//
// Fyne 2.8 gained shadows and blur, so the pull to use them is strongest right
// now and that is exactly why the rule is written down: a shadow says something
// is ABOVE something else, and a card is not. Cards are separated by a step of
// the ladder, which is what Fleet and Linear do, and a window where everything
// casts a shadow says nothing about what is actually raised.
//
// The one thing genuinely above the content is the panel that opens when a kind
// is chosen. It covers the controls behind it, and without a shadow it reads as
// part of the layout rather than as something opened.
//
// # Why these numbers
//
// A blur wide enough to read as depth rather than as an outline, offset
// downwards so the light appears to come from above, and an alpha that is
// visible on a ground this dark. Judgements taken from the accepted drawing.
const (
	PanelShadowBlur   float32 = 26
	PanelShadowSpread float32 = 2
	PanelShadowDrop   float32 = 10
	PanelShadowAlpha  uint8   = 0x9C
)

// PanelShadowColour is that shadow as a colour.
//
// # Why black at an alpha rather than an entry of a ladder
//
// A shadow is an absence of light on whatever happens to be under it, not a
// colour of the palette. Tinting it would make it a coloured haze, and a haze
// over three different grounds would be three different hazes. It lives here
// rather than beside the shape that draws it because the guard on literals
// refuses a colour built anywhere under the window, and that refusal is right
// even here: "this one is not really a colour" is the sentence every colour
// could be introduced with.
func PanelShadowColour() color.NRGBA {
	return color.NRGBA{A: PanelShadowAlpha}
}

// MarkThickness is how wide the bar beside the current sidebar item is.
//
// # Why it is not the hairline
//
// Written as a hairline first and the render settled it: at one pixel the mark
// is a scratch beside the item rather than a mark on it, and the eye reads the
// background tint alone, which is one step of the ladder and not enough by
// itself. Three is the narrowest that reads as a deliberate bar.
//
// It is not a step of the spacing scale for the same reason the border width is
// not: the scale measures distances BETWEEN things, and this is the width of a
// drawn shape.
const MarkThickness float32 = 3

// ControlHeight is how tall every control in this window is.
//
// # Why one height rather than each control asking for its own
//
// A field beside a chooser beside a button, each as tall as its own content, is
// three heights in one row, and a row of three heights reads as three unrelated
// things rather than as one band of settings. One value is what makes them a
// set, and it is the same argument as the one for a closed scale of spacings.
//
// # Why this number
//
// The body rung with room above and below it for the edge and a margin, which
// is what a control needs before its text starts touching its own border. A
// judgement taken from the drawing the owner accepted, stated as one.
//
// What would change it: a rung of the type ladder taller than the body inside a
// control, which nothing asks for today.
const ControlHeight float32 = 34

// BadgeWashAlpha is how faintly a verdict colour is drawn under text of that
// same colour.
//
// # Why this is not another entry of a ladder
//
// Every shade above is fully opaque by decision, because a translucent entry
// would make its measured contrast a statement about a blend that nothing
// measured. This is not a colour: it is an alpha applied to one, and the point
// of applying it is that the text on top is STILL read against the window
// rather than against a new ground. A badge filled solidly would be a different
// pair, and a different pair is a different measurement.
//
// What would change it: an alpha high enough that the wash becomes the ground.
// At this value over the two grounds a badge is drawn on it is not, which is
// why the wash is the only place translucency appears in this window.
const BadgeWashAlpha uint8 = 0x24

// RowHeightMin and RowHeightMax bound a row of a list of values.
//
// # Why a row grows with the window at all
//
// Measured on 2026-09-08 while the drawing was being made: at 1600 by 1180 a
// list of ten values pinned to its minimum height left about 350 pixels of gap
// below it, and a card stretched to swallow that gap left the same 350 pixels
// of empty card, which reads as a fault rather than as room. Letting the rows
// take the space instead spends it on the thing the screen exists for, a larger
// target for the pointer when somebody copies a value.
//
// # Why there is a ceiling
//
// Without one, four values in a tall window become four bands the height of a
// paragraph, and the list stops reading as a list. Above the ceiling the extra
// height stays TRANSPARENT rather than growing the card, which is the whole
// difference from the variant that was measured and rejected.
//
// # Why these two numbers
//
// The floor is what a value at the value rung needs with room to breathe above
// and below it, and the ceiling is a little over half again, which is where a
// row stops looking like a row. Both are judgements taken from the drawing the
// owner accepted, and are stated as such.
const (
	RowHeightMin float32 = 42
	RowHeightMax float32 = 64
)

// HairlineThickness is how thick a drawn line is, as opposed to a stroke
// around a shape.
//
// # Why one, where BorderThickness above is two, and why that is not a
// contradiction
//
// BorderThickness is about a STROKE, which the toolkit centres on the edge of
// the rectangle it is given. At a width of one, half of that stroke falls on
// either side of a pixel boundary and every pixel of it arrives as a blend, so
// the colour asked for is not the colour drawn. That was measured on 2026-08-24
// and the number beside it is the fix.
//
// A line here is a filled rectangle laid out by the layout, so it lands on
// whole pixels the way any other shape does and arrives in the colour it was
// given. Two would make every divider in the window twice as loud as the
// drawing asks for, and a divider is the one thing that must not be.
//
// What would change it: a display scale where a rectangle of one stops landing
// on whole pixels, which is the same thing that would change the other value.
const HairlineThickness float32 = 1

// IconGrid and IconStrokeInFile are facts about the icon set rather than
// decisions of ours.
//
// Every file of the set is drawn on a square of 24 with a stroke of 2, and the
// loader needs both numbers to scale one into the other. They are here because
// the guard refuses a bare number anywhere under the window and draws no line
// between a fact and a decision, which is the right call: "this one is only a
// fact about a file" is a sentence every value could be introduced with.
//
// What would change them: another icon set. That is also what makes them worth
// naming, since a set drawn on a grid of 16 would need every call site found.
const (
	IconGrid         float32 = 24
	IconStrokeInFile float32 = 2
)

// How large an icon is drawn.
//
// # Why three sizes and not one
//
// An icon inside a control has to sit under the cap height of the label beside
// it, an icon naming a screen has to be found before the label is read, and
// the sheet that proves every icon renders wants them large enough to see a
// broken path. Those are three jobs and no single size does all three.
//
// # Why they are not steps of the spacing scale
//
// A spacing step is a distance BETWEEN things. This is the side of a square,
// and the two ladders answer to different constraints: an icon of 16 beside
// text of 13 is a judgement about optical weight, not about rhythm.
const (
	// IconSmall sits inside a control, beside a label at the body size.
	IconSmall float32 = 15
	// IconMedium names a screen in the sidebar and heads a panel.
	IconMedium float32 = 17
	// IconLarge is for the sheet that proves every icon renders at all.
	IconLarge float32 = 26
)

var shadesByName = func() map[string]Shade {
	byName := make(map[string]Shade, len(Neutral)+len(Accent)+len(Semantic)+1)
	for _, ladder := range [][]Shade{Neutral, Accent, Semantic} {
		for _, shade := range ladder {
			byName[shade.Name] = shade
		}
	}
	byName[White.Name] = White
	return byName
}()

var tokensByName = func() map[string]Token {
	byName := make(map[string]Token, len(Tokens))
	for _, token := range Tokens {
		byName[token.Name] = token
	}
	return byName
}()

var typeByName = func() map[string]TextStep {
	byName := make(map[string]TextStep, len(TypeScale))
	for _, step := range TypeScale {
		byName[step.Name] = step
	}
	return byName
}()

// Ink is the colour a view asks for by the name of a token.
//
// # Why an unknown name answers with text rather than with nothing
//
// The same reason Named above does, in the same words: a zero colour is
// transparent black and renders as a hole nobody can see, so a typo would
// become an invisible control rather than a visible fault. A view cannot stop,
// so the wrong answer is at least a loud one. The pair of tests closing the
// register from both sides is what keeps this branch unreachable.
func Ink(token string) color.NRGBA {
	entry, found := tokensByName[token]
	if !found {
		return shadesByName["neutral.12"].NRGBA()
	}
	return shadesByName[entry.Shade].NRGBA()
}

// TokenOf is the whole entry, for the bridge and for the test that measures it.
func TokenOf(name string) (Token, bool) {
	entry, found := tokensByName[name]
	return entry, found
}

// ShadeOf is one step of one ladder, for a test walking the ladders.
func ShadeOf(name string) (Shade, bool) {
	entry, found := shadesByName[name]
	return entry, found
}

// TypeSize is the height of type a view asks for by name.
//
// An unknown name answers with the body size, which is what every screen would
// have used had this ladder never existed. Loud is not available here either,
// and a rung wrong by one step is far less harmful than text of no size at all.
func TypeSize(name string) float32 {
	if step, found := typeByName[name]; found {
		return step.Size
	}
	return typeByName[TypeBody].Size
}
