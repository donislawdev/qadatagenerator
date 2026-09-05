package theme

import (
	"fmt"
	"testing"
)

// Every entry has to be reachable by the name the document uses, so a rename
// on one side cannot hide behind a duplicate on the other.
func TestEveryNameInThePaletteIsUsedOnce(t *testing.T) {
	seen := map[string]bool{}
	for _, entry := range Palette {
		if seen[entry.Name] {
			t.Errorf("the palette carries %s twice, so one of them is unreachable by name", entry.Name)
		}
		seen[entry.Name] = true
	}
	if seen[Background.Name] {
		t.Errorf("%s is both the reference and one of the subjects, so it would be measured against itself", Background.Name)
	}
}

func TestEveryStepOfTheScaleIsUsedOnce(t *testing.T) {
	seenName := map[string]bool{}
	seenStep := map[float32]bool{}
	for _, step := range Scale {
		if seenName[step.Name] {
			t.Errorf("the scale carries %s twice", step.Name)
		}
		// Two names for one distance is the thing a closed scale exists to
		// prevent, so it is a failure rather than a note.
		if seenStep[step.Step] {
			t.Errorf("%s repeats a distance the scale already has, which makes it a second name for one gap", step.Name)
		}
		seenName[step.Name] = true
		seenStep[step.Step] = true
	}
}

// The hex form is computed rather than stored, so this checks that the
// computation is the one the document is written in. A lower case result would
// still parse and would still make the bridge report a difference that is not
// one.
func TestTheHexFormMatchesTheChannelsItIsBuiltFrom(t *testing.T) {
	cases := []struct {
		entry Color
		want  string
	}{
		{Color{R: 0xE8, G: 0xE8, B: 0xE8}, "#E8E8E8"},
		{Color{R: 0x00, G: 0x00, B: 0x00}, "#000000"},
		{Color{R: 0xFF, G: 0xB9, B: 0x00}, "#FFB900"},
		{Color{R: 0x0A, G: 0x0B, B: 0x0C}, "#0A0B0C"},
	}
	for _, one := range cases {
		if got := one.entry.Hex(); got != one.want {
			t.Errorf("the hex form is %s, and the document writes %s", got, one.want)
		}
	}
}

// A translucent entry would make every measured contrast a statement about a
// blend that the measurement never saw.
func TestEveryColourIsFullyOpaque(t *testing.T) {
	for _, entry := range append([]Color{Background}, Palette...) {
		if got := entry.NRGBA().A; got != 0xFF {
			t.Errorf("%s is not opaque, its alpha is %#x", entry.Name, got)
		}
	}
}

// The scale claims that a gap never rounds to a different distance than its
// neighbour of the same name. That claim is arithmetic, so it is done here
// rather than trusted to the paragraph that states it.
func TestEveryStepSurvivesTheCommonScalingFactors(t *testing.T) {
	factors := []float32{1.0, 1.25, 1.5, 2.0}
	for _, step := range Scale {
		if int(step.Step)%SpacingModule != 0 {
			t.Errorf("%s is %v, which is not a multiple of the module %d", step.Name, step.Step, SpacingModule)
		}
		for _, factor := range factors {
			scaled := step.Step * factor
			if scaled != float32(int(scaled)) {
				t.Errorf("%s scaled by %v lands on %v, which is not a whole number of pixels", step.Name, factor, scaled)
			}
		}
	}
}

// A scale that is not ordered cannot be read as a scale, and the name of a
// step would stop saying anything about how big it is.
func TestTheScaleRisesFromTheSmallestStepToTheLargest(t *testing.T) {
	for i := 1; i < len(Scale); i++ {
		if Scale[i].Step <= Scale[i-1].Step {
			t.Errorf("%s is not larger than %s, so the order of the scale says nothing", Scale[i].Name, Scale[i-1].Name)
		}
	}
}

// An entry with no purpose is an entry nobody can argue with when it is time
// to remove it.
func TestEveryEntryStatesWhatItIsFor(t *testing.T) {
	for _, entry := range append([]Color{Background}, Palette...) {
		if entry.Purpose == "" {
			t.Errorf("the colour %s does not say what it is for", entry.Name)
		}
		if entry.Name == "" {
			t.Error("a colour has no name, so the bridge to the document cannot reach it")
		}
	}
	for _, step := range Scale {
		if step.Purpose == "" {
			t.Errorf("the step %s does not say what it is for", step.Name)
		}
	}
}

