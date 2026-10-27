package format_test

import (
	"errors"
	"fmt"
	"runtime"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/format"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// watched is a set of files that writes nothing anywhere and remembers when
// each file was opened and closed, and can be told to refuse a close.
type watched struct {
	events      []string
	refuseClose error
}

func (w *watched) Create(name string) (format.WriteCloser, error) {
	w.events = append(w.events, "open "+name)
	return &watchedFile{name: name, set: w}, nil
}

type watchedFile struct {
	name string
	set  *watched
}

func (f *watchedFile) Write(p []byte) (int, error) { return len(p), nil }

func (f *watchedFile) Close() error {
	f.set.events = append(f.set.events, "close "+f.name)
	return f.set.refuseClose
}

// writeWith runs a recipe through a writer into a set of files and hands back
// what the run said.
func writeWith(t *testing.T, rec recipe.Recipe, files format.Files) error {
	t.Helper()
	source := values.New(rec.Locale, rec.Seed)
	plan, err := generate.Prepare(rec, source, generate.NeverStops{})
	if err != nil {
		t.Fatalf("the plan was refused: %v", err)
	}
	writer, err := format.For(files, rec, generate.NeverStops{})
	if err != nil {
		t.Fatalf("the writer was refused: %v", err)
	}
	return generate.Run(plan, rec, source, writer, generate.NeverStops{})
}

// The file of a table is closed when the table ends, before the next one opens.
//
// 🔴 Every file used to stay open until the set of files was committed at the
// end of the run, so a recipe of many small tables held a handle for each of
// them at once and a system allowing a process a few hundred open files would
// stop it part way (outside review, 2026-10-22). The tree is one file and is
// closed when it is written.
func TestTheFileOfATableIsClosedBeforeTheNextOneOpens(t *testing.T) {
	for _, shape := range []struct {
		format    string
		perEntity bool
		events    []string
	}{
		{"csv", true, []string{"open customer.csv", "close customer.csv", "open order.csv", "close order.csv"}},
		{"json", true, []string{"open customer.json", "close customer.json", "open order.json", "close order.json"}},
		{"json", false, []string{"open dataset.json", "close dataset.json"}},
	} {
		rec := sample(t, shape.perEntity, 3)
		rec.Output.Format = shape.format
		files := &watched{}

		if err := writeWith(t, rec, files); err != nil {
			t.Fatalf("%s, a file per table %v: the run failed: %v", shape.format, shape.perEntity, err)
		}
		if fmt.Sprint(files.events) != fmt.Sprint(shape.events) {
			t.Errorf("%s, a file per table %v: the files went %v rather than %v",
				shape.format, shape.perEntity, files.events, shape.events)
		}
	}
}

// A file that could not be closed is a file that may not hold what was written,
// and the run says so rather than reporting it written.
func TestAFileThatCouldNotBeClosedIsNeverReportedAsWritten(t *testing.T) {
	refused := errors.New("the disk refused to close the file")
	for _, shape := range []struct {
		format    string
		perEntity bool
	}{{"csv", true}, {"json", true}, {"json", false}} {
		rec := sample(t, shape.perEntity, 3)
		rec.Output.Format = shape.format

		err := writeWith(t, rec, &watched{refuseClose: refused})

		if !errors.Is(err, refused) {
			t.Errorf("%s, a file per table %v: a close that failed was answered with %v",
				shape.format, shape.perEntity, err)
		} else if fault.KindOf(err) != fault.IO {
			t.Errorf("%s, a file per table %v: a close that failed is %v rather than %v",
				shape.format, shape.perEntity, fault.KindOf(err), fault.IO)
		}
	}
}

// heldAfter is how much of the heap is still in use once a writer has written a
// table of one column and the given number of rows into nowhere, with the
// writer itself still reachable.
//
// The source of values is warmed first, because a word list loads on first use
// and stays, and that is a cost of the source rather than of the writer.
func heldAfter(t *testing.T, rec recipe.Recipe, rows int) uint64 {
	t.Helper()
	source := values.New(rec.Locale, rec.Seed)
	rec.Entities[0].Count = rows
	plan, err := generate.Prepare(rec, source, generate.NeverStops{})
	if err != nil {
		t.Fatalf("the plan was refused: %v", err)
	}
	warm, err := format.For(&watched{}, rec, generate.NeverStops{})
	if err != nil {
		t.Fatal(err)
	}
	if err := generate.Run(plan, rec, source, warm, generate.NeverStops{}); err != nil {
		t.Fatal(err)
	}

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	writer, err := format.For(&watched{}, rec, generate.NeverStops{})
	if err != nil {
		t.Fatal(err)
	}
	if err := generate.Run(plan, rec, source, writer, generate.NeverStops{}); err != nil {
		t.Fatal(err)
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	runtime.KeepAlive(writer)
	if after.HeapAlloc < before.HeapAlloc {
		return 0
	}
	return after.HeapAlloc - before.HeapAlloc
}

// lonelyTable is a recipe of one table nobody belongs to, with one cheap column.
func lonelyTable(form string) recipe.Recipe {
	return recipe.Recipe{
		Version: "0.1", Seed: "held", Locale: "en-US",
		Output: recipe.Output{Format: form, FilePerEntity: true},
		Entities: []recipe.Entity{{Name: "reading", HasCount: true,
			Fields: []recipe.Field{{Name: "n", Type: "integer"}}}},
	}
}

// A writer that streams keeps nothing of the rows it has written.
//
// 🔴 Both did, until 2026-10-27, and an outside review measured it on
// 2026-10-22 as the peak working set of the process at two million rows of six
// fields: 797 MiB for a file per entity in the object form, which held every
// record until the end of the run, and 199 MiB for the comma separated form,
// which kept the row number of every row of every table for children that
// might name it, a table nobody belongs to included. Measured here on
// 2026-10-27, three runs each: a hundred thousand rows of one column left
// 8.7 MB in use in the first and 3.5 MB in the second on the commit before, and
// between half a kilobyte and eleven after. The ceiling is half a megabyte.
func TestAWriterThatStreamsKeepsNothingOfTheRowsItWrote(t *testing.T) {
	for _, form := range []string{"csv", "json"} {
		if held := heldAfter(t, lonelyTable(form), 100_000); held > 512<<10 {
			t.Errorf("%s: a hundred thousand rows written into nowhere left %d bytes in use",
				form, held)
		}
	}
}
