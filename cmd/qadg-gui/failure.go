package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
)

// # What this window does when it cannot go on (B-413)
//
// A failure before the window opens, or one that takes it down, is said on
// the error channel. On Windows this window starts without a console, so after
// a double click that channel leads nowhere and the failure was silence.
//
// One failure was worse than silent. Measured on 2026-11-16 on Windows Server
// 2025 in VirtualBox without 3D acceleration, started from File Explorer:
// Fyne 2.8.2 could not make a window without OpenGL, logged why, and then ran
// its loop with no window at all. The process stayed alive with nothing on the
// screen until somebody ended it. Started from a console the reason reached
// the console and the process stayed alive just the same.
//
// So a window that did not open is caught before the loop starts, and every
// failure that stops this window goes through aloud, which adds a dialog of
// the system when nobody hears the error channel.

// failed is the exit code of a window that stopped on a failure of its own,
// the number the command line gives a failure at run time (docs/09).
const failed = 1

// stop says why this window cannot go on and ends the process with code.
//
// The title is the name of the program in the dialog's title bar, which is
// what tells a person who double clicked something which program is talking.
func stop(heard bool, message string, code int) {
	aloud(heard, os.Stderr, showInDialog, i18n.English.T(i18n.WindowTitle), message)
	os.Exit(code)
}

// aloud says on the error channel what stops this window, and also in a
// dialog of the system when nobody hears that channel.
//
// The dialog says exactly what the channel says, so a person who reports it
// and a script that logged it quote the same words. It is not shown to
// somebody who can read the channel, because a dialog waits for a click and
// a script that started this window would wait with it.
func aloud(heard bool, channel io.Writer, dialog func(title, message string), title, message string) {
	// A channel that fails to take the message leaves the dialog as the one
	// place left to say it, and the dialog does not depend on the channel.
	_, _ = fmt.Fprintln(channel, message)
	if !heard {
		dialog(title, message)
	}
}

// showOrStop shows the window and stops the process, saying why, when the
// toolkit made no native window for it. A panic that came out of Show while a
// window stands is a defect of this program and goes on to lastCatch.
func showOrStop(window fyne.Window, toolkit listening, text i18n.Catalogue, heard bool) {
	opened, causes, defect := opens(window, toolkit)
	if !opened {
		stop(heard, notOpened(text, causes), failed)
	}
	if defect != nil {
		panic(defect)
	}
}

// opens shows the window and reports whether the toolkit made a native window
// for it, the system's words for why when it did not, and a panic that came
// out of Show while a window stands.
//
// # Two ways the toolkit says no
//
// Fyne makes the native window inside Show. When the system refuses it, Fyne
// writes why in its log and returns, with no error anybody can catch
// (driver.go:179-187 and window_desktop.go:841-845 in 2.8.2). That is the
// Windows machine without OpenGL. When GLFW itself did not start, as on Linux
// with no display, the first call Fyne makes into it after that panics
// instead, because go-gl/glfw panics on NotInitialized (error.go:193-224 in
// v0.1.0-pre.2). A panic with no window is the same refusal as a line in the
// log with no window, so both end in the same message, and the panic's words
// join the reasons.
//
// So the window is shown apart from running the loop, what the toolkit has
// logged since main began listening is kept, and the answer is read from the
// state of the window rather than from either kind of words.
func opens(window fyne.Window, toolkit listening) (opened bool, causes []string, defect any) {
	failure := show(window)
	causes = toolkit.done()
	if !windowOpened(window) {
		if failure != nil {
			causes = append(causes, fmt.Sprint(failure))
		}
		return false, causes, nil
	}
	return true, causes, failure
}

// show is Show with a panic caught and handed back rather than let through.
func show(window fyne.Window) (failure any) {
	defer func() { failure = recover() }()
	window.Show()
	return nil
}

