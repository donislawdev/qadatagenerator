// Package i18n answers what the window should say, by key rather than by
// literal.
//
// # Why the window never holds its own text
//
// The command line is English and stays English, because a flag and an exit
// code are a contract with somebody's pipeline. The window is translated, and
// the rule that makes translation possible is that no screen holds a sentence:
// a new language has to be a new file beside the catalogue and no change in Go
// at all. A literal in a view is the one thing that makes that impossible,
// because it is invisible to whoever adds a language.
//
// # Why the keys are constants and not bare strings
//
// A key typed as a string turns a typo into a label that renders as its own
// key, and nobody notices until somebody reads the screen. Declared as
// constants, a typo does not compile. A test closes the constants against the
// catalogue in both directions, so a key with no entry and an entry no screen
// asks for are both failures.
package i18n

import (
	_ "embed"
	"fmt"
	"sort"
	"strconv"

	"github.com/goccy/go-yaml"
)

// Key is the name a screen asks for text by.
type Key string

// The keys the window itself carries, rather than any one screen.
//
// The title stopped belonging to the About screen on 2026-08-17, when a second
// screen arrived and the window stopped being that one screen with a frame
// around it.
const (
	WindowTitle Key = "window.title"

	// The words the toolkit draws itself: the dialog a file is chosen in, the
	// menu a text field opens under the right button, the dialog an error is
	// shown in, and the words a chooser holds before anything is chosen.
	// Without these the toolkit takes the language of the SYSTEM,
	// so on a Polish desktop the file dialog spoke Polish in the middle of an
	// English interface (B-290). fynetheme.Speak hands them over.
	ToolkitOpen            Key = "toolkit.open"
	ToolkitSave            Key = "toolkit.save"
	ToolkitCancel          Key = "toolkit.cancel"
	ToolkitFile            Key = "toolkit.file"
	ToolkitFolder          Key = "toolkit.folder"
	ToolkitNewFolder       Key = "toolkit.new-folder"
	ToolkitCreateFolder    Key = "toolkit.create-folder"
	ToolkitName            Key = "toolkit.name"
	ToolkitParent          Key = "toolkit.parent"
	ToolkitShowHiddenFiles Key = "toolkit.show-hidden-files"
	ToolkitEnterFilename   Key = "toolkit.enter-filename"
	ToolkitOK              Key = "toolkit.ok"
	ToolkitError           Key = "toolkit.error"
	ToolkitYes             Key = "toolkit.yes"
	ToolkitNo              Key = "toolkit.no"
	ToolkitCut             Key = "toolkit.cut"
	ToolkitCopy            Key = "toolkit.copy"
	ToolkitPaste           Key = "toolkit.paste"
	ToolkitSelectAll       Key = "toolkit.select-all"
	ToolkitUndo            Key = "toolkit.undo"
	ToolkitRedo            Key = "toolkit.redo"
	ToolkitQuit            Key = "toolkit.quit"
	ToolkitSelectOne       Key = "toolkit.select-one"

	ScreenAbout    Key = "screen.about"
	ScreenManifest Key = "screen.manifest"
	ScreenTake     Key = "screen.take"
	ScreenRecipe   Key = "screen.recipe"
	ScreenQuick    Key = "screen.quick"

	// The headings the column groups its names under, added on 2026-09-08 with
	// DEC-225. They say what a person came to do rather than what the screens
	// are, which is the whole point of the order they now sit in.
	// The promise under the product's name at the foot of the column, added
	// 2026-09-09. It is the one sentence about this tool that is true before
	// anybody presses anything, and the accepted drawing puts it where a person
	// looks while deciding whether to trust the window at all.
	SidebarOffline Key = "sidebar.offline"

	SidebarGroupMake  Key = "sidebar.group.make"
	SidebarGroupCheck Key = "sidebar.group.check"
)

// The keys of the About screen.
const (
	AboutHeadingVersions   Key = "about.heading.versions"
	AboutHeadingLicenses   Key = "about.heading.licenses"
	AboutHeadingComponents Key = "about.heading.components"
	AboutHeadingData       Key = "about.heading.data"
	AboutHeadingNotices    Key = "about.heading.notices"

	AboutVersionTool     Key = "about.version.tool"
	AboutVersionCatalog  Key = "about.version.catalog"
	AboutVersionManifest Key = "about.version.manifest"
	AboutVersionRecipe   Key = "about.version.recipe"

	// The author stands first among the terms, because it is who grants them.
	AboutLicenseAuthor Key = "about.license.author"
	AboutLicenseCode   Key = "about.license.code"
	AboutLicenseSpec   Key = "about.license.spec"

	// The heads of the columns of the three lists on the screen: what it is,
	// which version, under which license, and the day that license was read.
	// A head over a column rather than a label in front of every value, which
	// turned thirty seven modules into a wall nobody could scan (DEC-250). The
	// first column is named by what the list holds.
	AboutColumnModule  Key = "about.column.module"
	AboutColumnSource  Key = "about.column.source"
	AboutColumnAsset   Key = "about.column.asset"
	AboutColumnVersion Key = "about.column.version"
	AboutColumnLicense Key = "about.column.license"
	AboutColumnChecked Key = "about.column.checked"

	// How many of each thing a list holds, beside its heading, in the two
	// forms a number asks of an English noun (see QuickValuesCounted).
	AboutCountModule  Key = "about.count.module"
	AboutCountModules Key = "about.count.modules"
	AboutCountList    Key = "about.count.list"
	AboutCountLists   Key = "about.count.lists"
	AboutCountSource  Key = "about.count.source"
	AboutCountSources Key = "about.count.sources"
	AboutCountAsset   Key = "about.count.asset"
	AboutCountAssets  Key = "about.count.assets"

	AboutEmptyComponents Key = "about.empty.components"
	AboutEmptyData       Key = "about.empty.data"

	// The typeface and the icon set, which are neither code nor reference
	// values and were named nowhere until 2026-09-08.
	AboutHeadingEmbedded Key = "about.heading.embedded"
	AboutEmptyEmbedded   Key = "about.empty.embedded"
	// AboutSaveNotices offers the licence texts of those as a file.
	//
	// A button rather than the text on the screen, and the reason is what
	// the terms actually ask for: the notice has to travel with the binary
	// and be available to whoever received it. Several kilobytes of licence
	// laid into a column would bury the four facts above it, and a person
	// who wants the text wants it as a file they can keep.
	AboutSaveNotices Key = "about.save.notices"
	// AboutNoticesHeader is the first line of that file, so a file found on
	// a disk a year later still says what it belongs to.
	AboutNoticesHeader Key = "about.notices.header"
)

