package takeview

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
	"github.com/donislawdev/qadatagenerator/internal/commandline"
	"github.com/donislawdev/qadatagenerator/internal/damage"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	"github.com/donislawdev/qadatagenerator/internal/take"
)

// The kinds of damage every test here orders, and the column none of them may
// touch.
const (
	orderedLayers = "L2"
	otherLayer    = "L3"
	leftAlone     = "note"
	separator     = ";" // punctuation: test data
)

// held is a place for a run to happen that does not happen anywhere.
//
// It keeps what the screen handed over instead of carrying it out, which is
// what lets a test stand in the middle of a run, and then carry the run out in
// its own time, on its own goroutine, in the order it chooses. Everything it
// runs is the real thing: the same closure the window hands to the same engine.
type held struct {
	work func() (take.Result, error)
	done func(take.Result, error)
}

func (h *held) background(work func() (take.Result, error), done func(take.Result, error)) {
	h.work, h.done = work, done
}

// carryOut does the run the screen asked for and gives the screen its answer,
// which is what the real background does with a goroutine in between.
func (h *held) carryOut(t *testing.T) {
	t.Helper()
	if h.work == nil {
		t.Fatal("the screen never handed a run over, so there is nothing to carry out")
	}
	h.done(h.work())
}

// screenOn builds the screen with a run that goes nowhere until a test says so.
func screenOn(t *testing.T, window fyne.Window) (*View, *held) {
	t.Helper()
	described, err := catalogyaml.Load()
	if err != nil {
		t.Fatalf("the catalog inside this binary could not be read: %v", err)
	}
	runs := &held{}
	// The read happens where it is asked for, so a test reads a real file and
	// finds the screen finished by the time the call returns. Where a read
	// happens is its own port, and the tests for that one hold it up on purpose.
	work := background.Work{Run: runs.background, Read: background.ReadHere}
	if window != nil {
		// The closing of the window is the one the program runs, so that a
		// test asking what the cross does during a run asks that code rather
		// than a stand-in for it.
		work.Hold = background.NewClosing(window, nil).Hold
	}
	view := New(i18n.English, window, described, take.Open, work, nil)
	// The tree is built the way the shell builds it, so that everything a test
	// reads is what a person would be looking at.
	view.Content()
	return view, runs
}

// sourceFile writes a file this tool did not make.
//
// Its fields are separated by semicolons, which is what a European spreadsheet
// produces and what makes the setting for the separator worth having. Reading it
// with the default separator gives one column holding whole lines, so a test
// that changes the setting can see that the change reached the reader.
func sourceFile(t *testing.T, dir string) string {
	t.Helper()
	lines := []string{strings.Join([]string{"id", "town", leftAlone}, separator)}
	for row := 1; row <= 6; row++ {
		lines = append(lines, strings.Join([]string{
			strconv.Itoa(row), "town " + strconv.Itoa(row), "a note",
		}, separator))
	}
	path := filepath.Join(dir, "contacts.csv")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatalf("cannot write the file the screen reads: %v", err)
	}
	return path
}

// opened is a screen with that file on it, read with the separator it uses,
// writing into the directory named.
func opened(t *testing.T, window fyne.Window, source, out string) (*View, *held) {
	t.Helper()
	view, runs := ready(t, window, source)
	view.out.SetText(out)
	return view, runs
}

// ready is a screen with that file on it, read with the separator it uses and
// with two kinds of damage ticked, and with nothing typed where the copy goes.
func ready(t *testing.T, window fyne.Window, source string) (*View, *held) {
	t.Helper()
	view, runs := screenOn(t, window)
	view.Open(source)
	if err := view.Unread(); err != nil {
		t.Fatalf("the screen could not read the file: %v", err)
	}
	view.delimiter.SetText(separator)
	view.ChooseLayers([]string{orderedLayers, otherLayer})
	return view, runs
}

