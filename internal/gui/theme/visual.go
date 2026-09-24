package theme

import "image/color"

// The visual system the window is drawn with: two ladders of twelve shades, a
// ladder of type sizes, and a register of tokens naming what each shade is for.
//
// # Why this replaced the palette of the first window
//
// That palette, deleted with DEC-253 and kept as history in docs/17 section 5,
// answered which colour carries which MEANING. An expectation of rejection was
// red because the manifest says rejection, and that entry was measured against
// the window.
//
// This file answers the question that made the owner call the window generic
// on 2026-09-08: which colour carries which PLACE. A card apart from the
// window, a control apart from the card, an edge apart from a divider. The
// first palette had two shades of ground and one of text, so there was nothing
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
// # Why a cool graphite rather than a warm near-black
//
// Until 2026-09-24 the ladder was bent warm and started at 0x100F0E. The owner
// looked at the finished window and said the colours were bland: too black,
// too grey, flat. Measured on the render the same day, 74 to 93 per cent of
// every screen was within a few steps of black, and the difference between the
// window and a card was too small to read as depth.
//
// So the ground rises to a graphite and the steps near the bottom are spread
// further apart, which is where depth is read on a dark theme. The tint is
// cool rather than warm because the accent above is cool, and a warm ground
// under a cool accent reads as two palettes. How far the ground can rise is
// not a taste: border.control has to clear 3.0 on the ground a pointer gives a
// control, and mark.focus has to clear 3.0 against border.control, so every
// step up at the bottom pushes the edge up and the focus mark towards white.
// The steps are computed from a target luminance each, in OKLCH, rather than
// picked by eye, and the register below is what says they still pass.
var Neutral = []Shade{
	{"neutral.1", 0x1B, 0x1E, 0x23, "the window itself"},
	{"neutral.2", 0x24, 0x28, 0x2D, "a ground set apart from the window: a card, the sidebar, a band of chrome"},
	{"neutral.3", 0x2B, 0x2F, 0x35, "the inside of a control at rest, and every other row of a list"},
	{"neutral.4", 0x30, 0x34, 0x3A, "a ground one step louder than a control at rest, which no token takes today"},
	{"neutral.5", 0x35, 0x3A, 0x41, "the row under the pointer"},
	{"neutral.6", 0x3D, 0x42, 0x49, "the quiet line: between two rows, and around a card"},
	{"neutral.7", 0x48, 0x4E, 0x56, "one step louder than the quiet line, for an edge that is found rather than followed"},
	{"neutral.8", 0x57, 0x5E, 0x66, "the loudest shade that is still an edge rather than a mark"},
	{"neutral.9", 0x66, 0x6D, 0x75, "a filled shape carrying no text of its own"},
	{"neutral.10", 0x7D, 0x84, 0x8D, "the edge of a control, which has to be seen rather than followed"},
	{"neutral.11", 0xAC, 0xB4, 0xBD, "text read second: the name above a field, the sentence beside a value"},
	{"neutral.12", 0xEB, 0xEF, 0xF4, "text read first"},
}

