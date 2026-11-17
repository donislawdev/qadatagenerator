package main

// # Why this window is a GUI program on Windows (B-392)
//
// Windows reads one number in the header of an executable and gives every
// program whose number says console a console window of its own. Go writes
// console unless told otherwise, so a double click on qadg-gui.exe opened a
// black console beside the window.
//
// The first line of C below makes that number say GUI for every build of this
// binary on Windows, from the command in the README to go install, with no
// flag anybody has to remember. The other way is the linker flag
// -H windowsgui, which would have to be repeated in every command that builds
// the window, and the macOS linker refuses it, so the README would need a
// command for Windows alone.
//
// It works because of where the line lands, measured on 2026-11-16 with Go
// 1.27.4 and MinGW-W64 GCC 16.2. Go hands the C linker -mconsole near the front
// of its command and the flags of this file near the end, and the linker keeps
// the last subsystem it is given. Written as -mwindows in the same place it
// still gives a console program, because the C driver turns -mconsole and
// -mwindows into one setting of its own and console wins. A change in that
// order would bring the console back without a word, which is why
// TestTheWindowStartsWithoutAConsole reads the header of a binary it builds
// rather than this line.
//
// The calls into Windows below go through cgo rather than through Go because
// the network guard refuses the syscall package in our own files
// (internal/guard/rules.go). They are attaching to and letting go of a console,
// asking whether the desktop can be seen, and showing a dialog. The one other
// call into Windows this binary makes itself is loading the software renderer,
// and software_windows.go says why that one goes through syscall.

/*
#cgo LDFLAGS: -Wl,--subsystem,windows
#include <windows.h>
*/
import "C"

import (
	"log"
	"os"
	"runtime"
	"unicode/utf16"
	"unsafe"
)

// speakThroughParentConsole points the standard streams this process is
// missing at the console of the program that started it, and returns what
// lets that console go again together with whether the error channel now leads
// anywhere. It does not after a double click, and failure.go then puts what
// the window has to say in a dialog instead.
//
// # What a GUI program loses, and what this gives back
//
// A GUI program starts with no console. Typed into a terminal, qadg-gui -h
// and a flag this window does not know then answered with nothing at all, and
// with no window either, measured on 2026-11-16. That is the silence DEC-253
// refuses for a wrong flag. So when a stream is missing and whatever started
// this window has a console, the window attaches to it and says what it has to
// say there. Started by a double click there is no such console, the attach
// fails and nothing changes.
//
// Only a missing stream is borrowed. A stream the program was handed is left
// where it leads, so a pipe (qadg-gui -h | more) and a redirection in cmd
// (qadg-gui -h > help.txt) still get the answer, and a script that waits for
// the window, as Start-Process -Wait does, still reads exit code 2 for a wrong
// flag. Measured on 2026-11-16, and so is what does not work: PowerShell 7.7
// leaves the file named after > empty for a GUI program, this build and one
// without any of this file alike.
//
// A terminal does not wait for a GUI program either, so what the window says
// arrives after the terminal has moved on, below its next prompt, and the
// exit code the terminal keeps is not this window's.
//
// # Why main lets go once the toolkit runs
//
// Windows passes Ctrl+C to every process attached to a console (Microsoft's
// page on CTRL+C and CTRL+BREAK signals, read 2026-11-16). A window still
// attached would close when somebody pressed Ctrl+C to stop another program
// in the same terminal, with whatever was typed into it. main lets go when the
// toolkit starts its loop, which in Fyne 2.8.2 comes after the native window
// is made, so a refusal to make one is still said in that terminal.
//
// Attaching and letting go also reset the console control handlers of this
// process, the one the Go runtime installs included (the same documentation,
// on SetConsoleCtrlHandler). While attached, a Ctrl+C in that terminal
// therefore ends this process at once rather than through the watch for
// signals. That is the short moment before the window opens, when there is
// nothing to save yet.
func speakThroughParentConsole() (release func(), heard bool) {
	outMissing, errMissing := missing(os.Stdout), missing(os.Stderr)
	if !outMissing && !errMissing {
		return func() {}, true
	}
	if C.AttachConsole(C.ATTACH_PARENT_PROCESS) == 0 {
		return func() {}, !errMissing
	}
	console, err := os.OpenFile("CONOUT$", os.O_RDWR, 0)
	if err != nil {
		C.FreeConsole()
		return func() {}, !errMissing
	}
	stdout, stderr, logged := os.Stdout, os.Stderr, log.Writer()
	if outMissing {
		os.Stdout = console
	}
	if errMissing {
		os.Stderr = console
		// The toolkit logs through the standard logger, which kept the stream
		// it was given before main started.
		log.SetOutput(console)
	}
	return func() {
		os.Stdout, os.Stderr = stdout, stderr
		log.SetOutput(logged)
		_ = console.Close()
		C.FreeConsole()
	}, true
}

// showInDialog puts a message in the dialog Windows draws for any program, with
// the error icon and one OK button, and returns once somebody presses it.
//
// # Why on a thread of its own
//
// A dialog like this keeps every window of the thread that shows it answering
// while it waits. After a panic the toolkit's window still exists on the main
// thread, and answering would call back into a toolkit that has just failed.
// Shown from another thread, that window simply stops answering behind the
// dialog, which is the truth about it.
//
// # Why not where nobody can see it
//
// A dialog on a desktop nobody sees waits for ever. Measured on 2026-11-16 on
// Windows Server 2025: the window started as a scheduled task of SYSTEM, in
// session 0 with no standard streams, was still waiting 40 seconds later,
// which is the process with nothing on the screen this dialog exists to
// replace. So it is shown only on a window station a person can see, and the
// message stays on the error channel and in the exit code either way.
func showInDialog(title, message string) {
	if !desktopCanBeSeen() {
		return
	}
	caption, text := wide(title), wide(message)
	done := make(chan struct{})
	go func() {
		// Never unlocked, so the thread ends with this goroutine and takes
		// nothing the dialog left on it to another one.
		runtime.LockOSThread()
		C.MessageBoxW(nil, (*C.WCHAR)(unsafe.Pointer(&text[0])),
			(*C.WCHAR)(unsafe.Pointer(&caption[0])), C.MB_OK|C.MB_ICONERROR|C.MB_SETFOREGROUND)
		close(done)
	}()
	<-done
}

// desktopCanBeSeen reports whether the window station of this process is one
// a person can see, which is what WSF_VISIBLE means on it. The station of a
// signed in person has it, the one services and tasks of SYSTEM run in does
// not. When Windows does not answer, the answer is yes, because a dialog
// shown in vain costs less than a failure said to nobody.
func desktopCanBeSeen() bool {
	station := C.GetProcessWindowStation()
	if station == nil {
		return true
	}
	var flags C.USEROBJECTFLAGS
	if C.GetUserObjectInformationW(C.HANDLE(unsafe.Pointer(station)), C.UOI_FLAGS,
		C.PVOID(unsafe.Pointer(&flags)), C.DWORD(unsafe.Sizeof(flags)), nil) == 0 {
		return true
	}
	return flags.dwFlags&C.WSF_VISIBLE != 0
}

// wide is text the way the wide functions of Windows read it: UTF-16 ending
// in a zero. A zero inside the text ends it early there, which loses the rest
// of a message and nothing else.
func wide(text string) []uint16 {
	return append(utf16.Encode([]rune(text)), 0)
}

// missing reports whether a standard stream leads nowhere: no handle at all, or
// a handle the system does not recognise.
func missing(stream *os.File) bool {
	if stream == nil {
		return true
	}
	_, err := stream.Stat()
	return err != nil
}
