package cli

import (
	"flag"
	"fmt"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// Everything about answering the question "how do I use this".
//
// Split out of run.go on 2026-08-05, when that file went over the hard ceiling
// on file length. The ceiling may only be lowered, so the answer is to divide
// rather than to raise it, and help is the part that divides cleanly: it prints
// and decides nothing.

// helpWith answers the most common thing somebody types, once its arguments
// have been read.
//
// # Why help reads flags although it takes none of its own
//
// Its help lists the flags of the tool as accepted before or after the command,
// as the help of every verb does, and --help is a flag of every verb. Until
// 2026-11-03 this verb read every argument after it as the name of a command,
// so qadg help --quiet was refused with "no help for --quiet", and qadg help
// --help the same way. --help written after help asks about help itself, as it
// asks about the verb it follows everywhere else, whatever command is named.
func helpWith(out *streams, given parsed[noFlags]) int {
	return answerHelp(out, given.rest)
}

// helpInsteadOf says what to type after a flag help does not define.
//
// A command written with dashes in front of it, qadg help --generate, is such a
// flag. While help read no flags the same slip was refused as a subject, beside
// the command it was close to, and reading flags must not lose that. The flag
// refused is the first one the set does not define, because the parse refuses
// them in the order typed, and none of the flags of help takes a value, so an
// argument that looks like a flag is one.
func helpInsteadOf(set *flag.FlagSet, arguments []string) string {
	for _, argument := range arguments {
		if argument == "--" {
			break
		}
		if !looksLikeFlag(argument) {
			continue
		}
		name, _, _ := strings.Cut(strings.TrimPrefix(argument[1:], "-"), "=")
		if set.Lookup(name) != nil {
			continue
		}
		if near, close := nearestOf(name, helpSubjects()); close {
			return problemHelpSubjectDo + " " + near
		}
		break
	}
	return problemHelpSubjectDo + " " + commandHelp
}

// answerHelp prints what was asked about.
//
// With nothing named, it is --help. With a command named, it is the help of
// that command, which is the same text that command prints for --help. One
// renderer rather than two, because two would answer the same question
// differently the first time somebody edited one of them.
func answerHelp(out *streams, subjects []string) int {
	switch {
	case len(subjects) == 0:
		printHelp(out)
		return finish(out, codeOK)
	case len(subjects) > 1:
		return usage(out, problem{
			what:    problemHelpArguments,
			why:     problemHelpArgumentsWhy,
			allowed: problemHelpArgumentsAllow,
			instead: problemHelpSubjectDo + " " + subjects[0],
		})
	}

	// 🔴 One subject that is not a command, and the refusal below knows it.
	// Until 2026-09-03 the only way to learn which generators this build has
	// was to name one it does not have and read the refusal, on both surfaces.
	// A vocabulary somebody can only reach by making a mistake is a vocabulary
	// this tool keeps to itself.
	if subjects[0] == subjectGenerators {
		out.emit(renderGenerators())
		return finish(out, codeOK)
	}

	doc, found := commandDocFor(subjects[0])
	if !found {
		refusal := unknownCommand(subjects[0])
		refusal.what = problemHelpSubject + subjects[0]
		refusal.why = problemHelpSubjectWhy
		refusal.allowed = strings.Join(helpSubjects(), ", ")
		refusal.instead = problemHelpSubjectDo
		if near, close := nearestOf(subjects[0], helpSubjects()); close {
			refusal.instead = problemHelpSubjectDo + " " + near
		}
		return usage(out, refusal)
	}
	out.emit(renderCommandHelp(doc))
	return finish(out, codeOK)
}

// helpSubjects is everything help answers about: every command, and the one
// word that is not one.
//
// The word comes last rather than in sorted order, because that is the order
// the sentence above it uses and because it is a different kind of thing from
// the five before it.
func helpSubjects() []string { return append(knownCommands(), subjectGenerators) }

// renderGenerators prints every generator this build carries, what it produces
// and where it can be produced.
//
// # Why the locales are columns rather than a sentence
//
// Whether a generator can be filled depends on the locale, and the answer is
// not the same for all of them: this build has no cities or regions for en-US
// and no register of postal codes for pl-PL. A list without that column would
// name three generators that produce nothing in the locale somebody is about to
// write, which is the shape of an answer that is true and useless.
//
// # Why a reason under the table rather than in the cell
//
// The reason is a sentence with the way out in it, and a sentence does not fit
// in a column. Every no gets one, so nothing is left as a bare no.
func renderGenerators() string {
	// The locale of this source decides nothing here: every question below
	// names the locale it asks about. It exists because the register is reached
	// through a source rather than as a package level table.
	source := values.New("", "help")
	locales := values.Locales()
	names := source.Names()

	width, produced := 0, 0
	for _, name := range names {
		width = max(width, len(name))
		produced = max(produced, len(producedBy(source, name)))
	}

	// Every line is trimmed on the right before it is joined. A column padded to
	// the width of its heading leaves spaces nobody can see and every diff can,
	// and this output is the kind of thing people paste into a bug report.
	var lines, missing []string
	head := fmt.Sprintf("  %-*s   %-*s", width, generatorsColumnName, produced, generatorsColumnProduces)
	for _, locale := range locales {
		head += "   " + locale
	}
	lines = append(lines, head)
	for _, name := range names {
		row := fmt.Sprintf("  %-*s   %-*s", width, name, produced, producedBy(source, name))
		for _, locale := range locales {
			answer := generatorsYes
			if fine, why := source.CanProduce(name, locale); !fine {
				answer = generatorsNo
				missing = append(missing, "  "+name+" in "+locale+": "+why)
			}
			row += fmt.Sprintf("   %-*s", len(locale), answer)
		}
		lines = append(lines, strings.TrimRight(row, " "))
	}

	var b strings.Builder
	b.WriteString(generatorsHeading + "\n" + generatorsUse + "\n\n")
	b.WriteString(strings.Join(lines, "\n"))
	b.WriteString("\n\n" + generatorsLocaleNote)
	if len(missing) > 0 {
		b.WriteString("\n\n" + generatorsMissingIntro + "\n" + strings.Join(missing, "\n"))
	}
	return b.String()
}

// producedBy says what one generator makes, naming the numbering scheme behind
// it when there is one.
//
// A scheme of no one country that covers some countries only names them too,
// because the columns beside it answer a different question: an account number
// is produced in every locale, and in each of them it is Polish (DEC-281).
func producedBy(source values.Source, generator string) string {
	kind, _ := source.TypeOf(generator)
	scheme, behind := source.ProviderOf(generator)
	if !behind {
		return kind
	}
	if countries := source.CountriesOf(generator); len(countries) > 0 {
		return kind + " (" + scheme + ", " + strings.Join(countries, ", ") + generatorsOnly + ")"
	}
	return kind + " (" + scheme + ")"
}

// declareVerbHelp gives a command the two spellings of the help flag and
// returns the question of whether either was given.
//
// A verb has to declare them itself. The parser of the tool stops at the first
// argument that is not a flag, so anything after the verb belongs to the verb,
// and without this qadg generate --help would be refused as an unknown flag.
func declareVerbHelp(set *flag.FlagSet) func() bool {
	long := set.Bool("help", false, "")
	short := set.Bool("h", false, "")
	return func() bool { return *long || *short }
}

// printCommandHelp prints what one verb says about itself.
func printCommandHelp(out *streams, name string) int {
	doc, found := commandDocFor(name)
	if !found {
		// Unreachable while every verb the dispatcher accepts is in the
		// register, which a test asserts. Answering rather than printing
		// nothing keeps a future verb from failing silently.
		return fail(out, fault.New(fault.Runtime, "no help for the command "+name))
	}
	out.emit(renderCommandHelp(doc))
	return finish(out, codeOK)
}

func printHelp(out *streams) {
	blocks := []string{
		summaryLine,
		examplesBlock,
		usageBlock,
		renderCommands(),
		renderFlags(),
		channelsBlock,
		exitCodesHeading + "\n" + renderExitCodes(),
	}
	out.emit(strings.Join(blocks, "\n\n"))
}
