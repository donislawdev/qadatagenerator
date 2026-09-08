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
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	toolkit "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/commandline"
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
	born    *widget.Entry
	seedBox *widget.Entry
	copy    *parts.Button
	again   *parts.Button

	// form carries every setting, and its rows are rebuilt rather than hidden.
	//
	// 🔴 One form rather than one per row, and the render is what settled it:
	// five forms stacked gave five label columns of five different widths, so
	// the labels stood in a ragged line. A form of the toolkit cannot hide one
	// of its items, which is why the rows are assembled again whenever what is
	// chosen changes.
	form *widget.Form

	value *widget.Label
	told  *widget.Label

	// The parts of the right column that have nothing in them until this
	// screen has produced something, and the sentence that stands where they
	// would be. Held here because settle shows and hides them, and a screen
	// built once cannot reach them any other way.
	produced   *fyne.Container
	terminal   *fyne.Container
	nothingYet fyne.CanvasObject

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

	v.copy = parts.NewButton(v.text.T(i18n.QuickCopy), v.putInClipboard)
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

	// The value is the subject of this screen, so it is the largest thing on
	// it and it is monospace: an identifier read off a screen is read digit by
	// digit, and a proportional font makes ones and sevens argue.
	//
	// 🔴 The sentence above was written before the window could say a size, so
	// until 2026-09-08 it described an intention and the value was drawn at the
	// same size as the labels beside it. Seen on a real window that day, it was
	// not the largest thing on the screen and was not even the largest thing in
	// its own column. The heading size is what the toolkit answers above the
	// body, so the claim is now the code as well as the comment.
	//
	// The accepted drawing asks for 30 against a body of 13. The largest size
	// this theme answers for is 24 against a body of 14, and going past it
	// would mean writing a number into the window, which is the one thing the
	// guard over values here refuses.
	v.value = widget.NewLabelWithStyle("", fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true, Monospace: true})
	v.value.SizeName = toolkit.SizeNameHeadingText
	v.told = widget.NewLabel("")
	v.told.Importance = widget.LowImportance

	v.form = widget.NewForm()

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
	v.locales.SetSelected(quick.DefaultLocale)
}

// lay puts the controls on the screen, apart from making them.
//
// Two functions rather than one, because building this screen passed the limit
// on the length of a function, and the limit is bent for nothing. The seam is
// the one that costs least: what the controls ARE does not change when the
// arrangement of them does.
func (v *View) lay() {
	// 🔴 Two columns, which is what the accepted drawing asks for and what the
	// window did not do until 2026-09-08.
	//
	// Seen on a real window that day: everything on this screen stood in one
	// column about 450 wide inside a content area of about 850, so most of the
	// screen was empty and the two buttons hung over the emptiness with the
	// thing they act on far above them. The drawing puts what somebody SETS on
	// the left and what they GET on the right, which also puts the buttons
	// beside what they copy rather than adrift below it.
	//
	// A grid of two rather than a split, because neither side is one a person
	// would want to resize: both hold a fixed number of short things. The split
	// on the recipe screen is a different case and says so there.
	settings := parts.Column(palette.NameStepGroup, v.form)

	// 🔴 What stands here before a value exists. Until 2026-09-08 the answer
	// was an empty box with a border round it and two buttons under it that
	// copied nothing, because the drawing this screen was built from only ever
	// shows a screen that has already produced something. The state the screen
	// OPENS in was never drawn, and it is the state a person sees first.
	v.nothingYet = subdued(v.text.T(i18n.QuickNothingYet))
	v.produced = parts.Column(palette.NameStepRelated,
		parts.Panel(v.value),
		parts.Row(palette.NameStepRelated, v.copy.OnScreen(), v.again.OnScreen()),
		v.told,
	)
	// The bridge to the other surface, and it comes last because it answers a
	// question somebody asks after they have what they came for: how do I put
	// this in a script.
	v.terminal = parts.Column(palette.NameStepRelated,
		parts.Section(v.text.T(i18n.CommandHeading)),
		parts.Panel(v.command),
		parts.Row(palette.NameStepRelated, v.copyCommand.OnScreen(), layout.NewSpacer()),
	)

	// In reading order within each column, and the left column before the
	// right, because the toolkit walks the objects in the order they were given
	// and not in the order they are drawn. The screen next door paid for that
	// once: the button at the top of it was the last stop.
	answer := parts.Column(palette.NameStepSection,
		parts.Section(v.text.T(i18n.QuickValueHeading)),
		v.nothingYet,
		v.produced,
		v.terminal,
	)

	// The gap between the two columns is carried by the right one, because the
	// grid of the toolkit spaces its cells by the theme's padding and takes no
	// distance of its own. Two columns four pixels apart read as one column.
	beside := palette.StepOf(palette.NameStepMajor)
	columns := container.New(layout.NewGridLayout(2), settings,
		container.New(layout.NewCustomPaddedLayout(0, 0, beside, 0), answer))

	// 🔴 The columns take the whole width rather than being held to the width
	// this screen was drawn at.
	//
	// Held, they were measured on 2026-09-08 and turned down the same day by the
	// owner looking at the result: at a window of 1670 the content stopped at
	// 852 and the right half of the screen was empty. The drawing does not say
	// what should happen at that width, because it was made at 900 - but what it
	// does say is that the two columns are each half of whatever there is, so
	// filling is the reading that follows it and holding was mine.
	//
	// What this costs is a wide field for a short value, and that cost is real:
	// a seed is eight characters in a control that is now most of a column. It
	// is the drawing's own arithmetic, and the alternative was a window that
	// looks unfinished, which is the louder fault of the two.
	v.content = parts.Inset(parts.Column(palette.NameStepSection,
		subdued(v.text.T(i18n.QuickNothingWritten)),
		columns,
	))
}