// The keys of the manifest browser.
//
// # Why the vocabularies are keys and the values beside them are not
//
// A mutator identifier, a column name and a seed are data. They read the same
// in every language and they are what somebody pastes into a report, so
// translating them would change what the screen is showing. An expectation, a
// confidence and a reason damage could not be applied are the opposite: three,
// two and six closed sets, each one a phrase this project writes for a person
// to read. Registers below turn each of those sets into keys, and a test closes
// them against the domain in both directions, so a seventh reason cannot reach
// a screen as an empty space.
const (
	ManifestOpen         Key = "manifest.open"
	ManifestOpenResponse Key = "manifest.open-response"
	ManifestEmpty        Key = "manifest.empty"
	ManifestUnreadable   Key = "manifest.unreadable"
	ManifestReading      Key = "manifest.reading"

	ManifestHeadingRun     Key = "manifest.heading.run"
	ManifestHeadingEntries Key = "manifest.heading.entries"
	ManifestHeadingLimits  Key = "manifest.heading.limits"
	// The card under the table says one thing at a time, an entry or the run,
	// and its heading says which. The first is the heading of an entry, the
	// second the button that brings the entry back in place of the run.
	ManifestHeadingEntry Key = "manifest.heading.entry"
	ManifestShowEntry    Key = "manifest.show-entry"

	// The sentences of the state before a manifest is open. They name what each
	// part of the screen will hold, which is the question somebody meeting this
	// screen has and the one an empty screen answers with silence.
	ManifestRunWait     Key = "manifest.run.wait"
	ManifestEntriesWait Key = "manifest.entries.wait"
	// Why the second button cannot be pressed yet. A control that is dead
	// without saying why reads as a fault in the program.
	ManifestResponseWait Key = "manifest.response.wait"

	// What each column of the entries table holds, said once in the state that
	// has no entries to say it with. A heading on its own is a word, and
	// "Outcome" is not a word somebody meeting this tool can work out.
	ManifestWaitMutator Key = "manifest.wait.mutator"
	ManifestWaitPlace   Key = "manifest.wait.place"
	ManifestWaitExpect  Key = "manifest.wait.expect"
	ManifestWaitStatus  Key = "manifest.wait.status"

	// The few words beside the two buttons of the band: what is open, how
	// much of it there is, and what is being done to it. Data stands between
	// them (a file name, a seed, a count), so each is a word rather than a
	// sentence.
	ManifestFactsEntry      Key = "manifest.facts.entry"
	ManifestFactsEntries    Key = "manifest.facts.entries"
	ManifestFactsSeed       Key = "manifest.facts.seed"
	ManifestFactsOf         Key = "manifest.facts.of"
	ManifestFactsAnswered   Key = "manifest.facts.answered"
	ManifestFactsReading    Key = "manifest.facts.reading"
	ManifestFactsUnreadable Key = "manifest.facts.unreadable"
	// What a comparison says about the response file as a whole, beside the
	// file rather than in a strip over the table (DEC-251): after a count, how
	// many records it never answered, or alone, that it disagreed with nothing.
	ManifestFactsNeverAnswered Key = "manifest.facts.never-answered"
	ManifestFactsAgreed        Key = "manifest.facts.agreed"

	// The tally at the heading of the table: how much of what was ordered
	// happened, or once a response is open, how much of it could be judged
	// and with what result. Words after a number, so they are separate from
	// the labels of the envelope, which stand before one.
	ManifestTallyOrdered    Key = "manifest.tally.ordered"
	ManifestTallyApplied    Key = "manifest.tally.applied"
	ManifestTallyNotApplied Key = "manifest.tally.not-applied"
	ManifestTallyCompared   Key = "manifest.tally.compared"
	ManifestTallyDisagree   Key = "manifest.tally.disagree"
	ManifestTallyNotSettled Key = "manifest.tally.not-settled"

	// The card under the table, which reads out one entry in full: what to do
	// before there is a file, what to do before an entry is chosen, and the
	// sentences an entry is read out with. The certainty is a sentence here
	// and a mark in the table, because the card has the room the cell has not.
	ManifestDetailWait       Key = "manifest.detail.wait"
	ManifestDetailSelect     Key = "manifest.detail.select"
	ManifestDetailEntry      Key = "manifest.detail.entry"
	ManifestDetailHard       Key = "manifest.detail.hard"
	ManifestDetailSoft       Key = "manifest.detail.soft"
	ManifestDetailNotApplied Key = "manifest.detail.not-applied"
	ManifestDetailBecause    Key = "manifest.detail.because"

	// The line that does the same in a terminal, with the response record as
	// an example until one is open.
	ManifestCommandResponse Key = "manifest.command.response"
	ManifestCommandExample  Key = "manifest.command.example"

	ManifestRunGenerated   Key = "manifest.run.generated"
	ManifestRunSeed        Key = "manifest.run.seed"
	ManifestRunLocale      Key = "manifest.run.locale"
	ManifestRunCatalog     Key = "manifest.run.catalog"
	ManifestRunSpec        Key = "manifest.run.spec"
	ManifestRunRecipe      Key = "manifest.run.recipe"
	ManifestRunPolicy      Key = "manifest.run.policy"
	ManifestRunTag         Key = "manifest.run.tag"
	ManifestRunSource      Key = "manifest.run.source"
	ManifestRunRowsTotal   Key = "manifest.run.rows-total"
	ManifestRunRowsMutated Key = "manifest.run.rows-mutated"

	ManifestCountRequested  Key = "manifest.count.requested"
	ManifestCountApplied    Key = "manifest.count.applied"
	ManifestCountNotApplied Key = "manifest.count.not-applied"

	ManifestEntryApplied    Key = "manifest.entry.applied"
	ManifestEntryNotApplied Key = "manifest.entry.not-applied"
	ManifestEntryBefore     Key = "manifest.entry.before"
	ManifestEntryAfter      Key = "manifest.entry.after"

	// What the outcome column says once a response record is open: the short
	// verdict of the application on the row, or that it gave none, or that
	// the comparison could not settle it. Short, because the column is 148
	// wide and the sentence stands in the card under the table.
	ManifestAnswerAccepted   Key = "manifest.answer.accepted"
	ManifestAnswerRejected   Key = "manifest.answer.rejected"
	ManifestAnswerNone       Key = "manifest.answer.none"
	ManifestAnswerNotSettled Key = "manifest.answer.not-settled"

	ManifestColumnMutator Key = "manifest.column.mutator"
	ManifestColumnPlace   Key = "manifest.column.place"
	ManifestColumnExpect  Key = "manifest.column.expect"
	ManifestColumnStatus  Key = "manifest.column.status"

	ManifestPlaceWholeFile Key = "manifest.place.whole-file"
	ManifestPlaceRow       Key = "manifest.place.row"
	ManifestPlaceByte      Key = "manifest.place.byte"

	ManifestExpectationRejection  Key = "manifest.expectation.rejection"
	ManifestExpectationAcceptance Key = "manifest.expectation.acceptance"
	ManifestExpectationWarning    Key = "manifest.expectation.warning"

	ManifestConfidenceHard Key = "manifest.confidence.hard"
	ManifestConfidenceSoft Key = "manifest.confidence.soft"
	// The mark a soft expectation carries in front of it in the table, and the
	// one line explaining it at the heading of the table before any file is
	// open. Once entries are on the screen the card under them explains the
	// certainty of each in a sentence.
	ManifestConfidenceMark Key = "manifest.confidence.mark"
	ManifestLegendMark     Key = "manifest.legend.mark"

	ManifestFailureUndescribed         Key = "manifest.failure.undescribed"
	ManifestFailureUnimplemented       Key = "manifest.failure.unimplemented"
	ManifestFailureRunLacksCapability  Key = "manifest.failure.run-lacks-capability"
	ManifestFailureFieldTypeNotCovered Key = "manifest.failure.field-type-not-covered"
	ManifestFailureSchemeNotCovered    Key = "manifest.failure.scheme-not-covered"
	ManifestFailureDeclinedByMutator   Key = "manifest.failure.declined-by-mutator"

	VerifyNoAnswer        Key = "verify.no-answer"
	VerifyVerdictAccepted Key = "verify.verdict.accepted"
	VerifyVerdictRejected Key = "verify.verdict.rejected"

	VerifyKindHole        Key = "verify.kind.hole"
	VerifyKindOverzealous Key = "verify.kind.overzealous"
	VerifyKindSilent      Key = "verify.kind.silent"
	VerifyKindMissing     Key = "verify.kind.missing"
)

