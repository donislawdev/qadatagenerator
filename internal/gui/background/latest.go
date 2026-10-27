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
// # What it does not do
//
// It does not stop the question in flight. A reading already going on runs to
// its end and the screen then drops its answer, by the count of questions it
// keeps for itself. Stopping it would need the reading to look up between
// records, and none of the readings asked through this does.
//
// # Why it needs no lock
//
// Ask is called on the thread that draws and the answer of a place of work
// arrives there too, which is the only place the state below is read or
// written. The goroutine doing the reading only builds the function that will
// run there.
type Latest struct {
	read Read
	// asking says that a question has been handed to the place of work and its
	// answer has not come back.
	asking bool
	// waiting is the newest question asked while one was in flight, or nothing.
	waiting *question
}

type question struct {
	read    func() func()
	crashed func(error) func()
}

// NewLatest makes a queue in front of one place of work.
func NewLatest(read Read) *Latest { return &Latest{read: read} }

// Ask is the port Read with the two rules above.
func (l *Latest) Ask(read func() func(), crashed func(error) func()) {
	if l.asking {
		l.waiting = &question{read: read, crashed: crashed}
		return
	}
	l.asking = true
	l.read(func() func() { return l.after(read()) },
		func(err error) func() { return l.after(crashed(err)) })
}

// after is what goes back to the thread that draws: the screen's own answer,
// and then the question that waited for it.
//
// 🔴 It is never nothing, even when the screen has nothing to do. A place of
// work handed nothing hands nothing back, so the queue would stay asking for
// ever and every later question would wait behind an answer that is never
// coming. A reading that comes apart arrives here as well, by the second
// function, which is what keeps a panic from leaving the queue stuck.
func (l *Latest) after(act func()) func() {
	return func() {
		l.asking = false
		if act != nil {
			act()
		}
		if next := l.waiting; next != nil {
			l.waiting = nil
			l.Ask(next.read, next.crashed)
		}
	}
}
