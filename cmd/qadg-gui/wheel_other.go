//go:build !windows

package main

// systemWheelLines has nothing to ask outside Windows, and says so. A Linux
// desktop has no setting for it a program can ask, and on macOS the toolkit's
// own wheel is kept (internal/gui/parts/wheel_darwin.go), so the window's
// default stands.
func systemWheelLines() (int, bool) { return 0, false }
