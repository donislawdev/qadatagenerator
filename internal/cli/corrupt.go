package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/damage"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// listOf writes names the way a sentence does.
//
// Two files joined by "and" read fine and three joined by two of them do not,
// which is what a second export form would have produced at every surface that
// announces a run.
func listOf(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	default:
		return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
	}
}

// summaryOf is what a person is told about the damage, on the log channel.
//
// # Why every category is named rather than only the worst
//
// A sentence listing what was applied and nothing else is true and leaves a
// false impression of the scope. A run where fifty nine of sixty ordered
// defects were not applied looks, from a count of applied ones alone, like a
// run that did almost nothing wrong.
//
// The counts arrive as numbers and the sentences are made here, because this
// channel is English always while the other surface is translated. One shared
// sentence would have to be wrong on one of them.
func summaryOf(counts damage.Counts, form string) []string {
	lines := []string{fmt.Sprintf("%s ordered, %d applied, manifest in %s",
		counted(counts.Requested, "defect"), counts.Applied,
		listOf(damage.Files(form)))}
	if missing := counts.Missing(); missing > 0 {
		lines = append(lines, fmt.Sprintf(
			"%s could not be applied, and the manifest carries the reason for each one",
			counted(missing, "defect")))
		// 🔴 The breakdown, because the bare count reads like a shortfall and
		// most of it usually is not one. A build whose catalog is ahead of its
		// engine leaves most of what was ordered unimplemented, which is nobody
		// fault and nothing to act on, while a handful of entries saying the
		// run lacks something is a recipe somebody can fix. Those two look
		// identical in a single number.
		lines = append(lines, counts.ByKind...)
	}
	if counts.Unpartnered > 0 {
		lines = append(lines, fmt.Sprintf(
			"%s about the integrity of the set were not ordered at all, because a table of "+
				"one row has no second record for such a defect to span",
			counted(counts.Unpartnered, "defect")))
	}
	return lines
}

// plannedDamage says what a policy will do, without doing any of it.
//
// # Why a dry run has to answer this at all
//
// A dry run is the first phase without the second, so it has to say everything
// the run would do rather than everything it would write. A recipe carrying a
// policy that went unmentioned would let somebody check their recipe, see a
// clean answer and learn only afterwards that most of the damage they asked for
// has no implementation behind it.
//
// It states what is certain rather than estimating. How many rows a share of
// them comes to is a draw, and a number here that the real run then missed
// would be worse than no number.
func plannedDamage(built recipe.Recipe) []string {
	if !built.Policy.Asks() {
		return nil
	}
	described, err := catalogyaml.Load()
	if err != nil {
		// The catalog is our own data inside our own binary, so a failure to
		// read it is not something a recipe can cause. The real run reports it,
		// and a dry run saying nothing here is better than one that invents a
		// number.
		return nil
	}
	// The numbers are the ones the recipe screen and the damaging verb show,
	// worked out by the same walk over the catalog (DEC-252).
	planned := damage.PlanOf(built.Policy.Layers, described)
	return []string{fmt.Sprintf(
		"the policy draws on %s described in %s, and this build implements %d of them. "+
			"A real run writes one manifest entry per ordered defect, applied or not",
		kindsOfDamage(planned.Ordered), strings.Join(built.Policy.Layers, ", "), planned.Implemented)}
}

// kindsOfDamage counts damage in words.
//
// Written out rather than handed to the general pluraliser, which appends to
// the end of what it is given and would say "kind of damages". The same four
// lines exist in the use case behind validate, and they are four lines rather
// than a shared helper because this text is read by a person deciding whether
// their policy does what they asked, and prose that stumbles there costs more
// than the duplication.
func kindsOfDamage(count int) string {
	if count == 1 {
		return "one kind of damage"
	}
	return strconv.Itoa(count) + " kinds of damage"
}

// tooLargeSentence is how this surface says that one file will not be read
// back.
//
// # Why the size is measured after the file is in place rather than estimated
//
// 🔴 The writer streams and has no ceiling, and the reader holds the whole
// document while a comparison runs, so it stops at a size written beside it.
// Nothing connected the two, and a run producing hundreds of thousands of
// entries could leave a manifest that verify and the viewer both refuse. The
// person who finds that out is the one running verify, days later, with the
// run that made it long finished. Both modes measure the file once it is on the
// disk and hand the size here (damage.Oversized).
//
// It is one function rather than one per caller because both verbs that damage
// reach this through the run's own result, and two wordings for one fact is how
// the two surfaces started drifting the last time.
//
// 🔴 Both of the things that make a manifest grow are named, because only one
// of them was and it is not the one that gets here first. A manifest carries an
// entry for every record damaged and a description of every column of the file.
// Measured on 2026-09-03: a file of 200 000 columns and three rows produced a
// manifest of 38.6 MB with not one entry in it, so a reader told only to damage
// fewer rows was being pointed away from the whole of the cause.
func tooLargeSentence(name string, size int64) string {
	return fmt.Sprintf(
		"%s is %d bytes, and this build reads a manifest of at most %d, so verify and the "+
			"window will refuse it. A manifest carries an entry for every record damaged "+
			"and a description of every column, so damage fewer records or work on a "+
			"narrower file",
		name, size, damage.ReadBackLimit)
}
