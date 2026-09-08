package paint_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/gui/assets"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	theme "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// The register closed from both sides.
//
// A constant naming a file that is not embedded reaches a panic the moment the
// screen holding it is drawn, which is a crash rather than a wrong pixel. A
// file nothing names is a kilobyte in the binary and a promise in the
// directory: somebody added it meaning to use it, and nothing says whether they
// forgot or changed their mind.
func TestEveryIconNameAndFileAnswerToEachOther(t *testing.T) {
	embedded := make(map[string]bool)
	for _, name := range assets.IconNames() {
		embedded[name] = true
	}
	named := make(map[string]bool, len(paint.IconNames))
	for _, name := range paint.IconNames {
		if named[name] {
			t.Errorf("the icon %q is named by two constants", name)
		}
		named[name] = true
		if !embedded[name] {
			t.Errorf("the constant for %q names no embedded file, so drawing it panics", name)
		}
	}
	for name := range embedded {
		if !named[name] {
			t.Errorf("the file %q is embedded and nothing names it, so it is a kilobyte nobody draws", name)
		}
	}
}

// The assumption the whole loader rests on, checked against every file rather
// than against the one that was opened while writing it.
//
// # Why this cannot be left to the render
//
// Both substitutions are silent when they miss. An icon whose colour was not
// replaced keeps currentColor, which the decoder turns into black, and black on
// this window is an invisible shape. An icon whose stroke was not replaced is
// simply the wrong weight beside its neighbours. Neither raises anything, and
// the second is the kind of thing an eye stops seeing after an hour.
func TestEveryEmbeddedIconCarriesWhatTheLoaderRewrites(t *testing.T) {
	for _, name := range assets.IconNames() {
		source := string(assets.Icon(name))
		if !strings.Contains(source, "currentColor") {
			t.Errorf("%s does not carry currentColor, so the loader cannot colour it and it would draw in whatever the file says", name)
		}
		if !strings.Contains(source, `stroke-width="2"`) {
			t.Errorf("%s does not carry the stroke width the loader rewrites, so its line will not match the icons beside it", name)
		}
	}
}

// The substitution actually happened, read off the resource the loader built.
//
// Asking whether the source strings are gone rather than whether the new ones
// are present, because a file could carry the new value already and pass a test
// written the other way round while the loader did nothing at all.
func TestTheLoaderLeavesNeitherSourceValueBehind(t *testing.T) {
	image := paint.Icon(paint.IconBolt, theme.TextAccent, theme.IconMedium)
	drawn := string(image.Resource.Content())

	if strings.Contains(drawn, "currentColor") {
		t.Error("the drawn icon still says currentColor, which the decoder turns into black on a dark window")
	}
	if strings.Contains(drawn, `stroke-width="2"`) {
		t.Error("the drawn icon still carries the stroke width of the file, so it does not follow the size it is drawn at")
	}

	accent := theme.Ink(theme.TextAccent)
	want := fmt.Sprintf("#%02x%02x%02x", accent.R, accent.G, accent.B)
	if !strings.Contains(drawn, want) {
		t.Errorf("the drawn icon does not carry %s, which is the colour of the token it was asked for", want)
	}
}

// The same icon asked for twice is the same resource, name and bytes.
//
// # Why this is a guard rather than an observation
//
// The first version of this loader numbered its resources with a counter, so
// the name of an icon depended on the order a screen happened to build its
// children in. The stored trees that guard these screens record resource names,
// so a screen would have differed from its stored copy for no reason a person
// could see, and the difference would have moved around between runs. That is
// rule 7 of this project, determinism, reaching a file name.
func TestTheSameIconTwiceIsTheSameResource(t *testing.T) {
	first := paint.Icon(paint.IconCopy, theme.TextSecondary, theme.IconSmall)
	second := paint.Icon(paint.IconCopy, theme.TextSecondary, theme.IconSmall)

	if first.Resource.Name() != second.Resource.Name() {
		t.Errorf("the same icon got two names, %q and %q", first.Resource.Name(), second.Resource.Name())
	}
	if string(first.Resource.Content()) != string(second.Resource.Content()) {
		t.Error("the same icon got two different bodies")
	}

	other := paint.Icon(paint.IconCopy, theme.TextPrimary, theme.IconSmall)
	if first.Resource.Name() == other.Resource.Name() {
		t.Error("two colours of one icon share a name, so the toolkit may hand back either of them")
	}
}

// The arithmetic the icons look like one family because of.
//
// A set drawn at three sizes with the stroke written in the file has a line
// twice as thick at the largest size as at the smallest. Holding the width on
// screen means the number written into the file has to RISE as the icon gets
// smaller, which is the property this checks rather than any single value.
func TestTheStrokeRisesAsTheIconGetsSmaller(t *testing.T) {
	small := paint.StrokeFor(theme.IconSmall)
	medium := paint.StrokeFor(theme.IconMedium)
	large := paint.StrokeFor(theme.IconLarge)

	if !(small >= medium && medium >= large) {
		t.Errorf("the stroke does not fall as the icon grows: %g at %g, %g at %g, %g at %g",
			small, theme.IconSmall, medium, theme.IconMedium, large, theme.IconLarge)
	}
	if small == large {
		t.Errorf("the smallest and the largest icon get the same stroke %g, so nothing is being held steady", small)
	}
}

// Both bounds are reached by sizes this window could ask for, and neither is
// crossed. A bound nothing reaches is a bound nobody can be sure works.
func TestTheStrokeStaysInsideItsBounds(t *testing.T) {
	for _, size := range []float32{theme.IconSmall, theme.IconMedium, theme.IconLarge} {
		stroke := paint.StrokeFor(size)
		if stroke < theme.IconStrokeFloor || stroke > theme.IconStrokeCeiling {
			t.Errorf("an icon of %g asks for a stroke of %g, outside the bounds %g to %g",
				size, stroke, theme.IconStrokeFloor, theme.IconStrokeCeiling)
		}
	}
	if paint.StrokeFor(theme.IconGrid*theme.IconGrid) != theme.IconStrokeFloor {
		t.Error("a very large icon should be pinned to the floor rather than drawn as a hairline")
	}
	if paint.StrokeFor(theme.HairlineThickness) != theme.IconStrokeCeiling {
		t.Error("a very small icon should be pinned to the ceiling rather than drawn as a blob")
	}
	if paint.StrokeFor(0) != theme.IconStrokeCeiling {
		t.Error("an icon of no size should answer rather than divide by nought")
	}
}

// An icon is drawn at the size it was asked for, which is the one thing a
// caller cannot check by looking at the file.
func TestAnIconIsDrawnAtTheSizeItWasAskedFor(t *testing.T) {
	image := paint.Icon(paint.IconShield, theme.StateSuccess, theme.IconLarge)
	if got := image.MinSize(); got.Width != theme.IconLarge || got.Height != theme.IconLarge {
		t.Errorf("an icon asked for at %g is %g by %g", theme.IconLarge, got.Width, got.Height)
	}
}
