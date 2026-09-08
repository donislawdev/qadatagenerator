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

	ScreenAbout    Key = "screen.about"
	ScreenManifest Key = "screen.manifest"
	ScreenTake     Key = "screen.take"
	ScreenRecipe   Key = "screen.recipe"
	ScreenQuick    Key = "screen.quick"

	// The headings the column groups its names under, added on 2026-09-08 with
	// DEC-225. They say what a person came to do rather than what the screens
	// are, which is the whole point of the order they now sit in.
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

	AboutLicenseCode Key = "about.license.code"
	AboutLicenseSpec Key = "about.license.spec"

	// Only the date carries a label. The module name, its version and its
	// license read as themselves on the line, and a label in front of each
	// turned thirty seven modules into a wall nobody could scan.
	AboutColumnChecked Key = "about.column.checked"

	AboutEmptyComponents Key = "about.empty.components"
	AboutEmptyData       Key = "about.empty.data"
)

// The keys of the take and corrupt screen.
//
// # Why the refusals are keys here and sentences at the other surface
//
// The mode refuses a run by kind rather than by sentence, because the command
// line is English by rule and this window shows whatever it was translated
// into. Every kind therefore needs words here, including the four that no
// control on this screen can produce: the layers are tick boxes and the
// manifest forms are a list, so nothing a person clicks can name one that does
// not exist. They are translated anyway, because the guard holding this
// register against the list of kinds cannot tell a refusal that is unreachable
// today from one that becomes reachable the day somebody swaps a list for a
// text field.
const (
	TakeChoose     Key = "take.choose"
	TakeEmpty      Key = "take.empty"
	TakeDrop       Key = "take.drop"
	TakeUnreadable Key = "take.unreadable"
	TakeReading    Key = "take.reading"

	TakeHeadingFile     Key = "take.heading.file"
	TakeHeadingSettings Key = "take.heading.settings"
	TakeHeadingPlan     Key = "take.heading.plan"
	TakeHeadingResult   Key = "take.heading.result"

	TakeFileRows              Key = "take.file.rows"
	TakeFileColumns           Key = "take.file.columns"
	TakeFileNothingDeclared   Key = "take.file.nothing-declared"
	TakeFileTakenFor          Key = "take.file.taken-for"
	TakeFileBecause           Key = "take.file.because"
	TakeFileAgreesWith        Key = "take.file.agrees-with"
	TakeFileByteOrderMark     Key = "take.file.byte-order-mark"
	TakeFileNotInferred       Key = "take.file.not-inferred"
	TakeFileOneColumn         Key = "take.file.one-column"
	TakeFileNothingRecognised Key = "take.file.nothing-recognised"

	TakeSettingLayers    Key = "take.setting.layers"
	TakeSettingRows      Key = "take.setting.rows"
	TakeSettingPerRow    Key = "take.setting.per-row"
	TakeSettingExclude   Key = "take.setting.exclude"
	TakeSettingDelimiter Key = "take.setting.delimiter"
	TakeSettingLocale    Key = "take.setting.locale"
	TakeSettingSeed      Key = "take.setting.seed"
	TakeSettingOut       Key = "take.setting.out"
	TakeSettingManifest  Key = "take.setting.manifest"
	TakeSettingOverwrite Key = "take.setting.overwrite"
	TakeSettingNone      Key = "take.setting.none"

	// The kinds of damage, said the way a person would rather than by the code
	// the catalogue files them under. Added 2026-09-08 with DEC-225: the codes
	// were the labels, and "L2" tells somebody opening this window nothing.
	// The code stays beside the name, because it is what the --layers flag
	// takes and what the manifest reports.
	TakeLayerBytes     Key = "take.layer.bytes"
	TakeLayerStructure Key = "take.layer.structure"
	TakeLayerFormat    Key = "take.layer.format"
	TakeLayerMeaning   Key = "take.layer.meaning"
	TakeLayerBetween   Key = "take.layer.between"
	TakeLayerAcross    Key = "take.layer.across"
	TakeLayerUnicode   Key = "take.layer.unicode"
	TakeLayerUnusual   Key = "take.layer.unusual"
	TakeLayerCode      Key = "take.layer.code"
	TakeLayersNote     Key = "take.layers.note"

	// How many defects one record may carry, said in words. The engine takes
	// exactly_one and any, and a list offering those two spellings asks a
	// person to read a setting file to use a window.
	TakePerRowExactlyOne Key = "take.per-row.exactly-one"
	TakePerRowAny        Key = "take.per-row.any"

	TakePlanKinds       Key = "take.plan.kinds"
	TakePlanImplemented Key = "take.plan.implemented"
	TakePlanWrites      Key = "take.plan.writes"
	TakePlanWritesTo    Key = "take.plan.writes-to"
	TakePlanEveryOrder  Key = "take.plan.every-order"

	// The words of a run: the one that starts it, the one that ends it early,
	// and what a person is told while it is going on.
	TakeRun         Key = "take.run"
	TakeStop        Key = "take.stop"
	TakeRunning     Key = "take.running"
	TakeRunStopping Key = "take.running.stopping"

	TakeResultWritten        Key = "take.result.written"
	TakeResultIn             Key = "take.result.in"
	TakeResultOrdered        Key = "take.result.ordered"
	TakeResultApplied        Key = "take.result.applied"
	TakeResultMissing        Key = "take.result.missing"
	TakeResultMissingWhy     Key = "take.result.missing.why"
	TakeResultUnpartnered    Key = "take.result.unpartnered"
	TakeResultUnpartneredWhy Key = "take.result.unpartnered.why"
	TakeFileMoreColumns      Key = "take.file.more.columns"
	TakeResultOversized      Key = "take.result.oversized"
	TakeResultOversizedWhy   Key = "take.result.oversized.why"
	TakeResultStopped        Key = "take.result.stopped"
	TakeResultStoppedWhy     Key = "take.result.stopped.why"
	TakeResultFailed         Key = "take.result.failed"
	TakeResultFailedWhy      Key = "take.result.failed.why"

	TakeRefusalAllowed Key = "take.refusal.allowed"
	TakeRefusalInstead Key = "take.refusal.instead"

	TakeRefusalNoLayers    Key = "take.refusal.no-layers"
	TakeRefusalNoLayersWhy Key = "take.refusal.no-layers.why"
	TakeRefusalNoLayersDo  Key = "take.refusal.no-layers.do"

	TakeRefusalUnknownLayer    Key = "take.refusal.unknown-layer"
	TakeRefusalUnknownLayerWhy Key = "take.refusal.unknown-layer.why"
	TakeRefusalUnknownLayerDo  Key = "take.refusal.unknown-layer.do"

	TakeRefusalDelimiter        Key = "take.refusal.delimiter"
	TakeRefusalDelimiterWhy     Key = "take.refusal.delimiter.why"
	TakeRefusalDelimiterAllowed Key = "take.refusal.delimiter.allowed"
	TakeRefusalDelimiterDo      Key = "take.refusal.delimiter.do"

	TakeRefusalManifestForm    Key = "take.refusal.manifest-form"
	TakeRefusalManifestFormWhy Key = "take.refusal.manifest-form.why"
	TakeRefusalManifestFormDo  Key = "take.refusal.manifest-form.do"

	TakeRefusalShare        Key = "take.refusal.share"
	TakeRefusalShareWhy     Key = "take.refusal.share.why"
	TakeRefusalShareAllowed Key = "take.refusal.share.allowed"
	TakeRefusalShareDo      Key = "take.refusal.share.do"

	TakeRefusalPerRow    Key = "take.refusal.per-row"
	TakeRefusalPerRowWhy Key = "take.refusal.per-row.why"
	TakeRefusalPerRowDo  Key = "take.refusal.per-row.do"

	TakeRefusalOverwriteSource        Key = "take.refusal.overwrite-source"
	TakeRefusalOverwriteSourceWhy     Key = "take.refusal.overwrite-source.why"
	TakeRefusalOverwriteSourceAllowed Key = "take.refusal.overwrite-source.allowed"
	TakeRefusalOverwriteSourceDo      Key = "take.refusal.overwrite-source.do"

	TakeRefusalUnnamed    Key = "take.refusal.unnamed"
	TakeRefusalUnnamedWhy Key = "take.refusal.unnamed.why"
	TakeRefusalUnnamedDo  Key = "take.refusal.unnamed.do"
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

	ManifestHeadingRun       Key = "manifest.heading.run"
	ManifestHeadingEntries   Key = "manifest.heading.entries"
	ManifestHeadingUnsettled Key = "manifest.heading.unsettled"
	ManifestHeadingLimits    Key = "manifest.heading.limits"

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

	ManifestColumnMutator    Key = "manifest.column.mutator"
	ManifestColumnPlace      Key = "manifest.column.place"
	ManifestColumnExpect     Key = "manifest.column.expect"
	ManifestColumnConfidence Key = "manifest.column.confidence"
	ManifestColumnStatus     Key = "manifest.column.status"
	ManifestColumnDetail     Key = "manifest.column.detail"

	ManifestPlaceWholeFile Key = "manifest.place.whole-file"
	ManifestPlaceRow       Key = "manifest.place.row"
	ManifestPlaceByte      Key = "manifest.place.byte"

	ManifestExpectationRejection  Key = "manifest.expectation.rejection"
	ManifestExpectationAcceptance Key = "manifest.expectation.acceptance"
	ManifestExpectationWarning    Key = "manifest.expectation.warning"

	ManifestConfidenceHard Key = "manifest.confidence.hard"
	ManifestConfidenceSoft Key = "manifest.confidence.soft"
	ManifestLegendSoft     Key = "manifest.legend.soft"

	ManifestFailureUndescribed         Key = "manifest.failure.undescribed"
	ManifestFailureUnimplemented       Key = "manifest.failure.unimplemented"
	ManifestFailureRunLacksCapability  Key = "manifest.failure.run-lacks-capability"
	ManifestFailureFieldTypeNotCovered Key = "manifest.failure.field-type-not-covered"
	ManifestFailureSchemeNotCovered    Key = "manifest.failure.scheme-not-covered"
	ManifestFailureDeclinedByMutator   Key = "manifest.failure.declined-by-mutator"

	VerifyCompared        Key = "verify.compared"
	VerifyAgreed          Key = "verify.agreed"
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

	RecipeHeadingRecipe   Key = "recipe.heading.recipe"
	RecipeHeadingProblems Key = "recipe.heading.problems"

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

	// The bar along the bottom, which is the half of this screen that writes.
	//
	// 🔴 The three reasons a run cannot start are three keys rather than one,
	// because a disabled button that says only that it is disabled sends
	// somebody looking for what they did wrong. Each of them names the one
	// thing to do next.
	RecipeWriteTo   Key = "recipe.write-to"
	RecipeChooseDir Key = "recipe.choose-dir"
	RecipeReplace   Key = "recipe.replace"
	RecipeGenerate  Key = "recipe.generate"
	// The heading over the controls that write, and the two keys of the
	// terminal line beneath them. The file name is an example rather than a
	// path, because this window does not write the recipe file.
	RecipeHeadingSave    Key = "recipe.heading.save"
	RecipeCommandFile    Key = "recipe.command.file"
	RecipeCommandExample Key = "recipe.command.example"
	RecipeStopRun        Key = "recipe.stop-run"
	RecipeWouldWrite     Key = "recipe.would-write"
	RecipeNothingYet     Key = "recipe.nothing-yet"
	RecipeNeedRecipe     Key = "recipe.need-recipe"
	RecipeNeedFix        Key = "recipe.need-fix"
	RecipeNeedDirectory  Key = "recipe.need-directory"
	RecipeDirHint        Key = "recipe.dir-hint"
	RecipeWriting        Key = "recipe.writing"
	RecipeStopping       Key = "recipe.stopping"
	RecipeWrote          Key = "recipe.wrote"
	RecipeIn             Key = "recipe.in"
	RecipeFile           Key = "recipe.file"
	RecipeFiles          Key = "recipe.files"
	RecipeRunStopped     Key = "recipe.run-stopped"
	RecipeRunFailed      Key = "recipe.run-failed"
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
	// What stands where the value will be, before there is one. The drawing
	// this screen was built from only ever shows a value, so the state this
	// screen opens in was never drawn at all, and it opened as an empty box
	// with a border round it.
	QuickNothingYet  Key = "quick.nothing-yet"
	QuickLabelLocale Key = "quick.label.locale"
	QuickLabelSeed   Key = "quick.label.seed"
	QuickLabelGender Key = "quick.label.gender"
	QuickLabelBorn   Key = "quick.label.born"
	QuickGenderAny   Key = "quick.gender.any"
	QuickCopy        Key = "quick.copy"
	QuickAgain       Key = "quick.again"
	QuickCopied      Key = "quick.copied"

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

	// The line a person can paste into a terminal to get the same thing.
	// docs/08 asks for this on every screen that produces something, and says
	// why: without it the window and the terminal stop being one product.
	CommandHeading Key = "command.heading"
	CommandCopy    Key = "command.copy"
	CommandCopied  Key = "command.copied"

	// One line at the top of each screen saying what it is for. The column
	// gives a screen its name, and a name is not a purpose: somebody opening
	// this window for the first time reads "Take and corrupt" and learns
	// nothing about what it would do to their file.
	PurposeTake     Key = "purpose.take"
	PurposeRecipe   Key = "purpose.recipe"
	PurposeManifest Key = "purpose.manifest"

	// The machine readable form of what this screen says. DEC-30 names the
	// reader and it is a compliance scanner, which reads documents rather than
	// paragraphs, so the window has to PRODUCE the file rather than show it.
	AboutSaveJSON Key = "about.save.json"
)

