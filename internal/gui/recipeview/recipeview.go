// Package recipeview builds the screen a person writes a recipe on and sees,
// as they type, what is wrong with it.
//
// # Why a text field rather than a form
//
// Decided in DEC-13 and not reopened here. A form built out of controls stops
// covering the whole of the format the moment the format grows, and then it
// quietly simplifies it, which breaks the promise that the window edits the
// same file the command line reads. What the window adds over a plain editor is
// not structure but an answer: the recipe is checked while it is written.
//
// # Why checking is the whole of the first step
//
// The screen was ordered because somebody who opens this window can generate
// nothing with it. The step that closes that ends in files on a disk, and this
// one deliberately does not: it reaches the point where a recipe can be written
// and judged, and stops there. That is the same order the take and corrupt
// screen was built in, where the preview came before the button that writes,
// and for the same reason. A control that writes before anybody has seen what
// it would write is a control nobody can check.
//
// # The shape, since 2026-09-15 (DEC-246)
//
// A band at the top holds where a run writes, whether it may replace what is
// there, and the one action of the screen. The body is two equal columns: the
// recipe on the left, and on the right the one card, saying what the recipe
// describes while somebody writes it, or what the last run did once they have
// pressed the button. A band at the foot holds the command that does the same
// in a terminal. It is the shape of the Quick and take screens, because the
// three are one program, and the columns are equal rather than a column and an
// aside because both halves want width: a recipe is written in lines and a
// preview is a table.
//
// 🔴 What the shape repairs is measured rather than felt. Until then the
// controls that write and the terminal line stood UNDER the answer in the
// right half of a divider, so at the size the owner keeps the window the button
// was below the fold (B-280), and this was the one screen of five with a
// divider, which the keyboard could not move.
package recipeview

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/app/validate"
	"github.com/donislawdev/qadatagenerator/internal/damage"
	"github.com/donislawdev/qadatagenerator/internal/fault"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
	"github.com/donislawdev/qadatagenerator/internal/produce"
)

// Check reads a recipe and says what is wrong with it.
//
// It is handed in rather than reached for, the way the take and corrupt screen
// takes its reader. What matters here is that the window and the command line
// ask the SAME question: this port is filled with the same use case, the same
// decoder, the same generator registry and the same catalog that qadg validate
// uses, so a recipe called clean here is a recipe called clean there. A screen
// that built its own answer would be a second implementation of the rule, and
// this project has paid for one of those already.
type Check func(name string, data []byte) Answer

// Answer is everything this screen learned about one recipe.
//
// # Why one answer rather than two ports
//
// Judging a recipe and working out what it would produce both begin by reading
// the same bytes, and a recipe is allowed to be 256 KB. Two ports would mean two
// readings on every key press, and the note on the reader says a document inside
// that limit can cost hundreds of megabytes to parse. They also have to agree:
// an approval drawn beside rows from an earlier recipe is the worst of the
// outcomes, because it looks like an answer.
type Answer struct {
	// Report is what the recipe is, as validate sees it.
	Report validate.Report
	// Sample is the first rows of each table the recipe describes. It is empty
	// when the report has problems, because a recipe that cannot be used
	// produces nothing to show.
	Sample produce.Sample
	// Refused is the failure of a recipe that passed validation and still
	// cannot be run.
	//
	// 🔴 This is not a gap between the two, it is the gap this project measured
	// on 2026-09-02 and recorded as B-205: the plan and the writer know things
	// the validator did not ask. One of those was repaired, and treating the
	// rest as impossible is how the next one would arrive as an approval drawn
	// over a refusal.
	Refused error
	// Damaged says the recipe orders damage, so that a run writes a manifest
	// beside the data and the rows of the sample are the rows before it.
	Damaged bool
	// Planned is how much of that damage this build carries out, split by the
	// layers the policy names. It is empty when nothing is ordered.
	//
	// 🔴 The window said nothing about a policy before 2026-09-24, and wrote
	// nothing for one either (B-300). A dry run on the command line says how
	// much of the order this build can carry out, and this is the same answer,
	// worked out by the same walk over the catalog (DEC-252).
	Planned damage.Planned
}

