package quickview

import (
	"os"
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
	application.Settings().SetTheme(fynetheme.Dark{})
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

	// Nothing is chosen, so there is nothing to show. A screen that produced a
	// value before being asked would be answering a question nobody put.
	if view.Value() != "" {
		t.Errorf("the screen shows %q before anything was chosen", view.Value())
	}

	view.UseSeed("for-the-test")
	view.Choose("pesel")

	produced := view.Value()
	if produced == "" {
		t.Fatal("choosing a value produced nothing, so this screen needs a button the design says it must not have")
	}

	// The same request through the mode, which is the one oracle both surfaces
	// answer to. A second implementation on this screen would pass a test that
	// only asked whether something appeared.
	expected, err := quick.Run(quick.Request{
		Generator: "pesel", Locale: quick.DefaultLocale, Seed: "for-the-test",
		Agrees: map[string]string{},
	})
	if err != nil {
		t.Fatalf("the mode refused the same request this screen made: %v", err)
	}
	if produced != expected {
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
