package cli

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// This file holds every piece of text the command line shows a person. It is
// the only file in the package allowed to contain one.
//
// Why one file instead of literals next to the code that prints them: the
// command line is English and only English, so it needs no translation
// machinery, but text scattered across a program drifts in tone, repeats
// itself and is impossible to review as a whole. Collecting it here costs
// nothing today and makes the later question of translation keys a mechanical
// change rather than a rewrite.
//
// Rules that apply to everything below. Sentence one says what the thing does,
// sentence two says the consequence or the limit, and there is no sentence
// three. No internal vocabulary: a reader has never heard of a layer or a
// cross cutting module. Flat hyphens only and no semicolons.

const (
	// summaryLine is the first thing a new user reads. It says what the tool
	// produces, which is the dataset and the claim about it, not just data.
	summaryLine = "qadg produces test data together with a manifest that says how the system under test should react to it."

	// examplesBlock comes before the usage pattern, and the order is the whole
	// point of it. A pattern like qadg <command> [arguments] tells somebody who
	// already knows the tool what the shape is. Somebody meeting it for the
	// first time wants a line they can paste, and every widely used command
	// line tool leads with one.
	examplesBlock = `Examples:
  qadg generate recipe.yaml
      Produce the dataset the recipe describes and write it beside a note
      saying what the run made.

  qadg validate recipe.yaml
      Read a recipe and report every problem it has, all of them at once.`

	usageBlock = `Usage:
  qadg <command> [arguments]
  qadg --version
  qadg --license`

	// commandsNote follows the list of commands. It says the one thing somebody
	// choosing between them has to know before running either.
	commandsNote = `  A run writes nothing until it has succeeded, so a run that fails or is
  stopped leaves the output directory as it found it, and an existing file
  stops the run rather than being overwritten.`

	// commandValidate is the name as typed. It is a public contract from the
	// first release, because it goes into other people's pipelines.
	commandValidate = "validate"

	// commandGenerate is the name as typed, and a public contract from the
	// first release for the same reason as the one above it.
	commandGenerate = "generate"

	// commandHelp is the name as typed, and a public contract for the same
	// reason as the two above it.
	//
	// It exists because it was the most common thing a person types and the one
	// answer this tool did not have. Until 2026-08-05 it ended with the code for
	// a wrong invocation, while --help worked.
	commandHelp = "help"

	// The names of the two shapes this build writes. They are the values of the
	// format key of a recipe as well as of the flag, so they are one pair of
	// constants rather than two.
	formatCSV  = "csv"
	formatJSON = "json"

	// runNoteName is the file that travels beside the data.
	//
	// 🔴 Deliberately not called a manifest. That word belongs to the public
	// contract of the second pillar, and a name taken in passing is a name that
	// stays for years.
	runNoteName = "run-note.json"

	problemGenerateArguments        = "generate takes one recipe"
	problemGenerateArgumentsWhy     = "it reads a recipe and writes the dataset that recipe describes"
	problemGenerateArgumentsAllowed = "one path to a recipe file"
	problemGenerateArgumentsDo      = "qadg generate recipe.yaml"

	// The two ways a run ends because something outside it said so. Both say
	// what is on the disk afterwards, because that is the only thing the reader
	// has to act on: a directory holding half a dataset would need cleaning by
	// hand and neither of these does.
	//
	// They are worded apart on purpose. The second one is read by somebody
	// looking at a log after the fact, with no memory of pressing anything, so
	// it has to say who stopped the run.
	stoppedByUser   = "stopped before the dataset was complete. No output file was left behind and the temporary files this run made are gone"
	stoppedBySystem = "stopped by the system before the dataset was complete. No output file was left behind and the temporary files this run made are gone"

	summaryTables = " tables, "
	summaryRows   = " rows written"

	flagsHeading = "Flags:"

	// The headings of the help of a single command.
	commandsHeading    = "Commands:"
	usageHeading       = "Usage:"
	exampleHeading     = "Example:"
	flagsOfHeading     = "Flags of "
	flagsSharedHeading = "Flags of the tool, accepted before or after the command:"

	// The fourth part of a refusal that came from below the command line.
	//
	// 🔴 These exist because until 2026-08-05 every failure carrying a kind
	// reached the user as one line: what happened, and nothing about what to do.
	// The rule says four parts and the command line kept it only where a
	// refusal was written out by hand. Found by running the binary rather than
	// by reading the code, because every test was green.
	//
	// One sentence per kind rather than per message. What went wrong is already
	// in the message, and the kind is exactly what decides the next move.
	doAboutIO      = "check the path exists and that this account may write there"
	doAboutConfig  = "qadg validate <recipe> reports every problem the recipe has at once"
	doAboutData    = "the recipe is valid and cannot be satisfied, so change what it asks for"
	doAboutStopped = "nothing was left behind, so the same command can simply be run again"
	doAboutRuntime = "this is a defect in the tool rather than in the recipe. Report it with the command that caused it"

	problemHelpSubject        = "no command called "
	problemHelpSubjectWhy     = "help describes a command this build has"
	problemHelpSubjectDo      = "qadg help"
	problemHelpArguments      = "help describes one command at a time"
	problemHelpArgumentsWhy   = "each command has its own flags and its own answer"
	problemHelpArgumentsAllow = "one command name, or nothing at all"

	// channelsBlock is in the help because a script that redirects standard
	// output depends on it and has no other way to learn it.
	channelsBlock = `Channels:
  Data goes to standard output. The log goes to standard error. A run that
  fails writes nothing to standard output.`

	exitCodesHeading = "Exit codes:"

	// licenseHeadingCode and the two below label the license report.
	licenseHeadingCode  = "Code license:"
	licenseHeadingSpec  = "Specification license:"
	licenseHeadingParts = "Third party components:"

	// licenseNoComponents states the empty case as a fact. "None" alone
	// reads like something is missing.
	licenseNoComponents = "  None. This build links no module outside the Go standard library."

	// licenseHeadingData introduces the reference word lists. They ship
	// inside the binary, so this is where their notice has to be.
	licenseHeadingData = "Reference data built into this binary:"
	licenseNoData      = "  None. This build carries no reference lists."

	// versionLabelTool and the three below label the human readable version
	// report. A machine reads --json instead.
	versionLabelTool     = "qadg"
	versionLabelCatalog  = "corruption catalog"
	versionLabelManifest = "manifest format"
	versionLabelRecipe   = "recipe schema"
)

