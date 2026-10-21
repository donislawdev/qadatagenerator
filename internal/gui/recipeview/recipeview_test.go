package recipeview_test

import (
	"strconv"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/app/validate"
	"github.com/donislawdev/qadatagenerator/internal/fault"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	"github.com/donislawdev/qadatagenerator/internal/gui/recipeview"
	"github.com/donislawdev/qadatagenerator/internal/gui/screens"
)

// here is a screen whose checking happens where it is asked for, so that a test
// can look at the answer on the line after it asked for one.
func here(t *testing.T, check recipeview.Check) *recipeview.View {
	t.Helper()
	view := recipeview.New(i18n.English, check, nil, background.Work{Read: background.ReadHere}, nil)
	view.Content()
	return view
}

// real is the check the window is really given, so that what these tests see is
// what a person sees rather than a stub agreeing with them.
func real(t *testing.T) recipeview.Check {
	t.Helper()
	described, err := catalogyaml.Load()
	if err != nil {
		t.Fatalf("this build cannot read its own catalog of damage: %v", err)
	}
	return recipeview.Checker(described)
}

// The screen judges what is written on it, through the real reader.
//
// 🔴 The two recipes are the ones the stored states are drawn from, so this is
// also what holds those states to being what they claim. A sample that quietly
// stopped being broken would leave a state called recipe-problems drawing an
// approval, and every stored tree would agree with it.
func TestTheScreenJudgesTheRecipeWrittenOnIt(t *testing.T) {
	view := here(t, real(t))

	view.SetRecipe(screens.CleanRecipe)
	report, checked := view.Report()
	if !checked {
		t.Fatalf("nothing was judged, so the field never reached the check")
	}
	if !report.Usable() {
		t.Errorf("the clean sample is not clean: %v", report.Problems)
	}

	view.SetRecipe(screens.BrokenRecipe)
	report, _ = view.Report()
	if report.Usable() {
		t.Errorf("the broken sample carries no problems, so the state drawn from it " +
			"is a picture of an approval")
	}
}

// A problem reaches the screen with the line it was written on.
//
// That line is the whole reason the previous session touched the reader, and a
// screen that dropped it on the way would leave that work with nothing to show
// for it on the surface it was done for.
func TestAProblemReachesTheScreenWithItsLine(t *testing.T) {
	view := here(t, real(t))
	view.SetRecipe(screens.BrokenRecipe)

	report, _ := view.Report()
	if len(report.Problems) == 0 {
		t.Fatalf("the broken sample produced no problem to carry a line")
	}
	found := report.Problems[0]
	if found.Line <= 0 {
		t.Errorf("the problem at %q carries no line, so the editor cannot say where to look",
			found.Where)
	}
	// The line of the FIELD rather than of the generator inside it, because the
	// place the problem names is the field. Written down because the first
	// version of this test expected the generator's line, one lower, and the
	// command line had been saying the field's line all along.
	wanted := lineOf(screens.BrokenRecipe, "email:")
	if found.Line != wanted {
		t.Errorf("the problem at %q says line %d and that field is written on line %d",
			found.Where, found.Line, wanted)
	}
}

// An empty screen is not an approved one.
//
// 🔴 The two look alike in a report and mean opposite things, so this is the
// one pair the drawing must never merge. Telling somebody their blank window
// has no objections is worse than saying nothing, because it is an answer.
// 🔴 The screen is CLEARED rather than started empty, and a mutation is why.
// The first version of this called SetRecipe("") on a field that was already
// empty, and the toolkit does not report a change that changes nothing, so the
// empty path was never reached and the test passed without ever running the
// rule it is named after. Clearing is also the road a person travels: they
// select what they wrote and delete it.
func TestAnEmptyScreenIsNotAnApprovedOne(t *testing.T) {
	var asked int
	view := here(t, func(string, []byte) recipeview.Answer {
		asked++
		return recipeview.Answer{}
	})

	view.SetRecipe(screens.CleanRecipe)
	if _, checked := view.Report(); !checked {
		t.Fatalf("nothing was judged, so this test never reaches what it is about")
	}
	sentSoFar := asked

	view.SetRecipe("")
	if _, checked := view.Report(); checked {
		t.Errorf("a cleared screen still counts as judged, so it would go on drawing " +
			"an answer about a recipe that is no longer there")
	}
	if asked != sentSoFar {
		t.Errorf("clearing the screen sent it to the reader again, and there is "+
			"nothing there to read: %d checks before, %d after", sentSoFar, asked)
	}
}

