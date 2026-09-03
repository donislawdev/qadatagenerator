package boundedfile_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/boundedfile"
)

// opened writes content to a file in the test's own directory and hands back an
// open handle, because the thing under test takes one.
func opened(t *testing.T, content []byte) *os.File {
	t.Helper()
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("cannot write the file this test reads: %v", err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("cannot open the file this test reads: %v", err)
	}
	t.Cleanup(func() { _ = file.Close() })
	return file
}

func TestAFileUnderTheLimitComesBackWhole(t *testing.T) {
	for _, size := range []int{0, 1, 100, 4096} {
		content := bytes.Repeat([]byte("a"), size)
		raw, err := boundedfile.Read(opened(t, content), 4096)
		if err != nil {
			t.Fatalf("a file of %d bytes under a 4096 byte limit was refused: %v", size, err)
		}
		if !bytes.Equal(raw, content) {
			t.Fatalf("a file of %d bytes came back as %d bytes", size, len(raw))
		}
	}
}

// A file exactly at the limit is accepted. This is the edge the extra byte in
// the read exists for, and getting it wrong turns the limit into limit minus
// one without anything saying so.
func TestAFileExactlyAtTheLimitIsAccepted(t *testing.T) {
	content := bytes.Repeat([]byte("a"), 64)
	raw, err := boundedfile.Read(opened(t, content), 64)
	if err != nil {
		t.Fatalf("a file exactly at the limit was refused: %v", err)
	}
	if len(raw) != 64 {
		t.Fatalf("a file exactly at the limit came back as %d bytes", len(raw))
	}
}

func TestAFileOverTheLimitIsRefused(t *testing.T) {
	_, err := boundedfile.Read(opened(t, bytes.Repeat([]byte("a"), 65)), 64)
	if !errors.Is(err, boundedfile.ErrTooLarge) {
		t.Fatalf("a file one byte over the limit gave %v rather than ErrTooLarge", err)
	}
}

// The point of this package is that the refusal costs a stat rather than a
// read, and the only way to see that from outside is to weigh it. A reader that
// pulled the contents in and measured afterwards would pass every other test
// here and fail this one, because it would have to allocate the file to refuse
// it.
func TestAnOversizedFileIsRefusedWithoutBeingRead(t *testing.T) {
	// The limit has to be large enough that reading up to it is expensive,
	// otherwise this passes for the wrong reason: a small limit bounds the read
	// to a few bytes and a reader that read first would look thrifty too.
	const limit = 4 << 20
	const size = limit + (1 << 20)
	file := opened(t, bytes.Repeat([]byte("a"), size))

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	if _, err := boundedfile.Read(file, limit); !errors.Is(err, boundedfile.ErrTooLarge) {
		t.Fatalf("a file over the limit gave %v rather than ErrTooLarge", err)
	}
	runtime.ReadMemStats(&after)

	// Room for the stat and the error, and nowhere near room for the file. A
	// reader that took the contents in first was measured at 21 MB against this
	// 64 kB, so the two sit about three hundred times apart and this does not
	// need a tight number to mean something.
	const roomToRefuse = 64 << 10
	if spent := after.TotalAlloc - before.TotalAlloc; spent > roomToRefuse {
		t.Fatalf("refusing a %d byte file allocated %d bytes, which is more than the %d it takes "+
			"to say no. The contents were reached for before the length was", size, spent, roomToRefuse)
	}
}

// The other half of why this package exists: a file that is accepted has to
// cost its own size once. io.ReadAll grows by doubling, so it holds the old
// buffer and the new one while the last doubling copies, and the sum of every
// buffer it made along the way is several times the file.
//
// This is the arithmetic behind the measurement in the package comment, made
// small enough to run every time.
func TestAnAcceptedFileCostsItsOwnSizeRatherThanSeveral(t *testing.T) {
	const limit = 8 << 20
	const size = 4 << 20
	file := opened(t, bytes.Repeat([]byte("a"), size))

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	raw, err := boundedfile.Read(file, limit)
	runtime.ReadMemStats(&after)
	if err != nil {
		t.Fatalf("a file well under the limit was refused: %v", err)
	}
	if len(raw) != size {
		t.Fatalf("a file of %d bytes came back as %d bytes", size, len(raw))
	}

	// One buffer for the file plus room for everything else. Reading the same
	// file by doubling allocates about four times this, so the two do not sit
	// anywhere near the same side of it.
	const roomToRead = size + size/2
	if spent := after.TotalAlloc - before.TotalAlloc; spent > roomToRead {
		t.Fatalf("reading a %d byte file allocated %d bytes, which is more than the %d one "+
			"buffer takes. The length was not used to size the first allocation", size, spent, roomToRead)
	}
}

// A source whose length says nothing about how much comes out of it still has
// to be bounded, because the length is an optimisation and never the guard.
func TestASourceThatCannotStateItsLengthIsStillBounded(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("cannot make the pipe this test reads: %v", err)
	}
	t.Cleanup(func() { _ = reader.Close() })
	go func() {
		_, _ = writer.Write(bytes.Repeat([]byte("a"), 300))
		_ = writer.Close()
	}()
	if _, err := boundedfile.Read(reader, 64); !errors.Is(err, boundedfile.ErrTooLarge) {
		t.Fatalf("a pipe carrying more than the limit gave %v rather than ErrTooLarge", err)
	}
}

func TestAShortSourceComesBackWholeEvenWithoutALength(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("cannot make the pipe this test reads: %v", err)
	}
	t.Cleanup(func() { _ = reader.Close() })
	go func() {
		_, _ = writer.Write([]byte("short"))
		_ = writer.Close()
	}()
	raw, err := boundedfile.Read(reader, 64)
	if err != nil {
		t.Fatalf("a pipe carrying less than the limit was refused: %v", err)
	}
	if string(raw) != "short" {
		t.Fatalf("a pipe carrying %q came back as %q", "short", string(raw))
	}
}

// A directory reports a length that has nothing to do with reading it, so the
// length must not be used to refuse or to accept it. The read fails, and the
// failure has to be the operating system's rather than a refusal about size.
func TestADirectoryIsNotJudgedOnTheLengthItReports(t *testing.T) {
	file, err := os.Open(t.TempDir())
	if err != nil {
		t.Fatalf("cannot open the directory this test reads: %v", err)
	}
	t.Cleanup(func() { _ = file.Close() })
	_, err = boundedfile.Read(file, 64)
	if errors.Is(err, boundedfile.ErrTooLarge) || errors.Is(err, boundedfile.ErrChangedWhileRead) {
		t.Fatalf("a directory was refused as though its reported length meant something: %v", err)
	}
}
