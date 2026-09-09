// Package quickview is the screen that answers the shortest question this tool
// takes: one value, now.
//
// # Who this screen is for, and why it is a screen of its own
//
// The persona it serves is the manual tester of docs/01, who wants a valid
// identifier of a particular kind and wants it in seconds, and who is not
// writing a recipe. Putting the same thing inside the recipe editor would have
// been less work and would have hidden the first thing this product promises
// inside a panel that persona has no reason to open.
//
// # Why the value appears without a button being pressed
//
// The promise is under five seconds. A screen that asks somebody to choose a
// generator and then press Produce has one step too many for that, so choosing
// is what produces, and the button beside it is for asking again.
package quickview

import (
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/commandline"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
	"github.com/donislawdev/qadatagenerator/internal/quick"
)

// Clipboard is where a value goes when somebody presses Copy.
//
// It is a port rather than the toolkit's own clipboard for the reason every
// other outside thing in this window is: a screen built to be read rather than
// shown has no application behind it, and the toolkit's clipboard asks for one.
// It is also the first time anything this tool produces leaves its own files
// and channels, so the way out is named rather than reached for.
type Clipboard interface {
	SetContent(string)
}

// View is the screen.
type View struct {
	text i18n.Catalogue
	// locale is which locale the values are drawn from. It is a field rather
	// than a constant because which roles a value reads depends on it, and the
	// screen that offers a field has to ask the same question the mode does.
	locale string
	// generators is the register this build carries, asked once and kept, so
	// that the list on screen and the list the mode knows are one list.
	generators []string
	clipboard  Clipboard

	chooser *parts.Select
	locales *parts.Select
	gender  *parts.Select
	// howMany chooses how many values to produce.
	//
	// 🔴 A list rather than a field, and the mode ACCEPTS more than this list
	// offers. That is the same distinction Genders() writes about a few files
	// away: what a command line takes and what a screen offers are different
	// questions, and a box a person can type any number into invites one that
	// fills the window with a wall nobody reads. The command this screen shows
	// carries the number, so anybody wanting more has the line to paste.
	howMany *parts.Select
	born    *widget.Entry
	seedBox *widget.Entry
	copy    *parts.PrimaryButton
	again   *parts.Button

	// settings holds whichever fields the chosen kind actually reads, and it is
	// refilled rather than hidden.
	//
	// 🔴 A row rather than the form of the toolkit, changed on 2026-09-09. A
	// form draws a column of labels beside a column of controls, which is the
	// right shape for a page of settings and the wrong one for a strip of
	// chrome four fields wide: it spent a third of the window on a column of
	// right aligned words. The label now stands ABOVE its control, which is
	// what the accepted drawing asks for and what paint.FieldLabel draws.
	settings *fyne.Container
	// showing is which roles have a field on screen, so that a test can ask
	// the screen rather than count pixels. It replaces reading the items of a
	// form, which no longer exists.
	showing map[string]bool

	// values are what the mode produced, kept as a list rather than as one
	// joined string.
	//
	// The screen draws them as numbered rows now, and a row is a thing the
	// layout can space, band and measure. Joined into a label they were one
	// object as tall as ten lines, which is why the count of them could not
	// change anything about how they looked.
	values []string
	told   *widget.Label

	// result is the card and its heading, or the sentence saying why there is
	// no card. Refilled by settle, for the reason settings is.
	result *fyne.Container
	// about is the panel describing the chosen kind, refilled whenever the kind
	// or the locale changes, which are the only two things it depends on.
	about *fyne.Container
	// terminal is the band at the foot, held so that settle can take it away
	// when there is no line to show.
	terminal fyne.CanvasObject

	// command is the line that would do this in a terminal, and the button
	// beside it puts that line in the clipboard.
	//
	// It is shown rather than run. A window able to run the command line would
	// be a second way to do everything on this screen, and the two would
	// answer differently the first time one of them changed.
	command     *widget.Label
	copyCommand *parts.Button

	// seed is the one this value came from, kept so that the screen can say it
	// and so that Copy and the label cannot disagree about which value is on
	// screen.
	seed    string
	content fyne.CanvasObject
}