// flagDoc is one entry of the single list of flags.
//
// The help text renders this list and the failure that names what would be
// accepted reads the same list. Two lists would answer the same question
// differently the first time somebody adds a flag and updates one of them.
type flagDoc struct {
	// shown is how the flag appears in the help text.
	shown string
	// names is every spelling accepted for it.
	names []string
	// what says what the flag does, in one sentence.
	what string
}

// commandDoc is everything one verb says about itself.
//
// # Why the verbs describe themselves in one place
//
// The list of commands in the help, the help of a single command, the names the
// dispatcher accepts and the refusal that names what would have worked all
// answer the same question. Until 2026-08-05 three of them were written by
// hand, and two were wrong from the day the second verb was released. A reader
// cannot tell prose about the code from the code, so the prose has to be the
// code.
type commandDoc struct {
	// name is the verb as typed. It is a frozen contract from the first release.
	name string
	// summary is the one line in the list of commands.
	summary string
	// arguments is what follows the verb in the usage pattern, or empty.
	arguments string
	// what says what the command does and then its most important limit. Two
	// sentences at most, because a third is never read.
	what string
	// example is a line somebody can paste.
	example string
	// flags are the flags of this verb alone. The flags of the tool are shown
	// beside them and come from the list below.
	flags []flagDoc
}

