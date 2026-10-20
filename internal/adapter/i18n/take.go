package i18n

// The keys of the take and corrupt screen.
//
// # Why a file of their own
//
// i18n.go passed the ceiling on the length of a file again after the keys of
// the catalogue of controls had moved out, and a ceiling may be lowered and
// never raised. This screen carries more keys than any other, and nobody
// reading about one screen has to walk past another. The list of every key the
// window may ask for stays in i18n.go, beside the keys of the other screens,
// because that list is what the test closes.
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
	TakeChoose Key = "take.choose"
	// TakeFileNone stands in the band where the name of the file will, so
	// that the band says what to do before there is a file. TakeFileReading
	// and TakeFileUnreadable stand beside the name while it is being read and
	// after it could not be, and the sentences about either go in the card.
	TakeFileNone       Key = "take.file.none"
	TakeFileReading    Key = "take.file.reading"
	TakeFileUnreadable Key = "take.file.unreadable"
	// TakeFileWait names what the panel will hold, so that the panel is waiting
	// rather than blank before a file is chosen, and TakePlanWait does the same
	// for the card.
	TakeFileWait   Key = "take.file.wait"
	TakePlanWait   Key = "take.plan.wait"
	TakeDrop       Key = "take.drop"
	TakeUnreadable Key = "take.unreadable"
	TakeReading    Key = "take.reading"

	TakeHeadingFile     Key = "take.heading.file"
	TakeHeadingSettings Key = "take.heading.settings"
	TakeHeadingPlan     Key = "take.heading.plan"
	TakeHeadingResult   Key = "take.heading.result"

	// The first and the last of the three numbered steps the screen reads in
	// since DEC-261. The middle one is TakeHeadingSettings. TakeMoreOptions
	// heads the settings a first run does not need, folded away until asked.
	TakeStepFile    Key = "take.step.file"
	TakeStepWrite   Key = "take.step.write"
	TakeMoreOptions Key = "take.more"

	TakeFileRows              Key = "take.file.rows"
	TakeFileColumns           Key = "take.file.columns"
	TakeFileNothingDeclared   Key = "take.file.nothing-declared"
	TakeFileTakenFor          Key = "take.file.taken-for"
	TakeFileBecause           Key = "take.file.because"
	TakeFileAgreesWith        Key = "take.file.agrees-with"
	TakeFileByteOrderMark     Key = "take.file.byte-order-mark"
	TakeFileNotWorkedOut      Key = "take.file.not-worked-out"
	TakeFileNotInferred       Key = "take.file.not-inferred"
	TakeFileOneColumn         Key = "take.file.one-column"
	TakeFileNothingRecognised Key = "take.file.nothing-recognised"

	// How many records and columns a file holds, beside its name in the band
	// and at the head of the panel. Both forms of each word, for the reason the
	// recipe screen gives at RecipeEntity.
	TakeCountRecord  Key = "take.count.record"
	TakeCountRecords Key = "take.count.records"
	TakeCountColumn  Key = "take.count.column"
	TakeCountColumns Key = "take.count.columns"

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

	// TakePlanCarried heads the rows saying, for each kind ordered, how much
	// of it this build carries out, and TakePlanOf is the word between the two
	// figures of a row: 3 of 9. Two sums stood here until 2026-09-15 and said
	// that something was missing without saying what (DEC-245).
	TakePlanCarried    Key = "take.plan.carried"
	TakePlanOf         Key = "take.plan.of"
	TakePlanWrites     Key = "take.plan.writes"
	TakePlanWritesTo   Key = "take.plan.writes-to"
	TakePlanEveryOrder Key = "take.plan.every-order"

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