// New builds the screen.
func New(text i18n.Catalogue, clipboard Clipboard) *View {
	view := &View{text: text, clipboard: clipboard}
	view.generators = quick.Generators(view.locale)
	view.build()
	return view
}

// Content is what the frame is given.
func (v *View) Content() fyne.CanvasObject { return v.content }

func (v *View) build() {
	v.chooser = parts.NewSelect(v.generators, func(string) { v.draw() })
	// The data locale, beside the interface language and never the same
	// control, which is the rule the screen next door already follows. It also
	// decides which values take account of a gender, so it sits above the two
	// fields it governs.
	v.locales = parts.NewSelect(quick.Locales(), func(picked string) {
		v.locale = picked
		v.reloadGenerators()
		v.draw()
	})
	v.gender = parts.NewSelect(v.genderChoices(), func(string) { v.draw() })
	v.howMany = parts.NewSelect(countChoices, func(string) { v.draw() })
	v.born = widget.NewEntry()
	// The shape of the date is on the field rather than only in the refusal
	// that follows getting it wrong, which is the same reason the seed field
	// shows the seed that was used.
	v.born.SetPlaceHolder(quick.DateShape)
	v.born.OnChanged = func(string) { v.draw() }
	// The seed is a field rather than a line of text, and that is what makes
	// this screen able to do what the flag does. The screen fills it with the
	// one it used when nobody said, so asking for the same value again is
	// reading it off the screen rather than out of a log.
	v.seedBox = widget.NewEntry()
	v.seedBox.OnChanged = func(typed string) {
		if typed == v.seed {
			return
		}
		v.seed = typed
		v.draw()
	}

	v.copy = parts.NewPrimaryButton(v.text.T(i18n.QuickCopy), v.putInClipboard)
	v.again = parts.NewButton(v.text.T(i18n.QuickAgain), v.drawAgain)

	// Monospace, because this is a line to be read character by character and
	// then typed or pasted.
	//
	// 🔴 Cut rather than wrapped, changed on 2026-09-08. It wrapped until then,
	// on the argument that a command ending in an ellipsis is not a command.
	// That argument was answered by the render on the day the line was put in a
	// box of its own: a wrapping label asks for a height of several lines
	// before anything has given it a width, so the box stood about a hundred
	// pixels tall around a line of nineteen. A box mostly full of nothing reads
	// as a box somebody forgot to fill.
	//
	// The ellipsis costs nothing here for the reason it costs nothing on the
	// take screen: the button beside it carries the whole line to the
	// clipboard, and what stays visible is the verb and the first flags, which
	// is what tells somebody whether this is the line they wanted.
	v.command = widget.NewLabelWithStyle("", fyne.TextAlignLeading,
		fyne.TextStyle{Monospace: true})
	v.command.Wrapping = fyne.TextWrapOff
	v.command.Truncation = fyne.TextTruncateEllipsis
	v.copyCommand = parts.NewButton(v.text.T(i18n.CommandCopy), v.putCommandInClipboard)

	// 🔴 The values are no longer ONE label, changed on 2026-09-09.
	//
	// They were a single label holding every value joined by newlines, at the
	// heading size, and the comment here explained at length why that label was
	// the largest thing on the screen. The trouble is what a label cannot be: a
	// row. Nothing could band every second value, nothing could give a value the
	// room to grow with the window, and the number beside a value had nowhere to
	// go, so a person comparing the fourth value against their own data had to
	// count lines with a finger.
	//
	// They are drawn as rows now, out of the primitives, and the list itself is
	// the state. What is left here is the sentence the screen says about them,
	// which is a paragraph and therefore still a label.
	v.told = widget.NewLabel("")
	v.told.Importance = widget.LowImportance
	v.told.Wrapping = fyne.TextWrapWord
	v.showing = map[string]bool{}

	// 🔴 In reading order, because the toolkit walks the objects in the order
	// they were given and not in the order they are drawn. The screen next door
	// paid for that once: the button at the top of it was the last stop of the
	// keyboard.
	v.lay()

	// 🔴 Last, and the order is the whole reason this line is here rather than
	// beside the control it sets. Choosing runs the callback, the callback
	// draws, and drawing writes into two labels and rebuilds a form. Setting it
	// where the chooser is built reached all of them before any existed, and
	// the window died on a nil pointer with every test green up to that point,
	// because nothing else on this screen calls its own callback while being
	// built.
	v.gender.SetSelected(v.text.T(i18n.QuickGenderAny))
	// Before the locale, because choosing a locale draws, and a draw before
	// the count has a choice would ask the mode for nought values and be
	// refused for something nobody touched.
	v.howMany.SetSelected(opensOn)
	v.locales.SetSelected(quick.DefaultLocale)
	// Last, because choosing a kind is what draws, and everything the drawing
	// reads has to be in place before it runs. This is the call that makes the
	// window open on a value rather than on an empty form.
	v.chooser.SetSelected(opensWith)
}

