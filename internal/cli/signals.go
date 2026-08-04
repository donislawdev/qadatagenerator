package cli

import (
	"os"
	"os/signal"

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
// # One signal, and the missing one is a decision rather than an oversight
//
// This catches the interrupt a person sends and not the termination a system
// sends. The reason is not the code: it is that naming SIGTERM means importing
// syscall, and the guard behind the promise of no outbound traffic refuses that
// import everywhere, because syscall can do whatever the operating system can
// and would route around every other name on its list. Buying the second signal
// costs a hole in that guard, so it is a question for the owner rather than a
// line added here. A run stopped by the system still ends with 143 from the
// shell, and it still leaves its temporary files behind.

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
	//
	// Which signal arrived is deliberately not kept. Only one is asked for, so
	// a field holding it would answer a question nobody can ask yet. It comes
	// back on the day a second signal does.
	stopping chan struct{}
}

// watchSignals starts watching and returns what a run asks between rows.
func watchSignals() *halt {
	watcher := &halt{
		signals:  make(chan os.Signal, 1),
		stopping: make(chan struct{}),
	}
	signal.Notify(watcher.signals, os.Interrupt)
	go watcher.wait()
	return watcher
}

// wait is the only goroutine this product starts.
//
// It reads the channel once and ends, either because a signal arrived or
// because the run finished and released the watch.
func (h *halt) wait() {
	if _, arrived := <-h.signals; !arrived {
		// The run ended on its own and released the watch.
		return
	}
	// Hand the signal back to the system before anything else. From here on a
	// second press ends the process the way it would have without this file,
	// which matters because the run only stops where it checks: somebody whose
	// run is stuck in a write on a full disk has to be able to get out.
	signal.Stop(h.signals)
	close(h.stopping)
}

// Stopped returns the failure that ends the run, or nil to carry on.
//
// The kind is the one the exit code table already carries, so an interrupted
// run ends with the number a shell would have reported for a process that never
// caught the signal at all.
func (h *halt) Stopped() error {
	select {
	case <-h.stopping:
		return fault.New(fault.Interrupted, stoppedByUser)
	default:
		return nil
	}
}

// release ends the watch, and after it the signal means again what it meant
// before the run started.
//
// Closing the channel is what lets the goroutine end. It is safe in this order
// and in no other: signal.Stop waits for a delivery already on its way, so
// after it returns nothing can send on a channel we are about to close.
func (h *halt) release() {
	signal.Stop(h.signals)
	close(h.signals)
}