// View is the screen and what it currently says about what is written on it.
type View struct {
	text  i18n.Catalogue
	check Check
	write Write
	work  background.Work
	// window is what a directory chooser opens over and what a run takes over
	// while it is going on, and nothing else. A screen that reaches further
	// into the window than that is a screen that cannot be built without one,
	// and every stored state of this one is built without.
	window fyne.Window

	editor *parts.Editor
	// answer is what the last finished check said about the recipe.
	answer Answer
	// crashed is set when reading the recipe came apart rather than finished.
	//
	// It is a field of its own rather than a fourth outcome inside Answer,
	// because Answer is what a check says about a recipe and this says nothing
	// about the recipe at all. It is cleared by the next check, like every
	// other thing on this screen that describes what is written beside it.
	crashed error
	// checked says whether anything has been judged yet, which is not the same
	// as a report with no problems in it. An empty screen has no report and a
	// valid recipe has one, and drawing them alike would tell somebody their
	// blank window had been approved.
	checked bool
	// asked counts the checks asked for, so that an answer which is no longer
	// the latest question is dropped rather than drawn.
	//
	// 🔴 Superseding rather than refusing, for the reason the screen next door
	// records: a recipe is typed one character at a time and every character
	// asks the question again. Refusing the new question would leave the answer
	// on screen describing a recipe that is no longer in the field, which is
	// the worst of the three outcomes because it looks like an answer.
	//
	// Read and written only on the thread that draws, which is what lets it be
	// a plain number rather than anything with a lock around it.
	asked int
	// checking is true while a check is in flight, so the screen can say so.
	checking bool

	// The one card and the heading over it, refilled with what is on screen.
	// The card is a column inside a scrolling area, so it hugs a sentence and
	// scrolls a preview, and the heading says whether it is about the recipe or
	// about the last run.
	cardHeading *fyne.Container
	cardBox     *fyne.Container
	card        *parts.KeyScroll
	content     fyne.CanvasObject

	// The two controls that answer what can be written here rather than what is
	// written. They belong to the build and not to the recipe, so neither is
	// rebuilt when the answer changes.
	example    *parts.Button
	generators *parts.Disclosure

	// The band along the top: where a run writes, whether it may replace what
	// is there, and the one action, which is the button that generates or the
	// button that stops, of which one is shown.
	out      *parts.FieldEntry
	choose   *parts.Button
	replace  *parts.Check
	generate *parts.Button
	stop     *parts.Button
	action   *fyne.Container
	// form is the extra form of the manifest a run exports, as the run takes
	// it: empty for none.
	form string
	// keys moves the keyboard off the button a run takes away and back when
	// the run ends, the same way the take screen does.
	keys parts.Handover

	// The band at the foot: the line that would do this in a terminal, and the
	// button beside it, which cannot be pressed because this window writes no
	// recipe file.
	command     fyne.CanvasObject
	commandText *widget.Label
	copyCommand *parts.Button

	// What a run is doing and what the last one left. The stage decides what
	// the card draws and the two beside it carry what there is to say.
	stage   stage
	halt    *background.Stopper
	written produce.Written
	failed  error
	// closing says somebody asked to leave while a run was going on, so that
	// the window is given up once the run has ended rather than while it is
	// still writing.
	closing      bool
	releaseWatch func()
}

// New builds the screen.
func New(text i18n.Catalogue, check Check, write Write, work background.Work,
	window fyne.Window) *View {
	view := &View{
		text: text, check: check, write: write, work: work, window: window,
		cardHeading: container.NewStack(), action: container.NewStack(),
	}
	view.buildBand()
	view.editor = parts.NewEditor(view.Recheck)
	view.editor.SetPlaceHolder(text.T(i18n.RecipePlaceholder))
	view.buildHints()
	view.buildCommand()
	view.lay()
	view.draw()
	return view
}

// Content is the whole screen.
func (v *View) Content() fyne.CanvasObject { return v.content }