// lay puts the controls on the screen, apart from making them.
//
// Two functions rather than one, because building this screen passed the limit
// on the length of a function, and the limit is bent for nothing. The seam is
// the one that costs least: what the controls ARE does not change when the
// arrangement of them does.
//
// The arrangement itself lives next door in shape.go, which holds the drawing
// and nothing else. This function is where the three parts are made and handed
// to it, and where the two that get refilled are created empty.
func (v *View) lay() {
	// Both are containers this screen refills rather than rebuilds, because a
	// screen built once cannot reach into itself any other way, and because
	// the alternative is hiding things that are drawn but empty, which is the
	// fault this screen was corrected for twice already.
	v.settings = container.NewStack()
	v.result = container.NewStack()
	v.about = container.NewStack()
	v.terminal = v.commandBand()

	// 🔴 The panel is wrapped so that it takes the height of what is IN it
	// rather than the height of the window. Given the edge of a border it would
	// be stretched to the full height, and a card with three paragraphs in it
	// and six hundred pixels of empty interior reads as a thing that failed to
	// load. The same rule the card of values follows, for the same reason: the
	// surplus stays on the bare window, where it reads as room.
	hugging := container.NewVBox(v.about)

	// The values take what is left and the panel keeps its width, which is the
	// way round that follows the content: a value is short and a paragraph has
	// a readable line length, so the one that must not stretch is the panel.
	//
	// The values come first in the container for the reason the whole screen
	// does: that is the order the keyboard walks, and they are what a person
	// came for.
	body := paint.Pad(palette.NameStepSection, palette.NameStepSection,
		palette.NameStepMajor, palette.NameStepMajor,
		container.New(layout.NewBorderLayout(nil, nil, nil, hugging),
			v.result, hugging))

	v.content = v.shape(v.settingsBand(v.settings), body, v.terminal)
}

// settle puts into the screen whatever this screen currently has to say.
//
// # Why the parts are REBUILT rather than shown and hidden
//
// Hiding was the old answer and it produced two defects in two days, both found
// on the pixels rather than by a test. A bordered box with nothing in it reads
// as a thing that failed to load, and the sentence explaining a refusal lived
// INSIDE the block that was hidden whenever there was no value, which is
// exactly when a refusal has something to say. Building the block that belongs
// to the current state cannot leave an empty frame behind and cannot hide a
// sentence inside one, because the state decides what exists rather than what
// is visible.
//
// # The three states, and why a refusal is not the absence of a value
//
// Values, a refusal, or a value with something said under it. The middle one is
// the one that was wrong before: asking only whether there is a value cannot
// tell a screen that produced nothing yet from one that answered no.
func (v *View) settle() {
	v.result.Objects = []fyne.CanvasObject{v.answer()}
	v.result.Refresh()

	// The line is written even for a request that was turned down, because
	// somebody whose request was refused may still want to show that request to
	// somebody else. So the band follows the LINE and not the values.
	if v.command.Text == "" {
		v.terminal.Hide()
		return
	}
	v.terminal.Show()
}

