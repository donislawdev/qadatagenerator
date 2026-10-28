package cli

import (
	"fmt"
	"io"
	"strings"
)

// Data goes to standard output and the log goes to standard error. Nothing
// crosses. A script that redirected standard output to a file relies on this
// directly, and one warning printed on the wrong channel makes that file
// unparseable.
//
// Everything that writes goes through this type. Calls to fmt.Fprintf spread
// across a program break the split at the first moment of inattention, and
// nothing catches it, because the output still looks fine on a terminal.
type streams struct {
	data  io.Writer
	log   io.Writer
	color bool
	level verbosity

	// verb is which command is running, empty before one is chosen.
	//
	// 🔴 It is here because the last part of a refusal, what to do instead,
	// stopped being answerable from the kind of the failure alone. One sentence
	// per kind was right while every verb worked from a recipe. The verb that
	// damages somebody else's file has none, and the sentence sending its user
	// to run validate on their recipe was a true sentence about the callers
	// that existed when it was written. Measured by running the binary, with
	// every test green.
	verb string

	// dataErr keeps the first failed write to the data channel. A run whose
	// output went nowhere must not end with success, and a broken pipe is
	// the ordinary way that happens.
	dataErr error

	// batch is data emitBatched has not written yet, always whole lines.
	batch []byte
}

type verbosity int

const (
	levelNormal verbosity = iota
	levelQuiet
	levelVerbose
)

// problem is a failure described the way a person needs to read it: what
// happened, why, what would have been accepted, and what to do instead. The
// last three are the ones usually left out, and without them the reader ends
// up in the source code.
type problem struct {
	what    string
	why     string
	allowed string
	instead string
}

// ANSI colors, used only when the log channel is a terminal. A CI log full of
// escape sequences is harder to read than one without color.
const (
	ansiReset  = "\x1b[0m"
	ansiRed    = "\x1b[31m"
	ansiYellow = "\x1b[33m"
)

func (s *streams) paint(color, text string) string {
	if !s.color {
		return text
	}
	return color + text + ansiReset
}

// replacement is what a control character from outside becomes on the log
// channel. It is one character wide, so a length already bounded upstream
// stays bounded.
const replacement = '�'

// plain removes the control characters from text that came from outside.
//
// # The finding this exists for
//
// A terminal obeys what it is sent. Measured on 2026-08-02: a recipe naming a
// key with an escape sequence in it put those bytes straight into one of our
// own messages. The sequence cleared the line and repainted the text, so a
// refusal could be made to read on screen as an approval.
//
// Neither --no-color nor NO_COLOR helped, and that is the part worth keeping
// in mind. Both govern the colour this tool adds. Neither has anything to do
// with what the data carries, so the promise about escape sequences in the
// command line contract was answering a different question than the one an
// attacker asks.
//
// # Why the log channel only
//
// The data channel is deliberately left alone. Data is the product, and layer
// L0 of the corruption catalog exists to inject control characters on purpose,
// so cleaning that channel would break the thing this tool is for. The log is
// prose meant for a person to read on a terminal, and that is the only place
// a terminal is listening.
//
// The machine readable report turned out to be safe already, since the JSON
// encoder writes  rather than the byte. A test says so out loud, so that
// changing the encoder cannot take that away quietly.
func plain(text string) string {
	if !strings.ContainsFunc(text, isControl) {
		return text
	}
	var built strings.Builder
	built.Grow(len(text))
	for _, symbol := range text {
		if isControl(symbol) {
			built.WriteRune(replacement)
			continue
		}
		built.WriteRune(symbol)
	}
	return built.String()
}

// isControl covers the C0 range, delete, and the C1 range that some terminals
// still act on.
func isControl(symbol rune) bool {
	return symbol < 0x20 || (symbol >= 0x7f && symbol <= 0x9f)
}

// emit writes one line of data to standard output.
func (s *streams) emit(text string) {
	if _, err := fmt.Fprintln(s.data, text); err != nil && s.dataErr == nil {
		s.dataErr = err
	}
}

