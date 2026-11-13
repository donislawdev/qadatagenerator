// Package appicon carries the picture the desktop shows for this program: a
// wooden spoon of red lentils.
//
// # Two icons on Windows, from two places
//
// The window gets its icon from the toolkit, through SetIcon in cmd/qadg-gui.
// File Explorer, a shortcut and the taskbar button of a program that is not
// running read a resource compiled into the .exe instead, and a binary with no
// such resource is drawn with the blank placeholder Windows keeps for anything
// it cannot identify. So one drawing reaches the desktop by two roads.
// lentils.svg is embedded here for the toolkit, and lentils.ico beside it is
// compiled into both executables by the resource scripts in cmd/qadg and
// cmd/qadg-gui. The command line binary gets the picture only that way. It has
// no window, so it never imports this package.
//
// # Where the icon does not reach, and why
//
// On macOS the toolkit puts no icon on a window, because the windowing library
// under it does not support one there, and the Dock shows the icon of an
// application bundle, which this project does not build yet. Under Wayland the
// toolkit skips the window icon as well, and the desktop takes it from a
// .desktop file, which is also a matter of packaging rather than of code.
//
// # Why the window is handed the drawing rather than a picture of it
//
// The toolkit draws an SVG icon itself, at 256 px, which is the size the
// drawing is made at. That size matters. Below its own size the toolkit's SVG
// rasteriser does not scale the width of a line, so a drawing asked for at
// 16 px comes out as a smudge. Measured on 2026-11-13 with fyne v2.8.2: a line
// 32 units wide on a 256 unit drawing was 32 px wide in a 256 px picture and
// still 32 px wide in a 128 px one. The window icon is safe from that, and the
// small pictures in lentils.ico are made by drawing at 256 px and reducing,
// which is what the test beside this file does.
//
// # How the files are made again after the drawing changes
//
// lentils.svg and lentils-small.svg are the masters. The small one is drawn for
// 32 px and below, with heavier lines and less detail, and only lentils.ico
// uses it. From the root of the repository, in this order:
//
//	go test ./internal/gui/appicon -run TestTheIconFileIsDrawnFromTheDrawings -update
//	windres -i cmd/qadg-gui/qadg-gui.rc -O coff -o cmd/qadg-gui/rsrc_windows_amd64.syso
//	windres -i cmd/qadg/qadg.rc -O coff -o cmd/qadg/rsrc_windows_amd64.syso
//
// windres comes with the C toolchain the window already needs on Windows. The
// .syso files are committed rather than built, because the Go toolchain links
// any .syso it finds in a main package and runs no generator of its own. Their
// names carry GOOS and GOARCH, so every other platform ignores them. The tests
// beside this file fail when a .syso is older than the icon file, or the icon
// file older than the drawings.
package appicon

import _ "embed"

// SVG is the drawing the window wears, 256 by 256 units.
//
//go:embed lentils.svg
var SVG []byte

// Name is what the toolkit is told the icon is called. The extension is what
// makes it read the bytes as a drawing rather than try to decode a bitmap.
const Name = "lentils.svg"