// reloadGenerators puts the list of values back in step with the locale.
//
// Every locale of this build carries every generator today, so this changes
// nothing that can be seen. It is here because the mode answers the question
// per locale and a screen that asked once at build time would be right by
// coincidence, which is the shape of defect this project keeps finding.
func (v *View) reloadGenerators() {
	v.generators = quick.Generators(v.locale)
	v.chooser.Options = v.generators
	v.chooser.Refresh()
}

// UseLocale picks a locale, the way a person clicking the list would.
func (v *View) UseLocale(locale string) { v.locales.SetSelected(locale) }

// Choose picks a generator, the way a person clicking the list would.
//
// Exported for the reason the other screens export theirs: the guard holding
// what every screen draws and the probe rendering each state have to reach a
// state where a value is on screen, and setting the fields directly would reach
// one the controls cannot produce.
func (v *View) Choose(generator string) { v.chooser.SetSelected(generator) }

// ChooseGender picks a gender, for the same reason.
func (v *View) ChooseGender(gender string) { v.gender.SetSelected(gender) }

// ChooseCount picks how many values to produce, for the same reason.
func (v *View) ChooseCount(howMany string) { v.howMany.SetSelected(howMany) }

// CountChoices is what this screen offers, so that a guard walks the list the
// screen has rather than a copy of it written in a test.
func CountChoices() []string { return append([]string(nil), countChoices...) }

// SetBorn types a date of birth, for the same reason.
func (v *View) SetBorn(date string) { v.born.SetText(date) }

// UseSeed fixes the seed, so that a picture of this screen is the same picture
// tomorrow. Without it every render would draw a different value and the stored
// screens would never match.
func (v *View) UseSeed(seed string) {
	v.seed = seed
	v.draw()
}

// Value is what is on screen, for a test that asks rather than reads pixels.
//
// Joined by newlines, which is the shape the command line writes and the shape
// every caller of this already expects. The screen draws them as rows, and this
// is the one place the two forms meet.
func (v *View) Value() string { return strings.Join(v.values, "\n") }

// drawAgain asks for another value of the same kind.
//
// It draws a new seed, which is what running the command a second time does.
// Everything else about the request is left as the person set it.
func (v *View) drawAgain() {
	v.seed = ""
	v.seedBox.SetText("")
	v.draw()
}

// draw produces the value for whatever is chosen, and says why when it cannot.
// draw works out what this screen says, then settles what is shown.
//
// Two steps rather than one, because the first has six ways out and the second
// has to happen on every one of them. Written as a single function it would be
// six calls to remember, and the day somebody added a seventh way out the
// screen would show an empty box again.
func (v *View) draw() {
	v.produce()
	v.settle()
}