// lay puts the controls on the screen, apart from making them.
//
// 🔴 Every layout here is given its objects in reading order, which is the
// order the keyboard walks them. container.NewBorder appends its edges after
// its middle, and this project has three times paid for the walk that comes out
// of that, the last time one function after writing a paragraph about it.
func (v *View) lay() {
	// 🔴 The card stands in a COLUMN inside the scrolling area, not in it
	// directly. A scrolling area hands its content the larger of the content's
	// minimum and its own size, so a card put straight into it stretched to
	// the foot of the column and the state the screen opens in was an empty
	// framed rectangle over two thirds of it, seen on the first render of the
	// mock. A column gives its child exactly its minimum, so the card hugs a
	// sentence, and with a preview in it is taller than the window and
	// scrolls. No layout of our own was needed for that, which matters: two
	// of those already stand red in the register without an entry.
	v.cardBox = paint.Column(palette.NameStepGroup)
	v.card = parts.NewKeyScroll(container.NewVBox(paint.Card(palette.NameStepGroup, v.cardBox)))

	recipeHeading := v.heading(i18n.RecipeHeadingRecipe)
	left := container.New(layout.NewBorderLayout(recipeHeading, nil, nil, nil),
		recipeHeading, v.editor.Framed())
	right := container.New(layout.NewBorderLayout(v.cardHeading, nil, nil, nil),
		v.cardHeading, v.card.Framed())
	// Two equal columns: as many columns as halves, so the count is read from
	// the slice rather than written as a bare number the theme has no name for.
	// The gap between them is one step of the scale, taken half from each cell,
	// so that the halves stay equal.
	halves := []fyne.CanvasObject{
		paint.Pad(palette.NameStepHairline, palette.NameStepHairline,
			palette.NameStepHairline, palette.NameStepRelated, left),
		paint.Pad(palette.NameStepHairline, palette.NameStepHairline,
			palette.NameStepRelated, palette.NameStepHairline, right),
	}
	columns := container.NewGridWithColumns(len(halves), halves...)
	body := paint.Pad(palette.NameStepSection, palette.NameStepSection,
		palette.NameStepMajor, palette.NameStepMajor, columns)

	band := v.settingsBand()
	v.command = v.commandBand()
	v.content = container.New(layout.NewBorderLayout(band, v.command, nil, nil),
		band, body, v.command)
}

// heading is the title of one column, with the line every section heading of
// this window carries.
func (v *View) heading(key i18n.Key) fyne.CanvasObject {
	return paint.Pad(palette.NameStepHairline, palette.NameStepRelated,
		palette.NameStepHairline, palette.NameStepHairline,
		paint.SectionHeading(v.text.T(key), paint.Spacer()))
}

// SetRecipe puts a recipe on the screen and judges it, without anybody typing.
//
// # Why this is exported
//
// The same reason the screen next door exports the way it opens a file: the
// guard that holds what every screen draws, and the probe that renders each
// state to a picture, both have to reach a state without a keyboard. Reaching
// it through the field the person types into, rather than by writing the answer
// directly, is what keeps those pictures pictures of a state somebody can get
// to.
// 🔴 It puts the text in and stops. SetText reaches OnChanged through
// updateTextAndRefresh in widget/entry.go, so the check below starts on its
// own, and calling Recheck here as well would run it twice. That is not merely
// wasteful: a second call of the same line is how a mutation aimed at the first
// one stopped proving anything in this project a session ago. What holds the
// assumption is a guard rather than this paragraph, because the day the toolkit
// stops calling back is the day this screen would silently answer about nothing.
func (v *View) SetRecipe(recipe string) {
	v.editor.SetText(recipe)
}

// Report is what the last finished check said, and whether there has been one.
func (v *View) Report() (validate.Report, bool) { return v.answer.Report, v.checked }

