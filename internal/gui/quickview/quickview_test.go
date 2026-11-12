package quickview

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/commandline"
	"github.com/donislawdev/qadatagenerator/internal/gui/fynetheme"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
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

// BenchmarkTheCardOfValues is what the ceiling on this screen is measured
// against, and it is here rather than in a comment because a constant that says
// it was measured has to be re-measurable.
//
// What it measures is the half that is NOT cheap. Producing is: a hundred
// thousand PESELs leave the command line in 402 ms including starting the
// process. Drawing them is another matter - every value is a row of several
// canvas objects built on the thread that draws, and nothing hands control back
// to the window while that happens.
//
// Run it as:
//
//	go test ./internal/gui/quickview/ -run x -bench Card -benchtime 20x
func BenchmarkTheCardOfValues(b *testing.B) {
	for _, howMany := range []int{500, 5000, 20000, 50000} {
		values := make([]string, howMany)
		for at := range values {
			values[at] = "66685370" + strconv.Itoa(at%10)
		}
		view := New(i18n.English, &heldClipboard{})
		b.Run(strconv.Itoa(howMany), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				card := view.valuesCard(values)
				// MinSize forces the layout to happen, which is where the cost
				// is. Building the objects without measuring them would time
				// the cheap half and call it the answer.
				card.MinSize()
			}
		})
	}
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
	// 🔴 Asked of the BUTTON rather than of the sentence above the card, which
	// is where this went until the owner reported on 2026-09-09 that they never
	// found it. The distance was the fault, so the distance is what the guard is
	// about, and since 2026-10-21 the button says it itself.
	if !view.copy.Confirming() || view.copy.Confirmed != i18n.English.T(i18n.QuickCopied) {
		t.Errorf("Copy all does not say %q after filling the clipboard, where a clipboard "+
			"nobody can see needs the screen to say it worked", i18n.English.T(i18n.QuickCopied))
	}
	// And it did not go into the sentence above the card as well. One label
	// serving both is how copying a set of values silently deleted the warning
	// that values may repeat - so the confirmation has to be absent from there,
	// not merely present beside the button.
	if strings.Contains(view.told.Text, i18n.English.T(i18n.QuickCopied)) {
		t.Errorf("the confirmation is also in the sentence above the card: %q", view.told.Text)
	}
}

// 🔴 Two confirmations never stand at once.
//
// Each button says a DIFFERENT thing was copied - a set of values, or a
// command line. Both lit at the same time would say the clipboard holds two
// things, and the one it actually holds is whichever button was pressed last.
func TestOnlyTheLastThingCopiedSaysSo(t *testing.T) {
	view := New(i18n.English, &heldClipboard{})
	view.Choose("pesel")

	view.putInClipboard()
	view.putCommandInClipboard()
	if view.copy.Confirming() {
		t.Error("Copy all still says it copied after the command was")
	}
	if !view.copyCommand.Confirming() {
		t.Error("copying the command said nothing on the button that did it")
	}

	// A new set of values means the clipboard no longer holds what is on
	// screen, so neither button may go on claiming it does.
	view.putInClipboard()
	view.Generate()
	if view.copy.Confirming() || view.copyCommand.Confirming() {
		t.Errorf("a fresh run left a confirmation standing: values %v, command %v",
			view.copy.Confirming(), view.copyCommand.Confirming())
	}
}

// 🔴 The guard for DEC-235, and it asks about the DIFFERENCE rather than about
// the button.
//
// Asking only "does Generate produce" would stay green with the old behaviour
// still in place, because a screen that produces on every keystroke also
// produces when a button is pressed. The two halves have to be checked apart:
// typing alone must NOT move the value, and pressing alone must.
//
// What this stops coming back: the field produced on every keystroke until
// 2026-09-09, so a thirty two character seed regenerated the set thirty two
// times. That is only odd until the count becomes a field somebody types a
// number into, and then it is a wall of work on the thread that draws.
func TestATypedFieldWaitsForTheButtonAndAChoiceDoesNot(t *testing.T) {
	view := New(i18n.English, &heldClipboard{})
	view.UseSeed("for-the-test")
	view.Choose("pesel")

	before := view.Value()
	if before == "" {
		t.Fatal("nothing was produced, so nothing below proves anything")
	}

	view.TypeSeed("typed-and-not-applied")
	if view.Value() != before {
		t.Error("the value moved while somebody was still typing the seed, so the field " +
			"produces on every keystroke and the button beside it decides nothing")
	}

	view.Generate()
	if view.Value() == before {
		t.Error("the seed was typed and the button pressed and the value did not move, " +
			"so what a person types reaches nothing")
	}

	// The other half of the decision: a CHOICE still produces at once, because
	// one tap and a result is the promise this verb makes in its first line.
	chosen := view.Value()
	view.Choose("nip")
	if view.Value() == chosen {
		t.Error("a kind was chosen and nothing was produced, so the screen now asks for " +
			"two acts where DEC-213 promised one")
	}
}

