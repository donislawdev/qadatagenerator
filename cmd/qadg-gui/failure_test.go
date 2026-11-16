package main

import (
	"bytes"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
)

// TestAFailureGoesToADialogOnlyWhenNobodyHearsTheChannel holds the one
// decision aloud makes. The channel always gets the message, and the dialog
// gets the same message only when that channel leads nowhere, because a dialog
// waits for a click and a script reading the channel would wait with it.
func TestAFailureGoesToADialogOnlyWhenNobodyHearsTheChannel(t *testing.T) {
	for _, heard := range []bool{true, false} {
		var channel bytes.Buffer
		var shown []string
		aloud(heard, &channel, func(title, message string) { shown = append(shown, title, message) },
			"Title", "what went wrong")
		if channel.String() != "what went wrong\n" {
			t.Errorf("heard=%v: the channel got %q and should get the message on a line of its own", heard, channel.String())
		}
		switch {
		case heard && shown != nil:
			t.Errorf("heard=%v: a dialog was shown with %q while somebody reads the channel", heard, shown)
		case !heard && (len(shown) != 2 || shown[0] != "Title" || shown[1] != "what went wrong"):
			t.Errorf("heard=%v: the dialog got %q and should get the title and the same message", heard, shown)
		}
	}
}

// nativeWindow is a window that answers RunNative with the context it holds.
type nativeWindow struct {
	fyne.Window
	context any
}

func (w nativeWindow) RunNative(run func(any)) { run(w.context) }

// TestAWindowWithoutANativeHandleDidNotOpen holds how windowOpened reads each
// context Fyne hands RunNative: a zero handle is a window the system did not
// make, on every desktop system, and anything it cannot read counts as open.
func TestAWindowWithoutANativeHandleDidNotOpen(t *testing.T) {
	cases := []struct {
		name   string
		window fyne.Window
		opened bool
	}{
		{"Windows, no handle", nativeWindow{context: driver.WindowsWindowContext{}}, false},
		{"Windows, a handle", nativeWindow{context: driver.WindowsWindowContext{HWND: 7}}, true},
		{"macOS, no handle", nativeWindow{context: driver.MacWindowContext{}}, false},
		{"macOS, a handle", nativeWindow{context: driver.MacWindowContext{NSWindow: 7}}, true},
		{"X11, no handle", nativeWindow{context: driver.X11WindowContext{}}, false},
		{"X11, a handle", nativeWindow{context: driver.X11WindowContext{WindowHandle: 7}}, true},
		{"Wayland, no handle", nativeWindow{context: driver.WaylandWindowContext{}}, false},
		{"Wayland, a handle", nativeWindow{context: driver.WaylandWindowContext{WaylandSurface: 7}}, true},
		{"a context nobody here knows", nativeWindow{context: driver.UnknownContext{}}, true},
		{"a window with no native side", struct{ fyne.Window }{}, true},
	}
	for _, each := range cases {
		if got := windowOpened(each.window); got != each.opened {
			t.Errorf("%s: windowOpened says %v and should say %v", each.name, got, each.opened)
		}
	}
}

// showing is a window whose Show does what the test asks before the native
// side is read: log a reason the way Fyne does, panic, or nothing.
type showing struct {
	nativeWindow
	logs   string
	panics any
}

func (w showing) Show() {
	if w.logs != "" {
		fyne.LogError("window creation error", errors.New(w.logs))
	}
	if w.panics != nil {
		panic(w.panics)
	}
}

// TestOpensReadsTheWindowAndNotTheWords holds the two ways the toolkit says no
// and the one way a panic stays a defect. A reason logged with no window and a
// panic with no window are both a window that did not open, with the words
// among the reasons. A panic with a window standing is handed back as a
// defect, because the system did make the window and something of ours broke.
func TestOpensReadsTheWindowAndNotTheWords(t *testing.T) {
	none, made := driver.WindowsWindowContext{}, driver.WindowsWindowContext{HWND: 7}
	refused := errors.New("NotInitialized: The GLFW library is not initialized")
	cases := []struct {
		name   string
		window showing
		opened bool
		causes []string
		defect any
	}{
		{"a window", showing{nativeWindow: nativeWindow{context: made}}, true, nil, nil},
		{"a logged reason and no window",
			showing{nativeWindow: nativeWindow{context: none}, logs: "APIUnavailable: WGL: no OpenGL"},
			false, []string{"APIUnavailable: WGL: no OpenGL"}, nil},
		{"a panic and no window", showing{nativeWindow: nativeWindow{context: none}, panics: refused},
			false, []string{refused.Error()}, nil},
		{"a panic with a window standing", showing{nativeWindow: nativeWindow{context: made}, panics: refused},
			true, nil, refused},
	}
	logged := log.Writer()
	t.Cleanup(func() { log.SetOutput(logged) })
	for _, each := range cases {
		log.SetOutput(&bytes.Buffer{})
		opened, causes, defect := opens(each.window, listen())
		if opened != each.opened || strings.Join(causes, "|") != strings.Join(each.causes, "|") || defect != each.defect {
			t.Errorf("%s: opens gave %v, %q, %v and should give %v, %q, %v",
				each.name, opened, causes, defect, each.opened, each.causes, each.defect)
		}
	}
}

