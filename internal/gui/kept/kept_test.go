package kept

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/test"
)

// values is every key and value the preferences hold right now.
//
// The toolkit's interface has no way to list keys, and the answer this package
// owes is about keys nobody here named. The store under a test application has
// the method, so it is asked for it, and a store without it fails the test
// rather than passing it on nothing.
func values(t *testing.T, prefs fyne.Preferences) map[string]any {
	t.Helper()
	lister, ok := prefs.(interface{ ReadValues(func(map[string]any)) })
	if !ok {
		t.Fatalf("the test application's preferences (%T) cannot list their keys", prefs)
	}
	found := map[string]any{}
	lister.ReadValues(func(all map[string]any) {
		for key, value := range all {
			found[key] = value
		}
	})
	return found
}

func TestTheSizeComesBackAsItWasClosed(t *testing.T) {
	prefs := test.NewTempApp(t).Preferences()
	New(prefs, "").RememberSize(fyne.NewSize(1301.5, 702))

	size, ok := New(prefs, "").Size()
	if !ok || size != fyne.NewSize(1301.5, 702) {
		t.Fatalf("the next start reads %v (%v), want the size the window was closed at", size, ok)
	}
}

func TestAFirstStartHasNoSize(t *testing.T) {
	if size, ok := New(test.NewTempApp(t).Preferences(), "").Size(); ok {
		t.Fatalf("nothing was ever kept and the store answers %v", size)
	}
}

// A window closed while minimised reports a size with a nought in it, and that
// size read back at the next start would open a window nobody can see.
func TestASizeWithANoughtIsNeitherWrittenNorUsed(t *testing.T) {
	for _, size := range []fyne.Size{
		fyne.NewSize(0, 700), fyne.NewSize(1200, 0), fyne.NewSize(-5, 700), {},
	} {
		prefs := test.NewTempApp(t).Preferences()
		New(prefs, "").RememberSize(size)
		if len(values(t, prefs)) != 0 {
			t.Errorf("closing at %v wrote %v", size, values(t, prefs))
		}
		prefs.SetFloat(keyWidth, float64(size.Width))
		prefs.SetFloat(keyHeight, float64(size.Height))
		if got, ok := New(prefs, "").Size(); ok {
			t.Errorf("a kept %v is used as a size to open at (%v)", size, got)
		}
	}
}

// TestForgetLeavesNothingTheFileChooserWrote opens the toolkit's own file
// chooser, which writes into the same preferences, and then forgets.
//
// The first check is the test's own canary: a chooser that wrote nothing would
// leave nothing to forget, and the test would pass on an empty store whatever
// the names in toolkitKeys were. So it requires that the chooser wrote, and
// only then that Forget left no key at all, ours or the toolkit's.
func TestForgetLeavesNothingTheFileChooserWrote(t *testing.T) {
	application := test.NewTempApp(t)
	window := test.NewWindow(nil)
	defer window.Close()
	window.Resize(fyne.NewSize(900, 700))
	chooser := dialog.NewFileOpen(func(fyne.URIReadCloser, error) {}, window)
	chooser.Show()
	chooser.Hide()

	prefs := application.Preferences()
	if len(values(t, prefs)) == 0 {
		t.Fatal("the toolkit's file chooser wrote nothing, so there is nothing for Forget to be checked against")
	}
	store := New(prefs, "")
	store.RememberSize(fyne.NewSize(1200, 800))
	store.Forget()

	if left := values(t, prefs); len(left) != 0 {
		t.Fatalf("Forget left %v behind. A key the toolkit writes that toolkitKeys does not name "+
			"keeps the chooser's folder after a person was told it was forgotten", left)
	}
}

// After Forget the window still closes, and closing writes the size. Without
// the flag that write would put back what the person had just asked to be
// forgotten, a few seconds after the screen said it was gone.
func TestAfterForgetClosingWritesNothing(t *testing.T) {
	prefs := test.NewTempApp(t).Preferences()
	store := New(prefs, "")
	store.RememberSize(fyne.NewSize(1200, 800))
	store.Forget()
	store.RememberSize(fyne.NewSize(1300, 900))

	if left := values(t, prefs); len(left) != 0 {
		t.Fatalf("closing after Forget wrote %v", left)
	}
}

func TestTheFolderIsTheOneItWasGiven(t *testing.T) {
	const folder = `C:\Users\someone\AppData\Roaming\fyne\dev.donislaw.qadg`
	if got := New(test.NewTempApp(t).Preferences(), folder).Folder(); got != folder {
		t.Fatalf("Folder is %q, want %q", got, folder)
	}
}
