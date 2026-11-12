package cli

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/adapter/format"
	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestexport"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// manifestFlagDoc is what --manifest does, for both verbs that write a
// manifest.
//
// 🔴 The list of forms is read from the register rather than typed here. This
// flag was accepted, refused with a list and named in two examples while being
// absent from the help of both verbs, so the one thing worth guarding is that
// the sentence describing it cannot fall behind what the build writes.
var manifestFlagDoc = flagDoc{"--manifest <form>", []string{"--manifest"},
	"Also write the manifest in this form beside the JSON one: " +
		strings.Join(manifestexport.Forms(), ", ") +
		". Without it only the JSON is written."}

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
  qadg quick pesel
      Produce one valid value and nothing else. The shortest thing this
      tool does, and the only one whose runs differ from each other.

  qadg generate recipe.yaml
      Produce the dataset the recipe describes and write it beside a note
      saying what the run made.

  qadg validate recipe.yaml
      Read a recipe and report every problem it has, all of them at once.

  qadg corrupt --layers L2,L3 contacts.csv
      Damage a file this tool did not make, and say beside it what should
      happen to every defect.`

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

	// commandCorrupt is the name as typed, and a public contract for the same
	// reason as the two above it.
	//
	// It is the verb of the second pillar working on its own: a file this tool
	// never produced goes in, and the same records with damage in them come out
	// beside a manifest saying what the damage was and what should happen to it.
	commandCorrupt = "corrupt"

	// commandVerify is the name as typed, and a public contract for the same
	// reason as the three above it.
	//
	// 🔴 It is the only verb that reports on an application rather than on its
	// own work, and the only one that returns the expectation code. That code
	// has been in the table since the first day with nothing able to return it.
	commandVerify = "verify"

	// commandQuick is the name as typed, and a public contract for the same
	// reason as the four above it.
	//
	// 🔴 It is the only verb whose default run is NOT repeatable. Somebody
	// asking for an identifier twice wants two of them, so a run without a
	// seed draws one. The seventh rule stays whole, because it says the same
	// seed gives the same result, and naming one brings that back entirely.
	// The help of this verb says so rather than leaving it to be found out.
	commandQuick = "quick"

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
	//
	// Taken from the package that implements the writers rather than spelled
	// again here. A name is how a user asks for one of them, and two copies of
	// it part company the first time a format is added.
	formatCSV  = format.NameCSV
	formatJSON = format.NameJSON

	problemQuickUnknownGenerator    = "no generator called "
	problemQuickUnknownGeneratorWhy = "the value asked for is not one this build can produce"
	problemQuickUnknownGeneratorDo  = "qadg help generators lists every one with what it produces"

	// problemQuickRoleNotRead is a prefix in two parts, because the sentence
	// names the generator and then the role, and the value of each arrives
	// with the refusal rather than being known here.
	problemQuickRoleNotRead     = "no account is taken of that value by "
	problemQuickRoleNotReadJoin = ", which was given a "
	problemQuickInLocale        = " in "
	problemQuickRoleNotReadWhy  = "one value is asked for here, so a value bound to something this generator ignores has no other column to be for. It is refused rather than accepted and forgotten"
	problemQuickRoleNotReadDo   = "leave the flag out, or ask for a value that does take account of it"

	problemQuickGender    = "not a gender this command takes: "
	problemQuickGenderWhy = "a gender may be written several ways, including the digits a database export gives, and this is not one of them"
	problemQuickGenderDo  = "qadg quick pesel --gender female"

	problemQuickDate        = "not a date this command takes: "
	problemQuickDateAllowed = "a calendar date written as YYYY-MM-DD"
	problemQuickDateWhy     = "a date of birth is read as a calendar date, and the scheme carrying it accepts one way of writing one"
	problemQuickDateDo      = "qadg quick pesel --birth-date 1980-05-05"

	// problemQuickUnavailable names the generator, and the locale follows it
	// the way it follows a role nothing reads. The locales that would do are
	// the allowed part, so the next move is written to read true even when
	// that list is empty.
	problemQuickUnavailable    = "this build cannot produce "
	problemQuickUnavailableWhy = "values of this kind are drawn from data this build carries for some locales, and it carries none for that one"
	problemQuickUnavailableDo  = "choose a locale it is produced in with --locale. qadg help generators lists the locales of every generator"

	problemQuickUnknownRefusal    = "a refusal arrived with no words for it"
	problemQuickUnknownRefusalWhy = "the mode refused this request and this surface has no sentence for that refusal, so the reason cannot be shown"

	problemQuickArguments        = "quick takes one generator"
	problemQuickArgumentsWhy     = "it produces values of the kind named, without a recipe and without writing anything"
	problemQuickArgumentsAllowed = "one generator name, such as pesel"
	problemQuickArgumentsDo      = "qadg quick pesel"

	// The shape of the number and its meaning are refused in different
	// places, and each says so in its own words. This one is about the text:
	// it never reached a number at all.
	problemQuickCountShape    = "not a number of values: "
	problemQuickCountShapeWhy = "how many values to produce is written as a whole number, in ordinary digits and with nothing around them"
	problemQuickCountShapeDo  = "qadg quick pesel --count 10"

	problemQuickCountRange    = "more values than this build can count to: "
	problemQuickCountRangeWhy = "the number given is larger than a whole number this machine holds, so it was never a number of values that could be produced"
	problemQuickCountRangeDo  = "ask for a number of values a machine could actually write"

	// A seed longer than the surfaces take. Shared by every verb with a
	// --seed flag and by the reader of a recipe, because the rule belongs to
	// the seed rather than to any one place a seed arrives from.
	//
	// The number is put in by whoever writes the message rather than spelled
	// out here, so that moving the limit moves one constant in the domain and
	// no sentence has to be edited to agree with it.
	problemSeedLong    = "a seed longer than this tool takes: "
	problemSeedLongWhy = "a seed is hashed for every value produced, and past this length that hashing stops fitting in the space set aside for it, which costs an allocation on every value rather than on every run"
	problemSeedLongDo  = "shorten the seed, or leave it out and let the run draw one"

	problemQuickCount        = "not a number of values this command takes: "
	problemQuickCountAllowed = "one or more"
	problemQuickCountWhy     = "this command answers a request for a value, and a run producing none cannot be told apart from one that failed without printing anything"
	problemQuickCountDo      = "qadg quick pesel --count 10"

	// quickSeedDrawn goes on the log channel after a run that chose its own
	// seed, because the value is on the data channel and a pipe must not see
	// this. Without it the one thing needed to ask for the same value again is
	// the one thing nobody has.
	quickSeedDrawn = "drawn from seed "

	problemGenerateArguments        = "generate takes one recipe"
	problemGenerateArgumentsWhy     = "it reads a recipe and writes the dataset that recipe describes"
	problemGenerateArgumentsAllowed = "one path to a recipe file"
	problemGenerateArgumentsDo      = "qadg generate recipe.yaml"

	problemVerifyArguments        = "verify takes a manifest and a response record"
	problemVerifyArgumentsWhy     = "it compares what a run promised about a dataset against what an application actually did with it"
	problemVerifyArgumentsAllowed = "two paths, the manifest first and the response record second"
	problemVerifyArgumentsDo      = "qadg verify expectation-manifest.json response.json"

	noteVerifyCompared = "compared "
	noteVerifyEntries  = " entries against what the application did"
	noteVerifyOneEntry = " entry against what the application did"
	noteVerifyAgreed   = "the application did what the manifest said it should, for every entry compared"
	noteVerifySoft     = " rest on rules belonging to your application rather than to the data, so they are reported and do not fail the run"
	noteVerifyLimit    = "not settled here: "

	problemCorruptArguments        = "corrupt takes one file"
	problemCorruptArgumentsWhy     = "it reads a file of records and writes those records back with damage in them"
	problemCorruptArgumentsAllowed = "one path to a comma separated file"
	problemCorruptArgumentsDo      = "qadg corrupt --layers L2,L3 records.csv"

	// The refusal for a run that named no layers.
	//
	// 🔴 There is no default on purpose. Choosing the layers is choosing what
	// damaging somebody's file means, and the cheapest default to defend, all of
	// them, takes apart the bytes and the container of a file whose owner wanted
	// to test an import. Adding a default later breaks nothing, because every
	// invocation without one is refused today.
	problemCorruptNoLayers    = "corrupt was not told which layers to damage"
	problemCorruptNoLayersWhy = "the layers decide what kind of damage this does, and there is no default because damaging the bytes of a file and damaging a value in it are different requests"
	problemCorruptNoLayersDo  = "qadg corrupt --layers L2,L3 records.csv"

	problemCorruptUnknownLayer    = "this build describes no damage in the layer "
	problemCorruptUnknownLayerWhy = "a layer with nothing described in it would order a run that produced nothing while looking as though it had"

	problemCorruptRows        = "the share of rows to damage cannot be read from "
	problemCorruptRowsWhy     = "it decides how much of the file is damaged, and a value nobody can read would leave that to a default the run never mentioned"
	problemCorruptRowsAllowed = "a percentage, such as 5% or 100%"
	problemCorruptRowsDo      = "qadg corrupt --rows 5% --layers L2,L3 records.csv"

	// 🔴 An unknown per row value used to pass. The engine reads anything that
	// is not the word for several as one defect per row, so a misspelling
	// quietly halved the damage and the run ended successfully.
	problemCorruptPerRow    = "no such setting for how many defects one row carries, "
	problemCorruptPerRowWhy = "anything else would be read as one defect per row, so a misspelling would quietly reduce the damage and the run would still succeed"
	problemCorruptPerRowDo  = "qadg corrupt --per-row any --layers L2,L3 records.csv"

	problemCorruptDelimiter        = "the delimiter has to be one character other than a quote or a line end, and this run was given "
	problemCorruptDelimiterWhy     = "it is the character that separates the fields of one line, both when this reads the file and when it writes it back, and a quote or a line end already means something there"
	problemCorruptDelimiterAllowed = "one character other than a quote or a line end, such as a comma, a semicolon or a tab"
	// The semicolon here is the VALUE of a flag rather than punctuation in a
	// sentence, and it is the one every European spreadsheet writes, so an
	// example using anything else would be an example nobody needs.
	problemCorruptDelimiterDo = "qadg corrupt --delimiter ; --layers L2,L3 records.csv" // punctuation: test data

	problemCorruptManifest    = "this build does not write a manifest in the form "
	problemCorruptManifestWhy = "the manifest is always written as JSON, and this flag asks for an extra form of it beside that one"
	problemCorruptManifestDo  = "qadg corrupt --manifest gherkin --layers L2,L3 records.csv"
	problemGenerateManifestDo = "qadg generate --manifest gherkin recipe.yaml"

	// The refusal that stands in front of the only irreversible thing this tool
	// does. It cannot fire while the output name carries its suffix, and it
	// stays because the day somebody changes how that name is built is the day
	// it is the only thing between that change and somebody losing their file.
	problemCorruptWouldOverwrite        = "this run would write over the file it is reading, "
	problemCorruptWouldOverwriteWhy     = "a file this run writes, the damaged records or the manifest, would replace the file they were made from, and nothing could bring it back"
	problemCorruptWouldOverwriteAllowed = "an output directory that is not where the source file is"
	problemCorruptWouldOverwriteDo      = "qadg corrupt --out ./damaged --layers L2,L3 records.csv"

	// problemCorruptUnnamedRefusal is what this surface says when the mode
	// refuses a run for a reason this surface has no words for.
	//
	// 🔴 It should be unreachable, and it exists because the alternative to
	// saying so is worse. Turning a refusal into words reads a register, and a
	// kind missing from it would otherwise come out as whichever entry the
	// lookup happened to land on. A message that names what it does not know
	// sends the reader somewhere useful. One that confidently names the wrong
	// cause sends them to fix something that was never broken.
	problemCorruptUnnamedRefusal    = "this build refused the run for a reason it cannot put into words, "
	problemCorruptUnnamedRefusalWhy = "the run was stopped for a real reason and this copy of the tool has no message for that reason, which is a fault in the tool rather than in what was asked of it"
	problemCorruptUnnamedRefusalDo  = "qadg corrupt --dry-run --layers L2,L3 records.csv"

	// What a run says about the guesses it had to make. They go on the log
	// channel for the person watching, and into the manifest for good.
	// 🔴 This sentence used to name the properties one by one, and it was a
	// second copy of a list that lives in the code that decides them. The day
	// relations came off that list, the copy went on saying they were not read
	// and nothing could have caught it. It now points at the list instead of
	// repeating it.
	noteNotInferred = "what this mode does not read out of a file is listed in the manifest, and damage needing any of it is recorded as not applied with the reason"
	// 🔴 Everything a run changes without being asked to belongs here rather
	// than in the manifest, where every entry is something somebody ordered.
	// The blank line half was measured while writing the test: a line with
	// nothing on it is not a record for a comma separated file, so it is
	// stepped over and never written back.
	noteRewritten     = "the output is written as UTF-8 with newline line endings and this tool's quoting, whatever the source used, and a line with nothing on it is not a record so it does not come back"
	noteByteOrderMark = "the source begins with a byte order mark and the output does not. That is a consequence of writing rather than damage that was ordered, so it is not in the manifest"
	// 🔴 The three parts of the line that ends a shortened list. The count and
	// the noun come first from the counter beside it, then what happened to
	// them, then where the whole of it is. The last part is the one that
	// differs by run: a dry run writes no manifest, so telling its reader to
	// open one would be a message whose last line contradicts its first, which
	// is a family of failure this project has already paid for.
	noteRestWorkedOut            = " worked out the same way and not named here. "
	noteRestInManifest           = "The manifest names every one."
	noteRestInManifestOfARealRun = "Run this without --dry-run and the manifest names every one."
	noteDryRunPolicy             = "the policy draws on "
	noteDryRunLayer              = "in "
	noteDryRunWrites             = "a real run would write "
	noteDryRunEveryOrder         = "A real run writes one manifest entry per ordered defect, applied or not"

	// noteFormUnused is said when an extra form of the manifest was asked for
	// and the recipe orders no damage, so there is no manifest to export. It
	// takes the form that was asked for (B-301, DEC-252).
	noteFormUnused = "--manifest %s has nothing to export, because this recipe orders no damage " +
		"and so no manifest is written. A policy section in the recipe is what makes a run write one"

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

	// The summary of a run counts its tables and its rows, and the two endings
	// keep a trial run from claiming a write that never happened. The counts
	// themselves go through counted, so that a line never reads 1 tables.
	summaryWritten = " written"
	summaryPlanned = " would be written"

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
	doAboutIO = "check the path exists and that this account may write there"
	// The data channel has no path of its own to check. It goes wherever the
	// command was sent, which is a program reading it or a file the shell
	// opened, and the two ways it stops taking writes are the program leaving
	// and the disk filling up. Until 2026-11-02 a reader that left early got
	// the sentence above, about a path nobody had named.
	doAboutOutput = "whatever the output was sent to stopped taking it, so what it received is cut short. " +
		"Check the program reading it, or the room left on the disk it is written to"
	doAboutConfig = "qadg validate <recipe> reports every problem the recipe has at once"
	// The same answer for the verb that is already that command. Telling a
	// reader who has just run validate to run validate is a fourth part that
	// costs a line and gives nothing, which is what the one above it does for
	// every other verb with a recipe.
	doAboutConfigValidate = "the message above is what this command found in the recipe. Change it there and run this again"
	doAboutData           = "the recipe is valid and cannot be satisfied, so change what it asks for"
	doAboutStopped        = "nothing was left behind, so the same command can simply be run again"
	doAboutRuntime        = "this is a defect in the tool rather than in anything you gave it. Report it with the command that caused it"

	// The same two, for the verb that has no recipe. A reader damaging a file
	// somebody sent them has nothing to validate and nothing to rewrite, so the
	// next move is in the flags they typed or in the file they were given.
	doAboutConfigFlags     = "check the flags of this command. qadg help "
	doAboutConfigFlagsTail = " lists them with what each one takes"
	// A verb reading documents it did not write needs a third answer. Sending
	// its reader to the flags would name the one thing that is not the problem,
	// which is the same failure the recipe advice made for the damaging verb.
	doAboutConfigDocument = "check that each file is the document it is meant to be. The message above names which one and what was wrong with it"
	doAboutDataCorrupt    = "the file cannot be used as it stands, so change the file or the flags describing it"

	// subjectGenerators is the one thing help describes that is not a command.
	//
	// 🔴 It is a word a person types, so it is a public contract from the first
	// release, exactly like a verb. It is not in the register of commands, and
	// that is deliberate rather than an omission: putting it there would make it
	// a verb of its own in the usage line, in the refusal for an unknown command
	// and in the parity register, and it is none of those. It is an answer to
	// the question help exists to answer.
	subjectGenerators = "generators"

	generatorsHeading        = "Generators this build can fill a field with."
	generatorsUse            = "A recipe names one like this:   pesel: { generator: pesel }"
	generatorsColumnName     = "generator"
	generatorsColumnProduces = "produces"
	generatorsYes            = "yes"
	generatorsNo             = "no"
	// generatorsOnly follows the countries a scheme of no one country covers,
	// as in "identifier (iban, PL only)".
	generatorsOnly       = " only"
	generatorsLocaleNote = "The columns are the locales this build carries word lists for. A recipe may name\n" +
		"another one, and then only the generators that need no list can be filled."
	generatorsMissingIntro = "Where a column says no, this build has nothing to draw from:"

	// 🔴 Both of these said command until 2026-09-03, and both stopped being
	// true the day help gained a subject that is not one. A refusal naming the
	// wrong kind of thing sends somebody looking for a verb that was never
	// missing.
	problemHelpSubject        = "no help for "
	problemHelpSubjectWhy     = "help describes a command this build has, or the generators it can fill a field with"
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

	// licenseHeadingCode and the three below label the license report.
	licenseHeadingAuthor = "Author:"
	licenseHeadingCode   = "Code license:"
	licenseHeadingSpec   = "Specification license:"
	licenseHeadingParts  = "Third party components:"

	// licenseNoComponents states the empty case as a fact. "None" alone
	// reads like something is missing.
	licenseNoComponents = "  None. This build links no module outside the Go standard library."

	// licenseHeadingData introduces the reference word lists. They ship
	// inside the binary, so this is where their notice has to be.
	licenseHeadingData = "Reference data built into this binary:"
	licenseNoData      = "  None. This build carries no reference lists."

	// licenseHeadingEmbedded introduces the typeface and the icon set. Both
	// travel inside the window binary, and the terms of both ask for the
	// notice to travel with them, which is why they are named here rather
	// than only in the repository.
	//
	// The heading says which binary, because this flag runs on the command
	// line one and that binary carries neither. A heading claiming otherwise
	// would be a false statement in the one place a compliance reader looks.
	licenseHeadingEmbedded = "Fonts and icons built into the window binary:"
	licenseNoEmbedded      = "  None. This build embeds no typeface or icon set."

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
			manifestFlagDoc,
			{"--overwrite", []string{"--overwrite"}, "Write over files that are already there. Without it an existing file stops the run."},
			{"--dry-run", []string{"--dry-run"}, "Say what would be produced and write nothing."},
		},
	},
	{
		name:      commandVerify,
		summary:   "Compare a manifest against what your application did.",
		arguments: "<manifest> <response>",
		what: "Reads the manifest a run wrote and a record of what your application did with that\n" +
			"data, and reports where the two disagree. It is the only command that fails because\n" +
			"of your application rather than because of itself.",
		example: "qadg verify expectation-manifest.json response.json",
		flags:   []flagDoc{},
	},
	{
		name:      commandCorrupt,
		summary:   "Damage a file this tool did not make.",
		arguments: "<file>",
		what: "Reads a comma separated file, works out what its columns hold by looking at the values,\n" +
			"and writes them back with damage in them beside a manifest saying what should happen to\n" +
			"each one. What could not be worked out is listed rather than guessed.",
		example: "qadg corrupt --layers L2,L3 --rows 5% contacts.csv",
		flags: []flagDoc{
			{"--layers <list>", []string{"--layers"}, "Which lines of defence to attack, such as L2,L3. Required, because there is no safe default."},
			{"--rows <share>", []string{"--rows"}, "The share of rows to damage, such as 5%. Defaults to 5%."},
			{"--out <dir>", []string{"--out"}, "Write the damaged file and the manifest here. Defaults to the current directory."},
			{"--seed <text>", []string{"--seed"}, "Draw from this seed. The same seed and the same file give the same damage."},
			{"--delimiter <char>", []string{"--delimiter"}, "The character between fields, for reading and for writing back. Defaults to a comma."},
			{"--per-row <how>", []string{"--per-row"}, "exactly_one or any, for how many defects one damaged row carries."},
			{"--exclude-fields <list>", []string{"--exclude-fields"}, "Columns to leave alone, whatever the layers say."},
			{"--locale <name>", []string{"--locale"}, "Which locale's damage is available. It does not decide what the columns hold, the values do."},
			manifestFlagDoc,
			{"--overwrite", []string{"--overwrite"}, "Write over files that are already there. Without it an existing file stops the run."},
			{"--dry-run", []string{"--dry-run"}, "Say what was worked out and what would be damaged, and write nothing."},
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
		name:      commandQuick,
		summary:   "Produce values, with no recipe and nothing written.",
		arguments: "<generator>",
		what: "Writes values of the kind named, one to a line, and nothing else. It is the only\n" +
			"command whose runs differ from each other: without a seed it draws a new one every\n" +
			"time, so asking twice gives different values, and naming a seed makes it repeatable.",
		example: "qadg quick pesel --count 10 --gender female",
		flags: []flagDoc{
			{"--count <number>", []string{"--count"}, "How many values to produce, one to a line. They are drawn rather than dealt, so with a short list of data behind them some may repeat."},
			{"--locale <name>", []string{"--locale"}, "Which locale to draw from. A generator that has no data for it is refused with the reason."},
			{"--seed <text>", []string{"--seed"}, "Draw from this seed. Without it a fresh one is drawn and named on the log channel."},
			{"--gender <female|male>", []string{"--gender"}, "The gender the value has to agree with. Refused for a generator that takes no account of it."},
			{"--birth-date <YYYY-MM-DD>", []string{"--birth-date"}, "The date of birth the value has to agree with. Refused for a generator that takes no account of it."},
		},
	},
	{
		name:      commandHelp,
		summary:   "Print this text, the help of one command, or what this build can generate.",
		arguments: "[command|generators]",
		what: "With nothing after it, it prints what qadg --help prints. A command named after it is\n" +
			"described on its own, with the flags it takes. The word generators lists every value\n" +
			"this build can fill a field with and says where each one can be produced.",
		example: "qadg help generators",
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
	{"--json", []string{"--json"}, "Print machine readable output. Supported by --version, --license, validate and generate. One report at a time."},
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

	// The flag exists, takes a value, and was the last thing on the line. It
	// was given the reason for a flag nobody defines until 2026-11-03, beside a
	// list of accepted flags naming the same one.
	problemFlagNeedsValueWhy     = "that flag takes a value, and nothing came after it to be one"
	problemFlagNeedsValueAllowed = "the flag with its value after it, or written as --flag=value"

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
	problemJSONUnsupportedWhy     = "--version and --license print machine readable output, and the other reports are prose"
	problemJSONUnsupportedAllowed = "--json with --version, or --json with --license"
	problemJSONUnsupportedDo      = "drop --json to get the readable form"

	// problemJSONTwoReports refuses two machine readable reports in one run.
	//
	// Both of them speak JSON, so this is not about one lacking the form. It is
	// about the channel: two documents printed one after the other are not a
	// document, and a parser reading the output stops after the first without
	// ever learning that a second was there.
	problemJSONTwoReports        = "--json was given with more than one report"
	problemJSONTwoReportsWhy     = "each report is its own document, and printing two of them on one channel gives a reader that stops after the first"
	problemJSONTwoReportsAllowed = "one report at a time: --json with --version, or --json with --license"
	problemJSONTwoReportsDo      = "run the tool twice, once for each report"

	// problemCrash reports that this program stopped in a way it does not
	// understand. It is worded so that a bug report can quote it.
	//
	// The words moved to the fault package on 2026-09-02, when the window
	// learned to survive a panic and had to name it the same way. They are
	// named here rather than reached for at each use, so that the rest of this
	// file keeps reading as one list of what this surface says.
	problemCrash   = fault.Crashed
	problemCrashDo = fault.CrashedDo

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

	summaryValid = " can be used. Understood "
	// The answer for a recipe that reads correctly and that the engine will
	// not run.
	//
	// It does not say can be used, and the difference is the whole sentence. A
	// recipe with nothing wrong in it whose run stops on its first phase is
	// neither of the two things this verb used to say, and answering with
	// either of them would be a true half beside a false one. The warning
	// above it carries what the engine said, so this is only the verdict.
	summaryValidUnrunnable = " reads correctly and a run of it would not be carried out. " +
		"Run qadg generate --dry-run to have that answered with an exit code."
	// What a run of a usable recipe would write.
	//
	// In the words the dry run already uses, because two sentences about one
	// number are how somebody comes to believe there are two numbers.
	summaryWouldWrite = "A run of it would write "
	// The two counts each carry both of their forms, the way the warning above
	// them does. What made this a pair rather than one word with a letter added
	// is written at understood below.
	summaryEntityOne  = " entity and "
	summaryEntityMany = " entities and "
	summaryFieldOne   = " field."
	summaryFieldMany  = " fields."
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

// understood says how much of the recipe was read, in the singular where there
// is one of it.
//
// # Why a pair of words rather than a letter added to one
//
// The window says this same sentence and builds its plural by adding an s,
// which is English and only English, and it says so in a comment because that
// is all a comment can do. Here the words were already two constants, the way
// the warning above them is, so each form has a place of its own to go the day
// a catalogue carries this line. That is the cheap half of B-184 rather than
// the whole of it: what is still missing is a language whose plural is not one
// more form but three.
//
// # Why this was worth a repair at all
//
// The two surfaces answered the same question in different words. The window
// read 1 entity, 1 field and the command line read 1 entities and 1 fields
// about the same recipe, which is the sixth rule broken by one line rather
// than by a missing function.
func understood(entities, fields int) string {
	sentence := strconv.Itoa(entities)
	if entities == 1 {
		sentence += summaryEntityOne
	} else {
		sentence += summaryEntityMany
	}
	sentence += strconv.Itoa(fields)
	if fields == 1 {
		return sentence + summaryFieldOne
	}
	return sentence + summaryFieldMany
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