// texts walks what is on screen and collects every string a person would read.
func texts(object fyne.CanvasObject) []string {
	switch typed := object.(type) {
	case *widget.Label:
		return []string{typed.Text}
	// A piece of canvas text is what the figures, the names and the headlines
	// of this screen are drawn as since it began drawing with paint, and it is
	// what a person reads as much as a label is.
	case *canvas.Text:
		return []string{typed.Text}
	case *parts.Button:
		return []string{typed.Text}
	case *widget.Button:
		return []string{typed.Text}
	case *container.Scroll:
		return texts(typed.Content)
	case *fyne.Container:
		var found []string
		for _, child := range typed.Objects {
			found = append(found, texts(child)...)
		}
		return found
	}
	return nil
}

func saying(all []string, want string) bool {
	for _, one := range all {
		if strings.Contains(one, want) {
			return true
		}
	}
	return false
}

// manifestIn reads the manifest a run left in a directory.
func manifestIn(t *testing.T, out string) manifest.Manifest {
	t.Helper()
	record, err := manifestjson.Load(filepath.Join(out, manifestjson.Name))
	if err != nil {
		t.Fatalf("the run left no manifest that this build can read: %v", err)
	}
	return record
}

// The button damages the file and the screen says what it left behind.
//
// 🔴 This is the one test on this screen that walks the whole road: a real file,
// the real engine and real bytes on a real disk. The stored trees of every state
// beside it are drawings of an outcome handed to the screen, which answers
// whether the screen DRAWS an outcome and never whether it can produce one.
func TestTheScreenDamagesTheFileAndSaysWhatItWrote(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "written")
	view, runs := opened(t, nil, sourceFile(t, dir), out)

	view.Start()
	if view.stage != stageRunning {
		t.Fatalf("pressing the button left the screen in stage %d rather than in a run", view.stage)
	}
	runs.carryOut(t)

	if view.stage != stageWritten {
		t.Fatalf("a run that finished left the screen in stage %d: %v", view.stage, view.failed)
	}
	damaged := take.OutputName(view.path)
	for _, name := range []string{damaged, manifestjson.Name} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Errorf("the run says it finished and %s is not there: %v", name, err)
		}
	}
	said := texts(view.runBox)
	for _, want := range []string{damaged, manifestjson.Name, out} {
		if !saying(said, want) {
			t.Errorf("the screen does not say %q about the run it just made. It said %q",
				want, said)
		}
	}
}

// With nothing typed where the copy goes, it goes beside the file it was made
// from, and the screen, the result and the command all name that directory
// (B-302).
//
// 🔴 The test first moves into a directory of its own, because that is the
// failure: until 2026-11-11 an empty choice meant the directory the program
// was started from, which for a window is wherever it happened to be launched,
// while the sentence at the top of the screen promised the copy beside the
// file. The two directories are different here on purpose, and nothing may
// land in the first.
func TestTheCopyIsWrittenBesideTheFileWhenNoDirectoryIsNamed(t *testing.T) {
	started := t.TempDir()
	t.Chdir(started)
	dir := t.TempDir()
	source := sourceFile(t, dir)
	view, runs := ready(t, nil, source)

	if view.out.Text != "" {
		t.Errorf("the field for the directory holds %q before anybody typed into it, "+
			"so its hint about the folder of the file is never seen", view.out.Text)
	}
	if !saying(texts(view.planBox), dir) {
		t.Errorf("the plan does not name %s, the folder of the file. It said %q", dir, texts(view.planBox))
	}
	// Named in the command although a terminal has a directory of its own: the
	// line has to write where this screen does, from wherever it is pasted. The
	// flag is asked about rather than the directory alone, because the file at
	// the end of the line names that directory as well, and a check for the
	// directory alone passed on the code it was written to catch.
	line := view.commandLine()
	if !strings.Contains(line, commandline.Out+" "+dir) && !strings.Contains(line, commandline.Out+" '"+dir) {
		t.Errorf("the command does not write into %s, so pasted elsewhere it writes elsewhere: %s",
			dir, line)
	}

	view.Start()
	runs.carryOut(t)
	if view.stage != stageWritten {
		t.Fatalf("the run did not finish: %v", view.failed)
	}
	if _, err := os.Stat(filepath.Join(dir, take.OutputName(source))); err != nil {
		t.Errorf("the damaged copy is not beside the file it was made from: %v", err)
	}
	left, err := os.ReadDir(started)
	if err != nil {
		t.Fatalf("cannot look at the directory the test started in: %v", err)
	}
	for _, entry := range left {
		t.Errorf("the run wrote %s into the directory the program was started from", entry.Name())
	}
	if !saying(texts(view.runBox), dir) {
		t.Errorf("the result does not name %s. It said %q", dir, texts(view.runBox))
	}
}

