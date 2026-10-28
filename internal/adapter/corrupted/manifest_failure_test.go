package corrupted_test

import (
	"errors"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/corrupted"
	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
)

// full is a file that takes nothing, which is what a disk out of room is to
// the document written to it.
type full struct{ tried int }

var errFull = errors.New("there is no room left on the disk")

func (f *full) Write([]byte) (int, error) {
	f.tried++
	return 0, errFull
}

// 🔴 A run whose manifest cannot be written stops within one buffer of entries,
// rather than producing the whole dataset and failing at its close.
//
// The real document rather than a stand in, because the promise is about the
// two together: the document holds entries in a buffer of a few kilobytes, so
// it learns of the failure when the buffer first goes to the file, and the run
// has to stop at that entry. Before 2026-10-28 all of the records went on.
func TestARunStopsWithinABufferOfAManifestThatCannotBeWritten(t *testing.T) {
	const rows = 1000
	disk := &full{}
	writer, next, _ := writerInto(t, corrupted.Tee{manifestjson.New(disk, "0.2")}, rows)

	var stopped error
	for row := 0; row < rows && stopped == nil; row++ {
		stopped = writer.Write(dataset.Record{Values: []string{"1250"}, Parent: dataset.NoParent})
	}
	if stopped == nil {
		t.Fatalf("all %d records reached the data beside a manifest nothing could be written to",
			next.rows)
	}
	t.Logf("the run stopped after %d of %d records", next.rows, rows)
	if !errors.Is(stopped, errFull) {
		t.Errorf("the run stopped with %v rather than with the failure of the disk", stopped)
	}
	if next.rows >= 100 {
		t.Errorf("%d records reached the data before the run noticed its manifest could not be "+
			"written, want fewer than one buffer of entries", next.rows)
	}
	if disk.tried != 1 {
		t.Errorf("the disk was tried %d times after it refused, want once", disk.tried)
	}
}
