package main

import (
	"os"
	"syscall"
)

// loadRendererBeside loads the software renderer from the opengl folder beside
// this executable (software.go).
func loadRendererBeside() error {
	dir, err := executableDir()
	if err != nil {
		return err
	}
	return loadSoftwareRenderer(dir)
}

// loadSoftwareRenderer loads the renderer's files from under exeDir, so that
// the toolkit's later request for opengl32.dll by name is answered by them.
//
// # Why before the toolkit, and why this is enough
//
// Windows hands a library already loaded under a name to every later request
// for that name (Microsoft's page on the search order for libraries, read on
// 2026-11-17). The window no longer imports opengl32.dll when it starts
// (third_party/go-gl-gl/PATCH.md), so before the toolkit exists nothing of
// that name is loaded and the first request is this one.
//
// # Why syscall here
//
// syscall.LoadDLL calls LoadLibrary with the absolute path worked out in
// softwareFiles, which loads that file and nothing else, and reads the
// system's reason in the same call when it fails. The network guard refuses
// syscall in our files, and guard.SyscallAllowed lets this file take LoadDLL
// from it and nothing more. Through cgo, like the rest of the calls into
// Windows in console_windows.go, the reason would have to be read in a second
// call, by which time the thread may have done something else.
//
// The driver variable is set in this process before the first file, because
// Mesa reads it when it starts. The first process sets it for the second one
// too, in the environment it hands over (again.go). Both are needed, because
// somebody can ask for the flag by hand.
func loadSoftwareRenderer(exeDir string) error {
	if err := os.Setenv(softwareDriver, softwareLLVMPipe); err != nil {
		return err
	}
	for _, path := range softwareFiles(exeDir) {
		// The error names the file and carries the system's words, so it goes
		// on as it is rather than wrapped in a second copy of the path.
		if _, err := syscall.LoadDLL(path); err != nil {
			return err
		}
	}
	return nil
}
