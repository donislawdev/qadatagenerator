//go:build !windows

package cli

import (
	"syscall"
	"testing"
	"time"

	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// The one thing the tests beside this cannot show: that the signal this tool
// asks for is the signal the runtime actually hands it. Everything else
// delivers into the channel by hand, which proves the arrangement works once
// something arrives and says nothing about whether anything ever will.
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
// A test may name syscall where the product may not, because the guard behind
// the promise of no outbound traffic reads the imports of the product only.
func TestARealSignalReachesTheWatch(t *testing.T) {
	watcher := watchSignals()
	defer watcher.release()

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGINT); err != nil {
		t.Fatalf("cannot send the signal to this process: %v", err)
	}
	select {
	case <-watcher.stopping:
	case <-time.After(5 * time.Second):
		t.Fatal("a signal was sent to this process and the watch never saw it")
	}

	if kind := fault.KindOf(watcher.Stopped()); kind != fault.Interrupted {
		t.Errorf("an interrupt arrived and the run ended as %v", kind)
	}
}
