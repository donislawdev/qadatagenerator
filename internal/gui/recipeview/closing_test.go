package recipeview

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	"github.com/donislawdev/qadatagenerator/internal/produce"
)

// watchedWindow is a window that says whether it was asked to close, and does
// not close itself.
//
// The toolkit's own test window carries out a close without asking, so a screen
// that hands the window back at the end of a run cannot be watched through one.
type watchedWindow struct {
	fyne.Window
	closed    int
	intercept func()
}

func (w *watchedWindow) Close() { w.closed++ }

// SetCloseIntercept keeps what the screen put there, so that a test can press
// the cross the way the toolkit would.
func (w *watchedWindow) SetCloseIntercept(intercept func()) { w.intercept = intercept }

// heldRun is a run that does not finish until the test says so.
type heldRun struct {
	work func() (produce.Written, error)
	done func(produce.Written, error)
}

func (h *heldRun) produce(work func() (produce.Written, error),
	done func(produce.Written, error)) {
	h.work, h.done = work, done
}

// finish lets the held run answer, which is what a real one does on the thread
// that draws once it is over.
func (h *heldRun) finish() {
	if h.done != nil {
		h.done(h.work())
	}
}

// heldWatch is the watch for a signal, with the signal itself sent by the test.
type heldWatch struct {
	asked    func()
	watching int
	released int
}

func (h *heldWatch) watch(asked func()) func() {
	h.asked = asked
	h.watching++
	return func() { h.released++ }
}

// writing is a screen in the middle of a run somebody can ask to stop.
func writing(t *testing.T, window fyne.Window) (*View, *heldRun, *heldWatch) {
	t.Helper()
	described, err := catalogyaml.Load()
	if err != nil {
		t.Fatalf("the catalog inside this binary could not be read: %v", err)
	}
	runs, watch := &heldRun{}, &heldWatch{}
	view := New(i18n.English, Checker(described), stillRun, background.Work{
		Read: background.ReadHere, Produce: runs.produce, Watch: watch.watch,
	}, window)
	view.Content()
	view.SetRecipe(cleanEnough)
	view.SetOut(t.TempDir())
	view.Start()
	return view, runs, watch
}

// stillRun is a way of writing that answers without touching a disk, because
// these tests are about what the window does while a run is going on rather
// than about what the run produces.
func stillRun([]byte, string, bool, generate.Halt) (produce.Written, error) {
	return produce.Written{Files: []string{"customer.csv"}, Rows: 3}, nil
}

const cleanEnough = `version: "0.1"
seed: "sample"
entities:
  customer:
    count: 3
    fields:
      name:
        generator: first_name
`

// 🔴 Closing the window during a run stops the run first.
//
// A process ending where it stands leaves the temporary files of the run and
// the marker holding a name in somebody's output directory, where the next run
// finds the marker and refuses for a reason that stopped being true. A person
// pressing the close button is asking to leave rather than to kill anything.
func TestAskingForTheWindowWhileWritingADatasetStopsItFirst(t *testing.T) {
	window := &watchedWindow{Window: test.NewWindow(nil)}
	t.Cleanup(func() { window.Window.Close() })
	view, runs, _ := writing(t, window)

	if window.intercept == nil {
		t.Fatal("a run is going on and the close button was left to the toolkit")
	}
	window.intercept()

	if window.closed != 0 {
		t.Fatal("the window was given up while the run was still going on")
	}
	runs.finish()
	if window.closed != 1 {
		t.Errorf("the window was not given up once the run had ended, closed %d times",
			window.closed)
	}
	if _, _, ran := view.Written(); !ran {
		t.Error("the run never answered")
	}
}

// 🔴 The system asking this program to stop means what the close button means.
//
// Both are the same ask from different people and both cost the same thing, so
// the second is routed into the answer the window already had. The watch lives
// exactly as long as the run.
func TestTheSystemAskingThisProgramToStopEndsTheDatasetRunFirst(t *testing.T) {
	window := &watchedWindow{Window: test.NewWindow(nil)}
	t.Cleanup(func() { window.Window.Close() })
	_, runs, watch := writing(t, window)

	if watch.watching != 1 {
		t.Fatalf("a run started and the watch was set up %d times", watch.watching)
	}
	watch.asked()
	if window.closed != 0 {
		t.Fatal("the window was given up while the run was still going on")
	}

	runs.finish()
	if window.closed != 1 {
		t.Errorf("the window was not given up once the run had ended, closed %d times",
			window.closed)
	}
	if watch.released != 1 {
		t.Errorf("the watch outlived the run it belongs to, released %d times", watch.released)
	}
}