// produce works out the value and the line that would ask a terminal for it.
func (v *View) produce() {
	v.told.SetText("")
	chosen := v.chooser.Selected
	if chosen == "" {
		v.values = nil
		v.command.SetText("")
		v.showRows("")
		return
	}
	v.showRows(chosen)

	if v.seed == "" {
		drawn, err := quick.NewSeed()
		if err != nil {
			v.values = nil
			v.told.SetText(err.Error())
			return
		}
		v.seed = drawn
	}
	// The field shows whichever seed produced what is on screen, whether the
	// person typed it or the screen drew it. SetText is guarded by the callback
	// above rather than here, so this cannot start drawing again.
	if v.seedBox.Text != v.seed {
		v.seedBox.SetText(v.seed)
	}

	// Set before the request is judged, and on purpose. A refusal is still
	// something a person may want to reproduce in a terminal, and a line that
	// appears only when everything worked would be missing exactly when
	// somebody wants to show it to somebody else.
	v.command.SetText(v.commandFor(chosen))

	asked := v.request(chosen)
	if refusal := quick.Check(asked); refusal.Refused() {
		v.values = nil
		v.told.SetText(v.text.T(refusedSays[refusal.Kind]))
		return
	}
	produced, err := quick.Run(asked)
	if err != nil {
		v.values = nil
		v.told.SetText(err.Error())
		return
	}
	// One to a row, which is the same order the command line writes them in, so
	// a person comparing the two surfaces sees one answer rather than two.
	v.values = produced
	// Said only where it can happen. One value cannot repeat, and a warning
	// standing beside a screen it does not apply to is how a person learns to
	// stop reading warnings.
	if len(produced) > 1 {
		v.told.SetText(v.text.T(i18n.QuickMayRepeat))
	}
}

// request is what the person on this screen asked for.
func (v *View) request(generator string) quick.Request {
	return quick.Request{
		Generator: generator, Locale: v.locale, Seed: v.seed, Agrees: v.bindings(),
		Count: v.wanted(),
	}
}

// wanted is how many values were chosen, as a number.
//
// The list holds text because a chooser does, and the one place it becomes a
// number is here. A choice that is not a number cannot arrive from a list this
// screen built, so it falls back to the head of that list rather than to a
// number written here: a nought would be refused by the mode and a person
// would see a refusal about something they never touched.
func (v *View) wanted() int {
	number, err := strconv.Atoi(v.howMany.Selected)
	if err != nil {
		number, _ = strconv.Atoi(opensOn)
	}
	return number
}

// bindings are the values the two fields carry, with an unchosen gender
// carrying nothing.
func (v *View) bindings() map[string]string {
	bound := map[string]string{}
	if v.gender.Selected != "" && v.gender.Selected != v.text.T(i18n.QuickGenderAny) {
		bound[quick.RoleGender] = v.gender.Selected
	}
	if v.born.Text != "" {
		bound[quick.RoleBirthDate] = v.born.Text
	}
	return bound
}

// showRows puts a field on screen only where the chosen value takes account of
// it.
//
// 🔴 Which values do is asked of the mode rather than written here, and the
// difference is measured: the design of this screen said the answer was one
// generator, and a measurement on 2026-09-05 found three, one of them only in
// Polish. A list written here would have hidden the gender beside a first name,
// which is the most obvious use this screen has.
//
// # Why the seed is given the room that is left over
//
// A seed is thirty two characters and the four fields before it are a word
// each, so a row that shared its width equally would cut the one value nobody
// can retype from memory. The layout gives the fixed fields what they ask for
// and the seed everything else, which is what the accepted drawing does and
// what stops this row needing a width written into it.
func (v *View) showRows(generator string) {
	fixed := []fyne.CanvasObject{
		labelled(v.text.T(i18n.QuickLabelValue), v.chooser.OnScreen()),
		// Second, and above the settings that only some generators read,
		// because how many is asked of every generator alike. The fields below
		// it come and go with what was chosen, so a field that never leaves
		// belongs before the ones that do.
		labelled(v.text.T(i18n.QuickLabelCount), v.howMany.OnScreen()),
		labelled(v.text.T(i18n.QuickLabelLocale), v.locales.OnScreen()),
	}
	showing := map[string]bool{}
	if quick.Reads(generator, quick.RoleGender, v.locale) {
		showing[quick.RoleGender] = true
		fixed = append(fixed, labelled(v.text.T(i18n.QuickLabelGender), v.gender.OnScreen()))
	}
	if quick.Reads(generator, quick.RoleBirthDate, v.locale) {
		showing[quick.RoleBirthDate] = true
		fixed = append(fixed, labelled(v.text.T(i18n.QuickLabelBorn), dated(v.born)))
	}

	// 🔴 The panel is settled BEFORE the check below, and it was settled after
	// it for as long as it took to render the screen once.
	//
	// That check asks whether the same optional FIELDS are already on screen,
	// and the panel does not depend on the fields at all: it depends on the
	// KIND. Choosing a kind that reads the same settings as the last one, which
	// is most of them, left the check true and the panel describing whatever was
	// chosen before. Opening the window hit exactly that path, so the panel was
	// simply not there. One condition cannot guard two things that change at
	// different moments, and the render is what said so.
	v.describe(generator)

	if sameFields(v.showing, showing) && len(v.settings.Objects) > 0 {
		return
	}
	v.showing = showing
	// 🔴 The seed takes the room the fixed fields did not, and the objects are
	// handed to the layout in the order the keyboard should walk them.
	//
	// This was container.NewBorder for as long as it took to walk the screen
	// once, and that helper appends the CENTRE first: the keyboard reached the
	// seed at stop six and the list of kinds at stop seven, so the first thing
	// a person tabbed into was the last field on the row. The comment beside
	// shape() in this package warns about exactly that, and it was written one
	// function before this one was got wrong, which is the whole argument for
	// deriving these numbers from tools/klawiatura rather than from reading.
	settings := parts.Row(palette.NameStepRelated, fixed...)
	seed := labelled(v.text.T(i18n.QuickLabelSeed), v.seedBox)
	v.settings.Objects = []fyne.CanvasObject{
		container.New(layout.NewBorderLayout(nil, nil, settings, nil), settings, seed),
	}
	v.settings.Refresh()
}

