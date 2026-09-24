package recipeview

import (
	"errors"
	"strconv"

	"fyne.io/fyne/v2"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/fault"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	"github.com/donislawdev/qadatagenerator/internal/gui/damageparts"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
	"github.com/donislawdev/qadatagenerator/internal/produce"
)

// Write produces the dataset a recipe describes and says what it left behind.
//
// It takes the bytes on screen rather than anything this screen worked out from
// them, so that the run reads the recipe the same way the check did and the
// window holds no half decoded state between the two. The settings are the
// ones that belong to a run rather than to the recipe: where it goes, whether
// it may replace what is there, and the extra form of the manifest.
type Write func(recipe []byte, settings produce.Settings,
	stop generate.Halt) (produce.Written, error)

// stage is what this screen is doing about a run, and what came of the last
// one.
//
// # Why one value rather than a pair of flags
//
// The same answer the take screen gives: a run going on, a run that has been
// asked to end, a run that wrote, one somebody stopped and one that failed are
// five things a person sees differently, and held as separate flags they can be
// set in combinations that mean nothing.
type stage int

const (
	// stageReady is nothing run since the recipe last changed, which is also
	// where a screen with nothing on it stands.
	stageReady stage = iota
	// stageRunning is a run going on.
	stageRunning
	// stageStopping is a run that has been asked to end and has not answered
	// yet. It is a state of its own because the answer arrives after the record
	// the run is on, which is a moment a person can see, and a button that
	// stayed pressable would take a second press meaning nothing.
	stageStopping
	// stageWritten is a run that finished.
	stageWritten
	// stageStopped is a run somebody ended.
	stageStopped
	// stageFailed is a run that ended on a failure.
	stageFailed
)

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
	// Everything the run reads is taken now and by value, so that nothing it
	// reads can be changed while it is going on. The settings are frozen for
	// the same reason one level up, and this is the half that does not depend
	// on a control staying disabled.
	recipe := []byte(v.editor.Text)
	settings := v.runSettings()
	halt := background.NewStopper()
	v.halt = halt
	v.stage = stageRunning
	v.freeze()
	v.draw()
	// Before the run is handed over rather than after it, because a place of
	// work that answers straight away carries the run out inside this call and
	// the end of the run has a keyboard of its own to place.
	v.keys.Begin(v.stop, v.generate)
	v.interceptClose()
	v.watchForClosing()
	v.work.Produce(func() (produce.Written, error) {
		return v.write(recipe, settings, halt)
	}, v.finished)
}

// runSettings is what the band says about the next run, as the run takes it.
func (v *View) runSettings() produce.Settings {
	return produce.Settings{Into: v.out.Text, Replace: v.replace.Checked, Manifest: v.form}
}

// Stop asks a run in flight to end, which is what the other button does.
//
// A run stops between rows and nowhere else, so this is an ask rather than an
// order, and the screen says so while it waits.
func (v *View) Stop() {
	if v.stage != stageRunning || v.halt == nil {
		return
	}
	v.halt.Stop()
	v.stage = stageStopping
	v.draw()
}

// finished is what a run answers with, back on the thread that draws.
func (v *View) finished(written produce.Written, err error) {
	v.halt = nil
	v.written, v.failed = written, err
	switch {
	case err == nil:
		v.stage = stageWritten
	case errors.Is(err, background.ErrStopped):
		v.stage = stageStopped
	default:
		v.stage = stageFailed
	}
	v.thaw()
	v.releaseClose()
	v.releaseWatchForClosing()
	if v.closing {
		// Somebody asked to leave while this was writing, and the run has now
		// ended, so there is nothing half written left to keep them here.
		v.closing = false
		v.leave()
		return
	}
	v.draw()
	v.keys.End(v.generate, v.stop)
}

// busy reports whether a run is going on.
func (v *View) busy() bool {
	return v.stage == stageRunning || v.stage == stageStopping
}

// forgetLastRun clears what this screen says about a run that is now about
// another recipe, or other settings.
func (v *View) forgetLastRun() {
	v.stage, v.written, v.failed = stageReady, produce.Written{}, nil
}

// canRun reports whether pressing the button now would produce anything.
//
// A check in flight refuses for a reason of its own: what stands on the screen
// describes the recipe as it was a keystroke ago, and a run started from it
// would write from bytes nobody has seen judged. The take screen refuses a run
// during a read for the same reason.
func (v *View) canRun() bool {
	return !v.busy() && !v.checking && v.checked && v.answer.Report.Usable() &&
		v.answer.Refused == nil && v.out.Text != ""
}

