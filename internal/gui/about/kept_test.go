package about

import (
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	"github.com/donislawdev/qadatagenerator/internal/license"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

// keeping stands in for the window's store: it says where, and counts the
// times it was asked to forget.
type keeping struct{ forgot int }

func (k *keeping) Folder() string { return `C:\Users\someone\AppData\Roaming\fyne\dev.donislaw.qadg` }
func (k *keeping) Forget()        { k.forgot++ }

// forgetButton is the button on the screen that forgets, or nil.
func forgetButton(t *testing.T, kept Kept) (*parts.Button, []string) {
	t.Helper()
	screen := Content(i18n.English, license.Current(), version.Current(), false, nil, kept)
	for _, button := range buttons(screen) {
		if button.Text == i18n.English.T(i18n.AboutForget) {
			return button, texts(screen)
		}
	}
	return nil, texts(screen)
}

// The screen says what the window keeps and in which folder, and Forget does
// what it says: it asks the store once, can then not be pressed again, and the
// line under it says what happened. The screen built with nothing kept is the
// canary of the first half: a section drawn whatever it was handed would pass
// the first half alone, and would show a folder that is not there.
func TestTheScreenSaysWhatTheWindowKeepsAndForgetsIt(t *testing.T) {
	kept := &keeping{}
	forget, all := forgetButton(t, kept)
	for _, want := range []string{
		i18n.English.T(i18n.AboutHeadingKept), i18n.English.T(i18n.AboutKeptWhat),
		i18n.English.T(i18n.AboutKeptIn), kept.Folder(), i18n.English.T(i18n.AboutForgetWhat),
	} {
		if !has(all, want) {
			t.Errorf("the screen does not say %q", want)
		}
	}
	if forget == nil || forget.Disabled() {
		t.Fatalf("there is no Forget button to press (found %v)", forget)
	}

	forget.OnTapped()
	if kept.forgot != 1 {
		t.Errorf("one press asked the store to forget %d times", kept.forgot)
	}
	if !forget.Disabled() {
		t.Error("Forget can still be pressed after it has forgotten everything it can for this run")
	}

	if absent, all := forgetButton(t, nil); absent != nil || has(all, i18n.English.T(i18n.AboutHeadingKept)) {
		t.Error("a screen built with nothing kept still offers to forget it")
	}
}

// The line under Forget changes its words rather than going away, so the
// screen does not move under the pointer and says what the press did.
func TestTheLineUnderForgetSaysItForgot(t *testing.T) {
	screen := Content(i18n.English, license.Current(), version.Current(), false, nil, &keeping{})
	pressed := 0
	for _, button := range buttons(screen) {
		if button.Text == i18n.English.T(i18n.AboutForget) {
			button.OnTapped()
			pressed++
		}
	}
	if pressed != 1 {
		t.Fatalf("the screen has %d Forget buttons, want one", pressed)
	}
	all := texts(screen)
	if !has(all, i18n.English.T(i18n.AboutForgotten)) || has(all, i18n.English.T(i18n.AboutForgetWhat)) {
		t.Errorf("after the press the screen does not say it forgot, or still says what a press would do: %q",
			all)
	}
}
