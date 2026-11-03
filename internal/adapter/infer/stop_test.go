package infer_test

import (
	"errors"
	"strconv"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/infer"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
)

// stopsFrom counts the asks and answers that the work has to stop from the ask
// numbered from onwards, or never when from is zero.
type stopsFrom struct {
	from  int
	asked int
}

func (s *stopsFrom) Stopped() error {
	s.asked++
	if s.from > 0 && s.asked >= s.from {
		return errStopped
	}
	return nil
}

var errStopped = errors.New("stopped by the test")

// 🔴 Working out a table asks whether to stop before every column it looks at
// and before every column of identifiers it pairs, and a stop at any of those
// asks ends it with no shape.
//
// Until 2026-11-03 it asked nowhere, and its cost grows with the columns, so a
// window that had asked a newer question waited for every column of a wide file
// to be paired against every other (B-356). The table holds two columns of
// identifiers among four, which is what tells an ask for every column of
// identifiers from one ask for all of them and from one for every column.
func TestWorkingOutATableAsksToStopBeforeEveryColumnAndEveryPairing(t *testing.T) {
	identifiers := produced(t, "pesel", 20)
	genders, _ := neighbours(t, identifiers)
	notes := make([]string, len(identifiers))
	for at := range notes {
		notes[at] = "note " + strconv.Itoa(at)
	}
	table := tableOf("first", "sex", "second", "note")
	samples := []dataset.Sample{
		sampleOf("first", identifiers), sampleOf("sex", genders),
		sampleOf("second", identifiers), sampleOf("note", notes),
	}
	const columns, identifierColumns = 4, 2

	never := &stopsFrom{}
	shape, err := infer.From(table, samples, schemes(), never)
	if err != nil {
		t.Fatalf("working out a table nothing stopped ended with %v", err)
	}
	recognised := 0
	for _, finding := range shape.Findings() {
		if finding.Scheme != "" {
			recognised++
		}
	}
	if recognised != identifierColumns {
		t.Fatalf("%d columns were taken for identifiers, want %d, so the asks counted below "+
			"are not the ones this test is about", recognised, identifierColumns)
	}
	if never.asked != columns+identifierColumns {
		t.Errorf("working out %d columns, %d of them identifiers, asked whether to stop %d times, "+
			"want once for every column and once for every column of identifiers",
			columns, identifierColumns, never.asked)
	}

	for from := 1; from <= columns+identifierColumns; from++ {
		stop := &stopsFrom{from: from}
		shape, err := infer.From(table, samples, schemes(), stop)
		if !errors.Is(err, errStopped) {
			t.Errorf("a stop at ask %d ended the working out with %v", from, err)
		}
		if stop.asked != from {
			t.Errorf("a stop at ask %d was asked %d times, so the work went on after it", from, stop.asked)
		}
		if len(shape.Findings()) != 0 || len(shape.Relations()) != 0 {
			t.Errorf("a stop at ask %d still handed back a shape, which nobody finished working out", from)
		}
	}
}