// The keys of the recipe editor.
//
// A problem is drawn from these in the same order the command line writes one,
// because somebody who reads one report and then the other is reading about the
// same recipe and should not have to translate between them.
const (
	RecipeEmpty       Key = "recipe.empty"
	RecipePlaceholder Key = "recipe.placeholder"
	RecipeChecking    Key = "recipe.checking"
	RecipeName        Key = "recipe.name"

	// What this screen says to somebody who has never written a recipe: the
	// offer of one that runs, and the vocabulary this build carries.
	RecipeExample           Key = "recipe.example"
	RecipeCanGenerate       Key = "recipe.can-generate"
	RecipeGenerator         Key = "recipe.generator"
	RecipeGenerators        Key = "recipe.generators"
	RecipeNotIn             Key = "recipe.not-in"
	RecipeNothingToDrawFrom Key = "recipe.nothing-to-draw-from"

	// The two columns and what the card over the right one is called: what
	// the recipe describes while somebody writes, and what the last run did
	// once they have pressed the button. One card, one heading saying which
	// (DEC-246), the way the take screen names its card.
	RecipeHeadingRecipe    Key = "recipe.heading.recipe"
	RecipeHeadingDescribes Key = "recipe.heading.describes"
	RecipeHeadingResult    Key = "recipe.heading.result"
	RecipeHeadingProblems  Key = "recipe.heading.problems"
	// RecipeStepGenerate heads the second of the two numbered steps the
	// screen reads in since DEC-261. The first is RecipeHeadingRecipe.
	RecipeStepGenerate Key = "recipe.step.generate"

	RecipeNoObjections Key = "recipe.no-objections"
	RecipeLine         Key = "recipe.line"
	RecipeAllowed      Key = "recipe.allowed"
	RecipeTry          Key = "recipe.try"
	RecipeUnchecked    Key = "recipe.unchecked"
	// 🔴 Both forms of each, rather than one and a letter. The letter is what
	// the window drew until 2026-09-02, and rendering read it back: four fields
	// of two entities came out as "2 entitys". English alone is enough to
	// break that rule, so the sixth language is not what this is waiting for.
	RecipeEntity   Key = "recipe.entity"
	RecipeEntities Key = "recipe.entities"
	RecipeField    Key = "recipe.field"
	RecipeFields   Key = "recipe.fields"

	// The preview. A table says how much of itself is on screen, and the three
	// sentences below it are the states a table can be in that are not a table.
	RecipeRows          Key = "recipe.rows"
	RecipeShowing       Key = "recipe.showing"
	RecipeOf            Key = "recipe.of"
	RecipeRow           Key = "recipe.row"
	RecipeNoRows        Key = "recipe.no-rows"
	RecipeNoFields      Key = "recipe.no-fields"
	RecipeAsText        Key = "recipe.as-text"
	RecipeParentOutside Key = "recipe.parent-outside"
	RecipeRefused       Key = "recipe.refused"
	RecipeCrashed       Key = "recipe.crashed"
	RecipeWillNotDo     Key = "recipe.will-not-do"

	// The band along the top, which is the half of this screen that writes.
	//
	// 🔴 The reasons a run cannot start are not keys of their own any more
	// (DEC-246). Until 2026-09-15 three sentences stood under the button. Now
	// the card carries the reason where it arises: the empty screen says what
	// to do, a problem is its own reason, and the one reason left is the
	// directory, said under the files a run would write.
	RecipeWriteTo   Key = "recipe.write-to"
	RecipeChooseDir Key = "recipe.choose-dir"
	RecipeReplace   Key = "recipe.replace"
	RecipeGenerate  Key = "recipe.generate"
	// The two keys of the terminal line at the foot. The file name is an
	// example rather than a path, because this window does not write the
	// recipe file.
	RecipeCommandFile    Key = "recipe.command.file"
	RecipeCommandExample Key = "recipe.command.example"
	RecipeStopRun        Key = "recipe.stop-run"
	// What the card says about a run: the files it would write before one, and
	// what it left behind after one.
	RecipePlanWrites    Key = "recipe.plan.writes"
	RecipeNeedDirectory Key = "recipe.need-directory"
	RecipeDirHint       Key = "recipe.dir-hint"
	RecipeWriting       Key = "recipe.writing"
	RecipeStopping      Key = "recipe.stopping"
	RecipeResultWritten Key = "recipe.result.written"
	RecipeResultRows    Key = "recipe.result.rows"
	RecipeIn            Key = "recipe.in"
	RecipeRunStopped    Key = "recipe.run-stopped"
	RecipeRunFailed     Key = "recipe.run-failed"
	// What the card says about the damage a recipe orders (DEC-252): that the
	// rows of the sample are the rows before it, and that an extra form of the
	// manifest has nothing to export when the recipe orders none.
	RecipeSampleBeforeDamage Key = "recipe.sample-before-damage"
	RecipeFormUnused         Key = "recipe.form-unused"
)

