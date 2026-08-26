package takeview

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	"github.com/donislawdev/qadatagenerator/internal/take"
)

// perRowDefault is the setting a run has when nobody names one.
//
// The mode reads an empty setting as one defect per record, and this screen
// says that out loud instead of offering a blank choice. A list whose first
// entry is empty asks a person to work out what empty means, and the answer
// here is a real option with a name of its own.
const perRowDefault = "exactly_one"

// buildControls makes every setting a control, once.
//
// Once rather than on each draw, because a control rebuilt under somebody is a
// control that loses what they typed and where their keyboard was standing.
func (v *View) buildControls() {
	v.chosen.PerRow = perRowDefault
	v.layers = widget.NewCheckGroup(take.LayersIn(v.described), func(chosen []string) {
		v.chosen.Layers = chosen
		v.settingChanged()
	})
	v.layers.Horizontal = true

	v.share = v.entry(v.chosen.Rows, func(typed string) { v.chosen.Rows = typed })
	v.exclude = v.entry("", func(typed string) { v.chosen.ExcludeFields = take.NamesIn(typed) })
	v.locale = v.entry(v.chosen.Locale, func(typed string) { v.chosen.Locale = typed })
	v.seed = v.entry(v.chosen.Seed, func(typed string) { v.chosen.Seed = typed })
	v.out = v.entry(v.chosen.Out, func(typed string) { v.chosen.Out = typed })

	// The separator is the one setting that decides how the file is read, so
	// changing it reads the file again. Everything else only changes what a run
	// would do with what has already been read.
	v.delimiter = v.entry(v.chosen.Delimiter, func(typed string) { v.chosen.Delimiter = typed })
	v.delimiter.OnChanged = func(typed string) {
		v.chosen.Delimiter = typed
		v.reopen()
	}

	v.perRow = widget.NewSelect(take.PerRowChoices(), func(picked string) {
		v.chosen.PerRow = picked
		v.settingChanged()
	})
	v.perRow.SetSelected(perRowDefault)
	v.manifest = widget.NewSelect(v.manifestChoices(), func(picked string) {
		v.chosen.Manifest = v.formNamed(picked)
		v.settingChanged()
	})
	v.manifest.SetSelected(v.text.T(i18n.TakeSettingNone))
	// Replacing what is already there is off until somebody says otherwise, and
	// that is the same answer the command line gives. A run that quietly wrote
	// over a file in somebody's working directory is the failure the writer of
	// files exists to prevent, and a window is where an accidental second press
	// is easiest.
	v.overwrite = widget.NewCheck("", func(ticked bool) {
		v.chosen.Overwrite = ticked
		v.settingChanged()
	})

	v.chooseFile = parts.NewButton(v.text.T(i18n.TakeChoose), v.choose)
	v.damage = parts.NewButton(v.text.T(i18n.TakeRun), v.Start)
	v.stop = parts.NewButton(v.text.T(i18n.TakeStop), v.Stop)
	v.settingsBox.Objects = []fyne.CanvasObject{
		heading(v.text.T(i18n.TakeHeadingSettings)), v.form(),
	}
}

// ChooseLayers ticks kinds of damage, the way a person clicking them would.
//
// # Why this is exported and why it goes through the control
//
// Something outside this package has to reach the state where a run is possible
// without anybody clicking: the guard holding what every screen draws, and the
// probe rendering them. Setting the settings directly would reach a state the
// controls cannot produce, so this ticks the boxes and lets the same callback
// run that a click runs.
func (v *View) ChooseLayers(kinds []string) {
	v.layers.SetSelected(kinds)
}

// entry is a text setting that answers as it is typed into.
func (v *View) entry(start string, keep func(string)) *widget.Entry {
	field := widget.NewEntry()
	field.SetText(start)
	field.OnChanged = func(typed string) {
		keep(typed)
		v.settingChanged()
	}
	return field
}

