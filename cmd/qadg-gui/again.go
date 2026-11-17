package main

import (
	"errors"
	"os"
	"os/exec"
)

// This file is the one place in the product that starts a process, and it
// starts exactly one: this window again, for the software renderer (B-414,
// DEC-317). The network guard refuses os/exec everywhere else, and
// guard.SpawnAllowed names this file together with the one call it may make
// and where that call's program comes from, so a second process of any other
// kind is a finding rather than a line somebody has to notice. The command
// line binary starts nothing at all (DEC-169).

// startAgain starts this executable again with args, waits for it, and answers
// its exit code.
//
// The program is the path the system answers for the running process, never a
// name looked up on the search path, and there is no shell between, so nothing
// a person typed chooses what runs. The arguments are the ones this process
// was given plus the flag (software.go).
//
// The second process gets this one's streams, so whatever started this window
// reads what the second one says. After a double click those streams are not
// there, and the toolchain hands a child only the handles that exist (read in
// Go 1.27.4, syscall/exec_windows.go), so the second start works from a double
// click too, which is the case it is for.
//
// The driver variable goes into the second process's environment as well,
// after everything inherited, where the toolchain lets the last value of a
// name win. The second process sets it for itself too (software_windows.go).
func startAgain(args []string) (int, error) {
	self, err := os.Executable()
	if err != nil {
		return 0, err
	}
	second := exec.Command(self, args...)
	second.Stdin, second.Stdout, second.Stderr = os.Stdin, os.Stdout, os.Stderr
	second.Env = append(os.Environ(), softwareDriver+"="+softwareLLVMPipe)
	err = second.Run()
	var ended *exec.ExitError
	if errors.As(err, &ended) {
		// It ran and answered. Its code is the answer, and a code other than
		// nought is a refusal it has already said, not an error of ours.
		return ended.ExitCode(), nil
	}
	if err != nil {
		return 0, err
	}
	return 0, nil
}