// batchBytes is the most data emitBatched holds back before writing it.
//
// 🔴 512 is _POSIX_PIPE_BUF, the least PIPE_BUF any POSIX system may have, and
// PIPE_BUF is the largest write to a pipe the standard promises goes in whole
// (pubs.opengroup.org, limits.h and write(), checked 2026-10-28). A batch of
// whole lines no larger than that cannot end in the middle of a line when the
// process is killed while the reader is slow, which one write per line already
// promised. A larger batch can be cut there. Linux has 4096, and nothing on
// Windows promises either number.
//
// It costs almost nothing of what batching is for. A PESEL is twelve bytes with
// its line end, so a batch carries forty two of them, and two million values
// are fewer than fifty thousand writes rather than two million.
const batchBytes = 512

// emitBatched writes one line of data to standard output together with the
// lines around it, for a verb whose answer is many lines.
//
// One write per line is a call into the system per value. Two million PESELs
// read through a pipe took 6.43 s that way and 2.62 s in batches, with the same
// bytes (median of three runs after a discarded first, the two binaries
// interleaved, 2026-10-28). A line is never split between two writes. Whatever is still
// held is written by flushBatch, which the verb calls before it says anything
// on the log channel and before it ends.
func (s *streams) emitBatched(text string) {
	if len(s.batch)+len(text)+1 > batchBytes {
		s.flushBatch()
	}
	s.batch = append(s.batch, text...)
	s.batch = append(s.batch, '\n')
}

// flushBatch writes what emitBatched has held back, keeping the first failure
// the way emit does.
//
// After a failed write the batch is dropped rather than tried. A write that got
// through after one that did not would leave a gap in the middle of the values,
// where a reader expects at worst an end.
func (s *streams) flushBatch() {
	if len(s.batch) == 0 {
		return
	}
	if s.dataErr == nil {
		if _, err := s.data.Write(s.batch); err != nil {
			s.dataErr = err
		}
	}
	s.batch = s.batch[:0]
}

// report writes a failure on the log channel, in four lines.
//
// A failed write to the log is not kept, unlike one to the data channel. The
// log describes the run and the data is the run, so losing the description
// must not change what the run is worth. There is also nowhere left to
// complain about a log that cannot be written to.
// Every part goes through plain. Three of the four usually come from our own
// constants, where it does nothing, and the fourth usually carries a name that
// came out of somebody's recipe.
func (s *streams) report(p problem) {
	s.line(s.paint(ansiRed, prefixError) + plain(p.what))
	if p.why != "" {
		s.line(prefixDetail + plain(p.why))
	}
	if p.allowed != "" {
		s.line(prefixDetail + labelAllowed + plain(p.allowed))
	}
	if p.instead != "" {
		s.line(prefixDetail + labelTry + plain(p.instead))
	}
}

// line writes one line to the log channel and drops the result.
//
// The decision to drop it is the one explained above report, and it is made
// here once rather than at each of the calls that write a log line. A failed
// write repeated eight times would be eight places to remember the same
// reasoning, which is how a deliberate choice turns into an oversight.
func (s *streams) line(text string) {
	_, _ = fmt.Fprintln(s.log, text)
}

// warn writes something the user should see even when asked for less output.
// A warning that --quiet hides is a warning that never happened.
func (s *streams) warn(text string) {
	s.line(s.paint(ansiYellow, prefixWarning) + plain(text))
}

// info writes progress detail. It is silent unless the run asked for it,
// because ten thousand lines of progress is a storage bill rather than
// diagnostics.
func (s *streams) info(text string) {
	if s.level != levelVerbose {
		return
	}
	s.line(prefixInfo + plain(text))
}

// note writes something a person asked for, such as the outcome of a check.
// It is silent under --quiet, which asked for failures only.
func (s *streams) note(text string) {
	if s.level == levelQuiet {
		return
	}
	s.line(plain(text))
}

// detail writes a line that belongs to whatever was written above it.
func (s *streams) detail(text string) {
	if s.hidesDetail() {
		return
	}
	s.line(prefixDetail + plain(text))
}

// hidesDetail reports whether the lines belonging under a headline are being
// suppressed.
//
// A caller writing a headline needs to know, because a headline whose detail
// never arrives has to say enough on its own. Asking this beats comparing the
// level at the call site, which would put the meaning of --quiet in two places.
func (s *streams) hidesDetail() bool { return s.level == levelQuiet }