// form lays the settings out as labelled rows.
//
// The toolkit's own form is used rather than a grid of our own, so the labels
// line up the way every other program on this desktop lines them up.
func (v *View) form() fyne.CanvasObject {
	return widget.NewForm(
		widget.NewFormItem(v.text.T(i18n.TakeSettingLayers), v.layers),
		widget.NewFormItem(v.text.T(i18n.TakeSettingRows), v.share),
		widget.NewFormItem(v.text.T(i18n.TakeSettingPerRow), v.perRow),
		widget.NewFormItem(v.text.T(i18n.TakeSettingExclude), v.exclude),
		widget.NewFormItem(v.text.T(i18n.TakeSettingDelimiter), v.delimiter),
		widget.NewFormItem(v.text.T(i18n.TakeSettingLocale), v.locale),
		widget.NewFormItem(v.text.T(i18n.TakeSettingSeed), v.seed),
		widget.NewFormItem(v.text.T(i18n.TakeSettingOut), v.out),
		widget.NewFormItem(v.text.T(i18n.TakeSettingManifest), v.manifest),
		widget.NewFormItem(v.text.T(i18n.TakeSettingOverwrite), v.overwrite),
	)
}

// manifestChoices is every extra form of the manifest, with the absence of one
// named rather than left blank.
func (v *View) manifestChoices() []string {
	return append([]string{v.text.T(i18n.TakeSettingNone)}, take.ManifestForms()...)
}

// formNamed turns what was picked back into what the mode takes, where the
// name for no export is a word on screen and an empty setting underneath.
func (v *View) formNamed(picked string) string {
	if picked == v.text.T(i18n.TakeSettingNone) {
		return ""
	}
	return picked
}

// settingChanged redraws the answer to what a run would do, and the button
// that would do it.
//
// Nothing is read and nothing is written. The whole cost of this is asking the
// mode two questions about values already in memory.
//
// The button goes with the answer rather than being left alone, because the
// two say one thing between them: a setting that would be refused takes the
// answer away and has to take the way of acting on it away as well.
func (v *View) settingChanged() {
	if v.opened == nil {
		return
	}
	v.drawPlan()
	v.drawRun()
}

// reopen reads the file again, because the separator decides how it is read.
//
// # Why this is not simply Open on the same path
//
// 🔴 Open answers about a file somebody chose, and a failure there means there
// is no file, so it lets go of the path. Reading the same file again under a
// new setting is a different question, and a failure means the setting is
// wrong. The two shared one answer until 2026-08-26, and the separator is typed
// one character at a time: deleting the comma before typing the semicolon
// leaves an empty separator for one keystroke, which the reader refuses. The
// screen then forgot the path, and the next keystroke found nothing to reopen,
// so a person lost their open file to a backspace and could not get it back
// without choosing it again.
//
// What the last run did is not forgotten either, because it is about this same
// file and the file has not changed.
func (v *View) reopen() {
	if v.path == "" {
		return
	}
	opened, err := v.openFile(v.path, v.chosen)
	if err != nil {
		v.opened = nil
		v.showUnreadable(err)
		return
	}
	v.opened = opened
	v.redraw()
}

// drawPlan answers what a run would do, or why it would not run at all.
//
// A refusal takes the place of the answer rather than standing beside it,
// because they answer the same question. Its first line names the setting that
// is wrong, so nothing is lost by not putting it against the control.
func (v *View) drawPlan() {
	v.planBox.Objects = []fyne.CanvasObject{
		heading(v.text.T(i18n.TakeHeadingPlan)), v.planContent(),
	}
	v.planBox.Refresh()
}

func (v *View) planContent() fyne.CanvasObject {
	if refused := take.Check(v.chosen, v.described); refused.Refused() {
		return v.refusal(refused)
	}
	if refused := take.CheckOutput(v.path, v.chosen); refused.Refused() {
		return v.refusal(refused)
	}
	planned := v.opened.Plan(v.chosen, v.described)
	return container.NewVBox(
		pair(v.text.T(i18n.TakePlanKinds), counted(planned.Ordered, "kind")),
		pair(v.text.T(i18n.TakePlanImplemented), counted(planned.Implemented, "kind")),
		wrappedPair(v.text.T(i18n.TakePlanWrites),
			listOf(planned.Writes)+" "+v.text.T(i18n.TakePlanWritesTo)+" "+planned.Out),
		wrapped(v.text.T(i18n.TakePlanEveryOrder)),
	)
}

// pair is a label and the value beside it. Values are data rather than
// interface text, so they are not keys.
func pair(label, value string) fyne.CanvasObject {
	name := widget.NewLabelWithStyle(label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	return container.NewHBox(name, widget.NewLabel(value))
}

// listOf writes names the way a sentence does.
func listOf(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	default:
		joined := ""
		for at, name := range names[:len(names)-1] {
			if at > 0 {
				joined += ", "
			}
			joined += name
		}
		return joined + " and " + names[len(names)-1]
	}
}
