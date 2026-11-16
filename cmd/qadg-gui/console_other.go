//go:build !windows

package main

// speakThroughParentConsole has nothing to do outside Windows. There a program
// has the standard streams of whatever started it, a terminal or nothing at
// all, and no console window appears beside it either way. console_windows.go
// says what Windows does instead.
func speakThroughParentConsole() (release func()) { return func() {} }
