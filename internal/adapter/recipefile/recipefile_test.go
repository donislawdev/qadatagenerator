package recipefile_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/donislawdev/qadatagenerator/internal/adapter/recipefile"
	"github.com/donislawdev/qadatagenerator/internal/adapter/recipeyaml"
)

// put writes bytes for a test and fails it if they cannot be written.
func put(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// 🔴 A file opened and saved keeps the ends of its lines and its mark.
//
// A recipe kept in a repository on Windows usually has both, and a save that
// dropped them would change every line of it in the next commit for nobody's
// reason. The canary writes the same text in the plain form and requires the
// bytes to differ, so the comparison is shown able to fail.
func TestASavedFileKeepsItsLineEndsAndItsMark(t *testing.T) {
	path := filepath.Join(t.TempDir(), "customers.yaml")
	put(t, path, []byte("\xEF\xBB\xBFversion: \"0.1\"\r\nseed: s\r\n"))

	opened, err := recipefile.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if opened.Text != "version: \"0.1\"\nseed: s\n" {
		t.Fatalf("the field was given %q rather than plain lines without the mark", opened.Text)
	}
	if !opened.Form.CRLF || !opened.Form.Marked {
		t.Fatalf("the form read is %+v, but the file has both", opened.Form)
	}

	edited := opened.Text + "locale: pl-PL\n"
	if _, err := recipefile.Write(path, edited, opened.Form); err != nil {
		t.Fatal(err)
	}
	want := "\xEF\xBB\xBFversion: \"0.1\"\r\nseed: s\r\nlocale: pl-PL\r\n"
	if got := string(read(t, path)); got != want {
		t.Errorf("saved %q, want %q", got, want)
	}

	plainPath := filepath.Join(t.TempDir(), "plain.yaml")
	if _, err := recipefile.Write(plainPath, edited, recipefile.Form{}); err != nil {
		t.Fatal(err)
	}
	if string(read(t, plainPath)) == want {
		t.Error("canary: the plain form wrote the same bytes, so the comparison above proves nothing")
	}
}

// A file whose lines mostly end one way is given back ending that way.
func TestTheEndMostLinesHaveDecides(t *testing.T) {
	for _, tc := range []struct {
		name string
		data string
		crlf bool
	}{
		{"plain", "a: 1\nb: 2\n", false},
		{"windows", "a: 1\r\nb: 2\r\n", true},
		{"mostly windows", "a: 1\r\nb: 2\r\nc: 3\n", true},
		{"mostly plain", "a: 1\r\nb: 2\nc: 3\n", false},
		{"a tie", "a: 1\r\nb: 2\n", false},
		{"one line", "a: 1", false},
		{"nothing", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "r.yaml")
			put(t, path, []byte(tc.data))
			opened, err := recipefile.Read(path)
			if err != nil {
				t.Fatal(err)
			}
			if opened.Form.CRLF != tc.crlf {
				t.Errorf("CRLF read as %v", opened.Form.CRLF)
			}
			if strings.Contains(opened.Text, "\r\n") {
				t.Errorf("the field was given %q, which still ends lines with a carriage return", opened.Text)
			}
		})
	}
}

// A file the field cannot hold faithfully is refused, in the decoder's words.
func TestAFileTheFieldCannotHoldIsRefused(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range []struct {
		name string
		data []byte
		says string
	}{
		{"windows-1250", []byte("name: za\xbf\xf3\xb3\xe6\n"), "not valid UTF-8"},
		{"utf-16", []byte("\xFF\xFEv\x00:\x00"), "UTF-16 little endian"},
		{"too large", []byte(strings.Repeat("a", recipeyaml.MaxSize+1)), "larger than"},
		{"too large behind a mark", append([]byte("\xEF\xBB\xBF"),
			strings.Repeat("a", recipeyaml.MaxSize+1)...), "larger than"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name+".yaml")
			put(t, path, tc.data)
			_, err := recipefile.Read(path)
			if err == nil || !strings.Contains(err.Error(), tc.says) {
				t.Errorf("read answered %v, want a refusal saying %q", err, tc.says)
			}
		})
	}

	// The largest recipe is still a recipe, mark or no mark: the mark is not
	// counted against the limit, as the decoder does not count it.
	largest := filepath.Join(dir, "largest.yaml")
	put(t, largest, append([]byte("\xEF\xBB\xBF"), strings.Repeat("a", recipeyaml.MaxSize)...))
	if _, err := recipefile.Read(largest); err != nil {
		t.Errorf("a recipe of exactly the limit behind a mark was refused: %v", err)
	}

	if _, err := recipefile.Read(dir); err == nil || !strings.Contains(err.Error(), "folder") {
		t.Errorf("a folder was answered with %v rather than refused as one", err)
	}
}

