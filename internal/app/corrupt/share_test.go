package corrupt_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/app/corrupt"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// A share that is not a number is refused, and the ones that matter are the two
// no range test can catch.
//
// 🔴 Every comparison against NaN is false, so a gate written as "below nought
// or above a hundred" lets it through, and so does the draw that decides which
// rows are damaged. Measured on the built binary before this test existed: a
// policy that ordered sixty defects ordered none, and the run ended with a
// success code. That is the silence this whole package exists to make
// impossible, arriving through the one value arithmetic does not refuse.
//
// Infinity is here beside it because it comes from the same parser and reads as
// a number to everything downstream. It happens to fail the range test today,
// which is exactly why it belongs in a test: it passes for a reason nobody
// wrote down, and the next edit of that line could take it away.
func TestAPolicyShareThatIsNotANumberIsRefused(t *testing.T) {
	for _, written := range []string{"NaN%", "nan%", "NAN%", "+Inf%", "-Inf%", "inf%"} {
		if err := corrupt.UsableShare(written); err == nil {
			t.Errorf("a share of %q was accepted, and no row would ever be chosen", written)
		}
	}
}

// The shares that mean something still work, so that the refusal above did not
// buy itself by refusing everything.
func TestAShareThatIsANumberIsStillRead(t *testing.T) {
	for _, written := range []string{"", "0%", "5%", "12.5%", "100%"} {
		if err := corrupt.UsableShare(written); err != nil {
			t.Errorf("a share of %q was refused: %v", written, err)
		}
	}
}

// A share nobody can read stops the run before any row is chosen, rather than
// choosing none of them and reporting success.
//
// This is the same statement as the test above made where it can be seen: the
// selector is what turns a share into damage, so a share it cannot use has to
// fail on the way in rather than produce an empty run.
func TestASelectorRefusesAShareThatIsNotANumber(t *testing.T) {
	described := aCatalog(t, "L2.a.one")
	_, err := corrupt.NewSelector(recipe.Policy{Layers: []string{"L2"}, Rows: "NaN%"},
		described, aView("name"), seed.New("orders"))
	if err == nil {
		t.Fatal("a selector was built from a share of NaN, so the run would damage nothing")
	}
	if !strings.Contains(err.Error(), "NaN%") {
		t.Errorf("the refusal does not say which value it refused: %v", err)
	}
}