// labelled names a control, above it rather than beside it.
//
// It is one call to the drawing next door, named here so that the row above
// reads as a list of settings rather than as a list of layout instructions.
func labelled(name string, control fyne.CanvasObject) fyne.CanvasObject {
	return paint.FieldLabel(name, control)
}

// sameFields reports whether the same optional settings are already on screen,
// so that a redraw on every keystroke does not rebuild a row that has not
// changed.
//
// 🔴 It compares the ROLES rather than the labels, which is the correction that
// came with the row. The old one compared the words in a form, so two settings
// renamed to the same word in some future translation would have read as one
// unchanged row. A role is the thing the mode answers about and cannot collide.
func sameFields(held, wanted map[string]bool) bool {
	if len(held) != len(wanted) {
		return false
	}
	for role := range wanted {
		if !held[role] {
			return false
		}
	}
	return true
}

// Shows reports whether the field for that role is on screen, for a test that
// asks the screen rather than counting pixels.
func (v *View) Shows(role string) bool { return v.showing[role] }

// putInClipboard copies the value and says that it did.
//
// Saying so is not decoration. A clipboard cannot be seen, so a button that
// silently succeeded and a button that silently did nothing look the same.
func (v *View) putInClipboard() {
	if len(v.values) == 0 || v.clipboard == nil {
		return
	}
	v.clipboard.SetContent(v.Value())
	v.told.SetText(v.text.T(i18n.QuickCopied))
}

// commandFor is the line that asks a terminal for what this screen is showing.
//
// The seed is always written out, and that is the point of the line rather
// than a detail of it. Without one the command line draws a fresh seed and
// says so, so a line copied without it would produce a different value from
// the one on screen, which is the one thing this bridge must not do.
func (v *View) commandFor(generator string) string {
	bound := v.bindings()
	// The count is written only where it differs from what the verb does
	// without it. A line carrying --count 1 would say something the verb
	// already does, and every flag on this line is one more thing to read.
	asked := ""
	if v.howMany.Selected != commandDefault {
		asked = v.howMany.Selected
	}
	return commandline.New(commandline.Quick).
		Flag(commandline.Count, asked).
		Flag(commandline.Locale, v.locale).
		Flag(commandline.Gender, bound[quick.RoleGender]).
		Flag(commandline.BirthDate, bound[quick.RoleBirthDate]).
		Flag(commandline.Seed, v.seed).
		Argument(generator).
		String()
}

