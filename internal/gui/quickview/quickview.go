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
// # Why choosing produces and typing does not
//
// The promise is under five seconds, so a screen that asks somebody to choose a
// generator and then press a button has one step too many: choosing is what
// produces. A field somebody TYPES into is the other half, and it waits for
// Generate (DEC-235) - because a field that produced on every keystroke made a
// thirty two character seed regenerate the set thirty two times, and would make
// a typed count of five hundred do it at 5, at 50 and at 500.
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
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
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
	// confirmed is the control saying it filled the clipboard, nil for none.
	confirmed confirmer

	chooser *parts.Select
	locales *parts.Select
	gender  *parts.Select
	// howMany is how many values to produce, typed rather than chosen.
	//
	// 🔴 It was a list of four until 2026-09-09, on the argument that a box
	// somebody can type any number into invites one that fills the window with
	// a wall nobody reads. The owner overruled it in one sentence: a list
	// LIMITS the number, and the limit was four arbitrary points rather than a
	// measurement. What replaces the argument is a measured ceiling and a
	// refusal that names the command line, which has none - so the wall is
	// still refused and the number is no longer somebody's guess.
	howMany *parts.FieldEntry
	born    *parts.FieldEntry
	seedBox *parts.FieldEntry
	copy    *parts.Button
	again   *parts.Button
	// generate is the one action this screen is for, and the only filled
	// button on it (DEC-235).
	//
	// It took that role from copy, which held it from 2026-09-08 only because
	// it was then the sole thing here anybody presses on purpose. Copying is
	// what a person does AFTER they have what they came for.
	generate *parts.Button

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
	// showing is which optional settings have a field on screen: the roles the
	// chosen kind reads, and the locale where it changes the kind (under
	// settingLocale). A test asks the screen this rather than counting pixels,
	// and the request and the line at the foot carry only what it holds.
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
	// list is the card of values, built once and handed each new set, and
	// listShown is it with the ring the keyboard lights (valuesCard says why).
	list      *parts.ScrolledRows
	listShown fyne.CanvasObject

	// result is the card and its heading, or the sentence saying why there is
	// no card. Refilled by settle, for the reason settings is.
	result *fyne.Container
	// about is the panel describing the chosen kind, refilled whenever the kind
	// or the locale changes, which are the only two things it depends on, and
	// described is which kind and locale it holds now.
	about     *fyne.Container
	described describedFor
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

// buildChoosers makes the three lists of the screen: the kind of value, the
// locale it is drawn from and the gender.
//
// It and the two after it are apart from build because that function passed
// the limit on the length of a function. Each makes one family of controls and
// none of them lays anything out, so where things stand is still decided in one
// place, which is lay.
func (v *View) buildChoosers() {
	v.chooser = parts.NewSelect(v.generators, func(string) { v.draw() })
	v.chooser.SetChoices(v.choices(), v.panelWords())
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
}