// Accent is the ladder the one action of a screen is drawn from, and the
// structure around it.
//
// # Why not green, amber or red, and it is an argument rather than a taste
//
// This product already spends green, amber and red on things that MEAN
// something: an expectation of acceptance, of a warning, of rejection. An
// accent taken from any of those hues would put a colour saying press this
// beside the same hue saying this was rejected.
//
// # Why blue, and it is a choice rather than an argument
//
// Until 2026-09-24 it was violet, chosen as the hue furthest from all three.
// The owner said the violet did not suit the window, and picked blue that day
// from renders of the same screens in blue, teal and fuchsia, with the ladder
// below identical in lightness for all three. Blue also shares the tint of
// the graphite under it, so the two ladders read as one palette. Teal was the
// most colourful of the three and fuchsia the one nearest the red of a
// rejection, which is the one argument against a hue this comment can make.
//
// # How much of it is spent, changed on 2026-09-24
//
// Until then: one filled shape per screen and the mark on the sidebar item,
// which measured as 0.1 to 1.2 per cent of the pixels of a screen, and the
// owner called the result bland. He chose the accent in navigation and in
// structure as well: the ground of the open screen in the sidebar, the icons
// of the quiet controls, the captions over groups, the line under a section
// heading, the header of a table. It is still not on text read first and not
// on the grounds of controls, so the one filled action stays the loudest thing
// on a screen.
//
// Every step is computed from a target luminance in OKLCH, shared by every
// hue that was rendered, so the hue could change without one pair in the
// register below moving.
var Accent = []Shade{
	{"accent.1", 0x1A, 0x21, 0x2B, "the window, tinted, for a screen that is entirely about one action"},
	{"accent.2", 0x21, 0x2A, 0x39, "a ground set apart, tinted: the header of a table"},
	{"accent.3", 0x25, 0x33, 0x48, "the inside of a chosen control at rest, and the sidebar item of the open screen"},
	{"accent.4", 0x27, 0x39, 0x53, "that ground under the pointer"},
	{"accent.5", 0x29, 0x3F, 0x5F, "that ground while pressed"},
	{"accent.6", 0x2A, 0x47, 0x71, "the quiet edge of a chosen control"},
	{"accent.7", 0x30, 0x55, 0x8C, "one step louder, and the line under a section heading"},
	{"accent.8", 0x33, 0x6B, 0xBC, "the edge of the field a search is typed into"},
	{"accent.9", 0x29, 0x71, 0xD5, "the fill of the one action a screen is for"},
	{"accent.10", 0x3F, 0x85, 0xEC, "that fill under the pointer"},
	{"accent.11", 0x72, 0xAB, 0xFF, "the accent where it has to be READ: an icon, a prompt, a caption, the mark on the current item"},
	{"accent.12", 0xDA, 0xE8, 0xFE, "text on a tinted ground"},
}

// Semantic is the three colours that carry a verdict rather than a place.
//
// They were stated here rather than borrowed from the first palette because a
// contrast is a statement about a PAIR, and these are read on the grounds of
// this ladder while that palette was measured against a window of another
// colour. One entry serving both would have reported one number about two
// pairs.
var Semantic = []Shade{
	{"semantic.success", 0x3D, 0xA7, 0x66, "a value inside the range it belongs to, and a safe harbour a provider can claim"},
	{"semantic.warning", 0xE8, 0xA3, 0x3D, "something to look at before trusting it"},
	// Lightened on 2026-09-24 in the same hue, from 0xE05561. The graphite
	// ground made the old red measure 4.13 on a card where a word needs 4.5.
	{"semantic.danger", 0xEE, 0x62, 0x6D, "a refusal, and a value outside what the schema allows"},
}

// White is the one colour here that belongs to no ladder.
//
// It exists because the label on the accent fill is read ON that fill, and the
// top of the neutral ladder is not light enough for it: neutral.12 on accent.9
// measures 4.12 where WCAG 1.4.3 asks for 4.5. White measures 4.75. Both
// re-measured on 2026-09-24 on the blue accent, and the violet one before it
// gave 4.35 and 5.14, so the reason survived the change of hue. A ladder
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
	// The two states of bg.current under a pointer, since 2026-09-24.
	BgCurrentHover   = "bg.current-hover"
	BgCurrentPressed = "bg.current-pressed"
	BgHover          = "bg.hover"
	BgPressed        = "bg.pressed"
	BgSelection      = "bg.selection"
	BgHeader         = "bg.header"

	LineQuiet     = "line.quiet"
	LineSection   = "line.section"
	BorderControl = "border.control"
	BorderPressed = "border.pressed"

	FillPrimary = "fill.primary"
	FillHover   = "fill.hover"
	FillPressed = "fill.pressed"
	MarkCurrent = "mark.current"
	MarkActive  = "mark.active"
	MarkFocus   = "mark.focus"
	InkIcon     = "ink.icon"

	TextPrimary   = "text.primary"
	TextSecondary = "text.secondary"
	TextAccent    = "text.accent"
	TextCaption   = "text.caption"
	TextOnFill    = "text.on-fill"
	TextOnMark    = "text.on-mark"

	StateSuccess = "state.success"
	StateWarning = "state.warning"
	StateDanger  = "state.danger"
)

