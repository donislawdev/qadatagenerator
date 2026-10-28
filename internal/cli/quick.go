package cli

import (
	"errors"
	"flag"
	"io"
	"strconv"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/quick"
)

// runQuick writes one value and nothing else.
//
// # Why this verb has no output beyond the value
//
// It is the one command whose whole answer is meant to be piped, pasted or read
// off a terminal in a second. A note about what was produced, a manifest or a
// summary would each be another line to skip past, and the run writes nothing
// to a disk for any of them to describe.
//
// The seed is the exception to that silence and it is deliberate: a run that
// drew its own seed says so on the log channel, because otherwise the one thing
// somebody needs in order to ask for the same value again is the one thing they
// cannot get. The log channel rather than the data channel, so a pipe reading
// the value never sees it.
func runQuick(out *streams, arguments []string, global crossCutting) int {
	set := flag.NewFlagSet(commandQuick, flag.ContinueOnError)
	set.SetOutput(io.Discard)
	local := declareCrossCutting(set)
	askedForHelp := declareVerbHelp(set)
	locale := set.String("locale", "", "")
	seed := set.String("seed", "", "")
	gender := set.String("gender", "", "")
	birthDate := set.String("birth-date", "", "")
	// A string rather than set.Int, as every other flag of this tool is, so
	// that a number nobody could read is refused in the four parts this
	// project writes rather than in the flag package's own one line.
	count := set.String("count", "1", "")

	// 🔴 The name is taken off the front before the flags are read, and this
	// is not a stylistic choice. The flag package stops at the first argument
	// that is not a flag, so qadg quick pesel --gender female would leave the
	// two flags unparsed and the verb would refuse a command that looks right
	// to everybody. Both documents describing this verb write the name first,
	// and so does anybody typing it.
	//
	// Taking it only when it is genuinely first keeps the other order working
	// too: with a flag in front, the flag package reads the flags and leaves
	// the name behind as the one argument, which is where the code below looks
	// for it. Scanning for the first argument that is not a flag would have
	// been wrong, because the VALUE of a flag is not a flag either.
	named := ""
	if len(arguments) > 0 && !strings.HasPrefix(arguments[0], "-") {
		named, arguments = arguments[0], arguments[1:]
	}

	if err := set.Parse(arguments); err != nil {
		return usage(out, flagFailure(err, acceptedFlagsOf(commandQuick),
			problemHelpSubjectDo+" "+commandQuick))
	}
	if askedForHelp() {
		return printCommandHelp(out, commandQuick)
	}
	flags := global.merge(local)
	if code, ok := applyVerbosity(out, flags); !ok {
		return code
	}

	named, arity := oneGenerator(named, set.Args())
	if arity.what != "" {
		return usage(out, arity)
	}

	if refusal, tooLong := seedRefused(*seed); tooLong {
		return usage(out, refusal)
	}

	wanted, shape := wholeCount(*count)
	if shape.what != "" {
		return usage(out, shape)
	}

	return writeValues(out, *locale, *seed, named, wanted, boundBy(set, *gender, *birthDate))
}

// oneGenerator is the one generator the command line named, whether it stood
// before the flags or after them, or the refusal when there was not exactly one.
//
// Apart from runQuick because that function passed the limit on the length of
// a function. The seam is the one wholeCount below already uses: the flags are
// read there, and what the arguments left over mean is answered here.
func oneGenerator(named string, left []string) (string, problem) {
	if named == "" && len(left) == 1 {
		named, left = left[0], nil
	}
	if named == "" || len(left) > 0 {
		return "", problem{
			what:    problemQuickArguments,
			why:     problemQuickArgumentsWhy,
			allowed: problemQuickArgumentsAllowed,
			instead: problemQuickArgumentsDo,
		}
	}
	return named, problem{}
}

// wholeCount reads how many values were asked for, and says which way the text
// failed rather than that it failed.
//
// 🔴 # Why only the SHAPE is decided here
//
// A count below one is refused too, and not here: it is refused by the mode,
// which is where the reason lives, so that the window and the command line
// cannot come to disagree about it. What cannot be asked of the mode is text
// that never became a number, because the mode takes an int. So this answers
// "is this a number" and the mode answers "is this a number of values".
//
// strconv.Atoi is base ten whatever the digits look like, which matters here
// for a reason this project has already paid for: a count of 010 in a recipe
// once produced eight rows, because the reader it went through read a leading
// nought as octal. Atoi reads 010 as ten, and the guard beside this says so.
func wholeCount(text string) (int, problem) {
	// 🔴 The reading is the mode's since 2026-09-09, so that the window's field
	// and this flag cannot come to disagree about what 010 means. What stays
	// here is the WORDING: this surface says what, why and instead, and the
	// window says one sentence out of a catalogue.
	number, refused := quick.CountFrom(text)
	if refused.Kind == quick.NotRefused {
		return number, problem{}
	}
	_, err := strconv.Atoi(text)
	if errors.Is(err, strconv.ErrRange) {
		return 0, problem{
			what:    problemQuickCountRange + text,
			why:     problemQuickCountRangeWhy,
			instead: problemQuickCountRangeDo,
		}
	}
	return 0, problem{
		what:    problemQuickCountShape + text,
		why:     problemQuickCountShapeWhy,
		instead: problemQuickCountShapeDo,
	}
}

// boundBy is what the two consistency flags said, and nothing for the ones that
// were left out.
//
// A flag left out is not the same as a flag given empty, and the agreement
// carries only what was actually said. Binding an empty gender would make the
// run refuse a value nobody typed.
func boundBy(set *flag.FlagSet, gender, birthDate string) map[string]string {
	agrees := map[string]string{}
	if isSet(set, "gender") {
		agrees[provider.RoleGender] = gender
	}
	if isSet(set, "birth-date") {
		agrees[provider.RoleBirthDate] = birthDate
	}
	return agrees
}