// buildFields makes the three fields somebody types into: how many values, the
// date of birth and the seed. None of them produces while being typed into.
func (v *View) buildFields() {
	// Typed, so it waits for the button like the seed does (DEC-235). Without
	// that, typing 500 would produce at 5, at 50 and at 500.
	v.howMany = parts.NewFieldEntry()
	v.howMany.OnSubmitted = func(string) { v.draw() }
	// 🔴 Judged on every keystroke and NOT produced on any (DEC-235): the field
	// turns red the moment it holds a number Generate would refuse, and the
	// sentence saying why still waits for the press (B-363). Set before Bounded,
	// which keeps whatever was set here and calls it after its own filter.
	v.howMany.OnChanged = func(typed string) { v.howMany.SetInvalid(v.countIsWrong(typed)) }
	// 🔴 The field will not take a character that could never be a count, and
	// it will not take more digits than the ceiling has. Added on 2026-09-09
	// after the owner typed a thirty seven digit number into the real window:
	// the screen answered that it was not a whole number, which is not what was
	// wrong with it, and the command at the foot offered the same number to
	// copy into a terminal.
	//
	// The length comes from the ceiling rather than from a number written here,
	// so a decision that moves the ceiling moves this with it.
	parts.Bounded(&v.howMany.Entry, countDigits, parts.Digits)
	v.born = parts.NewFieldEntry()
	// The characters a date is made of, for the same reason. The SHAPE is
	// still the mode's to refuse, since 9999-99-99 is made of these characters
	// and is not a date.
	parts.Bounded(&v.born.Entry, len(quick.DateShape), parts.DigitsAndDashes)
	// The shape of the date is on the field rather than only in the refusal
	// that follows getting it wrong, which is the same reason the seed field
	// shows the seed that was used.
	v.born.SetPlaceHolder(quick.DateShape)
	// A date is typed, so it waits like the seed does (DEC-235). Half a date is
	// a refusal, and a refusal shown while somebody is still typing the year is
	// the screen arguing with a person mid sentence.
	v.born.OnSubmitted = func(string) { v.draw() }
	// The seed is a field rather than a line of text, and that is what makes
	// this screen able to do what the flag does. The screen fills it with the
	// one it used when nobody said, so asking for the same value again is
	// reading it off the screen rather than out of a log.
	v.seedBox = parts.NewFieldEntry()
	// A seed the surfaces will take, with the limit and the measurement behind
	// it in the domain, so this field and the --seed flag of three verbs cannot
	// come to accept different things. No filter on the characters: a seed is
	// any text at all, and narrowing that here would refuse a seed the command
	// line takes.
	parts.Bounded(&v.seedBox.Entry, seed.Longest, nil)
	// 🔴 NOTHING happens while somebody types, and that is DEC-235.
	//
	// Until 2026-09-09 this produced on every keystroke, so a thirty two
	// character seed regenerated the set thirty two times and a person watched
	// values from seeds they never asked for go by. With a list of four counts
	// that was merely odd. The moment the count becomes a field somebody types
	// a number into, typing 10000 would regenerate at 1, 10, 100, 1000 and
	// 10000, on the thread that draws - so the button is what makes that
	// change safe rather than a decoration beside it.
	//
	// Enter does the same as the button, because a person who typed a seed
	// has their hands on the keyboard.
	v.seedBox.OnSubmitted = func(string) { v.draw() }
}

// buildActions makes the buttons of the screen, the line a terminal would take
// and the two marks saying that the clipboard was filled.
func (v *View) buildActions() {
	// "Copy all" rather than "Copy" since 2026-10-21, at the owner's word: a
	// row copies ONE value when clicked, so the button over the rows has to
	// say that it takes every one of them.
	v.copy = parts.NewButton(v.text.T(i18n.QuickCopyAll), v.putInClipboard)
	v.copy.ShowIcon(paint.IconCopy)
	v.generate = parts.NewPrimaryButton(v.text.T(i18n.QuickGenerate), v.draw)
	v.again = parts.NewButton(v.text.T(i18n.QuickAgain), v.drawAgain)
	v.again.ShowIcon(paint.IconAgain)

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
	v.copyCommand.ShowIcon(paint.IconCopy)

	// 🔴 The button that filled the clipboard says so itself, for a moment
	// (parts.Button.Confirm). A clipboard cannot be seen, so a person who finds
	// no confirmation cannot tell a button that worked from one that did
	// nothing - which is the owner's report of 2026-09-09 that put a sentence
	// beside each button. That sentence then stayed until the next copy, still
	// claiming the clipboard held what it held an hour ago, and the owner asked
	// on 2026-10-21 for the button to say it instead. Before 2026-09-09 the
	// message went into the sentence above the card, seven hundred pixels from
	// the button, and replaced the warning that values may repeat.
	v.copy.Confirmed = v.text.T(i18n.QuickCopied)
	v.copyCommand.Confirmed = v.text.T(i18n.CommandCopied)
}