// commandDocs is the register of verbs. Adding one here is what makes it
// appear in the help, in the refusals and in the near miss suggestion.
var commandDocs = []commandDoc{
	{
		name:      commandGenerate,
		summary:   "Produce the dataset a recipe describes.",
		arguments: "<recipe>",
		what: "Reads a recipe, produces the records it describes and writes them as CSV or as JSON,\n" +
			"beside a note saying what the run made. Nothing appears under its real name until the\n" +
			"whole run has succeeded, so a run that fails or is stopped leaves nothing behind.",
		example: "qadg generate --out ./data recipe.yaml",
		flags: []flagDoc{
			{"--out <dir>", []string{"--out"}, "Write the dataset here rather than where the recipe says."},
			{"--seed <text>", []string{"--seed"}, "Draw from this seed rather than the one in the recipe. The same seed gives the same data."},
			{"--format <name>", []string{"--format"}, "Write csv or json rather than the format the recipe asks for."},
			{"--overwrite", []string{"--overwrite"}, "Write over files that are already there. Without it an existing file stops the run."},
			{"--dry-run", []string{"--dry-run"}, "Say what would be produced and write nothing."},
		},
	},
	{
		name:      commandValidate,
		summary:   "Read a recipe and report every problem it has.",
		arguments: "<recipe>",
		what: "Reports every problem at once rather than stopping at the first, and lists separately\n" +
			"what it did not check. A recipe that passes says what would be generated, not that the\n" +
			"data exists.",
		example: "qadg validate --json recipe.yaml",
	},
	{
		name:      commandHelp,
		summary:   "Print this text, or the help of one command.",
		arguments: "[command]",
		what: "Answers the same as --help. A command named after it is described on its own, with the\n" +
			"flags it takes.",
		example: "qadg help generate",
	},
}

// commandDocFor returns what a verb says about itself.
func commandDocFor(name string) (commandDoc, bool) {
	for _, doc := range commandDocs {
		if doc.name == name {
			return doc, true
		}
	}
	return commandDoc{}, false
}

// reportFlagDocs are the flags that make the tool report about itself instead
// of doing anything. A command never takes them: asking for a report and for a
// command at once would leave one of the two unanswered, so both are refused.
var reportFlagDocs = []flagDoc{
	{"--version", []string{"--version"}, "Print the four version numbers this build speaks."},
	{"--license", []string{"--license"}, "Print the licenses of this tool and of everything inside it."},
}

// crossCuttingFlagDocs are the flags every command accepts, on either side of
// the verb.
var crossCuttingFlagDocs = []flagDoc{
	{"--json", []string{"--json"}, "Print machine readable output. Supported by --version, validate and generate."},
	{"--no-color", []string{"--no-color"}, "Turn color off even when the output is a terminal."},
	{"--quiet", []string{"--quiet"}, "Print failures only on the log channel."},
	{"--verbose", []string{"--verbose"}, "Print progress detail on the log channel."},
	{"-h, --help", []string{"-h", "--help"}, "Print this text."},
}

// flagDocs is every flag of the tool, for the main help and for the refusal
// that names what would have been accepted there.
//
// 🔴 The split above is not tidiness. This was one list until 2026-08-05, and
// the refusal shown for a bad flag after a verb listed --version and --license
// among the accepted ones, which that parser has never taken. A message naming
// something that does not work sends the reader to look for their own mistake.
var flagDocs = append(append([]flagDoc{}, reportFlagDocs...), crossCuttingFlagDocs...)

// renderFlags formats the flags of the tool for the help text.
func renderFlags() string { return renderFlagList(flagsHeading, flagDocs) }

// renderFlagList formats one heading and the flags under it, aligned.
func renderFlagList(heading string, docs []flagDoc) string {
	width := 0
	for _, doc := range docs {
		if len(doc.shown) > width {
			width = len(doc.shown)
		}
	}
	var b strings.Builder
	b.WriteString(heading)
	for _, doc := range docs {
		fmt.Fprintf(&b, "\n  %-*s   %s", width, doc.shown, doc.what)
	}
	return b.String()
}

