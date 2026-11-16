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
// AttachConsole and FreeConsole are called through cgo rather than through Go
// because the network guard refuses the syscall package in our own files
// (internal/guard/rules.go), and these are the only two calls into Windows this
// binary makes itself.

/*
#cgo LDFLAGS: -Wl,--subsystem,windows
#include <windows.h>
*/
import "C"

import (
	"log"
	"os"
)

// speakThroughParentConsole points the standard streams this process is
// missing at the console of the program that started it, and returns what
// lets that console go again.
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
func speakThroughParentConsole() (release func()) {
	outMissing, errMissing := missing(os.Stdout), missing(os.Stderr)
	if !outMissing && !errMissing {
		return func() {}
	}
	if C.AttachConsole(C.ATTACH_PARENT_PROCESS) == 0 {
		return func() {}
	}
	console, err := os.OpenFile("CONOUT$", os.O_RDWR, 0)
	if err != nil {
		C.FreeConsole()
		return func() {}
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
	}
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
