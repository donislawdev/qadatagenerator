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
	"github.com/donislawdev/qadatagenerator/internal/fault"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
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
	if v.readingNow() {
		// The file is being read again under a setting somebody has just
		// changed, so what this screen holds describes a file that is about to
		// be replaced. A run started from it would damage the file it was told
		// about a moment ago and write a manifest saying so. The button is
		// disabled for the length of a read, and this is the half that does not
		// depend on a control staying disabled.
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
	halt := background.NewStopper()
	v.halt = halt
	v.stage = stageRunning
	v.freeze()
	v.drawRun()
	// Before the run is handed over rather than after it, because a place of
	// work that answers straight away carries the run out inside this call and
	// the end of the run has a keyboard of its own to place.
	v.hadKeyboard = v.handKeyboardOver(v.stop, v.damage)
	v.interceptClose()
	v.watchForClosing()
	v.work.Run(func() (take.Result, error) {
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
	v.halt.Stop()
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
	case errors.Is(err, background.ErrStopped):
		v.stage = stageStopped
	default:
		v.stage = stageFailed
	}
	v.thaw()
	v.releaseClose()
	v.releaseWatchForClosing()
	if v.closing {
		// Somebody asked for the window while the run was going on, and the run
		// was stopped rather than the window taken away from underneath it.
		v.closing = false
		v.leave()
		return
	}
	v.drawRun()
	v.giveTheKeyboardBack(v.damage, v.stop)
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

// watchForClosing makes the system asking this program to stop mean what the
// close button means.
//
// # Why a signal is answered the same way as the close button
//
// They are the same ask from different people. A person clicking the cross and
// a container being stopped both want this program to end, and both cost the
// same thing if it ends where it stands: the temporary files of the run and the
// marker holding its name are left in somebody's output directory, and the next
// run finds the marker and refuses for a reason that stopped being true. The
// window already knew how to answer one of them, so the other is routed into
// the same answer rather than given a second one that could drift from it.
//
// # Why the watch lives no longer than the run
//
// Outside a run nothing is half written, so a person killing this window gets
// the instant end their desktop would have given them anyway. That is the same
// boundary the command line draws, and drawing it differently here would be the
// parity rule broken in the other direction.
func (v *View) watchForClosing() {
	if v.work.Watch == nil {
		// A caller assembling this screen with nobody looking at it has no
		// process to be asked about. Said out loud rather than left to a nil
		// call, because the states of this screen are built exactly that way.
		return
	}
	v.releaseWatch = v.work.Watch(v.stoppedFromOutside)
}

func (v *View) releaseWatchForClosing() {
	if v.releaseWatch == nil {
		return
	}
	v.releaseWatch()
	v.releaseWatch = nil
}

// stoppedFromOutside is what a signal means once it reaches the thread that
// draws.
//
// 🔴 The run may already have ended by the time this arrives, and that is
// ordinary rather than exotic: the signal is noticed on a goroutine and acted
// on here, and a run that stops in between is a run with nothing left to take
// back. Setting the flag anyway would arm the NEXT run to close the window
// behind whoever started it.
func (v *View) stoppedFromOutside() {
	if !v.busy() {
		v.leave()
		return
	}
	v.closing = true
	v.Stop()
}

// leave gives up the window, which is the answer to being asked to stop once
// there is nothing left to clean up.
func (v *View) leave() {
	if v.window != nil {
		v.window.Close()
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
	for _, control := range append(v.settingControls(), v.chooseFile) {
		if usable {
			control.Enable()
			continue
		}
		control.Disable()
	}
}

// settingControls is every setting of a run, WITHOUT the button that chooses
// the file.
//
// 🔴 The separation exists because the two states that disable these want
// opposite things from that button. A run in flight disables it as well: the
// file is being read from, and choosing another one halfway would damage one
// file and describe another. A screen with no file yet must leave it the one
// live thing on the screen, because it is the only way forward and a screen
// where nothing at all can be pressed reads as broken rather than as waiting.
func (v *View) settingControls() []fyne.Disableable {
	return []fyne.Disableable{
		v.layers, v.share, v.perRow, v.exclude, v.delimiter, v.locale, v.seed,
		v.out, v.manifest, v.overwrite,
	}
}

// previewSettings leaves the settings on screen for somebody who has chosen no
// file, disabled, so that the screen says what it does before it is asked to do
// it.
//
// 🔴 # Why they are shown at all before there is a file
//
// Measured on 2026-09-08 with tools/pustka.py at the size the window is
// actually kept: this screen stood 64.0 per cent empty in the state it OPENS
// in, with a heading, two sentences and one button in the top third and
// nothing under them. The state a program opens in is the one nobody draws and
// the first one a person meets, and what it said about this tool was that the
// tool does almost nothing.
//
// Everything under here already existed and was hidden. Showing it costs no new
// content and answers the question somebody opening this screen actually has,
// which is what this thing is going to do to their file.
//
// # Why disabled rather than live
//
// Every one of them describes a run of a file, and there is no file. A live
// control that silently governs nothing is the worse of the two, and this
// screen already answers a run in flight the same way: disabled beside its
// reason rather than absent, because a control that vanishes leaves a person
// looking for it.
func (v *View) previewSettings() {
	for _, control := range v.settingControls() {
		control.Disable()
	}
	// Live, and it is the only one. Everything else on this screen is waiting
	// for what this button produces.
	v.chooseFile.Enable()
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
	if v.refusedNow() || v.readingNow() {
		// Disabled rather than absent, and the reason is directly above it in
		// the answer to what would happen. A control that vanishes leaves a
		// person looking for it, and one that is disabled beside its reason is
		// the same information without the search.
		//
		// A read in flight disables it for a reason of its own: what stands on
		// the screen describes the file as it was read a moment ago, and a run
		// started from it would damage the file and then say, in a manifest,
		// that it had been told something else.
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
// taste: both of the toolkit's endless indicators animate forever from the moment
// they are shown, and their geometry comes from a clock. One travels a rectangle on
// a fifty millisecond tick and the other turns three circles on a two second loop.
// Either way that geometry is part of the stored tree of every screen, so the guard
// comparing those trees would be red or green depending on when it ran. A guard that
// fails at random is a guard somebody turns off.
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
		container.NewHBox(v.stop.OnScreen()),
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
			wrapped(fault.Says(v.failed)),
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
	}
	// 🔴 Directly under the files, because it is about those files rather than
	// about the damage, and everything below this line counts defects. A
	// manifest this build will not read back is the one thing a finished run can
	// leave behind that shows up nowhere else: the files are written, the counts
	// add up, and the refusal arrives days later on another screen with nothing
	// connecting it to the run that caused it. The command line has said so
	// since it learned to and this screen said nothing at all.
	if len(v.result.Oversized) > 0 {
		lines = append(lines,
			pair(v.text.T(i18n.TakeResultOversized), listOf(namesOf(v.result.Oversized))),
			v.notice(i18n.TakeResultOversizedWhy))
	}
	lines = append(lines,
		pair(v.text.T(i18n.TakeResultOrdered), strconv.Itoa(counts.Requested)),
		pair(v.text.T(i18n.TakeResultApplied), strconv.Itoa(counts.Applied)))
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

// namesOf is the files of an oversized list, because the screen names them and
// the sizes belong to the sentence rather than to the list.
func namesOf(big []take.Oversize) []string {
	names := make([]string, 0, len(big))
	for _, one := range big {
		names = append(names, one.Name)
	}
	return names
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