// build makes the controls, lays them out and opens the screen on a value. The
// order of those steps matters, and the comments inside say why.
func (v *View) build() {
	v.buildChoosers()
	v.buildFields()
	v.buildActions()

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
	v.list = NewValueList()
	v.listShown = v.list.Framed()

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
	v.howMany.SetText(opensOn)
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
	v.chooser.SetChoices(v.choices(), v.panelWords())
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

// Generate presses the button, the way a person with a typed seed would.
//
// It goes through the control rather than calling draw, so a button wired to
// nothing fails the guard that uses this instead of passing it.
func (v *View) Generate() { v.generate.Tapped(nil) }

// TypeSeed puts characters in the seed field WITHOUT applying them, which is
// what a person typing does. Generate is the second half.
//
// It exists so that a guard can tell the two halves apart. Everything else in
// this file that sets a field also produces, and a test using those could not
// see the difference DEC-235 makes.
func (v *View) TypeSeed(seed string) { v.seedBox.SetText(seed) }

// ChooseGender picks a gender, for the same reason.
func (v *View) ChooseGender(gender string) { v.gender.SetSelected(gender) }

// ChooseCount picks how many values to produce, for the same reason.
func (v *View) ChooseCount(howMany string) {
	v.howMany.SetText(howMany)
	v.draw()
}

// Told is what this screen currently says beside the values, or instead of
// them. A guard reads it rather than looking for a sentence in the tree.
func (v *View) Told() string { return v.told.Text }

// MostShown is the largest number of values this screen draws at once, so that
// a guard asks the screen rather than repeating the number.
//
// It replaces CountChoices, which handed out the four options the list used to
// offer. There is no list any more: the field takes a number and the ceiling is
// the only thing left that a guard could disagree with.
func MostShown() int { return mostShown }

// SetBorn types a date of birth and applies it, which is two acts since
// DEC-235: a typed field no longer produces on its own, so this is the field
// plus the button rather than the field alone.
func (v *View) SetBorn(date string) {
	v.born.SetText(date)
	v.draw()
}

// UseSeed fixes the seed, so that a picture of this screen is the same picture
// tomorrow. Without it every render would draw a different value and the stored
// screens would never match.
// 🔴 It puts the seed in the FIELD and then produces, rather than setting the
// value behind it. Since DEC-235 the field is the request and produce reads it
// there, so setting only the field behind would be overwritten by the empty box
// on the next line - every render would have drawn a random seed and the stored
// screens would have stopped matching, which is exactly what this method exists
// to prevent.
func (v *View) UseSeed(seed string) {
	v.seedBox.SetText(seed)
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
	v.unconfirm()
	chosen := v.chooser.Selected
	if chosen == "" {
		v.values = nil
		v.command.SetText("")
		v.showRows("")
		return
	}
	v.showRows(chosen)

	// What the field holds is the request, and it is read HERE rather than as
	// somebody types (DEC-235). Trimmed, because a seed pasted out of a log
	// carries the spaces around it and a person cannot see them.
	v.seed = strings.TrimSpace(v.seedBox.Text)

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

	// 🔴 Asked BEFORE the request is built, because a request carries an int
	// and the field carries text. The two questions - is this a number, and is
	// this a number of values - keep their two answers: the first is the mode's
	// CountFrom, the second is the mode's Check in ask below, and only the
	// ceiling in between belongs to this screen.
	wanted, saidAboutCount := v.countAsked(v.howMany.Text)
	if saidAboutCount != "" {
		v.values = nil
		v.told.SetText(saidAboutCount)
		return
	}
	v.ask(chosen, wanted)
}

// ask hands the mode the request and puts what came back on the screen: the
// values, or the sentence saying why there are none.
//
// Apart from produce because that function passed the limit on the length of a
// function. The seam is where everything this screen decides on its own is
// done and only the mode's answer is left.
func (v *View) ask(chosen string, wanted int) {
	asked := v.request(chosen, wanted)
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
func (v *View) request(generator string, count int) quick.Request {
	return quick.Request{
		Generator: generator, Locale: v.localeAsked(), Seed: v.seed, Agrees: v.bindings(),
		Count: count,
	}
}

// localeAsked is the locale the request carries, which is none where the screen
// shows no choice of locale.
//
// For the reason bindings sends only what is on screen: a locale chosen beside
// a first name and kept while a PESEL is chosen would be a setting nobody can
// see. Leaving it out changes no value, which the guard beside the register
// measures, and keeps the line at the foot free of a flag that does nothing.
func (v *View) localeAsked() string {
	if !v.showing[settingLocale] {
		return ""
	}
	return v.locale
}

// countAsked is how many values were asked for, or the sentence saying why the
// screen will not ask for them. An empty sentence means the number is good.
//
// # What is decided here and what is not
//
// Only the CEILING, which is a limit of this screen and of nothing else. Whether
// the text is a number at all is quick.CountFrom, so the field and the flag read
// 010 the same way. Whether the number is a number of VALUES - at least one - is
// quick.Check further down, so a nought is refused by the same rule on both
// surfaces. Writing either of those here would be a second answer to a question
// that already has one.
func (v *View) countAsked(typed string) (int, string) {
	// Trimmed, because a field is pasted into and a person cannot see the
	// spaces they brought with them. The flag is NOT trimmed, and the reason is
	// written beside CountFrom.
	wanted, refused := quick.CountFrom(strings.TrimSpace(typed))
	if refused.Refused() {
		return 0, v.text.T(refusedSays[refused.Kind])
	}
	if wanted > mostShown {
		// The number is put into the sentence here rather than into the
		// catalogue, the same way the heading of the card carries its count:
		// a translated sentence must not have to be edited when a measurement
		// moves the ceiling.
		return 0, strconv.Itoa(mostShown) + " " + v.text.T(i18n.QuickRefusedCountMany)
	}
	return wanted, ""
}

// countIsWrong reports whether a press of Generate would refuse the count
// field as it now stands, asked while somebody types so that the field can say
// so at once (B-363).
//
// It asks what the press asks, in the same order: countAsked for the shape and
// the ceiling, then the mode's own rule that a count is at least one. The guard
// beside this file types numbers, presses, and holds the two answers equal.
//
// An empty field is not wrong here, although the press refuses it. Somebody
// who has just deleted the old number is half way to typing the new one, and a
// field turning red under a deleted digit argues with them mid sentence.
func (v *View) countIsWrong(typed string) bool {
	if strings.TrimSpace(typed) == "" {
		return false
	}
	wanted, said := v.countAsked(typed)
	return said != "" || quick.CheckCount(wanted).Refused()
}

// bindings are the values the two fields carry, with an unchosen gender
// carrying nothing.
//
// 🔴 Only a field that is ON SCREEN is part of the request, which is what
// v.showing records. A field showRows took away keeps what was typed into it,
// so coming back to a kind that reads it applies it again, but it is not sent
// while hidden. It was sent until 2026-11-12, and the mode refuses a role the
// chosen kind does not read: a gender chosen for a first name turned the next
// ISBN into a refusal naming a field nobody could see or clear, and the command
// at the foot carried the same flag (B-365).
func (v *View) bindings() map[string]string {
	bound := map[string]string{}
	if v.showing[quick.RoleGender] && v.gender.Selected != "" &&
		v.gender.Selected != v.text.T(i18n.QuickGenderAny) {
		bound[quick.RoleGender] = v.gender.Selected
	}
	if v.showing[quick.RoleBirthDate] && v.born.Text != "" {
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
	showing := map[string]bool{}
	// 🔴 The locale only where another locale gives something else, asked of
	// the mode for the reason the roles below are. Until 2026-11-12 it stood
	// beside every kind, and for thirteen of eighteen it changed nothing a
	// person could see (B-367).
	if quick.DependsOnLocale(generator) {
		showing[settingLocale] = true
	}
	if quick.Reads(generator, quick.RoleGender, v.locale) {
		showing[quick.RoleGender] = true
	}
	if quick.Reads(generator, quick.RoleBirthDate, v.locale) {
		showing[quick.RoleBirthDate] = true
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

	// 🔴 NOTHING is built before this line, and until 2026-09-14 the row of
	// fixed fields was built above it, on every draw, whether or not the return
	// below then threw it away.
	//
	// Building it dressed the count field again each time, and dressing a field
	// hands its focus to the grounds of THAT dressing. Opening the window draws
	// three times and rebuilds the row once, so the grounds on the screen were
	// the first dressing's and the focus went to the third's: the keyboard
	// arrived on the count field and not one pixel changed, measured by
	// tools/ognisko.py on the real window (B-279). The seed field, dressed below
	// this line and therefore only when the row is really rebuilt, showed its
	// ring the whole time, which is what narrowed it to the order of these
	// lines. A field is dressed once now, so the fault cannot come back by this
	// door, and this line is here so that it cannot come back by a similar one.
	if sameFields(v.showing, showing) && len(v.settings.Objects) > 0 {
		return
	}
	v.showing = showing
	v.placeSettings(v.settingsRow(showing))
}

// settingsRow is the row of settings for the fields the chosen kind reads, the
// seed last and given whatever room the others leave.
//
// It and placeSettings are apart from showRows because that function passed
// the limit on the length of a function. The seam is the early return in
// showRows: everything after it builds, and runs only when the fields on screen
// have to change.
func (v *View) settingsRow(showing map[string]bool) fyne.CanvasObject {
	fixed := []fyne.CanvasObject{
		labelled(v.text.T(i18n.QuickLabelValue), v.chooser.OnScreen()),
		// Second, and above the settings that only some generators read,
		// because how many is asked of every generator alike. The fields below
		// it come and go with what was chosen, so a field that never leaves
		// belongs before the ones that do.
		labelled(v.text.T(i18n.QuickLabelCount),
			paint.AtLeast(paint.FieldWidth(countSample), 0, parts.Standing(parts.Field(v.howMany)))),
	}
	// Before the gender, because the locale decides whether a surname reads one.
	if showing[settingLocale] {
		fixed = append(fixed, labelled(v.text.T(i18n.QuickLabelLocale), v.locales.OnScreen()))
	}
	if showing[quick.RoleGender] {
		fixed = append(fixed, labelled(v.text.T(i18n.QuickLabelGender), v.gender.OnScreen()))
	}
	if showing[quick.RoleBirthDate] {
		fixed = append(fixed, labelled(v.text.T(i18n.QuickLabelBorn), dated(parts.Field(v.born))))
	}
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
	// 🔴 The seed stands IN the row with the others and is as wide as a seed,
	// not as wide as whatever is left over.
	//
	// It was the CENTRE of a border layout until 2026-09-09, and a centre is
	// handed every pixel the edges did not want - so the field ran to the edge
	// of the window and the owner called it giga long. He was describing
	// arithmetic: its width was the window minus four controls, whatever that
	// happened to be. Measured from the face rather than written down, so it
	// follows the type ladder.
	// 🔴 The seed is as wide as a seed, the fourth arrangement this row has had,
	// and the history is the reason for the sample it is measured from.
	//
	// It was the centre of a border with nothing on its right, so it ran to the
	// edge of the window: giga long, said the owner. Given the width of a thirty
	// two character seed, the row asked for 98 px more than the button left it
	// in the states showing a gender and a date of birth, and Fyne does not clip,
	// so the field drew under Generate (found by tools/wylewa.py --poziom). It
	// was then the space between the fields and the button, which cannot
	// overflow and was again as long as the window allowed - and on 2026-10-21
	// the owner said the fields are of different lengths and each should fit
	// the value it takes (DEC-261). paint.SeedSample is fourteen characters,
	// eighteen fewer than the sample that overflowed, and the button now
	// follows the seed rather than standing at the edge of the window, so the
	// room after it is empty rather than lent to the field.
	seed := labelled(v.text.T(i18n.QuickLabelSeed),
		paint.AtLeast(paint.FieldWidth(paint.SeedSample), 0, parts.Standing(parts.Field(v.seedBox))))
	return paint.Row(palette.NameStepRelated, append(fixed, seed)...)
}

// placeSettings puts the row of settings in the band with the one action of
// the screen beside it.
func (v *View) placeSettings(withSeed fyne.CanvasObject) {
	// The button stands beside the fields rather than under a label of its own,
	// so it has to drop by the height of the label it does not have. The drop
	// is measured on the label itself, not read off the type ladder: a line of
	// text is taller than its type size, and the ladder left the button four
	// units above the fields (B-310).
	action := paint.Unlabelled(v.generate)

	// 🔴 Objects in READING order, which is not the order container.NewBorder
	// would produce. That helper appends the centre first, so the keyboard
	// would reach Generate before the settings it acts on. This package has
	// been caught by that once already, one function above.
	// 🔴 ONE object, named, and given to the layout twice.
	//
	// It was built twice for as long as it took to look at the window: the edge
	// the layout was told about and the object in the list were two different
	// containers holding the same button. A border layout finds its edges by
	// COMPARING the objects it was given, so it recognised neither as the right
	// edge, laid the button out as a second centre, and it drew across the whole
	// band over the four fields. Nothing fell over, because Fyne does not clip.
	//
	// Neither the gap probe nor the overflow probe saw it: overlapping siblings
	// stop being adjacent, so the gap count went DOWN, and a child drawing over
	// its siblings is still inside its parent. The stored tree blessed it,
	// because a pattern records what happened. The owner saw it.
	//
	// The button follows the seed since 2026-10-21 and the room after it is
	// left empty, so that no field is handed width it has no value for.
	padded := paint.PadLeft(palette.NameStepGroup, action)
	v.settings.Objects = []fyne.CanvasObject{
		paint.Lead(container.NewHBox(withSeed, padded), paint.Spacer()),
	}
	v.settings.Refresh()
}

// countDigits is how many digits the count field takes: ONE MORE than the
// ceiling has.
//
// 🔴 One more since 2026-10-21. With exactly as many, a number over the ceiling
// could not be typed at all, and the owner typing a million got the first five
// digits of it - ten thousand values, and not a word about the zeros that went.
// One digit more lets every number over the ceiling reach the screen, which
// then refuses it in words and points at the command line, which takes any.
var countDigits = len(strconv.Itoa(mostShown)) + 1

// countSample is the SHAPE of the longest count the field takes, used to size
// the field: the ceiling with one more digit. The guard beside this file keeps
// it in step with countDigits.
var countSample = strconv.Itoa(mostShown) + strconv.Itoa(0)

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

// settingLocale is the key the choice of locale is held under in v.showing,
// beside the roles. It cannot be mistaken for one, because a role is a name the
// mode reads and the mode has no role of this name.
const settingLocale = "locale"

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
	v.confirmOnly(v.copy)
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
	if strings.TrimSpace(v.howMany.Text) != commandDefault {
		asked = strings.TrimSpace(v.howMany.Text)
	}
	return commandline.New(commandline.Quick).
		Flag(commandline.Count, asked).
		Flag(commandline.Locale, v.localeAsked()).
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
	v.confirmOnly(v.copyCommand)
}

// copyOne puts a single value in the clipboard, which is what clicking a row
// does, and the row says so where the pointer is, in the word the Copy all
// button uses: the same act on a smaller amount.
func (v *View) copyOne(row *parts.TapRow, value string) {
	if v.clipboard == nil {
		return
	}
	v.clipboard.SetContent(value)
	v.confirmOnly(row)
}

// confirmer is a control that can say for a while that it filled the
// clipboard, and stop saying it.
type confirmer interface {
	Confirm()
	Settle()
}

// confirmOnly lets the control that filled the clipboard say so, and stops the
// one that said it before.
//
// 🔴 One at a time, as it was when the words stood beside the buttons: the
// values and the command line are different things, and two controls saying
// they filled the clipboard would say it holds both, when it holds whichever
// was copied last.
func (v *View) confirmOnly(next confirmer) {
	if v.confirmed != nil && v.confirmed != next {
		v.confirmed.Settle()
	}
	next.Confirm()
	v.confirmed = next
}

// unconfirm stops whichever control is saying it filled the clipboard, which
// is what a new set of values means: the clipboard no longer holds what is on
// screen.
func (v *View) unconfirm() {
	if v.confirmed != nil {
		v.confirmed.Settle()
		v.confirmed = nil
	}
}

// mostShown is the largest number of values this screen draws at once.
//
// 🔴 It is the mode's number rather than one written here, and that move was
// forced by a guard rather than chosen: the register of literals in
// internal/guard refuses any bare number inside the window, because a number
// there is almost always a distance and distances come from the scale. This one
// is a COUNT and the guard has no way to tell the two apart - so the honest
// place for it is beside the mode that produces the values, where what bounds
// it is written out.
const mostShown = quick.MostShownAtOnce

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
// It belongs to the country of the locale this screen opens on, which is en-US
// and puts the kinds of that country under the first heading of the list, so the
// kind on screen stands where a person opening the list looks first. The choice
// of locale itself is not beside it, because an SSN is the same in every
// locale (B-367). It has a reserved range, so the values shown before anybody
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
// different refusal. The guard beside this file holds it against quick.Kinds in
// both directions.
var refusedSays = map[quick.RefusalKind]i18n.Key{
	quick.UnknownGenerator: i18n.QuickRefusedGenerator,
	quick.RoleNotRead:      i18n.QuickRefusedRole,
	quick.UnreadableGender: i18n.QuickRefusedGender,
	quick.UnreadableDate:   i18n.QuickRefusedDate,
	quick.CountNotPositive: i18n.QuickRefusedCount,
	quick.CountNotWhole:    i18n.QuickRefusedCountShape,
	quick.Unavailable:      i18n.QuickRefusedUnavailable,
}