// quickRefusalSays is every refusal this mode can produce, in the words this
// surface uses for it.
//
// A register rather than a switch, for the reason written beside the one next
// door: a switch has a last branch, and a kind added to the mode and forgotten
// here lands in it, producing a confident four part message about a different
// refusal entirely. A register can be held against the list of kinds in both
// directions, and the guard beside it does that.
var quickRefusalSays = map[quick.RefusalKind]refusalWords{
	quick.UnknownGenerator: {
		what: problemQuickUnknownGenerator, why: problemQuickUnknownGeneratorWhy,
		instead: problemQuickUnknownGeneratorDo,
	},
	quick.RoleNotRead: {
		what: problemQuickRoleNotRead, why: problemQuickRoleNotReadWhy,
		instead: problemQuickRoleNotReadDo,
	},
	quick.UnreadableGender: {
		what: problemQuickGender, why: problemQuickGenderWhy,
		instead: problemQuickGenderDo,
	},
	quick.UnreadableDate: {
		what: problemQuickDate, why: problemQuickDateWhy,
		allowed: problemQuickDateAllowed, instead: problemQuickDateDo,
	},
	quick.CountNotPositive: {
		what: problemQuickCount, why: problemQuickCountWhy,
		allowed: problemQuickCountAllowed, instead: problemQuickCountDo,
	},
}

// quickRefusalOf turns a refusal of the mode into the four parts a person reads.
func quickRefusalOf(refusal quick.Refusal) problem {
	words, known := quickRefusalSays[refusal.Kind]
	if !known {
		// Loud rather than a message about the wrong thing. A kind with no
		// words here is a defect of ours, and the guard beside this register
		// exists so that this branch stays unreachable.
		return problem{
			what:    problemQuickUnknownRefusal,
			why:     problemQuickUnknownRefusalWhy,
			instead: doAboutRuntime,
		}
	}
	what := words.what + refusal.Value
	if refusal.Kind == quick.RoleNotRead {
		what += problemQuickRoleNotReadJoin + refusal.Role + inLocale(refusal.Locale)
	}
	// The register wins where it has an answer, because a refusal whose
	// accepted values are a sentence rather than a list has no list to carry.
	allowed := words.allowed
	if allowed == "" {
		allowed = strings.Join(refusal.Allowed, ", ")
	}
	return problem{
		what:    what,
		why:     words.why,
		allowed: allowed,
		instead: words.instead,
	}
}

// inLocale names the locale only where it is what makes the answer no.
//
// A surname reads the gender in Polish and not in English, so a refusal leaving
// the locale out would read as a rule of the tool rather than as a property of
// the data somebody asked for.
func inLocale(locale string) string {
	if locale == "" {
		return ""
	}
	return problemQuickInLocale + locale
}

// writeValues draws the values and puts them on the data channel as they come.
//
// Split from reading the command line above it for the reason every other verb
// here is split the same way: what was asked for and what is done about it are
// two things, and the second one is what a test wants to reach.
//
// 🔴 # Why a signal is not caught here, unlike in generate and corrupt
//
// The command line contract catches a signal only where a run has something to
// take back (docs/09), and this verb writes no file. Catching it would also
// change what a shell does: measured 2026-10-28 with bash 5.3.9, a loop whose
// command catches the interrupt and exits with 130 runs on to its next round,
// while a loop whose command dies of it ends. This is the verb people put in a
// loop. A value is never cut in half by the kill either, see batchBytes.
func writeValues(out *streams, locale, chosenSeed, generator string, count int, agrees map[string]string) int {
	drawn := chosenSeed
	if drawn == "" {
		fresh, err := quick.NewSeed()
		if err != nil {
			return fail(out, err)
		}
		drawn = fresh
	}

	asked := quick.Request{
		Generator: generator, Locale: locale, Seed: drawn, Agrees: agrees, Count: count,
	}
	if refusal := quick.Check(asked); refusal.Refused() {
		return usage(out, quickRefusalOf(refusal))
	}

	// One value to a line, and nothing around them. A count, a heading or a
	// blank line between them would each be something for a pipe to strip, and
	// this is the one verb whose whole answer is meant to be piped. The first
	// failed write ends the drawing, because nothing after it can arrive.
	wrote := false
	err := quick.Each(asked, func(value string) error {
		wrote = true
		out.emitBatched(value)
		return out.dataErr
	})
	out.flushBatch()
	if chosenSeed == "" && wrote {
		// note rather than info, and the difference is the whole point: info is
		// silent unless somebody asked for progress detail, and this is not
		// progress. It is the one thing needed to ask for the same value
		// again, so it is silent only under --quiet, which asked for failures
		// alone.
		//
		// Said after the value rather than before it, so that a person reading
		// a terminal sees what they asked for first. The two channels are
		// separate, so the order only matters to a pair of eyes.
		//
		// Said also when the output failed after the first value, because a
		// reader that took the first values and left needs this line to get
		// the same ones again. Not said when nothing was drawn at all.
		out.note(quickSeedDrawn + drawn)
	}
	if err != nil {
		return finishFailed(out, err)
	}
	return finish(out, codeOK)
}

// isSet reports whether that flag was given on the command line.
//
// The flag package has no other way to tell an absent flag from one given its
// zero value, and here the difference is the whole meaning: an absent gender
// draws one, and an empty gender is a value no scheme can encode.
func isSet(set *flag.FlagSet, name string) bool {
	given := false
	set.Visit(func(f *flag.Flag) {
		if f.Name == name {
			given = true
		}
	})
	return given
}
