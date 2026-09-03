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
	"fyne.io/fyne/v2"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
	"github.com/donislawdev/qadatagenerator/internal/damage"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/domain/response"
	"github.com/donislawdev/qadatagenerator/internal/fault"
	"github.com/donislawdev/qadatagenerator/internal/gui/about"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	"github.com/donislawdev/qadatagenerator/internal/gui/manifestview"
	"github.com/donislawdev/qadatagenerator/internal/gui/recipeview"
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
	// Recipe is what is written on the recipe editor before the picture is
	// taken. It goes in through the field a person types into, so the state is
	// one somebody reaches by typing rather than one only this list produces.
	Recipe string
	// Into is the directory the recipe editor writes to, as somebody typed it.
	//
	// 🔴 A word rather than a real path, and never one this machine has. A
	// stored tree carries whatever is in the field, so a real directory would
	// put somebody's home in a file this repository keeps and would differ
	// between two machines running the same guard.
	Into string
	// Checking leaves the judging of the recipe unanswered, which is the screen
	// a person looks at while a large one is being read. It is the recipe
	// editor's twin of Reading below, and separate for the same reason: the two
	// screens are told to wait by different places of work.
	Checking bool
	// Reading leaves the read of the file unanswered, which is the screen a
	// person looks at while a large one is being walked.
	//
	// It is a place of work that never answers rather than a flag the screen
	// carries, so this reaches the state by the one road a person reaches it
	// by. A screen told to draw itself as reading would be a picture of a
	// drawing rather than of a state.
	Reading bool
	// Keyboard is applied after the window exists, because focus with no canvas
	// has nothing to land on.
	Keyboard Keyboard
}

// All is every state this window can be put into, for both screens.
//
// One list for both screens on purpose. A list written for one screen stays a
// list of one screen forever, and nobody notices, because it is green.
func All() []State {
	states := recipeStates()
	states = append(states, takeStates()...)
	states = append(states, browserStates()...)
	return append(states, aboutStates()...)
}

// takeStates are the states of the screen that damages somebody else's file,
// which is the screen with the most of them.
func takeStates() []State {
	states := append(takeStatesBeforeARun(), takeStatesOfTheKeyboard()...)
	return append(states, takeStatesOfARun()...)
}

// takeStatesBeforeARun are the states in which nothing has been written.
func takeStatesBeforeARun() []State {
	return []State{

		{
			Name:    "take-empty",
			Purpose: "no file chosen, a sentence rather than an empty rectangle",
			Screen:  1,
		},
		{
			Name:        "take-refused",
			Purpose:     "a file read and no kind of damage chosen, so the screen says why it would not run",
			Screen:      1,
			NeedsSource: true,
		},
		{
			Name:        "take-plan",
			Purpose:     "a file read and two kinds of damage chosen, so the screen says what a run would do",
			Screen:      1,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
		},
	}
}

