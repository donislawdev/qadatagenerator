//go:build !windows

package cli

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
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
	for _, sample := range []struct {
		name string
		sent syscall.Signal
		kind fault.Kind
	}{
		{"a person pressing the key", syscall.SIGINT, fault.Interrupted},
		{"the system taking the process down", syscall.SIGTERM, fault.Terminated},
	} {
		t.Run(sample.name, func(t *testing.T) {
			watcher := watchSignals()
			defer watcher.release()

			if err := syscall.Kill(syscall.Getpid(), sample.sent); err != nil {
				t.Fatalf("cannot send the signal to this process: %v", err)
			}
			select {
			case <-watcher.stopping:
			case <-time.After(5 * time.Second):
				t.Fatal("a signal was sent to this process and the watch never saw it")
			}

			if kind := fault.KindOf(watcher.Stopped()); kind != sample.kind {
				t.Errorf("the run ended as %v rather than as %v", kind, sample.kind)
			}
		})
	}
}

// How a process ends can be watched only from outside it, and a test that
// ended its own process would end every test after it. The tests below start
// this test binary again, where TestEndHelper does what a stopped run does.

// endHelperAsked names what the helper process does: INT or TERM to stop itself
// with that signal, CODE to end as a run that failed on its own.
const endHelperAsked = "QADG_END_HELPER"

// TestEndHelper is not a test. It acts only in the process the tests below
// start, and there it ends the process.
func TestEndHelper(t *testing.T) {
	asked := os.Getenv(endHelperAsked)
	if asked == "" {
		return
	}
	env := FromProcess(nil, os.Stdout, os.Stderr)
	if asked == "CODE" {
		env.Exit(int(codeExpectation))
	}
	sent := map[string]syscall.Signal{"INT": syscall.SIGINT, "TERM": syscall.SIGTERM}[asked]
	watcher := watchSignals()
	if err := syscall.Kill(syscall.Getpid(), sent); err != nil {
		t.Fatalf("cannot send the signal to this process: %v", err)
	}
	select {
	case <-watcher.stopping:
	case <-time.After(5 * time.Second):
		t.Fatal("a signal was sent to this process and the watch never saw it")
	}
	code, _ := codeFor(fault.KindOf(watcher.Stopped()))
	watcher.release()
	env.Exit(int(code))
}

// 🔴 A bash loop goes on to its next round after a program that exited with
// 130, and stops after one that died of the interrupt. This is the test that
// a stopped run is the second kind (B-346).
func TestAStoppedRunEndsByTheSignalThatStoppedIt(t *testing.T) {
	for _, sample := range []struct {
		name  string
		asked string
		sent  syscall.Signal
	}{
		{"a person pressing the key", "INT", syscall.SIGINT},
		{"the system taking the process down", "TERM", syscall.SIGTERM},
	} {
		t.Run(sample.name, func(t *testing.T) {
			status, output := endHelper(t, sample.asked, false)
			if !status.Signaled() || status.Signal() != sample.sent {
				t.Errorf("the process ended with exit status %d rather than by %v, "+
					"so a loop in bash around it goes on to its next round\n%s",
					status.ExitStatus(), sample.sent, output)
			}
		})
	}
}

func TestARunThatFailedOnItsOwnEndsWithItsCode(t *testing.T) {
	status, output := endHelper(t, "CODE", false)
	if status.Signaled() || status.ExitStatus() != int(codeExpectation) {
		t.Errorf("a run that failed with code %d ended with exit status %d, signalled %v\n%s",
			codeExpectation, status.ExitStatus(), status.Signaled(), output)
	}
}

// A shell starts a command in the background with the interrupt ignored, and
// the system ignores it again once the watch stops. Sending it would wait out
// the whole grace for a signal that never arrives.
func TestAnInterruptIgnoredFromTheStartEndsWithItsCodeAtOnce(t *testing.T) {
	started := time.Now()
	status, output := endHelper(t, "INT", true)
	took := time.Since(started)
	if status.Signaled() || status.ExitStatus() != int(codeInterrupted) {
		t.Errorf("the process ended with exit status %d, signalled %v, rather than with %d\n%s",
			status.ExitStatus(), status.Signaled(), codeInterrupted, output)
	}
	if took >= resendGrace/2 {
		t.Errorf("the process took %v to end, so it waited for a signal the system ignores", took)
	}
}

// endHelper starts this test binary again as TestEndHelper and returns how
// that process ended, with what it wrote.
func endHelper(t *testing.T, asked string, interruptIgnored bool) (syscall.WaitStatus, string) {
	t.Helper()
	self := os.Args[0]
	command := exec.Command(self, "-test.run=^TestEndHelper$")
	if interruptIgnored {
		// What a shell does for a command it runs in the background. exec
		// keeps an ignored signal ignored.
		command = exec.Command("/bin/sh", "-c", `trap "" INT && exec "$0" -test.run='^TestEndHelper$'`, self)
	}
	command.Env = append(os.Environ(), endHelperAsked+"="+asked)
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	var exited *exec.ExitError
	if err := command.Run(); err != nil && !errors.As(err, &exited) {
		t.Fatalf("cannot start the helper process: %v", err)
	}
	return command.ProcessState.Sys().(syscall.WaitStatus), output.String()
}
