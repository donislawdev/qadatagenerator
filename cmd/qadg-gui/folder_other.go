//go:build !windows

package main

import (
	"net/url"
	"path/filepath"

	"github.com/donislawdev/qadatagenerator/internal/gui/outward"
)

// systemShowsFolder hands a folder to the toolkit, which hands it to open on
// macOS and to xdg-open on Linux (app/app_desktop_darwin.go and app/app_xdg.go
// in Fyne 2.8.2).
//
// The address is built from parts, so a path holding a hash or a percent sign
// is escaped by the parser rather than read as the start of a fragment. This
// file is the one place outside internal/gui/outward allowed to call OpenURL,
// and the guard holds it to addresses of the file scheme built this way
// (guard.ToolkitNetworkAllowed, DEC-336). Measured on macOS on 2026-11-24:
// open finds a folder named in Polish and Japanese letters from such an
// address, and refuses the same name with a suffix that is not there.
func systemShowsFolder(desktop outward.Desktop) func(string) error {
	return func(path string) error {
		return desktop.OpenURL(&url.URL{Scheme: "file", Path: filepath.ToSlash(path)})
	}
}