// TokenNames is every token constant above, so a test can close the register
// from the side the constants are on. The same reason StepNames exists.
var TokenNames = []string{
	BgWindow, BgSurface, BgControl, BgCurrent, BgCurrentHover, BgCurrentPressed,
	BgHover, BgPressed, BgSelection, BgHeader,
	LineQuiet, LineSection, BorderControl, BorderPressed,
	FillPrimary, FillHover, FillPressed, MarkCurrent, MarkActive, MarkFocus, InkIcon,
	TextPrimary, TextSecondary, TextAccent, TextCaption, TextOnFill, TextOnMark,
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
	// ThresholdSelection is ours, and it is a CEILING dressed as a threshold.
	//
	// 🔴 It exists because 3.0 is arithmetically unreachable here, which was
	// measured rather than assumed. A selection is a fill UNDER text, so it
	// answers to two numbers at once: it has to be seen against the ground of
	// the field, and the text has to stay readable ON it. Asked of every step
	// of both ladders on 2026-09-09 with tools/zeton.py:
	//
	//	3.0 against bg.control, text still at 4.5   ZERO steps
	//	2.0 against bg.control, text still at 4.5   neutral.8 (2.55 / 5.41)
	//	                                            accent.8  (2.60 / 5.32)
	//
	// That is the same shape of answer as DEC-179, where a focus indicator
	// under text turned out to have a hard arithmetic limit, and the same
	// answer follows: the number is stated as ours, with the measurement
	// beside it, rather than a WCAG threshold quietly lowered to fit.
	//
	// Set at 2.5 rather than at 2.60, so the entry has somewhere to move
	// within the ladder without the guard turning red.
	//
	// 🔴 Re-measured on 2026-09-24 on the graphite ladder, and the room is
	// gone: accent.8 measures 2.54 against bg.control and 4.58 under text,
	// because lifting the inside of a field pushes the first number down and
	// the text on the selection pushes the second. The sentence that stood
	// here also said neutral.8 would pass if grey were preferred. It no longer
	// does, at 2.05. A later ground lifted any further has to be measured
	// against this pair first.
	ThresholdSelection = 2.5
	// ThresholdStateChange is ours, and it answers a question WCAG does not
	// ask: whether a person can see that a control changed under their pointer.
	//
	// It exists because the fill of a pressed button cannot be measured against
	// the band it stands on. Measured on 2026-09-09, the accent ladder has
	// exactly ONE step that both clears 3.0 against that band and carries white
	// at 4.5 - the step the button rests on. So the states have to move off it,
	// and what matters about them is the difference from the RESTING fill,
	// which is always underneath them: nobody ever sees a hovered button
	// without having seen the same button a moment earlier.
	//
	// Set at 1.5, which the two steps chosen clear at 1.58 and 1.97 on the
	// blue accent of 2026-09-24, and cleared at 1.67 and 2.10 on the violet.
	ThresholdStateChange = 1.5
)