// An answer about a recipe nobody has on screen any more is not drawn.
//
// 🔴 This is the fault the window closed for reading somebody else's file, and
// it arrives here by the same road: a recipe is typed one character at a time
// and each character asks the question again. The check is deliberately made to
// answer out of order, because in order it cannot fail.
func TestAnAnswerAboutARecipeNobodyIsWaitingForIsNotDrawn(t *testing.T) {
	var answer []func()
	later := background.Work{Read: func(read func() func(), _ func(error) func()) {
		answer = append(answer, read())
	}}
	view := recipeview.New(i18n.English, func(_ string, data []byte) recipeview.Answer {
		// The report names the recipe it was asked about, so the assertion
		// below is about WHICH answer was kept rather than about how many.
		return recipeview.Answer{Report: validate.Report{Name: string(data)}}
	}, nil, later, nil)
	view.Content()

	view.SetRecipe("first")
	view.SetRecipe("second")
	if len(answer) != 2 {
		t.Fatalf("expected two checks in flight and there are %d", len(answer))
	}

	// The stale one lands last, which is the order that breaks a screen with no
	// counter in it.
	answer[1]()
	answer[0]()

	report, _ := view.Report()
	if report.Name != "second" {
		t.Errorf("the screen is showing the answer about %q while %q is in the field",
			report.Name, "second")
	}
}

// While a recipe is being judged the screen says so.
//
// A window that has not answered yet and a window that has stopped answering
// look identical, and the sentence is the only thing that tells them apart.
func TestTheScreenSaysThatItIsChecking(t *testing.T) {
	waiting := background.Work{Read: func(func() func(), func(error) func()) {}}
	view := recipeview.New(i18n.English, func(string, []byte) recipeview.Answer {
		return recipeview.Answer{}
	}, nil, waiting, nil)
	view.Content()

	view.SetRecipe(screens.CleanRecipe)
	if !view.Checking() {
		t.Errorf("a check that has not answered leaves the screen saying nothing about it")
	}
	if drawn := rendered(t, view); !strings.Contains(drawn, i18n.English.T(i18n.RecipeChecking)) {
		t.Errorf("the screen does not say it is checking, and a person cannot tell "+
			"that from a window that has stopped answering.\n%s", drawn)
	}
}

// A check of the next keystroke leaves the answer about the last one standing.
//
// 🔴 The owner reported on 2026-10-21 that the card blinked once a letter while
// they typed: every keystroke replaced the answer with the sentence saying a
// check is going on, and the answer came back a moment later. The second check
// is held here so that the screen is asked in exactly that moment.
func TestTypingDoesNotBlankTheAnswerWhileTheNextCheckRuns(t *testing.T) {
	var answer []func()
	check := real(t)
	held := background.Work{Read: func(read func() func(), _ func(error) func()) {
		answer = append(answer, read())
	}}
	view := recipeview.New(i18n.English, check, nil, held, nil)
	view.Content()

	view.SetRecipe(screens.BrokenRecipe)
	if len(answer) != 1 {
		t.Fatalf("expected one check in flight and there are %d", len(answer))
	}
	answer[0]()
	report, _ := view.Report()
	if len(report.Problems) == 0 {
		t.Fatalf("the broken sample has no problems, so this test proves nothing")
	}

	view.SetRecipe(screens.BrokenRecipe + " ")
	if !view.Checking() {
		t.Fatalf("the second keystroke started no check")
	}
	drawn := spoken(rendered(t, view))
	if strings.Contains(drawn, i18n.English.T(i18n.RecipeChecking)) {
		t.Errorf("the answer on screen was replaced by the sentence about checking, " +
			"which is the blink the owner reported")
	}
	// The place rather than the sentence, because the sentence wraps and the
	// spoken line is cut into one piece per line drawn.
	if !strings.Contains(drawn, report.Problems[0].Where) {
		t.Errorf("the problem found a keystroke ago is gone while the next check runs:\n%s", drawn)
	}
}

// What the screen draws says where each problem is, in words.
//
// The tree is asked rather than the model, because a line held in a report and
// absent from the screen is the family of fault this window has paid for four
// times over.
func TestTheDrawnAnswerCarriesThePlaceAndTheLine(t *testing.T) {
	view := here(t, real(t))
	// About words, so the spoken line rather than the markup: this sentence
	// carries a path, a line number and a reason, and it wraps.
	view.SetRecipe(screens.BrokenRecipe)

	drawn := spoken(rendered(t, view))
	report, _ := view.Report()
	found := report.Problems[0]
	// 🔴 The whole sentence rather than the words of it one at a time. Looked
	// for separately, the word line was found in the markup itself the day this
	// screen gained a table: a scrolling area draws a linearGradient for its
	// shadow, and a mutation that took the line out of every problem went
	// unnoticed. A drawing is text, so an assertion about it has to be about
	// what the screen SAYS rather than about letters that appear somewhere.
	wanted := found.Where + " (" + i18n.English.T(i18n.RecipeLine) + " " +
		strconv.Itoa(found.Line) + "): " + found.What
	if !strings.Contains(drawn, wanted) {
		t.Errorf("the drawn answer does not carry %q, so a person reading the screen "+
			"gets less than a person reading the command line.\n%s", wanted, drawn)
	}
}