// putCommandInClipboard copies the line and says that it did, for the reason
// the value's own button says it.
func (v *View) putCommandInClipboard() {
	if v.command.Text == "" || v.clipboard == nil {
		return
	}
	v.clipboard.SetContent(v.command.Text)
	v.told.SetText(v.text.T(i18n.CommandCopied))
}

// countChoices are how many values this screen offers to produce.
//
// A ladder rather than a range, because it is read off a list, and the top of
// it is where a person stops reading values and starts piping them, which is
// the point at which the command beside it is the better tool. The mode itself
// has no upper limit, and the recipe reader says why: a number chosen today
// with no measurement behind it would become a contract.
var countChoices = []string{"1", "10", "25", "50"}

// 🔴 opensOn and commandDefault are both "how many by default" and they are
// NOT the same number. Holding them under one name is the mistake this pair is
// written out to avoid.
//
// opensOn is what this SCREEN starts with. It is ten rather than one because
// of what a person sees when the window opens, which is the state no drawing
// ever shows: with one value the screen stood 57.5% empty at the size the
// window is actually kept, measured with tools/pustka.py, and ten brings that
// to 36.8%. A tester filling the same form over and over wants a handful, and
// the one who wants exactly one takes the top of the list.
//
// commandDefault is what the COMMAND LINE does when the flag is left out, and
// that stays one: it is the older behaviour, the verb is the one whose answer
// is meant to be piped, and changing it would alter what an existing script
// produces. The command shown on this screen therefore carries --count
// whenever the screen differs from it, which with the setting above is from
// the moment the window opens.
const (
	opensOn        = "10"
	commandDefault = "1"
)

// opensWith is the kind of value this SCREEN starts on, and it belongs to the
// pair above rather than beside it: it is a third answer to "what does the
// window do before anybody has said anything", and the command line has no
// counterpart at all, since naming a generator there is required.
//
// 🔴 Until 2026-09-08 this screen opened on nothing chosen, and it is the
// screen the window itself opens on, so that was the first thing anybody saw of
// this product: an empty form and, at the size the window is kept, 76.9 per
// cent black. Measured with tools/pustka.py. Opening on a value fills that with
// the product rather than with anything written to fill it.
//
// # Why this one
//
// It agrees with the locale this screen also opens on, which is en-US, so the
// first thing on screen is not an identifier of a country the locale beside it
// does not name. It has a reserved range, so the values shown before anybody
// has chosen anything cannot belong to a real person, which is rule nine and
// matters most exactly here, where the values appear without being asked for.
// And it is recognisable, which the first screen of a tool has to be.
//
// Nothing is read from a disk and nothing is written, so DEC-35 does not reach
// this: what it forbids is a surface that ACTS on opening, and producing a
// value in memory is what this screen is.
const opensWith = "ssn"

// genderChoices are the spellings the mode takes, with the absence of a choice
// named rather than left blank.
func (v *View) genderChoices() []string {
	return append([]string{v.text.T(i18n.QuickGenderAny)}, quick.Genders()...)
}

// refusedSays is every refusal the mode can produce, in the words of this
// surface.
//
// A register rather than a switch, for the reason the command line writes
// beside its own: a switch has a last branch, and a kind added to the mode and
// forgotten here would land in it and show a confident sentence about a
// different refusal.
var refusedSays = map[quick.RefusalKind]i18n.Key{
	quick.UnknownGenerator: i18n.QuickRefusedGenerator,
	quick.RoleNotRead:      i18n.QuickRefusedRole,
	quick.UnreadableGender: i18n.QuickRefusedGender,
	quick.UnreadableDate:   i18n.QuickRefusedDate,
	quick.CountNotPositive: i18n.QuickRefusedCount,
}

func subdued(text string) fyne.CanvasObject {
	label := widget.NewLabel(text)
	label.Importance = widget.LowImportance
	label.Wrapping = fyne.TextWrapWord
	return label
}
