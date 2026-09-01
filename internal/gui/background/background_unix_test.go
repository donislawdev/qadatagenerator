//go:build !windows

package background

import (
	"syscall"
	"testing"
	"time"
)

// The one thing the tests beside this cannot show: that the signals this window
// asks for are the signals the runtime actually hands it. Everything else calls
// the watch by hand, which proves the arrangement works once something arrives
// and says nothing about whether anything ever will.
//
// 🔴 Read the failure of this one carefully. If the request for the signal is
// gone, the process is taken down by the interrupt it no longer catches, so
// this goes red as a killed test binary rather than as a message. That is a
// real red and an honest one, and it is written here so that nobody spends an
// hour looking for the panic.
//
// There is no counterpart on Windows. A process there has no portable way to
// raise a signal at itself, and the version of this that used the console
// control handler would be testing the operating system rather than this file.
// The command line half has the same pair of tests and the same gap, and
// tools/platformy.py is what runs both of them on a real Linux.
func TestARealSignalReachesTheWindow(t *testing.T) {
	for _, sample := range []struct {
		name string
		sent syscall.Signal
	}{
		{"a person pressing the key", syscall.SIGINT},
		{"the system taking the process down", syscall.SIGTERM},
	} {
		t.Run(sample.name, func(t *testing.T) {
			// The ask is answered on the thread that draws, and there is no
			// window here, so it is caught rather than acted on. What this
			// guard is about is whether it arrives at all.
			arrived := make(chan struct{})
			release := watchForClosing(func() { close(arrived) }, func(act func()) { act() })
			defer release()

			if err := syscall.Kill(syscall.Getpid(), sample.sent); err != nil {
				t.Fatalf("cannot send the signal to this process: %v", err)
			}
			select {
			case <-arrived:
			case <-time.After(5 * time.Second):
				t.Fatal("a signal was sent to this process and the window never heard it")
			}
		})
	}
}