// A clean recipe is approved together with what was read of it.
//
// A bare approval of a file the tool misunderstood is the one answer this
// product must never give, so the counts are part of the answer rather than
// decoration beside it.
func TestAnApprovalSaysWhatWasRead(t *testing.T) {
	view := here(t, real(t))
	view.SetRecipe(screens.CleanRecipe)

	drawn := rendered(t, view)
	if !strings.Contains(drawn, i18n.English.T(i18n.RecipeNoObjections)) {
		t.Fatalf("a usable recipe is not approved on screen.\n%s", drawn)
	}
	for _, wanted := range []string{i18n.English.T(i18n.RecipeEntity), i18n.English.T(i18n.RecipeField)} {
		if !strings.Contains(drawn, wanted) {
			t.Errorf("the approval does not say %q was read, so it is a bare approval.\n%s",
				wanted, drawn)
		}
	}
}

// rendered is what the screen draws, as text.
func rendered(t *testing.T, view *recipeview.View) string {
	t.Helper()
	window := test.NewWindow(view.Content())
	t.Cleanup(window.Close)
	window.Resize(fyne.NewSize(900, 700))
	return test.RenderToMarkup(window.Canvas())
}

// lineOf is which line of a recipe carries a piece of text, counting from one.
func lineOf(recipe, text string) int {
	for at, line := range strings.Split(recipe, "\n") {
		if strings.Contains(line, text) {
			return at + 1
		}
	}
	return 0
}

// A check that comes apart is drawn as a defect of this build, and not as a
// refusal of the recipe.
//
// 🔴 The screen already had somewhere a failure could have been put, and using
// it would have been wrong. A refusal here says that the recipe reads correctly
// and still cannot be run, which is a sentence about what somebody wrote. A
// panic is a sentence about us. This tool exists to argue against manifests
// that blame an application for damage the tool did, and a screen blaming a
// recipe for a defect of ours is the same dishonesty on the other surface.
func TestACheckThatComesApartIsDrawnAsADefectOfThisBuildRatherThanARefusal(t *testing.T) {
	crashing := background.Work{Read: func(_ func() func(), crashed func(error) func()) {
		crashed(fault.FromPanic("the checker came apart"))()
	}}
	view := recipeview.New(i18n.English, func(string, []byte) recipeview.Answer {
		return recipeview.Answer{}
	}, nil, crashing, nil)
	view.Content()

	view.SetRecipe(screens.CleanRecipe)

	// Four checks and every one of them is about words.
	drawn := spoken(rendered(t, view))
	if !strings.Contains(drawn, fault.Crashed) {
		t.Errorf("the screen does not say the build came apart, so somebody is left "+
			"looking at a window that answered nothing.\n%s", drawn)
	}
	if !strings.Contains(drawn, i18n.English.T(i18n.RecipeCrashed)) {
		t.Errorf("the screen names the defect and not what to do about it, which is the "+
			"one thing a person can actually do with it.\n%s", drawn)
	}
	if strings.Contains(drawn, i18n.English.T(i18n.RecipeRefused)) {
		t.Errorf("a defect of this build is drawn under the heading for a recipe that "+
			"cannot be run, which blames the person who wrote it.\n%s", drawn)
	}
	if strings.Contains(drawn, i18n.English.T(i18n.RecipeNoObjections)) {
		t.Errorf("a defect of this build is drawn beside an approval, so the screen says "+
			"the recipe is fine and that it came apart at the same time.\n%s", drawn)
	}
}

// The next check takes the crash off the screen, because what is drawn under a
// recipe has to be about the recipe that is there now.
func TestACheckThatFinishesTakesTheCrashOffTheScreen(t *testing.T) {
	crash := true
	work := background.Work{Read: func(read func() func(), crashed func(error) func()) {
		if crash {
			crashed(fault.FromPanic("the checker came apart"))()
			return
		}
		read()()
	}}
	view := recipeview.New(i18n.English, func(string, []byte) recipeview.Answer {
		return recipeview.Answer{}
	}, nil, work, nil)
	view.Content()

	view.SetRecipe(screens.CleanRecipe)
	crash = false
	view.SetRecipe(screens.CleanRecipe + "\n")

	if drawn := rendered(t, view); strings.Contains(drawn, fault.Crashed) {
		t.Errorf("a crash from an earlier check is still on screen under a recipe that has "+
			"been judged since.\n%s", drawn)
	}
}