// A change made on the disk behind the window is noticed, and a file that is
// gone is not a change anybody loses work to.
func TestAChangeBehindTheWindowIsNoticed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "r.yaml")
	put(t, path, []byte("a: 1\n"))
	opened, err := recipefile.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if recipefile.Changed(path, opened.Stamp) {
		t.Fatal("an untouched file reads as changed")
	}

	put(t, path, []byte("a: 22\n"))
	later := time.Now().Add(time.Minute)
	if err := os.Chtimes(path, later, later); err != nil {
		t.Fatal(err)
	}
	if !recipefile.Changed(path, opened.Stamp) {
		t.Error("a file written by somebody else reads as unchanged")
	}

	saved, err := recipefile.Write(path, "a: 3\n", opened.Form)
	if err != nil {
		t.Fatal(err)
	}
	if recipefile.Changed(path, saved) {
		t.Error("the file this build has just saved reads as changed")
	}

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if recipefile.Changed(path, saved) {
		t.Error("a file that is gone reads as changed")
	}
}

// 🔴 A save puts a new file under the name rather than writing into the old
// one, which is what leaves the old one whole when the save fails half way.
//
// Asked of the file system rather than of the code: the file under the name
// after the save is a different file from the one before it. A save writing
// into the old file in place would leave it the same file, and the failure
// half way through would leave it half written.
func TestASaveReplacesTheFileRatherThanWritingIntoIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "r.yaml")
	put(t, path, []byte("a: 1\n"))
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// On Windows the identity of a file is read by its path the first time it
	// is compared, so without this both answers would describe whatever is
	// under the name after the save and always match (seen 2026-11-25).
	os.SameFile(before, before)
	if _, err := recipefile.Write(path, "a: 2\n", recipefile.Form{}); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(before, after) {
		t.Error("the save wrote into the file in place, so a failure half way would leave it half written")
	}
	leftovers, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(leftovers) != 0 {
		t.Errorf("the save left %v beside the file", leftovers)
	}
}

// A file marked read-only is refused rather than replaced, and stays as it was.
func TestAReadOnlyFileIsNotReplaced(t *testing.T) {
	path := filepath.Join(t.TempDir(), "r.yaml")
	put(t, path, []byte("a: 1\n"))
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
	if _, err := recipefile.Write(path, "a: 2\n", recipefile.Form{}); err == nil ||
		!strings.Contains(err.Error(), "read-only") {
		t.Errorf("a read-only file was answered with %v", err)
	}
	if string(read(t, path)) != "a: 1\n" {
		t.Error("the read-only file was changed")
	}
}

// The mode of the file being replaced is kept, so a recipe a build account
// could read stays readable to it.
func TestASaveKeepsTheModeOfTheFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows keeps one bit of the mode, read-only, which the test above holds")
	}
	path := filepath.Join(t.TempDir(), "r.yaml")
	put(t, path, []byte("a: 1\n"))
	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := recipefile.Write(path, "a: 2\n", recipefile.Form{}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Errorf("the file is %v after the save, it was 0640", info.Mode().Perm())
	}
}

// A name that is a link keeps being a link, and the file it leads to is the one
// that changes.
func TestASaveThroughALinkChangesTheFileItLeadsTo(t *testing.T) {
	dir := t.TempDir()
	target, link := filepath.Join(dir, "real.yaml"), filepath.Join(dir, "link.yaml")
	put(t, target, []byte("a: 1\n"))
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("this account cannot make a link here: %v", err)
	}
	if _, err := recipefile.Write(link, "a: 2\n", recipefile.Form{}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("the link was replaced by a file")
	}
	if string(read(t, target)) != "a: 2\n" {
		t.Error("the file the link leads to was not changed")
	}
}