// takeStatesOfTheKeyboard are the places on this screen the keyboard can stand,
// which are worth a picture each because what marks them is a colour and a
// shape rather than a word.
func takeStatesOfTheKeyboard() []State {
	return []State{
		{
			Name: "take-focus-file",
			Purpose: "the keyboard on the button that chooses a file, which is the first " +
				"thing on this screen it reaches and was for a while the last",
			Screen:      1,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Keyboard:    Keyboard{Tabs: 5},
		},
		{
			Name: "take-focus-setting",
			Purpose: "the keyboard in a text setting, which is the one control the toolkit " +
				"marks with the chosen colour rather than the focus colour",
			Screen:      1,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Keyboard:    Keyboard{Tabs: 16},
		},
		{
			Name: "take-focus-pane",
			Purpose: "the keyboard on the pane that says what the file holds, which the " +
				"toolkit's own scrolling area never lets it reach",
			Screen:      1,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Keyboard:    Keyboard{Tabs: 6},
		},
		{
			Name: "take-focus-damage",
			Purpose: "the keyboard at the far end of the screen, on the button that damages " +
				"the file, having passed every setting of the run on the way",
			Screen:      1,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Keyboard:    Keyboard{Tabs: 25},
		},
		{
			Name: "take-reading",
			Purpose: "a file being read, which for a file this mode is for means walking " +
				"all of it while the window stays answerable",
			Screen:      1,
			NeedsSource: true,
			Reading:     true,
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
			Screen:      1,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Ran:         RanInFlight,
		},
		{
			Name: "take-focus-stop",
			Purpose: "the keyboard on the way to stop a run, which is the only thing on " +
				"this screen it can reach while one is going on",
			Screen:      1,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Ran:         RanInFlight,
			Keyboard:    Keyboard{Tabs: 7},
		},
		{
			Name:        "take-written",
			Purpose:     "a run that finished and carried out everything it ordered",
			Screen:      1,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Ran:         RanWritten,
		},
		{
			Name: "take-partial",
			Purpose: "a run that wrote its files with some of the ordered damage unapplied, " +
				"which is the state a count of what was applied hides",
			Screen:      1,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Ran:         RanPartial,
		},
		{
			Name:        "take-stopped",
			Purpose:     "a run somebody ended, which leaves nothing behind and says so",
			Screen:      1,
			NeedsSource: true,
			Layers:      []string{"L2", "L3"},
			Ran:         RanStopped,
		},
		{
			Name:        "take-failed",
			Purpose:     "a run that ended on a failure, with what failed and what that left",
			Screen:      1,
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
			Screen:  2,
		},
		{
			Name:          "browser-promised",
			Purpose:       "a manifest open, which is what a run promised about a dataset",
			Screen:        2,
			NeedsManifest: true,
		},
		{
			Name:          "browser-compared",
			Purpose:       "the same manifest beside what an application did about it",
			Screen:        2,
			NeedsManifest: true,
			NeedsResponse: true,
		},
		{
			Name:          "browser-focus-button",
			Purpose:       "the keyboard on the button that opens a manifest",
			Screen:        2,
			NeedsManifest: true,
			Keyboard:      Keyboard{Tabs: 5},
		},
		{
			Name:          "browser-focus-cell",
			Purpose:       "the keyboard inside the table, three rows down and one column across",
			Screen:        2,
			NeedsManifest: true,
			Keyboard: Keyboard{
				Tabs: 7,
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
			Screen:  3,
		},
		{
			Name:     "about-focus-name",
			Purpose:  "the keyboard on the name of the screen being read",
			Screen:   3,
			Keyboard: Keyboard{Tabs: 4},
		},
		{
			Name: "about-focus-body",
			Purpose: "the keyboard in the text of the screen, which the toolkit's own " +
				"scrolling area never allows it to reach",
			Screen:   3,
			Keyboard: Keyboard{Tabs: 5},
		},
		{
			Name: "about-scrolled",
			Purpose: "the same screen two pages further down, reached with the keyboard " +
				"alone, which is three quarters of it that a mouse used to be needed for",
			Screen: 3,
			Keyboard: Keyboard{
				Tabs: 5,
				Keys: []fyne.KeyName{fyne.KeyPageDown, fyne.KeyPageDown},
			},
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
	view := manifestview.New(text, nil, manifestview.Readers{}, background.ReadHere)
	// The read happens where it is asked for, because this caller has no
	// drawing thread to hand anything back to and wants the screen finished by
	// the time the call returns. Nothing is watching for a signal either: these
	// screens belong to a test and to a probe rather than to a process somebody
	// can ask to stop.
	takeScreen := takeview.New(text, nil, describedDamage(), take.Open, background.Work{
		Run:  outcomeOf(state.Ran),
		Read: readingFor(state.Reading),
	})
	// 🔴 The way to write is handed in as nothing at all. A state of this list
	// is a picture, and a picture that wrote a dataset would leave it either in
	// this repository or in a directory whose name then appears in every stored
	// tree. The button is drawn in whatever state it is in, which is what these
	// pictures are of, and that the button really writes is held by a test in
	// the screen's own package against a directory of its own.
	recipeScreen := recipeview.New(text, recipeview.Checker(describedDamage()), nil,
		background.Work{Read: readingFor(state.Checking)}, nil)
	frame := shell.New([]shell.Screen{
		{Name: text.T(i18n.ScreenRecipe), Content: recipeScreen.Content()},
		{Name: text.T(i18n.ScreenTake), Content: takeScreen.Content(), OnDropped: takeScreen.Dropped},
		{Name: text.T(i18n.ScreenManifest), Content: view.Content()},
		{Name: text.T(i18n.ScreenAbout), Content: about.Content(text, license.Current(), version.Current())},
	})
	if state.Into != "" {
		recipeScreen.SetOut(state.Into)
	}
	if state.Recipe != "" {
		recipeScreen.SetRecipe(state.Recipe)
	}
	if state.NeedsManifest {
		view.Open(data.Promised)
	}
	if state.NeedsResponse && data.HasActual {
		view.Compare(data.Actual)
	}
	if state.NeedsSource && data.SourcePath != "" {
		takeScreen.Open(data.SourcePath)
	}
	if !state.Reading {
		// A state whose read never answers stops here, because everything below
		// asks the screen for something only a finished read can give it.
		settle(takeScreen, state)
	}
	frame.Open(state.Screen)
	return frame.Content()
}

// settle takes the screen that damages a file the rest of the way, once its
// file has been read.
//
// 🔴 It is a call of its own rather than a branch inside Assemble, and the
// reason is a mutation that went uncaught on 2026-09-01. Written as an early
// return, this left Assemble with TWO calls asking the frame for a screen, and
// the mutation aimed at the one that says every state renders on the first
// screen then landed on a branch whose state is on the first screen anyway. A
// guard that had been green for a fortnight was quietly answering about
// nothing.
func settle(takeScreen *takeview.View, state State) {
	// The failure is deliberately not swallowed into an empty screen: a state
	// that says it needs a file and then draws one that has none is a picture
	// of something nobody asked for.
	//
	// It is asked for after the fact rather than handed back, because the read
	// no longer happens where it is asked for. The place of work this screen
	// was given reads where it stands, so by here it has finished.
	if err := takeScreen.Unread(); err != nil {
		panic("the sample source file could not be read: " + err.Error())
	}
	if len(state.Layers) > 0 {
		takeScreen.ChooseLayers(state.Layers)
	}
	if state.Ran != RanNothing {
		// The same call the button makes, so the state is one a person reaches
		// by pressing it rather than one only this list can produce.
		takeScreen.Start()
	}
}

// readingFor is the place a read happens, which for one state is nowhere.
//
// A read that never answers is what a person is looking at while a large file
// is walked, and it is the only way to photograph that state: a read that
// answers has finished by the time anything could take the picture.
func readingFor(waiting bool) background.Read {
	if waiting {
		return func(func() func(), func(error) func()) {}
	}
	return background.ReadHere
}

// outcomeOf is the place a run happens, for a state that wants a particular
// outcome of one.
//
// It never carries the work out. What it hands back is what a real run of that
// shape hands back, by the same road, so the screen cannot tell the difference
// and the drawing is the drawing of a real outcome.
func outcomeOf(ran Ran) background.Run {
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
			done(take.Result{}, background.ErrStopped)
		case RanFailed:
			done(take.Result{}, fault.New(fault.IO, sampleFailure).DoInstead(sampleFailureInstead))
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
	//
	// 🔴 It is built the way the writer builds it, with what went wrong and
	// what to do about it apart, because that is what this state is a picture
	// of. Until 2026-09-03 the two were one sentence, and the half of it that
	// this window showed told the reader to pass a flag, which is a thing this
	// window does not have.
	sampleFailure        = "out/contacts.corrupted.csv already exists, and nothing has been written"
	sampleFailureInstead = "delete it, choose another directory, or allow replacing"
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