// A directory typed without its root is a folder beside the file, and the
// screen names where it is rather than repeating the word that was typed
// (B-207). Until 2026-11-11 it counted from where the program was started, and
// the plan said "in written" about a directory nobody could find.
func TestADirectoryTypedWithoutItsRootIsBesideTheFile(t *testing.T) {
	started := t.TempDir()
	t.Chdir(started)
	dir := t.TempDir()
	source := sourceFile(t, dir)
	view, runs := opened(t, nil, source, "written")
	whole := filepath.Join(dir, "written")

	if !saying(texts(view.planBox), whole) {
		t.Errorf("the plan does not name %s. It said %q", whole, texts(view.planBox))
	}
	view.Start()
	runs.carryOut(t)
	if view.stage != stageWritten {
		t.Fatalf("the run did not finish: %v", view.failed)
	}
	if _, err := os.Stat(filepath.Join(whole, take.OutputName(source))); err != nil {
		t.Errorf("the damaged copy is not in the folder typed beside the file: %v", err)
	}
	if left, err := os.ReadDir(started); err != nil || len(left) > 0 {
		t.Errorf("the run wrote %v into the directory the program was started from (%v)", left, err)
	}
}

// Before anybody presses the button the screen says what a run would do and
// writes nothing.
//
// This is what the command line calls a dry run. In a window it is not a switch
// somebody turns on: it is what the screen does until it is told to do the other
// thing, which is why the register of what the window can reach names this test
// for that flag.
func TestTheScreenSaysWhatARunWouldDoBeforeItRunsAnything(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "written")
	view, _ := opened(t, nil, sourceFile(t, dir), out)

	if _, err := os.Stat(out); err == nil {
		t.Error("the screen made the output directory before anybody asked for a run")
	}
	planned := view.opened.Plan(view.chosen, view.described)
	said := texts(view.planBox)
	// A row per named layer, with how much of it this build carries out, and
	// the files a run would write (DEC-245). The sum stood here until
	// 2026-09-15 and said that something was missing without saying what.
	if len(planned.ByLayer) == 0 {
		t.Fatal("the plan names no layer, so this proves nothing about the rows")
	}
	wanted := []string{take.OutputName(view.path)}
	for _, layer := range planned.ByLayer {
		wanted = append(wanted, strconv.Itoa(layer.Implemented)+" of "+strconv.Itoa(layer.Ordered))
	}
	for _, want := range wanted {
		if !saying(said, want) {
			t.Errorf("the screen does not say %q about what a run would do. It said %q", want, said)
		}
	}
}

