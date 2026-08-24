// Package screens says which screens this window has, in which states, and how
// each state is reached.
//
// # Why this exists rather than a list in each place that needs one
//
// Two things need to put a screen into a state without a person: the guard that
// checks the window still draws what it drew yesterday, and the probe in tools
// that renders every state to a picture. Written twice, those two lists drift,
// and the drift is silent because both keep passing. The neighbouring project
// measured exactly that on the day their second screen arrived: their probe
// asked every screen for a field that only one of them had, so two states were
// never photographed and nothing said so.
//
// So the list lives once, here, and both callers walk it.
//
// # Why the sample is built in code rather than read from a file
//
// A manifest on disk would have to be regenerated whenever the format moves,
// and the thing that notices a format move is precisely the guard reading it.
// A manifest built here moves with the model in the same commit, and the golden
// files below it are then a statement about the drawing rather than about the
// reader.
package screens

import (
	"errors"

	"fyne.io/fyne/v2"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
	"github.com/donislawdev/qadatagenerator/internal/damage"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/domain/response"
	"github.com/donislawdev/qadatagenerator/internal/gui/about"
	"github.com/donislawdev/qadatagenerator/internal/gui/manifestview"
	"github.com/donislawdev/qadatagenerator/internal/gui/shell"
	"github.com/donislawdev/qadatagenerator/internal/gui/takeview"
	"github.com/donislawdev/qadatagenerator/internal/license"
	"github.com/donislawdev/qadatagenerator/internal/take"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

// Data is what the states are drawn from.
type Data struct {
	Promised  manifest.Manifest
	Actual    response.Record
	HasActual bool
	// SourcePath is a file this tool did not make, for the screen whose whole
	// subject is somebody else's file.
	//
	// It is a path rather than the contents, because that screen reads a file
	// the way a person hands it one. Building its state from a value in memory
	// would photograph a screen nobody can reach.
	SourcePath string
}

// Keyboard is a walk through the window, as a person would make it.
//
// Tabs are counted rather than named, because the toolkit moves focus by
// position and naming a control here would be a second register of what the
// window already holds.
type Keyboard struct {
	Tabs int
	Keys []fyne.KeyName
}

// Ran is what a run of the take and corrupt mode left, for the states that show
// one.
//
// # Why the outcome is declared rather than produced
//
// A real run writes files, and a list of states that wrote files would put them
// either in this repository or in a directory whose name then appears in every
// stored tree. Both make the guard comparing those trees answer about something
// other than the drawing.
//
// So these states photograph the DRAWING of an outcome, from an outcome handed
// to the screen the same way a real run hands it one. What holds the other half,
// that the button really damages a file and really says what it wrote, is a test
// in internal/gui/takeview which runs the whole thing into a directory of its
// own. Neither answers the other's question.
type Ran int

const (
	// RanNothing is a screen where nobody has pressed the button.
	RanNothing Ran = iota
	// RanInFlight is a run going on, which is the state that carries the way to
	// stop it.
	RanInFlight
	// RanWritten is a run that carried out everything it ordered.
	RanWritten
	// RanPartial is a run that wrote its files with some of the ordered damage
	// unapplied. docs/17 calls this the fifth state, and it is the one this
	// product exists to make visible.
	RanPartial
	// RanStopped is a run somebody ended.
	RanStopped
	// RanFailed is a run that ended on a failure.
	RanFailed
)

// State is one screen in one condition, and how to get there.
type State struct {
	// Name is the file name a picture of this state gets, so it is a word
	// somebody reads in a directory listing rather than an index.
	Name string
	// Purpose says what this state is worth looking at for.
	Purpose string
	// Screen is which screen of the shell it lives on.
	Screen int
	// NeedsManifest and NeedsResponse say what the state cannot be built
	// without, so a caller with neither skips it loudly instead of drawing an
	// empty screen and calling it the state that was asked for.
	NeedsManifest bool
	NeedsResponse bool
	// NeedsSource says the state cannot be built without a file this tool did
	// not make.
	NeedsSource bool
	// Layers are the kinds of damage ticked before the picture is taken.
	//
	// They go through the control rather than into the settings behind it, so
	// this reaches the state a person reaches by clicking and not one only this
	// list can produce.
	Layers []string
	// Ran is what a run left behind, for the states that show one. A state with
	// anything but RanNothing presses the button, so it also needs the kinds of
	// damage above, or the screen would refuse the run it was asked for.
	Ran Ran
	// Keyboard is applied after the window exists, because focus with no canvas
	// has nothing to land on.
	Keyboard Keyboard
}

// All is every state this window can be put into, for both screens.
//
// One list for both screens on purpose. A list written for one screen stays a
// list of one screen forever, and nobody notices, because it is green.
func All() []State {
	states := takeStates()
	states = append(states, browserStates()...)
	return append(states, aboutStates()...)
}

// takeStates are the states of the screen that damages somebody else's file,
// which is the screen with the most of them.
func takeStates() []State {
	return append(takeStatesBeforeARun(), takeStatesOfARun()...)
}

// takeStatesBeforeARun are the states in which nothing has been written.
func takeStatesBeforeARun() []State {
	return []State{

		{
			Name:    "take-empty",
			Purpose: "no file chosen, a sentence rather than an empty rectangle",
			Screen:  0,
		},
		{
			Name:        "take-refused",
			Purpose:     "a file read and no kind of damage chosen, so the screen says why it would not run",
			Screen:      0,
			NeedsSource: true,
		},
		{
			Name:        "take-plan",
			Purpose:     "a file read and two kinds of damage chosen, so the screen says what a run would do",
			Screen:      0,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
		},
		{
			Name: "take-focus-file",
			Purpose: "the keyboard on the button that chooses a file, which is the first " +
				"thing on this screen it reaches and was for a while the last",
			Screen:      0,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Keyboard:    Keyboard{Tabs: 4},
		},
		{
			Name: "take-focus-setting",
			Purpose: "the keyboard in a text setting, which is the one control the toolkit " +
				"marks with the chosen colour rather than the focus colour",
			Screen:      0,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Keyboard:    Keyboard{Tabs: 13},
		},
		{
			Name: "take-focus-damage",
			Purpose: "the keyboard at the far end of the screen, on the button that damages " +
				"the file, having passed every setting of the run on the way",
			Screen:      0,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Keyboard:    Keyboard{Tabs: 22},
		},
	}
}

// takeStatesOfARun are the states a run puts the screen into.
//
// Every one of them ticks kinds of damage first, because a state pressing the
// button without them would be a picture of the screen refusing the run rather
// than of the run.
func takeStatesOfARun() []State {
	return []State{
		{
			Name:        "take-running",
			Purpose:     "a run going on, with the settings out of reach and the way to stop it in reach",
			Screen:      0,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Ran:         RanInFlight,
		},
		{
			Name: "take-focus-stop",
			Purpose: "the keyboard on the way to stop a run, which is the only thing on " +
				"this screen it can reach while one is going on",
			Screen:      0,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Ran:         RanInFlight,
			Keyboard:    Keyboard{Tabs: 4},
		},
		{
			Name:        "take-written",
			Purpose:     "a run that finished and carried out everything it ordered",
			Screen:      0,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Ran:         RanWritten,
		},
		{
			Name: "take-partial",
			Purpose: "a run that wrote its files with some of the ordered damage unapplied, " +
				"which is the state a count of what was applied hides",
			Screen:      0,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Ran:         RanPartial,
		},
		{
			Name:        "take-stopped",
			Purpose:     "a run somebody ended, which leaves nothing behind and says so",
			Screen:      0,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Ran:         RanStopped,
		},
		{
			Name:        "take-failed",
			Purpose:     "a run that ended on a failure, with what failed and what that left",
			Screen:      0,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Ran:         RanFailed,
		},
	}
}

// browserStates are the states of the manifest browser.
func browserStates() []State {
	return []State{
		{
			Name:    "browser-empty",
			Purpose: "no manifest open, a sentence rather than an empty rectangle",
			Screen:  1,
		},
		{
			Name:          "browser-promised",
			Purpose:       "a manifest open, which is what a run promised about a dataset",
			Screen:        1,
			NeedsManifest: true,
		},
		{
			Name:          "browser-compared",
			Purpose:       "the same manifest beside what an application did about it",
			Screen:        1,
			NeedsManifest: true,
			NeedsResponse: true,
		},
		{
			Name:          "browser-focus-button",
			Purpose:       "the keyboard on the button that opens a manifest",
			Screen:        1,
			NeedsManifest: true,
			Keyboard:      Keyboard{Tabs: 4},
		},
		{
			Name:          "browser-focus-cell",
			Purpose:       "the keyboard inside the table, three rows down and one column across",
			Screen:        1,
			NeedsManifest: true,
			Keyboard: Keyboard{
				Tabs: 6,
				Keys: []fyne.KeyName{fyne.KeyDown, fyne.KeyDown, fyne.KeyDown, fyne.KeyRight},
			},
		},
	}
}

// aboutStates are the states of the screen naming what this build is.
func aboutStates() []State {
	return []State{
		{
			Name:    "about",
			Purpose: "versions, licences and every third party component",
			Screen:  2,
		},
		{
			Name:     "about-focus-name",
			Purpose:  "the keyboard on the name of the screen being read",
			Screen:   2,
			Keyboard: Keyboard{Tabs: 3},
		},
	}
}

// Buildable reports whether this state can be drawn from the data given, and
// says what is missing when it cannot.
func (s State) Buildable(data Data) (bool, string) {
	if s.NeedsManifest && data.Promised.SpecVersion() == "" {
		return false, "no manifest"
	}
	if s.NeedsResponse && !data.HasActual {
		return false, "no response record"
	}
	if s.NeedsSource && data.SourcePath == "" {
		return false, "no source file"
	}
	return true, ""
}

// Assemble builds the whole window in one state, and hands back its content.
//
// The screen is put into its state through the same calls the file chooser
// makes, so nothing here can reach a state a person cannot.
func Assemble(text i18n.Catalogue, data Data, state State) fyne.CanvasObject {
	view := manifestview.New(text, nil, manifestview.Readers{})
	takeScreen := takeview.New(text, nil, describedDamage(), take.Open, outcomeOf(state.Ran))
	frame := shell.New([]shell.Screen{
		{Name: text.T(i18n.ScreenTake), Content: takeScreen.Content(), OnDropped: takeScreen.Dropped},
		{Name: text.T(i18n.ScreenManifest), Content: view.Content()},
		{Name: text.T(i18n.ScreenAbout), Content: about.Content(text, license.Current(), version.Current())},
	})
	if state.NeedsManifest {
		view.Open(data.Promised)
	}
	if state.NeedsResponse && data.HasActual {
		view.Compare(data.Actual)
	}
	if state.NeedsSource && data.SourcePath != "" {
		// The failure is deliberately not swallowed into an empty screen: a
		// state that says it needs a file and then draws one that has none is a
		// picture of something nobody asked for.
		if err := takeScreen.Open(data.SourcePath); err != nil {
			panic("the sample source file could not be read: " + err.Error())
		}
	}
	if len(state.Layers) > 0 {
		takeScreen.ChooseLayers(state.Layers)
	}
	if state.Ran != RanNothing {
		// The same call the button makes, so the state is one a person reaches
		// by pressing it rather than one only this list can produce.
		takeScreen.Start()
	}
	frame.Open(state.Screen)
	return frame.Content()
}

// outcomeOf is the place a run happens, for a state that wants a particular
// outcome of one.
//
// It never carries the work out. What it hands back is what a real run of that
// shape hands back, by the same road, so the screen cannot tell the difference
// and the drawing is the drawing of a real outcome.
func outcomeOf(ran Ran) takeview.Background {
	return func(_ func() (take.Result, error), done func(take.Result, error)) {
		switch ran {
		case RanInFlight:
			// Nothing is answered, so the run is still going when the picture
			// is taken.
		case RanWritten:
			done(sampleRun(sampleOrdered), nil)
		case RanPartial:
			done(sampleRun(samplePartial), nil)
		case RanStopped:
			done(take.Result{}, takeview.ErrStopped)
		case RanFailed:
			done(take.Result{}, errors.New(sampleFailure))
		}
	}
}

// What a sample run of this mode did, as numbers.
//
// They are numbers rather than a run because of what the states above are for.
// The share of records is a draw, so a real run of this file would produce a
// count that moves whenever the sample file or the seed moves, and every stored
// tree would then be a statement about the draw rather than about the screen.
const (
	sampleOrdered     = 14
	samplePartial     = 9
	sampleUnpartnered = 2
	// sampleFailure is a failure of the kind the writer of files reports, in
	// its own words, so that the state shows the length of message a person
	// really gets rather than a short one chosen to fit.
	sampleFailure = "out/contacts.corrupted.csv already exists. Delete it, choose another " +
		"directory, or tick the setting that allows replacing. Nothing has been written"
)

// sampleRun is what a finished run left, with as many of the ordered defects
// applied as the caller asks for.
func sampleRun(applied int) take.Result {
	return take.Result{
		Written: []string{"contacts" + take.Suffix, manifestjson.Name},
		Out:     "out",
		Counts: damage.Counts{
			Requested: sampleOrdered, Applied: applied, Unpartnered: sampleUnpartnered,
		},
	}
}

// describedDamage is the catalog this build carries.
//
// A failure here means the binary is not what it was built as, and every state
// below it would then be a picture of a screen that cannot exist, so it stops
// rather than drawing one.
func describedDamage() catalog.Catalog {
	described, err := catalogyaml.Load()
	if err != nil {
		panic("this build cannot read its own catalog of damage: " + err.Error())
	}
	return described
}

// Walk applies the keyboard of a state to a canvas that already exists.
//
// It presses rather than calls: Tab goes through the focus manager and an arrow
// goes to whatever holds focus, which is the road a key travels in a real
// window. Calling a method on the control directly reaches places a key cannot,
// and would answer a question nobody asked.
func Walk(canvas fyne.Canvas, state State) {
	for i := 0; i < state.Keyboard.Tabs; i++ {
		canvas.FocusNext()
	}
	for _, key := range state.Keyboard.Keys {
		if focused := canvas.Focused(); focused != nil {
			focused.TypedKey(&fyne.KeyEvent{Name: key})
		}
	}
}

// Sample is a manifest and a response record small enough to read and wide
// enough to carry every kind of entry the screen draws differently.
//
// Every colour of the palette that means something appears at least once, and
// so does damage that was ordered and could not be applied, because that is the
// entry this product exists to make visible.
func Sample() Data {
	promised, problems := manifest.New("0.1", manifest.Run{
		GeneratedAt:    "2026-08-18T09:19:59Z",
		Seed:           "policy-walkthrough",
		Locale:         "pl-PL",
		CatalogVersion: "0.1",
		Recipe:         "policy walkthrough",
		RowsTotal:      6,
		RowsMutated:    4,
	}, sampleDefects())
	if len(problems) > 0 {
		// A sample that is not a valid manifest would make every golden file
		// below it a statement about a document this tool would refuse.
		panic("the sample manifest is not valid: " + problems[0])
	}
	return Data{Promised: promised, Actual: sampleResponse(), HasActual: true}
}

func sampleDefects() []manifest.Defect {
	return []manifest.Defect{{
		ID:       "one",
		Location: manifest.Location{Entity: "customer", Row: 1, Field: "pesel"},
		Mutator:  "L3.checksum.off_by_one", Layer: catalog.Domain, Applied: true,
		ValueBefore: "44051401359", HasValueBefore: true,
		ValueAfter: "44051401358", HasValueAfter: true,
		Expectation: catalog.Rejection, Confidence: catalog.Hard,
		Rationale: "the check digit no longer agrees with the digits before it",
	}, {
		ID:       "two",
		Location: manifest.Location{Entity: "customer", Row: 2, Field: "last_name"},
		Mutator:  "L2.string.empty", Layer: catalog.FieldShape, Applied: true,
		ValueBefore: "Fronczek", HasValueBefore: true,
		ValueAfter: "", HasValueAfter: true,
		Expectation: catalog.Rejection, Confidence: catalog.Soft,
		Rationale: "a required column arrived empty",
	}, {
		ID:       "three",
		Location: manifest.Location{Entity: "customer", Row: 3, Field: "gender"},
		Mutator:  "L2.coercion.text_to_number", Layer: catalog.FieldShape, Applied: true,
		ValueBefore: "F", HasValueBefore: true,
		ValueAfter: "6", HasValueAfter: true,
		Expectation: catalog.Acceptance, Confidence: catalog.Soft,
		Rationale: "a value outside the set the column is documented to carry",
	}, {
		ID:       "four",
		Location: manifest.Location{Entity: "customer", Row: 4, Field: "quantity"},
		Mutator:  "L2.type.string_in_numeric", Layer: catalog.FieldShape, Applied: true,
		ValueBefore: "3492", HasValueBefore: true,
		ValueAfter: "n/a", HasValueAfter: true,
		Expectation: catalog.Warning, Confidence: catalog.Soft,
		Rationale: "text where the column is documented to carry a number",
	}, {
		ID:       "five",
		Location: manifest.Location{Entity: "customer", Row: 5, Field: "birth_date"},
		Mutator:  "L2.date.impossible", Layer: catalog.FieldShape, Applied: false,
		FailureKind:   manifest.Unimplemented,
		FailureReason: "this build carries no implementation of them yet, only its description",
		Expectation:   catalog.Rejection, Confidence: catalog.Hard,
		Rationale: "a date that never happened",
	}, {
		ID:       "six",
		Location: manifest.Location{Entity: "customer", Row: 6, Field: "ssn"},
		Mutator:  "L3.sentinel.placeholder_value", Layer: catalog.Domain, Applied: false,
		FailureKind:   manifest.Unimplemented,
		FailureReason: "this build carries no implementation of them yet, only its description",
		Expectation:   catalog.Acceptance, Confidence: catalog.Soft,
		Rationale: "a placeholder an issuer never gives out",
	}}
}

// sampleResponse answers about four of the six rows, and that is the point.
//
// One row disagrees with a hard expectation, one disagrees with a soft one, and
// two rows are absent, which is what makes the screen say what the comparison
// could not settle rather than pretending it settled everything.
func sampleResponse() response.Record {
	return response.Record{
		SpecVersion: "0.1",
		Rows: []response.Row{
			{Number: 1, Verdict: response.Rejected, Code: "ERR_CHECKSUM"},
			{Number: 2, Verdict: response.Accepted},
			{Number: 3, Verdict: response.Accepted},
			{Number: 4, Verdict: response.Rejected, Code: "ERR_TYPE"},
		},
	}
}
