package main

// Showing a folder in File Explorer (DEC-336).
//
// # Why not the way the toolkit opens a page
//
// The toolkit hands an address to rundll32 and url.dll (app/app_windows.go in
// Fyne 2.8.2). Measured on Windows Server 2025 on 2026-11-24 with
// tools/vm/folder_otwarcie.ps1: a folder named in Polish or Japanese letters
// did not open from a file address, escaped into plain ASCII, nor from a bare
// path, no window appeared and nothing reported a failure, while ShellExecuteW
// opened the right folder for every one of eight paths, those two, a share and
// a name holding a comma, a hash and a percent sign among them. A path in
// Polish letters is an ordinary path for the people this program is for.
//
// The call goes through cgo, like the calls in console_windows.go, and for the
// reason given there the network guard refuses C in our own files except in
// the files guard.CgoAllowed names, each with the names it may take. This file
// is the third, allowed by the owner on 2026-11-24 for this call and the two
// that set up COM around it. The pointers it hands Windows are the one name it
// takes from unsafe, in guard.UnsafeAllowed.

// The library holding the two calls of COM is not among those the linker of
// MinGW joins by itself, where the one holding ShellExecuteW is (measured on
// 2026-11-24: without the line below the build stops on CoInitializeEx and
// CoUninitialize alone).

/*
#cgo LDFLAGS: -lole32
#include <windows.h>
#include <objbase.h>
#include <shellapi.h>
*/
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"

	"github.com/donislawdev/qadatagenerator/internal/gui/outward"
)

// systemShowsFolder is File Explorer, which needs nothing of the toolkit.
func systemShowsFolder(outward.Desktop) func(string) error { return showInExplorer }

// showInExplorer asks the shell to open a folder, the way the documentation
// of ShellExecuteW gives for it: the verb open, a whole path, and COM set up
// first on the thread that asks, single threaded and without OLE1, because the
// shell may hand the folder to an extension that needs it (learn.microsoft.com,
// the remarks under ShellExecuteW, read 2026-11-24).
//
// The thread is held for the length of the call, because COM is set up for a
// thread and Go would otherwise be free to move this goroutine between the
// call that sets it up and the call that needs it. COM is let go only when
// this call set it up. An answer saying the thread already runs another model
// is no reason to stop, and the documentation says the call often needs no COM
// at all.
func showInExplorer(path string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if C.CoInitializeEx(nil, C.COINIT_APARTMENTTHREADED|C.COINIT_DISABLE_OLE1DDE) >= 0 {
		defer C.CoUninitialize()
	}
	verb, folder := wide("open"), wide(path)
	answer := C.ShellExecuteW(nil, (*C.WCHAR)(unsafe.Pointer(&verb[0])),
		(*C.WCHAR)(unsafe.Pointer(&folder[0])), nil, nil, C.SW_SHOWNORMAL)
	// The answer is a number dressed as a handle, and above 32 means success.
	if code := uintptr(unsafe.Pointer(answer)); code <= 32 {
		return fmt.Errorf("the shell did not open the folder, ShellExecuteW answered %d", code)
	}
	return nil
}
