package takeview

import (
	"fyne.io/fyne/v2"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/damageparts"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
	"github.com/donislawdev/qadatagenerator/internal/take"
)

// perRowDefault is the setting a run has when nobody names one.
//
// The mode reads an empty setting as one defect per record, and this screen
// says that out loud instead of offering a blank choice. A list whose first
// entry is empty asks a person to work out what empty means, and the answer
// here is a real option with a name of its own.
const perRowDefault = "exactly_one"

// Samples the width of a field is measured from, so that no width is written
// as a number here. Each is the longest value the field is for: a share is at
// most a hundred percent, a separator is one character with room to see it,
// and a locale is a language and a country.
const (
	shareSample     = "100%"
	delimiterSample = ",,"
	localeSample    = "en-US"
)

// buildControls makes every setting a control, once.
//
// Once rather than on each draw, because a control rebuilt under somebody is a
// control that loses what they typed and where their keyboard was standing.
//
// 🔴 Every setting is alive from the moment the screen opens (DEC-245). Until
// 2026-09-15 all of them were dead until a file had been read, with a sentence
// saying so, and nothing was protected by that: the plan needs the file, the
// settings do not, and the command line takes its flags in any order. The one
// thing that waits for the file is the button that damages it.
func (v *View) buildControls() {
	v.chosen.PerRow = perRowDefault
	v.buildCommand()
	v.layers = v.buildKinds()

	v.share = v.entry(v.chosen.Rows, func(typed string) { v.chosen.Rows = typed })
	v.exclude = v.entry("", func(typed string) { v.chosen.ExcludeFields = take.NamesIn(typed) })
	v.locale = v.entry(v.chosen.Locale, func(typed string) { v.chosen.Locale = typed })
	v.seed = v.entry(v.chosen.Seed, func(typed string) { v.chosen.Seed = typed })
	v.buildOut()

	// The separator is the one setting that decides how the file is read, so
	// changing it reads the file again. Everything else only changes what a run
	// would do with what has already been read.
	v.delimiter = v.entry(v.chosen.Delimiter, func(typed string) { v.chosen.Delimiter = typed })
	v.delimiter.OnChanged = func(typed string) {
		v.chosen.Delimiter = typed
		v.reopen()
	}

	v.perRow = parts.NewSelect(v.perRowLabels(), func(picked string) {
		v.chosen.PerRow = v.valueOfPerRowLabel(picked)
		v.settingChanged()
	})
	v.perRow.SetSelected(v.labelOfPerRow(perRowDefault))
	v.manifest = parts.NewSelect(v.manifestChoices(), func(picked string) {
		v.chosen.Manifest = v.formNamed(picked)
		v.settingChanged()
	})
	v.manifest.SetSelected(v.text.T(i18n.TakeSettingNone))
	// Replacing what is already there is off until somebody says otherwise, and
	// that is the same answer the command line gives. A run that quietly wrote
	// over a file in somebody's working directory is the failure the writer of
	// files exists to prevent, and a window is where an accidental second press
	// is easiest.
	v.overwrite = parts.NewCheck(v.text.T(i18n.TakeSettingOverwrite), func(ticked bool) {
		v.chosen.Overwrite = ticked
		v.settingChanged()
	})

	// Built after every control it holds, since the body is laid out once.
	v.options = parts.NewDisclosure(v.text.T(i18n.TakeMoreOptions), v.optionsBody())

	v.chooseFile = parts.NewButton(v.text.T(i18n.TakeChoose), v.choose)
	v.chooseFile.ShowIcon(paint.IconFolder)
	// The same words the recipe screen puts on its button for the same job.
	v.chooseOut = parts.NewButton(v.text.T(i18n.RecipeChooseDir), v.chooseDirectory)
	v.chooseOut.ShowIcon(paint.IconFolder)
	v.damage = parts.NewPrimaryButton(v.text.T(i18n.TakeRun), v.Start)
	v.stop = parts.NewButton(v.text.T(i18n.TakeStop), v.Stop)
	// The button that chooses a file is the one control always in the tree,
	// which is what the keyboard needs to find the canvas through and to fall
	// back onto.
	v.keys = parts.Handover{Anchor: v.chooseFile}
}

// buildOut makes the field naming the folder the damaged copy goes to.
//
// 🔴 Empty rather than the "." of the command line, with a hint saying what
// empty means (B-302). The dot is where the person typing stands, and a window
// has no such place: started from a file manager it is the folder of the
// program, and the copy landed there while the sentence at the top of this
// screen promised it beside the file. Empty is the folder of the file, filled
// in by asked at every question rather than typed into the field, so choosing
// another file moves it and a folder somebody named stays named.
func (v *View) buildOut() {
	v.chosen.Out = ""
	v.out = v.entry(v.chosen.Out, func(typed string) { v.chosen.Out = typed })
	v.out.SetPlaceHolder(v.text.T(i18n.TakeSettingOutHint))
}