// Tokens is every token, the shade behind it, and what it is measured against.
//
// # Which numbers in the comments below are about today
//
// The ladders changed on 2026-09-24, from a warm near-black with a violet
// accent to a cool graphite with a blue one. A number below that carries an
// earlier date is a record of that day and was measured on the old ladders.
// A number stated as what a pair measures NOW was re-measured on 2026-09-24,
// and the test beside this file measures every pair on every run, so where
// the two disagree the test is right.
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
//     They became neutral.11, and their quietness moved off lightness and
//     onto size and capitals, which the ladder of type sizes below carries.
//     Since 2026-09-24 they are text.caption, in the accent.
//   - the sentence beside a value was neutral.10, measuring 3.25 on the row
//     under the pointer. It is neutral.11 here.
//   - the mark on the current sidebar item was accent.9 on neutral.4, which
//     measures 2.91 against a threshold of 3.0. It is accent.11, measuring
//     5.88.
//   - the label on the accent fill was already white and stays white, which
//     is the one of the five that needed no change.
//
// None of the five changed the direction: the accent stayed the accent and
// quiet stayed quieter than loud. What changed is which rung of the ladder
// each one took.
var Tokens = []Token{
	{BgWindow, "neutral.1", ThresholdGround, nil,
		"the window, which everything else is measured against"},
	{BgSurface, "neutral.2", ThresholdGround, nil,
		"a card, the sidebar and a band of chrome, set apart from the window by one step rather than by a shadow"},
	{BgControl, "neutral.3", ThresholdGround, nil,
		"the inside of a field or a chooser, and every other row of a list"},
	// Tinted since 2026-09-24, and neutral.4 until then. The owner chose the
	// accent in navigation, and the open screen is the one place of it every
	// screen shares. The chosen option of a panel stands on it too, so being
	// chosen looks the same in both.
	{BgCurrent, "accent.3", ThresholdGround, nil,
		"the sidebar item naming the screen that is open, and the chosen option of a panel"},
	// The pointer on the open screen, which is where it always is right after
	// a click. Until these two existed the pointer put bg.hover there, a grey
	// that covered the tint, and the live window of 2026-09-24 showed it.
	{BgCurrentHover, "accent.4", ThresholdGround, nil,
		"the sidebar item of the open screen with the pointer on it, still tinted"},
	{BgCurrentPressed, "accent.5", ThresholdGround, nil,
		"that item held down, one step further along the same tint"},
	{BgHover, "neutral.5", ThresholdGround, nil,
		"the row under the pointer, and the ground of any control the pointer is over"},
	// 🔴 The ground of a control being pressed, and the one entry of this
	// register whose NEIGHBOUR had to move with it.
	//
	// Radix puts the states of a control on steps 3 to 5, and this ladder
	// spent 4 on the open screen of the sidebar until that moved to the accent
	// on 2026-09-24, so a pressed control lands on 6. That step is the quiet
	// line, which is the first sign it is being asked to do a second job, and
	// the measurement says so outright: border.control stands 2.68 on it where
	// WCAG 1.4.11 asks for 3.0. It clears 3.03 on bg.hover and falls off one
	// step later. Re-measured on the graphite. The warm ladder gave 2.80 and
	// 3.25, the same shape.
	//
	// So a pressed control does not keep its resting edge. That is not a
	// workaround for a shortage of steps - it is what pressing looks like here,
	// and border.pressed below carries it.
	{BgPressed, "neutral.6", ThresholdGround, nil,
		"the ground of a control under the finger, one step past the ground it takes under the pointer"},
	// 🔴 The one ground in this register that is NOT a ground, and it is the
	// entry the owner asked for on 2026-09-09: "I select the seed or how many
	// and I cannot see it".
	//
	// Until then the toolkit's ColorNameSelection was answered with bg.current,
	// which is the sidebar item of the open screen - one step of the ladder
	// away from the inside of a field, measuring 1.09 where a person has to see
	// which characters they picked. Nobody chose that: the token was written
	// for the sidebar and the selection got it because both wanted "a ground
	// slightly lighter than the one under it". That is the same family as
	// ColorNameFocus and ColorNamePrimary, where one toolkit colour serves
	// several jobs at once, and the third time this project has paid for it.
	//
	// 🔴 It serves SIX jobs here, counted in the toolkit source rather than
	// assumed: selected text (widget/selectable.go:307), the selected row of a
	// list, the mark on a table cell (widget/table.go:1218, which the manifest
	// browser draws), a tree node, a grid tile and a file icon. So this shade
	// is measured against every ground any of those stand on, not just the
	// inside of a field.
	//
	// The accent rather than the grey of neutral.8: grey lands between
	// bg.hover and bg.pressed, so a selection would look like a control the
	// pointer happens to be over. The accent is the one hue in this palette
	// that never carries a verdict, and selection is the only other thing in
	// this window a person makes happen on purpose. Written when the accent was
	// violet, and the reason does not depend on which hue it is. Grey has also
	// stopped passing since the graphite of 2026-09-24, at 2.05.
	{BgSelection, "accent.8", ThresholdSelection, []string{BgWindow, BgSurface, BgControl},
		"the ground under text somebody selected, and under the row or cell a list has marked"},
	// The toolkit paints the header row of a table in its own dark grey when
	// nobody answers for it, which on this ground was a band of a third colour
	// belonging to neither ladder. Joined on 2026-09-24 with the accent in
	// structure.
	{BgHeader, "accent.2", ThresholdGround, nil,
		"the header row of a table, tinted so the names of the columns read as structure rather than as data"},

	{LineQuiet, "neutral.6", ThresholdGuide, []string{BgWindow, BgSurface},
		"the line between two rows and the edge of a card, which the eye follows rather than reads"},
	{LineSection, "accent.7", ThresholdGuide, []string{BgWindow, BgSurface},
		"the line running from a section heading to the far edge, which says where a part of a screen begins"},
	{BorderControl, "neutral.10", ThresholdControl, []string{BgWindow, BgSurface, BgControl, BgHover},
		"the edge saying a shape is a control rather than a word, which DEC-211 settled has to be drawn"},
	{BorderPressed, "neutral.11", ThresholdControl, []string{BgPressed},
		"the edge of a control being pressed, which brightens because the resting edge measures 2.68 on the ground pressing gives it"},

	{FillPrimary, "accent.9", ThresholdControl, []string{BgSurface},
		"the fill of the one action a screen is for, which is the only thing carrying hierarchy here"},
	// 🔴 The two entries that made the label change colour, and the arithmetic
	// that left no other road. Measured on 2026-09-09.
	//
	// A filled button under the pointer has to look different from one at rest,
	// and the only thing it has to say that with is its fill. Moving that fill
	// moves the ground its label is read on, and the two thresholds close in
	// from opposite sides:
	//
	//   - for white to reach 4.5 the fill needs a luminance of at most 0.1833
	//   - for the dark ink to reach 4.5 it needs a luminance of at least 0.1968
	//
	// The two do not meet. Any fill between them carries NO legible label at
	// all, which is the same shape of finding as DEC-179 and is why this is
	// written down rather than left as a choice somebody made.
	//
	// accent.9 sits at 0.1544, below the gap, so it takes white at 5.14.
	// accent.10 sits at 0.1983, above it, so white falls to 4.23 and the dark
	// ink clears at 4.53. There is no step in between, so the ink changes with
	// the state.
	//
	// The step DOWN was measured too and it fails for the other reason:
	// accent.8 carries white at 6.29 and stands 2.85 on a card, under the 3.0 a
	// control needs to be found at all.
	// 🔴 The two states of the filled button, and they go DOWN the ladder.
	//
	// They went up until 2026-09-09, which made the fill lighter than white
	// could be read on, so the label turned dark to compensate. The owner saw
	// the sheet of states and refused it: nobody changes the colour of the text
	// on a button. He is right, and going down keeps one ink for every state.
	//
	// Measured against the RESTING fill rather than against the band the button
	// stands on, and the reason is beside ThresholdStateChange: the ladder has
	// one step that is both visible on that band and readable in white, so
	// there is no room for three. What a person has to see here is that
	// something changed, and the resting fill is what it changed FROM.
	//
	// Re-measured on 2026-09-24 on the blue accent:
	//
	//	rest     accent.9   white 4.75, and 3.12 against the band
	//	hover    accent.7   white 7.49, 1.58 from rest
	//	pressed  accent.6   white 9.38, 1.97 from rest
	{FillHover, "accent.7", ThresholdStateChange, []string{FillPrimary},
		"the fill of the one action of a screen while the pointer is on it, a shade down from where it rests"},
	{FillPressed, "accent.6", ThresholdStateChange, []string{FillPrimary},
		"that fill under the finger, one shade further down again"},

	// The grounds of a check box joined on 2026-09-15 (DEC-245): the mark in a
	// ticked box is this shade, on the ground of a field and on that ground
	// under the pointer. A MARK rather than text.accent, and the difference
	// is measured: the same shade on bg.pressed is 4.33 (4.49 on the violet),
	// which fails a label and passes a mark, and a tick is a mark.
	{MarkCurrent, "accent.11", ThresholdControl, []string{BgSurface, BgCurrent, BgControl, BgHover, BgPressed},
		"the bar beside the sidebar item the window is on, and the mark in a ticked check box"},
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
	// Measured across it, on the blue of 2026-09-24: accent.9 gives 1.26,
	// accent.10 gives 1.04, accent.11 gives 1.62. Only accent.12 clears the
	// threshold, at 3.05, and its stated job is text on a tinted ground. So the
	// mark rides the top of the neutral ladder, where it clears at 3.27. The
	// violet gave 1.26, 1.04, 1.60, 3.03 and 3.44, so the reason held across
	// the change of hue.
	//
	// What this costs, said plainly: a ticked box and a focused field are not
	// in the accent. A focus mark that cannot be seen is worse than one that is
	// not tinted.
	{MarkActive, "neutral.12", ThresholdControl, []string{BorderControl, BgControl, BgSurface},
		"what the toolkit paints on a stock control that is chosen or holds the keyboard, which has to differ from the resting edge it replaces"},

	// 🔴 The edge of the control the keyboard is on, and the second token in
	// this register measured against another MARK rather than against a
	// ground.
	//
	// It replaced a ring drawn INSIDE the control on 2026-09-09, after the
	// owner looked at one and said the white ring in the button was awful. He
	// was right twice over: it is not what anybody else does, and it made the
	// filled button carry two shapes. Read up on it rather than guessed - NN/g
	// and the WCAG 2.2 guidance agree that focus is an OUTLINE of at least two
	// pixels, that it needs 3:1 against what is adjacent to it, and that it
	// must differ from the UNFOCUSED state of the same control. Hover is a
	// different thing and changes the ground instead.
	//
	// That last requirement is what settled the shade, and it settled it by
	// killing the obvious answer. The accent this window reads with, accent.11,
	// measures 1.62 against border.control, so a focused edge in it would be a
	// colour change nobody could see. Measured across both ladders, re-measured
	// on the graphite and blue of 2026-09-24:
	//
	//	accent.11   1.62 against the resting edge   REFUSED
	//	neutral.11  1.80                            REFUSED
	//	accent.12   3.05                            taken
	//	neutral.12  3.27, and it is white, which is what was refused
	//
	// So the edge is the top of the accent ladder: 3.05 against the edge it
	// replaces, 10.86 against the inside of a field, 3.84 on the filled button
	// and 6.05 on that fill under the pointer. One shade, every control, which
	// is the whole point of it.
	//
	// 🔴 This pair is the one that decides how far the ground may rise. Lifting
	// the ground lifts bg.hover, border.control has to clear 3.0 on bg.hover,
	// and this shade has to clear 3.0 against border.control - so every step up
	// at the bottom of the neutral ladder pushes this shade towards white. At
	// 3.05 there is almost nothing left.
	//
	// bg.hover joined the grounds on 2026-09-14, the day the text field began
	// answering the pointer: a field with the keyboard on it and the pointer
	// over it draws this edge on that ground, as a quiet button already did.
	// line.quiet joined the same day, because the frame of a scrolling area
	// lands on the edge of the card it holds and recolours it.
	// The three grounds of the open sidebar item joined on 2026-09-24, when
	// that item began drawing its own ground: the keyboard on it draws this
	// edge on whichever of the three the pointer has put it on.
	{MarkFocus, "accent.12", ThresholdControl,
		[]string{BorderControl, LineQuiet, BgWindow, BgSurface, BgControl, BgHover, FillPrimary, FillHover,
			BgCurrent, BgCurrentHover, BgCurrentPressed},
		"the edge of the control the keyboard is on, which every control shows the same way"},
	// A mark rather than text, since an icon beside a label is found by its
	// shape and read by the words next to it: WCAG 1.4.11, not 1.4.3.
	{InkIcon, "accent.11", ThresholdControl, []string{BgWindow, BgSurface, BgControl, BgCurrent, BgCurrentHover, BgCurrentPressed, BgHover, BgPressed},
		"the icon on a quiet button and on a sidebar item, which is most of the accent a screen carries"},

	{TextPrimary, "neutral.12", ThresholdText, []string{BgWindow, BgSurface, BgControl, BgCurrent, BgCurrentHover, BgCurrentPressed, BgHover, BgPressed, BgSelection},
		"the value, the label and every word read first"},
	// bg.header joined on 2026-09-24: the names of the columns of the manifest
	// browser are written in this ink (manifestview/table.go, CreateHeader).
	{TextSecondary, "neutral.11", ThresholdText, []string{BgWindow, BgSurface, BgControl, BgCurrent, BgHover, BgPressed, BgHeader},
		"the name above a field and the sentence explaining a value, which captions shared until they took text.caption"},
	// bg.hover joined the grounds on 2026-09-14 with the panel a kind is
	// picked from: the chosen row keeps its accent name under the pointer,
	// and the pointer moves the ground. Computed before it was added, since
	// the test beside this file would otherwise have been the first to
	// know: accent.11 on neutral.5 was 5.20, and is 4.90 on the graphite and
	// blue of 2026-09-24.
	{TextAccent, "accent.11", ThresholdText, []string{BgWindow, BgSurface, BgCurrent, BgHover},
		"the prompt of a command line, the icon heading a card or a notice, and the name of the kind that is chosen"},
	// Its own token rather than text.accent, although it is the same shade
	// today: a caption is a heading over a group and the accent text is a
	// thing chosen, and a later change to one should not move the other.
	{TextCaption, "accent.11", ThresholdText, []string{BgWindow, BgSurface},
		"a caption over a group: the heading of a column of the sidebar and the name of a fact on a card"},
	{TextOnFill, "white", ThresholdText, []string{FillPrimary, FillHover, FillPressed},
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
	// 🔴 Two grounds were added to this token on 2026-09-09, and the entry is
	// worth reading before a third is.
	//
	// It was written for one thing: the flood the toolkit paints on a stock
	// control holding the keyboard. The two fills beside it are a different
	// thing that happens to need the same answer - a dark ink on a light violet
	// - because the accent ladder above accent.9 carries no white label at all.
	//
	// So this is the ink for a light accent ground, whatever put the control on
	// one. What would make that wrong is a ground arriving here that is light
	// for some third reason and wants a different ink, and the measurement is
	// the guard rather than the wording.
	//
	// 🔴 Corrected on 2026-09-24: the two fills are no longer here. They left
	// when the states of the filled button went DOWN the ladder on 2026-09-09
	// and kept white, which the grounds of text.on-fill say, and this comment
	// went on naming them for two weeks. Today the one ground is mark.current,
	// where this ink measures 7.15 on the blue (7.55 on the violet).
	{TextOnMark, "neutral.1", ThresholdText, []string{MarkCurrent},
		"the label and the icon on a light accent ground: the flood the toolkit paints on a focused control"},

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
	// RadiusCheck rounds the box of a check box. Less than a control, because
	// the box is eighteen across and at six it read as a radio button on the
	// render of 2026-09-15 (DEC-245). A radius follows the size of the shape
	// it rounds, so a small shape gets a small one.
	RadiusCheck float32 = 4
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

// PanelWidth and PanelHeight bound the panel a kind is picked from, which is
// the one thing in this window that floats over the content.
//
// # Why the panel has a size of its own rather than the width of its chooser
//
// The chooser under it is as wide as its widest value, which is a word. The
// panel shows that word, a second column beside it and a heading above, so a
// panel the width of the chooser would cut its own rows. It is one width for
// every list it opens over, so a person opening the second chooser sees the
// same panel as over the first.
//
// # Why the height is a ceiling and not a size
//
// Below it the panel is as tall as its rows, and above it the rows scroll.
// The ceiling matters because Fyne does not clip: a panel of fixed height
// with more rows than fit would let the rows run out of its bottom edge and
// draw on whatever stands under it, without failing, which is what B-258
// measured on the mock. The rows are in a scrolling area, which is the one
// container that does clip, and the ceiling is what makes that area finite.
//
// # Why these numbers
//
// Both taken from the accepted drawing, tools/makieta -otwarty. The height
// leaves room under a panel opened at StartHeight for the shadow to be seen
// below it, and the width holds the longest heading with the check mark
// beside it. What would change them: a second column wider than a mask, or a
// window opened smaller than StartHeight, where the panel takes the room it
// has rather than this.
const (
	PanelWidth  float32 = 430
	PanelHeight float32 = 470
)

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
	// CheckBox is the side of the box of a check box: a small icon with room
	// for its edge, so the mark inside it is the small icon and nothing has to
	// be scaled.
	CheckBox float32 = 18
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
