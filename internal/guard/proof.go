package guard

// The register of protected behaviours and of how we know each guard can fail.
//
// # Why a list of guards needs its own guard
//
// A green test suite says nothing about a behaviour nobody guards. The only
// proof a guard works is the sentence "this test would be red if I broke the
// thing it protects", checked by breaking it. That check lives in a script
// outside this repository, and a script whose list is kept by hand is a script
// somebody will one day forget to extend. This project wrote that rule down
// before it had a way to keep it.
//
// So the register closes the loop from the other end. Every guard is in
// exactly one of three states, and which one is visible rather than assumed.
// The state that matters most is the third: a guard nobody has proven is
// listed as such instead of looking the same as a proven one.
//
// # What this closes and what it does not
//
// Every test in this package is enumerated mechanically, so a guard added here
// without an entry fails the build. Guards elsewhere are listed by hand, and a
// new one added without an entry is still invisible. That is the honest limit
// of this mechanism and it is written here rather than left to be discovered.

// ProofState is how we know a guard can go red.
type ProofState string

const (
	// ByMutation means the mutation list carries an entry that breaks the
	// protected behaviour, and the named test goes red when it runs. This is
	// the strongest of the three.
	ByMutation ProofState = "mutation"
	// ByProbe means there is no product code under the guard to break. It
	// reads the tree, compares two lists or checks a document, so the fault
	// is planted in a fixture inside the test itself.
	ByProbe ProofState = "probe"
	// Unproven means nobody has shown this one can fail. Said out loud so
	// that it does not read like the other two. This list may only shrink.
	Unproven ProofState = "unproven"
)

// Guard is one protected behaviour together with the test that protects it.
type Guard struct {
	// Behaviour is what stops being true if this guard is removed. One
	// sentence, in the present tense, about the product rather than about the
	// test.
	Behaviour string
	// Test is the name of the function that goes red.
	Test string
	// Package is where that function lives, relative to the module root.
	Package string
	// Proof is how we know it can go red.
	Proof ProofState
}

// UnprovenCeiling is a ratchet. It may only be lowered.
//
// Raising it to make a run green is the same act as editing a golden value for
// the same reason. Lowering it is ordinary work.
//
// It stood at one until the window existed, and the reason was written here
// rather than left to be rediscovered: the rule that the command line binary
// never reaches the window could not be broken on purpose while there was no
// window to reach, so a mutation adding that import reported a broken tree
// instead of a caught fault. The condition for the return was stated as the
// day the graphical half exists, that day came with the theme package, and
// the mutation now compiles and is caught. This is what a condition for a
// return is for.
const UnprovenCeiling = 0

