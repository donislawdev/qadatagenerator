package background

import (
	"errors"
	"testing"
)

// heldPlace is a place of work that keeps every read it is handed until the
// test answers it, the way a goroutine would some time later.
type heldPlace struct {
	reads   []func() func()
	crashes []func(error) func()
}

func (h *heldPlace) read(read func() func(), crashed func(error) func()) {
	h.reads = append(h.reads, read)
	h.crashes = append(h.crashes, crashed)
}

// answer carries out the read handed over nth and hands its answer back.
func (h *heldPlace) answer(t *testing.T, nth int) {
	t.Helper()
	if nth >= len(h.reads) {
		t.Fatalf("the queue handed over %d reads and this wants the one numbered %d",
			len(h.reads), nth)
	}
	h.reads[nth]()()
}

// asking is a question that records which one it was when it is read.
func asking(read *[]string, name string) func(*Stopper) func() {
	return func(stale *Stopper) func() {
		// Asked where the reading happens, as a real reading asks it. In the
		// test with a goroutine that is what the race detector has to see
		// beside the queue stopping it from the other side.
		_ = stale.Stopped()
		*read = append(*read, name)
		return func() {}
	}
}

func noCrash(error) func() { return func() {} }

// 🔴 One question in flight, and of those asked meanwhile only the newest is
// asked next.
//
// Until 2026-10-27 every keystroke in a recipe started a parse of its own at
// once, side by side, and every character of a separator walked the whole of
// somebody's file beside the others. All the answers but the last were thrown
// away, so the work bought nothing except the memory it took.
func TestOnlyTheNewestOfTheQuestionsAskedMeanwhileIsAsked(t *testing.T) {
	place := &heldPlace{}
	queue := NewLatest(place.read)
	var read []string

	queue.Ask(asking(&read, "first"), noCrash)
	queue.Ask(asking(&read, "second"), noCrash)
	queue.Ask(asking(&read, "third"), noCrash)
	if len(place.reads) != 1 {
		t.Fatalf("three questions put %d reads in flight rather than one", len(place.reads))
	}

	place.answer(t, 0)
	if len(place.reads) != 2 {
		t.Fatalf("the read that ended left %d reads behind it rather than one", len(place.reads)-1)
	}
	place.answer(t, 1)
	if len(read) != 2 || read[1] != "third" {
		t.Errorf("the queue asked %q, and the question waiting was the newest one", read)
	}
	if len(place.reads) != 2 {
		t.Errorf("a question nobody asked again was asked a second time: %d reads", len(place.reads))
	}
}

// A read that comes apart frees the queue for the question waiting behind it.
//
// 🔴 A panic during a read arrives by the second function, and a queue that
// waited for the first one only would stay asking for ever, with every later
// keystroke waiting behind an answer that is never coming.
func TestAReadThatComesApartFreesTheQueue(t *testing.T) {
	place := &heldPlace{}
	queue := NewLatest(place.read)
	var read []string

	crashed := false
	queue.Ask(asking(&read, "first"), func(error) func() {
		return func() { crashed = true }
	})
	place.crashes[0](errors.New("the reading came apart"))()
	if !crashed {
		t.Fatal("the screen's way out of a crash was not taken")
	}

	// A crash of a question already asked again is dropped like its answer
	// would be, and still frees the queue for the one waiting.
	staleCrash := false
	queue.Ask(asking(&read, "second"), func(error) func() {
		return func() { staleCrash = true }
	})
	queue.Ask(asking(&read, "third"), noCrash)
	place.crashes[1](errors.New("the reading came apart"))()
	if staleCrash {
		t.Error("the crash of a question asked again was drawn over the newer one")
	}
	if len(place.reads) != 3 {
		t.Fatalf("the read that came apart left the question behind it unasked: %d reads",
			len(place.reads))
	}
}

// 🔴 A question asked again is told to stop, and what it answers is dropped.
//
// Until 2026-10-27 a reading in flight ran to its end while the newer question
// waited behind it, and only the screen's own count kept its answer off the
// screen (B-341). A reading that asks the Stopper between records now ends at
// the next one.
func TestAQuestionAskedAgainIsStoppedAndItsAnswerDropped(t *testing.T) {
	place := &heldPlace{}
	queue := NewLatest(place.read)
	var stops []*Stopper
	var drawn []string
	question := func(name string) func(*Stopper) func() {
		return func(stale *Stopper) func() {
			stops = append(stops, stale)
			return func() { drawn = append(drawn, name) }
		}
	}

	queue.Ask(question("first"), noCrash)
	queue.Ask(question("second"), noCrash)
	queue.Ask(question("third"), noCrash)
	place.answer(t, 0)
	if stops[0].Stopped() == nil {
		t.Error("the question in flight was not told that it had been asked again")
	}
	if len(drawn) != 0 {
		t.Errorf("the answer to a question asked again was drawn: %q", drawn)
	}

	place.answer(t, 1)
	if stops[1].Stopped() != nil {
		t.Error("the newest question was told to stop while nothing newer had been asked")
	}
	if len(drawn) != 1 || drawn[0] != "third" {
		t.Errorf("the screen drew %q, and the one answer worth drawing is the newest", drawn)
	}
}