// Recheck judges what is on the screen, off the thread that draws.
//
// # Why this cannot happen where it is asked for
//
// A recipe is allowed to be 256 KB, and the note on the reader says a document
// inside that limit can cost hundreds of megabytes to parse. Doing that on the
// drawing thread on every keystroke is the fault the window closed once already
// when reading somebody else's file, and it looks the same from outside: the
// window stops answering and nothing says why.
func (v *View) Recheck(recipe string) {
	v.asked++
	wanted := v.asked
	// 🔴 What the last run did is let go of on the first keystroke after it
	// (DEC-246). The card says one thing: what the recipe describes, or what
	// the last press did, whichever is newer, and a result kept beside a
	// recipe it was not made from would belong to neither. The files of that
	// run are still on the disk, and the manifest browser is where their
	// detail lives. A run going on is not let go of, because it is the newer
	// thing for as long as it goes on.
	if !v.busy() {
		v.forgetLastRun()
	}

	// An empty screen is not a recipe with no problems. It gets a sentence
	// saying what to do, which is point eight of the GUI checklist, and no
	// check is started for it.
	if recipe == "" {
		v.checking = false
		v.checked = false
		v.answer = Answer{}
		v.crashed = nil
		v.draw()
		return
	}

	v.checking = true
	v.draw()

	name := v.text.T(i18n.RecipeName)
	data := []byte(recipe)
	check := v.check
	v.work.Read(func() func() {
		report := check(name, data)
		return func() {
			if wanted != v.asked {
				// A later keystroke has already asked the question again, so
				// this answer is about a recipe nobody has on screen.
				return
			}
			v.checking = false
			v.checked = true
			v.answer = report
			v.crashed = nil
			v.draw()
		}
	}, func(err error) func() {
		// 🔴 A crash is NOT drawn as a refusal, though the screen has one and
		// it would have fitted. A refusal here says that the recipe reads
		// correctly and still cannot be run, which is a sentence about what
		// somebody wrote. This is a sentence about us, and putting our defect
		// under their heading is the dishonesty this whole tool argues against.
		return func() {
			if wanted != v.asked {
				return
			}
			v.checking = false
			v.checked = true
			v.answer = Answer{}
			v.crashed = err
			v.draw()
		}
	})
}

// Checking reports whether a check is going on, for a caller with nobody
// looking at the screen.
func (v *View) Checking() bool { return v.checking }

// draw puts on the screen everything that depends on its state: the card and
// its heading, the one action in the band, and the terminal line.
func (v *View) draw() {
	v.drawCard()
	v.drawAction()
	v.drawCommand()
}

// drawCard fills the card with the one thing it says, and the heading over it
// with which thing that is.
//
// A run, going on or finished, takes the place of the answer rather than
// standing beside it, because both answer the question of what pressing the
// button means and a screen that shows both at once makes a person work out
// which one is current. Among the answers, the problems TAKE THE PLACE of the
// verdict for the same reason. It is the same shape the take screen uses.
func (v *View) drawCard() {
	heading := i18n.RecipeHeadingDescribes
	var blocks []fyne.CanvasObject
	switch {
	case v.busy():
		heading = i18n.RecipeHeadingResult
		blocks = v.runningContent()
	case v.stage != stageReady:
		heading = i18n.RecipeHeadingResult
		blocks = v.outcome()
	default:
		blocks = v.answerContent()
	}
	v.cardBox.Objects = blocks
	v.cardBox.Refresh()
	v.cardHeading.Objects = []fyne.CanvasObject{v.heading(heading)}
	v.cardHeading.Refresh()
}

// answerContent is what the card says about the recipe on the screen, and
// under it in every case the list of what this build can generate.
//
// The list is last and in every answer, because it answers what can be written
// rather than anything about what is written. Closed until somebody opens it,
// and it stays open through the next answer because it is the same widget put
// back rather than a new one built. It stays in THIS card because this one
// scrolls, which is what an open list of seventeen names needs: a border gives
// its bottom edge the height that edge asks for, and rendering the open list
// as one ended the process on 2026-09-08 with a field laid out at minus 171.
func (v *View) answerContent() []fyne.CanvasObject {
	var blocks []fyne.CanvasObject
	switch {
	case v.checking:
		blocks = append(blocks, v.paragraph(v.text.T(i18n.RecipeChecking)))
	case v.crashed != nil:
		blocks = append(blocks, v.crash()...)
	case !v.checked:
		blocks = append(blocks, v.empty()...)
	case len(v.answer.Report.Problems) > 0:
		blocks = append(blocks, v.problems()...)
	case v.answer.Refused != nil:
		// 🔴 A recipe the validator approved and the plan will not run. It takes
		// the place of the approval rather than standing under it, for the same
		// reason the problems do: two answers to one question make a person work
		// out which is current, and here one of them would be wrong.
		blocks = append(blocks, v.refusal()...)
	default:
		blocks = append(blocks, v.clean()...)
		blocks = append(blocks, v.preview()...)
	}
	blocks = append(blocks, v.willNotDo()...)
	blocks = append(blocks, v.unchecked()...)
	return append(blocks, paint.Hairline(palette.LineQuiet), v.generators.OnScreen())
}

