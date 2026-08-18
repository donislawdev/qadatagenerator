package take

import (
	"path/filepath"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
)

// loaded is the catalog this build carries, which is what the layer checks are
// answered against.
func loaded(t *testing.T) catalog.Catalog {
	t.Helper()
	described, err := catalogyaml.Load()
	if err != nil {
		t.Fatalf("the catalog inside this binary could not be read: %v", err)
	}
	return described
}

// usable is a set of settings this build accepts, which every case below then
// spoils in exactly one way.
//
// Starting from something that passes is the point. A case built from nothing
// would be refused for the first missing thing whatever the case meant to test,
// and the test would go green for the wrong reason.
func usable() Settings {
	settings := Defaults()
	settings.Layers = []string{"L2"}
	return settings
}

func TestSettingsThisBuildAcceptsAreNotRefused(t *testing.T) {
	if refusal := Check(usable(), loaded(t)); refusal.Refused() {
		t.Fatalf("a run this build can carry out was refused as %s, so every case "+
			"below would be passing for the wrong reason", refusal.Kind.Name())
	}
}

func TestEachUnusableSettingIsRefusedForItsOwnReason(t *testing.T) {
	for _, given := range []struct {
		what     string
		spoil    func(*Settings)
		expected RefusalKind
	}{
		{"no layer at all", func(s *Settings) { s.Layers = nil }, NoLayers},
		{"a layer nobody describes", func(s *Settings) { s.Layers = []string{"L9"} }, UnknownLayer},
		{"a delimiter of two characters", func(s *Settings) { s.Delimiter = "||" },
			UnreadableDelimiter},
		{"a delimiter of none", func(s *Settings) { s.Delimiter = "" }, UnreadableDelimiter},
		// The form named here is one the specification lists and this build does
		// not write, which is the one somebody is most likely to try.
		{"a manifest form this build does not write",
			func(s *Settings) { s.Manifest = "junit" }, UnknownManifestForm},
		{"a share nobody can read", func(s *Settings) { s.Rows = "banana" }, UnreadableShare},
		// Shaped like a percentage and not a number. It reached the engine once,
		// where it chose no row at all and the run ended reporting success.
		{"a share that is not a number", func(s *Settings) { s.Rows = "NaN%" }, UnreadableShare},
		{"a per row setting nobody knows", func(s *Settings) { s.PerRow = "two" }, UnknownPerRow},
	} {
		t.Run(given.what, func(t *testing.T) {
			settings := usable()
			given.spoil(&settings)
			refusal := Check(settings, loaded(t))
			if !refusal.Refused() {
				t.Fatalf("%s was accepted, so a run would start on a setting this "+
					"build cannot carry out", given.what)
			}
			if refusal.Kind != given.expected {
				t.Errorf("%s was refused as %s rather than as %s, so the reader is "+
					"told to fix something that is not what is wrong",
					given.what, refusal.Kind.Name(), given.expected.Name())
			}
		})
	}
}

// TestALayerIsAnsweredBeforeAValueIs pins the order of the two kinds of check.
//
// Both are wrong in this case and only one can be reported. The layer is the
// one to report, because it is the setting with no default: somebody who named
// no layer has not started yet, while somebody with an unreadable share has a
// run in mind and one thing to correct.
func TestALayerIsAnsweredBeforeAValueIs(t *testing.T) {
	settings := usable()
	settings.Layers = nil
	settings.Rows = "banana"
	if refusal := Check(settings, loaded(t)); refusal.Kind != NoLayers {
		t.Errorf("a run with no layer and an unreadable share was refused as %s, "+
			"so the reader is sent to the share while the run has not been "+
			"told what to damage at all", refusal.Kind.Name())
	}
}