// The keys of the Quick value screen.
const (
	QuickNothingWritten Key = "quick.nothing-written"
	QuickLabelValue     Key = "quick.label.value"
	// QuickValueHeading names the part of the screen the value appears in,
	// which is not the same thing as the label on the list that chooses which
	// KIND of value. Both read "Value" until 2026-09-08, on one screen, and
	// the accepted drawing names the list "Kind" for exactly that reason.
	QuickValueHeading Key = "quick.heading.value"
	// QuickValuesCounted is the same heading once there is a NUMBER in front
	// of it, which is a different word and not a different sentence.
	//
	// 🔴 It exists because the screen said "10 Value" on the day it learned to
	// draw a list, and that is the shape of fault a catalogue of single words
	// invites: the code joined a number to a noun nobody had told it to
	// pluralise. Keeping the two forms as two keys is what lets a translator
	// answer for a language where the number changes the noun, and a language
	// where the number governs several forms needs a count aware catalogue
	// rather than a third key here.
	QuickValuesCounted Key = "quick.heading.values-counted"

	// What the panel beside the values says about the KIND that produced
	// them. Every one of these is rendered from a fact the mode derived from
	// what a provider declares, so none of them is a sentence somebody wrote
	// about a scheme and nothing checks.
	QuickAboutProduces   Key = "quick.about.produces"
	QuickAboutCheckDigit Key = "quick.about.check-digit"
	QuickAboutRange      Key = "quick.about.range"
	// QuickAboutFormat heads the written form of one value, and the two
	// after it are what the places in that form may hold.
	//
	// 🔴 They are words for a COUNT the mode works out, not sentences about
	// a scheme. The window puts the number in front of one of them, which is
	// the same shape as the count over the card of values, and it is what
	// keeps a claim about a scheme out of a translation file where nothing
	// could check it.
	QuickAboutFormat          Key = "quick.about.format"
	QuickAboutDigits          Key = "quick.about.digits"
	QuickAboutLettersOrDigits Key = "quick.about.letters-or-digits"
	// QuickAboutOr joins the counts of a scheme with more than one form, as
	// in "15 or 16 digits". A word rather than a sentence, for the reason
	// the two above give.
	QuickAboutOr Key = "quick.about.or"
	// 🔴 The four sentences below are the only place this window tells
	// somebody whether a value it just produced could belong to a real
	// person, and both halves of that are said out loud.
	//
	// A scheme with a reserved block gets the reassuring one. A scheme
	// without one gets a sentence saying so plainly rather than getting
	// nothing, because a missing line reads as reassurance and rule nine of
	// the handbook is about exactly this. Two schemes in this build narrow
	// what they generate WITHOUT claiming a block, which is why the wording
	// says what is set aside rather than what is generated.
	QuickAboutCheckDigitCarried Key = "quick.about.check-digit.carried"
	QuickAboutCheckDigitNone    Key = "quick.about.check-digit.none"
	QuickAboutRangeReserved     Key = "quick.about.range.reserved"
	QuickAboutRangeNone         Key = "quick.about.range.none"
	// QuickAboutHarbour is the badge on a kind whose values come from a block
	// nobody is ever issued.
	QuickAboutHarbour Key = "quick.about.harbour"
	// QuickAboutUnavailable heads the reason this build cannot produce the
	// chosen kind in the chosen locale.
	QuickAboutUnavailable Key = "quick.about.unavailable"
	// What stands where the value will be, before there is one. The drawing
	// this screen was built from only ever shows a value, so the state this
	// screen opens in was never drawn at all, and it opened as an empty box
	// with a border round it.
	QuickLabelLocale Key = "quick.label.locale"
	QuickLabelSeed   Key = "quick.label.seed"
	QuickLabelGender Key = "quick.label.gender"
	QuickLabelBorn   Key = "quick.label.born"
	QuickGenderAny   Key = "quick.gender.any"
	// QuickCopy is what a row of values says under the pointer, QuickCopyAll
	// the button over them, and QuickCopied what either says for a moment once
	// it has filled the clipboard (2026-10-21, in place of a sentence that stood
	// beside the button until the next copy).
	QuickCopy    Key = "quick.copy"
	QuickCopyAll Key = "quick.copy-all"
	QuickAgain   Key = "quick.again"
	QuickCopied  Key = "quick.copied"
	// QuickGenerate names the one action this screen is for.
	//
	// It exists because a field somebody TYPES into stopped producing on every
	// keystroke (DEC-235), so there has to be something that says when what was
	// typed should take effect. Choosing from a list still produces at once.
	QuickGenerate Key = "quick.generate"
	// QuickRefusedCountShape is text asked for as a number that never became
	// one, and QuickRefusedCountMany is more values than this SCREEN shows.
	//
	// 🔴 The second one is a limit of the window and of nothing else, so it
	// names the line at the foot of the screen, which carries no limit at all.
	// A screen that refuses without saying where the thing it refused can be
	// had is a screen that made somebody's problem its own.
	QuickRefusedCountShape Key = "quick.refused.count-shape"
	QuickRefusedCountMany  Key = "quick.refused.count-many"
	// QuickLabelCount names the list that chooses how many values to produce,
	// and QuickMayRepeat is the limit that goes with it. A person asking for
	// fifty of something drawn from a short list of data will see repeats, and
	// this screen says so rather than letting them be found out. The engine
	// makes no promise of uniqueness, because that is a promise a recipe makes
	// with a rule of its own.
	QuickLabelCount Key = "quick.label.count"
	QuickMayRepeat  Key = "quick.may-repeat"
	// QuickRefusedCount is a refusal this screen cannot reach, because the
	// number comes from a list. It is worded for the reason the other two
	// unreachable ones are: the register is closed in both directions.
	QuickRefusedCount Key = "quick.refused.count"

	// The refusals this mode can produce, in the words of this surface.
	//
	// Two of the four cannot be reached from this screen, because a gender is
	// chosen from a list and a field that is not read is not shown. They are
	// worded anyway: the register is closed in both directions by a test, and
	// a screen that met an unworded refusal would have nothing to say.
	QuickRefusedGenerator Key = "quick.refused.generator"
	QuickRefusedRole      Key = "quick.refused.role"
	QuickRefusedGender    Key = "quick.refused.gender"
	QuickRefusedDate      Key = "quick.refused.date"

	// The panel a kind is picked from, which groups the kinds under headings
	// and filters them as somebody types (DEC-244).
	//
	// The headings follow quick.Group: one for each kind of shelf the mode
	// names, and one per country a scheme belongs to. A country is worded
	// here rather than left as its code because a person reads a list of
	// headings, and PL beside United States is a code beside a name. The
	// guard beside the quick screen holds that every country a scheme
	// carries has a name here, so a scheme for a new country cannot arrive
	// without one.
	QuickGroupInternational Key = "quick.group.international"
	QuickGroupPeople        Key = "quick.group.people"
	QuickGroupPlaces        Key = "quick.group.places"
	CountryPL               Key = "country.PL"
	CountryUS               Key = "country.US"
	// QuickPickPrompt is what the empty search field says. QuickPickKinds is
	// the noun after the count in the foot of the panel, and QuickPickMatches
	// replaces both when a filter is on, with the two counts in order:
	// matching, then all. QuickPickNothing stands where the rows were when
	// nothing matches, because a list that quietly goes blank reads as a
	// list that broke.
	QuickPickPrompt  Key = "quick.pick.prompt"
	QuickPickKinds   Key = "quick.pick.kinds"
	QuickPickMatches Key = "quick.pick.matches"
	QuickPickEnter   Key = "quick.pick.enter"
	QuickPickNothing Key = "quick.pick.nothing"
	// What a person calls each kind, standing beside the code a recipe uses in
	// the panel a kind is picked from, and found by the search there (DEC-262,
	// 2026-10-21). One per generator of the build. The key is built from the
	// generator when the panel is drawn, so the guard beside the quick screen
	// walks every generator and holds that each has its name here.
	KindAba        Key = "kind.aba"
	KindBirthDate  Key = "kind.birth_date"
	KindCard       Key = "kind.card"
	KindCity       Key = "kind.city"
	KindEan        Key = "kind.ean"
	KindFirstName  Key = "kind.first_name"
	KindGender     Key = "kind.gender"
	KindIban       Key = "kind.iban"
	KindImei       Key = "kind.imei"
	KindIsbn       Key = "kind.isbn"
	KindLastName   Key = "kind.last_name"
	KindNip        Key = "kind.nip"
	KindPesel      Key = "kind.pesel"
	KindPostalCode Key = "kind.postal_code"
	KindRegion     Key = "kind.region"
	KindRegon      Key = "kind.regon"
	KindSsn        Key = "kind.ssn"
	KindVin        Key = "kind.vin"

	// The line a person can paste into a terminal to get the same thing.
	// docs/08 asks for this on every screen that produces something, and says
	// why: without it the window and the terminal stop being one product.
	CommandHeading Key = "command.heading"
	CommandCopy    Key = "command.copy"
	CommandCopied  Key = "command.copied"

	// One line at the top of each screen saying what it is for. The column
	// gives a screen its name, and a name is not a purpose: somebody opening
	// this window for the first time read "Take and corrupt", the name of that
	// screen until 2026-10-21, and learned nothing about what it would do to
	// their file.
	PurposeTake     Key = "purpose.take"
	PurposeRecipe   Key = "purpose.recipe"
	PurposeManifest Key = "purpose.manifest"
	PurposeAbout    Key = "purpose.about"

	// The machine readable form of what this screen says. DEC-30 names the
	// reader and it is a compliance scanner, which reads documents rather than
	// paragraphs, so the window has to PRODUCE the file rather than show it.
	AboutSaveJSON Key = "about.save.json"
)

