package manifestexport

import (
	"io"

	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
)

// Export is a form the entries of a run are written in beside the canonical
// manifest.
//
// Every form is a peer of that manifest rather than a reading of it: all of
// them are handed the same entries at the same moment, so none is another's
// input and they cannot drift. What separates them is not the information,
// which is the same, but who opens the file and what that costs.
type Export interface {
	// Add takes one entry, at the moment it happens.
	Add(defect manifest.Defect)
	// Finish writes whatever closes the file. A form left unfinished is a file
	// that stops mid table without saying so.
	Finish() error
}

// The forms this build writes, and the file each is written to.
//
// 🔴 One table rather than a switch in each place that needs an answer. The
// flag that refuses an unknown form, the code that opens the file and the
// sentences telling a person what was written all read from here, so a form
// added to some of them and not the others cannot happen.
const (
	FormGherkin  = "gherkin"
	FormCSV      = "csv"
	FormMarkdown = "markdown"
)

var files = []struct {
	form string
	name string
}{
	// The stem matches the canonical manifest beside it so that the files of
	// one run are obviously one run's output, and the extension is the one the
	// tool that opens each form expects.
	{FormGherkin, "expectation-manifest.feature"},
	{FormCSV, "expectation-manifest.csv"},
	{FormMarkdown, "expectation-manifest.md"},
}

// GherkinName is the file the Gherkin form is written to.
const GherkinName = "expectation-manifest.feature"

// Forms are the shapes this build can export a manifest in, in the order a
// message lists them.
//
// The canonical JSON is deliberately absent: it is always written and is not
// something to ask for.
func Forms() []string {
	named := make([]string, 0, len(files))
	for _, known := range files {
		named = append(named, known.form)
	}
	return named
}

// FileFor is the file a form is written to, empty when no export was asked for.
//
// It is here rather than at the surfaces that announce a run, because a run
// says what it wrote in more than one place and those places must not disagree
// about how many files there are. An export nobody names is a file the caller
// asked for and has no confirmation of, which reads exactly like a form that
// was never written.
func FileFor(form string) string {
	for _, known := range files {
		if known.form == form {
			return known.name
		}
	}
	return ""
}

// Usable reports whether a form is one this build writes.
//
// The empty string is usable and means no export, which is the default. That is
// the one answer a caller has to be able to give without naming anything.
func Usable(form string) bool {
	return form == "" || FileFor(form) != ""
}

// New starts the export a form asks for, or nothing when none was asked for.
//
// The catalog and the number of rows are taken rather than discovered, because
// they are what makes the bounded columns bounded: the widest identifier this
// build can ever write is in the catalog, and the widest row number is the
// count of rows the run was given.
func New(form string, to io.Writer, described catalog.Catalog, rows int) Export {
	switch form {
	case FormGherkin:
		return NewGherkin(to, described, rows)
	case FormCSV:
		return NewCSV(to)
	case FormMarkdown:
		return NewMarkdown(to, described, rows)
	}
	return nil
}
