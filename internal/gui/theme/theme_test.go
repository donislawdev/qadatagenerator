package theme

import (
	"fmt"
	"testing"
)

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

// A step with no purpose is a step nobody can argue with when it is time to
// remove it.
func TestEveryStepStatesWhatItIsFor(t *testing.T) {
	for _, step := range Scale {
		if step.Purpose == "" {
			t.Errorf("the step %s does not say what it is for", step.Name)
		}
	}
}

// A scale that empties itself would let every other test here pass by having
// nothing to check, which is the shape this project has been bitten by before.
func TestTheScaleIsNotEmpty(t *testing.T) {
	if len(Scale) == 0 {
		t.Fatal("the scale is empty, so every check above passes by checking nothing")
	}
}

// The register of names has to close from both sides.
//
// One direction alone answers half the question, which this project learned
// when a flag lived in the parser and not in the help. A constant with no step
// resolves to the fallback and a view shows the wrong distance. A step with no
// constant is a distance no screen can ask for without typing a bare string,
// which is what the constants exist to prevent.
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
// not, because a view cannot stop to report a missing distance.
func TestStepOfFindsWhatIsThereAndNeverAnswersNought(t *testing.T) {
	if got := StepOf(NameStepSection); got != 24 {
		t.Errorf("the section step resolved to %v", got)
	}
	if StepOf("no.such.step") == 0 {
		t.Error("an unknown step resolved to nothing, which collapses a layout silently")
	}
}

func ExampleStepOf() {
	fmt.Println(NameStepRelated, StepOf(NameStepRelated))
	// Output: related 12
}
