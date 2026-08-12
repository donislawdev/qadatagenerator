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
	known := map[Role]bool{RoleText: true, RoleSurface: true, RoleAuxiliary: true, RoleBackground: true}
	for _, entry := range append([]Color{Background}, Palette...) {
		if !known[entry.Role] {
			t.Errorf("%s carries the role %q, which no threshold answers to", entry.Name, entry.Role)
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

func ExampleColor_Hex() {
	fmt.Println(Palette[1].Name, Palette[1].Hex())
	// Output: text #E8E8E8
}