// The role decides which threshold the contrast is measured against, so an
// unknown role would be measured against nothing at all.
func TestEveryColourCarriesAKnownRole(t *testing.T) {
	known := map[Role]bool{
		RoleText: true, RoleSurface: true, RoleAuxiliary: true,
		RoleOnFill: true, RoleBackground: true,
	}
	for _, entry := range append([]Color{Background}, Palette...) {
		if !known[entry.Role] {
			t.Errorf("%s carries the role %q, which no threshold answers to", entry.Name, entry.Role)
		}
	}
}

// TestEveryColourReadOnAnotherSaysWhichOne closes MeasuredAgainst from both
// sides.
//
// One side alone answers half the question. An entry that carries RoleOnFill
// and names no fill is a colour whose threshold cannot be computed at all, and
// the bridge would have to either invent a reference or skip the entry. A row
// naming an entry that is not in the palette is a rule about nothing, which
// reads like a rule until somebody looks.
func TestEveryColourReadOnAnotherSaysWhichOne(t *testing.T) {
	for _, entry := range Palette {
		if entry.Role != RoleOnFill {
			continue
		}
		if _, named := MeasuredAgainst[entry.Name]; !named {
			t.Errorf("%s is read on another entry and does not say which one", entry.Name)
		}
	}
	for name, fill := range MeasuredAgainst {
		read, found := Lookup(name)
		if !found {
			t.Errorf("MeasuredAgainst names %q, which the palette does not have", name)
			continue
		}
		if read.Role != RoleOnFill {
			t.Errorf("%s is measured against %s and carries the role %q rather than %q",
				name, fill, read.Role, RoleOnFill)
		}
		if _, found := Lookup(fill); !found {
			t.Errorf("%s is measured against %q, which the palette does not have", name, fill)
		}
	}
}

// A palette that empties itself would let every other test here pass by having
// nothing to check, which is the shape this project has been bitten by before.
func TestThePaletteAndTheScaleAreNotEmpty(t *testing.T) {
	if len(Palette) == 0 {
		t.Fatal("the palette is empty, so every check above passes by checking nothing")
	}
	if len(Scale) == 0 {
		t.Fatal("the scale is empty, so every check above passes by checking nothing")
	}
}

// The register of names has to close from both sides.
//
// One direction alone answers half the question, which this project learned
// the session before when a flag lived in the parser and not in the help. A
// constant with no entry resolves to the fallback and a view shows the wrong
// colour. An entry with no constant is a colour no screen can ask for without
// typing a bare string, which is what the constants exist to prevent.
func TestEveryColourNameAndEntryAnswerToEachOther(t *testing.T) {
	entries := map[string]bool{}
	for _, entry := range Palette {
		entries[entry.Name] = true
	}
	constants := map[string]bool{}
	for _, name := range ColorNames {
		constants[name] = true
		if !entries[name] {
			t.Errorf("the constant for %q has no entry in the palette, so a view asking for it gets the fallback", name)
		}
	}
	for name := range entries {
		if !constants[name] {
			t.Errorf("the palette carries %q and no constant names it, so a view can only ask by typing it", name)
		}
	}
}

func TestEveryStepNameAndEntryAnswerToEachOther(t *testing.T) {
	entries := map[string]bool{}
	for _, step := range Scale {
		entries[step.Name] = true
	}
	constants := map[string]bool{}
	for _, name := range StepNames {
		constants[name] = true
		if !entries[name] {
			t.Errorf("the constant for %q has no step in the scale", name)
		}
	}
	for name := range entries {
		if !constants[name] {
			t.Errorf("the scale carries %q and no constant names it", name)
		}
	}
}

// The lookup has to find what is there and has to stay visible when it does
// not, because a view cannot stop to report a missing colour.
func TestLookupFindsWhatIsThereAndStaysVisibleWhenItDoesNot(t *testing.T) {
	if got := Named(NameRejection); got.Name != NameRejection {
		t.Errorf("looking up %q found %q", NameRejection, got.Name)
	}
	if got := Named(Background.Name); got.Name != Background.Name {
		t.Error("the background is not reachable by name, and a window needs it")
	}
	missing := Named("no.such.colour")
	if missing.NRGBA().A != 0xFF {
		t.Error("an unknown name resolved to something transparent, which renders as a hole nobody can see")
	}
	if got := StepOf(NameStepSection); got != StepOf(NameStepSection) || got == 0 {
		t.Error("a known step did not resolve to a distance")
	}
	if StepOf("no.such.step") == 0 {
		t.Error("an unknown step resolved to nothing, which collapses a layout silently")
	}
}

func ExampleColor_Hex() {
	fmt.Println(Palette[1].Name, Palette[1].Hex())
	// Output: text #E8E8E8
}