// A forgotten question is told to stop, its answer is dropped, and a question
// waiting behind it is never asked.
//
// The recipe screen forgets when it is emptied, which asks nothing new, so
// nothing else would supersede the check in flight.
func TestAForgottenQuestionIsStoppedAndNothingWaitingIsAsked(t *testing.T) {
	place := &heldPlace{}
	queue := NewLatest(place.read)
	var stops []*Stopper
	var drawn []string
	question := func(name string) func(*Stopper) func() {
		return func(stale *Stopper) func() {
			stops = append(stops, stale)
			return func() { drawn = append(drawn, name) }
		}
	}

	// Nothing newer is asked before forgetting, so forgetting is the only
	// thing that can stop the question in flight.
	queue.Ask(question("first"), noCrash)
	queue.Forget()
	place.answer(t, 0)
	if stops[0].Stopped() == nil {
		t.Error("the forgotten question in flight was not told to stop")
	}
	if len(drawn) != 0 {
		t.Errorf("after forgetting, the screen drew %q", drawn)
	}

	// A question waiting behind the one in flight is let go as well.
	queue.Ask(question("second"), noCrash)
	queue.Ask(question("third"), noCrash)
	queue.Forget()
	place.answer(t, 1)
	if len(drawn) != 0 || len(place.reads) != 2 {
		t.Errorf("after forgetting, the screen drew %q and the queue asked %d questions", drawn,
			len(place.reads))
	}

	queue.Ask(question("fourth"), noCrash)
	place.answer(t, 2)
	if len(drawn) != 1 || drawn[0] != "fourth" {
		t.Errorf("a question asked after forgetting was answered as %q", drawn)
	}
}

// A read with nothing to do about it still frees the queue.
//
// A screen is allowed to decide its answer no longer matters and hand back
// nothing, and a place of work handed nothing hands nothing back. The queue
// has to be told anyway, or the next question waits for ever.
func TestAReadWithNothingToDoStillFreesTheQueue(t *testing.T) {
	place := &heldPlace{}
	queue := NewLatest(place.read)
	var read []string

	queue.Ask(func(*Stopper) func() { return nil }, noCrash)
	queue.Ask(asking(&read, "second"), noCrash)
	answer := place.reads[0]()
	if answer == nil {
		t.Fatal("the queue handed the place of work nothing to hand back, so it never hears " +
			"that the read ended")
	}
	answer()
	if len(place.reads) != 2 {
		t.Errorf("the question behind a read with nothing to do was never asked: %d reads",
			len(place.reads))
	}
}

// The queue in front of the real place of work, with a goroutine in between.
//
// The tests above hold every read, so nothing in them runs anywhere else. This
// one reads on a goroutine of its own and answers through a stand-in for the
// thread that draws, which is what the race detector needs to see to say that
// nothing in the queue is touched from the goroutine.
func TestTheQueueWorksInFrontOfAReadOnAnotherGoroutine(t *testing.T) {
	drawing := make(chan func(), 4)
	place := func(read func() func(), crashed func(error) func()) {
		readInBackground(read, crashed, func(act func()) { drawing <- act })
	}
	queue := NewLatest(place)
	var read []string

	queue.Ask(asking(&read, "first"), noCrash)
	queue.Ask(asking(&read, "second"), noCrash)
	queue.Ask(asking(&read, "third"), noCrash)
	(<-drawing)()
	(<-drawing)()
	if len(read) != 2 || read[0] != "first" || read[1] != "third" {
		t.Errorf("the queue read %q, and the questions worth reading were the first and the "+
			"newest", read)
	}
}

// A question asked from inside an answer is asked once the answer is over.
//
// With the place of work a caller with no window uses, the read and its answer
// happen inside the call, so the queue must be free again by then.
func TestTheQueueIsFreeOnceAReadHereHasAnswered(t *testing.T) {
	queue := NewLatest(ReadHere)
	var read []string
	queue.Ask(asking(&read, "first"), noCrash)
	queue.Ask(asking(&read, "second"), noCrash)
	if len(read) != 2 {
		t.Errorf("two questions asked one after the other where the reading happens at once "+
			"were read %d times", len(read))
	}
}
