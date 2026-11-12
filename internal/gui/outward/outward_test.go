package outward

import (
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
)

// recorder is a desktop that remembers what it was handed and answers with
// what the test set.
type recorder struct {
	handed []string
	answer error
}

func (r *recorder) OpenURL(address *url.URL) error {
	r.handed = append(r.handed, address.String())
	return r.answer
}

// Each button opens its own page and no other, and says which page it is.
//
// The addresses are compared with the constants of this package rather than
// written out again, and that is not circular: what this asks is that the
// project button reaches the project page and the support button the support
// page. That the constants are the two pages DEC-276 names is asked by the guard
// over the toolkit, which holds them in its own register.
func TestEveryLinkOpensItsOwnPage(t *testing.T) {
	desktop := &recorder{}
	refused := 0
	links := Links(i18n.English, desktop, background.ReadHere, func(string, string) { refused++ })

	want := []struct {
		name, icon, page string
	}{
		{i18n.English.T(i18n.SidebarLinkProject), paint.IconCode, projectPage},
		{i18n.English.T(i18n.SidebarLinkSupport), paint.IconHeart, supportPage},
	}
	if len(links) != len(want) {
		t.Fatalf("%d links, want %d", len(links), len(want))
	}
	for at, link := range links {
		if link.Name != want[at].name || link.Icon != want[at].icon {
			t.Errorf("link %d is %q with %q, want %q with %q", at, link.Name, link.Icon, want[at].name, want[at].icon)
		}
		desktop.handed = nil
		link.Open()
		if len(desktop.handed) != 1 || desktop.handed[0] != want[at].page {
			t.Errorf("pressing %q handed the system %v, want only %s", link.Name, desktop.handed, want[at].page)
		}
	}
	if refused != 0 {
		t.Errorf("pages the system opened were reported as refused %d times", refused)
	}
}

// A page the system did not open is said, with its address and the system's
// reason, rather than leaving a button that did nothing.
//
// The reason is passed on as the system gave it, because it is the one part of
// the message a person can act on: no browser installed, no program for links.
func TestAPageTheSystemDidNotOpenIsSaidWithItsAddress(t *testing.T) {
	desktop := &recorder{answer: errors.New(`exec: "xdg-open": executable file not found in $PATH`)}
	var title, message string
	links := Links(i18n.English, desktop, background.ReadHere, func(said, body string) { title, message = said, body })

	links[1].Open()
	if title != i18n.English.T(i18n.SidebarLinkRefusedTitle) {
		t.Errorf("the title is %q", title)
	}
	for _, part := range []string{i18n.English.T(i18n.SidebarLinkRefusedBody), supportPage,
		i18n.English.T(i18n.SidebarLinkRefusedReason), "xdg-open"} {
		if !strings.Contains(message, part) {
			t.Errorf("the message does not say %q:\n%s", part, message)
		}
	}
}

// Opening that comes apart is said the same way, through the way out the port
// for reading offers, rather than ending the window.
func TestOpeningThatComesApartIsSaidRatherThanEndingTheWindow(t *testing.T) {
	desktop := &recorder{}
	var message string
	crashing := func(_ func() func(), crashed func(error) func()) {
		crashed(errors.New("came apart"))()
	}
	Links(i18n.English, desktop, crashing, func(_, body string) { message = body })[0].Open()
	if !strings.Contains(message, projectPage) || !strings.Contains(message, "came apart") {
		t.Errorf("a crash while opening was not said with the address and the cause:\n%s", message)
	}
}

// The address is handed over through the port for reading and never on the
// thread that pressed the button.
//
// The port here keeps what it was given and runs nothing, so a desktop that is
// reached anyway was reached directly, which on Windows and macOS holds the
// window for as long as the helper passing the address on takes.
func TestTheAddressIsHandedOverThroughThePortAndNotDirectly(t *testing.T) {
	desktop := &recorder{}
	var kept []func() func()
	holding := func(read func() func(), _ func(error) func()) { kept = append(kept, read) }
	for _, link := range Links(i18n.English, desktop, holding, func(string, string) {}) {
		link.Open()
	}
	if len(desktop.handed) != 0 {
		t.Fatalf("the system was handed %v without the port running anything", desktop.handed)
	}
	if len(kept) != 2 {
		t.Fatalf("the port was asked %d times for two presses", len(kept))
	}
	for _, read := range kept {
		if act := read(); act != nil {
			t.Error("a page the system opened left something to do on the drawing thread")
		}
	}
	if len(desktop.handed) != 2 {
		t.Errorf("running what the port kept handed the system %v, want both pages", desktop.handed)
	}
}

// The desktop a picture is given opens nothing and reports nothing wrong, so a
// picture of the window can press these buttons without leaving it.
func TestNowhereOpensNothingAndRefusesNothing(t *testing.T) {
	refused := 0
	for _, link := range Links(i18n.English, Nowhere{}, background.ReadHere, func(string, string) { refused++ }) {
		link.Open()
	}
	if refused != 0 {
		t.Errorf("a desktop that opens nothing was reported as refusing %d times", refused)
	}
}