// TestARefusalCarriesWhatWouldHaveBeenTaken guards the half of a refusal that
// is easiest to leave out.
//
// Naming what is wrong without naming what would have been right sends the
// reader to the source code, which is the failure the four part shape exists to
// prevent. Only the kinds whose answer is a set are asked here. The others have
// an answer that is a sentence, and a sentence belongs to the surface.
func TestARefusalCarriesWhatWouldHaveBeenTaken(t *testing.T) {
	for _, given := range []struct {
		what  string
		spoil func(*Settings)
	}{
		{"no layer at all", func(s *Settings) { s.Layers = nil }},
		{"a layer nobody describes", func(s *Settings) { s.Layers = []string{"L9"} }},
		{"a manifest form this build does not write", func(s *Settings) { s.Manifest = "junit" }},
		{"a per row setting nobody knows", func(s *Settings) { s.PerRow = "two" }},
	} {
		t.Run(given.what, func(t *testing.T) {
			settings := usable()
			given.spoil(&settings)
			refusal := Check(settings, loaded(t))
			if len(refusal.Allowed) == 0 {
				t.Errorf("%s is refused as %s and nothing says what would have been "+
					"taken instead, so the reader has to find out from the source",
					given.what, refusal.Kind.Name())
			}
		})
	}
}

// TestNoOrdinaryRunIsRefusedForOverwritingItsOwnSource is one half of the check
// guarding the thing that cannot be taken back.
//
// 🔴 The other half is missing on purpose, and this is what measuring it showed.
// I expected to reach the refusal by handing it a source already carrying the
// suffix. It cannot be reached that way, or any other: the entity is the base
// name without its last extension, so a source named contacts.corrupted.csv
// produces contacts.corrupted.corrupted.csv, and the two names move apart
// instead of meeting. The comment on CheckOutput says exactly this, and the
// measurement agrees with it.
//
// So no test can show that refusal firing while the name is built this way, and
// writing one that appeared to would mean testing something else by accident.
// What shows it can fire is a mutation that changes how the name is built,
// which is where that proof lives.
//
// What is left here is worth keeping on its own. A check that never fires is
// one careless line away from being a check that always fires, and this one
// stands in front of every ordinary run of the mode.
func TestNoOrdinaryRunIsRefusedForOverwritingItsOwnSource(t *testing.T) {
	settings := usable()
	for _, source := range []string{
		filepath.Join("/data", "contacts.csv"),
		filepath.Join("/data", "contacts"+Suffix),
		"contacts.csv",
	} {
		if refusal := CheckOutput(source, settings); refusal.Refused() {
			t.Errorf("a run over %s was refused as %s, which would stop the mode "+
				"doing the one thing it is for", source, refusal.Kind.Name())
		}
	}
}

// TestTwoNamesForOnePlaceAreSeenAsOne checks the comparison the refusal above
// rests on, since nothing reachable from outside can.
func TestTwoNamesForOnePlaceAreSeenAsOne(t *testing.T) {
	for _, given := range []struct {
		what        string
		left, right string
		same        bool
	}{
		{"the same path twice", "/data/contacts.csv", "/data/contacts.csv", true},
		{"a path that walks back on itself",
			filepath.Join("/data", "damaged", "..", "contacts.csv"), "/data/contacts.csv", true},
		{"two files in one directory", "/data/contacts.csv", "/data/others.csv", false},
		{"one name in two directories", "/data/contacts.csv", "/other/contacts.csv", false},
	} {
		if got := sameFile(given.left, given.right); got != given.same {
			t.Errorf("%s: %q and %q were taken for %v, and this comparison is what "+
				"stands between a damaged file and the records it was made from",
				given.what, given.left, given.right, got)
		}
	}
}

func TestALayerListIsReadTheSameWayWhoeverTypedIt(t *testing.T) {
	for _, given := range []struct {
		written string
		want    []string
	}{
		{"L2,L3", []string{"L2", "L3"}},
		{"L2, L3", []string{"L2", "L3"}},
		{" L2 ,, L3 ,", []string{"L2", "L3"}},
		{"", nil},
		{" , ", nil},
	} {
		got := NamesIn(given.written)
		if len(got) != len(given.want) {
			t.Errorf("%q was read as %v, and both surfaces hand this the text a "+
				"person typed, so a second reading of it is a second set of rules",
				given.written, got)
			continue
		}
		for at, name := range given.want {
			if got[at] != name {
				t.Errorf("%q was read as %v rather than %v", given.written, got, given.want)
				break
			}
		}
	}
}
