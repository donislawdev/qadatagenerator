package takeview

import (
	"strconv"

	"fyne.io/fyne/v2"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
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
	v.out = v.entry(v.chosen.Out, func(typed string) { v.chosen.Out = typed })

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

	v.chooseFile = parts.NewButton(v.text.T(i18n.TakeChoose), v.choose)
	v.chooseFile.ShowIcon(paint.IconFolder)
	v.damage = parts.NewPrimaryButton(v.text.T(i18n.TakeRun), v.Start)
	v.stop = parts.NewButton(v.text.T(i18n.TakeStop), v.Stop)
	// The button that chooses a file is the one control always in the tree,
	// which is what the keyboard needs to find the canvas through and to fall
	// back onto.
	v.keys = parts.Handover{Anchor: v.chooseFile}
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

// settingsColumn lays the settings out under a heading of their own: the
// kinds in a grid that wraps to the width it is given, then the settings that
// decide how much is damaged, then the ones that decide where it goes.
//
// # Why a grid for the kinds
//
// Eight kinds in one column took a quarter of the screen and pushed everything
// under them below the fold, measured on the render of 2026-09-15. A grid
// wraps them into as many columns as the width allows, two at the size the
// owner keeps the window, and the cell is measured from the longest name, so
// a translation that is longer costs a column rather than an overlap.
func (v *View) settingsColumn() fyne.CanvasObject {
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
	kinds := paint.Column(palette.NameStepHairline,
		paint.Caption(v.text.T(i18n.TakeSettingLayers)),
		paint.Grid(cell, v.layers.controls()...))

	// The seed takes what the two fields before it leave, and the directory
	// what the chooser after it leaves: a seed and a path are the two values
	// nobody can retype from memory, so they get the room.
	howMuch := paint.Lead(
		paint.Row(palette.NameStepRelated,
			labelled(v.text.T(i18n.TakeSettingRows),
				paint.AtLeast(paint.FieldWidth(shareSample), 0, parts.Standing(parts.Field(v.share)))),
			labelled(v.text.T(i18n.TakeSettingPerRow), v.perRow.OnScreen())),
		paint.PadLeft(palette.NameStepRelated,
			labelled(v.text.T(i18n.TakeSettingSeed), parts.Standing(parts.Field(v.seed)))))
	leaveAlone := labelled(v.text.T(i18n.TakeSettingExclude), parts.Standing(parts.Field(v.exclude)))
	where := paint.Trail(
		labelled(v.text.T(i18n.TakeSettingOut), parts.Standing(parts.Field(v.out))),
		paint.PadLeft(palette.NameStepRelated,
			labelled(v.text.T(i18n.TakeSettingManifest), v.manifest.OnScreen())))
	replace := paint.AtLeast(0, palette.ControlHeight, v.overwrite.OnScreen())

	return paint.Column(palette.NameStepGroup, kinds, howMuch, leaveAlone, where, replace)
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
	if refused := take.Check(v.chosen, v.described); refused.Refused() {
		return v.refusal(refused)
	}
	if refused := take.CheckOutput(v.path, v.chosen); refused.Refused() {
		return v.refusal(refused)
	}
	planned := v.opened.Plan(v.chosen, v.described)
	// The names stand in a column as wide as the longest of them, so that the
	// figures line up and can be compared down the card.
	names := paint.OptionNameWidth(v.layerLabels()...)
	rows := []fyne.CanvasObject{paint.Caption(v.text.T(i18n.TakePlanCarried))}
	for _, layer := range planned.ByLayer {
		rows = append(rows, v.layerRow(layer, names))
	}
	return paint.Column(palette.NameStepRelated,
		paint.Column(palette.NameStepTight, rows...),
		paint.Hairline(palette.LineQuiet),
		v.fact(i18n.TakePlanWrites,
			listOf(planned.Writes)+" "+v.text.T(i18n.TakePlanWritesTo)+" "+planned.Out),
		v.paragraph(v.text.T(i18n.TakePlanEveryOrder)))
}

// layerRow is one named kind of damage with how much of it this build carries
// out: the name where the tick box has it, and "3 of 9" in figures, because a
// figure is what somebody compares across rows.
func (v *View) layerRow(layer take.LayerPlan, names float32) fyne.CanvasObject {
	return paint.Row(palette.NameStepRelated,
		paint.AtLeast(names, 0, paint.Strong(v.labelOfLayer(layer.Layer), palette.TypeBody, palette.TextPrimary)),
		paint.Mono(strconv.Itoa(layer.Implemented)+" "+v.text.T(i18n.TakePlanOf)+" "+
			strconv.Itoa(layer.Ordered), palette.TypeBody, palette.TextPrimary))
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