// empty is what the card says to somebody who has written nothing yet: what to
// do, and the one control that does it for them.
//
// 🔴 The offer of an example belongs to the empty screen and to no other
// state. A control offering to replace what somebody has written is a control
// that has to ask first, and one sitting there disabled in every other state is
// a line of furniture. It stands beside the sentence that explains it rather
// than in the heading of the column (DEC-246), and pressing it is one of the
// ways the screen stops being empty, which is why it puts the keyboard in the
// field afterwards.
func (v *View) empty() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		v.paragraph(v.text.T(i18n.RecipeEmpty)),
		paint.Lead(v.example.OnScreen(), paint.Spacer()),
	}
}

// refusal says that the recipe reads correctly and cannot be run anyway.
//
// The failure carries the four parts a refusal has, already assembled by
// whoever raised it, so it is drawn as the sentence it is rather than taken
// apart here.
func (v *View) refusal() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		v.told(paint.IconAlert, palette.StateWarning, v.text.T(i18n.RecipeRefused)),
		v.paragraph(fault.Says(v.answer.Refused)),
	}
}

// crash says that this build came apart while reading the recipe.
//
// What happened comes first and what to do about it second, which is the order
// a refusal uses on the other surface. The first line names itself as a defect
// of ours, because that is how the failure was built, so the second line does
// not have to argue it and can be the one thing a person can actually do.
func (v *View) crash() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		paint.Marked(paint.IconBug, palette.StateDanger, sentence(fault.Says(v.crashed))),
		v.paragraph(v.text.T(i18n.RecipeCrashed)),
	}
}

// willNotDo lists what the plan says the run will not do.
//
// # Why this is a second list beside what was not checked
//
// They answer different questions and this project has one rule about that: a
// list that answers two is a list somebody reads as answering the easier one.
// What was not checked is about the READING of the recipe, and this is about
// the RUN it describes: a column this build has no values for, a setting that
// changes nothing in the form asked for. Until now the window carried the first
// and dropped the second, so a column that will come out empty had nowhere to
// be said.
func (v *View) willNotDo() []fyne.CanvasObject {
	if len(v.answer.Sample.Notes) == 0 {
		return nil
	}
	lines := []fyne.CanvasObject{paint.Caption(v.text.T(i18n.RecipeWillNotDo) + " (" +
		strconv.Itoa(len(v.answer.Sample.Notes)) + ")")}
	for _, note := range v.answer.Sample.Notes {
		lines = append(lines, v.paragraph(note.Where+": "+note.What))
	}
	return []fyne.CanvasObject{paint.Column(palette.NameStepHairline, lines...)}
}

// problems lists what is wrong, each at the line it was written on.
func (v *View) problems() []fyne.CanvasObject {
	blocks := []fyne.CanvasObject{
		v.told(paint.IconAlert, palette.StateDanger, v.text.T(i18n.RecipeHeadingProblems)+" ("+
			strconv.Itoa(len(v.answer.Report.Problems))+")"),
	}
	for _, found := range v.answer.Report.Problems {
		lines := []fyne.CanvasObject{sentence(v.place(found.Where, found.Line) + found.What)}
		if found.Allowed != "" {
			lines = append(lines, v.paragraph(v.text.T(i18n.RecipeAllowed)+": "+found.Allowed))
		}
		if found.Hint != "" {
			lines = append(lines, v.paragraph(v.text.T(i18n.RecipeTry)+": "+found.Hint))
		}
		blocks = append(blocks, paint.Column(palette.NameStepHairline, lines...))
	}
	return blocks
}