// 🔴 A field that is not on screen is not part of the request.
//
// The owner chose a gender for a first name and then asked for an ISBN, and
// the screen refused with "That value takes no account of what was chosen
// beside it". The gender field had gone, because an ISBN reads no gender, and
// the choice made in it was still sent, so the refusal named a field nobody
// could see or clear. The command at the foot carried the same flag, which the
// command line refuses too.
//
// The way back is checked as well: the choice is kept while its field is
// hidden, so returning to a kind that reads it applies it again rather than
// asking for it twice.
func TestAFieldTheChosenKindDoesNotReadIsNotSent(t *testing.T) {
	view := New(i18n.English, &heldClipboard{})
	view.UseSeed("for-the-test")
	view.Choose("pesel")
	view.ChooseGender("female")
	view.SetBorn("1990-05-17")
	if view.Value() == "" {
		t.Fatalf("a PESEL with a gender and a date produced nothing, so nothing below "+
			"proves anything: %q", view.Told())
	}

	view.Choose("isbn")
	if view.Value() == "" {
		t.Errorf("an ISBN was refused after a gender and a date were chosen for a PESEL, "+
			"with fields that are no longer on screen: %q", view.Told())
	}
	for _, flag := range []string{commandline.Gender, commandline.BirthDate} {
		if strings.Contains(view.command.Text, flag) {
			t.Errorf("the command for an ISBN carries %s, which the command line refuses: %q",
				flag, view.command.Text)
		}
	}

	view.Choose("pesel")
	if !strings.Contains(view.command.Text, commandline.Gender+" female") {
		t.Errorf("back on a PESEL the gender chosen before is no longer applied: %q",
			view.command.Text)
	}
}

// TestTheCountFieldShowsARefusalWhileItIsTyped holds B-363: the count field is
// marked as wrong the moment it holds a number Generate would refuse, and not
// only after the press.
//
// 🔴 The mark and the press are held to ONE rule rather than each to a list of
// numbers written here. Every number below is typed, judged by the mark, then
// pressed, and the press has to refuse exactly the ones the mark called wrong.
// A field that turned red by a rule of its own would drift from the refusal the
// first time either moved, and a person would be told no by one and yes by the
// other.
//
// An empty field is not marked, which is the one place the two differ on
// purpose: a person clearing the field to type another number is half way
// through, and a field going red under a deleted digit is the screen arguing
// mid sentence, the reason the date of birth waits for Enter.
func TestTheCountFieldShowsARefusalWhileItIsTyped(t *testing.T) {
	view := New(i18n.English, &heldClipboard{})
	view.UseSeed("for-the-test")
	view.Choose("pesel")

	view.howMany.SetText("")
	if view.howMany.Invalid() {
		t.Error("an empty count field is marked as wrong while somebody may be about to type a number")
	}
	ceiling := MostShown()
	for _, typed := range []string{
		"0", "00", "1", "007", "10", strconv.Itoa(ceiling - 1), strconv.Itoa(ceiling),
		strconv.Itoa(ceiling + 1), strings.Repeat("9", countDigits),
	} {
		shown := view.Value()
		view.howMany.SetText(typed)
		marked := view.howMany.Invalid()
		if view.Value() != shown {
			t.Errorf("typing %q produced values before Generate was pressed (DEC-235)", typed)
		}
		view.Generate()
		refused := view.Value() == ""
		if marked != refused {
			t.Errorf("the count field holding %q is marked wrong %v and Generate refuses it %v, "+
				"so the mark and the press disagree", typed, marked, refused)
		}
	}
}

// TestTheLocaleIsOfferedOnlyBesideAKindItChanges holds B-367: the choice of
// locale stands beside a kind only where another locale gives something else,
// which is asked of the mode rather than listed here.
//
// 🔴 The second half is the one a screen gets wrong while looking right. A
// locale chosen beside a first name and then hidden beside a PESEL must leave
// the request and the line at the foot, or it is a setting nobody can see,
// which is the shape B-365 had. Coming back to a kind that reads it applies it
// again, the way a hidden gender is applied again.
func TestTheLocaleIsOfferedOnlyBesideAKindItChanges(t *testing.T) {
	view := New(i18n.English, &heldClipboard{})
	view.UseSeed("for-the-test")
	for _, generator := range quick.Generators(quick.DefaultLocale) {
		view.Choose(generator)
		if shown, changes := view.Shows(settingLocale), quick.DependsOnLocale(generator); shown != changes {
			t.Errorf("beside %s the choice of locale is shown %v where the mode says the locale "+
				"changes the value %v", generator, shown, changes)
		}
	}

	view.Choose("first_name")
	view.UseLocale("pl-PL")
	view.Choose("pesel")
	if asked := view.request("pesel", 1).Locale; asked != "" {
		t.Errorf("a PESEL is asked for in %q while the screen shows no locale", asked)
	}
	if strings.Contains(view.command.Text, commandline.Locale) {
		t.Errorf("the command for a PESEL names a locale the screen does not show: %q",
			view.command.Text)
	}

	view.Choose("first_name")
	if !strings.Contains(view.command.Text, commandline.Locale+" pl-PL") {
		t.Errorf("back on a first name the locale chosen before is no longer applied: %q",
			view.command.Text)
	}
}

