//go:build !windows

package main

// speakThroughParentConsole has nothing to do outside Windows. There a program
// has the standard streams of whatever started it, a terminal or nothing at
// all, and no console window appears beside it either way. console_windows.go
// says what Windows does instead.
//
// It answers that the error channel is heard because nothing here can tell a
// launcher that keeps it in a log from one that throws it away, and a dialog
// is not shown on these systems anyway.
func speakThroughParentConsole() (release func(), heard bool) { return func() {}, true }

// showInDialog does nothing outside Windows. macOS and Linux have no dialog a
// program can show without a library of its own for it or a second program,
// and the guard refuses starting a second program. What would go in it still
// goes to the error channel, which is where a desktop that started the window
// keeps it, when it keeps it at all.
func showInDialog(string, string) {}