// settle shows the parts of this screen that have something in them, and hides
// the ones that do not.
//
// # Why hidden rather than drawn empty
//
// A box with a border and nothing in it reads as a thing that failed to load.
// Seen on a real window on 2026-09-08 at the size the owner runs it: the screen
// opened showing two of them, one under a heading saying Value and one under a
// heading saying the same in a terminal, with a button beneath each that would
// have copied an empty string.
//
// # Why the two are settled separately
//
// The command is written even when the value was refused, which is a decision
// made where it is written and worth keeping: somebody whose request was turned
// down may still want to show that request to somebody else. So the terminal
// row follows the command and the value follows the value, and a refusal shows
// the line that caused it with no value above it.
func (v *View) settle() {
	if v.value.Text == "" {
		v.produced.Hide()
		v.nothingYet.Show()
	} else {
		v.produced.Show()
		v.nothingYet.Hide()
	}
	if v.command.Text == "" {
		v.terminal.Hide()
	} else {
		v.terminal.Show()
	}
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
func (v *View) Value() string { return v.value.Text }

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
		v.value.SetText("")
		v.command.SetText("")
		v.showRows("")
		return
	}
	v.showRows(chosen)

	if v.seed == "" {
		drawn, err := quick.NewSeed()
		if err != nil {
			v.value.SetText("")
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
		v.value.SetText("")
		v.told.SetText(v.text.T(refusedSays[refusal.Kind]))
		return
	}
	produced, err := quick.Run(asked)
	if err != nil {
		v.value.SetText("")
		v.told.SetText(err.Error())
		return
	}
	v.value.SetText(produced)
}

// request is what the person on this screen asked for.
func (v *View) request(generator string) quick.Request {
	return quick.Request{
		Generator: generator, Locale: v.locale, Seed: v.seed, Agrees: v.bindings(),
	}
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
func (v *View) showRows(generator string) {
	rows := []*widget.FormItem{
		widget.NewFormItem(v.text.T(i18n.QuickLabelValue), v.chooser.OnScreen()),
		widget.NewFormItem(v.text.T(i18n.QuickLabelLocale), v.locales.OnScreen()),
	}
	if quick.Reads(generator, quick.RoleGender, v.locale) {
		rows = append(rows, widget.NewFormItem(v.text.T(i18n.QuickLabelGender), v.gender.OnScreen()))
	}
	if quick.Reads(generator, quick.RoleBirthDate, v.locale) {
		rows = append(rows, widget.NewFormItem(v.text.T(i18n.QuickLabelBorn), v.born))
	}
	rows = append(rows, widget.NewFormItem(v.text.T(i18n.QuickLabelSeed), v.seedBox))

	if sameRows(v.form.Items, rows) {
		return
	}
	v.form.Items = rows
	v.form.Refresh()
}

// sameRows reports whether the form already holds these rows, so that a redraw
// on every keystroke does not rebuild a form that has not changed.
func sameRows(held, wanted []*widget.FormItem) bool {
	if len(held) != len(wanted) {
		return false
	}
	for at := range held {
		if held[at].Text != wanted[at].Text {
			return false
		}
	}
	return true
}

// Shows reports whether the field for that role is on screen, for a test that
// asks the screen rather than counting pixels.
func (v *View) Shows(role string) bool {
	label := v.text.T(i18n.QuickLabelGender)
	if role == quick.RoleBirthDate {
		label = v.text.T(i18n.QuickLabelBorn)
	}
	for _, row := range v.form.Items {
		if row.Text == label {
			return true
		}
	}
	return false
}

// putInClipboard copies the value and says that it did.
//
// Saying so is not decoration. A clipboard cannot be seen, so a button that
// silently succeeded and a button that silently did nothing look the same.
func (v *View) putInClipboard() {
	if v.value.Text == "" || v.clipboard == nil {
		return
	}
	v.clipboard.SetContent(v.value.Text)
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
	return commandline.New(commandline.Quick).
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
}

func subdued(text string) fyne.CanvasObject {
	label := widget.NewLabel(text)
	label.Importance = widget.LowImportance
	label.Wrapping = fyne.TextWrapWord
	return label
}
