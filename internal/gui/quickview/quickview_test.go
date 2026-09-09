package quickview

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/fynetheme"
	"github.com/donislawdev/qadatagenerator/internal/quick"
)

// TestMain starts an application, because a control laid out without one is a
// nil dereference rather than a quiet no operation: measuring a piece of text
// asks the running application how wide it is.
func TestMain(m *testing.M) {
	application := test.NewApp()
	// The theme of this product rather than the one the toolkit tests with,
	// which answers "color not defined" for every name of this palette.
	application.Settings().SetTheme(fynetheme.Current())
	os.Exit(m.Run())
}

// heldClipboard is a clipboard of this test, so that pressing Copy can be
// checked without touching the clipboard of the machine running it.
type heldClipboard struct{ content string }

func (h *heldClipboard) SetContent(text string) { h.content = text }

// TestChoosingAValueProducesOneAndCopyingItReachesTheClipboard is the claim the
// register of parity makes about this screen: that the window really does what
// the verb does.
//
// # Why the clipboard is checked rather than the button being pressed
//
// A clipboard cannot be seen. A button that silently succeeded and a button
// that silently did nothing look the same on a screen and in a stored tree, so
// the only thing that tells them apart is asking where the value went.
func TestChoosingAValueProducesOneAndCopyingItReachesTheClipboard(t *testing.T) {
	clipboard := &heldClipboard{}
	view := New(i18n.English, clipboard)

	// 🔴 What the screen OPENS on, asked before anything is touched, because it
	// is the one state no drawing ever shows and the first one a person sees.
	//
	// This used to assert the opposite: that nothing is shown before anything is
	// chosen, on the argument that a screen producing a value unasked answers a
	// question nobody put. That argument lost to a measurement on 2026-09-08.
	// The window opens on this screen, so an empty form was the first thing
	// anybody saw of this product, and at the size the window is kept it was
	// 76.9 per cent black. A screen whose whole promise is a value in under five
	// seconds can afford to have produced one already.
	//
	// It is ten values rather than one, and of the kind named by opensWith,
	// both by measurement rather than by taste. Nothing is read from a disk and
	// nothing is written, so this is not the surface acting on its own that
	// DEC-35 forbids.
	if got := len(strings.Split(view.Value(), "\n")); got != 10 {
		t.Errorf("the window opens showing %d value(s) where it is meant to open on ten, "+
			"which is the state a person meets first", got)
	}
	if !strings.HasPrefix(view.command.Text, "qadg quick") ||
		!strings.Contains(view.command.Text, opensWith) {
		t.Errorf("the window opens showing %q, which does not reproduce the values beside it",
			view.command.Text)
	}

	view.UseSeed("for-the-test")
	view.Choose("pesel")

	// One value from here down, so that what Copy carries is a single thing
	// and this test goes on being about the clipboard.
	view.ChooseCount("1")
	produced := view.Value()
	if produced == "" {
		t.Fatal("choosing a value produced nothing, so this screen needs a button the design says it must not have")
	}

	// The same request through the mode, which is the one oracle both surfaces
	// answer to. A second implementation on this screen would pass a test that
	// only asked whether something appeared.
	expected, err := quick.Run(quick.Request{
		Generator: "pesel", Locale: quick.DefaultLocale, Seed: "for-the-test",
		Agrees: map[string]string{}, Count: 1,
	})
	if err != nil {
		t.Fatalf("the mode refused the same request this screen made: %v", err)
	}
	if produced != strings.Join(expected, "\n") {
		t.Errorf("the screen shows %q where the mode produces %q for the same request",
			produced, expected)
	}

	view.putInClipboard()
	if clipboard.content != produced {
		t.Errorf("Copy put %q in the clipboard while %q was on screen",
			clipboard.content, produced)
	}
	if view.told.Text != i18n.English.T(i18n.QuickCopied) {
		t.Errorf("Copy said %q, where a clipboard nobody can see needs the screen to say it worked",
			view.told.Text)
	}
}