// renderCommands formats the list of verbs for the main help.
//
// It is built from the register rather than written out, so that a verb cannot
// exist and go unmentioned. That is not a hypothetical: the list said no command
// generates data for as long as the command that does had been released.
func renderCommands() string {
	width := 0
	for _, doc := range commandDocs {
		if len(doc.name+" "+doc.arguments) > width {
			width = len(doc.name + " " + doc.arguments)
		}
	}
	var b strings.Builder
	b.WriteString(commandsHeading)
	for _, doc := range commandDocs {
		fmt.Fprintf(&b, "\n  %-*s   %s", width, doc.name+" "+doc.arguments, doc.summary)
	}
	b.WriteString("\n\n" + commandsNote)
	return b.String()
}

// renderCommandHelp is the help of one verb.
//
// The flags of the verb and the flags of the tool are both shown, under headings
// that say which is which. Both are accepted here, so leaving either out would
// send somebody to the source to find out what they may write.
func renderCommandHelp(doc commandDoc) string {
	blocks := []string{
		doc.summary,
		doc.what,
		usageHeading + "\n  qadg " + usageLine(doc),
		exampleHeading + "\n  " + doc.example,
	}
	if len(doc.flags) > 0 {
		blocks = append(blocks, renderFlagList(flagsOfHeading+doc.name+":", doc.flags))
	}
	return strings.Join(append(blocks, renderFlagList(flagsSharedHeading, crossCuttingFlagDocs)), "\n\n")
}

// usageLine builds the pattern of one command, leaving out the parts it has
// none of rather than printing a gap where they would go.
func usageLine(doc commandDoc) string {
	parts := []string{doc.name}
	if len(doc.flags) > 0 {
		parts = append(parts, "[flags]")
	}
	if doc.arguments != "" {
		parts = append(parts, doc.arguments)
	}
	return strings.Join(parts, " ")
}

// acceptedFlags lists every spelling, for the moment something else was given.
func acceptedFlags() string { return spellingsOf(flagDocs) }

// acceptedFlagsOf lists what one command takes, which is its own flags and the
// ones every command shares. It is not the list of what the tool takes: the
// reports are refused after a verb, so naming them here would send the reader
// after a flag that cannot work where they are standing.
func acceptedFlagsOf(command string) string {
	docs := append([]flagDoc{}, crossCuttingFlagDocs...)
	if doc, found := commandDocFor(command); found {
		docs = append(docs, doc.flags...)
	}
	return spellingsOf(docs)
}

