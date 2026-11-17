//go:build !windows

package main

import "runtime"

// loadRendererBeside answers that no software renderer comes with the window
// on this system.
//
// Nothing is carried beside the window on Linux or macOS (software.go), so
// -software-gl says so there rather than doing nothing without a word, which
// is a flag somebody waits on.
func loadRendererBeside() error {
	return notShipped{runtime.GOOS}
}
