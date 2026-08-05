package catalog_test

import (
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
)

func twoEntries(t *testing.T) catalog.Catalog {
	t.Helper()
	second := usable()
	second.ID = "L3.checksum.off_by_one"
	second.Layer = catalog.Domain
	built, problems := catalog.New("0.1", "draft", []catalog.Entry{usable(), second})
	if len(problems) > 0 {
		t.Fatalf("the fixture catalog was refused: %v", problems)
	}
	return built
}

// The closure is checked with identifiers handed in rather than read from the
// registry, and that is the whole point of the shape.
//
// The registry is empty in this build, so a closure that read it directly could
// not be seen to fail at all: it would pass against a build with no mechanism
// behind it, which is exactly how three guards were found standing green during
// the last piece of work. Handing the identifiers in is what lets a test break
// the rule on purpose.
func TestAnImplementationWithNoEntryIsNamed(t *testing.T) {
	built := twoEntries(t)
	undescribed := catalog.Undescribed(built, []catalog.ID{
		"L2.number.text_in_numeric",
		"L4.coherence.invented_by_somebody",
	})
	if len(undescribed) != 1 || undescribed[0] != "L4.coherence.invented_by_somebody" {
		t.Fatalf("got %v, and the one implementation nobody described is the defect", undescribed)
	}
}

func TestAnImplementationOfEveryEntryLeavesNothingUndescribed(t *testing.T) {
	built := twoEntries(t)
	if got := catalog.Undescribed(built, built.IDs()); len(got) != 0 {
		t.Errorf("got %v, and every one of these has an entry", got)
	}
}

// The other direction is progress rather than a defect, so it is reported and
// not failed. A catalog entry describes damage worth doing whether or not this
// build can do it yet.
func TestAnEntryWithNoImplementationIsReported(t *testing.T) {
	built := twoEntries(t)
	unregistered := catalog.Unregistered(built, []catalog.ID{"L2.number.text_in_numeric"})
	if len(unregistered) != 1 || unregistered[0] != "L3.checksum.off_by_one" {
		t.Fatalf("got %v, and L3.checksum.off_by_one has no implementation here", unregistered)
	}

	if got := catalog.Unregistered(built, nil); len(got) != built.Len() {
		t.Errorf("with nothing implemented %d entries are covered, which cannot be", built.Len()-len(got))
	}
}

// Nothing implemented is a legal state and has to stay expressible, because it
// is what the build was in on the day these functions were written and what a
// stripped down build could be in again.
func TestNothingImplementedIsAnAnswerRatherThanAFailure(t *testing.T) {
	built := twoEntries(t)
	if got := catalog.Undescribed(built, nil); len(got) != 0 {
		t.Errorf("with nothing implemented, %v were reported as undescribed", got)
	}
	if got := catalog.Unregistered(built, nil); len(got) != built.Len() {
		t.Errorf("with nothing implemented, %d of %d entries were counted as covered",
			built.Len()-len(got), built.Len())
	}
}