func spellingsOf(docs []flagDoc) string {
	var names []string
	for _, doc := range docs {
		names = append(names, doc.names...)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// acceptedInvocations lists the commands as well, for the moment nothing at
// all was given and naming only the flags would hide half the answer.
//
// The commands come from the same list the dispatcher switches on. This named
// one of them by hand until 2026-08-04 and had been wrong since the day the
// second verb arrived: somebody who ran qadg with nothing was told about
// validate and never learned that generate exists.
func acceptedInvocations() string {
	return strings.Join(knownCommands(), ", ") + ", " + acceptedFlags()
}

// Failure text.
//
// Every message below carries four things: what happened, why, what would be
// accepted, and what to do instead. Three of the four are the ones people
// leave out, and a message without them sends the reader to the source code.

const (
	problemNoCommand    = "no command given"
	problemNoCommandWhy = "a command says what to do, and this run said nothing"
	problemNoCommandDo  = "qadg --help"

	// problemUnknownCommandNear replaces the plain reason when what was typed
	// is close enough to a real command to be a slip of the fingers rather
	// than a different word. Naming the near miss is the difference between a
	// message that ends the attempt and one that finishes it.
	problemUnknownCommandNear     = "this build does not define that command, and "
	problemUnknownCommandNearTail = " is one small edit away"

	problemUnknownFlagWhy = "this build does not define that flag"
	problemUnknownFlagDo  = "qadg --help lists every flag this build accepts"

	// The flag exists and was handed a value. Every flag here is a switch, so
	// there is nothing a value could mean.
	problemFlagTakesNoValueWhy     = "that flag is a switch, so it is either given or not given and takes no value"
	problemFlagTakesNoValueAllowed = "the flag on its own, such as --json"
	problemFlagTakesNoValueDo      = "remove the equals sign and whatever follows it"

	problemUnknownCommandWhy = "this build does not define that command"
	problemUnknownCommandDo  = "qadg --help"

	problemCommandAndReport        = "a command and a report were asked for at once"
	problemCommandAndReportWhy     = "answering one would leave the other unanswered"
	problemCommandAndReportAllowed = "a command, or one of --help, --license and --version"
	problemCommandAndReportDo      = "run them as two commands"

	problemQuietAndVerbose        = "--quiet and --verbose were both given"
	problemQuietAndVerboseWhy     = "one asks for less log output and the other for more"
	problemQuietAndVerboseAllowed = "one of the two, or neither"
	problemQuietAndVerboseDo      = "drop the flag you did not mean"

	problemJSONUnsupported        = "--json was given, but the requested output has no machine readable form"
	problemJSONUnsupportedWhy     = "only --version prints machine readable output, and mixing it with another report would put JSON and prose on the same channel"
	problemJSONUnsupportedAllowed = "--json with --version alone"
	problemJSONUnsupportedDo      = "drop --json to get the readable form"

	// problemCrash reports that this program stopped in a way it does not
	// understand. It is worded so that a bug report can quote it.
	problemCrash   = "internal defect: the run stopped unexpectedly"
	problemCrashDo = "report this together with the command that produced it"

	// problemMissingKind reports a defect in this program rather than a
	// mistake by the user. It is worded so that a bug report can quote it.
	problemMissingKind        = "internal defect: a failure reached the surface without a kind"
	problemMissingKindWhy     = "the exit code is a property of the failure, and this one carried none"
	problemMissingKindAllowed = "every failure inside this tool sets its kind where it happens"
	problemMissingKindDo      = "report this together with the command that produced it"

	problemRecipeIsDirectory        = "cannot read the recipe at "
	problemRecipeIsDirectoryWhy     = "that path is a directory and a recipe is a single file"
	problemRecipeIsDirectoryAllowed = "a path to one recipe file"
	problemRecipeIsDirectoryDo      = "name the recipe inside that directory, such as qadg validate recipes/customer.yaml"

	problemValidateArguments        = "validate needs exactly one recipe to read"
	problemValidateArgumentsWhy     = "it checks one file at a time, so that the report says which file it is about"
	problemValidateArgumentsAllowed = "one path to a recipe"
	problemValidateArgumentsDo      = "qadg validate recipe.yaml"
)

// Text of the validate report.
const (
	// The warning introducing what this build could not look at. It is a
	// warning rather than a quiet note, because a report that hides its own
	// blind spots invites trust it has not earned.
	//
	// It carries the count rather than the word some, and names the way to see
	// the entries when they are being suppressed. The entries print on a
	// channel --quiet silences, so without this the warning survived and its
	// whole content did not.
	problemUncheckedOne  = " declaration was not checked"
	problemUncheckedMany = " declarations were not checked"
	problemUncheckedHow  = ". Run without --quiet to see which."

	summaryValid    = " can be used. Understood "
	summaryEntities = " entities and "
	summaryFields   = " fields."
)

// uncheckedHeadline builds the warning about what was not looked at.
//
// It takes whether the detail is hidden rather than reading the level itself,
// so that the decision stays where the channels are and this only writes the
// sentence.
func uncheckedHeadline(count int, detailHidden bool) string {
	headline := strconv.Itoa(count) + problemUncheckedMany
	if count == 1 {
		headline = strconv.Itoa(count) + problemUncheckedOne
	}
	if detailHidden {
		headline += problemUncheckedHow
	}
	return headline
}

// Log prefixes. They stay stable because people grep CI logs for them.
const (
	prefixError   = "error: "
	prefixWarning = "warning: "
	prefixInfo    = "info: "
	// prefixDetail indents the three lines that follow a failure headline.
	prefixDetail = "       "
	// labelAllowed and labelTry name the last two of the four parts.
	labelAllowed = "allowed: "
	labelTry     = "try: "
)