// freeze and thaw put the settings out of reach for the length of a run.
//
// Everything the run reads was taken by value before it started, so this is
// not what makes the run safe. It is what stops a person from believing they
// have changed something that is about to be written. The field the recipe is
// written in stays alive: what was taken is on its way to the disk, and the
// next run takes what is in the field then.
func (v *View) freeze() { v.setControls(false) }
func (v *View) thaw()   { v.setControls(true) }

func (v *View) setControls(usable bool) {
	for _, control := range []fyne.Disableable{v.out, v.choose, v.replace, v.manifest} {
		if usable {
			control.Enable()
			continue
		}
		control.Disable()
	}
}

// drawAction puts in the band the button that generates or the button that
// stops, never both: a run in flight has one way out and a screen at rest has
// one way forward, and a person looks in one place for either.
//
// 🔴 A disabled button carries its reason in the card beside it rather than
// under itself (DEC-246). Point seven of the way this project writes for
// people still holds - a control that can do nothing here has to say what to
// do instead - and the card is where that is said: the empty screen says what
// to do, a problem is its own reason, and the directory is asked for under the
// files a run would write.
func (v *View) drawAction() {
	v.action.Objects = []fyne.CanvasObject{v.actionNow()}
	v.action.Refresh()
}

// actionNow is the button the band holds in this state, enabled or not.
func (v *View) actionNow() fyne.CanvasObject {
	if v.busy() {
		v.stop.Enable()
		if v.stage == stageStopping {
			v.stop.Disable()
		}
		return v.stop.OnScreen()
	}
	v.generate.Enable()
	if !v.canRun() {
		v.generate.Disable()
	}
	// The primary button goes to the layout itself rather than through
	// OnScreen. OnScreen exists to hand over the container Outlined builds, and
	// this control draws its own fill and its own focus ring, so there is no
	// edge beside it to hand over.
	return v.generate
}

// runningContent is what the card says while a run is going on.
//
// # Why there is nothing here that moves
//
// The same reason the take screen gives: both of the toolkit's endless
// indicators animate forever from the moment they are shown, and their
// geometry comes from a clock, so it would be part of the stored tree of every
// screen and the guard comparing those trees would be red or green depending
// on when it ran. What a person needs is here in words: that the dataset is
// being written, that nothing is on the disk under its own name yet, and the
// way to stop, which stands in the band where the button that started it stood.
func (v *View) runningContent() []fyne.CanvasObject {
	said := i18n.RecipeWriting
	if v.stage == stageStopping {
		said = i18n.RecipeStopping
	}
	return []fyne.CanvasObject{
		paint.Marked(paint.IconInfo, palette.TextAccent, v.paragraph(v.text.T(said)))}
}

// outcome is what the last run left behind.
func (v *View) outcome() []fyne.CanvasObject {
	switch v.stage {
	case stageWritten:
		return v.wrote()
	case stageStopped:
		return []fyne.CanvasObject{
			paint.Marked(paint.IconInfo, palette.TextAccent, v.paragraph(v.text.T(i18n.RecipeRunStopped)))}
	case stageFailed:
		return []fyne.CanvasObject{
			paint.Marked(paint.IconWarn, palette.StateWarning, v.paragraph(fault.Says(v.failed))),
			v.paragraph(v.text.T(i18n.RecipeRunFailed))}
	default:
		return nil
	}
}

// wrote says what the run left and where, and the where is the path as it was
// typed, because that is the name a person will look for it under.
//
// The names come from the run rather than from the sample, for the reason the
// sample gets them from the writer: which files a form produces is the writer's
// decision, and a screen listing them from the recipe would be right about one
// form and wrong about the next.
//
// A run of a recipe with a policy then says what the damage did, in the block
// the take screen shows after its run, because the counts are the same facts
// from the same engine (DEC-252). A run that was asked for an extra form of the
// manifest and wrote none says so, the way the command line does.
func (v *View) wrote() []fyne.CanvasObject {
	lines := []fyne.CanvasObject{
		v.told(paint.IconCheck, palette.StateSuccess, v.text.T(i18n.RecipeResultWritten)),
		v.paragraph(listOf(v.written.Files) + " " + v.text.T(i18n.RecipeIn) + " " + v.out.Text),
		v.fact(i18n.RecipeResultRows, strconv.Itoa(v.written.Rows)),
	}
	if v.written.Damaged {
		lines = append(lines, paint.Column(palette.NameStepTight,
			damageparts.Counts(v.text, v.written.Counts, v.written.Oversized)...))
	} else if v.form != "" {
		lines = append(lines, v.formUnused())
	}
	return lines
}

// listOf writes names the way a sentence does.
func listOf(names []string) string {
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

// Written is what the last run left, for a caller with nobody looking at the
// screen, and whether there has been one since the recipe last changed.
func (v *View) Written() (produce.Written, error, bool) {
	return v.written, v.failed, v.stage == stageWritten || v.stage == stageStopped || v.stage == stageFailed
}