// Every setting on the screen reaches the run rather than only the control.
//
// # Why this is one test and not eleven
//
// The register of what the window can reach names a test beside every capability
// it claims, and eleven of those capabilities are the eleven settings of one
// verb. They are all proved the same way, by making a real run with values
// nobody would arrive at by default and reading them back out of what the run
// wrote, so eleven copies of that would be eleven copies of one arrangement.
func TestEverySettingOnTheScreenReachesTheRun(t *testing.T) {
	dir := t.TempDir()
	source := sourceFile(t, dir)
	out := filepath.Join(dir, "written")
	view, runs := opened(t, nil, source, out)

	view.seed.SetText("chosen-by-hand")
	view.locale.SetText("pl-PL")
	view.exclude.SetText(leftAlone)
	view.share.SetText("100%")
	view.perRow.SetSelected(perRowDefault)
	view.manifest.SetSelected(take.ManifestForms()[0])

	view.Start()
	runs.carryOut(t)
	if view.stage != stageWritten {
		t.Fatalf("the run did not finish: %v", view.failed)
	}

	record := manifestIn(t, out)
	if record.Run().Seed != "chosen-by-hand" {
		t.Errorf("the seed on the screen did not reach the run: the manifest carries %q",
			record.Run().Seed)
	}
	if record.Run().Locale != "pl-PL" {
		t.Errorf("the data locale on the screen did not reach the run: the manifest carries %q",
			record.Run().Locale)
	}
	if record.Run().Source != source {
		t.Errorf("the manifest names %q as the file this came from", record.Run().Source)
	}
	if columns := len(view.opened.Table().Columns); columns != 3 {
		t.Errorf("the field separator on the screen did not reach the reader: the file was "+
			"read as %d columns rather than 3", columns)
	}
	// The extra form of the manifest is a file of its own, so its presence is
	// what says the list reached the run.
	for _, name := range damage.Files(take.ManifestForms()[0]) {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Errorf("the extra form of the manifest was chosen and %s is not there: %v", name, err)
		}
	}
	perRow := map[int]int{}
	for _, defect := range record.Defects() {
		if defect.Layer != catalog.Layer(orderedLayers) && defect.Layer != catalog.Layer(otherLayer) {
			t.Errorf("the run ordered damage of kind %q, which nobody ticked", defect.Layer)
		}
		if defect.Location.Field == leftAlone {
			t.Errorf("the column named on screen as one to leave alone carries %s", defect.Mutator)
		}
		// Every ordered defect counts here rather than only the applied ones.
		// The setting decides how many are ORDERED for one record, and what a
		// build happens to be able to carry out is a different question.
		perRow[defect.Location.Row]++
	}
	for row, many := range perRow {
		if many > 1 {
			t.Errorf("one defect per record was chosen and record %d carries %d", row, many)
		}
	}
}

// The share of records reaches the run, which is visible only by moving it.
//
// A share nothing was damaged at and a share everything was damaged at order
// different numbers of defects. One run on its own would prove that a number is
// written rather than that this setting decides it.
//
// What is counted is what was ORDERED rather than what reached the data. This
// build carries an implementation of five of the entries its catalog describes,
// so a file of six rows can easily draw six defects and apply none of them, and
// an assertion about the applied ones would be an assertion about which entries
// the draw happened to pick.
func TestTheShareOfRecordsDecidesHowMuchIsDamaged(t *testing.T) {
	dir := t.TempDir()
	source := sourceFile(t, dir)

	none := filepath.Join(dir, "none")
	view, runs := opened(t, nil, source, none)
	view.share.SetText("0%")
	view.Start()
	runs.carryOut(t)
	if view.stage != stageWritten {
		t.Fatalf("the run at no share did not finish: %v", view.failed)
	}
	if ordered := manifestIn(t, none).Requested(); ordered != 0 {
		t.Errorf("a share of no records ordered %d defects", ordered)
	}

	every := filepath.Join(dir, "every")
	view, runs = opened(t, nil, source, every)
	view.share.SetText("100%")
	view.Start()
	runs.carryOut(t)
	if view.stage != stageWritten {
		t.Fatalf("the run at every record did not finish: %v", view.failed)
	}
	if ordered, rows := manifestIn(t, every).Requested(), view.opened.Table().Rows; ordered != rows {
		t.Errorf("a share of every record ordered %d defects for a file of %d records",
			ordered, rows)
	}
}

// Replacing what is already there happens when it is asked for and not before.
func TestFilesAlreadyThereAreOnlyReplacedWhenTheSettingSaysSo(t *testing.T) {
	dir := t.TempDir()
	source := sourceFile(t, dir)
	out := filepath.Join(dir, "written")

	view, runs := opened(t, nil, source, out)
	view.Start()
	runs.carryOut(t)
	if view.stage != stageWritten {
		t.Fatalf("the first run did not finish: %v", view.failed)
	}

	view.Start()
	runs.carryOut(t)
	if view.stage != stageFailed {
		t.Fatalf("a second run over the same directory ended in stage %d rather than refusing",
			view.stage)
	}
	if !saying(texts(view.runBox), view.failed.Error()) {
		t.Error("the screen does not say what stopped the run")
	}

	view.overwrite.SetChecked(true)
	view.Start()
	runs.carryOut(t)
	if view.stage != stageWritten {
		t.Fatalf("the setting that allows replacing was ticked and the run still refused: %v",
			view.failed)
	}
}

