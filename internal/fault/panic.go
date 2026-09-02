package fault

import "fmt"

// Crashed and CrashedDo are what both surfaces say when this program stops in
// a way it does not understand.
//
// # Why the words live here rather than beside either surface
//
// They were the command line's until 2026-09-02, and the window could not
// import them, because a window importing the command line would be one
// surface built out of the other. The window then grew the same need, and the
// answer to two surfaces needing one sentence is the same one this project
// gives everywhere else: the sentence moves to what they already share.
//
// They are worded so that a bug report can quote them, which is the whole of
// what a person can usefully do with a defect of ours.
const (
	Crashed   = "internal defect: the run stopped unexpectedly"
	CrashedDo = "report this together with the command that produced it"
)

// FromPanic turns a recovered panic into a failure with a kind.
//
// # Why a panic becomes a value at all
//
// A panic that nobody recovers ends the process with Go's own exit code, which
// is 2, and 2 is the number this tool uses for a wrong invocation. A pipeline
// would read a defect of ours as a mistake of its own. The command line has
// answered this since it could run anything. The window had no answer until
// 2026-09-02 and needs one more badly, because a window that ends mid sentence
// takes whatever was typed into it along.
//
// # What this is not for
//
// It is not tidying up. Measured on 2026-09-02: a deferred call on the
// goroutine that panics runs even when nothing recovers it, so the files a run
// was writing are removed either way. What this saves is the account of what
// happened, and on a window also the window.
//
// The recovered value is shortened on the way in, on the same terms as every
// other piece of text that reaches a person from outside a message we wrote.
func FromPanic(recovered any) *Error {
	return New(Runtime, Crashed+": "+Shorten(fmt.Sprint(recovered)))
}
