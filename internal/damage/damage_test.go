package damage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestexport"
	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
	"github.com/donislawdev/qadatagenerator/internal/damage"
)

// A run says which of the files it wrote this build will not read back.
//
// 🔴 This is worked out here rather than on either surface because it used to
// be worked out on the command line alone, and the window therefore said
// nothing: somebody damaging a wide file in the window got their files, correct
// counts, and a refusal from another screen days later with nothing connecting
// the two. Both surfaces and both modes now read the same answer.
//
// The files are made by length rather than by a run wide enough to produce a
// real one, because what is under test is which files are named and on what
// grounds, not the arithmetic that reaches the size.
func TestARunSaysWhichFilesWillNotBeReadBack(t *testing.T) {
	canonical := manifestjson.Name
	for _, given := range []struct {
		what  string
		sizes map[string]int64
		form  string
		want  []string
	}{
		{"nothing oversized",
			map[string]int64{"a.corrupted.csv": 10, canonical: 10}, "", nil},
		{"a manifest one byte past the limit",
			map[string]int64{"a.corrupted.csv": 10, canonical: damage.ReadBackLimit + 1}, "",
			[]string{canonical}},
		{"a manifest exactly at the limit is not named",
			map[string]int64{canonical: damage.ReadBackLimit}, "", nil},
		// The damaged file is the one somebody asked for and nothing reads it
		// back as a manifest, so its size is not this warning's business however
		// large it gets.
		{"a damaged file larger than the limit",
			map[string]int64{"a.corrupted.csv": damage.ReadBackLimit + 1}, "", nil},
		// 🔴 The object form of a dataset ends in .json like a manifest does,
		// and it is the dataset rather than something read back. It is here
		// because the mode that generates writes exactly this file into the
		// same directory as its manifest.
		{"a dataset in the object form larger than the limit",
			map[string]int64{"dataset.json": damage.ReadBackLimit + 1, canonical: 10}, "", nil},
		// An export is read by a person or by another tool, never back by this
		// build, so it is not measured against what this build reads.
		{"an export larger than the limit",
			map[string]int64{manifestexport.FileFor("csv"): damage.ReadBackLimit + 1, canonical: 10},
			"csv", nil},
		{"a file that is not there at all",
			map[string]int64{}, "", nil},
	} {
		t.Run(given.what, func(t *testing.T) {
			dir := t.TempDir()
			for name, size := range given.sizes {
				file, err := os.Create(filepath.Join(dir, name))
				if err != nil {
					t.Fatalf("cannot make %s: %v", name, err)
				}
				if err := file.Truncate(size); err != nil {
					t.Fatalf("cannot give %s its length: %v", name, err)
				}
				if err := file.Close(); err != nil {
					t.Fatalf("cannot close %s: %v", name, err)
				}
			}

			found := damage.Oversized(dir, given.form)

			if len(found) != len(given.want) {
				t.Fatalf("%s named %v, want %v", given.what, found, given.want)
			}
			for i, one := range found {
				if one.Name != given.want[i] {
					t.Errorf("named %q at %d, want %q", one.Name, i, given.want[i])
				}
				if one.Bytes != given.sizes[one.Name] {
					t.Errorf("said %s is %d bytes, and it is %d",
						one.Name, one.Bytes, given.sizes[one.Name])
				}
			}
		})
	}
}

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
