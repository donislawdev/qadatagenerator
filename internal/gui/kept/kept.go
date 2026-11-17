// Package kept is the little the window keeps on this computer between runs:
// how big it was when it was closed, and the folder its file chooser was last
// in. The owner asked for both on 2026-11-17, together with a way to forget
// them (DEC-325).
//
// # Where it is kept, and why there rather than in a file of ours
//
// In the toolkit's preferences, which live in a folder named by the
// application's identifier under the user's settings (cmd/qadg-gui says which
// identifier and why it must not change). The toolkit's own file chooser
// writes its last folder there the moment the window has an identifier, so a
// file of ours would be a second file on somebody's disk to hold two numbers.
//
// The toolkit writes that file by truncating it in place rather than by
// replacing it (fyne v2.8.2, app/preferences_nonweb.go), so a machine that
// stops mid write leaves a short file. The next start then says on the error
// channel that the preferences did not load, and opens the way a first start
// does. What is lost is a window size and a folder, which is why this file is
// not held to the rule for the data a person makes with this tool.
//
// # Why it is a type of its own rather than calls on the toolkit's preferences
//
// The window and the About screen both reach it, and the promise Forget makes
// has to hold for both: once a person has pressed it, closing the window must
// not write the size back. A flag that lives beside the keys is the one place
// that promise can be kept and checked without a window.
package kept

import "fyne.io/fyne/v2"

// The keys this window writes. Renaming either after a release leaves every
// remembered size behind under the old name, so they are spelled once, here.
const (
	keyWidth  = "windowWidth"
	keyHeight = "windowHeight"
)

// toolkitKeys are what the toolkit's file chooser writes into the same file:
// the folder it was last in and how it lays files out (fyne v2.8.2,
// dialog/file.go:35-36).
//
// Forget removes them as well as ours. A person who pressed Forget and then saw
// the chooser open where it was last would have been told something untrue.
// The names belong to the toolkit and can change under an upgrade, which is
// why TestForgetLeavesNothingTheFileChooserWrote opens the toolkit's own
// chooser and holds these against what it really writes.
var toolkitKeys = []string{"fyne:fileDialogLastFolder", "fyne:fileDialogViewLayout"}

// Store is what the window keeps, over the toolkit's preferences.
//
// Every method runs on the thread that draws: the size is read before the
// window is shown, written as it closes, and Forget is the press of a button.
type Store struct {
	prefs  fyne.Preferences
	folder string
	forgot bool
}

// New is a store over the given preferences. folder is where the toolkit keeps
// them on this computer, as the system spells it, which the About screen shows.
func New(prefs fyne.Preferences, folder string) *Store {
	return &Store{prefs: prefs, folder: folder}
}

// Size is how big the window was when it was last closed, and whether there is
// such a size. A first start, a size forgotten and a size with a nought in it
// all answer false, and the window then opens at the size a first start does.
func (s *Store) Size() (fyne.Size, bool) {
	size := fyne.NewSize(float32(s.prefs.Float(keyWidth)), float32(s.prefs.Float(keyHeight)))
	return size, Worth(size)
}

// RememberSize writes down how big the window is as it closes.
//
// Nothing is written after Forget, or closing the window would put back what
// the person just asked to be forgotten. Nothing is written for a size with a
// nought in it either, which is what a window closed while minimised reports.
func (s *Store) RememberSize(size fyne.Size) {
	if s.forgot || !Worth(size) {
		return
	}
	s.prefs.SetFloat(keyWidth, float64(size.Width))
	s.prefs.SetFloat(keyHeight, float64(size.Height))
}

// Folder is where all of this is kept on this computer.
func (s *Store) Folder() string { return s.folder }

// Forget removes everything this window and its file chooser have kept, and
// stops the size being written again for the rest of this run.
//
// The values go and the file stays. The file is the toolkit's, and the folder
// it is in is created again at every start whether anything is kept or not.
func (s *Store) Forget() {
	s.forgot = true
	s.prefs.RemoveValue(keyWidth)
	s.prefs.RemoveValue(keyHeight)
	for _, key := range toolkitKeys {
		s.prefs.RemoveValue(key)
	}
}

// Worth says whether a size is one a window can be opened at.
//
// Exported so that the one rule is asked at both ends, before a size is written
// and before one is used, because two copies of "is this a size" is how a
// window comes back as nothing on one machine and not on another.
func Worth(size fyne.Size) bool {
	return size.Width > 0 && size.Height > 0
}
