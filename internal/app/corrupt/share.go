package corrupt

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// How many defects a row carries when the policy asks for more than one.
//
// The recipe format offers two answers, exactly_one and any, and only the first
// is a number. This is what any means here, and it is a decision rather than a
// reading of the format: a row carrying every kind of damage at once would tell
// a test nothing, because the first rejection hides the rest.
const perRowManyCount = 3

// The values the recipe format accepts for per_row. They are written out rather
// than imported from the parser, because the parser owns what a file may say
// and this owns what the engine does about it.
const (
	perRowExactlyOne = "exactly_one"
	perRowAny        = "any"
)

// UsableShare and UsablePerRow report whether a policy taken from somewhere
// other than a recipe can be used, without building anything.
//
// # Why a caller needs to ask before it has a schema
//
// A selector is built from a policy and a schema view, and a schema view of
// somebody else's file costs a whole pass over that file. A share that cannot
// be read is therefore found after the reading rather than before it, which
// makes a typo in a flag cost a scan of a file the user chose the size of.
//
// The rules stay where they are and only the question is exported, so there is
// no second place where a share means something slightly different. The wording
// of the refusal belongs to whoever took the value, because only they know
// whether it came from a line of a recipe or from a flag.
func UsableShare(written string) error {
	_, err := shareOf(written)
	return err
}

// UsablePerRow reports whether a per row setting is one this engine knows.
//
// 🔴 An unknown value has to be refused rather than treated as the safer of the
// two. The engine reads anything that is not the word for several as one defect
// per row, so a misspelling would quietly halve the damage somebody asked for
// and the run would end successfully. The recipe format refuses it on the way
// in, and a policy arriving by flag had no such gate.
func UsablePerRow(written string) error {
	switch written {
	case "", perRowExactlyOne, perRowAny:
		return nil
	default:
		return fault.New(fault.Config, fmt.Sprintf(
			"per_row says %q, and this engine knows %s and %s",
			written, perRowExactlyOne, perRowAny))
	}
}

// PerRowChoices are the values per row accepts, for a message that lists them.
func PerRowChoices() []string { return []string{perRowExactlyOne, perRowAny} }

// share is how much of the data a policy asks to damage.
type share struct {
	// percent is the portion of rows, from zero to one hundred.
	percent float64
	// all is set when every row is damaged, so that a hundred percent does not
	// depend on a draw coming out right.
	all bool
}

// defaultShare is used when a policy names layers and says nothing about how
// much to damage.
//
// Five percent, which is what the recipe format shows in its own example. A
// policy that ordered damage and got none would look like a build that ignored
// it.
const defaultShare = 5

// shareOf reads the portion of rows a policy asks for.
//
// # Why an unreadable value is a failure rather than a default
//
// Until this existed the value was carried as text and never looked at, so
// rows: banana passed validation in silence. A policy meant to damage five
// percent of rows would then damage whatever the default happened to be, and
// the run would end successfully. A condition that quietly stops narrowing
// anything is worse than one that is refused, because the report still reads
// like a report.
func shareOf(written string) (share, error) {
	trimmed := strings.TrimSpace(written)
	if trimmed == "" {
		return share{percent: defaultShare}, nil
	}
	if !strings.HasSuffix(trimmed, "%") {
		return share{}, fault.New(fault.Config, fmt.Sprintf(
			"policy.rows says %q, and this build reads a percentage such as 5%%. "+
				"Particular row numbers are named in the recipe format and are not read yet, "+
				"so write a percentage or remove the line", written))
	}
	value, err := strconv.ParseFloat(strings.TrimSuffix(trimmed, "%"), 64)
	if err != nil {
		return share{}, fault.New(fault.Config, fmt.Sprintf(
			"policy.rows says %q, which is not a number followed by a percent sign", written))
	}
	// 🔴 Not a number and infinity are refused before the range is compared, and
	// the order is the whole point. Every comparison against NaN is false, so the
	// range test below lets it through, and so does the draw that decides which
	// rows are damaged: a share of NaN chooses no row at all. Measured before
	// this line existed: a policy that ordered sixty defects ordered none, and
	// the run ended with success.
	//
	// The recipe parser has refused this since it learned to read a share, and
	// this file did not, because reading a percentage got written twice. The
	// header of this file promises the rules live in one place, so the two are
	// answered the same way here rather than argued about again.
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return share{}, fault.New(fault.Config, fmt.Sprintf(
			"policy.rows says %q, which is not a number this build can count rows with. "+
				"Write a plain number between 0 and 100 followed by a percent sign", written))
	}
	if value < 0 || value > 100 {
		return share{}, fault.New(fault.Config, fmt.Sprintf(
			"policy.rows says %q, and a share of rows runs from 0%% to 100%%", written))
	}
	return share{percent: value, all: value >= 100}, nil
}

// identify names one entry so that a test can refer to the case.
//
// The name is built from where the damage is and what it is, rather than from a
// counter, so that the same seed gives the same names and a test written against
// one manifest still finds its case in the next run.
func identify(entity string, row int, mutator catalog.ID) string {
	return fmt.Sprintf("%s-%d-%s", entity, row, mutator)
}
