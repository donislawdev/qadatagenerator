// Package outward is the one place in the window that hands an address to the
// system, and the two addresses it may hand over.
//
// # Why this does not break untouchable rule 8
//
// The rule is about what the product does: no connection of its own, no
// telemetry, no check for updates. Pressing one of these two buttons makes no
// connection either. The address goes to the system, which opens it in whatever
// program is registered for it, and only because a person pressed the button
// (DEC-276). This program sends nothing and fetches nothing.
//
// # Why it is a package with nothing else in it
//
// The guard over the toolkit's calls that leave this machine refuses OpenURL
// everywhere in the window except in this file, and refuses this file naming
// any address beyond the two it lists. An exception living in a file that also
// does other work is an exception covering more than it says, so the file does
// this and nothing besides.
package outward

import (
	"net/url"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	"github.com/donislawdev/qadatagenerator/internal/gui/shell"
)

// The two pages, and the only addresses this window knows.
//
// Here rather than in the catalogue of words, because a translation must not
// be able to change where a button goes. The guard holds the same two in its
// own register and fails when this file names another or loses one, so adding
// a third page is an edit to the guard as well, which is the point.
const (
	projectPage = "https://github.com/donislawdev/qadatagenerator"
	supportPage = "https://donislawdev.com/support/"
)

// Desktop is what hands an address to the system. The toolkit's application is
// one, and a test or a picture of the window hands in one that opens nothing.
type Desktop interface {
	OpenURL(address *url.URL) error
}

// Refused is what the window shows when the system did not open a page: a
// title and a message, both ready to read.
type Refused func(title, message string)

// Links are the two pages, in the order the foot of the column shows them.
//
// No caller hands in an address, and none can. The pages are chosen here, so
// the only way to open a third one is to write it into this file, where the
// guard reads it.
//
// # Why the address is handed over off the thread that draws
//
// The toolkit waits for the helper that passes the address on: on Windows and
// on macOS it calls Run rather than Start (app/app_windows.go:59 and
// app/app_desktop_darwin.go:24 in Fyne 2.8.2). Windows asks which program
// should open a web page when none is chosen yet, and the helper waits for that
// answer, so on the thread that draws the window would stop answering for as
// long as the question stays open.
func Links(text i18n.Catalogue, desktop Desktop, read background.Read, refused Refused) []shell.Link {
	return []shell.Link{
		{Name: text.T(i18n.SidebarLinkProject), Icon: paint.IconCode,
			Open: opening(text, desktop, read, refused, projectPage)},
		{Name: text.T(i18n.SidebarLinkSupport), Icon: paint.IconHeart,
			Open: opening(text, desktop, read, refused, supportPage)},
	}
}

// opening is what pressing the button for one page does.
//
// A failure is shown rather than swallowed. A button that does nothing looks
// exactly like a window that stopped responding, and the address in the
// message is the way to the page when the system will not open it.
func opening(text i18n.Catalogue, desktop Desktop, read background.Read, refused Refused,
	page string) func() {
	tell := func(reason error) func() {
		message := text.T(i18n.SidebarLinkRefusedBody) + paragraph + page +
			paragraph + text.T(i18n.SidebarLinkRefusedReason) + line + reason.Error()
		return func() { refused(text.T(i18n.SidebarLinkRefusedTitle), message) }
	}
	return func() {
		read(func() func() {
			address, err := url.Parse(page)
			if err == nil {
				err = desktop.OpenURL(address)
			}
			if err != nil {
				return tell(err)
			}
			return nil
		}, tell)
	}
}

// What separates the parts of the message. A line break inside a sentence of
// the catalogue would be a translator deciding the layout of the dialog.
const (
	line      = "\n"
	paragraph = "\n\n"
)
