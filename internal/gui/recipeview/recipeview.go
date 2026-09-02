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
package recipeview

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/app/validate"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
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
}

// View is the screen and what it currently says about what is written on it.
type View struct {
	text  i18n.Catalogue
	check Check
	write Write
	work  background.Work
	// window is what a directory chooser opens over, and nothing else. A screen
	// that reaches further into the window than that is a screen that cannot be
	// built without one, and every stored state of this one is built without.
	window fyne.Window

	editor *parts.Editor
	// answer is what is drawn under the recipe: the problems, or the sentence
	// that there are none, or the sentence saying the recipe is being read.
	answerBox  *fyne.Container
	answerPane *parts.KeyScroll
	// answer is what the last finished check said about the recipe.
	answer Answer
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

	// The bar along the bottom and what the last run left behind.
	out      *widget.Entry
	choose   *parts.Button
	replace  *widget.Check
	run      *parts.Button
	status   *fyne.Container
	halt     *background.Stopper
	stopping bool
	written  produce.Written
	failed   error
	ran      bool
	// closing says somebody asked to leave while a run was going on, so that
	// the window is given up once the run has ended rather than while it is
	// still writing.
	closing      bool
	releaseWatch func()
}

// New builds the screen.
func New(text i18n.Catalogue, check Check, write Write, work background.Work,
	window fyne.Window) *View {
	view := &View{text: text, check: check, write: write, work: work, window: window}
	view.buildBar()
	view.editor = parts.NewEditor(view.Recheck)
	view.editor.SetPlaceHolder(text.T(i18n.RecipePlaceholder))
	view.answerBox = container.NewVBox()
	// 🔴 Built here rather than in Content, because it can take focus. The
	// screen next door builds its panes in Content and carries B-203 for it: a
	// second call there replaces a focusable widget while the keyboard is still
	// standing on the old one, which is unreachable today and one refactor away
	// from being a bug. Built once, the question cannot arise.
	view.answerPane = parts.NewKeyScroll(view.answerBox)
	return view
}

// Content is the whole screen.
//
// The recipe is above and what is wrong with it below, each given the full
// width, and the divider between them belongs to the person.
//
// 🔴 Not the left and right of the screen next door, and the difference is
// measured rather than a matter of taste. That screen splits sideways because
// both of its halves are narrow. Here both halves want width: the window opens
// at 900 with the column of screen names taking 220, which leaves 680 for the
// content and 340 for half of it, and the longest line of the documented
// recipe is 93 characters. A recipe folded at 340 pixels is a recipe nobody can
// read, and a problem is a sentence.
//
// The parts are named in reading order, because container.NewBorder appends its
// edges after its middle and this project has twice paid for the walk that
// comes out of that.
func (v *View) Content() fyne.CanvasObject {
	// 🔴 The vertical split rather than a fixed division, so that a person
	// reading a long recipe can take the room from the answer and a person
	// reading eleven problems can take it back. Neither of the two is the one
	// that always deserves more.
	split := container.NewVSplit(v.editor, v.answerPane.Framed())
	bar := v.bar()
	// 🔴 No share is set here, and the empty line is the decision. Giving the
	// recipe the larger half reads as obviously right and is a number nobody
	// measured, and the guard over values in the window said so: the view holds
	// neither colours nor distances of its own, and a proportion invented at the
	// one place it is used is exactly what that guard was written before the
	// first screen to prevent. The toolkit's own half and half is taken instead,
	// and the divider belongs to the person, who is the only one here who knows
	// whether they are reading the recipe or the answer.
	v.draw()
	// 🔴 The split is named before the bar, because the layout walks its objects
	// in the order they are given and a person reads this screen downwards. The
	// bar is drawn along the bottom either way, so the two orders differ only in
	// where the keyboard goes, and this project has twice paid for the walk that
	// comes out of taking the drawing for the order.
	return parts.Inset(container.New(layout.NewBorderLayout(nil, bar, nil, nil), split, bar))
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

	// An empty screen is not a recipe with no problems. It gets a sentence
	// saying what to do, which is point eight of the GUI checklist, and no
	// check is started for it.
	if recipe == "" {
		v.checking = false
		v.checked = false
		v.answer = Answer{}
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
			v.draw()
		}
	})
}

// Checking reports whether a check is going on, for a caller with nobody
// looking at the screen.
func (v *View) Checking() bool { return v.checking }

