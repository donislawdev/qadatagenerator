package outward

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
)

// shelf is a file manager that remembers which folders it was asked to show
// and answers with what the test set.
type shelf struct {
	shown  []string
	answer error
}

func (s *shelf) ShowFolder(path string) error {
	s.shown = append(s.shown, path)
	return s.answer
}

// pressed makes the button for one folder, presses it as a person would, and
// says what the file manager was handed and what the dialog said.
func pressed(t *testing.T, manager FileManager, folder string) (title, message string) {
	t.Helper()
	folders := NewFolders(i18n.English, manager, background.ReadHere,
		func(said, body string) { title, message = said, body })
	button := folders.Button(func() string { return folder })
	if button.Text != i18n.English.T(i18n.FolderOpen) {
		t.Fatalf("the button says %q", button.Text)
	}
	test.Tap(button)
	return title, message
}

// The button hands the file manager the folder of the last run, made whole,
// and says nothing when it opened.
//
// The folder is written without its root, the way the field on the recipe
// screen may hold it, because that is the case where the file manager, a
// program standing somewhere else, would look in the wrong place if handed the
// path as typed.
func TestTheFolderButtonShowsTheFolderOfTheRunMadeWhole(t *testing.T) {
	// The canary: a file manager that was handed nothing has to read as a
	// failure of this check, or the check below would pass on a button that
	// does nothing.
	if handedOnly(nil, "x") {
		t.Fatal("an empty record passed as one folder shown, so this test could not fail")
	}

	manager := &shelf{}
	title, _ := pressed(t, manager, "out")
	whole, err := filepath.Abs("out")
	if err != nil {
		t.Fatal(err)
	}
	if !handedOnly(manager.shown, whole) {
		t.Errorf("the file manager was handed %q, want only %q", manager.shown, whole)
	}
	if title != "" {
		t.Errorf("a folder that opened was reported as refused: %q", title)
	}
}

// handedOnly reports whether the file manager was asked for exactly one
// folder, and that one.
func handedOnly(shown []string, want string) bool {
	return len(shown) == 1 && shown[0] == want
}

// A folder that is gone is said in the window's own words with the whole path,
// and a folder the system would not open is said with the system's reason, so
// a button that did nothing is never silent.
func TestAFolderThatDidNotOpenIsSaidWithItsPath(t *testing.T) {
	whole, err := filepath.Abs("gone")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		answer error
		says   []string
		never  []string
	}{
		{"gone", ErrNoFolder,
			[]string{i18n.English.T(i18n.FolderRefusedMissing), whole},
			[]string{i18n.English.T(i18n.FolderRefusedReason)}},
		{"refused by the system", errors.New("the shell did not open the folder, ShellExecuteW answered 5"),
			[]string{i18n.English.T(i18n.FolderRefusedBody), whole, i18n.English.T(i18n.FolderRefusedReason),
				"ShellExecuteW answered 5"},
			[]string{i18n.English.T(i18n.FolderRefusedMissing)}},
	}
	for _, each := range cases {
		t.Run(each.name, func(t *testing.T) {
			title, message := pressed(t, &shelf{answer: each.answer}, "gone")
			if title != i18n.English.T(i18n.FolderRefusedTitle) {
				t.Errorf("the title is %q", title)
			}
			for _, part := range each.says {
				if !strings.Contains(message, part) {
					t.Errorf("the message does not say %q:\n%s", part, message)
				}
			}
			for _, part := range each.never {
				if strings.Contains(message, part) {
					t.Errorf("the message says %q, which belongs to the other case:\n%s", part, message)
				}
			}
		})
	}
}