// Keys is every key a screen may ask for, so a test can close the register
// from the side the code is on.
var Keys = []Key{
	WindowTitle, ScreenAbout, ScreenManifest, ScreenTake, ScreenRecipe, ScreenQuick,
	SidebarGroupMake, SidebarGroupCheck,

	QuickNothingWritten, QuickLabelValue, QuickValueHeading, QuickNothingYet,
	QuickLabelGender, QuickLabelBorn,
	QuickLabelLocale, QuickLabelSeed,
	QuickGenderAny, QuickCopy, QuickAgain, QuickCopied,
	QuickRefusedGenerator, QuickRefusedRole, QuickRefusedGender, QuickRefusedDate,
	CommandHeading, CommandCopy, CommandCopied,
	PurposeTake, PurposeRecipe, PurposeManifest, AboutSaveJSON,

	AboutHeadingVersions, AboutHeadingLicenses, AboutHeadingComponents,
	AboutHeadingData, AboutHeadingNotices,
	AboutVersionTool, AboutVersionCatalog, AboutVersionManifest, AboutVersionRecipe,
	AboutLicenseCode, AboutLicenseSpec,
	AboutColumnChecked,
	AboutEmptyComponents, AboutEmptyData,

	ManifestOpen, ManifestOpenResponse, ManifestEmpty, ManifestUnreadable,
	ManifestReading,
	ManifestHeadingRun, ManifestHeadingEntries, ManifestHeadingUnsettled,
	ManifestHeadingLimits,
	ManifestRunGenerated, ManifestRunSeed, ManifestRunLocale, ManifestRunCatalog,
	ManifestRunSpec, ManifestRunRecipe, ManifestRunPolicy, ManifestRunTag,
	ManifestRunSource, ManifestRunRowsTotal, ManifestRunRowsMutated,
	ManifestCountRequested, ManifestCountApplied, ManifestCountNotApplied,
	ManifestEntryApplied, ManifestEntryNotApplied,
	ManifestEntryBefore, ManifestEntryAfter,
	ManifestColumnMutator, ManifestColumnPlace, ManifestColumnExpect,
	ManifestColumnConfidence, ManifestColumnStatus, ManifestColumnDetail,
	ManifestPlaceWholeFile, ManifestPlaceRow, ManifestPlaceByte,
	ManifestExpectationRejection, ManifestExpectationAcceptance,
	ManifestExpectationWarning,
	ManifestConfidenceHard, ManifestConfidenceSoft, ManifestLegendSoft,
	ManifestFailureUndescribed, ManifestFailureUnimplemented,
	ManifestFailureRunLacksCapability, ManifestFailureFieldTypeNotCovered,
	ManifestFailureSchemeNotCovered, ManifestFailureDeclinedByMutator,

	TakeChoose, TakeEmpty, TakeDrop, TakeUnreadable, TakeReading,
	TakeHeadingFile, TakeHeadingSettings, TakeHeadingPlan, TakeHeadingResult,
	TakeFileRows, TakeFileColumns, TakeFileNothingDeclared,
	TakeFileTakenFor, TakeFileBecause, TakeFileAgreesWith,
	TakeFileByteOrderMark, TakeFileNotInferred,
	TakeFileOneColumn, TakeFileNothingRecognised,
	TakeLayerBytes, TakeLayerStructure, TakeLayerFormat, TakeLayerMeaning,
	TakeLayerBetween, TakeLayerAcross, TakeLayerUnicode, TakeLayerUnusual,
	TakeLayerCode, TakeLayersNote, TakePerRowExactlyOne, TakePerRowAny,

	TakeSettingLayers, TakeSettingRows, TakeSettingPerRow, TakeSettingExclude,
	TakeSettingDelimiter, TakeSettingLocale, TakeSettingSeed, TakeSettingOut,
	TakeSettingManifest, TakeSettingOverwrite, TakeSettingNone,
	TakePlanKinds, TakePlanImplemented, TakePlanWrites, TakePlanWritesTo,
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

	VerifyCompared, VerifyAgreed, VerifyNoAnswer,
	VerifyVerdictAccepted, VerifyVerdictRejected,
	VerifyKindHole, VerifyKindOverzealous, VerifyKindSilent, VerifyKindMissing,

	RecipeEmpty, RecipePlaceholder, RecipeChecking, RecipeName,
	RecipeExample, RecipeCanGenerate, RecipeGenerator, RecipeGenerators,
	RecipeNotIn, RecipeNothingToDrawFrom,
	RecipeHeadingRecipe, RecipeHeadingProblems,
	RecipeNoObjections, RecipeLine, RecipeAllowed, RecipeTry,
	RecipeUnchecked, RecipeEntity, RecipeEntities, RecipeField, RecipeFields,
	RecipeShowing, RecipeOf, RecipeRow, RecipeRows, RecipeNoRows, RecipeNoFields,
	RecipeAsText, RecipeParentOutside, RecipeRefused, RecipeCrashed, RecipeWillNotDo,
	RecipeWriteTo, RecipeChooseDir, RecipeReplace, RecipeGenerate, RecipeStopRun,
	RecipeHeadingSave, RecipeCommandFile, RecipeCommandExample,
	RecipeWouldWrite, RecipeNothingYet, RecipeNeedRecipe, RecipeNeedFix,
	RecipeNeedDirectory, RecipeDirHint, RecipeWriting, RecipeStopping, RecipeWrote, RecipeIn,
	RecipeFile, RecipeFiles, RecipeRunStopped, RecipeRunFailed,
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
