package main

// How far one notch of the mouse wheel moves, as Windows is set to: Settings,
// Bluetooth and devices, Mouse, "Lines to scroll at a time" (B-420).
//
// The call goes through cgo, like the calls in console_windows.go, and for the
// reason given there the network guard refuses C in our own files except in
// the files guard.CgoAllowed names, each with the names it may take. This file
// is the second, allowed by the owner on 2026-11-23 for this one call, and the
// pointer it hands Windows is the one name it takes from unsafe, in
// guard.UnsafeAllowed. A second call means a new name there.

/*
#include <windows.h>
*/
import "C"

import "unsafe"

// systemWheelLines is how many lines Windows is set to scroll for one notch of
// the wheel, -1 when it is set to a whole screen at a time, and false when the
// system did not answer, so that the window's default stands.
//
// Asked at every notch rather than once, so a change in Settings reaches a
// window that is already open, as it reaches every other program. One call into
// a library every window already has loaded, against a notch that costs a frame
// to draw.
func systemWheelLines() (int, bool) {
	var lines C.UINT
	if C.SystemParametersInfoW(C.SPI_GETWHEELSCROLLLINES, 0, C.PVOID(unsafe.Pointer(&lines)), 0) == 0 {
		return 0, false
	}
	if lines == C.WHEEL_PAGESCROLL {
		return -1, true
	}
	return int(lines), true
}
