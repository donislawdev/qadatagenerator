package damage_test

import (
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestexport"
	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
	"github.com/donislawdev/qadatagenerator/internal/damage"
)

// A run says what it wrote in several places, and a form added to only some of
// them would leave the rest quietly announcing the wrong set of files.
//
// This walks the forms this build exports rather than naming three of them, so
// a fourth is covered on the day it is added instead of on the day somebody
// remembers this file.
func TestEveryFormThisBuildExportsIsNamedAmongTheFilesARunWrites(t *testing.T) {
	forms := manifestexport.Forms()
	if len(forms) == 0 {
		t.Fatal("this build exports no form at all, so this test compared nothing")
	}
	for _, form := range forms {
		files := damage.Files(form)
		if len(files) != 2 {
			t.Errorf("a run exporting %q says it writes %v", form, files)
			continue
		}
		if files[0] != manifestjson.Name {
			t.Errorf("a run exporting %q does not name the canonical manifest first: %v",
				form, files)
		}
		if want := manifestexport.FileFor(form); files[1] != want {
			t.Errorf("a run exporting %q names %q rather than %q", form, files[1], want)
		}
	}
}

// The empty form is usable and means no export, which is the one answer a
// caller has to be able to give without naming anything.
func TestARunAskedForNoExportWritesTheCanonicalManifestAlone(t *testing.T) {
	files := damage.Files("")
	if len(files) != 1 || files[0] != manifestjson.Name {
		t.Fatalf("a run without an export says it writes %v", files)
	}
}

// A form this build does not export adds no second file, rather than naming a
// file nothing would write. Whoever chose the form is refused earlier, and this
// is what keeps the announcement true if that refusal is ever moved.
func TestAFormThisBuildDoesNotExportAddsNoFileToTheAnnouncement(t *testing.T) {
	unknown := "parquet"
	if manifestexport.Usable(unknown) {
		t.Fatalf("%q was chosen for this test because this build does not export it, "+
			"and it now does", unknown)
	}
	files := damage.Files(unknown)
	if len(files) != 1 || files[0] != manifestjson.Name {
		t.Fatalf("a run asked for an unknown form says it writes %v", files)
	}
}

// A count of what was applied is true and leaves a false impression of the
// scope: a run where fifty nine of sixty ordered defects went nowhere looks,
// from the applied count alone, like a run that barely did anything wrong. Both
// surfaces show the shortfall and both read it from here, which is the only
// place the subtraction happens.
func TestTheShortfallIsWhatWasOrderedLessWhatReachedTheData(t *testing.T) {
	counted := []struct {
		how       string
		counts    damage.Counts
		shortfall int
	}{
		{"a run where nearly everything went nowhere",
			damage.Counts{Requested: 60, Applied: 1}, 59},
		{"a run that applied everything it ordered",
			damage.Counts{Requested: 12, Applied: 12}, 0},
		{"a run that ordered nothing at all",
			damage.Counts{}, 0},
	}
	for _, kind := range counted {
		if missing := kind.counts.Missing(); missing != kind.shortfall {
			t.Errorf("%s left %d ordered defects unapplied rather than %d",
				kind.how, missing, kind.shortfall)
		}
	}
}
