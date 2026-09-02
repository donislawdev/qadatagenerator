package recipeview

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	"github.com/donislawdev/qadatagenerator/internal/produce"
)

// Write produces the dataset a recipe describes and says what it left behind.
//
// It takes the bytes on screen rather than anything this screen worked out from
// them, so that the run reads the recipe the same way the check did and the
// window holds no half decoded state between the two.
type Write func(recipe []byte, into string, replace bool,
	stop generate.Halt) (produce.Written, error)

// bar is the strip along the bottom: where to write, whether to replace, and
// the button.
//
// # Why along the bottom rather than inside the answer
//
// The answer above it grows with the recipe. A table of a wide entity, eleven
// problems, two lists of what will not happen: all of them push whatever is
// under them further down, and the one thing a person came to press would be
// the thing furthest away. The screen next door put its settings in the
// scrolling half and paid for it once already, when the longest paragraph of a
// file description pushed four settings out of the window.
//
// It is an edge of the layout rather than a third pane, because the divider of
// a pane cannot be moved from the keyboard and every one of them is a debt.
func (v *View) bar() fyne.CanvasObject {
	where := container.NewBorder(nil, nil,
		widget.NewLabel(v.text.T(i18n.RecipeWriteTo)), v.choose, v.out)
	buttons := container.NewBorder(nil, nil, v.replace, v.run, widget.NewLabel(""))
	return container.NewVBox(where, buttons, v.status)
}

// buildBar makes the controls once.
//
// They are built here rather than in Content for the reason the answer pane is:
// each of them can hold the keyboard, and a second call of Content would
// replace a control somebody is standing on. The screen next door carries
// B-203 for exactly that.
func (v *View) buildBar() {
	v.out = widget.NewEntry()
	// 🔴 A short hint rather than the sentence under the button. Rendering
	// showed the two side by side saying the same thing twice, and a
	// placeholder is meant to be an example of what goes in the field rather
	// than a second copy of the instruction beside it.
	v.out.SetPlaceHolder(v.text.T(i18n.RecipeDirHint))
	v.out.OnChanged = func(string) { v.drawBar() }
	v.choose = parts.NewButton(v.text.T(i18n.RecipeChooseDir), v.chooseDirectory)
	v.replace = widget.NewCheck(v.text.T(i18n.RecipeReplace), func(bool) {})
	v.run = parts.NewButton(v.text.T(i18n.RecipeGenerate), v.Start)
	v.status = container.NewVBox()
}

// chooseDirectory asks for a directory the way the system asks for one.
//
// A field alone would be enough for somebody who knows the path they want, and
// is a poor answer for the person this screen was built for. The field stays,
// because a path pasted from somewhere else is the fastest way in for the other
// half of the people who use this.
func (v *View) chooseDirectory() {
	if v.window == nil || v.running() {
		return
	}
	dialog.ShowFolderOpen(func(chosen fyne.ListableURI, err error) {
		if err != nil || chosen == nil {
			return
		}
		v.out.SetText(chosen.Path())
	}, v.window)
}

// Start writes the dataset, off the thread that draws.
//
// # Why the reasons are checked here as well as at the button
//
// The button is disabled whenever a run would be refused, so none of this can
// be reached by a person. It is checked anyway, so that the sentence "this
// screen never starts a run it said it would refuse" is true of the screen
// rather than of one control on it. The screen next door says the same about
// its own button and for the same reason.
func (v *View) Start() {
	if !v.canRun() {
		return
	}
	recipe := []byte(v.editor.Text)
	into, replace := v.out.Text, v.replace.Checked
	halt := background.NewStopper()
	v.halt = halt
	v.stopping = false
	v.freeze()
	v.drawBar()
	// Before the run is handed over rather than after it, because a place of
	// work that answers straight away carries the run out inside this call.
	v.interceptClose()
	v.watchForClosing()
	v.work.Produce(func() (produce.Written, error) {
		return v.write(recipe, into, replace, halt)
	}, v.finished)
}

// Stop asks a run in flight to end, which is what the button becomes.
//
// A run stops between rows and nowhere else, so this is an ask rather than an
// order, and the screen says so while it waits.
func (v *View) Stop() {
	if !v.running() || v.halt == nil {
		return
	}
	v.halt.Stop()
	v.stopping = true
	v.drawBar()
}

// finished is what a run answers with, back on the thread that draws.
func (v *View) finished(written produce.Written, err error) {
	v.halt = nil
	v.stopping = false
	v.written, v.failed = written, err
	v.ran = true
	v.releaseClose()
	v.releaseWatchForClosing()
	v.thaw()
	v.drawBar()
	if v.closing {
		// Somebody asked to leave while this was writing, and the run has now
		// ended, so there is nothing half written left to keep them here.
		v.leave()
	}
}