// clean says the recipe can be used, what was read to say so, and what a run
// would write.
//
// The counts are there for the same reason the command line prints them: they
// are how somebody sees that the recipe was READ rather than skimmed, and a
// bare approval of a file the tool misunderstood is the one answer this product
// must never give.
func (v *View) clean() []fyne.CanvasObject {
	verdict := paint.Column(palette.NameStepTight,
		v.told(paint.IconCheck, palette.StateSuccess, v.text.T(i18n.RecipeNoObjections)),
		v.paragraph(v.text.Counted(v.answer.Report.Entities, i18n.RecipeEntity, i18n.RecipeEntities)+
			", "+v.text.Counted(v.answer.Report.Fields, i18n.RecipeField, i18n.RecipeFields)))
	blocks := []fyne.CanvasObject{verdict}
	if would := v.wouldWrite(); would != nil {
		blocks = append(blocks, would)
	}
	return append(blocks, paint.Hairline(palette.LineQuiet))
}

// wouldWrite names the files a run would make and where, which is the answer a
// dry run gives at the other surface, or says that there is nowhere yet.
//
// The names come from the sample rather than from the recipe, because the
// sample got them by watching the writer ask for them. A screen working them
// out from the entities would be right about the comma separated form and
// wrong about the one that writes a single file.
//
// 🔴 The one reason the button can be disabled while the recipe is clean is
// said HERE, under the files, rather than beside the button (DEC-246): it is
// the first thing in the card after the verdict, so it stands above the fold,
// and it is about exactly the thing it stands under.
func (v *View) wouldWrite() fyne.CanvasObject {
	if len(v.answer.Sample.Files) == 0 {
		return nil
	}
	data := make([]string, 0, len(v.answer.Sample.Files))
	for _, file := range v.answer.Sample.Files {
		data = append(data, file.Name)
	}
	// The manifest and the note beside the data are named by the function the
	// run names them with, so that this list and the one after the run are one
	// sentence rather than two that agree today.
	names := produce.Writes(data, v.answer.Damaged, v.form)
	if v.out.Text == "" {
		return paint.Column(palette.NameStepHairline,
			paint.Caption(v.text.T(i18n.RecipePlanWrites)),
			v.paragraph(listOf(names)),
			v.paragraph(v.text.T(i18n.RecipeNeedDirectory)))
	}
	return v.fact(i18n.RecipePlanWrites, listOf(names)+" "+v.text.T(i18n.RecipeIn)+" "+v.out.Text)
}

// unchecked says what this build did not look at.
//
// It is drawn whether or not there were problems, for the reason the report
// type gives: a clean answer that hides its own blind spots invites trust it
// has not earned.
func (v *View) unchecked() []fyne.CanvasObject {
	if !v.checked || len(v.answer.Report.Unchecked) == 0 {
		return nil
	}
	lines := []fyne.CanvasObject{paint.Caption(v.text.T(i18n.RecipeUnchecked) + " (" +
		strconv.Itoa(len(v.answer.Report.Unchecked)) + ")")}
	for _, skipped := range v.answer.Report.Unchecked {
		lines = append(lines, v.paragraph(skipped.Where+": "+skipped.What+" - "+skipped.Reason))
	}
	return []fyne.CanvasObject{paint.Column(palette.NameStepHairline, lines...)}
}

// place names where a problem is, in the same words the command line uses.
//
// A problem about the file as a whole has no place inside it, and a place that
// cannot be resolved to exactly one line carries no line rather than a wrong
// one. Both of those are ordinary answers, so both are drawn as the absence of
// a word rather than as a zero.
func (v *View) place(where string, line int) string {
	if where == "" {
		return ""
	}
	if line <= 0 {
		return where + ": "
	}
	return where + " (" + v.text.T(i18n.RecipeLine) + " " + strconv.Itoa(line) + "): "
}
