package format_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/format"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// What these two hold is what changed on 2026-09-03, when the object writer
// stopped assembling the whole document in memory and started putting it
// through a buffer.
//
// A buffer keeps the first failure to itself and turns every later write into
// nothing, which is what makes writing twenty pieces without checking twenty
// results correct. It is also what makes forgetting the one check at the end a
// silent truncation instead of a compile error, and a run that reports a file
// it could not write is the failure this project is written against.

// takesOnly is a file that accepts a fixed number of bytes and then refuses,
// which is what a full disk looks like from inside a writer.
type takesOnly struct {
	left    int
	written strings.Builder
}

func (t *takesOnly) Write(p []byte) (int, error) {
	if len(p) > t.left {
		taken := t.left
		t.written.Write(p[:taken])
		t.left = 0
		return taken, errDiskFull{}
	}
	t.written.Write(p)
	t.left -= len(p)
	return len(p), nil
}

func (t *takesOnly) Close() error { return nil }

type errDiskFull struct{}

func (errDiskFull) Error() string { return "no space left on device" }

// oneFile hands the same file to whoever asks, whatever the name.
type oneFile struct{ file *takesOnly }

func (o oneFile) Create(string) (format.WriteCloser, error) { return o.file, nil }

// writeObjects runs a recipe through the object writer and hands back what the
// run said, rather than failing the test the way the helper beside it does.
func writeObjects(t *testing.T, rec recipe.Recipe, files format.Files, stop generate.Halt) error {
	t.Helper()
	source := values.New(rec.Locale, rec.Seed)
	plan, err := generate.Prepare(rec, source, generate.NeverStops{})
	if err != nil {
		t.Fatalf("the plan was refused: %v", err)
	}
	writer, err := format.NewJSON(files, rec, stop)
	if err != nil {
		t.Fatalf("the writer was refused: %v", err)
	}
	return generate.Run(plan, rec, source, writer, generate.NeverStops{})
}

// 🔴 The four cases are two shapes against two moments. The shape matters
// because one writes at the end of each table and the other writes everything
// at the end of the run, and a check remembered in one of them is exactly how
// the two would drift. The moment matters more: with a budget of nothing the
// failure arrives when the buffer is emptied, and with a budget smaller than
// the document it arrives while the walk is still going and has to be
// remembered until the end. Only the second one proves the buffer keeps it.
func TestAFileThatCouldNotBeWrittenIsNeverReportedAsWritten(t *testing.T) {
	cases := []struct {
		name      string
		perEntity bool
		parents   int
		children  int
		budget    int
	}{
		{name: "a file per table, refused at the flush",
			perEntity: true, parents: 4, children: 4, budget: 0},
		{name: "a file per table, refused during the walk",
			perEntity: true, parents: 200, children: 400, budget: 100},
		{name: "one file, refused at the flush",
			perEntity: false, parents: 4, children: 4, budget: 0},
		{name: "one file, refused during the walk",
			perEntity: false, parents: 200, children: 400, budget: 100},
	}
	for _, one := range cases {
		t.Run(one.name, func(t *testing.T) {
			// The parents are raised as well as the children, because a child
			// may not ask for more rows than its parents can share out, and
			// what this case needs is a document longer than one buffer.
			rec := sample(t, one.perEntity, one.children)
			rec.Entities[0].Count = one.parents
			rec.Output.Format = "json"
			file := &takesOnly{left: one.budget}

			err := writeObjects(t, rec, oneFile{file: file}, generate.NeverStops{})

			if err == nil {
				t.Fatalf("the disk took %d bytes of a document of %d and the run said it "+
					"was written", one.budget, file.written.Len())
			}
			if kind := fault.KindOf(err); kind != fault.IO {
				t.Errorf("a disk that stopped taking bytes is a %v and this run called it %v: %v",
					fault.IO, kind, err)
			}
			if !errors.Is(err, errDiskFull{}) {
				t.Errorf("the run answered %q, which does not carry what the disk said", err)
			}
		})
	}
}

// stopsAfter answers `carry on` a fixed number of times and then ends the run,
// which is somebody pressing the button while the document is being written.
type stopsAfter struct {
	left int
	why  error
}

func (s *stopsAfter) Stopped() error {
	if s.left > 0 {
		s.left--
		return nil
	}
	return s.why
}

// 🔴 The run itself is given a halt that never stops, so what is under test is
// the writing and not the producing. The tree is the only shape where those two
// are different moments: every record has already been made and the file is
// still being put together. Until 2026-10-27 the file per entity was one as
// well, because it held its records until the table ended. It writes each one
// as it arrives now, so the question the run asks before every record is the
// one that stops it, and asking the writer's own here would be asking a
// question that shape no longer has.
//
// The document that was left half written is not checked for its shape, and
// that is deliberate. Nothing keeps it: the bytes are under a temporary name
// and the caller throws them away, which is the arrangement the store holds and
// its own tests state.
func TestAStopWhileTheObjectsAreWrittenEndsTheRunRatherThanTheFile(t *testing.T) {
	rec := sample(t, false, 40)
	rec.Entities[0].Count = 40
	rec.Output.Format = "json"
	asked := errors.New("the window asked this run to stop")
	files := newHeld()

	err := writeObjects(t, rec, files, &stopsAfter{left: 3, why: asked})

	if !errors.Is(err, asked) {
		t.Fatalf("the tree was stopped and the run answered %v", err)
	}
	for name, built := range files.written {
		if strings.HasSuffix(built.String(), "]\n") || strings.HasSuffix(built.String(), "}\n") {
			t.Errorf("%s was finished off although the run was stopped part way", name)
		}
	}
}