// TestTheToolkitStillNamesTheCause logs through Fyne's own function, so an
// upgrade that changes how Fyne words a failure fails here instead of quietly
// dropping the system's reason from the message. The reason is the one Fyne
// logged on 2026-11-16 on a machine without OpenGL.
func TestTheToolkitStillNamesTheCause(t *testing.T) {
	const reason = "APIUnavailable: WGL: The driver does not appear to support OpenGL"
	logged := log.Writer()
	t.Cleanup(func() { log.SetOutput(logged) })
	said := &toolkitWords{}
	log.SetOutput(said)
	fyne.LogError("window creation error", errors.New(reason))
	log.SetOutput(logged)

	if got := said.causes(); len(got) != 1 || got[0] != reason {
		t.Fatalf("the reason read out of Fyne's log is %q and should be [%q]", got, reason)
	}
}

// refusing is an error channel that leads nowhere, the way the logger's own
// writer is after a double click on Windows.
type refusing struct{}

func (refusing) Write([]byte) (int, error) { return 0, errors.New("the handle is invalid") }

// TestTheReasonIsKeptWhenTheChannelLeadsNowhere starts listening while the
// logger writes to a channel that refuses every write, and asks that the
// reason is kept anyway. A planted listener that puts the channel first is
// asked the same and has to lose the reason, so the test can tell the order
// apart rather than pass on any order.
func TestTheReasonIsKeptWhenTheChannelLeadsNowhere(t *testing.T) {
	const reason = "APIUnavailable: WGL: The driver does not appear to support OpenGL"
	logged := log.Writer()
	t.Cleanup(func() { log.SetOutput(logged) })

	log.SetOutput(refusing{})
	planted := &toolkitWords{}
	log.SetOutput(io.MultiWriter(refusing{}, planted))
	fyne.LogError("window creation error", errors.New(reason))
	if got := planted.causes(); len(got) != 0 {
		t.Fatalf("a listener behind a refusing channel kept %q, so this test cannot see the order", got)
	}

	log.SetOutput(refusing{})
	toolkit := listen()
	fyne.LogError("window creation error", errors.New(reason))
	if got := toolkit.done(); len(got) != 1 || got[0] != reason {
		t.Fatalf("with the channel leading nowhere the reason kept is %q and should be [%q]", got, reason)
	}
}

// TestWhatTheToolkitSaidIsKeptWithinItsLimit writes far more than is kept and
// asks that the start survives, the rest is dropped, and every write still
// reports success, because a logger that sees a failed write stops nobody.
func TestWhatTheToolkitSaidIsKeptWithinItsLimit(t *testing.T) {
	said := &toolkitWords{}
	first := "  Cause: the first reason\n"
	for _, chunk := range []string{first, strings.Repeat("x", toolkitWordsKept), "  Cause: too late\n"} {
		if n, err := said.Write([]byte(chunk)); n != len(chunk) || err != nil {
			t.Fatalf("a write of %d bytes answered %d, %v", len(chunk), n, err)
		}
	}
	if len(said.kept) != toolkitWordsKept {
		t.Errorf("%d bytes kept and the limit is %d", len(said.kept), toolkitWordsKept)
	}
	if got := said.causes(); len(got) != 1 || got[0] != "the first reason" {
		t.Errorf("the reasons kept are %q and should be only the first one", got)
	}
}

// TestTheWindowIsShownApartFromItsLoop reads this package's own files and asks
// that nothing calls ShowAndRun and that something calls showOrStop.
// ShowAndRun runs the loop whether or not the system made a window, which is
// the process with no window B-413 removed, and every other test here would
// stay green if it came back. Planted sources first, so a check that sees
// nothing cannot pass for one that found nothing.
func TestTheWindowIsShownApartFromItsLoop(t *testing.T) {
	planted := map[string]bool{
		"package main\nfunc main() { w.ShowAndRun() }\n":                true,
		"package main\nfunc main() { showOrStop(w, t, text, heard) }\n": false,
	}
	for source, calls := range planted {
		if got := callsNamed(t, "planted.go", []byte(source), "ShowAndRun"); got != calls {
			t.Fatalf("the check reads ShowAndRun in a planted file as %v and it is %v:\n%s", got, calls, source)
		}
	}

	sources, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	shown := false
	for _, name := range sources {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if callsNamed(t, name, source, "ShowAndRun") {
			t.Errorf("%s calls ShowAndRun, which runs the loop even when the system made no window", name)
		}
		shown = shown || callsNamed(t, name, source, "showOrStop")
	}
	if !shown {
		t.Error("no file of this package calls showOrStop, so nothing stops a window the system did not make")
	}
}

// callsNamed reports whether source calls a function or method with this name.
func callsNamed(t *testing.T, name string, source []byte, called string) bool {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), name, source, 0)
	if err != nil {
		t.Fatalf("%s does not parse: %v", name, err)
	}
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fun := call.Fun.(type) {
		case *ast.Ident:
			found = found || fun.Name == called
		case *ast.SelectorExpr:
			found = found || fun.Sel.Name == called
		}
		return true
	})
	return found
}

// TestTheMessageCarriesTheReasonOnlyWhenThereIsOne holds the shape of what a
// window that did not open says: what happened, then the system's words under
// their heading when the toolkit logged any, then what to do.
func TestTheMessageCarriesTheReasonOnlyWhenThereIsOne(t *testing.T) {
	text := i18n.English
	body, heading, do := text.T(i18n.WindowNotOpened), text.T(i18n.WindowNotOpenedReason), text.T(i18n.WindowNotOpenedDo)

	with := notOpened(text, []string{"first", "second"})
	if want := body + "\n\n" + heading + "\nfirst\nsecond\n\n" + do; with != want {
		t.Errorf("with two reasons the message is\n%q\nand should be\n%q", with, want)
	}
	without := notOpened(text, nil)
	if want := body + "\n\n" + do; without != want {
		t.Errorf("with no reason the message is\n%q\nand should be\n%q", without, want)
	}
}