// running reports whether a run is going on.
func (v *View) running() bool { return v.halt != nil }

// canRun reports whether pressing the button now would produce anything.
func (v *View) canRun() bool {
	return !v.running() && v.checked && v.answer.Report.Usable() &&
		v.answer.Refused == nil && v.out.Text != ""
}

// freeze and thaw put the settings out of reach for the length of a run.
//
// Everything the run reads was taken by value before it started, so this is
// not what makes the run safe. It is what stops a person from believing they
// have changed something that is about to be written.
func (v *View) freeze() { v.setControls(false) }
func (v *View) thaw()   { v.setControls(true) }

func (v *View) setControls(usable bool) {
	for _, control := range []fyne.Disableable{v.out, v.choose, v.replace} {
		if usable {
			control.Enable()
			continue
		}
		control.Disable()
	}
}

// drawBar puts the button in the state it should be in and says why.
//
// 🔴 A disabled button carries the reason beside it rather than only being
// grey. Point seven of the way this project writes for people: a control that
// can do nothing here has to say what to do instead, and "disabled" is not that
// sentence.
func (v *View) drawBar() {
	v.status.Objects = nil
	if v.running() {
		v.run.SetText(v.text.T(i18n.RecipeStopRun))
		v.run.OnTapped = v.Stop
		v.run.Enable()
		v.status.Add(sentence(v.text.T(v.runningWords())))
		v.status.Refresh()
		return
	}

	v.run.SetText(v.text.T(i18n.RecipeGenerate))
	v.run.OnTapped = v.Start
	if v.canRun() {
		v.run.Enable()
	} else {
		v.run.Disable()
	}
	v.status.Add(sentence(v.said()))
	v.status.Refresh()
}

func (v *View) runningWords() i18n.Key {
	if v.stopping {
		return i18n.RecipeStopping
	}
	return i18n.RecipeWriting
}

// said is the one sentence under the button.
//
// The order is the order somebody meets the reasons in: nothing written yet is
// the last of them, because it is the only one that is not an instruction.
func (v *View) said() string {
	switch {
	case v.failed != nil:
		return v.failed.Error()
	case v.ran:
		return v.wrote()
	case !v.checked:
		return v.text.T(i18n.RecipeNeedRecipe)
	case !v.answer.Report.Usable() || v.answer.Refused != nil:
		return v.text.T(i18n.RecipeNeedFix)
	case v.out.Text == "":
		return v.text.T(i18n.RecipeNeedDirectory)
	default:
		return v.wouldWrite()
	}
}

// wouldWrite names the files a run would make, which is the answer a dry run
// gives at the other surface.
//
// The names come from the sample rather than from the recipe, because the
// sample got them by watching the writer ask for them. A screen working them
// out from the entities would be right about the comma separated form and
// wrong about the one that writes a single file.
func (v *View) wouldWrite() string {
	if len(v.answer.Sample.Files) == 0 {
		return v.text.T(i18n.RecipeNothingYet)
	}
	names := make([]string, 0, len(v.answer.Sample.Files))
	for _, file := range v.answer.Sample.Files {
		names = append(names, file.Name)
	}
	return v.text.T(i18n.RecipeWouldWrite) + " " + join(names) + ". " +
		v.text.T(i18n.RecipeNothingYet)
}

// wrote says what the run left and where, and the where is the path as it was
// resolved rather than as it was typed.
func (v *View) wrote() string {
	return v.text.T(i18n.RecipeWrote) + " " +
		counted(len(v.written.Files), v.text.T(i18n.RecipeFile), v.text.T(i18n.RecipeFiles)) +
		" " + v.text.T(i18n.RecipeIn) + " " + v.out.Text + ", " +
		counted(v.written.Rows, v.text.T(i18n.RecipeRow), v.text.T(i18n.RecipeRows)) + "."
}

// join writes a list of names the way a person reads one.
func join(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	}
	all := ""
	for at, name := range names[:len(names)-1] {
		if at > 0 {
			all += ", "
		}
		all += name
	}
	return all + " and " + names[len(names)-1]
}

// SetOut puts a directory in the field, without anybody typing it.
//
// Exported for the reason SetRecipe is: the guard holding what every screen
// draws and the probe that photographs each one both have to reach the state
// where the button can be pressed, and reaching it through the field a person
// types into is what keeps those pictures pictures of a state somebody can get
// to.
func (v *View) SetOut(into string) { v.out.SetText(into) }

// Written is what the last run left, for a caller with nobody looking at the
// screen.
func (v *View) Written() (produce.Written, error, bool) {
	return v.written, v.failed, v.ran
}