// TestEveryControlOnTheQuickScreenReachesTheValue holds the other half of the
// parity claim: that each control is connected to what it appears to control.
//
// # Why every control is moved rather than the code being read
//
// A control wired to nothing looks exactly like a control wired to something,
// in the source and in a stored tree alike. What tells them apart is changing
// it and seeing the value move, and this project has already paid for the
// difference on the screen next door.
func TestEveryControlOnTheQuickScreenReachesTheValue(t *testing.T) {
	view := New(i18n.English, &heldClipboard{})
	view.UseSeed("for-the-test")
	view.Choose("pesel")
	start := view.Value()
	if start == "" {
		t.Fatal("nothing was produced, so nothing below proves anything")
	}

	// 🔴 Each control is moved between two CONTRASTING values rather than from
	// nothing to something, and the first version of this test was red by
	// accident for want of that. An unbound gender is drawn, and the tenth
	// digit of a PESEL carries the gender in its parity, so going from unbound
	// to female leaves the value alone half the time. Female against male
	// cannot: the two are opposite parities by construction.
	moves := []struct {
		named      string
		one, other func()
	}{
		{"the seed", func() { view.seedBox.SetText("one-seed") },
			func() { view.seedBox.SetText("another-seed") }},
		{"the gender", func() { view.ChooseGender("female") },
			func() { view.ChooseGender("male") }},
		{"the date of birth", func() { view.SetBorn("1950-01-31") },
			func() { view.SetBorn("2001-12-01") }},
	}
	for _, move := range moves {
		move.one()
		first := view.Value()
		move.other()
		if view.Value() == first {
			t.Errorf("%s was changed and the value did not move, so that control reaches nothing",
				move.named)
		}
	}

	// The locale is checked apart from the three above, because what it moves
	// is not only the value: it decides which of the two fields are on screen
	// at all, and a locale wired to the list alone would look right here.
	view.Choose("last_name")
	view.UseLocale("en-US")
	if view.Shows(quick.RoleGender) {
		t.Error("a surname in English takes no account of a gender and the field is on screen anyway, " +
			"which promises an influence it does not have")
	}
	view.UseLocale("pl-PL")
	if !view.Shows(quick.RoleGender) {
		t.Error("a surname in Polish takes account of a gender and the field is not on screen, " +
			"so the screen hides a choice that would work")
	}

	// 🔴 The count is checked apart from the three above, and it has to be:
	// every one of them moves the VALUE, and this one does not. It moves how
	// many there are, and the test written by pattern would have asked whether
	// the value changed, found that it had not, and reported a control that
	// reaches nothing while it reached everything it was for.
	view.Choose("pesel")
	for _, howMany := range CountChoices() {
		view.ChooseCount(howMany)
		wanted, err := strconv.Atoi(howMany)
		if err != nil {
			t.Fatalf("this screen offers %q as a number of values and it is not one", howMany)
		}
		got := len(strings.Split(view.Value(), "\n"))
		if got != wanted {
			t.Errorf("%s was chosen and %d value(s) are on screen, so the list of how many "+
				"reaches the screen without deciding what it shows", howMany, got)
		}
	}
}

// 🔴 TestAskingForMoreValuesLeavesTheOnesAlreadyThereAlone holds the property
// that makes a count safe to change while somebody is reading.
//
// The values are addressed by row, counting up from the first, so asking for
// more can only ADD to what was there. A mode that drew them some other way
// would reshuffle everything on screen each time somebody asked for one more,
// and the person reading the value they came for would watch it move.
//
// This is the same property the guard beside the mode holds against a recipe,
// asked here of the screen: it is the one that decides whether the address is
// counted or invented.
func TestAskingForMoreValuesLeavesTheOnesAlreadyThereAlone(t *testing.T) {
	view := New(i18n.English, &heldClipboard{})
	view.UseSeed("for-the-test")
	view.Choose("pesel")

	view.ChooseCount("1")
	one := strings.Split(view.Value(), "\n")

	view.ChooseCount("10")
	ten := strings.Split(view.Value(), "\n")

	if len(ten) <= len(one) {
		t.Fatalf("asking for ten gave %d values against %d for one", len(ten), len(one))
	}
	if ten[0] != one[0] {
		t.Errorf("the first value was %q when one was asked for and %q when ten were, so "+
			"asking for more reshuffles what somebody is already reading", one[0], ten[0])
	}
}

// TestManyValuesSayTheyMayRepeatAndOneDoesNot holds a limit this screen has to
// state, and holds it in both directions.
//
// The values are drawn rather than dealt, so a generator with a short list of
// data behind it repeats, and somebody counting on fifty distinct values would
// find that out from the data instead. Saying so beside one value would be the
// other failure: a warning about something that cannot happen is how a person
// learns to stop reading them.
func TestManyValuesSayTheyMayRepeatAndOneDoesNot(t *testing.T) {
	view := New(i18n.English, &heldClipboard{})
	view.UseSeed("for-the-test")
	view.Choose("pesel")

	view.ChooseCount("1")
	if view.told.Text == i18n.English.T(i18n.QuickMayRepeat) {
		t.Error("one value is said to possibly repeat, and one value has nothing to repeat with")
	}

	view.ChooseCount("10")
	if view.told.Text != i18n.English.T(i18n.QuickMayRepeat) {
		t.Errorf("ten values are on screen and the limit on them is not said. It says %q",
			view.told.Text)
	}
}

