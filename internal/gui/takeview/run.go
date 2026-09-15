package takeview

import (
	"errors"
	"strconv"
	"time"

	"fyne.io/fyne/v2"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/fault"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
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
// The separation exists because a run in flight disables that button as well,
// for a reason of its own: the file is being read from, and choosing another
// one halfway would damage one file and describe another. The settings are
// alive from the moment the screen opens (DEC-245), so this is the one list
// of things a run puts out of reach and gives back.
func (v *View) settingControls() []fyne.Disableable {
	return []fyne.Disableable{
		v.layers, v.share, v.perRow, v.exclude, v.delimiter, v.locale, v.seed,
		v.out, v.manifest, v.overwrite,
	}
}

// drawRun draws the one action of the screen in the band, and whatever the
// last run left in the card.
//
// The band shows the button that damages or the button that stops, never
// both: a run in flight has one way out and a screen at rest has one way
// forward, and a person looks in one place for either.
func (v *View) drawRun() {
	v.action.Objects = []fyne.CanvasObject{v.actionNow()}
	v.action.Refresh()
	v.runBox.Objects = v.runContent()
	v.runBox.Refresh()
	// The plan stands in the card only while there is nothing newer to say:
	// a run going on, or what the last one did, takes its place, and the
	// heading over the card says which it is.
	heading := i18n.TakeHeadingPlan
	if len(v.runBox.Objects) > 0 {
		v.planBox.Hide()
		if !v.busy() {
			heading = i18n.TakeHeadingResult
		}
	} else {
		v.planBox.Show()
	}
	v.cardHeading.Objects = []fyne.CanvasObject{v.heading(heading)}
	v.cardHeading.Refresh()
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
	v.damage.Enable()
	if v.refusedNow() || v.readingNow() {
		// Disabled rather than absent, and the reason is in the card beside
		// it, in the answer to what would happen. A control that vanishes
		// leaves a person looking for it, and one that is disabled beside its
		// reason is the same information without the search.
		//
		// A read in flight disables it for a reason of its own: what stands on
		// the screen describes the file as it was read a moment ago, and a run
		// started from it would damage the file and then say, in a manifest,
		// that it had been told something else.
		v.damage.Disable()
	}
	return v.damage
}

// runContent is what the card says about a run: that one is going on, or what
// the last one left, or nothing while there has been none.
func (v *View) runContent() []fyne.CanvasObject {
	if v.busy() {
		return v.runningContent()
	}
	outcome := v.outcome()
	if outcome == nil {
		return nil
	}
	return []fyne.CanvasObject{outcome}
}

// refusedNow reports whether the settings as they stand would be refused.
func (v *View) refusedNow() bool {
	if v.opened == nil {
		return true
	}
	return take.Check(v.chosen, v.described).Refused() ||
		take.CheckOutput(v.path, v.chosen).Refused()
}

// runningContent is what the card says while a run is going on.
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
// that nothing is on the disk yet under its own name, and the way to stop, which
// stands in the band where the button that started it stood.
func (v *View) runningContent() []fyne.CanvasObject {
	said := i18n.TakeRunning
	if v.stage == stageStopping {
		said = i18n.TakeRunStopping
	}
	return []fyne.CanvasObject{
		v.marked(paint.IconInfo, palette.TextAccent, v.paragraph(v.text.T(said)))}
}

// outcome is what the last run left behind, or nothing when there has not been
// one.
func (v *View) outcome() fyne.CanvasObject {
	switch v.stage {
	case stageWritten:
		return v.written()
	case stageStopped:
		return paint.Column(palette.NameStepTight,
			v.told(i18n.TakeResultStopped, paint.IconInfo, palette.TextAccent),
			v.paragraph(v.text.T(i18n.TakeResultStoppedWhy)))
	case stageFailed:
		return paint.Column(palette.NameStepTight,
			v.told(i18n.TakeResultFailed, paint.IconWarn, palette.StateWarning),
			v.paragraph(fault.Says(v.failed)),
			v.paragraph(v.text.T(i18n.TakeResultFailedWhy)))
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
		v.fact(i18n.TakeResultWritten,
			listOf(v.result.Written)+" "+v.text.T(i18n.TakeResultIn)+" "+v.result.Out),
	}
	// 🔴 A file this build will not read back is said HERE, before the counts,
	// because it changes what the counts are worth: a manifest nobody can open
	// on the next screen is a manifest that has to be read some other way. The
	// fact comes from the run and the words are this surface's, which is the
	// same split as every refusal on this screen.
	if len(v.result.Oversized) > 0 {
		lines = append(lines,
			v.pair(i18n.TakeResultOversized, listOf(namesOf(v.result.Oversized))),
			v.notice(i18n.TakeResultOversizedWhy))
	}
	lines = append(lines,
		v.pair(i18n.TakeResultOrdered, strconv.Itoa(counts.Requested)),
		v.pair(i18n.TakeResultApplied, strconv.Itoa(counts.Applied)))
	if missing := counts.Missing(); missing > 0 {
		lines = append(lines,
			v.pair(i18n.TakeResultMissing, strconv.Itoa(missing)),
			v.notice(i18n.TakeResultMissingWhy))
	}
	if counts.Unpartnered > 0 {
		lines = append(lines,
			v.pair(i18n.TakeResultUnpartnered, strconv.Itoa(counts.Unpartnered)),
			v.paragraph(v.text.T(i18n.TakeResultUnpartneredWhy)))
	}
	return paint.Column(palette.NameStepTight, lines...)
}

// pair is a label and the value beside it. Values are data rather than
// interface text, so they are not keys.
func (v *View) pair(label i18n.Key, value string) fyne.CanvasObject {
	return paint.Row(palette.NameStepTight,
		paint.Strong(v.text.T(label), palette.TypeBody, palette.TextPrimary),
		paint.Text(value, palette.TypeBody, palette.TextSecondary))
}

// namesOf is the names of the files that are too large to read back, for the
// sentence that lists them.
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
func (v *View) told(key i18n.Key, icon, ink string) fyne.CanvasObject {
	return v.marked(icon, ink, paint.Strong(v.text.T(key), palette.TypeBody, palette.TextPrimary))
}
