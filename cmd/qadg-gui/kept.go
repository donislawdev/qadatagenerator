package main

import (
	"path/filepath"

	"fyne.io/fyne/v2"

	"github.com/donislawdev/qadatagenerator/internal/gui/kept"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// appID names this window to the toolkit, which keeps its preferences in a
// folder of that name under the user's settings: on Windows
// %APPDATA%\fyne\dev.donislaw.qadg, on macOS under ~/Library/Preferences/fyne,
// on Linux under the user's configuration folder (fyne v2.8.2,
// internal/app/config_*.go). What is kept there is internal/gui/kept.
//
// 🔴 Renaming it after a release leaves every saved size and folder behind
// under the old name, so it does not change. The form is the one the owner's
// other two windows use, dev.donislaw and the name of the command, chosen by
// the owner on 2026-11-17 (DEC-325). It is not the identifier of the macOS
// application bundle, which the release names on its own as
// com.donislawdev.qadg-gui, the same split the other two have.
const appID = "dev.donislaw.qadg"

// keptBy is the store over this application's preferences, with the folder
// they are in spelled the way the system spells it. The toolkit hands the
// folder over as the path of a URI, with forward slashes on every system.
func keptBy(application fyne.App) *kept.Store {
	return kept.New(application.Preferences(), filepath.FromSlash(application.Storage().RootURI().Path()))
}

// openingSize is the size the window opens at: the size it was closed at last
// time, or the size a first start opens at when nothing is kept.
//
// A size kept on a large screen and opened on a small one is used as it is.
// This toolkit has no portable way to ask how big the screen is, and the
// window opens where the system puts a new window, which keeps its title bar
// on the screen, so the person can still move it and make it smaller.
func openingSize(store *kept.Store) fyne.Size {
	if size, ok := store.Size(); ok {
		return size
	}
	return fyne.NewSize(palette.StartWidth, palette.StartHeight)
}
