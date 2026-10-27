package background

// Latest asks a place of work one question at a time, and of the questions
// asked meanwhile keeps only the newest.
//
// # Why at most one, and why the newest
//
// A recipe is judged again on every keystroke, and a file is read again on
// every character of its separator. Until 2026-10-27 each of those was a
// goroutine of its own, started at once and never stopped, so typing a sentence
// into a long recipe started one parse per letter side by side, and the note on
// the recipe reader says a document inside its limit can cost hundreds of
// megabytes to parse. All the answers but the last were thrown away when they
// arrived, so that work bought nothing except the memory it took.
//
// One question in flight keeps the cost of typing to the cost of one question.
// One waiting, replaced by every newer one, keeps the answer at most one
// question behind what is on screen, and a question nobody would look at the
// answer to is never asked.
//
// # Why the question in flight is stopped
//
// A newer question makes the one in flight a question nobody will look at the
// answer to, and until 2026-10-27 it ran to its end anyway while the newer one
// waited behind it (B-341). So every question is handed a Stopper of its own,
// and asking again stops the one in flight: a reading that asks it between
// records ends at the next one, and the queue drops what it answers, because
// that answer is about something nobody has on screen. Reading somebody's file
// asks between records. Judging a recipe asks while the plan shares rows out,
// and not while the document is parsed, which happens inside a library.
//
// # Why it needs no lock
//
// Ask is called on the thread that draws and the answer of a place of work
// arrives there too, which is the only place the state below is read or
// written. The goroutine doing the reading only builds the function that will
// run there, and only reads the Stopper it was handed, whose channel lives in
// background.go with the reasons stated there.
type Latest struct {
	read Read
	// asking is the Stopper of the question handed to the place of work whose
	// answer has not come back, or nothing while there is none.
	asking *Stopper
	// waiting is the newest question asked while one was in flight, or nothing.
	waiting *question
}

type question struct {
	read    func(stale *Stopper) func()
	crashed func(error) func()
}

// NewLatest makes a queue in front of one place of work.
func NewLatest(read Read) *Latest { return &Latest{read: read} }

// Ask is the port Read with the rules above. read is handed the Stopper that
// says its question has been asked again.
func (l *Latest) Ask(read func(stale *Stopper) func(), crashed func(error) func()) {
	if l.asking != nil {
		l.asking.Stop()
		l.waiting = &question{read: read, crashed: crashed}
		return
	}
	stale := NewStopper()
	l.asking = stale
	l.read(func() func() { return l.after(stale, read(stale)) },
		func(err error) func() { return l.after(stale, crashed(err)) })
}

// Forget lets go of every question asked so far: the one in flight is told to
// stop and its answer is dropped, and the one waiting is never asked.
//
// It is for a screen that stops wanting an answer without asking anything new,
// which the recipe screen does when it is emptied. Without it that screen kept
// a count of its own beside the queue, and two ways of telling a stale answer
// apart are one more than anybody can check.
func (l *Latest) Forget() {
	if flight := l.asking; flight != nil {
		flight.Stop()
	}
	l.waiting = nil
}

// after is what goes back to the thread that draws: the screen's own answer,
// unless its question was asked again meanwhile, and then the question that
// waited for it.
//
// 🔴 It is never nothing, even when the screen has nothing to do. A place of
// work handed nothing hands nothing back, so the queue would stay asking for
// ever and every later question would wait behind an answer that is never
// coming. A reading that comes apart arrives here as well, by the second
// function, which is what keeps a panic from leaving the queue stuck.
//
// The answer of a question asked again is dropped whether or not the reading
// noticed, because a reading that finished before it could is about the same
// thing nobody is looking at.
func (l *Latest) after(stale *Stopper, act func()) func() {
	return func() {
		l.asking = nil
		if act != nil && !stale.Asked() {
			act()
		}
		if next := l.waiting; next != nil {
			l.waiting = nil
			l.Ask(next.read, next.crashed)
		}
	}
}
