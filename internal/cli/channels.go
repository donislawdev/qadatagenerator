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

	// dataErr keeps the first failed write to the data channel. A run whose
	// output went nowhere must not end with success, and a broken pipe is
	// the ordinary way that happens.
	dataErr error
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
	if s.level == levelQuiet {
		return
	}
	s.line(prefixDetail + plain(text))
}
