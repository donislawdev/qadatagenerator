package takeview

import (
	"errors"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/take"
)

// stage is what this screen is doing, and what came of the last thing it did.
//
// # Why one value rather than a pair of flags
//
// docs/17 asks a screen for five states rather than one. Two of them, the empty
// screen and the full one, are answered by whether a file is open. The other
// three are answered here: a run going on, a run that did not finish, and a run
// that wrote its files with part of the ordered damage unapplied. Held as
// separate flags they can be set in combinations that mean nothing, and every
// drawing would then have to decide which flag wins.
type stage int

const (
	// stageReady is nothing run since the file was opened, which is also where
	// a screen with no file at all stands.
	stageReady stage = iota
	// stageRunning is a run going on.
	stageRunning
	// stageStopping is a run that has been asked to end and has not answered
	// yet. It is a state of its own because the answer arrives after the record
	// the run is on, which is a moment a person can see, and a button that
	// stayed pressable would take a second press meaning nothing.
	stageStopping
	// stageWritten is a run that finished, whether or not everything ordered
	// could be applied.
	stageWritten
	// stageStopped is a run somebody ended.
	stageStopped
	// stageFailed is a run that ended on a failure.
	stageFailed
)

// Start damages the file, which is what the button does.
//
// # Why this is exported
//
// The same reason Open and ChooseLayers are. Something outside this package has
// to reach the states of a run without anybody clicking: the guard holding what
// every screen draws, and the probe rendering them to pictures. It carries the
// whole state change rather than the drawing alone, so nothing outside can reach
// a state a person cannot.
//
// # Why a refused run is refused here as well as at the button
//
// The button is disabled while the settings would be refused, so this cannot be
// reached by a person. It is checked anyway, because the check is what makes the
// sentence "this screen never starts a run it said it would refuse" true of the
// screen rather than of one control on it.
func (v *View) Start() {
	if v.opened == nil || v.busy() {
		return
	}
	if take.Check(v.chosen, v.described).Refused() {
		return
	}
	if take.CheckOutput(v.path, v.chosen).Refused() {
		return
	}

	// Everything the run touches is taken now and by value, so that nothing it
	// reads can be changed while it is going on. The controls are frozen for
	// the same reason one level up, and this is the half that does not depend
	// on a control staying disabled.
	opened, settings, described := v.opened, v.chosen, v.described
	halt := newStopper()
	v.halt = halt
	v.stage = stageRunning
	v.freeze()
	v.drawRun()
	v.interceptClose()
	v.run(func() (take.Result, error) {
		return opened.Corrupt(settings, described, halt, time.Now())
	}, v.finished)
}

// Stop asks a run in flight to end, which is what the other button does.
//
// A run stops between records and nowhere else, so this is an ask rather than
// an order and the screen says so while it waits.
func (v *View) Stop() {
	if v.stage != stageRunning || v.halt == nil {
		return
	}
	v.halt.stop()
	v.stage = stageStopping
	v.drawRun()
}

// busy reports whether a run is going on.
func (v *View) busy() bool {
	return v.stage == stageRunning || v.stage == stageStopping
}

// finished is what a run answers with, back on the thread that draws.
//
// Nothing is closed here. The walk over the file belongs to the run and the run
// ends it, including a run that was stopped between two records, which is what
// keeps the next press of the button damaging the whole file rather than the
// part of it that was left. That is a property of the mode rather than of this
// screen, so it lives there and is guarded there.
func (v *View) finished(result take.Result, err error) {
	v.halt = nil
	v.result, v.failed = result, err
	switch {
	case err == nil:
		v.stage = stageWritten
	case errors.Is(err, ErrStopped):
		v.stage = stageStopped
	default:
		v.stage = stageFailed
	}
	v.thaw()
	v.releaseClose()
	if v.closing {
		// Somebody asked for the window while the run was going on, and the run
		// was stopped rather than the window taken away from underneath it.
		v.closing = false
		if v.window != nil {
			v.window.Close()
		}
		return
	}
	v.drawRun()
}

// interceptClose makes the closing of the window stop the run first.
//
// # Why the window is taken over for the length of a run and not for longer
//
// Closing a window ends the process, and a process ending in the middle of a
// run leaves the temporary files of that run and the marker holding its name in
// somebody's output directory, where the next run finds the marker and refuses.
// That is what a killed run costs, and a person pressing the close button is not
// killing anything: they are asking to leave.
//
// So for as long as a run is going on, that ask stops the run and the window
// closes once it has stopped. Outside a run the window closes the way every
// other window does, which is why this is given back rather than left in place.
func (v *View) interceptClose() {
	if v.window == nil {
		return
	}
	v.window.SetCloseIntercept(func() {
		v.closing = true
		v.Stop()
	})
}

func (v *View) releaseClose() {
	if v.window != nil {
		v.window.SetCloseIntercept(nil)
	}
}

// freeze puts every setting out of reach for the length of a run.
//
// A setting changed while a run is going on would describe a run other than the
// one happening, and the answer on screen would then belong to neither. The
// file cannot be changed either, which is why the button that chooses one goes
// with them.
func (v *View) freeze() { v.setControls(false) }

// thaw gives them back.
func (v *View) thaw() { v.setControls(true) }

