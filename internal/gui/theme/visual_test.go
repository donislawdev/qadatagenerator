package theme_test

import (
	"math"
	"testing"

	theme "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// The contrast of a pair, by the formula WCAG 2.2 states.
//
// # Why it is written here rather than taken from tools/kontrast.py
//
// That script measures the same register a second time and runs when somebody
// or the gate runs it. This ladder is measured on every `go test`, which is the
// difference between a rule with a guard and a rule with a habit. The two
// implementations are also a second hand on the same arithmetic, and the pair
// has caught a wrong number in this project before.
func luminance(c struct{ R, G, B uint8 }) float64 {
	channel := func(v uint8) float64 {
		f := float64(v) / 255
		if f <= 0.03928 {
			return f / 12.92
		}
		return math.Pow((f+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(c.R) + 0.7152*channel(c.G) + 0.0722*channel(c.B)
}

func contrast(a, b theme.Shade) float64 {
	first := luminance(struct{ R, G, B uint8 }{a.R, a.G, a.B})
	second := luminance(struct{ R, G, B uint8 }{b.R, b.G, b.B})
	if first < second {
		first, second = second, first
	}
	return (first + 0.05) / (second + 0.05)
}

func shadeOfToken(t *testing.T, name string) theme.Shade {
	t.Helper()
	token, found := theme.TokenOf(name)
	if !found {
		t.Fatalf("token %q is not in the register", name)
	}
	shade, found := theme.ShadeOf(token.Shade)
	if !found {
		t.Fatalf("token %q names shade %q, which no ladder holds", name, token.Shade)
	}
	return shade
}

// The measurement this whole register exists for.
//
// # Why the grounds come from the entry rather than from the test
//
// A first version of this measurement read every token against every ground in
// the window and reported that no control passed, because it measured a button
// against the row highlight and no button stands there. The ground set is part
// of the claim, so it is stated beside the claim and walked from there. A
// token that quietly grows a new ground it is drawn on without saying so here
// is the one failure this cannot see, and that is what the guard on literals
// is for: a screen cannot name a colour, so it can only ever draw a token.
func TestEveryTokenReachesItsThresholdOnEveryGroundItStandsOn(t *testing.T) {
	for _, token := range theme.Tokens {
		if token.Threshold == theme.ThresholdGround {
			continue
		}
		ink := shadeOfToken(t, token.Name)
		for _, groundName := range token.On {
			ground := shadeOfToken(t, groundName)
			measured := contrast(ink, ground)
			if measured < token.Threshold {
				t.Errorf(
					"%s (%s) on %s (%s) measures %.2f, and it answers to %.1f\n"+
						"  what this token is for: %s\n"+
						"  the fix is a different shade in the register, never a lower threshold here",
					token.Name, token.Shade, groundName, ground.Name,
					measured, token.Threshold, token.Purpose)
			}
		}
	}
}

// A token that answers to a threshold but names no ground is a claim nothing
// can check, and a ground carrying a threshold is a measurement about itself.
func TestATokenIsEitherAGroundOrIsMeasuredAgainstOne(t *testing.T) {
	for _, token := range theme.Tokens {
		isGround := token.Threshold == theme.ThresholdGround
		switch {
		case isGround && len(token.On) > 0:
			t.Errorf("%s is a ground and also names grounds of its own, so it would be measured against itself", token.Name)
		case !isGround && len(token.On) == 0:
			t.Errorf("%s answers to %.1f and names nothing to measure it against, so the threshold checks nothing", token.Name, token.Threshold)
		}
	}
}

// Every ground a token names has to be a token, or the register has an entry
// nobody can resolve and the measurement above would skip it silently.
func TestEveryGroundNamedByATokenIsItselfAToken(t *testing.T) {
	for _, token := range theme.Tokens {
		for _, groundName := range token.On {
			if _, found := theme.TokenOf(groundName); !found {
				t.Errorf("%s says it is drawn on %q, which is not a token", token.Name, groundName)
			}
		}
	}
}

// The register closed from both sides. A constant with no entry is a name a
// screen can ask for and never get, and an entry with no constant is a colour
// only a string literal can reach, which the guard on literals forbids.
func TestEveryTokenNameAndEntryAnswerToEachOther(t *testing.T) {
	entries := make(map[string]bool, len(theme.Tokens))
	for _, token := range theme.Tokens {
		if entries[token.Name] {
			t.Errorf("token %q appears twice, so one of the two is dead", token.Name)
		}
		entries[token.Name] = true
	}
	constants := make(map[string]bool, len(theme.TokenNames))
	for _, name := range theme.TokenNames {
		constants[name] = true
		if !entries[name] {
			t.Errorf("the constant %q has no entry, so a screen asking for it gets the fallback", name)
		}
	}
	for name := range entries {
		if !constants[name] {
			t.Errorf("the entry %q has no constant, so only a bare string could reach it", name)
		}
	}
}

// A token naming a shade no ladder holds resolves to the fallback, which is a
// screen drawn in the wrong colour with nothing said about it.
func TestEveryTokenNamesAShadeThatExists(t *testing.T) {
	for _, token := range theme.Tokens {
		if _, found := theme.ShadeOf(token.Shade); !found {
			t.Errorf("token %s names shade %q, which no ladder holds", token.Name, token.Shade)
		}
	}
}

// A ladder whose lightness does not rise step by step is not a ladder. This is
// the property that makes twelve steps worth keeping when only some of them
// have a token: a border is made louder by moving one step, and that is only
// true while the steps are in order.
func TestEveryLadderRisesStepByStep(t *testing.T) {
	ladders := map[string][]theme.Shade{"neutral": theme.Neutral, "accent": theme.Accent}
	for name, ladder := range ladders {
		for i := 1; i < len(ladder); i++ {
			previous := luminance(struct{ R, G, B uint8 }{ladder[i-1].R, ladder[i-1].G, ladder[i-1].B})
			current := luminance(struct{ R, G, B uint8 }{ladder[i].R, ladder[i].G, ladder[i].B})
			if current <= previous {
				t.Errorf("the %s ladder does not rise from %s to %s, so the two are not one step apart in any useful sense",
					name, ladder[i-1].Name, ladder[i].Name)
			}
		}
	}
}

// Two shades a person cannot tell apart are two names for one colour, and a
// register holding both invites a screen to pick the wrong one and see nothing
// change. The threshold is ours: below 1.03 no ground reads as set apart from
// the one beside it, which is the whole job of steps 1 to 5.
func TestNoTwoShadesOfALadderAreTheSameColour(t *testing.T) {
	ladders := map[string][]theme.Shade{"neutral": theme.Neutral, "accent": theme.Accent}
	for name, ladder := range ladders {
		for i := 1; i < len(ladder); i++ {
			if measured := contrast(ladder[i-1], ladder[i]); measured < 1.03 {
				t.Errorf("%s and %s of the %s ladder measure %.3f apart, which is one colour under two names",
					ladder[i-1].Name, ladder[i].Name, name, measured)
			}
		}
	}
}

// Every shade is named once across all three ladders, or ShadeOf answers with
// whichever entry happened to be built last.
func TestEveryShadeIsNamedOnce(t *testing.T) {
	seen := make(map[string]bool)
	for _, ladder := range [][]theme.Shade{theme.Neutral, theme.Accent, theme.Semantic} {
		for _, shade := range ladder {
			if seen[shade.Name] {
				t.Errorf("shade %q is declared twice", shade.Name)
			}
			seen[shade.Name] = true
		}
	}
	if seen[theme.White.Name] {
		t.Errorf("white is also a step of a ladder, and it is stated as belonging to none")
	}
}

// TestEveryColourIsFullyOpaque holds that every shade the window is drawn with
// is opaque. A translucent shade would make every measured contrast a statement
// about a blend that the measurement never saw.
//
// It asked the same of the first palette until that palette was deleted with
// DEC-253, and moved here with the question rather than going with the palette.
func TestEveryColourIsFullyOpaque(t *testing.T) {
	ladders := [][]theme.Shade{theme.Neutral, theme.Accent, theme.Semantic, {theme.White}}
	for _, ladder := range ladders {
		for _, shade := range ladder {
			if got := shade.NRGBA().A; got != 0xFF {
				t.Errorf("%s is not opaque, its alpha is %#x", shade.Name, got)
			}
		}
	}
}

// A shade with no sentence saying what it is for is a colour somebody added to
// make one screen work. The same rule the spacing scale holds.
func TestEveryShadeStatesWhatItIsFor(t *testing.T) {
	for _, ladder := range [][]theme.Shade{theme.Neutral, theme.Accent, theme.Semantic} {
		for _, shade := range ladder {
			if shade.Job == "" {
				t.Errorf("shade %q says nothing about what it is for", shade.Name)
			}
		}
	}
	for _, token := range theme.Tokens {
		if token.Purpose == "" {
			t.Errorf("token %q says nothing about what stops being expressible without it", token.Name)
		}
	}
}

// The ladder of type sizes, closed from both sides for the same reason the
// tokens are.
func TestEveryTypeNameAndEntryAnswerToEachOther(t *testing.T) {
	entries := make(map[string]bool, len(theme.TypeScale))
	for _, step := range theme.TypeScale {
		if entries[step.Name] {
			t.Errorf("type size %q appears twice", step.Name)
		}
		entries[step.Name] = true
		if step.Purpose == "" {
			t.Errorf("type size %q says nothing about what it is for", step.Name)
		}
	}
	for _, name := range theme.TypeNames {
		if !entries[name] {
			t.Errorf("the constant %q has no rung, so a view asking for it gets the body size", name)
		}
	}
	for name := range entries {
		found := false
		for _, constant := range theme.TypeNames {
			if constant == name {
				found = true
			}
		}
		if !found {
			t.Errorf("the rung %q has no constant, so only a bare string could reach it", name)
		}
	}
}

// Two rungs of the same height are two names for one size, and a ladder that
// does not fall step by step is a ladder whose order says nothing.
func TestTheTypeLadderFallsStepByStep(t *testing.T) {
	for i := 1; i < len(theme.TypeScale); i++ {
		previous, current := theme.TypeScale[i-1], theme.TypeScale[i]
		if current.Size >= previous.Size {
			t.Errorf("%s is %g and %s is %g, so the ladder does not fall from the first to the second",
				previous.Name, previous.Size, current.Name, current.Size)
		}
	}
}

// Every rung is small text by the WCAG definition, which is what lets one
// threshold answer for all of them. WCAG turns at 24, or at 19 when bold, and
// this window has no bold text above the section heading.
func TestNoRungOfTheTypeLadderIsLargeTextByWCAG(t *testing.T) {
	const wcagLargeText = 24
	for _, step := range theme.TypeScale {
		if step.Size >= wcagLargeText {
			t.Errorf("%s is %g, which WCAG calls large text and measures against 3.0 rather than %.1f, so ThresholdText no longer answers for the whole ladder",
				step.Name, step.Size, theme.ThresholdText)
		}
	}
}

// Ink answers for every constant, and answers visibly for a name that is not
// ours. The fallback branch is unreachable while the register is closed, so it
// is proved by a name nothing can produce rather than left unproved.
func TestInkAnswersForEveryTokenAndStaysVisibleWhenItDoesNot(t *testing.T) {
	for _, name := range theme.TokenNames {
		if got := theme.Ink(name); got.A == 0 {
			t.Errorf("Ink(%q) is fully transparent, which renders as a hole nobody can see", name)
		}
	}
	fallback := theme.Ink("no.such.token")
	if fallback.A == 0 {
		t.Error("Ink of an unknown name is transparent, so a typo would become an invisible control")
	}
	if fallback != theme.Ink(theme.TextPrimary) {
		t.Error("Ink of an unknown name should be the text colour, which is the loudest answer available")
	}
}

// The same for the ladder of sizes, where the fallback is the body size.
func TestTypeSizeAnswersForEveryRungAndFallsBackToTheBody(t *testing.T) {
	for _, name := range theme.TypeNames {
		if theme.TypeSize(name) <= 0 {
			t.Errorf("TypeSize(%q) is not a height", name)
		}
	}
	if theme.TypeSize("no.such.rung") != theme.TypeSize(theme.TypeBody) {
		t.Error("TypeSize of an unknown name should be the body size")
	}
}

// The stroke of an icon is computed rather than written, so the bounds have to
// leave a range for it to land in. A floor above the ceiling would silently
// pin every icon to one width and undo the reason the target exists.
func TestTheIconStrokeBoundsLeaveARangeToLandIn(t *testing.T) {
	if theme.IconStrokeFloor >= theme.IconStrokeCeiling {
		t.Errorf("the stroke floor %g is not below the ceiling %g, so every icon gets one width",
			theme.IconStrokeFloor, theme.IconStrokeCeiling)
	}
	if theme.IconStrokeTarget <= 0 {
		t.Error("the stroke target has to be a width on screen")
	}
}
