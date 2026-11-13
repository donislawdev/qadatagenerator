package take

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
	"github.com/donislawdev/qadatagenerator/internal/app/corrupt"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
)

// namedRows is how many records the files below hold. The entries about people
// take turns with the other entries of their layer, so the file has to be long
// enough for each of them to come round several times.
const namedRows = 60

// peopleFile writes a file this tool did not make, under the headers given, with
// the same row in every record.
func peopleFile(t *testing.T, dir string, header, row string) string {
	t.Helper()
	lines := []string{header}
	for range namedRows {
		lines = append(lines, row)
	}
	path := filepath.Join(dir, "people.csv")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatalf("cannot write the file this test damages: %v", err)
	}
	return path
}

// namesDamaged runs the layer of unusual but valid values over every record of
// a file and returns what the manifest says about the entries about names.
func namesDamaged(t *testing.T, header, row string) []manifest.Defect {
	t.Helper()
	dir := t.TempDir()
	settings := usable()
	settings.Layers = []string{"REV"}
	settings.Rows = "100%"
	settings.Out = filepath.Join(dir, "out")
	opened, err := Open(peopleFile(t, dir, header, row), settings, corrupt.NeverStops{})
	if err != nil {
		t.Fatalf("the file could not be read: %v", err)
	}
	defer func() { _ = opened.Close() }()
	if _, err := opened.Corrupt(settings, loaded(t), corrupt.NeverStops{}, time.Now()); err != nil {
		t.Fatalf("the run did not finish: %v", err)
	}
	record, err := manifestjson.Load(filepath.Join(settings.Out, manifestjson.Name))
	if err != nil {
		t.Fatalf("the manifest could not be read back: %v", err)
	}
	var names []manifest.Defect
	for _, defect := range record.Defects() {
		if strings.HasPrefix(string(defect.Mutator), "REV.name.") {
			names = append(names, defect)
		}
	}
	return names
}

// 🔴 In somebody else's file, a surname with an apostrophe lands only under a
// header naming surnames, a name with accents under one naming a given name or
// a surname, and nothing about names lands on a column of towns.
//
// This is the whole road a header travels: read in the inference, answered by
// the column, aimed by the selector, carried out by the damage, written to the
// manifest. Each step has its own test, and none of them sees the others drop
// the answer.
func TestNamesLandOnlyUnderAHeaderNamingThem(t *testing.T) {
	landed := map[catalog.ID]map[string]int{}
	for _, defect := range namesDamaged(t, "miasto,imi\U00000119,nazwisko", "Szczyrk,Anna,Nowak") {
		if !defect.Applied {
			t.Errorf("%s was refused in row %d of a file with a column for it: %s",
				defect.Mutator, defect.Location.Row, defect.FailureReason)
			continue
		}
		if landed[defect.Mutator] == nil {
			landed[defect.Mutator] = map[string]int{}
		}
		landed[defect.Mutator][defect.Location.Field]++
	}
	allowed := map[catalog.ID][]string{
		"REV.name.apostrophe":   {"nazwisko"},
		"REV.name.null_surname": {"nazwisko"},
		"REV.name.diacritics":   {"imi\U00000119", "nazwisko"},
		"REV.name.cjk":          {"imi\U00000119", "nazwisko"},
	}
	for id, columns := range allowed {
		if len(landed[id]) == 0 {
			t.Errorf("%s never landed, so this test says nothing about where it lands", id)
		}
		for column, times := range landed[id] {
			if !strings.Contains(","+strings.Join(columns, ",")+",", ","+column+",") {
				t.Errorf("%s landed %d times on %q, which may carry only %v", id, times, column,
					columns)
			}
		}
	}
}

// Without a header naming them, every order about names is refused in the
// manifest rather than guessed onto a column of text, and the refusal is about
// what the run lacks rather than about the value.
func TestWithoutAHeaderNamingThemNamesAreRefusedRatherThanGuessed(t *testing.T) {
	refused := namesDamaged(t, "miasto,uwagi", "Szczyrk,Anna Nowak")
	if len(refused) == 0 {
		t.Fatal("no order about names was made, so this test says nothing about refusing them")
	}
	for _, defect := range refused {
		if defect.Applied || defect.FailureKind != manifest.RunLacksCapability {
			t.Errorf("%s in row %d came out applied %v, kind %q: %s", defect.Mutator,
				defect.Location.Row, defect.Applied, defect.FailureKind, defect.FailureReason)
		}
	}
}