// A run somebody stops leaves nothing behind and the screen says so.
func TestARunSomebodyStopsLeavesNothingBehind(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "written")
	view, runs := opened(t, nil, sourceFile(t, dir), out)

	view.Start()
	view.Stop()
	if view.stage != stageStopping {
		t.Fatalf("asking a run to stop left the screen in stage %d", view.stage)
	}
	runs.carryOut(t)

	if view.stage != stageStopped {
		t.Fatalf("a run that was stopped ended in stage %d: %v", view.stage, view.failed)
	}
	left, err := os.ReadDir(out)
	if err == nil && len(left) > 0 {
		t.Errorf("a stopped run left %d files in the output directory", len(left))
	}
	if !saying(texts(view.runBox), i18n.English.T(i18n.TakeResultStopped)) {
		t.Error("the screen does not say that the run was stopped")
	}
}

// Nothing on the screen may be changed while a run is going on.
func TestTheSettingsAreOutOfReachWhileARunIsGoingOn(t *testing.T) {
	dir := t.TempDir()
	source := sourceFile(t, dir)
	view, runs := opened(t, nil, source, filepath.Join(dir, "written"))

	view.Start()
	frozen := map[string]fyne.Disableable{
		"kinds of damage": view.layers, "share": view.share, "defects per record": view.perRow,
		"columns to leave alone": view.exclude, "field separator": view.delimiter,
		"data locale": view.locale, "seed": view.seed, "output directory": view.out,
		"extra manifest form": view.manifest, "replacing files": view.overwrite,
		"the button that chooses a file": view.chooseFile,
	}
	for named, control := range frozen {
		if !control.Disabled() {
			t.Errorf("%s can still be changed while a run is going on, so the answer on screen "+
				"would belong to neither the run nor the settings", named)
		}
	}
	// A file dropped onto the window during a run arrives by a road no control
	// guards, so the refusal is asked about by its effect rather than by an
	// answer: the screen has to be holding the file the run is walking.
	view.Open(sourceFile(t, t.TempDir()))
	if view.path != source {
		t.Errorf("a file dropped onto the window during a run replaced the one being walked, "+
			"leaving the screen holding %q", view.path)
	}

	runs.carryOut(t)
	for named, control := range frozen {
		if control.Disabled() {
			t.Errorf("%s was left out of reach after the run had ended", named)
		}
	}
}

// Asking for the window during a run stops the run and closes afterwards.
//
// A window taken away in the middle of a run ends the process where it stands,
// which leaves the temporary files of that run and the marker holding its name
// in somebody's output directory, where the next run finds the marker and
// refuses. Somebody pressing the close button is asking to leave rather than
// asking for that.
func TestAskingForTheWindowDuringARunStopsItFirst(t *testing.T) {
	dir := t.TempDir()
	inner := test.NewWindow(nil)
	defer inner.Close()
	window := &recordingWindow{Window: inner}
	view, runs := opened(t, window, sourceFile(t, dir), filepath.Join(dir, "written"))

	if window.intercept != nil {
		t.Fatal("the screen took the close button over before there was any run to protect")
	}
	view.Start()
	if window.intercept == nil {
		t.Fatal("a run began and the close button still ends the process where it stands")
	}

	window.intercept()
	if view.stage != stageStopping {
		t.Fatalf("asking for the window did not stop the run: the screen is in stage %d", view.stage)
	}
	if window.closed != 0 {
		t.Fatal("the window went away while the run was still walking the file")
	}

	runs.carryOut(t)
	if window.closed != 1 {
		t.Errorf("the run ended and the window was not given to whoever asked for it: closed %d times",
			window.closed)
	}
	if window.intercept != nil {
		t.Error("the close button is still taken over after the run has ended, so the window " +
			"would refuse to close for a reason that no longer exists")
	}
}

// recordingWindow is a window that says what was asked of it.
//
// The toolkit's own test window carries out a close without asking whoever
// asked to be asked, which is the whole subject of the test above, so what it
// answers has to be a window that keeps both.
type recordingWindow struct {
	fyne.Window
	intercept func()
	closed    int
}

func (w *recordingWindow) SetCloseIntercept(callback func()) { w.intercept = callback }

func (w *recordingWindow) Close() { w.closed++ }