// 🔴 The sample the count field is sized from has to be the shape of the
// largest count the screen takes, or the field is measured against a number
// nobody can type into it.
//
// It replaces the same guard for the seed, whose object stopped existing: the
// seed field takes the room between the fields and the button now, because
// giving it a width of its own made it draw under that button.
func TestTheCountFieldIsSizedForTheLargestCountItTakes(t *testing.T) {
	if len(countSample) != countDigits {
		t.Errorf("the field is sized for %q while it takes %d digits, so the "+
			"widest number it accepts may not fit in it", countSample, countDigits)
	}
	if countDigits <= len(strconv.Itoa(quick.MostShownAtOnce)) {
		t.Errorf("the field takes %d digits and the ceiling has as many, so a number over "+
			"it is cut to fit without a word rather than refused", countDigits)
	}
}

// 🔴 A count over the ceiling is refused in words, not cut to fit.
//
// The owner typed a million on 2026-10-21 and got ten thousand values: the field
// dropped every digit past the fifth and the screen drew what was left.
func TestACountOverTheCeilingIsRefusedRatherThanCut(t *testing.T) {
	view := New(i18n.English, &heldClipboard{})
	typed := strconv.Itoa(quick.MostShownAtOnce) + strconv.Itoa(0) + strconv.Itoa(0)
	test.Type(view.howMany, typed)
	view.Choose("pesel")
	view.Generate()

	if got := len(strings.Split(view.Value(), "\n")); view.Value() != "" && got > quick.MostShownAtOnce {
		t.Errorf("the screen drew %d values, over the ceiling of %d", got, quick.MostShownAtOnce)
	}
	if view.Value() != "" {
		t.Errorf("a count over the ceiling produced values, where it is refused: %q typed as %q",
			typed, view.howMany.Text)
	}
	if !strings.Contains(view.told.Text, strconv.Itoa(quick.MostShownAtOnce)) {
		t.Errorf("the refusal does not name the ceiling, so nobody learns what the screen takes: %q",
			view.told.Text)
	}
}

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
		// Typed AND applied, which is two acts since DEC-235. The guard below
		// is the one that watches the halves apart.
		{"the seed", func() { view.UseSeed("one-seed") },
			func() { view.UseSeed("another-seed") }},
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
	// 🔴 The numbers below are not a list the screen offers, because since the
	// field replaced the chooser there is no such list. They are the EDGES of
	// what it takes: one, a middling number, and the ceiling itself - which is
	// the value most likely to be off by one and the one no chooser ever
	// reached.
	view.Choose("pesel")
	for _, howMany := range []string{"1", "7", strconv.Itoa(MostShown())} {
		view.ChooseCount(howMany)
		wanted, err := strconv.Atoi(howMany)
		if err != nil {
			t.Fatalf("this test asks for %q values and that is not a number", howMany)
		}
		got := len(strings.Split(view.Value(), "\n"))
		if got != wanted {
			t.Errorf("%s was asked for and %d value(s) are on screen, so how many "+
				"reaches the screen without deciding what it shows", howMany, got)
		}
	}
}

// 🔴 The guard for the ceiling, and it asks about the EDGE rather than about a
// big number.
//
// A test asking for a million would pass against any ceiling at all, including
// one written by accident as ten. Asking for the ceiling and for one more than
// the ceiling is the only pair that pins the number itself, and the refusal has
// to NAME the command line - a screen that stops at no makes its limit the
// person's problem.
func TestTheCeilingIsRefusedByOneAndSaysWhereToGetMore(t *testing.T) {
	view := New(i18n.English, &heldClipboard{})
	view.UseSeed("for-the-test")
	view.Choose("pesel")

	view.ChooseCount(strconv.Itoa(MostShown()))
	if got := len(strings.Split(view.Value(), "\n")); got != MostShown() {
		t.Fatalf("the ceiling itself was refused: asked for %d and got %d",
			MostShown(), got)
	}

	view.ChooseCount(strconv.Itoa(MostShown() + 1))
	if view.Value() != "" {
		t.Errorf("one more than the ceiling produced %d value(s), so the ceiling is not "+
			"where the constant says it is", len(strings.Split(view.Value(), "\n")))
	}
	said := view.Told()
	if !strings.Contains(said, strconv.Itoa(MostShown())) {
		t.Errorf("the refusal is %q and does not say what the limit IS, so a person "+
			"has to guess how far down to count", said)
	}
	if !strings.Contains(strings.ToLower(said), "command line") {
		t.Errorf("the refusal is %q and does not point at the line under it, which is "+
			"the thing that has no limit", said)
	}
}