// draw puts the answer under the recipe.
//
// The problems TAKE THE PLACE of the verdict rather than standing beside it,
// because both answer the same question and a screen that shows both at once
// makes a person work out which one is current. It is the same shape the screen
// next door uses for a refusal.
func (v *View) draw() {
	v.answerBox.Objects = nil
	switch {
	case v.checking:
		v.answerBox.Add(sentence(v.text.T(i18n.RecipeChecking)))
	case !v.checked:
		v.answerBox.Add(sentence(v.text.T(i18n.RecipeEmpty)))
	case len(v.answer.Report.Problems) > 0:
		v.drawProblems()
	case v.answer.Refused != nil:
		// 🔴 A recipe the validator approved and the plan will not run. It takes
		// the place of the approval rather than standing under it, for the same
		// reason the problems do: two answers to one question make a person work
		// out which is current, and here one of them would be wrong.
		v.drawRefusal()
	default:
		v.drawClean()
		v.drawPreview()
	}
	v.drawWillNotDo()
	v.drawUnchecked()
	v.answerBox.Refresh()
	// The bar says what a run would do now, so it moves with every answer.
	v.drawBar()
}

// drawRefusal says that the recipe reads correctly and cannot be run anyway.
//
// The failure carries the four parts a refusal has, already assembled by
// whoever raised it, so it is drawn as the sentence it is rather than taken
// apart here.
func (v *View) drawRefusal() {
	v.answerBox.Add(heading(v.text.T(i18n.RecipeRefused)))
	v.answerBox.Add(sentence(v.answer.Refused.Error()))
}

// drawWillNotDo lists what the plan says the run will not do.
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
func (v *View) drawWillNotDo() {
	if len(v.answer.Sample.Notes) == 0 {
		return
	}
	v.answerBox.Add(widget.NewSeparator())
	v.answerBox.Add(heading(v.text.T(i18n.RecipeWillNotDo) + " (" +
		strconv.Itoa(len(v.answer.Sample.Notes)) + ")"))
	for _, note := range v.answer.Sample.Notes {
		v.answerBox.Add(detail(note.Where + ": " + note.What))
	}
}

// drawProblems lists what is wrong, each at the line it was written on.
func (v *View) drawProblems() {
	v.answerBox.Add(heading(v.text.T(i18n.RecipeHeadingProblems) + " (" +
		strconv.Itoa(len(v.answer.Report.Problems)) + ")"))
	for _, found := range v.answer.Report.Problems {
		v.answerBox.Add(sentence(v.place(found.Where, found.Line) + found.What))
		if found.Allowed != "" {
			v.answerBox.Add(detail(v.text.T(i18n.RecipeAllowed) + ": " + found.Allowed))
		}
		if found.Hint != "" {
			v.answerBox.Add(detail(v.text.T(i18n.RecipeTry) + ": " + found.Hint))
		}
	}
}

// drawClean says the recipe can be used, and what was read to say so.
//
// The counts are there for the same reason the command line prints them: they
// are how somebody sees that the recipe was READ rather than skimmed, and a
// bare approval of a file the tool misunderstood is the one answer this product
// must never give.
func (v *View) drawClean() {
	v.answerBox.Add(heading(v.text.T(i18n.RecipeNoObjections)))
	v.answerBox.Add(detail(
		counted(v.answer.Report.Entities, v.text.T(i18n.RecipeEntity),
			v.text.T(i18n.RecipeEntities)) + ", " +
			counted(v.answer.Report.Fields, v.text.T(i18n.RecipeField),
				v.text.T(i18n.RecipeFields))))
}

// counted writes a number with the thing it counts, in whichever form of the
// word fits it.
//
// 🔴 Both forms come from the catalogue rather than one of them plus a letter.
// The letter was what this screen drew until 2026-09-02, and rendering read it
// back: two entities came out as "2 entitys". English alone breaks that rule,
// so this was never waiting for a language with three plural forms. What is
// still waiting for one is B-184, because a language that needs a third form
// needs a third key and a rule for choosing between them, and that is a
// mechanism rather than a second word.
func counted(many int, one, rest string) string {
	if many == 1 {
		return "1 " + one
	}
	return strconv.Itoa(many) + " " + rest
}

// drawUnchecked says what this build did not look at.
//
// It is drawn whether or not there were problems, for the reason the report
// type gives: a clean answer that hides its own blind spots invites trust it
// has not earned.
func (v *View) drawUnchecked() {
	if !v.checked || len(v.answer.Report.Unchecked) == 0 {
		return
	}
	v.answerBox.Add(widget.NewSeparator())
	v.answerBox.Add(heading(v.text.T(i18n.RecipeUnchecked) + " (" +
		strconv.Itoa(len(v.answer.Report.Unchecked)) + ")"))
	for _, skipped := range v.answer.Report.Unchecked {
		v.answerBox.Add(detail(skipped.Where + ": " + skipped.What + " - " + skipped.Reason))
	}
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