// windowKeys is every key the screens of the window may ask for. Keys, beside
// the keys of the catalogue of controls, is what the test closes.
var windowKeys = []Key{
	ToolkitOpen, ToolkitSave, ToolkitCancel, ToolkitFile, ToolkitFolder, ToolkitNewFolder,
	ToolkitCreateFolder, ToolkitName, ToolkitParent, ToolkitShowHiddenFiles, ToolkitEnterFilename,
	ToolkitOK, ToolkitError, ToolkitYes, ToolkitNo, ToolkitCut, ToolkitCopy, ToolkitPaste,
	ToolkitSelectAll, ToolkitUndo, ToolkitRedo, ToolkitQuit, ToolkitSelectOne,
	WindowTitle, ScreenAbout, ScreenManifest, ScreenTake, ScreenRecipe, ScreenQuick,
	SidebarGroupMake, SidebarGroupCheck, SidebarOffline,

	QuickNothingWritten, QuickLabelValue, QuickValueHeading, QuickValuesCounted,
	QuickAboutProduces, QuickAboutCheckDigit, QuickAboutRange,
	QuickAboutFormat, QuickAboutDigits, QuickAboutLettersOrDigits, QuickAboutOr,
	QuickAboutCheckDigitCarried, QuickAboutCheckDigitNone,
	QuickAboutRangeReserved, QuickAboutRangeNone,
	QuickAboutHarbour, QuickAboutUnavailable,
	QuickLabelGender, QuickLabelBorn,
	QuickLabelLocale, QuickLabelSeed,
	QuickGenderAny, QuickCopy, QuickCopyAll, QuickAgain, QuickCopied, QuickGenerate,
	QuickLabelCount, QuickMayRepeat,
	QuickRefusedGenerator, QuickRefusedRole, QuickRefusedGender, QuickRefusedDate,
	QuickRefusedCount, QuickRefusedCountShape, QuickRefusedCountMany,
	QuickGroupInternational, QuickGroupPeople, QuickGroupPlaces, CountryPL, CountryUS,
	QuickPickPrompt, QuickPickKinds, QuickPickMatches, QuickPickEnter, QuickPickNothing,
	KindAba, KindBirthDate, KindCard, KindCity, KindEan, KindFirstName, KindGender, KindIban, KindImei,
	KindIsbn, KindLastName, KindNip, KindPesel, KindPostalCode, KindRegion, KindRegon, KindSsn, KindVin,
	CommandHeading, CommandCopy, CommandCopied,
	PurposeTake, PurposeRecipe, PurposeManifest, PurposeAbout, AboutSaveJSON,

	AboutHeadingVersions, AboutHeadingLicenses, AboutHeadingComponents,
	AboutHeadingData, AboutHeadingNotices,
	AboutVersionTool, AboutVersionCatalog, AboutVersionManifest, AboutVersionRecipe,
	AboutLicenseAuthor, AboutLicenseCode, AboutLicenseSpec,
	AboutColumnModule, AboutColumnSource, AboutColumnAsset,
	AboutColumnVersion, AboutColumnLicense, AboutColumnChecked,
	AboutCountModule, AboutCountModules, AboutCountList, AboutCountLists,
	AboutCountSource, AboutCountSources, AboutCountAsset, AboutCountAssets,
	AboutEmptyComponents, AboutEmptyData,
	AboutHeadingEmbedded, AboutEmptyEmbedded, AboutSaveNotices, AboutNoticesHeader,

	ManifestOpen, ManifestOpenResponse, ManifestEmpty, ManifestUnreadable,
	ManifestReading,
	ManifestHeadingRun, ManifestHeadingEntries, ManifestHeadingLimits,
	ManifestHeadingEntry, ManifestShowEntry,
	ManifestRunWait, ManifestEntriesWait, ManifestResponseWait,
	ManifestWaitMutator, ManifestWaitPlace, ManifestWaitExpect, ManifestWaitStatus,
	ManifestFactsEntry, ManifestFactsEntries, ManifestFactsSeed, ManifestFactsOf,
	ManifestFactsAnswered, ManifestFactsReading, ManifestFactsUnreadable,
	ManifestFactsNeverAnswered, ManifestFactsAgreed,
	ManifestTallyOrdered, ManifestTallyApplied, ManifestTallyNotApplied,
	ManifestTallyCompared, ManifestTallyDisagree, ManifestTallyNotSettled,
	ManifestDetailWait, ManifestDetailSelect, ManifestDetailEntry,
	ManifestDetailHard, ManifestDetailSoft, ManifestDetailNotApplied, ManifestDetailBecause,
	ManifestCommandResponse, ManifestCommandExample,
	ManifestRunGenerated, ManifestRunSeed, ManifestRunLocale, ManifestRunCatalog,
	ManifestRunSpec, ManifestRunRecipe, ManifestRunPolicy, ManifestRunTag,
	ManifestRunSource, ManifestRunRowsTotal, ManifestRunRowsMutated,
	ManifestCountRequested, ManifestCountApplied, ManifestCountNotApplied,
	ManifestEntryApplied, ManifestEntryNotApplied,
	ManifestEntryBefore, ManifestEntryAfter,
	ManifestAnswerAccepted, ManifestAnswerRejected, ManifestAnswerNone,
	ManifestAnswerNotSettled,
	ManifestColumnMutator, ManifestColumnPlace, ManifestColumnExpect,
	ManifestColumnStatus,
	ManifestPlaceWholeFile, ManifestPlaceRow, ManifestPlaceByte,
	ManifestExpectationRejection, ManifestExpectationAcceptance,
	ManifestExpectationWarning,
	ManifestConfidenceHard, ManifestConfidenceSoft, ManifestConfidenceMark,
	ManifestLegendMark,
	ManifestFailureUndescribed, ManifestFailureUnimplemented,
	ManifestFailureRunLacksCapability, ManifestFailureFieldTypeNotCovered,
	ManifestFailureSchemeNotCovered, ManifestFailureDeclinedByMutator,

	TakeChoose, TakeFileNone, TakeFileReading, TakeFileUnreadable, TakeFileWait, TakePlanWait,
	TakeDrop, TakeUnreadable, TakeReading,
	TakeHeadingFile, TakeHeadingSettings, TakeHeadingPlan, TakeHeadingResult,
	TakeStepFile, TakeStepWrite, TakeMoreOptions,
	TakeFileRows, TakeFileColumns, TakeFileNothingDeclared,
	TakeFileTakenFor, TakeFileBecause, TakeFileAgreesWith,
	TakeFileByteOrderMark, TakeFileNotWorkedOut, TakeFileNotInferred,
	TakeFileOneColumn, TakeFileNothingRecognised,
	TakeCountRecord, TakeCountRecords, TakeCountColumn, TakeCountColumns,
	TakeLayerBytes, TakeLayerStructure, TakeLayerFormat, TakeLayerMeaning,
	TakeLayerBetween, TakeLayerAcross, TakeLayerUnicode, TakeLayerUnusual,
	TakeLayerCode, TakeLayersNote, TakePerRowExactlyOne, TakePerRowAny,

	TakeSettingLayers, TakeSettingRows, TakeSettingPerRow, TakeSettingExclude,
	TakeSettingDelimiter, TakeSettingLocale, TakeSettingSeed, TakeSettingOut,
	TakeSettingOutHint, TakeSettingManifest, TakeSettingOverwrite, TakeSettingNone,
	TakePlanCarried, TakePlanOf, TakePlanWrites, TakePlanWritesTo,
	TakePlanEveryOrder,
	TakeRun, TakeStop, TakeRunning, TakeRunStopping,
	TakeResultWritten, TakeResultIn, TakeResultOrdered, TakeResultApplied,
	TakeResultMissing, TakeResultMissingWhy,
	TakeResultUnpartnered, TakeResultUnpartneredWhy,
	TakeFileMoreColumns,
	TakeResultOversized, TakeResultOversizedWhy,
	TakeResultStopped, TakeResultStoppedWhy,
	TakeResultFailed, TakeResultFailedWhy,
	TakeRefusalAllowed, TakeRefusalInstead,
	TakeRefusalNoLayers, TakeRefusalNoLayersWhy, TakeRefusalNoLayersDo,
	TakeRefusalUnknownLayer, TakeRefusalUnknownLayerWhy, TakeRefusalUnknownLayerDo,
	TakeRefusalDelimiter, TakeRefusalDelimiterWhy, TakeRefusalDelimiterAllowed,
	TakeRefusalDelimiterDo,
	TakeRefusalManifestForm, TakeRefusalManifestFormWhy, TakeRefusalManifestFormDo,
	TakeRefusalShare, TakeRefusalShareWhy, TakeRefusalShareAllowed, TakeRefusalShareDo,
	TakeRefusalPerRow, TakeRefusalPerRowWhy, TakeRefusalPerRowDo,
	TakeRefusalOverwriteSource, TakeRefusalOverwriteSourceWhy,
	TakeRefusalOverwriteSourceAllowed, TakeRefusalOverwriteSourceDo,
	TakeRefusalUnnamed, TakeRefusalUnnamedWhy, TakeRefusalUnnamedDo,

	VerifyNoAnswer,
	VerifyVerdictAccepted, VerifyVerdictRejected,
	VerifyKindHole, VerifyKindOverzealous, VerifyKindSilent, VerifyKindMissing,

	RecipeEmpty, RecipePlaceholder, RecipeChecking, RecipeName,
	RecipeExample, RecipeCanGenerate, RecipeGenerator, RecipeGenerators,
	RecipeNotIn, RecipeNothingToDrawFrom,
	RecipeHeadingRecipe, RecipeHeadingDescribes, RecipeHeadingResult, RecipeHeadingProblems,
	RecipeStepGenerate,
	RecipeNoObjections, RecipeLine, RecipeAllowed, RecipeTry,
	RecipeUnchecked, RecipeEntity, RecipeEntities, RecipeField, RecipeFields,
	RecipeShowing, RecipeOf, RecipeRow, RecipeRows, RecipeNoRows, RecipeNoFields,
	RecipeAsText, RecipeParentOutside, RecipeRefused, RecipeCrashed, RecipeWillNotDo,
	RecipeWriteTo, RecipeChooseDir, RecipeReplace, RecipeGenerate, RecipeStopRun,
	RecipeCommandFile, RecipeCommandExample,
	RecipePlanWrites, RecipeNeedDirectory, RecipeDirHint, RecipeWriting, RecipeStopping,
	RecipeResultWritten, RecipeResultRows, RecipeIn, RecipeRunStopped, RecipeRunFailed,
	RecipeSampleBeforeDamage, RecipeFormUnused,
}