// A refusal is on the screen, and the sentence for a screen nobody has used is
// not on it at the same time.
//
// 🔴 WRITTEN AFTER THE DEFECT IT GUARDS, found on 2026-09-08 by measuring
// emptiness rather than by any test. What this screen says lived inside the
// block holding the value, and that block is hidden whenever there is no value.
// A refusal is exactly the case where there is no value, so every refusal this
// screen can give was hidden by the call that hid the empty box beside it, and
// what stood there instead told somebody to choose a kind they had chosen two
// lines above. Both refusals rendered that way and every test here was green,
// because a test reads the text of a label and a person sees whether it is
// drawn.
//
// It asks about VISIBILITY rather than about the text, because the text was
// right the whole time. Both refusals are covered rather than one: one comes
// from this screen judging what was typed, the other from the mode judging what
// this build carries, and they travel by different paths to the same label.
func TestARefusalIsShownRatherThanHiddenWithTheValue(t *testing.T) {
	for _, refusal := range []struct {
		name      string
		generator string
		born      string
	}{
		{"a date this screen cannot read", "pesel", "05-05-1980"},
		{"a value this build has no data for", "city", ""},
	} {
		t.Run(refusal.name, func(t *testing.T) {
			view := New(i18n.English, &heldClipboard{})
			view.UseSeed("for-the-test")
			view.Choose(refusal.generator)
			if refusal.born != "" {
				view.SetBorn(refusal.born)
			}

			if view.told.Text == "" {
				t.Fatal("nothing was said about a request that was turned down")
			}
			if !view.told.Visible() {
				t.Errorf("the screen says %q and does not show it", view.told.Text)
			}
			// The sentence for a screen nobody has chosen anything on used to be
			// checked here too, and it is gone rather than dropped: this screen
			// chooses a kind while it is built, so that state cannot be reached
			// and the line describing it was removed with it.
			//
			// 🔴 This asked whether an empty value box was HIDDEN until
			// 2026-09-09, and there is no such box any more. The card is built
			// from the values, so a refusal cannot leave an empty frame behind
			// to be hidden. The question that survives the change is the one it
			// was always about: a refusal must not stand beside values.
			if view.Value() != "" {
				t.Errorf("the request was turned down and %d value(s) are on screen beside the refusal",
					len(view.values))
			}
		})
	}
}

// TestAFieldIsShownOnlyBesideAValueThatReadsIt holds the rule the design of this
// screen states, against the measurement rather than against the sentence.
//
// 🔴 The design said one generator reads a gender. Measured on 2026-09-05 there
// are three, one of them only in Polish, so a screen following the sentence
// would have hidden the gender beside a first name, which is the most obvious
// use this screen has.
func TestAFieldIsShownOnlyBesideAValueThatReadsIt(t *testing.T) {
	view := New(i18n.English, &heldClipboard{})
	view.UseSeed("for-the-test")

	for _, generator := range quick.Generators(quick.DefaultLocale) {
		view.Choose(generator)
		for _, role := range []string{quick.RoleGender, quick.RoleBirthDate} {
			reads := quick.Reads(generator, role, quick.DefaultLocale)
			if view.Shows(role) != reads {
				t.Errorf("beside %s the field for %s is shown %v where the mode says it is read %v",
					generator, role, view.Shows(role), reads)
			}
		}
	}
}

// The line this screen offers reproduces the value it is showing.
//
// 🔴 THE SEED IS WHAT THIS IS ABOUT. Without one the command line draws a
// fresh seed and says so on the log channel, so a line copied from here
// without the seed would hand somebody a DIFFERENT value from the one they
// were looking at when they copied it. A bridge between two surfaces that
// quietly changes the answer on the way across is worse than no bridge, and
// nothing else here would notice: the line looks right, runs, and produces a
// perfectly good value that is not the one on screen.
func TestTheCommandOfferedReproducesTheValueOnScreen(t *testing.T) {
	view := New(i18n.English, &heldClipboard{})
	view.UseSeed("for-the-test")
	view.Choose("pesel")
	view.ChooseGender("female")
	view.SetBorn("1980-05-05")

	line := view.command.Text
	for _, wanted := range []string{
		"qadg quick", "pesel", "--seed", "for-the-test",
		"--gender", "female", "--birth-date", "1980-05-05",
	} {
		if !strings.Contains(line, wanted) {
			t.Errorf("the line offered is %q and it does not name %q, so pasting it does not "+
				"produce what this screen is showing", line, wanted)
		}
	}
}

// A setting nobody touched is not written into the line.
//
// A flag carrying an empty value is not the same request as no flag at all,
// and the command line refuses the first. Writing them out anyway would hand
// somebody a line their own tool turns down.
func TestASettingLeftAloneIsNotWrittenIntoTheCommand(t *testing.T) {
	view := New(i18n.English, &heldClipboard{})
	view.UseSeed("for-the-test")
	view.Choose("last_name")

	if line := view.command.Text; strings.Contains(line, "--gender") ||
		strings.Contains(line, "--birth-date") {
		t.Errorf("the line offered is %q and it names a setting this value takes no account "+
			"of, which is a flag the command line refuses for this generator", line)
	}
}
