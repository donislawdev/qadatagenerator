package fynetheme

import (
	"image/color"
	"os"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	fynetest "fyne.io/fyne/v2/test"
	toolkit "fyne.io/fyne/v2/theme"

	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// The built in theme reads the running application to answer for the colours
// derived from the user's accent, so anything that delegates to it needs an
// application to exist. Measured rather than assumed: without this the
// fall through panics inside the toolkit rather than returning a colour, and
// the panic names the toolkit rather than the missing application.
//
// It also means the fall through is only safe after the application is
// created, which is the order the command in cmd/qadg-gui already uses.
//
// The tests below asked these questions of Dark, the theme of the first
// window, until it was deleted with DEC-253. They ask them of Visual now,
// because the questions are about the theme the window wears and not about
// either type.
func TestMain(m *testing.M) {
	fynetest.NewApp()
	os.Exit(m.Run())
}

// Every colour this adapter claims has to come from the measured register.
//
// Without this the adapter could answer with anything and the test of the
// register would still be green, because that test measures the tokens and
// never asks what the window is actually handed. The two checks are halves of
// one question and this is the half nobody else has.
func TestEveryColourAnsweredHereComesFromTheMeasuredPalette(t *testing.T) {
	known := map[color.NRGBA]string{}
	for _, token := range palette.Tokens {
		known[palette.Ink(token.Name)] = token.Name
	}

	for name := range tokenFor {
		got := Visual{}.Color(name, toolkit.VariantDark)
		nrgba, ok := got.(color.NRGBA)
		if !ok {
			t.Errorf("%s answered with something other than an opaque colour", name)
			continue
		}
		if _, found := known[nrgba]; !found {
			t.Errorf("%s answered with %v, which is the ink of no measured token", name, nrgba)
		}
	}
}

// 🔴 The toolkit's shadow is painted nowhere in this window, which is the look
// the owner chose on 2026-11-12 together with no theme of a control's own
// (B-370). Each name the window paints with nothing says what disappears, and
// none is also given a colour, which would leave one of the two answers dead.
//
// The canary is a name the window does paint, the colour of its text. Read the
// same way, it has to come back painted, or every check below would pass on a
// reading that sees nothing.
func TestTheShadowOfTheToolkitIsPaintedNowhere(t *testing.T) {
	if _, _, _, alpha := (Visual{}).Color(toolkit.ColorNameForeground, toolkit.VariantDark).RGBA(); alpha == 0 {
		t.Fatal("the colour of the text reads as painted with nothing, so this reading cannot tell the two apart")
	}
	if _, _, _, alpha := (Visual{}).Color(toolkit.ColorNameShadow, toolkit.VariantDark).RGBA(); alpha != 0 {
		t.Error("the toolkit's shadow is painted, so it comes back along every list that scrolls, " +
			"on the recipe editor and at the end of a field holding more than it shows")
	}
	for name, what := range unpainted {
		if _, _, _, alpha := (Visual{}).Color(name, toolkit.VariantDark).RGBA(); alpha != 0 {
			t.Errorf("%s is meant to be painted with nothing and is answered with a colour", name)
		}
		if strings.TrimSpace(what) == "" {
			t.Errorf("%s is painted with nothing and nothing says what that takes away", name)
		}
		if _, coloured := tokenFor[name]; coloured {
			t.Errorf("%s is both painted with nothing and given a token", name)
		}
	}
}

// A name this project has no opinion about has to fall through to the toolkit
// rather than to a guess of ours, and it must never come back invisible.
func TestAnUnclaimedColourFallsThroughAndIsNeverInvisible(t *testing.T) {
	got := Visual{}.Color(fyne.ThemeColorName("no.such.colour"), toolkit.VariantDark)
	_, _, _, alpha := got.RGBA()
	if alpha == 0 {
		t.Error("an unclaimed colour came back fully transparent, which renders as a hole")
	}
}

// A distance comes from the closed scale and a type size from the ladder, so
// neither is a number invented in the adapter.
//
// It asked Dark the opposite about type, that the size be the toolkit's own,
// because the ladder did not exist yet. Visual answers the body size of the
// ladder instead, which is what made the embedded faces readable at a size of
// ours, so the question turned round with the theme.
func TestSizesComeFromTheScaleAndTheLadder(t *testing.T) {
	steps := map[float32]bool{}
	for _, step := range palette.Scale {
		steps[step.Step] = true
	}
	for name := range stepFor {
		if got := (Visual{}).Size(name); !steps[got] {
			t.Errorf("%s answered %v, which is not a step of the closed scale", name, got)
		}
	}

	if got, want := (Visual{}).Size(toolkit.SizeNameText), palette.TypeSize(palette.TypeBody); got != want {
		t.Errorf("the text size is %v where the body rung of the ladder is %v", got, want)
	}
	for _, rung := range palette.TypeNames {
		if got, want := (Visual{}).Size(fyne.ThemeSizeName(rung)), palette.TypeSize(rung); got != want {
			t.Errorf("a label asking for %s gets %v where the ladder says %v", rung, got, want)
		}
	}
}

// The theme answers the same way whichever variant is asked for, because there
// is one theme. A light desktop getting the dark window is the decision working
// rather than a branch somebody forgot.
func TestBothVariantsAnswerTheSameBecauseThereIsOneTheme(t *testing.T) {
	for name := range tokenFor {
		dark := Visual{}.Color(name, toolkit.VariantDark)
		light := Visual{}.Color(name, toolkit.VariantLight)
		if dark != light {
			t.Errorf("%s differs between variants, and this product has one theme", name)
		}
	}
}

// A font and an icon have to resolve to something, or every control renders
// blank and the cause is far from the symptom.
func TestFontsAndIconsResolve(t *testing.T) {
	styles := []fyne.TextStyle{
		{}, {Bold: true}, {Italic: true}, {Bold: true, Italic: true},
		{Monospace: true}, {Monospace: true, Bold: true}, {Symbol: true},
	}
	for _, style := range styles {
		if (Visual{}).Font(style) == nil {
			t.Errorf("no font resolved for %+v, so every label in that style would be blank", style)
		}
	}
	for _, name := range []fyne.ThemeIconName{
		toolkit.IconNameConfirm, toolkit.IconNameArrowDropDown, toolkit.IconNameArrowDropUp,
	} {
		if (Visual{}).Icon(name) == nil {
			t.Errorf("no icon resolved for %s", name)
		}
	}
}

// Nothing a control can be painted with may equal the colour that says where
// the keyboard is.
//
// 🔴 This is the guard for a defect a person saw on a real window on
// 2026-08-18 and no test could: the toolkit marks focus by blending the focus
// colour into the background of the control, so a control whose background is
// already that colour shows nothing at all when it takes focus. The name of the
// open screen was painted in the primary colour, the theme of that day answered
// for primary and for focus with one value, and pressing Tab onto that name
// changed not a single pixel.
//
// It compares what the theme answers rather than what the rendered tree calls
// it, and the difference is the whole point. The tree named those two
// rectangles fillColor="primary" and fillColor="focus", which reads like two
// colours and was one.
func TestNoBackgroundOfAControlIsTheColourOfFocus(t *testing.T) {
	focus := (Visual{}).Color(toolkit.ColorNameFocus, toolkit.VariantDark)
	backgrounds := []fyne.ThemeColorName{
		toolkit.ColorNamePrimary,
		toolkit.ColorNameButton,
		toolkit.ColorNameSelection,
		toolkit.ColorNameHover,
		toolkit.ColorNameBackground,
		toolkit.ColorNameInputBackground,
		toolkit.ColorNameMenuBackground,
		toolkit.ColorNameOverlayBackground,
		toolkit.ColorNameHeaderBackground,
	}
	for _, name := range backgrounds {
		if got := (Visual{}).Color(name, toolkit.VariantDark); got == focus {
			t.Errorf("a control painted with %s carries the focus colour already, so focus lands on it "+
				"and nothing on the screen changes", name)
		}
	}
}