// Guards is the register.
//
// It is deliberately not every test in the project. A test that checks one
// function does its arithmetic is not a guard. A guard is a test standing
// between the product and a promise it makes to somebody outside it.
var Guards = []Guard{
	// --- the architecture guards, all of them in this package ---------------
	{"every refusal either side of the boundary between exit code 3 and 5 says why it is there", "TestEveryRefusalEitherSideOfTheBoundarySaysWhy", "internal/guard", ByMutation},
	{"a reason in that register is an argument rather than the kind said twice", "TestEveryReasonInTheRegisterSaysSomething", "internal/guard", ByMutation},
	{"no package can reach outside this machine", "TestRepositoryOpensNothing", "internal/guard", ByMutation},
	{"the scanner sees an import that reaches outside", "TestNetworkCatchesAnImportThatReachesOutside", "internal/guard", ByProbe},
	{"parsing an address is not talking to one", "TestNetworkAllowsAddressParsing", "internal/guard", ByProbe},
	{"network code outside the standard library is caught too", "TestTheNetworkGuardSeesNetworkPackagesOutsideTheStandardLibrary", "internal/guard", ByProbe},
	{"a name that merely looks like a network package is not a finding", "TestTheNetworkGuardDoesNotFireOnNamesThatMerelyLookAlike", "internal/guard", ByProbe},
	{"the scanner reports a dependency that reaches outside", "TestNetworkReportsADependencyThatReachesOutside", "internal/guard", ByProbe},
	{"the limit of the import scan is stated where it runs", "TestTheNetworkGuardOnlySeesImportsWrittenHere", "internal/guard", ByProbe},
	{"the dependency rule holds across the whole tree", "TestRepositoryRespectsTheLayers", "internal/guard", ByMutation},
	{"the scanner sees the domain reaching upwards", "TestLayersCatchesTheDomainReachingUpwards", "internal/guard", ByMutation},
	{"the scanner sees a use case reaching an adapter", "TestLayersCatchesAUseCaseReachingAnAdapter", "internal/guard", ByMutation},
	{"the scanner sees the base package depending on ours", "TestLayersCatchesTheBasePackageDependingOnUs", "internal/guard", ByProbe},
	{"a package may be split into files without the rule noticing", "TestLayersAllowsAPackageToImportItsOwnSubpackage", "internal/guard", ByProbe},
	{"an indirect import is found, not only a direct one", "TestReachFindsAnIndirectPath", "internal/guard", ByProbe},
	{"the command line binary never pulls in the window", "TestCommandLineBinaryNeverReachesTheWindow", "internal/guard", ByMutation},
	{"the window is present enough for that isolation to mean something", "TestTheWindowIsPresentEnoughForTheIsolationToMeanSomething", "internal/guard", ByProbe},
	{"the guards themselves never ship inside the product", "TestGuardsShipWithNoBinary", "internal/guard", ByMutation},
	{"the independent validators never ship inside the product", "TestTheIndependentValidatorsShipWithNoBinary", "internal/guard", ByMutation},
	{"a test file importing a test only package is not a finding", "TestTestOnlyIgnoresTestFiles", "internal/guard", ByProbe},
	{"a rule about a package that is gone is reported, not passed", "TestHasSeesTheDifferenceBetweenAbsentAndClean", "internal/guard", ByProbe},
	{"the import scan actually read the tree from disk", "TestGuardsAreActuallyLoadedFromDisk", "internal/guard", ByProbe},
	{"no concurrency enters the product without a decision", "TestTheProductStartsNoConcurrentWorkWithoutSayingSo", "internal/guard", ByMutation},
	{"the scanner sees every way of starting concurrent work", "TestConcurrencyNoticesEachWayOfStartingWork", "internal/guard", ByMutation},
	{"the scanner reads syntax rather than prose", "TestConcurrencyStaysQuietAboutProse", "internal/guard", ByMutation},
	{"a file on the allow list is exempt and nothing else is", "TestConcurrencyRespectsTheAllowList", "internal/guard", ByProbe},
	{"the concurrency scan actually reached the product", "TestTheConcurrencyScanReachesTheProduct", "internal/guard", ByProbe},

	// --- the one exception to the ban on reaching the operating system -------
	//
	// The termination signal has no name outside package syscall, and that
	// package is refused everywhere because it can do whatever the system can.
	// The allowance is therefore a file and a list of identifiers rather than a
	// package, and these guards close it from both ends.
	{"the operating system is not reachable outside the one name that is allowed", "TestTheProductNamesSyscallOnlyWhereItIsAllowed", "internal/guard", ByMutation},
	{"the exception covers one package in the network rule and no more", "TestTheNetworkRuleStillRefusesSyscallOutsideTheAllowedPackage", "internal/guard", ByMutation},
	{"the scanner sees a file nobody allowed", "TestSyscallScannerSeesAFileNobodyAllowed", "internal/guard", ByProbe},
	{"the scanner sees a name outside what the allowance covers", "TestSyscallScannerSeesANameOutsideTheAllowance", "internal/guard", ByProbe},
	{"an allowance for a file that stopped using it is reported rather than left open", "TestSyscallAllowanceNeverNamesAFileThatDoesNotUseIt", "internal/guard", ByProbe},
	{"a test may reach the system where the product may not", "TestSyscallScannerIgnoresTests", "internal/guard", ByProbe},
	{"the scan actually reached the product", "TestTheSyscallScanReachesTheProduct", "internal/guard", ByProbe},

	// --- two runs into one directory ----------------------------------------
	//
	// The package promises that a file which already exists stops the run. It
	// used to hold only for a file present when the run started, because
	// asking and renaming are two moments with a whole run in between.
	{"a second run cannot take a name the first one is writing", "TestASecondRunCannotTakeANameTheFirstIsWriting", "internal/adapter/store", ByMutation},
	{"the marker does not outlive the run that took it", "TestTheMarkerIsGoneOnceTheResultIsInPlace", "internal/adapter/store", ByMutation},
	{"a discarded run leaves nothing that blocks the next one", "TestADiscardedRunLeavesNoMarkerBehind", "internal/adapter/store", ByMutation},

	// --- the window, whose values may only live in one package --------------
	//
	// These stand before the first screen rather than after it. Written
	// afterwards, the literal guard would have had to tolerate whatever that
	// screen happened to do, and the tolerance would have looked like a rule.
	// The first three are probes because the only package under the window
	// today is the theme itself, which is the one place exempt, so there is no
	// product code for a mutation to break yet.
	{"the window takes every colour and every distance from the theme", "TestTheWindowTakesEveryValueFromTheTheme", "internal/guard", ByProbe},
	{"the scanner sees a value the window wrote for itself", "TestTheLiteralGuardFiresOnTheViewAndNotOnTheTheme", "internal/guard", ByProbe},
	{"the scanner sees a layout and a widget the window wrote for itself", "TestTheHandmadeGuardSeesBothKindsAndIsLoudWhenThereIsNothingToRead", "internal/guard", ByProbe},
	{"a window that is gone is reported rather than read as clean", "TestTheLiteralGuardIsLoudWhenTheWindowIsGone", "internal/guard", ByProbe},
	{"the theme is exempt by the list and not by its name", "TestTheThemeIsExemptByTheListAndNotByItsName", "internal/guard", ByProbe},
	{"a spacing step survives the scaling factors a desktop applies", "TestEveryStepSurvivesTheCommonScalingFactors", "internal/gui/theme", ByMutation},
	{"the scale never gives one distance two names", "TestEveryStepOfTheScaleIsUsedOnce", "internal/gui/theme", ByMutation},
	{"a colour of the palette is never translucent, so its measured contrast is about what is shown", "TestEveryColourIsFullyOpaque", "internal/gui/theme", ByMutation},
	{"the background is the reference of the measurement and never one of its subjects", "TestEveryNameInThePaletteIsUsedOnce", "internal/gui/theme", ByMutation},
	{"a colour constant and a palette entry answer to each other", "TestEveryColourNameAndEntryAnswerToEachOther", "internal/gui/theme", ByMutation},

	// --- the window never asks the toolkit to leave this machine ------------
	//
	// The import scan answers about our own packages. The toolkit is one
	// import carrying net/http inside it, so that scan cannot see this. The
	// owner settled on 2026-08-12 that the unbreakable rule is about what the
	// product does rather than about what its closure contains, and a promise
	// about behaviour needs a guard about behaviour.
	{"nothing we wrote asks the toolkit to leave this machine", "TestNothingInTheWindowAsksTheToolkitToLeaveThisMachine", "internal/guard", ByMutation},
	{"the scanner sees a call that leaves the machine", "TestTheToolkitGuardSeesACallThatLeavesTheMachine", "internal/guard", ByProbe},
	{"a window that is gone is reported rather than read as clean", "TestTheToolkitGuardIsLoudWhenTheWindowIsGone", "internal/guard", ByProbe},
	{"every banned call records why it is banned", "TestEveryBannedToolkitCallSaysWhyItIsBanned", "internal/guard", ByProbe},

	// --- what the window says, and in which language -------------------------
	{"the About screen carries every fact the licence flag prints", "TestTheScreenCarriesEveryFactTheLicenceFlagPrints", "internal/gui/about", ByMutation},
	{"the About screen carries all four versions", "TestTheScreenCarriesAllFourVersions", "internal/gui/about", ByMutation},
	{"no label renders as its own translation key", "TestNoLabelRendersAsItsOwnKey", "internal/gui/about", ByMutation},
	{"a translation key and a catalogue entry answer to each other", "TestEveryKeyAndEveryEntryAnswerToEachOther", "internal/adapter/i18n", ByMutation},
	{"interface text keeps the punctuation this repository uses", "TestNoEntryUsesPunctuationThisRepositoryRefuses", "internal/adapter/i18n", ByMutation},
	{"every colour the window shows comes from the measured palette", "TestEveryColourAnsweredHereComesFromTheMeasuredPalette", "internal/gui/fynetheme", ByMutation},
	{"a colour nobody claimed is visible rather than a hole", "TestAnUnclaimedColourFallsThroughAndIsNeverInvisible", "internal/gui/fynetheme", ByMutation},
	{"nothing a control is painted with equals the colour that says where the keyboard is", "TestNoBackgroundOfAControlIsTheColourOfFocus", "internal/gui/fynetheme", ByMutation},
	{"the keyboard meets the names of the screens before the screen itself", "TestTheKeyboardMeetsTheNamesBeforeTheScreen", "internal/gui/shell", ByMutation},
	{"the screen a person is on is marked by a shape rather than by colour alone", "TestTheOpenScreenIsMarkedByAShapeRatherThanByColourAlone", "internal/gui/shell", ByMutation},
	{"the keyboard meets the controls of a screen before its entries", "TestTheKeyboardMeetsTheButtonsBeforeTheEntries", "internal/gui/manifestview", ByMutation},
	{"the cell the keyboard stands on is marked, and the mark leaves with the focus", "TestTheCellTheKeyboardStandsOnIsMarkedAndTheMarkLeavesWithIt", "internal/gui/manifestview", ByMutation},
	{"the mark of the keyboard is drawn rather than merely recorded", "TestTheMarkOfTheKeyboardReachesThePixels", "internal/gui/manifestview", ByMutation},
	{"the keyboard meets the file before everything the file is used for", "TestTheKeyboardMeetsTheFileBeforeEverythingItIsUsedFor", "internal/gui/takeview", ByMutation},
	{"the keyboard follows the two buttons of a run as they take each other's place", "TestTheKeyboardFollowsTheButtonsOfARun", "internal/gui/takeview", ByMutation},
	{"a run leaves nobody standing on a setting it has put out of reach", "TestARunDoesNotLeaveTheKeyboardOnSomethingItPutOutOfReach", "internal/gui/takeview", ByMutation},
	{"the keyboard is given back to somebody who started a run with it and then reached for the mouse", "TestTheKeyboardIsGivenBackToSomebodyWhoReachedForTheMouse", "internal/gui/takeview", ByMutation},
	{"a run made with the mouse leaves the keyboard where the mouse left it", "TestARunStartedWithTheMouseLeavesTheKeyboardAlone", "internal/gui/takeview", ByMutation},
	{"somebody who walks away from a run they started is not walked back", "TestSomebodyWhoWalksAwayFromARunIsNotWalkedBack", "internal/gui/takeview", ByMutation},
	{"the keyboard has somewhere to go when the control that arrived cannot take it", "TestTheKeyboardHasSomewhereToGoWhenTheButtonCannotTakeIt", "internal/gui/takeview", ByMutation},
	{"the keyboard can be seen on every kind of control this screen carries", "TestTheKeyboardCanBeSeenOnEveryKindOfControlHere", "internal/gui/takeview", ByMutation},
	{"a manifest opened from outside reaches the same state as the chooser leaves behind", "TestOpeningAManifestFromOutsideReachesTheSameStateAsTheChooser", "internal/gui/manifestview", ByMutation},
	{"a response record with no manifest beside it is refused rather than drawn", "TestAResponseRecordWithNoManifestIsRefused", "internal/gui/manifestview", ByMutation},
	{"every state of every screen still draws what it drew before", "TestEveryScreenStillDrawsWhatItDrewBefore", "internal/gui/screens", ByMutation},
	{"every state on the list can be built from the sample, so none goes unrendered", "TestEveryStateCanBeBuiltFromTheSample", "internal/gui/screens", ByMutation},
	{"a state that needs a manifest is refused without one rather than drawn empty", "TestAStateThatNeedsAManifestIsRefusedWithoutOne", "internal/gui/screens", ByMutation},
	{"a button answers to Enter as well as to the space bar", "TestAButtonIsPressedByEnterAndBySpace", "internal/gui/parts", ByMutation},
	{"the keyboard reaches a scrolling area, which the toolkit's own never allows", "TestTheKeyboardReachesAScrollingArea", "internal/gui/parts", ByMutation},
	{"every key a person expects to drive a scrolling area with drives it", "TestTheSixKeysDriveAScrollingArea", "internal/gui/parts", ByMutation},
	{"a page keeps the line somebody was reading rather than jumping past it", "TestAPageKeepsTheLineSomebodyWasReading", "internal/gui/parts", ByMutation},
	{"a scrolling area with nothing to scroll stays put", "TestAScrollingAreaWithNothingToScrollStaysPut", "internal/gui/parts", ByProbe},
	{"the frame says where the keyboard is, and stops saying it when it leaves", "TestTheFrameSaysWhereTheKeyboardIs", "internal/gui/parts", ByMutation},
	{"the frame is thick enough to arrive on the screen", "TestTheFrameIsThickEnoughToBeSeen", "internal/gui/parts", ByMutation},
	{"a button nobody can press with the pointer stays unpressable from the keyboard", "TestADisabledButtonIgnoresEnter", "internal/gui/parts", ByMutation},
	{"the keyboard can leave the field a recipe is written in", "TestTheKeyboardCanLeaveTheEditor", "internal/gui/parts", ByMutation},
	{"the screen judges the recipe that is actually written on it", "TestTheScreenJudgesTheRecipeWrittenOnIt", "internal/gui/recipeview", ByMutation},
	{"an answer about a recipe nobody has on screen any more is not drawn", "TestAnAnswerAboutARecipeNobodyIsWaitingForIsNotDrawn", "internal/gui/recipeview", ByMutation},
	{"an empty screen is not an approved one", "TestAnEmptyScreenIsNotAnApprovedOne", "internal/gui/recipeview", ByMutation},
	{"a drawn problem carries the place and the line it belongs to", "TestTheDrawnAnswerCarriesThePlaceAndTheLine", "internal/gui/recipeview", ByMutation},
	{"the screen says it is judging a recipe while it is", "TestTheScreenSaysThatItIsChecking", "internal/gui/recipeview", ByMutation},
	{"both panes of the take and corrupt screen can be read without a mouse", "TestBothPanesOfThisScreenCanBeReadWithoutAMouse", "internal/gui/takeview", ByMutation},
	{"reading somebody's file happens somewhere other than the thread that draws, and the screen says so", "TestReadingAFileHappensSomewhereOtherThanTheThreadThatDraws", "internal/gui/takeview", ByMutation},
	{"nothing damages a file while the description of it is being replaced", "TestNoRunBeginsWhileTheFileIsBeingReadAgain", "internal/gui/takeview", ByMutation},
	{"an answer about a file nobody is waiting for any more is not drawn", "TestAnAnswerAboutAFileNobodyIsWaitingForIsNotDrawn", "internal/gui/takeview", ByMutation},
	{"the system asking this program to stop ends the run before the window is given up", "TestTheSystemAskingThisProgramToStopEndsTheRunFirst", "internal/gui/takeview", ByMutation},
	{"a run that ends on its own gives the two signals back", "TestARunThatEndsOnItsOwnGivesTheSignalsBack", "internal/gui/takeview", ByMutation},
	{"a signal arriving after a run gives the window up rather than arming the next run to close it", "TestASignalThatArrivesAfterTheRunGivesTheWindowUpAtOnce", "internal/gui/takeview", ByMutation},
	{"reading a document says so on the browser and puts its controls out of reach", "TestReadingADocumentSaysSoAndPutsTheControlsOutOfReach", "internal/gui/manifestview", ByMutation},
	{"an answer about a document nobody is waiting for any more is not drawn", "TestAnAnswerAboutADocumentNobodyIsWaitingForIsNotDrawn", "internal/gui/manifestview", ByMutation},
	{"a watch released twice does not take the window down with it", "TestReleasingAWatchTwiceIsSafe", "internal/gui/background", ByMutation},
	{"a stopper answers the same thing however often it is asked", "TestAStopperAnswersTheSameThingHoweverOftenItIsAsked", "internal/gui/background", ByMutation},
	// There is a third test in that package, TestARealSignalReachesTheWindow,
	// and it is deliberately absent for the same reason its twin on the
	// command line is. It is the only one proving the runtime delivers the
	// signals this window asks for, and it cannot run on Windows, where the
	// mutation runner runs. An entry claiming it was proven by mutation would
	// be a sentence nobody has checked.
	{"a response record opened as a manifest is told apart rather than judged as one", "TestAResponseRecordOpenedAsAManifestIsToldApart", "internal/adapter/manifestjson", ByMutation},

	// --- the register guarding the register ---------------------------------
	{"a guard added here cannot stay out of the register", "TestEveryGuardInThisPackageIsRegistered", "internal/guard", ByMutation},
	{"the register never names a test that is gone", "TestEveryRegisteredTestExistsAndIsNamedOnce", "internal/guard", ByMutation},
	{"the count of unproven guards can only fall", "TestTheUnprovenListOnlyShrinks", "internal/guard", ByMutation},

	// --- the anchors, which say what holds up a rule taken from outside -----
	{"a provider cannot exist without something outside this project supporting its rules", "TestEveryProviderInTheTreeCarriesAnAnchor", "internal/guard", ByMutation},
	{"the scanner sees a provider nobody anchored", "TestAnchorScannerSeesAProviderNobodyAnchored", "internal/guard", ByProbe},
	{"the anchor register never names a provider that is gone", "TestEveryAnchorNamesAProviderThatExists", "internal/guard", ByMutation},
	{"the scanner sees an anchor pointing nowhere", "TestAnchorScannerSeesAnEntryPointingNowhere", "internal/guard", ByProbe},
	{"claiming the source worked the example requires the test that reproduces it", "TestEveryWorkedExampleAnchorNamesATestThatExists", "internal/guard", ByMutation},
	{"a rule measured once says how many values and when, so it can be measured again", "TestEveryMeasurementAnchorCanBeRepeated", "internal/guard", ByMutation},
	{"the scanner sees a measurement nobody could repeat", "TestAnchorScannerSeesAMeasurementNobodyCouldRepeat", "internal/guard", ByProbe},
	{"a rule called ours by construction says which source was asked and left it open", "TestARuleCalledOursHasToSayWhatItAsked", "internal/guard", ByProbe},
	{"the count of rules resting on nothing but our own reading can only fall", "TestTheUnanchoredListOnlyShrinks", "internal/guard", ByMutation},

	// --- determinism, the foundation everything else stands on --------------
	{"the derived bytes have not moved since they were frozen", "TestTheDerivationHasNotMoved", "internal/domain/seed", ByMutation},
	{"the message fed to the hash is built the way the contract says", "TestTheMessageIsBuiltTheWayTheContractSays", "internal/domain/seed", ByMutation},
	{"asking about one place does not disturb another", "TestAskingAboutMorePlacesLeavesTheOthersAlone", "internal/domain/seed", ByMutation},
	{"two places whose names concatenate alike get different answers", "TestPlacesThatConcatenateAlikeStayApart", "internal/domain/seed", ByMutation},
	{"a bounded draw does not lean towards the low end", "TestIntNDoesNotLeanTowardsLowValues", "internal/domain/seed", ByMutation},
	{"deriving a value allocates nothing", "TestDerivingAValueDoesNotAllocateMoreThanItDid", "internal/domain/seed", ByMutation},

	// --- the recipe, which arrives from somebody else's repository ----------
	{"every problem in a recipe is reported at once", "TestEveryProblemIsReportedAtOnce", "internal/adapter/recipeyaml", ByMutation},
	{"each declared rule of the recipe is enforced", "TestProblems", "internal/adapter/recipeyaml", ByMutation},
	{"the order of fields is the order the user wrote", "TestFieldOrderIsTheOrderTheUserWrote", "internal/adapter/recipeyaml", ByMutation},
	{"what was not checked is reported rather than passed over", "TestGeneratorSettingsAreCarriedAndReportedAsUnchecked", "internal/adapter/recipeyaml", ByMutation},
	{"a recipe too large to read is refused before it reaches the parser", "TestARecipeTooLargeToReadIsRefusedBeforeParsing", "internal/adapter/recipeyaml", ByMutation},
	{"a recipe nested too deeply is refused before it reaches the parser", "TestARecipeNestedTooDeeplyIsRefusedBeforeParsing", "internal/adapter/recipeyaml", ByMutation},
	{"a recipe saved by Notepad is read rather than refused", "TestARecipeSavedByNotepadIsRead", "internal/adapter/recipeyaml", ByMutation},
	{"a file holding two recipes is refused rather than half read", "TestAFileHoldingTwoRecipesIsRefusedRatherThanHalfRead", "internal/adapter/recipeyaml", ByMutation},
	{"the layouts that are still one recipe keep working", "TestTheLayoutsThatAreStillOneRecipeKeepWorking", "internal/adapter/recipeyaml", ByMutation},
	{"a long value does not come back as a long message", "TestALongValueDoesNotComeBackAsALongMessage", "internal/adapter/recipeyaml", ByMutation},
	{"shortening a message never splits a character", "TestShorteningAMessageNeverSplitsACharacter", "internal/adapter/recipeyaml", ByMutation},
	{"a value that is not a number cannot switch off the checks that compare it", "TestAShareThatIsNotANumberIsRefused", "internal/adapter/recipeyaml", ByMutation},
	{"the nesting limit covers braces as well as square brackets", "TestARecipeNestedInBracesIsRefusedBeforeParsing", "internal/adapter/recipeyaml", ByMutation},
	{"nesting is counted across both kinds of bracket together", "TestNestingIsCountedAcrossBothKindsOfBracket", "internal/adapter/recipeyaml", ByMutation},
	{"a recipe in an encoding this build cannot read is told which one it is in", "TestARecipeInTheWrongEncodingIsNamedRatherThanPuzzledOver", "internal/adapter/recipeyaml", ByMutation},
	{"a recipe in a single byte encoding is refused before its names are damaged", "TestARecipeInASingleByteEncodingIsRefused", "internal/adapter/recipeyaml", ByMutation},
	{"accented names survive a recipe written in UTF-8", "TestTheSameRecipeInUTF8KeepsItsAccentedNames", "internal/adapter/recipeyaml", ByMutation},
	{"a number too large to hold is named rather than wrapped into a negative one", "TestANumberTooLargeToHoldIsNamedRatherThanWrapped", "internal/adapter/recipeyaml", ByMutation},
	{"a consistency rule that means nothing is refused rather than carried", "TestARuleThatSaysNothingIsRefused", "internal/adapter/recipeyaml", ByMutation},
	{"a consistency rule cannot name its own field", "TestARuleNamingItsOwnFieldIsRefused", "internal/adapter/recipeyaml", ByMutation},
	{"a recipe that cannot be used ends with the configuration code", "TestAnUnusableRecipeIsAConfigurationFailure", "internal/app/validate", ByMutation},

	// --- the command line, which enters somebody else's pipeline ------------
	{"the exit codes match the frozen contract", "TestTableMatchesTheContract", "internal/cli", ByMutation},
	{"a cross cutting flag works on either side of the verb", "TestTheMachineReadableFlagWorksOnEitherSideOfTheVerb", "internal/cli", ByMutation},
	{"an oversized recipe is refused without first being held in memory", "TestAnOversizedRecipeIsNotHeldInMemoryToBeRefused", "internal/cli", ByMutation},
	{"a control character from outside never reaches the terminal", "TestControlCharactersFromARecipeNeverReachTheLogChannel", "internal/cli", ByMutation},
	{"the data channel keeps the control characters it was given", "TestTheDataChannelIsNotCleanedOfControlCharacters", "internal/cli", ByMutation},
	{"the machine readable report escapes control characters", "TestTheMachineReadableReportEscapesControlCharacters", "internal/cli", ByProbe},
	{"the help leads with an example somebody can paste", "TestTheHelpBeginsWithAnExampleBeforeTheUsagePattern", "internal/cli", ByMutation},
	{"a mistyped command is answered with the one it is near", "TestACommandOneEditAwayIsSuggested", "internal/cli", ByMutation},
	{"a word nothing like a command is not offered as a near miss", "TestAWordNothingLikeACommandIsRefusedWithoutASuggestion", "internal/cli", ByMutation},
	{"a refusal keeps its four parts whether or not it suggests", "TestBothShapesOfTheUnknownCommandRefusalKeepFourParts", "internal/cli", ByMutation},
	{"every flag the help promises is one the parser accepts", "TestEveryDocumentedFlagIsAcceptedByTheParser", "internal/cli", ByMutation},
	{"a warning keeps its meaning when the detail under it is suppressed", "TestTheUncheckedWarningSurvivesQuietWithItsMeaningIntact", "internal/cli", ByMutation},
	{"a flag that exists is never reported as one that does not", "TestAFlagGivenAValueIsToldItIsASwitch", "internal/cli", ByMutation},
	{"a directory given as a recipe is refused in our own words", "TestADirectoryGivenAsARecipeIsRefusedInOurOwnWords", "internal/cli", ByMutation},
	{"the parser and the colour scan agree about a repeated flag", "TestTheLastSpellingOfNoColourWins", "internal/cli", ByMutation},

	// --- capabilities and the provider contract -----------------------------
	{"a missing capability is reported in a stable order", "TestMissingReportsWhatIsAbsent", "internal/domain/capability", ByMutation},
	{"a capability no provider can bring is not a provider capability", "TestTheCapabilitiesNoProviderCanBringAreNotProviderCapabilities", "internal/domain/capability", ByMutation},
	{"a declared capability has to be backed by its interface", "TestDeclaringACapabilityWithoutImplementingItFails", "internal/domain/provider", ByMutation},
	{"a provider cannot claim somebody else's capability", "TestAProviderCannotClaimSomebodyElsesCapability", "internal/domain/provider", ByMutation},
	{"claiming unassigned targets requires the side that writes them", "TestClaimingUnassignedTargetsWithoutTheWriteSideIsRefused", "internal/domain/provider", ByMutation},
	{"a scheme with no safe harbour can still offer targets to corrupt into", "TestAProviderWithNoHarbourCanStillOfferUnassignedTargets", "internal/domain/provider", ByMutation},

	// --- reference data and its attribution ---------------------------------
	{"every corpus is attributed and every attribution has a corpus", "TestEveryListIsAttributedAndEveryAttributionHasAList", "internal/adapter/provider/corpus", ByMutation},
	{"the notice at the top of a corpus never becomes data", "TestTheHeaderDoesNotLeakIntoTheEntries", "internal/adapter/provider/corpus", ByMutation},
	// These two read the embedded files rather than exercising a branch, so
	// breaking the code they defend leaves them green: every list carried today
	// is free of carriage returns and none of them is empty. They are probes,
	// and calling them mutation proven would be the register flattering itself.
	{"no entry of a word list carries an invisible line ending", "TestNoEntryCarriesAnInvisibleLineEnding", "internal/adapter/provider/corpus", ByProbe},
	{"a list with no entries is refused rather than handed over to be drawn from", "TestAListWithNoEntriesIsRefusedLikeAMissingOne", "internal/adapter/provider/corpus", ByProbe},

	// --- the identifiers, where a wrong rule reaches a real person ----------
	{"every generated Polish number satisfies an independent validator", "TestEveryGeneratedValueSatisfiesTheIndependentValidator", "internal/adapter/provider/pl", ByMutation},
	{"a bound date and gender survive into the Polish number", "TestABoundDateAndGenderSurviveIntoTheNumber", "internal/adapter/provider/pl", ByMutation},
	{"every generated Social Security number sits where none is assigned", "TestEveryGeneratedValueSitsInTheAreaThatIsNeverAssigned", "internal/adapter/provider/us", ByMutation},
	{"neither part of a Social Security number falls to nought", "TestTheGroupAndTheSerialNeverFallToNought", "internal/adapter/provider/us", ByMutation},
	{"the shape check passes the areas that look ordinary", "TestTheAreasThatLookOrdinaryPassValidationAndAreStillNeverAssigned", "internal/adapter/provider/us", ByMutation},
	{"moving a value where it already sits is refused rather than silent", "TestMovingAValueIntoTheBlockItAlreadyOccupiesIsRefused", "internal/adapter/provider/us", ByMutation},
	{"a part too large for its width is refused rather than truncated", "TestAPartTooLargeForItsWidthIsRefusedRatherThanTruncated", "internal/adapter/provider/us", ByMutation},
	{"the personal number weights match the arithmetic the ministry published", "TestTheWorkedExampleFromTheMinistryIsReproduced", "internal/adapter/provider/pl", ByMutation},
	{"every generated Polish tax number satisfies an independent validator", "TestEveryGeneratedTaxNumberSatisfiesTheIndependentValidator", "internal/adapter/provider/pl", ByMutation},
	{"the tax number weights match the arithmetic the ministry published", "TestTheWorkedExampleFromTheSourceIsReproduced", "internal/adapter/provider/pl", ByMutation},
	{"a body no check digit can complete is refused by name, not as a mismatch", "TestABodyNoCheckDigitCanCompleteIsRefusedByName", "internal/adapter/provider/pl", ByMutation},
	{"the way a tax number was written survives being rebuilt", "TestTheThreeWaysOfWritingOneNumberAreAllAccepted", "internal/adapter/provider/pl", ByMutation},
	{"the constrained ninth digit does not collapse to one value", "TestTheConstrainedNinthDigitStillVaries", "internal/adapter/provider/pl", ByMutation},
	{"every generated Polish statistical number satisfies an independent validator", "TestEveryGeneratedStatisticalNumberSatisfiesTheIndependentValidator", "internal/adapter/provider/pl", ByMutation},
	{"the eleventh residue is written as nought instead of voiding the number", "TestTheEleventhResidueIsWrittenAsNoughtRatherThanRefusingTheNumber", "internal/adapter/provider/pl", ByMutation},
	{"recomputing a statistical check digit never refuses", "TestRecomputingTheCheckDigitNeverRefuses", "internal/adapter/provider/pl", ByMutation},
	{"a local unit number is refused by name rather than called malformed", "TestALocalUnitNumberIsRefusedByNameRatherThanAsMalformed", "internal/adapter/provider/pl", ByMutation},
	{"the one weakness of the folded check digit is exactly where it is claimed", "TestASingleDigitErrorSurvivesExactlyWhenTheResidueFoldsOntoNought", "internal/adapter/provider/pl", ByMutation},
	{"a bound gender the scheme cannot read is refused rather than quietly drawn", "TestAGenderThisSchemeCannotReadIsRefusedRatherThanDrawn", "internal/adapter/provider/pl", ByMutation},
	{"the gender spellings the scheme does read keep working", "TestTheGenderSpellingsTheSchemeReadsStillWork", "internal/adapter/provider/pl", ByMutation},
	{"a check digit position is answered about a number and not about a length", "TestTheCheckDigitPositionIsRefusedForValuesThatAreNotDigits", "internal/adapter/provider/pl", ByMutation},
	{"a form this build does not check is not reported as a defect", "TestALocalUnitNumberIsUnsupportedRatherThanInvalid", "internal/adapter/provider/pl", ByMutation},
	{"a value that is genuinely wrong is not excused as unchecked", "TestABrokenStatisticalNumberIsStillInvalidRatherThanUnsupported", "internal/adapter/provider/pl", ByMutation},

	// --- US.aba, the scheme with no safe harbour ----------------------------
	{"every generated routing number satisfies an independent validator", "TestEveryGeneratedRoutingNumberSatisfiesTheIndependentValidator", "internal/adapter/provider/us", ByMutation},
	{"no generated routing number sits in a series the registrar has not handed out", "TestEveryGeneratedRoutingNumberSitsInAnIssuedSeries", "internal/adapter/provider/us", ByMutation},
	{"a scheme with no harbour does not claim one", "TestTheRoutingNumberClaimsNoSafeHarbour", "internal/adapter/provider/us", ByMutation},
	{"the source still says spare series are stock rather than sanctuary", "TestTheSpareSeriesAreStockRatherThanSanctuary", "internal/adapter/provider/us", ByMutation},
	{"a corruption aimed at the allocation table breaks nothing else", "TestMovingIntoASpareSeriesLeavesAWellFormedNumberTheTableRefuses", "internal/adapter/provider/us", ByMutation},
	{"recomputing a routing check digit never refuses", "TestRecomputingARoutingCheckDigitNeverRefuses", "internal/adapter/provider/us", ByMutation},
	{"the way a routing number was written survives being rebuilt", "TestBothWaysOfWritingARoutingNumberAreAccepted", "internal/adapter/provider/us", ByMutation},
	{"the generated routing numbers have not moved since they were frozen", "TestTheGeneratedRoutingNumbersHaveNotMoved", "internal/adapter/provider/us", ByMutation},

	// --- US.postal, where the reference data is the product --------------------
	//
	// The only provider here whose guards mostly stand in front of a file rather
	// than in front of arithmetic. One of the mutations below breaks the data
	// instead of the code, because that is where a fault would actually live.
	{"a mismatched pair is genuinely mismatched rather than agreeing", "TestADifferentPlaceIsRealAndActuallyDifferent", "internal/adapter/provider/us", ByMutation},
	{"a code the registry cannot place is refused rather than paired with an invention", "TestACodeTheRegistryDoesNotCarryIsRefusedRatherThanInvented", "internal/adapter/provider/us", ByMutation},
	{"a real code missing from the registry is not reported as a defect", "TestValidationAnswersAboutShapeAndNotAboutMembership", "internal/adapter/provider/us", ByMutation},
	{"no place name ships carrying the term the census appends to it", "TestNoPlaceNameStillCarriesTheTermTheCensusAppends", "internal/adapter/provider/us", ByMutation},
	{"the generated postal codes have not moved since they were frozen", "TestTheGeneratedPostalCodesHaveNotMoved", "internal/adapter/provider/us", ByMutation},
	{"every code in the registry answers with the place beside it", "TestEveryCodeInTheRegistryResolvesToItsPlace", "internal/adapter/provider/us", ByProbe},
	{"one seed gives one mismatched pair, every time", "TestADifferentPlaceIsTheSameAnswerEveryTime", "internal/adapter/provider/us", ByProbe},

	// --- iban, where the arithmetic is quoted and only its reach is measured ---
	//
	// The reverse of every other identifier here. The three constants of the
	// arithmetic are guarded against a normative act rather than against a
	// measurement, so a guard going red means the code stopped matching a
	// sentence somebody can go and read.
	{"every generated account number satisfies an independent validator", "TestEveryGeneratedAccountNumberSatisfiesTheIndependentValidator", "internal/adapter/provider/intl", ByMutation},
	{"pillar one only produces values whose correctness can be checked in full", "TestEveryGeneratedAccountNumberIsOneThisBuildChecksInFull", "internal/adapter/provider/intl", ByMutation},
	{"every generated settlement number is one the central bank could have assigned", "TestEveryGeneratedSettlementNumberClosesUnderTheActInForce", "internal/adapter/provider/intl", ByMutation},
	{"the weight table the act published is reproduced, and it forces the modulus", "TestTheWeightTableThePolishActPublishesIsReproduced", "internal/adapter/provider/intl", ByMutation},
	{"the quoted act still fixes the rotation and the accepted remainder", "TestTheSourceStillFixesTheRotationAndTheAcceptedRemainder", "internal/adapter/provider/intl", ByMutation},
	{"the act's own wrong name for its letter values is kept and contradicted rather than corrected", "TestTheActsOwnNameForItsLetterValuesIsWrongAndTheNumbersAreRight", "internal/adapter/provider/intl", ByMutation},
	{"check digits no calculation reaches are refused at both ends of the range", "TestCheckDigitsNoCalculationCanReachAreRefused", "internal/adapter/provider/intl", ByMutation},
	{"a country whose length was never read is answered as unchecked rather than approved", "TestAnAccountNumberFromAnotherCountryIsUnsupportedRatherThanApproved", "internal/adapter/provider/intl", ByMutation},
	{"a value that is genuinely wrong is not excused as unchecked", "TestABrokenAccountNumberFromAnotherCountryIsStillInvalid", "internal/adapter/provider/intl", ByMutation},
	{"a settlement number that could not have been assigned is named rather than reported as arithmetic", "TestASettlementNumberTheBankCouldNotHaveAssignedIsNamedSeparately", "internal/adapter/provider/intl", ByMutation},
	{"a Polish value of the wrong length is told about its length", "TestAPolishAccountNumberOfTheWrongLengthIsToldAboutItsLength", "internal/adapter/provider/intl", ByMutation},
	{"the spacing a value arrived in survives being rebuilt", "TestTheSpacingOfAnAccountNumberSurvivesBeingRebuilt", "internal/adapter/provider/intl", ByMutation},
	{"the check digit offsets follow the spacing rather than assuming there is none", "TestTheCheckDigitPositionsFollowTheSpacing", "internal/adapter/provider/intl", ByMutation},
	{"a scheme with no harbour and no spare block claims neither", "TestTheAccountNumberClaimsNoSafeHarbour", "internal/adapter/provider/intl", ByMutation},
	{"the generated account numbers have not moved since they were frozen", "TestTheGeneratedAccountNumbersHaveNotMoved", "internal/adapter/provider/intl", ByMutation},

	// --- the first pillar end to end: plan, engine, writers, disk -----------
	//
	// Everything below stood green and unproven until 2026-08-04. These are the
	// promises a recipe makes to whoever runs it, and the two at the top are the
	// ones the whole product rests on: the same seed gives the same bytes, and
	// an identifier agrees with the columns it was bound to.
	{"the same seed gives the same bytes, from the platform's line endings to the order of the columns", "TestTheSameSeedGivesTheSameBytes", "internal/adapter/format", ByMutation},
	{"an identifier agrees with the columns the recipe bound it to", "TestTheIdentifierAgreesWithTheColumnsItWasBoundTo", "internal/adapter/format", ByMutation},
	{"a child names a parent that exists in the table it points at", "TestARecipeBecomesFilesWithTheRelationFollowable", "internal/adapter/format", ByMutation},
	{"adding a field leaves the columns that were already there where they were", "TestAddingAFieldLeavesTheOtherColumnsWhereTheyWere", "internal/adapter/format", ByMutation},
	{"the count a child states is exact and the cardinality bounds how it is spread", "TestTheChildCountIsExactAndTheParentsShareIt", "internal/adapter/format", ByMutation},
	{"a count the bounds cannot reach is refused with the arithmetic rather than trimmed", "TestACountTheBoundsCannotReachIsRefusedWithTheArithmetic", "internal/adapter/format", ByMutation},
	{"the nested form puts children inside the parent they belong to", "TestTheNestedFormPutsChildrenInsideTheirParent", "internal/adapter/format", ByMutation},
	{"the tree carries every grandchild under the parent it belongs to, at three levels where a subset stops standing in for the whole table", "TestTheTreeCarriesEveryGrandchildUnderTheParentItBelongsTo", "internal/adapter/format", ByMutation},
	{"one file per table means one file per table", "TestTheFlatObjectFormWritesOneFilePerTable", "internal/adapter/format", ByMutation},
	{"a value carrying a delimiter or a quote survives being written", "TestAValueCarryingTheDelimiterOrAQuoteSurvives", "internal/adapter/format", ByMutation},
	{"a field asked to be unique never repeats a value", "TestAUniqueFieldNeverRepeatsAValue", "internal/adapter/format", ByMutation},
	{"a uniqueness nothing can satisfy is refused with the number of rows that made it impossible", "TestAUniqueFieldWithTooFewValuesIsRefusedWithTheArithmetic", "internal/adapter/format", ByMutation},

	// --- where a value comes from -------------------------------------------
	{"a Polish surname never reaches a woman in the masculine form", "TestAPolishSurnameNeverGivesAWomanTheMasculineForm", "internal/adapter/values", ByMutation},
	{"a locale with no register of postal codes says so rather than inventing one", "TestAPostalCodeIsRefusedWhereThereIsNoRegister", "internal/adapter/values", ByMutation},
	{"a maximum length is honoured by choosing an entry that fits", "TestALimitIsHonouredByChoosingRatherThanCutting", "internal/adapter/values", ByMutation},
	{"a fixed width value too long for its field is refused rather than cut", "TestAFixedWidthValueTooLongForItsFieldIsRefused", "internal/adapter/values", ByMutation},
	{"bytes and characters are counted apart, which is why the unit is carried", "TestBytesAndCharactersAreCountedDifferently", "internal/adapter/values", ByMutation},
	{"a generator this build lacks is refused by name and the known ones can be listed", "TestAnUnknownGeneratorIsRefusedAndTheKnownOnesCanBeListed", "internal/adapter/values", ByMutation},
	{"what this build says it can produce is what it produces, for every generator in every locale", "TestWhatThisBuildSaysItCanProduceIsWhatItProduces", "internal/adapter/values", ByMutation},
	{"drawing under a limit leaves the shared word list as it was, so the next row draws from the list somebody chose", "TestDrawingUnderALimitLeavesTheSharedListAlone", "internal/adapter/values", ByMutation},
	{"a generator this build cannot fill in the recipe's locale is refused by the plan rather than on the first row", "TestAGeneratorTheLocaleCannotFillIsRefusedByThePlan", "internal/app/generate", ByMutation},
	{"validate refuses the generator and locale pair that generate refuses, so the two verbs agree about one file", "TestAGeneratorTheLocaleCannotFillIsAProblem", "internal/app/validate", ByMutation},
	{"the role a field plays comes from its generator and not from the name of the column", "TestTheRoleComesFromTheGeneratorAndOnlyTwoPlayOne", "internal/adapter/values", ByMutation},
	{"every drawn date is a day that exists", "TestEveryDrawnDateIsARealDay", "internal/adapter/values", ByMutation},

	// --- the disk, where a mistake costs somebody else's files ---------------
	{"nothing appears under its real name before the whole run has succeeded", "TestNothingAppearsUnderItsRealNameBeforeTheRunSucceeds", "internal/adapter/store", ByMutation},
	{"two things in one run never share a destination, whether or not the run may overwrite", "TestTwoThingsInOneRunCannotShareAName", "internal/adapter/store", ByMutation},
	{"a rename that fails takes back the ones that had already succeeded", "TestARenameThatFailsTakesTheEarlierOnesBack", "internal/adapter/store", ByMutation},
	{"a run that ends early takes its temporary files with it", "TestADiscardedRunLeavesNothingBehind", "internal/adapter/store", ByMutation},
	{"an existing file stops the run unless overwriting was asked for", "TestAnExistingFileStopsTheRunUnlessOverwritingIsAllowed", "internal/adapter/store", ByMutation},
	{"a name the file system would not keep is refused rather than repaired", "TestANameTheFileSystemWouldNotKeepIsRefusedRatherThanRepaired", "internal/adapter/store", ByMutation},

	// --- the verb, as somebody's pipeline meets it ---------------------------
	{"a second run into the same directory is refused rather than overwriting", "TestASecondRunIntoTheSameDirectoryIsRefused", "internal/cli", ByMutation},
	{"a dataset carrying identifiers says about itself that it describes nobody", "TestTheNoteSaysTheValuesAreSynthetic", "internal/cli", ByMutation},
	{"the record of the run names the settings the command line overrode, which is what it exists to answer", "TestTheNoteSaysWhichSettingsTheCommandLineOverrode", "internal/cli", ByMutation},
	{"a dry run answers the same arithmetic and touches no disk", "TestADryRunAnswersWithoutWritingAnything", "internal/cli", ByMutation},
	{"the summary says what the run did, and every count agrees with its word", "TestTheSummarySaysWhatTheRunDidAndCountsAgreeWithTheirWords", "internal/cli", ByMutation},
	{"every capability of the command line is answered for from the side of the window", "TestEveryCapabilityOfTheCommandLineIsAnsweredForFromTheWindow", "internal/cli", ByMutation},
	{"a claim that the window reaches something names a test that is in the tree", "TestEveryClaimAboutTheWindowNamesATestThatExists", "internal/cli", ByMutation},
	{"a defect spanning several records survives being written and read back", "TestADefectSpanningSeveralRecordsSurvivesBeingWrittenAndRead", "internal/adapter/manifestjson", ByMutation},
	{"a byte offset survives being written and read back, where its loss would be silent", "TestAByteOffsetSurvivesBeingWrittenAndRead", "internal/adapter/manifestjson", ByMutation},
	{"nothing in the window rebuilds a piece of the toolkit without saying why", "TestNothingInTheWindowRebuildsTheToolkitWithoutSayingWhy", "internal/guard", ByMutation},
	{"an entry that did not happen says why on the screen, without one column repeating another", "TestAnUnappliedEntrySaysWhyOnTheScreen", "internal/gui/manifestview", ByMutation},
	{"the second state shows what the application did with every row that could be judged", "TestTheSecondStateShowsWhatTheApplicationDidWithEveryJudgedRow", "internal/gui/manifestview", ByMutation},
	{"a document written to a version this build cannot read is refused rather than compared", "TestADocumentFromAVersionThisBuildCannotReadIsRefused", "internal/cli", ByMutation},
	{"a document from a newer minor version is read and the run says what that costs", "TestADocumentFromANewerMinorVersionIsReadAndSaidSo", "internal/cli", ByMutation},
	{"what a build can do with a version it was handed follows the major line", "TestWhatThisBuildCanDoWithAVersion", "internal/version", ByMutation},
	{"tables whose rows do not add up to a countable total are refused", "TestRowsThatDoNotAddUpToACountableTotalAreRefused", "internal/app/generate", ByMutation},
	{"a setting the command line moved is recorded with both sides", "TestAFlagThatOverridesTheRecipeSaysSo", "internal/cli", ByMutation},
	{"every surface that lists what this tool accepts lists all of it", "TestEverySurfaceListingCommandsNamesAllOfThem", "internal/cli", ByMutation},
	{"a flag named in the help of a command is one that command accepts", "TestEveryFlagAVerbDocumentsIsOneItAccepts", "internal/cli", ByMutation},
	{"every shape of place a problem can name is reported at the line it was written on", "TestEveryShapeOfPlaceIsReportedAtTheLineItWasWrittenOn", "internal/adapter/recipeyaml", ByMutation},
	{"a path two places in a recipe claim is left without a line rather than pointing at one of them", "TestAPathTwoPlacesClaimIsLeftWithoutALine", "internal/adapter/recipeyaml", ByMutation},
	{"a place nobody wrote is reported without a line rather than with a wrong one", "TestAPlaceNobodyWroteIsReportedWithoutALine", "internal/adapter/recipeyaml", ByProbe},
	{"a problem raised above the decoder is pointed at the same line the decoder would have pointed at", "TestAProblemRaisedAboveTheDecoderCarriesTheLineOfItsPlace", "internal/app/validate", ByMutation},
	{"validate refuses exactly what generate would refuse", "TestValidateRefusesARecipeGenerateWouldRefuse", "internal/cli", ByMutation},
	{"the two surfaces count what they understood of a recipe in the same words", "TestWhatWasUnderstoodIsCountedInTheSingularWhenThereIsOneOfIt", "internal/cli", ByMutation},
	{"validate, a dry run and a run agree about a relation colliding with a field", "TestTheThreeAnswersAboutARelationCollidingWithAFieldAgree", "internal/cli", ByMutation},
	{"a writer refuses a table whose relation and field want one column", "TestAWriterRefusesATableWhoseRelationWantsAColumnAFieldAlreadyHas", "internal/adapter/format", ByMutation},
	{"a preview is the first rows of the run it stands for", "TestASampleIsTheFirstRowsTheRunItselfWouldWrite", "internal/preview", ByMutation},
	{"the rows a recipe would produce are on the screen it is written on", "TestTheRowsOnScreenAreTheRowsTheSampleCarries", "internal/gui/recipeview", ByMutation},
	{"a value is not cut short in a column with room for it", "TestAValueIsNotCutWhenItsColumnHasRoomForIt", "internal/gui/recipeview", ByMutation},
	{"a recipe that produces no rows says so rather than drawing an empty table", "TestATableWithNoRowsIsASentenceRatherThanAnEmptyTable", "internal/gui/recipeview", ByMutation},
	{"a recipe the plan refuses is not drawn as an approval", "TestARecipeThePlanRefusesIsDrawnAsARefusalAndNotAsAnApproval", "internal/gui/recipeview", ByMutation},
	{"what was counted is written in the form of the word that fits it", "TestTheCountsOnScreenUseTheRightFormOfEachWord", "internal/gui/recipeview", ByMutation},
	{"a parent outside the preview is named rather than left blank", "TestAParentOutsideTheSampleIsNamedRatherThanLeftBlank", "internal/gui/recipeview", ByMutation},
	{"the key naming the parent in a preview is the one the run writes", "TestTheKeyNamingTheParentIsTheOneTheRunWrites", "internal/preview", ByMutation},
	{"a refusal says what to do next whichever layer it came from", "TestAFailureFromBelowStillSaysWhatToDo", "internal/cli", ByMutation},

	// --- being stopped from outside -----------------------------------------
	//
	// There is a fourth test beside these three, TestARealSignalReachesTheWatch,
	// and it is deliberately absent. It is the only one that proves the runtime
	// delivers the signal we ask for, and it cannot run on Windows, where the
	// mutation runner runs. An entry claiming it was proven by mutation would be
	// a sentence nobody has checked, which is the one thing this register exists
	// to prevent.
	{"a run stopped part way leaves neither an output file nor a temporary one", "TestARunStoppedPartWayLeavesNothingBehind", "internal/cli", ByMutation},
	{"an interrupted run ends with the code a shell would have reported", "TestTheWatchEndsTheRunAsAnInterruptedOne", "internal/cli", ByMutation},
	{"a run told to stop stays told, however many times it asks", "TestTheWatchGivesTheSameAnswerEveryTimeItIsAsked", "internal/cli", ByMutation},
	{"a run nobody interrupted is never reported as interrupted", "TestAWatchReleasedBeforeAnySignalStopsNothing", "internal/cli", ByMutation},

	// --- what the tool says its own output is -------------------------------
	//
	// Not a licence obligation. Several of these schemes have no block that is
	// never issued, so a correct value cannot be shown not to belong to
	// somebody, and saying so is the whole of the defence. A README does not
	// travel with a file of data.
	{"the tool says every value it produces is synthetic", "TestLicenseSaysEveryValueIsSynthetic", "internal/cli", ByMutation},
	{"that notice still says what to make of a resemblance", "TestTheSyntheticNoticeSaysWhatItHasTo", "internal/license", ByMutation},

	// The machine readable form of the licence report (DEC-146).
	//
	// The reader is a compliance scanner, and the failure it cannot survive is
	// a registry row that reaches the printed form and not the document. Prose
	// reads complete whether or not a dependency is missing from it, so the
	// count is the only thing that can tell.
	{"the licence document carries every registry row", "TestTheMachineReadableLicenseCarriesWhatTheReadableOneDoes", "internal/cli", ByMutation},
	{"both forms of the licence report name the same projects", "TestBothLicenseFormsNameTheSameProjects", "internal/cli", ByMutation},

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
	{"damaging a file where it sits leaves the source alone", "TestDamagingAFileInItsOwnDirectoryLeavesTheSourceAlone", "internal/cli", ByMutation},
	{"the evidence for every column reaches the manifest", "TestTheManifestCarriesTheEvidenceForEveryColumn", "internal/cli", ByMutation},
	{"the same file and seed give the same damage", "TestTheSameFileAndSeedGiveTheSameDamage", "internal/cli", ByMutation},

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
}

// CountByProof returns how many guards are in each state.
func CountByProof() map[ProofState]int {
	counts := make(map[ProofState]int, 3)
	for _, guard := range Guards {
		counts[guard.Proof]++
	}
	return counts
}