// The catalogue travels inside the binary, because a window that needs a file
// beside it to name its own buttons is a window that breaks when somebody
// moves the executable.
//
//go:embed locales/en.yaml
var englishBytes []byte

// Catalogue is one language.
type Catalogue struct {
	language string
	text     map[Key]string
}

// English is the catalogue every build carries.
//
// It is parsed once at start rather than on each lookup. Parsing here can fail
// only if the embedded file stops being the file this package ships, which a
// test rules out, so the failure is turned into a catalogue that answers with
// its own keys rather than into a panic that takes the window down.
var English = func() Catalogue {
	loaded, err := Parse("en", englishBytes)
	if err != nil {
		return Catalogue{language: "en", text: map[Key]string{}}
	}
	return loaded
}()

// Parse reads a catalogue from the bytes of a locale file.
func Parse(language string, raw []byte) (Catalogue, error) {
	var flat map[string]string
	if err := yaml.Unmarshal(raw, &flat); err != nil {
		return Catalogue{}, fmt.Errorf("the %s catalogue is not readable: %w", language, err)
	}
	text := make(map[Key]string, len(flat))
	for name, value := range flat {
		text[Key(name)] = value
	}
	return Catalogue{language: language, text: text}, nil
}

// Language is the code of this catalogue.
func (c Catalogue) Language() string { return c.language }