// A count that never became a number is refused by its own sentence rather than
// by the one about asking for fewer than one value.
func TestTextThatIsNotANumberIsRefusedAsAShape(t *testing.T) {
	view := New(i18n.English, &heldClipboard{})
	view.UseSeed("for-the-test")
	view.Choose("pesel")

	view.ChooseCount("ten")
	if view.Value() != "" {
		t.Error("a count of \"ten\" produced values, so the field takes text as a number")
	}
	if said := view.Told(); said != i18n.English.T(i18n.QuickRefusedCountShape) {
		t.Errorf("the refusal is %q where it should be the one about the shape of a "+
			"number, so the screen blames the wrong thing", said)
	}

	// 🔴 The trap DEC-170 cost this project once: a leading nought must be TEN,
	// not eight. It is checked on this surface as well as on the flag, because
	// the two read the text with the same call and a guard on one of them would
	// not notice the day that stopped being true.
	view.ChooseCount("010")
	if got := len(strings.Split(view.Value(), "\n")); got != 10 {
		t.Errorf("a count of 010 produced %d values, so the field reads a leading "+
			"nought as octal", got)
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
// right the whole time. Both refusals are covered rather than one: one is about
// what was typed, the other about what this build carries, and a screen that
// shows one of them may still hide the other.
func TestARefusalIsShownRatherThanHiddenWithTheValue(t *testing.T) {
	for _, refusal := range []struct {
		name      string
		generator string
		locale    string
		born      string
	}{
		{"a date this screen cannot read", "pesel", "", "05-05-1980"},
		// The postal code of pl-PL since 2026-11-12, when the city of en-US got
		// its data (DEC-278) and stopped being a refusal (DEC-92 holds this one).
		{"a value this build has no data for", "postal_code", "pl-PL", ""},
	} {
		t.Run(refusal.name, func(t *testing.T) {
			view := New(i18n.English, &heldClipboard{})
			view.UseSeed("for-the-test")
			if refusal.locale != "" {
				view.UseLocale(refusal.locale)
			}
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

// 🔴 One value out of many, which is what the owner asked for on 2026-09-09:
// "I cannot copy a single value from values".
//
// The guard asks for the value of the row that was clicked rather than for any
// value, because a row wired to the wrong closure is the failure this shape
// invites: the rows are built in a loop and a captured variable that moved on
// would give every row the last value in the set.
func TestClickingOneValueCopiesThatValueAndNotTheSet(t *testing.T) {
	clipboard := &heldClipboard{}
	view := New(i18n.English, clipboard)
	view.ChooseCount("4")
	view.UseSeed("for-the-test")
	view.Choose("pesel")

	values := strings.Split(view.Value(), "\n")
	if len(values) != 4 {
		t.Fatalf("the screen produced %d values, so this cannot ask about the third", len(values))
	}

	rows := tapRowsOf(view.Content())
	if len(rows) != len(values) {
		t.Fatalf("the card holds %d rows for %d values", len(rows), len(values))
	}
	rows[2].Tapped(nil)

	if clipboard.content != values[2] {
		t.Errorf("clicking the third row copied %q, where the third value is %q",
			clipboard.content, values[2])
	}
	if clipboard.content == view.Value() {
		t.Error("clicking one row copied the whole set")
	}
	if !rows[2].Confirmed() {
		t.Error("a clipboard nobody can see was filled without the row saying so")
	}
	if rows[1].Confirmed() || view.copy.Confirming() {
		t.Error("something other than the row that was clicked says it was copied")
	}
}

// tapRowsOf finds the rows of the card, in the order they are drawn.
func tapRowsOf(object fyne.CanvasObject) []*parts.TapRow {
	var found []*parts.TapRow
	var walk func(fyne.CanvasObject)
	walk = func(each fyne.CanvasObject) {
		switch shape := each.(type) {
		case *parts.TapRow:
			found = append(found, shape)
		case *fyne.Container:
			for _, child := range shape.Objects {
				walk(child)
			}
		case fyne.Widget:
			// 🔴 Walked through the renderer, because a card is a widget and a
			// widget hides its children behind one. A walk that only follows
			// containers reports an empty card and reads like a screen that
			// drew nothing.
			for _, child := range test.WidgetRenderer(shape).Objects() {
				walk(child)
			}
		}
	}
	walk(object)
	return found
}
