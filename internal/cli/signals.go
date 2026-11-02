package cli

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// This file is the only place in the product that starts concurrent work, and
// it is named in guard.ConcurrencyAllowed for exactly that reason. Everything
// concurrent about interrupting a run lives here, so that the list stays one
// file long and the rest of the tree stays single threaded.
//
// # What a signal costs this tool, and what it does not
//
// Every file is written under a temporary name and renamed once the whole run
// has succeeded, so an interrupted run never leaves a file that looks complete.
// The damage a signal can do is therefore small and exact: temporary files left
// in somebody's output directory, named after our entities, that nothing will
// ever clean up. That is what this removes and it is the whole of the problem.
//
// # Why the watch does not start until there is something to clean
//
// Reading a recipe and planning a run touch no disk. Nothing is caught during
// those, so a person who presses the key while the tool is starting gets the
// instant end their shell would have given them anyway. The watch covers the
// part of the run that writes, which is the part with something to take back.
//
// # Two signals, and the second one is why this file may write syscall
//
// The interrupt a person sends is os.Interrupt and needs nothing special. The
// termination a system sends has no name outside package syscall, and that
// package is refused everywhere else, because it can do whatever the operating
// system can and would route around every other name the network guard lists.
//
// The exception is therefore not for this package and not for this file as a
// whole. It is for one name. guard.SyscallAllowed says which file may write
// syscall and which identifiers it may take from it, a scanner reads the syntax
// tree and reports anything else, and both directions are closed: an entry
// naming a file that has stopped importing syscall is a finding too.
//
// This matters for runs nobody is watching. A container stopped with docker
// stop and a cancelled CI job both send the termination signal, and those are
// exactly the runs where a leftover temporary file goes unnoticed.
//
// # Why a stopped run ends by the signal that stopped it
//
// 🔴 A shell tells a program that died of the interrupt apart from one that
// exited with 130, and bash acts on the difference. A loop around a program
// that died of it stops. A loop around one that exited with 130 goes on to the
// next round, on the reading that the program dealt with the key itself and
// wants to carry on. Measured with the binary on bash 5.3.9 and 3.2.57: the
// rounds after an interrupted generate ran to the end, while the same loop
// around quick, which catches nothing, stopped at once (B-346).
//
// So once a run has cleaned up, the signal is sent again and the process ends
// the way it would have if nothing had caught it. A shell still reports 130
// and 143, and a program that started this one sees the signal, as it already
// does for quick and for a run stopped before its first write.

// halt watches for the signal that ends a run from outside it.
//
// It answers a question rather than acting. The goroutine below notices the
// signal and the run itself decides where to stop, so the files are removed on
// the same goroutine that made them and nothing races with a write in progress.
// A handler that deleted the files itself would be doing that while the writer
// was still using them.
type halt struct {
	// signals is where the runtime delivers. It is buffered because a signal
	// sent while nobody is receiving is dropped, and the first one is the one
	// that matters.
	signals chan os.Signal
	// stopping is closed once a signal has arrived. Closing rather than sending
	// is what makes the answer repeatable: a send can be taken once, and a
	// second reader would then be told the run may carry on.
	stopping chan struct{}
	// received is written before stopping is closed and read only after that
	// close has been seen. The close is what orders the two, which is why this
	// needs no lock and why it must not be read any other way.
	received os.Signal
}

// watchSignals starts watching and returns what a run asks between rows.
func watchSignals() *halt {
	watcher := &halt{
		signals:  make(chan os.Signal, 1),
		stopping: make(chan struct{}),
	}
	signal.Notify(watcher.signals, os.Interrupt, syscall.SIGTERM)
	go watcher.wait()
	return watcher
}

// wait is the only goroutine this product starts.
//
// It reads the channel once and ends, either because a signal arrived or
// because the run finished and released the watch.
func (h *halt) wait() {
	received, arrived := <-h.signals
	if !arrived {
		// The run ended on its own and released the watch.
		return
	}
	// Hand the signal back to the system before anything else. From here on a
	// second press ends the process the way it would have without this file,
	// which matters because the run only stops where it checks: somebody whose
	// run is stuck in a write on a full disk has to be able to get out.
	signal.Stop(h.signals)
	h.received = received
	close(h.stopping)
}

// Stopped returns the failure that ends the run, or nil to carry on.
//
// The two kinds are the two the exit code table already carries, so a run
// stopped from outside ends with the number a shell would have reported for a
// process that never caught the signal at all. They stay apart because a
// pipeline reads them apart: one says a person was there and one says nobody
// was.
func (h *halt) Stopped() error {
	select {
	case <-h.stopping:
		if h.received == syscall.SIGTERM {
			return fault.New(fault.Terminated, stoppedBySystem)
		}
		return fault.New(fault.Interrupted, stoppedByUser)
	default:
		return nil
	}
}

// release ends the watch, and after it the signals mean again what they meant
// before the run started.
//
// Closing the channel is what lets the goroutine end. It is safe in this order
// and in no other: signal.Stop waits for a delivery already on its way, so
// after it returns nothing can send on a channel we are about to close.
func (h *halt) release() {
	signal.Stop(h.signals)
	close(h.signals)
}

// resendGrace is how long Exit waits for the signal it sent to end the process.
//
// A signal this process may receive reaches a thread in far less, so the wait
// is not spent on any run that ends normally. It is a backstop: if the signal
// does not arrive, the run still ends, with the code a shell would have shown.
const resendGrace = 2 * time.Second

// Exit ends the process with the code Run returned.
//
// A run stopped by a signal ends by that signal on Linux and macOS, as the
// section above explains. Every other code ends the process as it is, and so
// does every code on Windows, where there is no signal to send again.
func (env Environment) Exit(code int) {
	if sent, ok := signalBehind(runtime.GOOS, exitCode(code), env.InterruptIgnored); ok {
		resend(sent)
	}
	os.Exit(code)
}

// signalBehind names the signal behind a code, when sending it again would end
// the process. It takes the system by name, so that the answer for each system
// is checked on whichever one the tests run on.
func signalBehind(system string, code exitCode, interruptIgnored bool) (os.Signal, bool) {
	if system == "windows" {
		return nil, false
	}
	switch code {
	case codeInterrupted:
		// A shell starts a command in the background with the interrupt
		// ignored. The watch took that away while the run wrote, and since it
		// stopped the system ignores the interrupt again, so sending it would
		// only wait out the grace.
		if interruptIgnored {
			return nil, false
		}
		return os.Interrupt, true
	case codeTerminated:
		return syscall.SIGTERM, true
	}
	return nil, false
}

// resend sends the signal to this process and waits for it to arrive.
//
// The watch has handed the signal back to the runtime by now, so the runtime
// treats it as a signal nobody asked for. It ends the process by that signal,
// or with 128 plus its number when this is the first process of a container,
// which a signal cannot end that way. resend returns only if neither happened.
func resend(sent os.Signal) {
	self, err := os.FindProcess(os.Getpid())
	if err != nil {
		return
	}
	if self.Signal(sent) != nil {
		return
	}
	time.Sleep(resendGrace)
}

// interruptIgnored reports whether this process was started with the
// interrupt ignored.
//
// 🔴 It is right only until the first watch. signal.Notify takes the ignoring
// away, and once the watch stops the system ignores the interrupt again while
// signal.Ignored keeps saying it does not. That is why FromProcess asks, before
// any run has started.
func interruptIgnored() bool {
	return signal.Ignored(os.Interrupt)
}
