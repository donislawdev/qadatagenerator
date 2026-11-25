// Package recipefile reads a recipe file into the field a person edits it in,
// and writes the field back into the file in the shape the file had.
//
// # Why a package of its own
//
// The command line reads a recipe as bytes and hands them to the decoder, and
// it never writes one. The window does both, and between the two the text
// passes through a field that holds lines rather than bytes. What the field
// cannot hold has to be decided once and in one place: which files it refuses,
// how the ends of lines and the mark at the start survive the trip, and how a
// file changed on the disk behind the window is noticed.
//
// # The shape a file keeps
//
// A file written on Windows often ends its lines with a carriage return and a
// line feed, and Notepad starts it with a byte order mark. The field is given
// plain line feeds and no mark, and a save puts both back as they were, so a
// recipe kept in somebody's repository does not change on every line because
// it was opened here. A file mixing both ends of lines is given back with the
// end most of its lines had, which is what a code editor does with one.
package recipefile

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/donislawdev/qadatagenerator/internal/adapter/recipeyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/store"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// Form is how a file wrote what the field holds as plain lines.
type Form struct {
	// CRLF says the file ends its lines with a carriage return before the
	// line feed.
	CRLF bool
	// Marked says the file starts with the byte order mark of UTF-8.
	Marked bool
}

// Stamp is what a file looked like on the disk when this build last read it or
// wrote it, so that a change made by somebody else in between can be noticed
// before a save replaces it.
type Stamp struct {
	Size     int64
	Modified time.Time
}

// Opened is a recipe file as the window holds it.
type Opened struct {
	// Path is where the file is, made absolute, so that the line for a
	// terminal and a later save name the same file whatever the working
	// directory of either.
	Path string
	// Text is what goes in the field.
	Text  string
	Form  Form
	Stamp Stamp
}

// mark is the byte order mark of UTF-8.
var mark = []byte{0xEF, 0xBB, 0xBF}

// Read reads a recipe file for the field.
//
// It refuses a folder, a file larger than this build reads as a recipe, and a
// file that is not UTF-8 text, and the last two in the words the decoder uses
// for the same refusal (recipeyaml.Unreadable says why the field cannot take
// them). It reads at most one byte more than the largest recipe plus its mark,
// for the reason the command line gives: refusing a file past the limit must
// not cost the memory the limit exists to refuse.
func Read(path string) (Opened, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return Opened{}, fault.Wrap(fault.IO, err, "cannot open "+path)
	}
	file, err := os.Open(absolute)
	if err != nil {
		return Opened{}, fault.Wrap(fault.IO, err, "cannot open "+absolute).
			DoInstead("check that the file exists and that this account may read it")
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return Opened{}, fault.Wrap(fault.IO, err, "cannot open "+absolute)
	}
	if info.IsDir() {
		return Opened{}, fault.New(fault.IO, absolute+" is a folder rather than a recipe file").
			DoInstead("choose the recipe file inside it")
	}
	data, err := io.ReadAll(io.LimitReader(file, int64(recipeyaml.MaxSize+len(mark)+1)))
	if err != nil {
		return Opened{}, fault.Wrap(fault.IO, err, "cannot read "+absolute)
	}
	if problem, refused := recipeyaml.Unreadable(data); refused {
		return Opened{}, fault.New(fault.Config, filepath.Base(absolute)+" cannot be opened: "+problem.What).
			DoInstead(problem.Hint)
	}
	text, form := plain(data)
	return Opened{Path: absolute, Text: text, Form: form, Stamp: stampOf(info)}, nil
}

// plain takes the mark off the bytes and the carriage returns off the ends of
// the lines, and says what it took.
func plain(data []byte) (string, Form) {
	var form Form
	if bytes.HasPrefix(data, mark) {
		form.Marked = true
		data = data[len(mark):]
	}
	text := string(data)
	both := strings.Count(text, "\r\n")
	// The ends of lines most of the file has, a tie going to the plain line
	// feed the field holds.
	form.CRLF = both > strings.Count(text, "\n")-both
	return strings.ReplaceAll(text, "\r\n", "\n"), form
}

// Write writes the text of the field into a file in the given form, whole or
// not at all, and says what the file looks like afterwards.
//
// The text is written whatever it says and however large it is. A recipe this
// build would refuse to run is still somebody's work, and refusing to keep it
// would lose it.
func Write(path, text string, form Form) (Stamp, error) {
	data := strings.ReplaceAll(text, "\r\n", "\n")
	if form.CRLF {
		data = strings.ReplaceAll(data, "\n", "\r\n")
	}
	written := []byte(data)
	if form.Marked {
		written = append(append([]byte{}, mark...), written...)
	}
	if err := store.Replace(path, written); err != nil {
		return Stamp{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		// Written and then gone, or unreadable for a moment. The file is in
		// place, so this is not a failure of the save, and an empty stamp
		// means the next save asks before replacing whatever is there.
		return Stamp{}, nil
	}
	return stampOf(info), nil
}

// Changed reports whether the file is no longer what the stamp describes.
//
// A file that is gone has not changed in the sense that matters here: there is
// nothing on the disk a save would replace.
func Changed(path string, since Stamp) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	now := stampOf(info)
	// Equal rather than ==, which would also compare the location the two
	// times are read in.
	return now.Size != since.Size || !now.Modified.Equal(since.Modified)
}

func stampOf(info os.FileInfo) Stamp {
	return Stamp{Size: info.Size(), Modified: info.ModTime()}
}
