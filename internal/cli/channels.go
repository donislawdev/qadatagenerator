package cli

import (
	"fmt"
	"io"
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
func (s *streams) report(p problem) {
	fmt.Fprintln(s.log, s.paint(ansiRed, prefixError)+p.what)
	if p.why != "" {
		fmt.Fprintln(s.log, prefixDetail+p.why)
	}
	if p.allowed != "" {
		fmt.Fprintln(s.log, prefixDetail+labelAllowed+p.allowed)
	}
	if p.instead != "" {
		fmt.Fprintln(s.log, prefixDetail+labelTry+p.instead)
	}
}

// warn writes something the user should see even when asked for less output.
// A warning that --quiet hides is a warning that never happened.
func (s *streams) warn(text string) {
	fmt.Fprintln(s.log, s.paint(ansiYellow, prefixWarning)+text)
}

// info writes progress detail. It is silent unless the run asked for it,
// because ten thousand lines of progress is a storage bill rather than
// diagnostics.
func (s *streams) info(text string) {
	if s.level != levelVerbose {
		return
	}
	fmt.Fprintln(s.log, prefixInfo+text)
}

// note writes something a person asked for, such as the outcome of a check.
// It is silent under --quiet, which asked for failures only.
func (s *streams) note(text string) {
	if s.level == levelQuiet {
		return
	}
	fmt.Fprintln(s.log, text)
}

// detail writes a line that belongs to whatever was written above it.
func (s *streams) detail(text string) {
	if s.level == levelQuiet {
		return
	}
	fmt.Fprintln(s.log, prefixDetail+text)
}