// buildKinds makes one check box per kind of damage this build describes, in
// the order the catalogue gives them, each telling the same callback.
//
// Ticking one changes the whole set of chosen kinds rather than adding a code,
// so that the order of the codes is the catalogue's and not the order somebody
// clicked in. The command line and the manifest both list them that way.
func (v *View) buildKinds() kindBoxes {
	var kinds kindBoxes
	for _, code := range v.layerCodes() {
		kinds = append(kinds, kindBox{code: code, box: parts.NewCheck(v.labelOfLayer(code), func(bool) {
			v.chosen.Layers = v.layers.chosen()
			v.settingChanged()
		})})
	}
	return kinds
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
// The kinds arrive as the catalogue names them, because that is what a caller
// outside a window has: a layer code, the same string --layers takes.
func (v *View) ChooseLayers(kinds []string) {
	v.layers.tick(kinds)
}

// entry is a text setting that answers as it is typed into, dressed the way
// every field of this window is.
func (v *View) entry(start string, keep func(string)) *parts.FieldEntry {
	field := parts.NewFieldEntry()
	field.SetText(start)
	field.OnChanged = func(typed string) {
		keep(typed)
		v.settingChanged()
	}
	return field
}

// kindsGrid is the kinds of damage in a grid that wraps to the width it is
// given.
//
// # Why a grid for the kinds
//
// Eight kinds in one column took a quarter of the screen and pushed everything
// under them below the fold, measured on the render of 2026-09-15. A grid
// wraps them into as many columns as the width allows, two at the size the
// owner keeps the window, and the cell is measured from the longest name, so
// a translation that is longer costs a column rather than an overlap.
func (v *View) kindsGrid() fyne.CanvasObject {
	labels := make([]string, 0, len(v.layers))
	for _, kind := range v.layers {
		labels = append(labels, kind.box.Text)
	}
	// The cell is the widest box plus a step of the scale, because the grid
	// puts its cells edge to edge and this window's padding is nought: without
	// the step the longest name touched the box of the next column, seen on
	// the first render.
	cell := fyne.NewSize(paint.CheckWidth(labels...)+palette.StepOf(palette.NameStepRelated),
		palette.ControlHeight)
	return paint.Grid(cell, v.layers.controls()...)
}

// optionsBody is the settings a first run does not need, which the screen
// keeps folded under one heading until somebody opens it (DEC-261).
//
// # Why these and not the others
//
// Each of them has an answer that is right for somebody who has never run this
// before: every record may be damaged, one defect each, seed nought, no column
// spared, a comma between fields, the locale of the window, no extra form of
// the manifest. What a person has to decide is which file, what to break and
// where the copy goes, and those stay in sight. The separator is here as well,
// although it decides how the file is read, because the panel beside the steps
// says when a file looks as if it was read with the wrong one.
func (v *View) optionsBody() fyne.CanvasObject {
	// Every field as wide as the value it takes (DEC-261), the seed and the
	// columns to spare measured from the samples the Quick screen shares, and
	// what is left of each line left empty rather than lent to a field.
	howMuch := paint.Row(palette.NameStepRelated,
		labelled(v.text.T(i18n.TakeSettingRows),
			paint.AtLeast(paint.FieldWidth(shareSample), 0, parts.Standing(parts.Field(v.share)))),
		labelled(v.text.T(i18n.TakeSettingPerRow), v.perRow.OnScreen()),
		labelled(v.text.T(i18n.TakeSettingSeed),
			paint.AtLeast(paint.FieldWidth(paint.SeedSample), 0, parts.Standing(parts.Field(v.seed)))))
	leaveAlone := paint.Lead(labelled(v.text.T(i18n.TakeSettingExclude),
		paint.AtLeast(paint.FieldWidth(paint.NamesSample), 0, parts.Standing(parts.Field(v.exclude)))),
		paint.Spacer())
	reading := paint.Row(palette.NameStepRelated,
		labelled(v.text.T(i18n.TakeSettingDelimiter),
			paint.AtLeast(paint.FieldWidth(delimiterSample), 0, parts.Standing(parts.Field(v.delimiter)))),
		labelled(v.text.T(i18n.TakeSettingLocale),
			paint.AtLeast(paint.FieldWidth(localeSample), 0, parts.Standing(parts.Field(v.locale)))),
		labelled(v.text.T(i18n.TakeSettingManifest), v.manifest.OnScreen()))
	return paint.Pad(palette.NameStepTight, palette.NameStepHairline,
		palette.NameStepHairline, palette.NameStepHairline,
		paint.Column(palette.NameStepGroup, howMuch, leaveAlone, reading))
}

// OpenOptions unfolds the settings a first run does not need, the way a person
// pressing their heading would. Exported for the reason Open is: the guard
// holding what every screen draws and the probe rendering them have to reach
// the state with nobody pressing anything.
func (v *View) OpenOptions() { v.options.Open() }

// manifestChoices is every extra form of the manifest, with the absence of one
// named rather than left blank. The recipe screen offers the same list.
func (v *View) manifestChoices() []string { return damageparts.FormChoices(v.text) }

// formNamed turns what was picked back into what the mode takes.
func (v *View) formNamed(picked string) string { return damageparts.FormNamed(v.text, picked) }

// settingChanged redraws the answer to what a run would do, and the button
// that would do it.
//
// Nothing is read and nothing is written. The whole cost of this is asking the
// mode two questions about values already in memory.
//
// The button goes with the answer rather than being left alone, because the
// two say one thing between them: a setting that would be refused takes the
// answer away and has to take the way of acting on it away as well. The
// command follows too, since it is these settings written as flags.
func (v *View) settingChanged() {
	v.drawCommand()
	if v.opened == nil {
		return
	}
	// 🔴 What the last run did is let go of the moment a setting changes,
	// since 2026-09-15. The card says one thing: what pressing the button now
	// would do, or what the last press did, whichever is newer. A result kept
	// beside settings it was not made with would belong to neither, which is
	// the same reason the settings are frozen for the length of a run. The
	// files of that run are still on the disk and the manifest browser is
	// where their detail lives.
	if !v.busy() {
		v.forgetLastRun()
	}
	v.drawPlan()
	v.drawRun()
}

// asked is the settings as a run takes them: what the controls hold, with the
// folder of the file that was read where nobody named a folder to write to.
//
// 🔴 The plan, the button, the run and the command all ask through this and
// never through chosen. They answer one question between them, and a folder
// filled in for three of them and left empty for the fourth would have the
// screen promise one folder and write into another.
func (v *View) asked() take.Settings { return v.chosen.Beside(v.path) }

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
// What the last run did is let go of, as it is for any other setting since
// 2026-09-15: the separator is a setting of the run as much as of the read,
// and a result kept beside a setting it was not made with belongs to neither.
// Like Open it hands the read over rather than doing it here, and for the
// sharper version of the same reason: this one is asked on every keystroke, so
// reading the file where the key was pressed would mean a window that stops
// answering the keyboard exactly while somebody is typing into it.
func (v *View) reopen() {
	v.drawCommand()
	// 🔴 A file chosen and still being read is the one the newest question is
	// about, so the separator goes to it, and by the road a chosen file takes.
	// Until 2026-10-27 this read v.path alone, which is the file whose reading
	// has ANSWERED. A separator typed while a newly chosen file was read sent
	// the previous file to the reader, the answer about the chosen one was
	// dropped as stale, and the screen ended on a file the person had moved
	// away from. With no file before it the separator never reached the read at
	// all, and the screen described the file under a setting it no longer
	// showed.
	if v.reading != "" && v.reading != v.path {
		v.Open(v.reading)
		return
	}
	if v.path == "" {
		return
	}
	if !v.busy() {
		v.forgetLastRun()
	}
	v.startReading(v.path, func(opened *take.Opened, err error) {
		if err != nil {
			v.opened = nil
			v.showUnreadable(err)
			return
		}
		v.opened = opened
		v.redraw()
	})
}

// drawPlan answers what a run would do, or why it would not run at all.
//
// A refusal takes the place of the answer rather than standing beside it,
// because they answer the same question. Its first line names the setting that
// is wrong, so nothing is lost by not putting it against the control.
func (v *View) drawPlan() {
	v.planBox.Objects = []fyne.CanvasObject{v.planContent()}
	v.planBox.Refresh()
}

func (v *View) planContent() fyne.CanvasObject {
	asked := v.asked()
	if refused := take.Check(asked, v.described); refused.Refused() {
		return v.refusal(refused)
	}
	if refused := take.CheckOutput(v.path, asked); refused.Refused() {
		return v.refusal(refused)
	}
	planned := v.opened.Plan(asked, v.described)
	// The names stand in a column as wide as the longest kind this screen
	// offers rather than the longest ticked, so that ticking one does not move
	// the figures of the others.
	return paint.Column(palette.NameStepRelated,
		damageparts.PlanRows(v.text, planned.Planned, damageparts.NameWidth(v.text, v.layerCodes())),
		paint.Hairline(palette.LineQuiet),
		v.fact(i18n.TakePlanWrites,
			listOf(planned.Writes)+" "+v.text.T(i18n.TakePlanWritesTo)+" "+planned.Out),
		v.paragraph(v.text.T(i18n.TakePlanEveryOrder)))
}

// fact is a caption over a value, the way the panel beside the values on the
// Quick screen states one.
func (v *View) fact(name i18n.Key, value string) fyne.CanvasObject {
	return paint.Column(palette.NameStepHairline,
		paint.Caption(v.text.T(name)),
		v.paragraph(value))
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