// windowOpened reports whether the toolkit made a native window for this one.
//
// It asks for the native handle through the one public way to reach it. The
// handle is zero when there is no window behind it, on each desktop system
// the toolkit runs on (RunNative in window_windows.go, window_darwin.go,
// window_x11.go, window_wayland.go and window_x11wayland.go of Fyne 2.8.2). A
// context this does not know, from a later version or another system, counts
// as a window that opened, because refusing to start on a guess would be
// worse than the silence this replaces.
func windowOpened(window fyne.Window) bool {
	native, ok := window.(driver.NativeWindow)
	if !ok {
		return true
	}
	handle := uintptr(1)
	native.RunNative(func(context any) {
		switch made := context.(type) {
		case driver.WindowsWindowContext:
			handle = made.HWND
		case driver.MacWindowContext:
			handle = made.NSWindow
		case driver.X11WindowContext:
			handle = made.WindowHandle
		case driver.WaylandWindowContext:
			handle = made.WaylandSurface
		}
	})
	return handle != 0
}

// notOpened is what the window says when the toolkit made no native window
// for it: what happened and what it needs, the system's own words for why when
// the toolkit logged any, and what to do instead.
func notOpened(text i18n.Catalogue, causes []string) string {
	parts := []string{text.T(i18n.WindowNotOpened)}
	if len(causes) > 0 {
		parts = append(parts, text.T(i18n.WindowNotOpenedReason)+"\n"+strings.Join(causes, "\n"))
	}
	parts = append(parts, text.T(i18n.WindowNotOpenedDo))
	return strings.Join(parts, "\n\n")
}

// listening is what the toolkit logs from the moment main asks for the window
// until it is shown.
//
// It starts before NewWindow rather than before Show because each of the two
// can be where the system says no. GLFW starts when the first window is asked
// for, and on Linux with no display that is where the real reason is logged,
// "PlatformUnavailable: Failed to detect any supported platform", while Show
// only adds that GLFW is not initialized (measured on 2026-11-16 in a
// container with no display). It does not start before app.New, which logs
// things that have nothing to do with a window, such as a cache folder it
// could not make.
type listening struct {
	logged io.Writer
	said   *toolkitWords
}

// listen starts keeping what the standard logger is given, on top of where it
// already goes.
//
// 🔴 What is kept comes FIRST. io.MultiWriter stops at the first writer that
// fails, and after a double click the logger's own writer is an error channel
// that leads nowhere, so every write to it fails. With the kept words second,
// the dialog on the machine without OpenGL came without the system's reason,
// measured on 2026-11-16, while the same build started from a console had it.
// The kept words never fail, so the channel behind them still gets every line.
func listen() listening {
	toolkit := listening{logged: log.Writer(), said: &toolkitWords{}}
	log.SetOutput(io.MultiWriter(toolkit.said, toolkit.logged))
	return toolkit
}

// done hands the logger back its own writer and returns the reasons kept.
func (toolkit listening) done() []string {
	log.SetOutput(toolkit.logged)
	return toolkit.said.causes()
}

// toolkitWords keeps the first lines the toolkit logs while it makes the
// window, so a window that did not open can say why in the system's words.
//
// Fyne 2.8.2 writes the reason through the standard logger, on a line holding
// "Cause:" (fyne.LogError). If a later version writes it some other way, the
// message loses that one part and keeps the rest, and
// TestTheToolkitStillNamesTheCause fails on the upgrade.
//
// It has no lock of its own because the standard logger holds one: every
// write and log.SetOutput take the same mutex in Go 1.27.4 (log.go, outMu). So
// writes arrive one at a time, and once listening.done has handed the logger
// back its own writer nothing writes here again before causes reads.
type toolkitWords struct {
	kept []byte
}

// toolkitWordsKept is how much of the log is kept. The reason is one of the
// first lines, so this only bounds a toolkit that logs a great deal.
const toolkitWordsKept = 4096

func (w *toolkitWords) Write(said []byte) (int, error) {
	if room := toolkitWordsKept - len(w.kept); room > 0 {
		w.kept = append(w.kept, said[:min(room, len(said))]...)
	}
	return len(said), nil
}

// causes is the text after "Cause:" on every line that holds one.
func (w *toolkitWords) causes() []string {
	var found []string
	for _, line := range strings.Split(string(w.kept), "\n") {
		if _, cause, ok := strings.Cut(line, "Cause:"); ok {
			if cause = strings.TrimSpace(cause); cause != "" {
				found = append(found, cause)
			}
		}
	}
	return found
}