func (v *View) setControls(usable bool) {
	controls := []fyne.Disableable{
		v.layers, v.share, v.perRow, v.exclude, v.delimiter, v.locale, v.seed,
		v.out, v.manifest, v.overwrite, v.chooseFile,
	}
	for _, control := range controls {
		if usable {
			control.Enable()
			continue
		}
		control.Disable()
	}
}

// drawRun draws the button that starts a run, and whatever the last one left.
func (v *View) drawRun() {
	v.runBox.Objects = v.runContent()
	v.runBox.Refresh()
}

func (v *View) runContent() []fyne.CanvasObject {
	if v.busy() {
		return v.runningContent()
	}
	drawn := []fyne.CanvasObject{container.NewHBox(v.damage)}
	v.damage.Enable()
	if v.refusedNow() {
		// Disabled rather than absent, and the reason is directly above it in
		// the answer to what would happen. A control that vanishes leaves a
		// person looking for it, and one that is disabled beside its reason is
		// the same information without the search.
		v.damage.Disable()
	}
	if outcome := v.outcome(); outcome != nil {
		drawn = append(drawn, heading(v.text.T(i18n.TakeHeadingResult)), outcome)
	}
	return drawn
}

// refusedNow reports whether the settings as they stand would be refused.
func (v *View) refusedNow() bool {
	if v.opened == nil {
		return true
	}
	return take.Check(v.chosen, v.described).Refused() ||
		take.CheckOutput(v.path, v.chosen).Refused()
}

// runningContent is the state docs/17 calls work in progress.
//
// # Why there is nothing here that moves
//
// A bar that travels while a run goes on is what every other program shows, and
// this window does not show one. The reason is measured rather than a matter of
// taste: the toolkit draws both of its endless indicators as a rectangle whose
// position comes from a clock, that position is part of the stored tree of every
// screen, and the guard comparing those trees would then be red or green
// depending on when it ran. A guard that fails at random is a guard somebody
// turns off.
//
// What a person actually needs is here in words: that the file is being damaged,
// that nothing is on the disk yet under its own name, and the way to stop.
func (v *View) runningContent() []fyne.CanvasObject {
	said := i18n.TakeRunning
	v.stop.Enable()
	if v.stage == stageStopping {
		said = i18n.TakeRunStopping
		v.stop.Disable()
	}
	return []fyne.CanvasObject{
		container.NewHBox(v.stop),
		beside(theme.InfoIcon(), wrapped(v.text.T(said))),
	}
}

// outcome is what the last run left behind, or nothing when there has not been
// one.
func (v *View) outcome() fyne.CanvasObject {
	switch v.stage {
	case stageWritten:
		return v.written()
	case stageStopped:
		return container.NewVBox(
			v.told(i18n.TakeResultStopped, theme.InfoIcon()),
			wrapped(v.text.T(i18n.TakeResultStoppedWhy)),
		)
	case stageFailed:
		return container.NewVBox(
			v.told(i18n.TakeResultFailed, theme.WarningIcon()),
			wrapped(v.failed.Error()),
			wrapped(v.text.T(i18n.TakeResultFailedWhy)),
		)
	default:
		return nil
	}
}

// written says what a finished run produced.
//
// # Why every count and not the applied one
//
// A run where most of what was ordered went nowhere looks, from the applied
// count alone, like a run that barely did anything. That is the state docs/17
// calls a partial run, and it is the one this product exists to make visible, so
// it is marked rather than left to whoever compares two numbers.
//
// The counts are written as bare numbers beside their labels rather than as
// phrases of their own, because the label already names what is being counted
// and the number is data rather than a sentence.
func (v *View) written() fyne.CanvasObject {
	counts := v.result.Counts
	lines := []fyne.CanvasObject{
		wrappedPair(v.text.T(i18n.TakeResultWritten),
			listOf(v.result.Written)+" "+v.text.T(i18n.TakeResultIn)+" "+v.result.Out),
		pair(v.text.T(i18n.TakeResultOrdered), strconv.Itoa(counts.Requested)),
		pair(v.text.T(i18n.TakeResultApplied), strconv.Itoa(counts.Applied)),
	}
	if missing := counts.Missing(); missing > 0 {
		lines = append(lines,
			pair(v.text.T(i18n.TakeResultMissing), strconv.Itoa(missing)),
			v.notice(i18n.TakeResultMissingWhy))
	}
	if counts.Unpartnered > 0 {
		lines = append(lines,
			pair(v.text.T(i18n.TakeResultUnpartnered), strconv.Itoa(counts.Unpartnered)),
			wrapped(v.text.T(i18n.TakeResultUnpartneredWhy)))
	}
	return container.NewVBox(lines...)
}

// told is a headline with a shape beside it.
//
// The shape rather than a colour on its own, for the reason measured for this
// palette: the colours that carry meaning differ in hue while barely differing
// in lightness, so somebody who does not separate hues sees one shade of grey.
func (v *View) told(key i18n.Key, mark fyne.Resource) fyne.CanvasObject {
	headline := widget.NewLabel(v.text.T(key))
	headline.TextStyle = fyne.TextStyle{Bold: true}
	headline.Wrapping = fyne.TextWrapWord
	return beside(mark, headline)
}
