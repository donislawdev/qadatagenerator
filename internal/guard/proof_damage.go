package guard

// The half of the register about the second pillar: the catalog of damage, the
// manifest of expectations, the mutators, and everything that reads or writes
// either of them.
//
// It is a continuation of proof.go and not a register of its own. The reason
// the register is in two files is written beside toolGuards there, and every
// tool that reads it reads both.
var damageGuards = []Guard{
	// --- the corruption catalog ---------------------------------------------
	//
	// The catalog is data with one seam to the code, and the seam is an
	// identifier. These cover the two ways that arrangement fails: the shipped
	// copy of the data drifting away from the published one, and an entry
	// arriving in a state nothing downstream can use.
	//
	// The first of them protects a file rather than a function, which is the
	// same shape the postal registry needed. A copy of the catalog is only
	// allowed to exist because something compares it with the original.
	{"the catalog in the binary is the published one", "TestTheEmbeddedCatalogIsThePublishedOneByteForByte", "internal/adapter/catalogyaml", ByMutation},
	{"a broken entry stops the whole catalog rather than going missing from it", "TestOneBrokenEntryRefusesTheWholeCatalog", "internal/domain/catalog", ByMutation},
	{"a published example still satisfies the scheme it belongs to", "TestTheExamplesSatisfyWhatTheirEntriesClaim", "internal/adapter/catalogyaml", ByMutation},
	{"a parameter default arrives as the catalog wrote it, null included", "TestParameterDefaultsSurviveIncludingTheOneWrittenAsNull", "internal/adapter/catalogyaml", ByMutation},
	{"a mutator with behaviour and no entry is named", "TestAnImplementationWithNoEntryIsNamed", "internal/domain/catalog", ByMutation},
	//
	// The other direction of that closure, an entry with no behaviour, is not
	// here. It cannot fail in this build: nothing is implemented, so every
	// entry is uncovered and a guard asserting equality would pass against a
	// build with no engine at all. It lands with the first mutator.
	{"an entry of a scheme we can compute cannot go unexamined", "TestNoEntryOfAKnownSchemeGoesUnclaimed", "internal/adapter/catalogyaml", ByMutation},
	{"a parameter reaches the model with the name the file gave it", "TestAParameterWithNoNameIsRefused", "internal/adapter/catalogyaml", ByMutation},
	{"the published manifest example carries identifiers a validator would accept", "TestTheIdentifiersInThePublishedManifestExampleAreSound", "internal/adapter/catalogyaml", ByMutation},

	// --- the expectation manifest -------------------------------------------
	//
	// The first two are the rule the product rests on. Damage that could not be
	// applied is still an entry and still says why, because a dataset where
	// twenty seven of forty ordered defects happened looks exactly like a
	// complete one and will be trusted as one.
	{"damage that did not happen still says why", "TestAnEntryThatDidNotHappenHasToSayWhy", "internal/domain/manifest", ByMutation},
	{"an entry that did not happen describes no change", "TestAnEntryThatDidNotHappenCannotCarryValues", "internal/domain/manifest", ByMutation},
	{"the summary cannot disagree with the entries under it", "TestTheCountsComeFromTheEntriesAndNotFromWhoeverBuiltThem", "internal/domain/manifest", ByMutation},
	{"a set integrity defect names every record it spans", "TestALocationHasToSayEnoughForItsLayer", "internal/domain/manifest", ByMutation},

	// --- the two halves of a mutator, and the seam between them -------------
	//
	// The catalog says what damage is and what a test should conclude from it.
	// The implementations say what it does. These hold the seam: an
	// implementation with no entry has no declared expectation, and one filed
	// under somebody else's identifier would be handed the wrong entry.
	{"damage that runs has an entry declaring what to expect of it", "TestEveryImplementedMutatorHasACatalogEntry", "internal/adapter/mutators", ByMutation},
	{"damage is filed under the identifier it implements", "TestEveryMutatorIsFiledUnderItsOwnIdentifier", "internal/adapter/mutators", ByMutation},
	{"damage that would change nothing is refused rather than reported as done", "TestDamageThatWouldChangeNothingIsRefusedWithAReason", "internal/adapter/mutators", ByMutation},
	{"a mutator meant for a whole file cannot be aimed at a column", "TestTheEntrySaysWhichFieldsItSuits", "internal/domain/catalog", ByMutation},

	// --- damage to a check digit, and the dimension it forced ---------------
	//
	// 🔴 Every national identifier is a field of type identifier and they
	// disagree about what they carry. Damage aimed at a check digit is
	// meaningless for a scheme that has none, and applied there it would write
	// a hard expectation of rejection over a value nothing rejects. The type
	// cannot answer that, so the entry is asked which schemes it covers and the
	// schema says which one a column follows.
	{"a value damaged at its check digit is one its own scheme refuses", "TestEverySchemeThisDamageCoversStopsAcceptingTheDamagedValue", "internal/adapter/mutators", ByMutation},
	{"the published example is what this build produces", "TestTheCatalogExampleIsWhatThisMutatorProduces", "internal/adapter/mutators", ByMutation},
	{"damage refuses a position where the expectation it declares is not certain", "TestChecksumDamageThatCannotBeDoneIsRefusedWithAReason", "internal/adapter/mutators", ByMutation},
	{"damage meant for named schemes cannot be aimed at another", "TestTheEntrySaysWhichSchemesItSuits", "internal/domain/catalog", ByMutation},
	{"a column says which numbering scheme it follows", "TestAFieldDrawnFromASchemeSaysWhichOne", "internal/adapter/values", ByMutation},
	{"this build does not quietly stop carrying out damage it used to", "TestThisBuildHasNotLostAMutator", "internal/adapter/mutators", ByMutation},

	// --- damage that has to build a value rather than rewrite one -----------
	//
	// 🔴 Layer four damage leaves a value that is correct on its own and wrong
	// only beside the column next to it, which is the defect applications
	// almost never catch and the reason the second pillar exists. Producing one
	// means rebuilding the identifier around a different answer, check digit
	// included, so the scheme itself has to do the writing and has to reach the
	// mutator to do it.
	{"the scheme behind a column reaches the mutator that damages it", "TestTheSchemeBehindAColumnReachesTheMutator", "internal/app/corrupt", ByMutation},
	{"an identifier left disagreeing with its row is still correct on its own", "TestTheGenderFlippedIdentifierIsStillCorrectOnItsOwn", "internal/adapter/mutators", ByMutation},
	{"a value that did not move is never recorded as damaged", "TestASchemeThatRebuildsNothingIsRefused", "internal/adapter/mutators", ByMutation},
	{"damage about two fields disagreeing needs both of them to exist", "TestGenderCoherenceDamageRequiresARelationToDisagreeWith", "internal/adapter/mutators", ByMutation},
	{"a birth date moved by years moves its year and nothing else", "TestTheDateDriftedIdentifierIsStillCorrectOnItsOwn", "internal/adapter/mutators", ByMutation},
	{"the published example of the date damage is what this build produces", "TestTheCatalogExampleIsWhatTheDateMutatorProduces", "internal/adapter/mutators", ByMutation},
	{"a birth date with nowhere to land is refused rather than moved elsewhere", "TestABirthDateWithNowhereToLandIsRefusedWithTheDateNamed", "internal/adapter/mutators", ByMutation},
	{"a knob that is not a whole number is refused rather than read as nought", "TestDateDamageThatCannotBeDoneIsRefusedWithAReason", "internal/adapter/mutators", ByMutation},
	{"damage about a date disagreeing needs the date column to exist", "TestDateCoherenceDamageRequiresARelationToDisagreeWith", "internal/adapter/mutators", ByMutation},

	// --- moving a value into a block nobody is issued from ------------------
	//
	// 🔴 Pillar one already generates inside a never issued block wherever a
	// scheme has one, so the question that means anything is not whether the
	// value is reserved but whether it already sits in the block this damage
	// aims at. Those are two different lines of defence, and telling them apart
	// is what makes the mutator applicable at all rather than never.
	{"a value moved into a spare block leaves the one it was generated in", "TestTheMovedValueLeavesTheBlockThisToolGeneratesIn", "internal/adapter/mutators", ByMutation},
	{"a scheme refusing to move a value is never recorded as damage done", "TestRangeDamageThatCannotBeDoneIsRefusedWithAReason", "internal/adapter/mutators", ByMutation},
	{"the published example of the range damage is what this build produces", "TestTheCatalogExampleIsWhatTheRangeMutatorProduces", "internal/adapter/mutators", ByMutation},

	// --- going past a declared limit, and landing on it ----------------------
	//
	// 🔴 The value has to end up past the limit the COLUMN declares rather than
	// merely longer than it was, and it has to land on the mark rather than step
	// over it. A length check written with an off by one error accepts exactly
	// one more than it should, so a value two past is refused by the broken
	// check as surely as by a correct one and the defect stays hidden.
	{"a lengthened value ends up past the limit its column declares", "TestTheValueEndsUpPastTheDeclaredLimit", "internal/adapter/mutators", ByMutation},
	// 🔴 A guard about cost rather than about an answer, because the quadratic
	// version produced the very same value. The limit comes from the recipe,
	// which is untrusted input, and counting the whole value on every pass made
	// eighty thousand characters take 939 ms.
	{"reaching a declared limit costs time in proportion to it", "TestReachingALargeLimitCostsTimeInProportionToIt", "internal/adapter/mutators", ByMutation},
	{"a knob missing or malformed is refused rather than guessed at", "TestLengthDamageThatCannotBeDoneIsRefusedWithAReason", "internal/adapter/mutators", ByMutation},
	{"a capability belonging to a column is asked of the column", "TestACapabilityThatBelongsToAColumnIsAskedOfTheColumn", "internal/app/corrupt", ByMutation},
	{"damage needing something of a column lands where it is", "TestDamageNeedingSomethingOfTheColumnLandsWhereItIs", "internal/app/corrupt", ByMutation},
	{"a value that cannot land on the mark is refused rather than left past it", "TestAValueThatCannotLandOnTheMarkIsRefusedRatherThanOvershot", "internal/adapter/mutators", ByMutation},
	{"a value that is already empty is never recorded as emptied", "TestAValueThatIsAlreadyEmptyIsRefusedWithAReason", "internal/adapter/mutators", ByMutation},

	// --- saying why, in a form a machine can act on -------------------------
	//
	// 🔴 The sentence beside an unapplied defect is written for a person and is
	// free to be reworded whenever a reword makes it clearer. Anything
	// downstream telling "this build cannot do it yet" from "your recipe does
	// not carry what it needs" would otherwise be matching on English prose,
	// which is the pattern that already gave this project a real defect in the
	// routing number oracle. The kind travels beside the reason and neither
	// replaces the other.
	{"damage that did not happen says of what kind, not only in prose", "TestAnEntryThatDidNotHappenHasToSayOfWhatKind", "internal/domain/manifest", ByMutation},
	{"two different reasons to refuse are told apart", "TestEveryRefusalCarriesTheKindThatMatches", "internal/app/corrupt", ByMutation},

	// --- the engine: one ordered damage, exactly one entry ------------------
	//
	// The failure this has to make impossible is not damage going wrong but
	// damage going missing. A dataset where twenty seven of forty ordered
	// defects happened looks exactly like a complete one.
	{"damage that did not happen comes back saying why", "TestEveryWayDamageCanFailStillProducesAnEntry", "internal/app/corrupt", ByMutation},
	{"damage that cannot be undone offers no way back", "TestIrreversibleDamageDoesNotCarryTheOriginal", "internal/app/corrupt", ByMutation},
	{"damage lands only on the fields its entry covers", "TestDamageAimedAtAFieldItDoesNotCoverIsRefusedWithBothSides", "internal/app/corrupt", ByMutation},
	{"damage lands only on the numbering schemes its entry covers", "TestDamageAimedAtASchemeItDoesNotCoverIsRefusedWithBothSides", "internal/app/corrupt", ByMutation},
	{"nothing the engine produces is unrecordable", "TestNoEntryThisEngineProducesIsOneTheManifestWouldRefuse", "internal/app/corrupt", ByMutation},

	// --- the corruption policy ----------------------------------------------
	//
	// The catalog reached validate in the same commit as the policy key, so the
	// two verbs have never disagreed about a policy. That is the lesson of the
	// generator registry applied before it could be paid for a second time.
	{"validate refuses a policy corrupt would do nothing about", "TestAPolicyNamingALayerWithNoDamageIsRefused", "internal/app/validate", ByMutation},
	{"a policy is told which of its damage this build cannot do yet", "TestAPolicyIsToldWhatThisBuildWillNotDoYet", "internal/app/validate", ByMutation},
	{"a policy section that orders nothing is refused", "TestAPolicyThatOrdersNothingIsRefused", "internal/adapter/recipeyaml", ByMutation},

	// --- the seam between what a recipe declares and what the catalog aims at -
	//
	// 🔴 The catalog aims damage at field types and a recipe declares
	// generators, and until 2026-08-10 nothing joined the two. A field with a
	// generator carried no type, an empty type matches no entry except the two
	// using the reserved word any, so an ordinary recipe left fifty of the fifty
	// two entries inapplicable for a reason nobody had chosen. Two halves, each
	// correct about its own half, which is the shape of the quiet mismatch the
	// audit found in PL.pesel.
	{"the type of a field follows from its generator", "TestAFieldWithAGeneratorGetsTheTypeThatGeneratorProduces", "internal/adapter/values", ByMutation},

	// --- values for a field described by type alone --------------------------
	{"a number stays inside the column it sits in", "TestANumberHonoursTheLengthOfItsColumn", "internal/adapter/values", ByMutation},
	{"a decimal carries the same places on every row", "TestADecimalAlwaysCarriesTwoPlaces", "internal/adapter/values", ByMutation},
	{"a column this build will leave empty is named", "TestAColumnThisBuildWillLeaveEmptyIsNamed", "internal/app/generate", ByMutation},

	// --- choosing what damage lands where -----------------------------------
	//
	// Three separate choices with three different mechanisms, and each of them
	// can go wrong on its own. Taking the mutator in order is what makes a
	// policy cover a line of defence rather than sample it, and drawing the row
	// from the seed is determinism applied to damage rather than to values.
	{"every kind of damage is ordered before any repeats", "TestEveryKindOfDamageIsOrderedBeforeAnyRepeats", "internal/app/corrupt", ByMutation},
	{"which rows are damaged follows from the seed", "TestAShareOfRowsDamagesAboutThatShare", "internal/app/corrupt", ByMutation},
	{"a column the policy excluded is never damaged", "TestAnExcludedColumnIsNeverAimedAt", "internal/app/corrupt", ByMutation},
	{"an exclusion emptying any entity is refused before the run, and not only one emptying the first", "TestAnExclusionThatEmptiesAnyEntityIsRefused", "internal/app/corrupt", ByMutation},
	{"damage is aimed at a column of the scheme it covers when the dataset has one", "TestDamageMeantForOneSchemeLandsOnThatScheme", "internal/app/corrupt", ByMutation},
	{"a policy that excludes everything is refused rather than silent", "TestAPolicyThatExcludesEverythingIsRefused", "internal/app/corrupt", ByMutation},
	// 🔴 The same statement as the recipe reader's entry above, in the second
	// place that reads a share, and it is here because the first one did not
	// travel. That package has carried the rule, the reasoning, a named guard
	// and a mutation since it learned to read a percentage. This one was written
	// six days later without any of them, and a share of NaN passed both range
	// tests and then chose no row at all. Measured on the binary: a policy that
	// ordered sixty defects ordered none and the run ended with success.
	{"a share that is not a number is refused rather than damaging nothing", "TestAPolicyShareThatIsNotANumberIsRefused", "internal/app/corrupt", ByMutation},

	// --- the manifest on disk ------------------------------------------------
	//
	// 🔴 The entries are written as they happen, so the model cannot be asked
	// about the whole document at the end: by then it is on a disk. The check
	// moved to the last moment where refusing still means something, and the
	// first run of the binary proved it was needed.
	{"an entry the model refuses never reaches the file", "TestAnEntryTheModelWouldRefuseNeverReachesTheFile", "internal/adapter/manifestjson", ByMutation},
	{"a value emptied on purpose is written rather than dropped", "TestAValueThatIsEmptyOnPurposeIsWrittenRatherThanDropped", "internal/adapter/manifestjson", ByMutation},
	{"a run with no entries is still a document", "TestARunWithNoEntriesIsStillADocument", "internal/adapter/manifestjson", ByMutation},

	// --- reading somebody else's file ----------------------------------------
	//
	// 🔴 Everything here defends one sentence: a column is only said to hold
	// something when the values prove it. The expensive half is the refusal to
	// name a column from a validator that answers about shape, because that
	// refusal costs a real mutator on a real column of social security numbers
	// and buys the promise that a hard expectation is never published on a guess.
	{"a scheme with no check digit never names a column", "TestAColumnOfRealSocialSecurityNumbersIsStillNotRecognised", "internal/adapter/infer", ByMutation},
	{"too few values tell a scheme from a coincidence", "TestTooFewValuesLeaveAColumnUnrecognised", "internal/adapter/infer", ByMutation},
	{"every value has to fit before a column is named", "TestOneValueThatDoesNotFitRefusesTheColumn", "internal/adapter/infer", ByMutation},
	{"two schemes fitting equally well name neither", "TestTwoSchemesAcceptingEverythingLeaveTheColumnUnrecognised", "internal/adapter/infer", ByMutation},
	{"what this way of looking never reads is said out loud", "TestTheLimitsOfThisWayOfLookingAreStated", "internal/adapter/infer", ByMutation},
	{"a column of national identifiers is recognised from its values", "TestAColumnOfNationalIdentifiersIsRecognisedFromItsValues", "internal/adapter/infer", ByMutation},
	{"nothing a file cannot declare is claimed about a column", "TestNothingBeyondTypeAndSchemeIsClaimed", "internal/adapter/infer", ByMutation},

	// The reader. A value quietly mangled on the way in is damage nobody
	// ordered and the manifest does not describe, which is the failure this
	// whole product exists to find in other people's tools.
	{"a file with a header and no records is refused", "TestAHeaderWithNoRecordsIsRefused", "internal/adapter/sourcecsv", ByMutation},
	{"a header that cannot name a column is refused", "TestAHeaderThatCannotNameAColumnIsRefused", "internal/adapter/sourcecsv", ByMutation},
	{"broken quoting is refused rather than salvaged", "TestBrokenQuotingIsRefusedRatherThanSalvaged", "internal/adapter/sourcecsv", ByMutation},
	{"the row count is known before the first record", "TestTheRowCountIsKnownBeforeTheFirstRecord", "internal/adapter/sourcecsv", ByMutation},
	{"a ragged row is refused rather than shifted into place", "TestARaggedRowIsRefused", "internal/adapter/sourcecsv", ByMutation},
	{"the sample is capped and the count is not", "TestTheSampleIsCappedAndTheCountIsNot", "internal/adapter/sourcecsv", ByMutation},
	// 🔴 The third road to the channel for the same class of failure. The first
	// was the recipe reader and is bounded, the second is the errors of the
	// numbering schemes and is still open. This one is the worst placed: a
	// recipe has a hard size limit and a data file deliberately has none.
	{"a hostile column name does not come back amplified", "TestALongColumnNameDoesNotComeBackAsALongMessage", "internal/adapter/sourcecsv", ByMutation},
	// --- exporting the manifest -----------------------------------------------
	//
	// 🔴 An export is a peer of the canonical manifest rather than a rendering
	// of it afterwards, so these guard the same promise from another side: a row
	// in an export is an instruction to a test, and a wrong one is worse than a
	// missing one.
	{"only damage that happened becomes a row a test can send", "TestOnlyAppliedDamageBecomesARowAndTheRestIsCounted", "internal/adapter/manifestexport", ByMutation},
	{"the exported table carries the keyword that makes it pasteable", "TestTheTableCarriesTheKeywordThatMakesItPasteable", "internal/adapter/manifestexport", ByMutation},
	{"a table character in somebody's column name does not break the table", "TestAValueCarryingATableCharacterDoesNotBreakTheTable", "internal/adapter/manifestexport", ByMutation},
	// 🔴 A guard on a decision rather than on a behaviour. Padding the one
	// unbounded column to its widest value would multiply a hostile column name
	// by the number of rows, which is the amplification class arriving in a
	// place nobody would look for it.
	{"one enormous column name does not pad every other row", "TestOneEnormousColumnNameDoesNotPadEveryOtherRow", "internal/adapter/manifestexport", ByMutation},
	// 🔴 A file written and not named is a silence, and this one was found by
	// running the tool rather than by reading it. The guard closes both ways:
	// naming the export in every run would pass a search for the name and would
	// promise a file most runs never write.
	{"a run names the export it wrote and names it only then", "TestARunNamesTheExportItWroteAndOnlyThen", "internal/cli", ByMutation},
	// 🔴 The three forms differ by who opens them, and that decides what each
	// one carries. These guard the decisions rather than the rendering: the
	// spreadsheet form is a review of the whole run and keeps what was not
	// applied, the report form is about damage that happened and counts the
	// rest in words.
	{"every ordered entry has a row in the spreadsheet form", "TestEveryOrderedEntryHasARowAndSaysWhetherItHappened", "internal/adapter/manifestexport", ByMutation},
	{"the report form shows what happened and counts the rest", "TestTheReportShowsWhatHappenedAndCountsTheRest", "internal/adapter/manifestexport", ByMutation},
	// 🔴 The one injection class the catalog names as aimed at the file an
	// application exports, arriving in the file this tool exports. A report
	// that runs code when it is opened is our defect and not a property of the
	// spreadsheet.
	{"a value a spreadsheet would run is written so it shows", "TestAValueASpreadsheetWouldRunIsWrittenSoItShows", "internal/adapter/manifestexport", ByMutation},
	{"a value a mutator produced never reaches the spreadsheet form", "TestTheValuesThemselvesAreNotInTheSpreadsheetForm", "internal/adapter/manifestexport", ByMutation},
	// 🔴 The same class as the missing Examples keyword, one form along: without
	// the rule the document renders as a paragraph, every renderer accepts it
	// and what arrives in the report is unreadable.
	{"the report form carries the rule that makes it render", "TestTheTableCarriesTheRuleThatMakesItRender", "internal/adapter/manifestexport", ByMutation},
	{"a pipe in a column name does not add a column to the report", "TestAPipeInAColumnNameDoesNotAddAColumn", "internal/adapter/manifestexport", ByMutation},
	{"every form this build lists writes to a file of its own", "TestEveryFormHasItsOwnFileAndNothingElseDoes", "internal/adapter/manifestexport", ByMutation},
	// 🔴 The direction that was missing. Two tests already went from the
	// documentation to the parser, and a flag in the parser and absent from the
	// documentation satisfied both. That is how --manifest came to be accepted
	// by two verbs while appearing in the help of neither.
	{"every flag a verb accepts is one it documents", "TestEveryFlagAVerbAcceptsIsOneItDocuments", "internal/cli", ByMutation},

	// 🔴 The one property that came off the list of what a file cannot say,
	// and it came off because it is the one that can be measured. What a
	// gender column looks like in somebody else's file cannot be known, so
	// what is proved is the mapping rather than the vocabulary. The threshold
	// is the same bar the schemes answer to and it is counted in ways of being
	// dealt out rather than in records, because a split of seven to one agrees
	// by accident one time in four.
	{"a column standing for the encoded gender is tied to it", "TestAColumnStandingForTheEncodedGenderIsTiedToIt", "internal/adapter/infer", ByMutation},
	{"an agreement worth nothing is refused", "TestAnAgreementWorthNothingIsRefused", "internal/adapter/infer", ByMutation},
	// 🔴 Empty cells are dropped when a column is sampled, so two columns line
	// up by position only when neither has a gap. Reading them by position
	// compares one person's identifier against another person's gender and
	// calls the disagreement evidence.
	{"two columns are lined up by record and not by position", "TestColumnsAreLinedUpByRecordAndNotByPosition", "internal/adapter/infer", ByMutation},
	// 🔴 The model gained a member and the writer kept its own document
	// type, so both halves were correct on their own while the file said
	// less than the run knew. Nothing compared the two until this did.
	{"an inferred relation reaches the file with its basis", "TestAnInferredRelationReachesTheFileWithItsBasis", "internal/adapter/manifestjson", ByMutation},

	// 🔴 The verb that reports on somebody else's application, and the only one
	// that returns the expectation code. What it must never do is assert more
	// than it was given: a soft disagreement belongs to the application's own
	// rules, and a question the response record cannot answer is not a pass.
	{"only a hard disagreement returns the expectation code", "TestOnlyAHardDisagreementReturnsTheExpectationCode", "internal/cli", ByMutation},
	{"a soft disagreement is reported and does not fail the run", "TestASoftDisagreementIsReportedAndDoesNotFailTheRun", "internal/app/verify", ByMutation},
	{"a question the record cannot answer is left unsettled", "TestAQuestionTheRecordCannotAnswerIsLeftUnsettled", "internal/app/verify", ByMutation},
	// 🔴 Measured on a real engine: a row taken, text turned into zero and a
	// string cut to the column width, with no error at all. A system holding
	// something other than what it was sent took a different value.
	{"a value quietly changed is told from one taken as sent", "TestAValueQuietlyChangedIsToldFromOneTakenAsSent", "internal/app/verify", ByMutation},
	{"records nobody reacted to are counted from the dataset", "TestRecordsNobodyReactedToAreCountedFromTheDataset", "internal/app/verify", ByMutation},
	{"looking and finding nothing is not the same as not looking", "TestLookingAndFindingNothingIsNotTheSameAsNotLooking", "internal/adapter/responsejson", ByMutation},
	{"an entry a scenario cannot check is left out and counted", "TestAnEntryAScenarioCannotCheckIsLeftOutAndCounted", "internal/adapter/manifestexport", ByMutation},

	// 🔴 The second road of the same class, and the one that needed a guard
	// rather than a repair. Thirteen messages across four schemes wrote a value
	// in with the quoting verb, so a fourteenth scheme would have written a
	// fourteenth. The guard is what makes the fourteenth impossible.
	{"no numbering scheme writes an unbounded value into a message", "TestNoSchemeWritesAValueIntoAMessageUnbounded", "internal/guard", ByMutation},
	{"the guard for that fires on code and stays quiet about prose", "TestTheQuotingGuardFiresOnCodeAndNotOnProse", "internal/guard", ByMutation},
	// 🔴 One test and three mutations, because the reconciliation has three
	// halves that fail differently: the columns, a file that shrank and a file
	// that grew. It closes the loud half of reading twice and not the race
	// itself, which is said where the refusal is built.
	{"a file replaced between the two readings is refused", "TestAFileReplacedBetweenTheTwoReadingsIsRefused", "internal/adapter/sourcecsv", ByMutation},

	// 🔴 Reading a file within a limit, weighed rather than read.
	//
	// Both of these are about memory, so both are proved by weighing what the
	// run allocates. A guard reading the contents back would pass either way,
	// because what changed is not what comes out of the reader but what it
	// costs to get it. Measured on a 38.6 MB manifest: 88.8 MB peak when the
	// buffer grew by doubling against 46.6 MB when the length was asked for
	// first, and 126.6 MB to refuse an oversized file against 15.6 MB once the
	// refusal was decided on the length.
	//
	// The limit each caller carries is untouched by both. What changed is what
	// staying under it, and being turned away by it, cost.
	{"an oversized file is refused on its length rather than on its contents", "TestAnOversizedFileIsRefusedWithoutBeingRead", "internal/adapter/boundedfile", ByMutation},
	{"an accepted file is held once rather than several times over", "TestAnAcceptedFileCostsItsOwnSizeRatherThanSeveral", "internal/adapter/boundedfile", ByMutation},
	// A length is an optimisation and never the guard, so a source that cannot
	// state one is still bounded by the read itself.
	{"a source that cannot state its length is still bounded", "TestASourceThatCannotStateItsLengthIsStillBounded", "internal/adapter/boundedfile", ByMutation},

	// The walk and the writer of a dataset that was read.
	{"a run reading a file can be stopped between rows", "TestARunStoppedBetweenRowsLeavesTheTableUnfinished", "internal/app/corrupt", ByMutation},
	{"a writer for a read dataset takes one table", "TestASecondTableIsRefused", "internal/adapter/format", ByMutation},
	{"a failure reading ends the run rather than the file", "TestAFailureReadingEndsTheRun", "internal/app/corrupt", ByMutation},

	// The manifest of a run that had to guess.
	{"an inference that hides its limits is refused", "TestAnInferenceThatDoesNotSayWhatItMissesIsRefused", "internal/domain/manifest", ByMutation},
	{"a conclusion about a column carries its evidence", "TestAColumnWithNoBasisIsRefused", "internal/domain/manifest", ByMutation},

	// The verb, run as a user runs it.
	//
	// 🔴 The last two guard the only irreversible thing this tool does and the
	// only sentence that became untrue when a second kind of run arrived. Both
	// were found by running the binary rather than by reading it.
	{"every refusal of corrupt is complete and names no recipe", "TestEveryRefusalOfThisVerbIsCompleteAndMentionsNoRecipe", "internal/cli", ByMutation},
	// 🔴 The same rule asked of the function that decides it, because the guard
	// above stopped being able to prove it on 2026-09-03. Every one of the
	// twelve refusals it exercises has since gained a next move of its own, so
	// none falls through to the sentence chosen by kind, and the mutation that
	// turns this verb into one with a recipe changes nothing any of them print.
	// Better for the reader and worse for the proof, so the proof moved. It
	// found a real one on the first run: the sentence for a defect in the tool
	// said it was a defect rather than one in the recipe, to a person who has
	// no recipe.
	{"no sentence corrupt inherits for a kind of failure mentions a recipe", "TestNoSentenceThisVerbInheritsMentionsARecipe", "internal/cli", ByMutation},
	{"damaging a file where it sits leaves the source alone", "TestDamagingAFileInItsOwnDirectoryLeavesTheSourceAlone", "internal/cli", ByMutation},
	{"the evidence for every column reaches the manifest", "TestTheManifestCarriesTheEvidenceForEveryColumn", "internal/cli", ByMutation},
	{"the same file and seed give the same damage", "TestTheSameFileAndSeedGiveTheSameDamage", "internal/cli", ByMutation},
	{"the report of what a run worked out stops growing with the file it describes, and what it leaves out is counted rather than dropped", "TestTheSchemaReportStopsGrowingWithTheFileItDescribes", "internal/cli", ByMutation},
	{"a file of ordinary width has every one of its columns named, so the threshold never becomes a cut", "TestAFileOfOrdinaryWidthHasEveryColumnNamed", "internal/cli", ByMutation},
	{"a dry run does not send its reader to a manifest it will not write", "TestADryRunDoesNotSendTheReaderToAManifestItWillNotWrite", "internal/cli", ByMutation},
	{"a refusal that knows its own next move keeps it, rather than being given the sentence for its kind on top", "TestARefusalThatKnowsTheNextMoveDoesNotGetTheOneForItsKind", "internal/cli", ByMutation},
	// 🔴 The same family, found where the report of 2026-09-02 said to look and
	// left standing there. A file past the size this build reads back is the
	// one refusal whose kind sends the reader to flags, and no flag decides how
	// large a manifest a run writes or how large a record somebody else's
	// application writes. The response record was carrying a move inside its
	// message as well, so the reader got two, the second contradicting the
	// first.
	{"a file too large to read back is refused with a move the reader can make", "TestAFileTooLargeToReadBackIsRefusedWithAMoveThatCanBeMade", "internal/cli", ByMutation},
	// The warning raised when a run writes one, which had no guard at all and
	// named one of the two things that make a manifest grow.
	{"the warning about a manifest too large to read back names both causes", "TestTheWarningAboutAManifestTooLargeToReadBackNamesBothCauses", "internal/cli", ByMutation},
	// 🔴 The same warning, moved to where both surfaces can reach it. It was
	// worked out on the command line alone, so the window said nothing at all
	// and somebody damaging a wide file there met the refusal days later on
	// another screen with nothing connecting it to the run that caused it. The
	// run now carries the fact and each surface words it. Since 2026-09-24 it is
	// worked out once for both modes (DEC-252), and the same table holds the
	// other side of it: a manifest inside the limit, a dataset and an export are
	// never named, however large.
	{"a run says which manifests it wrote will not be read back, and names nothing else", "TestARunSaysWhichFilesWillNotBeReadBack", "internal/damage", ByMutation},
	// 🔴 The account of the columns, bounded on one side and held open on the
	// other. An entry costs about 203 bytes, so 260 000 columns produced a
	// 50.2 MB manifest that verify refused, with not one defect entry in it. A
	// limit that bit on files people really have would be a cut rather than a
	// bound, so the guard checks both sides of it, and the count is what tells
	// a bounded account from a complete one.
	{"the account of the columns stops at a width and says how wide the file was", "TestTheAccountOfTheColumnsStopsAtAWidthAndSaysHowWideTheFileWas", "internal/take", ByMutation},
	{"an account of the columns that cannot say how wide the file was is refused", "TestAnAccountOfTheColumnsThatCannotSayHowWideTheFileWasIsRefused", "internal/domain/manifest", ByMutation},
	// 🔴 The same bound on the third surface, and the one where the medium is
	// worst. A column is six objects there and the panel lays out every one of
	// them on screen or not: measured 1088 bytes per column against about 170
	// on the log, so five thousand columns was 5.4 MB of widgets. The count is
	// shared with the log rather than chosen again, because two numbers would
	// mean the surfaces disagreed about how much of a file they account for.
	{"the panel about a file accounts for as many columns as this build lists and counts the rest", "TestThePanelAccountsForAsManyColumnsAsThisBuildListsAndCountsTheRest", "internal/gui/takeview", ByMutation},

	// --- the fourth part of a refusal, counted rather than repaired ---------
	//
	// 🔴 The family of a refusal carrying its own next move inside its message
	// was closed on 2026-09-02 on the two places somebody had looked at, and it
	// was never counted. This guard was written on 2026-09-03 and found FIFTEEN
	// more in the tree, including two put there earlier the same day by the
	// repair of five others found by hand. A message advising gets the sentence
	// for its kind added underneath, so the reader is handed two next moves and
	// the second is chosen by the kind rather than by the failure.
	{"no refusal advises inside its own message", "TestNoRefusalAdvisesInsideItsOwnMessage", "internal/guard", ByMutation},
	{"the scanner sees a message that advises", "TestTheAdviceGuardSeesAMessageThatAdvises", "internal/guard", ByProbe},
	{"the scanner leaves a statement alone", "TestTheAdviceGuardLeavesAStatementAlone", "internal/guard", ByProbe},
	{"the scanner leaves a move handed over alone", "TestTheAdviceGuardLeavesAMoveHandedOverAlone", "internal/guard", ByProbe},
	{"the vocabulary of generators can be asked for rather than stumbled into, and the list is the register in both directions", "TestHelpNamesEveryGeneratorThisBuildHasAndNothingElse", "internal/cli", ByMutation},
	{"a generator that cannot be produced in a locale says why, rather than being a bare no", "TestHelpSaysWhereAGeneratorCannotBeProducedAndWhy", "internal/cli", ByMutation},
	{"the verb that reports problems does not end its own refusal by telling the reader to run it", "TestValidateDoesNotSendTheReaderToValidate", "internal/cli", ByMutation},

	{"one ordered defect asks the register for its mutator once, so the mutator that runs is the one whose existence was checked", "TestOneOrderedDefectAsksTheRegisterForItsMutatorOnce", "internal/app/corrupt", ByMutation},

	// The mode itself, which both surfaces run one of.
	//
	// 🔴 The second of these guards a failure that has no compiler behind it.
	// A surface turns a refusal into words with a switch, and a switch has a
	// last branch: a kind added to the mode and forgotten at a surface comes
	// out described as some other refusal entirely, in a complete and
	// confident message about the wrong thing. It is proved by a probe rather
	// than by a mutation because what it protects against is an ADDITION, and
	// the fault therefore has to be planted as one.
	{"a setting this build cannot carry out is refused for its own reason", "TestEachUnusableSettingIsRefusedForItsOwnReason", "internal/take", ByMutation},
	{"every refusal of the mode has words of its own at this surface", "TestEveryRefusalOfTheModeHasWordsOfItsOwn", "internal/cli", ByProbe},

	// The screen for that mode.
	//
	// 🔴 The third of these is a rule written before anything can break it. The
	// catalog carries no security module today, so the guard passes with
	// nothing to look at, and the day one arrives it fails. Written afterwards
	// it would be written by somebody looking at a screen that already offers
	// it as one tick box among the others, which is the thing the rule forbids.
	{"every refusal of the mode has words on the screen too",
		"TestEveryRefusalOfTheModeHasWordsOnThisScreen", "internal/gui/takeview", ByMutation},
	{"a dropped file reaches the screen a person is looking at",
		"TestADroppedFileReachesTheScreenOnDisplay", "internal/gui/shell", ByMutation},
	{"the module that must be chosen deliberately is never one tick box among others",
		"TestNothingChosenDeliberatelyIsOfferedAsOneTickBoxAmongOthers",
		"internal/gui/takeview", ByProbe},

	// The screen writing, which arrived on 2026-08-19 with the state that shows
	// a run going on and the way to end one.
	//
	// 🔴 The one below them, which lives in the mode rather than on the screen,
	// is the one worth reading. Everything here would fail loudly, and that one
	// guards a way of failing quietly: a run made after a stopped one damaging
	// the records that were left rather than the file, and counting the whole
	// file in its manifest. Nothing about the result looks wrong, and the
	// command line cannot reach it at all.
	{"the button damages the file and the screen says what it left behind",
		"TestTheScreenDamagesTheFileAndSaysWhatItWrote", "internal/gui/takeview", ByMutation},
	{"the screen says what a run would do before anybody asks for one, and writes nothing",
		"TestTheScreenSaysWhatARunWouldDoBeforeItRunsAnything", "internal/gui/takeview", ByMutation},
	{"every setting on the screen reaches the run rather than only the control",
		"TestEverySettingOnTheScreenReachesTheRun", "internal/gui/takeview", ByMutation},
	{"the share of records on the screen decides how much of the file is damaged",
		"TestTheShareOfRecordsDecidesHowMuchIsDamaged", "internal/gui/takeview", ByMutation},
	{"files already there are replaced only when the screen was told they may be",
		"TestFilesAlreadyThereAreOnlyReplacedWhenTheSettingSaysSo",
		"internal/gui/takeview", ByMutation},
	{"a run somebody stops leaves nothing behind and the screen says so",
		"TestARunSomebodyStopsLeavesNothingBehind", "internal/gui/takeview", ByMutation},
	{"a run made after a stopped one damages the whole file rather than the rest of it",
		"TestARunMadeAfterAStoppedOneDamagesTheWholeFile", "internal/take", ByMutation},
	{"nothing on the screen can be changed while a run is going on",
		"TestTheSettingsAreOutOfReachWhileARunIsGoingOn", "internal/gui/takeview", ByMutation},
	{"asking for the window during a run stops the run before the process ends",
		"TestAskingForTheWindowDuringARunStopsItFirst", "internal/gui/takeview", ByMutation},
	{"a separator that is briefly unusable does not take the open file away, and the next usable one brings it back",
		"TestASeparatorThatIsBrieflyUnusableDoesNotTakeTheFileAway", "internal/gui/takeview", ByMutation},
	{"no screen holds a control the toolkit floods with the focus colour, since the words written across one measure 1.56 and nothing outside its renderer repairs that", "TestNoScreenHoldsAControlTheToolkitFloods", "internal/gui/screens", ByMutation},
	{"a chooser is outlined in the colour and the thickness the tokens say", "TestTheChooserIsOutlinedFromTheTokens", "internal/gui/parts", ByMutation},
	{"every screen in the sidebar carries an icon, in BOTH copies of that list, since only one of them is the one anybody runs", "TestEverySidebarScreenCarriesAnIcon", "internal/guard", ByMutation},
	{"the name of the product and its promise reach the screen, and neither is drawn when the frame was given none", "TestTheProductSaysWhatItIsAndWhatItPromises", "internal/gui/shell", ByMutation},
	{"the edge a face draws lands wholly inside its control, rather than half of it on whatever the control stands on", "TestTheEdgeOfAFaceIsDrawnInsideTheControl", "internal/gui/parts", ByMutation},
	{"a card of rows never asks for more height than the layout was given, since nothing here clips a child that does", "TestACardOfRowsNeverGrowsPastTheRoomItWasGiven", "internal/gui/paint", ByMutation},
	{"a card of rows still spends spare height on its rows, up to the ceiling the scale sets", "TestACardOfRowsStillGrowsWithTheWindow", "internal/gui/paint", ByMutation},
	{"a chooser nobody can use draws no edge, since an edge says a control can be used", "TestADisabledChooserShowsNoEdge", "internal/gui/parts", ByMutation},
	{"a chooser with the keyboard on it gains a mark that is not its resting edge", "TestTheKeyboardOnTheChooserIsVisible", "internal/gui/parts", ByMutation},
	{"a chooser keeps its width when something is chosen, so the row it stands in does not move", "TestTheChooserDoesNotChangeWidthWhenSomethingIsChosen", "internal/gui/parts", ByMutation},
	{"a chooser opens on Enter as well as on the keys the toolkit answers, and still walks its options with the arrows", "TestTheChooserOpensOnEnterAsWellAsOnSpace", "internal/gui/parts", ByMutation},
	{"a control somebody can use carries an edge of its own, so it is a control rather than a word", "TestEveryPressableControlDrawsItsOwnEdge", "internal/gui/screens", ByMutation},
	{"a register of which values take account of a consistency role says only what the values really do", "TestEveryGeneratorReadsExactlyTheRolesTheRegisterSays", "internal/adapter/values", ByMutation},
	{"a role that register names is a role a recipe can bind", "TestARoleTheRegisterNamesIsARoleThatExists", "internal/adapter/values", ByMutation},
	{"the screen that produces one value produces it by choosing, and copying it reaches the clipboard", "TestChoosingAValueProducesOneAndCopyingItReachesTheClipboard", "internal/gui/quickview", ByMutation},
	{"every control on that screen reaches the value", "TestEveryControlOnTheQuickScreenReachesTheValue", "internal/gui/quickview", ByMutation},
	{"a field is on that screen only beside a value that takes account of it", "TestAFieldIsShownOnlyBesideAValueThatReadsIt", "internal/gui/quickview", ByMutation},
	{"one value is refused rather than produced when what was bound to it is not read, not a gender, or not a date", "TestQuickRefusesWhatItCannotProduce", "internal/quick", ByMutation},
	{"the value one verb writes is the value a recipe of the same seed produces", "TestTheValueIsTheOneARecipeWouldProduce", "internal/quick", ByMutation},
	{"every spelling of a gender produces the value the canonical one does, in every reader of it", "TestEverySpellingOfAGenderProducesTheValueTheCanonicalOneDoes", "internal/adapter/values", ByMutation},
	{"a gender nobody can read is refused by the scheme and drawn around by a word list, which is the one place the readers differ", "TestAnUnreadableGenderIsRefusedByTheSchemeAndDrawnAroundByAWordList", "internal/adapter/values", ByMutation},
	{"a field the producer did not read back is not a field that survived", "TestAFieldNobodyReadBackIsNotAFieldThatSurvived", "internal/domain/response", ByMutation},
	{"a whole parameter is read from each of the three types it can arrive as", "TestAWholeNumberIsReadFromEachOfTheThreeTypesItCanArriveAs", "internal/domain/mutator", ByMutation},
	{"a run counts the records carrying damage as the manifest describes them", "TestWhatTheRunCountedAgreesWithWhatItWroteToTheManifest", "internal/adapter/corrupted", ByMutation},
	{"every manifest document of one run is handed every entry", "TestEverySinkIsHandedEveryEntryInTheOrderTheSinksWereGiven", "internal/adapter/corrupted", ByMutation},
	{"the shortfall of a run is what was ordered less what reached the data", "TestTheShortfallIsWhatWasOrderedLessWhatReachedTheData", "internal/damage", ByMutation},
	{"every kind of refused damage keeps a sentence a person can read", "TestTheShortfallIsBrokenDownByKindAndAccountsForEveryOrderThatFailed", "internal/adapter/corrupted", ByMutation},
	{"a run exporting a manifest form announces the file it writes", "TestEveryFormThisBuildExportsIsNamedAmongTheFilesARunWrites", "internal/damage", ByMutation},
	{"a transposition is never reported as damage over a value its scheme still accepts", "TestNoSchemeIsEverToldToRejectAValueItStillAccepts", "internal/adapter/mutators", ByMutation},
	{"a value that was already broken is not credited to a swap", "TestWhatThisRefusesAndTheSentenceItRefusesWith", "internal/adapter/mutators", ByMutation},
	{"the run seed reaches the mutator that draws a pair with it", "TestOneSeedDrawsOnePairAndDifferentSeedsDoNotAllDrawTheSameOne", "internal/adapter/mutators", ByMutation},
	{"a pair holding one digit twice is left out of the draw rather than drawn and refused", "TestAPairHoldingOneDigitTwiceIsNeverWhatGetsDrawn", "internal/adapter/mutators", ByMutation},
	{"a value in a form its scheme does not check is passed over rather than damaged", "TestAFormTheSchemeDoesNotCheckIsNotDamagedInTheFirstPlace", "internal/adapter/mutators", ByMutation},
	{"a swap landing in a form the scheme does not check is not called a rejection", "TestASwapLandingInAFormTheSchemeDoesNotCheckIsNotCalledARejection", "internal/adapter/mutators", ByMutation},
	{"the randomness a run was seeded with reaches the mutator that draws with it", "TestTheRunSeedReachesTheMutator", "internal/app/corrupt", ByMutation},
	{"a field takes no more characters than the surface can carry, so a paste stops at the limit rather than at a refusal one press later", "TestABoundedFieldStopsAtTheLengthItWasGiven", "internal/gui/parts", ByMutation},
	{"a field takes no character that could never be part of its answer", "TestABoundedFieldKeepsOnlyTheCharactersItTakes", "internal/gui/parts", ByMutation},
	{"every door a seed arrives through refuses one longer than the surfaces take, since a rule held at one door and not the next is this project's own failure mode", "TestEveryDoorASeedArrivesThroughRefusesOneTooLong", "internal/cli", ByMutation},
	{"a row under the pointer draws something else, since the chooser once went on receiving the event while drawing the same picture", "TestARowUnderThePointerDrawsSomethingElse", "internal/gui/parts", ByMutation},
	{"clicking one value copies that value rather than the set, since rows are built in a loop and a captured variable would give every row the last one", "TestClickingOneValueCopiesThatValueAndNotTheSet", "internal/gui/quickview", ByMutation},
	{"a recipe with a policy is damaged on its way to the disk and leaves a manifest beside the data, whichever surface runs it", "TestARecipeWithAPolicyIsDamagedOnTheWayAndLeavesAManifest", "internal/produce", ByMutation},
	{"an extra form asked of a recipe without a policy is reported as unused, and only then, so a surface can say why no manifest was written", "TestAnExtraFormWithoutAPolicyWritesNothingAndSaysSo", "internal/produce", ByMutation},
	{"a recipe with a policy run from the recipe screen is damaged with a manifest beside the data, and the card after the run says what the damage did", "TestARecipeWithAPolicyIsDamagedByTheScreenAndItSaysWhatTheDamageDid", "internal/gui/recipeview", ByMutation},
	{"an extra form of the manifest picked for a recipe without a policy is said on the card to have nothing to export, before the run and after it", "TestAnExtraFormOnARecipeWithoutAPolicySaysItHasNothingToExport", "internal/gui/recipeview", ByMutation},
	{"the first entry on the list of extra forms of the manifest gives a run no form at all, rather than a form called by the word for none", "TestTheFirstFormOnTheListIsNoFormAtAll", "internal/gui/damageparts", ByMutation},
	{"the plan of a take and corrupt run splits the damage ordered and carried out by layer, in the order the layers were named, and the rows sum to the totals", "TestThePlanSplitsWhatItWouldDoByLayerInTheOrderTheLayersWereNamed", "internal/take", ByMutation},
}
