package fynetheme

import (
	"image/color"
	"os"
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
func TestMain(m *testing.M) {
	fynetest.NewApp()
	os.Exit(m.Run())
}

// Every colour this adapter claims has to come from the measured palette.
//
// Without this the adapter could answer with anything and the contrast bridge
// would still be green, because the bridge compares the document against the
// palette and never asks what the window actually shows. The two checks are
// halves of one question and this is the half nobody had.
func TestEveryColourAnsweredHereComesFromTheMeasuredPalette(t *testing.T) {
	known := map[color.NRGBA]string{}
	for _, entry := range append([]palette.Color{palette.Background}, palette.Palette...) {
		known[entry.NRGBA()] = entry.Name
	}

	for name := range colourFor {
		got := Dark{}.Color(name, toolkit.VariantDark)
		nrgba, ok := got.(color.NRGBA)
		if !ok {
			t.Errorf("%s answered with something other than an opaque colour", name)
			continue
		}
		if _, found := known[nrgba]; !found {
			t.Errorf("%s answered with %v, which is in no measured entry", name, nrgba)
		}
	}
}

// A name this project has no opinion about has to fall through to the toolkit
// rather than to a guess of ours, and it must never come back invisible.
func TestAnUnclaimedColourFallsThroughAndIsNeverInvisible(t *testing.T) {
	got := Dark{}.Color(fyne.ThemeColorName("no.such.colour"), toolkit.VariantDark)
	_, _, _, alpha := got.RGBA()
	if alpha == 0 {
		t.Error("an unclaimed colour came back fully transparent, which renders as a hole")
	}
}

// Spacing comes from the closed scale, and a type size does not, because the
// scale says nothing about type and claiming otherwise would be a number with
// no measurement behind it.
func TestSpacingComesFromTheScaleAndTypeSizeDoesNot(t *testing.T) {
	steps := map[float32]bool{}
	for _, step := range palette.Scale {
		steps[step.Step] = true
	}
	for name := range sizeFor {
		if got := (Dark{}).Size(name); !steps[got] {
			t.Errorf("%s answered %v, which is not a step of the closed scale", name, got)
		}
	}

	ours := (Dark{}).Size(toolkit.SizeNameText)
	theirs := toolkit.DefaultTheme().Size(toolkit.SizeNameText)
	if ours != theirs {
		t.Errorf("the type size was invented here as %v instead of delegated as %v", ours, theirs)
	}
}

// The theme answers the same way whichever variant is asked for, because there
// is one theme. A light desktop getting the dark window is the decision working
// rather than a branch somebody forgot.
func TestBothVariantsAnswerTheSameBecauseThereIsOneTheme(t *testing.T) {
	for name := range colourFor {
		dark := Dark{}.Color(name, toolkit.VariantDark)
		light := Dark{}.Color(name, toolkit.VariantLight)
		if dark != light {
			t.Errorf("%s differs between variants, and this product has one theme", name)
		}
	}
}

// A font and an icon have to resolve to something, or every control renders
// blank and the cause is far from the symptom.
func TestFontsAndIconsResolve(t *testing.T) {
	if (Dark{}).Font(fyne.TextStyle{}) == nil {
		t.Error("no font resolved, so every label would be blank")
	}
	if (Dark{}).Icon(toolkit.IconNameConfirm) == nil {
		t.Error("no icon resolved")
	}
}

// Nothing a control can be painted with may equal the colour that says where
// the keyboard is.
//
// 🔴 This is the guard for a defect a person saw on a real window on
// 2026-08-18 and no test could: the toolkit marks focus by blending the focus
// colour into the background of the control, so a control whose background is
// already that colour shows nothing at all when it takes focus. The name of the
// open screen was painted in the primary colour, this theme answered for
// primary and for focus with one value, and pressing Tab onto that name changed
// not a single pixel.
//
// It compares what the theme answers rather than what the rendered tree calls
// it, and the difference is the whole point. The tree named those two
// rectangles fillColor="primary" and fillColor="focus", which reads like two
// colours and was one.
func TestNoBackgroundOfAControlIsTheColourOfFocus(t *testing.T) {
	focus := (Dark{}).Color(toolkit.ColorNameFocus, toolkit.VariantDark)
	backgrounds := []fyne.ThemeColorName{
		toolkit.ColorNamePrimary,
		toolkit.ColorNameButton,
		toolkit.ColorNameSelection,
		toolkit.ColorNameHover,
		toolkit.ColorNameBackground,
		toolkit.ColorNameInputBackground,
		toolkit.ColorNameMenuBackground,
		toolkit.ColorNameOverlayBackground,
	}
	for _, name := range backgrounds {
		if got := (Dark{}).Color(name, toolkit.VariantDark); got == focus {
			t.Errorf("a control painted with %s carries the focus colour already, so focus lands on it "+
				"and nothing on the screen changes", name)
		}
	}
}
