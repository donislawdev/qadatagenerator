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
  qadg validate recipe.yaml
      Read a recipe and report every problem it has, all of them at once.

  qadg validate --json recipe.yaml
      The same check as one JSON document, for a pipeline that decides on its
      own what is blocking.`

	usageBlock = `Usage:
  qadg <command> [arguments]
  qadg --version
  qadg --license`

	commandsBlock = `Commands:
  validate <recipe>   Read a recipe and report every problem it has.

  A recipe that passes says what will be generated. It does not say the data
  will be produced, because no command generates data in this build yet.`

	// commandValidate is the name as typed. It is a public contract from the
	// first release, because it goes into other people's pipelines.
	commandValidate = "validate"

	// commandGenerate is the name as typed, and a public contract from the
	// first release for the same reason as the one above it.
	commandGenerate = "generate"

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

	summaryTables = " tables, "
	summaryRows   = " rows written"

	flagsHeading = "Flags:"

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

var flagDocs = []flagDoc{
	{"--version", []string{"--version"}, "Print the four version numbers this build speaks."},
	{"--license", []string{"--license"}, "Print the licenses of this tool and of everything inside it."},
	{"--json", []string{"--json"}, "Print machine readable output. Supported by --version and validate."},
	{"--no-color", []string{"--no-color"}, "Turn color off even when the output is a terminal."},
	{"--quiet", []string{"--quiet"}, "Print failures only on the log channel."},
	{"--verbose", []string{"--verbose"}, "Print progress detail on the log channel."},
	{"-h, --help", []string{"-h", "--help"}, "Print this text."},
}

// renderFlags formats the list for the help text.
func renderFlags() string {
	width := 0
	for _, doc := range flagDocs {
		if len(doc.shown) > width {
			width = len(doc.shown)
		}
	}
	var b strings.Builder
	b.WriteString(flagsHeading)
	for _, doc := range flagDocs {
		fmt.Fprintf(&b, "\n  %-*s   %s", width, doc.shown, doc.what)
	}
	return b.String()
}

// acceptedFlags lists every spelling, for the moment something else was given.
func acceptedFlags() string {
	var names []string
	for _, doc := range flagDocs {
		names = append(names, doc.names...)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// acceptedInvocations lists the commands as well, for the moment nothing at
// all was given and naming only the flags would hide half the answer.
func acceptedInvocations() string {
	return commandValidate + ", " + acceptedFlags()
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
