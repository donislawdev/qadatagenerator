package manifestexport_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestexport"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// counted is a file that counts the writes it is handed.
//
// Write is all it has, like the file of a run. A fake that also took a string
// would be handed large pieces past the buffer and past this count.
type counted struct {
	kept   bytes.Buffer
	writes int
}

func (c *counted) Write(p []byte) (int, error) {
	c.writes++
	return c.kept.Write(p)
}

// takesUpTo is a file with room for so many bytes and no more.
type takesUpTo struct {
	room  int
	tried int
	kept  bytes.Buffer
}

var errNoRoom = errors.New("there is no room left on the disk")

func (f *takesUpTo) Write(p []byte) (int, error) {
	f.tried++
	if len(p) > f.room {
		taken := f.room
		f.kept.Write(p[:taken])
		f.room = 0
		return taken, errNoRoom
	}
	f.room -= len(p)
	f.kept.Write(p)
	return len(p), nil
}

// Every form reaches its file in pieces of a few kilobytes rather than one
// write per entry. Until 2026-10-28 the Gherkin and Markdown forms made a call
// to the system for every row (found by an outside review on 2026-10-22), and
// an export doubled the calls of a run that asked for one.
func TestEveryFormReachesItsFileInPiecesRatherThanAnEntryAtATime(t *testing.T) {
	const entries = 300
	for _, form := range manifestexport.Forms() {
		t.Run(form, func(t *testing.T) {
			file := &counted{}
			export := manifestexport.New(form, file, described(t), entries)
			for row := 1; row <= entries; row++ {
				if err := export.Add(applied(row, "pesel", "L3.checksum.off_by_one")); err != nil {
					t.Fatalf("entry %d was refused: %v", row, err)
				}
			}
			if err := export.Finish(); err != nil {
				t.Fatal(err)
			}
			if most := file.kept.Len()/4096 + 1; file.writes > most {
				t.Errorf("%d entries, %d bytes, reached the file in %d writes, want at most %d",
					entries, file.kept.Len(), file.writes, most)
			}
			if rows := bytes.Count(file.kept.Bytes(), []byte("off_by_one")); rows != entries {
				t.Errorf("the file carries %d of the %d rows", rows, entries)
			}
		})
	}
}

// 🔴 A form that cannot be written says so while entries still arrive, within
// one buffer of them, so that the run ends there rather than at its close. It
// says so at the entry during which the disk refused rather than at the one
// after.
func TestEveryFormSaysItCannotBeWrittenWithinOneBuffer(t *testing.T) {
	for _, form := range manifestexport.Forms() {
		t.Run(form, func(t *testing.T) {
			file := &takesUpTo{}
			export := manifestexport.New(form, file, described(t), 1000)
			for row := 1; row <= 200; row++ {
				err := export.Add(applied(row, "pesel", "L3.checksum.off_by_one"))
				if file.tried == 0 {
					if err != nil {
						t.Fatalf("entry %d was refused before the disk was tried: %v", row, err)
					}
					continue
				}
				if err == nil {
					t.Fatalf("the disk refused during entry %d and the entry was taken", row)
				}
				if !errors.Is(err, errNoRoom) || fault.KindOf(err) != fault.IO {
					t.Errorf("a full disk was reported as %v", err)
				}
				return
			}
			t.Fatal("two hundred entries, more than a buffer, were taken by a form whose file " +
				"takes nothing")
		})
	}
}

// 🔴 A form whose last lines cannot be written does not close. The buffer holds
// them until the close, and the set of files gives every file its real name
// right after it.
func TestEveryFormSaysItsLastLinesCouldNotBeWritten(t *testing.T) {
	for _, form := range manifestexport.Forms() {
		t.Run(form, func(t *testing.T) {
			file := &takesUpTo{room: 16}
			export := manifestexport.New(form, file, described(t), 1)
			if err := export.Add(applied(1, "pesel", "L3.checksum.off_by_one")); err != nil {
				t.Fatalf("an entry was refused before anything reached the file: %v", err)
			}
			err := export.Finish()
			if err == nil {
				t.Fatalf("the form closed although %d bytes of it reached the file", file.kept.Len())
			}
			if kind := fault.KindOf(err); kind != fault.IO {
				t.Errorf("a file out of room was reported as %v rather than as a failure to write", kind)
			}
		})
	}
}