// T is the text for a key.
//
// A missing key answers with the key itself. That is deliberate and it is the
// least bad of the three options: an empty string leaves a control with no
// label and looks like a layout fault, a panic takes down a window over a
// missing sentence, and the key at least names what is missing to whoever sees
// it. The pair of tests that close the register makes it unreachable in this
// build.
func (c Catalogue) T(key Key) string {
	if value, found := c.text[key]; found {
		return value
	}
	return string(key)
}

// Counted is a number with the thing it counts, in whichever of the two forms
// of the word fits it.
//
// # Why here rather than in each screen
//
// Written first in the take screen, then twice more in the manifest browser and
// the recipe screen as copies of each other, and a fourth copy would have come
// with the About screen on 2026-09-23. The two forms are the catalogue's, so
// the choice between them belongs beside the catalogue. The same limit as the
// note on QuickValuesCounted: a language whose number governs more than two
// forms needs a count aware catalogue, not a third key (B-184).
//
// Every screen counts through this since 2026-09-24 (B-294). The take screen
// was the last, and it was also the one adding a letter to a word written in
// the code, so its two words became keys on the way.
func (c Catalogue) Counted(many int, one, rest Key) string {
	if many == 1 {
		return strconv.Itoa(many) + " " + c.T(one)
	}
	return strconv.Itoa(many) + " " + c.T(rest)
}

// Names lists every key the catalogue carries, sorted, so a test comparing it
// against the constants is not also comparing two orderings.
func (c Catalogue) Names() []Key {
	names := make([]Key, 0, len(c.text))
	for name := range c.text {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool { return names[i] < names[j] })
	return names
}
