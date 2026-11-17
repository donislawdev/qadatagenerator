package guard

import "slices"

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

// Guards is the register.
//
// It is deliberately not every test in the project. A test that checks one
// function does its arithmetic is not a guard. A guard is a test standing
// between the product and a promise it makes to somebody outside it.
var Guards = slices.Concat(toolGuards, windowGuards, damageGuards)

// toolGuards is the part of the register about the tool itself and about the
// first pillar: the architecture, the identifiers, the engine that writes
// correct data, and the command line somebody's pipeline meets. The window has
// a part of its own, windowGuards in proof_window.go.
//
// # Why the register is in three files rather than one
//
// It reached the ceiling on the size of a file on 2026-09-05, at 500 lines of
// code against a limit of 500, and the limit may be lowered but never raised.
// A register is not ordinary code in this respect: code can be shortened, and a
// register grows with the number of things it describes, so the only move
// available is to cut it. The cut is at the seam between the two pillars this
// product is built on rather than at an arbitrary line, so that a new entry has
// an obvious home. The second cut, on 2026-10-28 at 509 lines, took the window
// out of this part, because it is one surface with sections of its own.
//
// Every tool that reads this register reads every proof*.go of this package,
// which is what keeps a part from going quietly uncounted.
var toolGuards = []Guard{
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
	{"a name taken from syscall under another name of the package is still seen", "TestSyscallScannerFollowsThePackageUnderAnotherName", "internal/guard", ByProbe},

	// --- C and the module of system calls, the same door wider (DEC-321) ------
	//
	// A file importing C can call any C function and write one above the
	// import, and golang.org/x/sys does for every system what syscall does. The
	// window calls Windows through C in one file, so that file is an allowance
	// of names like the syscall ones, and its comment above the import may hold
	// only build lines and system headers.
	{"every way of calling the system is refused, and a look-alike is not", "TestTheNetworkGuardRefusesEveryWayOfCallingTheSystem", "internal/guard", ByProbe},
	{"C is called only in the one file, and only for the names it lists", "TestTheProductCallsCOnlyWhereItIsAllowed", "internal/guard", ByMutation},
	{"the scan for C actually reached the product", "TestTheCScanReachesTheProduct", "internal/guard", ByProbe},
	{"a file calling C with no allowance, and a name beyond its entry, are seen", "TestCScannerSeesAFileNobodyAllowedAndANameBeyondItsEntry", "internal/guard", ByProbe},
	{"C and flags written above the import are seen, and the named linker flag with system headers is not", "TestCScannerSeesCWrittenAboveTheImport", "internal/guard", ByProbe},
	{"the exception for C covers one package in the network rule and no more", "TestTheNetworkRuleStillRefusesCOutsideTheAllowedPackage", "internal/guard", ByProbe},

	// --- what reaches the system with no import to show for it (DEC-322) ----
	//
	// The go command compiles assembly and C beside the Go files and links any
	// .syso, the compiler obeys go:linkname, and unsafe can call any address.
	// None of it is an import the rules above read, so each is read here.
	{"the go command compiles nothing of ours beside Go but resources", "TestTheProductCompilesNothingBesideGoButResources", "internal/guard", ByProbe},
	{"a file the go command compiles beside Go is seen, and a .syso beyond resources too", "TestTheScanBesideGoSeesWhatTheGoCommandCompiles", "internal/guard", ByProbe},
	{"only the compiler directives on the list stand in our files", "TestTheProductWritesOnlyTheCompilerDirectivesItMay", "internal/guard", ByProbe},
	{"a directive is seen wherever it stands, and what only looks like one is not", "TestTheDirectiveScanSeesWhatTheCompilerReads", "internal/guard", ByProbe},
	{"unsafe is imported in the one file only, for the names it lists", "TestTheProductImportsUnsafeOnlyWhereItIsAllowed", "internal/guard", ByProbe},
	{"a blank import of unsafe and a name beyond the entry are seen", "TestUnsafeScannerSeesABlankImportAndANameBeyondTheEntry", "internal/guard", ByProbe},
	{"the exception for unsafe covers one package in the network rule and no more", "TestTheNetworkRuleStillRefusesUnsafeOutsideTheAllowedPackage", "internal/guard", ByProbe},

	// --- two runs into one directory ----------------------------------------
	//
	// The package promises that a file which already exists stops the run. It
	// used to hold only for a file present when the run started, because
	// asking and renaming are two moments with a whole run in between.
	{"a second run cannot take a name the first one is writing", "TestASecondRunCannotTakeANameTheFirstIsWriting", "internal/adapter/store", ByMutation},
	{"the marker does not outlive the run that took it", "TestTheMarkerIsGoneOnceTheResultIsInPlace", "internal/adapter/store", ByMutation},
	{"a discarded run leaves nothing that blocks the next one", "TestADiscardedRunLeavesNoMarkerBehind", "internal/adapter/store", ByMutation},

	// --- the window: its values, what it says, and the keyboard in it ------
	//
	// In proof_window.go since 2026-10-28, when this file reached the ceiling
	// on its size. The window is one surface, so it is a seam and not a line.

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
	{"validate reports what only the engine knows about a recipe", "TestValidateRefusesARelationTheEngineWillNotOrder", "internal/cli", ByMutation},
	{"a recipe that reads correctly and cannot be run is warned about rather than refused", "TestARecipeThatCannotBeRunIsWarnedAboutRatherThanRefused", "internal/cli", ByMutation},
	{"what validate says a run would write is what the dry run says", "TestWhatValidateSaysARunWouldWriteIsWhatTheDryRunSays", "internal/cli", ByMutation},
	{"a relation wider than this build can plan for is refused rather than drawn from", "TestARangeWiderThanThisBuildCanPlanForIsRefused", "internal/app/generate", ByMutation},
	{"more parents than a share can be held for is refused on its own terms", "TestMoreParentsThanThisBuildCanShareOutBetweenIsRefused", "internal/app/generate", ByMutation},
	{"the reach of a relation is compared without forming the product", "TestTheReachIsComparedWithoutFormingTheProduct", "internal/app/generate", ByMutation},
	{"the ceiling on a plan leaves ordinary relations alone", "TestTheReachOfAnOrdinaryRelationIsNotRefused", "internal/app/generate", ByMutation},
	{"work handed to the background answers a panic instead of ending the window", "TestWorkThatComesApartIsAnsweredRatherThanEndingTheWindow", "internal/gui/background", ByMutation},
	{"a read that comes apart reaches the way out its screen chose", "TestAReadThatComesApartReachesTheWayOutTheScreenChose", "internal/gui/background", ByMutation},
	{"a check that comes apart is drawn as a defect of this build and not as a refusal", "TestACheckThatComesApartIsDrawnAsADefectOfThisBuildRatherThanARefusal", "internal/gui/recipeview", ByMutation},
	{"a finished check takes an earlier crash off the screen", "TestACheckThatFinishesTakesTheCrashOffTheScreen", "internal/gui/recipeview", ByMutation},
	{"the nesting limit covers a sequence written with dashes and no bracket at all", "TestARecipeNestedInDashesAlongOneLineIsRefusedBeforeParsing", "internal/adapter/recipeyaml", ByMutation},
	{"a dash counts whatever blanks follow it rather than a two byte token", "TestADashCountsWhateverBlanksFollowIt", "internal/adapter/recipeyaml", ByMutation},
	{"the dashes are counted within a line and not across the file", "TestTheDashesAreCountedWithinALineAndNotAcrossTheFile", "internal/adapter/recipeyaml", ByMutation},
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
	{"a file the disk stopped taking is never reported as written, whether the refusal arrives at the flush or while the walk is still going", "TestAFileThatCouldNotBeWrittenIsNeverReportedAsWritten", "internal/adapter/format", ByMutation},
	{"a run stopped while the objects are being written ends with the stop rather than with a finished file", "TestAStopWhileTheObjectsAreWrittenEndsTheRunRatherThanTheFile", "internal/adapter/format", ByMutation},

	// --- where a value comes from -------------------------------------------
	{"a Polish surname never reaches a woman in the masculine form", "TestAPolishSurnameNeverGivesAWomanTheMasculineForm", "internal/adapter/values", ByMutation},
	{"a locale with no register of postal codes says so rather than inventing one", "TestAPostalCodeIsRefusedWhereThereIsNoRegister", "internal/adapter/values", ByMutation},
	{"a maximum length is honoured by choosing an entry that fits", "TestALimitIsHonouredByChoosingRatherThanCutting", "internal/adapter/values", ByMutation},
	{"a fixed width value too long for its field is refused rather than cut", "TestAFixedWidthValueTooLongForItsFieldIsRefused", "internal/adapter/values", ByMutation},
	{"bytes and characters are counted apart, which is why the unit is carried", "TestBytesAndCharactersAreCountedDifferently", "internal/adapter/values", ByMutation},
	{"a generator this build lacks is refused by name and the known ones can be listed", "TestAnUnknownGeneratorIsRefusedAndTheKnownOnesCanBeListed", "internal/adapter/values", ByMutation},
	{"what this build says it can produce is what it produces, for every generator in every locale", "TestWhatThisBuildSaysItCanProduceIsWhatItProduces", "internal/adapter/values", ByMutation},
	{"drawing under a limit leaves the shared word list as it was, so the next row draws from the list somebody chose", "TestDrawingUnderALimitLeavesTheSharedListAlone", "internal/adapter/values", ByMutation},
	{"a draw asked for more digits than the number it returns can hold is capped rather than walking off the end of the arithmetic", "TestADrawWiderThanTheNumbersItReturnsIsCappedRatherThanPanicking", "internal/adapter/values", ByMutation},
	{"a generator this build cannot fill in the recipe's locale is refused by the plan rather than on the first row", "TestAGeneratorTheLocaleCannotFillIsRefusedByThePlan", "internal/app/generate", ByMutation},
	{"validate refuses the generator and locale pair that generate refuses, so the two verbs agree about one file", "TestAGeneratorTheLocaleCannotFillIsAProblem", "internal/app/validate", ByMutation},
	{"the role a field plays comes from its generator and not from the name of the column", "TestTheRoleComesFromTheGeneratorAndOnlyFourPlayOne", "internal/adapter/values", ByMutation},
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
	{"an entry that did not happen says why on the screen, its kind in the table and its reason in the card under it", "TestAnUnappliedEntrySaysWhyOnTheScreen", "internal/gui/manifestview", ByMutation},
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
	{"a preview is the first rows of the run it stands for", "TestASampleIsTheFirstRowsTheRunItselfWouldWrite", "internal/produce", ByMutation},
	{"the rows a recipe would produce are on the screen it is written on", "TestTheRowsOnScreenAreTheRowsTheSampleCarries", "internal/gui/recipeview", ByMutation},
	{"a value is not cut short in a column with room for it", "TestAValueIsNotCutWhenItsColumnHasRoomForIt", "internal/gui/recipeview", ByMutation},
	{"a recipe that produces no rows says so rather than drawing an empty table", "TestATableWithNoRowsIsASentenceRatherThanAnEmptyTable", "internal/gui/recipeview", ByMutation},
	{"a recipe the plan refuses is not drawn as an approval", "TestARecipeThePlanRefusesIsDrawnAsARefusalAndNotAsAnApproval", "internal/gui/recipeview", ByMutation},
	{"what was counted is written in the form of the word that fits it", "TestTheCountsOnScreenUseTheRightFormOfEachWord", "internal/gui/recipeview", ByMutation},
	{"a parent outside the preview is named rather than left blank", "TestAParentOutsideTheSampleIsNamedRatherThanLeftBlank", "internal/gui/recipeview", ByMutation},
	{"the button on the recipe editor writes the dataset and says what it wrote", "TestTheScreenWritesTheDatasetAndSaysWhatItWrote", "internal/gui/recipeview", ByMutation},
	{"a button that cannot run names the one thing that is missing", "TestADisabledButtonSaysWhatIsMissingInTheCard", "internal/gui/recipeview", ByMutation},
	{"the screen starts no run it would refuse", "TestAScreenWithNowhereToWriteStartsNothing", "internal/gui/recipeview", ByMutation},
	{"the files named before a run are the files the run writes", "TestWhatTheScreenSaysItWouldWriteIsWhatTheRunWrites", "internal/gui/recipeview", ByMutation},
	{"closing the window while a dataset is being written stops the run first", "TestAskingForTheWindowWhileWritingADatasetStopsItFirst", "internal/gui/recipeview", ByMutation},
	{"a preview is the first rows of what a run writes to a disk", "TestWhatIsWrittenBeginsWithTheRowsThePreviewShowed", "internal/produce", ByMutation},

	// --- what the recipe editor says to somebody who has never seen a recipe --
	{"the example this screen offers is a recipe that runs and produces rows", "TestTheExampleThisScreenOffersIsARecipeThatRuns", "internal/gui/recipeview", ByMutation},
	{"the offer of an example belongs to the empty screen, so nothing offers to replace what somebody wrote", "TestTheOfferOfAnExampleIsOnTheEmptyScreenAndNowhereElse", "internal/gui/recipeview", ByMutation},
	{"the list of what this build can generate is the register, all of it", "TestTheListNamesEveryGeneratorThisBuildHas", "internal/gui/recipeview", ByMutation},
	{"a generator that cannot be filled in a locale says why on this screen too", "TestTheListSaysWhereAGeneratorCannotBeFilled", "internal/gui/recipeview", ByMutation},
	{"the list stays open while somebody types, because the answer under it is redrawn on every key press", "TestTheListStaysOpenWhileSomebodyTypes", "internal/gui/recipeview", ByMutation},
	{"the files a run reports are the files its writer asked for", "TestTheNamesReportedAreTheNamesTheWriterAskedFor", "internal/produce", ByMutation},
	{"a run of a recipe that fails leaves nothing behind", "TestARunThatFailsLeavesNothingBehind", "internal/produce", ByMutation},
	{"a second run of a recipe refuses to replace files unless told", "TestASecondRunRefusesToReplaceFilesUnlessItIsTold", "internal/produce", ByMutation},
	{"the key naming the parent in a preview is the one the run writes", "TestTheKeyNamingTheParentIsTheOneTheRunWrites", "internal/produce", ByMutation},
	{"a refusal says what to do next whichever layer it came from", "TestAFailureFromBelowStillSaysWhatToDo", "internal/cli", ByMutation},

	// --- being stopped from outside -----------------------------------------
	//
	// There is a test beside these, TestARealSignalReachesTheWatch, and it is
	// deliberately absent. It is the only one that proves the runtime delivers
	// the signal we ask for, and it cannot run on Windows, where the mutation
	// runner runs. An entry claiming it was proven by mutation would be a
	// sentence nobody has checked, which is the one thing this register exists
	// to prevent.
	//
	// The tests that a stopped run ends by its signal once it has cleaned up
	// (signals_unix_test.go, B-346) are absent for the same reason. What can be
	// checked on any system is which code is sent again with which signal, and
	// that is the last entry here. The sending itself was mutated by hand on
	// Linux on 2026-11-02, and the record of it is in docs/13 under B-346.
	{"a run stopped part way leaves neither an output file nor a temporary one", "TestARunStoppedPartWayLeavesNothingBehind", "internal/cli", ByMutation},
	{"an interrupted run ends with the code a shell would have reported", "TestTheWatchEndsTheRunAsAnInterruptedOne", "internal/cli", ByMutation},
	{"a run told to stop stays told, however many times it asks", "TestTheWatchGivesTheSameAnswerEveryTimeItIsAsked", "internal/cli", ByMutation},
	{"a run nobody interrupted is never reported as interrupted", "TestAWatchReleasedBeforeAnySignalStopsNothing", "internal/cli", ByMutation},
	{"a run started in the background with the interrupt ignored ends with its code at once rather than waiting for a signal the system ignores", "TestEachCodeNamesTheSignalItIsSentAgainWith", "internal/cli", ByMutation},

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
	{"the calculation the regulation works out itself comes out the same here", "TestTheCalculationWorkedByTheRegulationComesOutTheSameHere", "internal/adapter/provider/intl", ByMutation},
	{"the residue that does not fit in a digit is written as the letter X", "TestTheEleventhResidueIsWrittenAsTheLetterX", "internal/adapter/provider/intl", ByMutation},
	{"the three letters the regulation leaves out of the alphabet are refused", "TestTheThreeLettersTheRegulationLeavesOutAreRefused", "internal/adapter/provider/intl", ByMutation},
	{"a generated number ends in the digits the regulation requires there", "TestEveryGeneratedNumberEndsInTheDigitsTheRegulationRequires", "internal/adapter/provider/intl", ByMutation},
	{"a generated number carries a model year the table can read back", "TestEveryGeneratedNumberCarriesAModelYearThatCanBeRead", "internal/adapter/provider/intl", ByMutation},
	{"every generator says what kind of value it produces", "TestEveryGeneratorSaysWhatKindOfValueItProduces", "internal/adapter/values", ByMutation},
	{"every generated article number sits in the prefix GS1 sets aside for demonstrations, so it cannot belong to a real article", "TestEveryGeneratedArticleNumberSitsInTheDemonstrationPrefix", "internal/adapter/provider/intl", ByMutation},
	{"the check digit weights are anchored at the right hand end of the value, as the standard prints them", "TestTheExamplesTheStandardsWorkComeOutTheSameHere", "internal/adapter/provider/intl", ByMutation},
	{"the independent validator takes its multipliers from the row the standard prints, so it is not a second copy of the rule the provider applies", "TestEveryGeneratedArticleNumberSatisfiesTheIndependentValidator", "internal/adapter/provider/intl", ByMutation},
	{"every generated equipment identity sits in the block the GSM Association gives to test equipment, so it cannot be a device supplied to the market", "TestEveryGeneratedEquipmentIdentitySitsInTheTestBlock", "internal/adapter/provider/intl", ByMutation},
	{"the example the specification works out itself comes out the same here, including the total it prints halfway through", "TestTheExampleTheSpecificationWorksComesOutTheSameHere", "internal/adapter/provider/intl", ByMutation},
	{"the equipment identity does not declare a reserved range, because codes inside the test block are issued to named manufacturers and nothing there is left free", "TestTheIdentityClaimsNoSafeHarbour", "internal/adapter/provider/intl", ByMutation},
	{"the sixteen digit form carrying a software version number is recognised and passed over rather than called wrong, because the specification applies no check digit to it", "TestTheSixteenDigitFormIsRecognisedRatherThanCalledWrong", "internal/adapter/provider/intl", ByMutation},
	{"the independent validator counts the products of step one against the number the specification names, so a relabelling that quietly weighs a different set of digits is refused rather than followed", "TestEveryGeneratedEquipmentIdentitySatisfiesTheIndependentValidator", "internal/adapter/provider/intl", ByMutation},
	{"the book number provider declares no reserved range, because it narrows generation into a window the agency has not allocated and that is not a harbour", "TestTheBookNumberClaimsNoSafeHarbour", "internal/adapter/provider/intl", ByMutation},
	{"every generated book number falls in the one window under prefix 978 that the agency leaves without a registration group, so no book carries the value today", "TestEveryGeneratedBookNumberSitsInTheUnallocatedWindow", "internal/adapter/provider/intl", ByMutation},
	{"the book number check digit uses the alternation of one and three counted from the left, which is what the manual prints every product of", "TestTheExampleTheManualWorksComesOutTheSameHere", "internal/adapter/provider/intl", ByMutation},
	{"the payment card approves fifteen digits only behind the American Express beginnings, so an equipment identity under the same formula is not taken for a card", "TestAnEquipmentIdentityIsNotACardThisBuildChecks", "internal/adapter/provider/intl", ByMutation},
	{"every scheme with a check digit is recognised in a column of its own values, so a new validator that accepts another scheme's values is caught the day it arrives", "TestEverySchemeIsRecognisedInAColumnOfItsOwnValues", "internal/adapter/infer", ByMutation},
	{"the payment card declares no reserved range, because nothing says no issuer has given a test number to a real card", "TestTheCardClaimsNoSafeHarbour", "internal/adapter/provider/intl", ByMutation},
	{"every card number this build writes is published as a test number by at least two of the four providers read, so the list is nobody's table", "TestEveryPublishedNumberHasTwoProvidersBehindIt", "internal/adapter/provider/intl", ByMutation},
	{"the card check digit and the independent validator both accept every published test number, which a wrong fold or a product added whole refuses", "TestEveryPublishedTestNumberSatisfiesTheCheckDigit", "internal/adapter/provider/intl", ByMutation},
	{"a scheme declaring that it lies inside another names only schemes this build carries, every value it produces is accepted by them, and they produce a value it refuses", "TestEveryNarrowerSchemeLiesStrictlyInsideTheSchemesItNames", "internal/adapter/values", ByMutation},
	{"a column of book numbers is recognised as isbn rather than refused, and the basis names the article number that accepted it too", "TestABookNumberColumnIsRecognisedAsTheNarrowerOfTwoSchemes", "internal/adapter/infer", ByMutation},
	{"the narrower scheme is chosen only when it names every other scheme that accepted the column, and never out of two that name each other", "TestTheNarrowerSchemeIsChosenOnlyWhenItNamesEveryOther", "internal/adapter/infer", ByMutation},
	{"a column of American Express numbers stays a refusal, because the card and the equipment identity accept it and neither lies inside the other", "TestAColumnTwoUnrelatedSchemesAcceptIsStillARefusal", "internal/adapter/infer", ByMutation},
	// 🔴 B-344: the pairing reads every identifier once, walks two columns
	// once and stops at the record that ends the last hypothesis. Each of
	// those is a way to answer differently from the rule, so the rule written
	// plainly is kept beside it and the two are put to the same tables.
	{"the pairing of two columns answers what comparing every record the plain way answers, over three thousand tables with gaps, unreadable identifiers and near misses", "TestThePairingAnswersWhatComparingEveryRecordWouldAnswer", "internal/adapter/infer", ByMutation},
	{"a date compared digit by digit answers what writing it out and comparing answers, for dates that fit a layout and dates that do not", "TestWritesAnswersWhatWritingTheDateOutWould", "internal/adapter/infer", ByMutation},
	{"one column of identifiers allocates as much against four hundred neighbours as against ten, so a pair of columns costs no allocation", "TestAPairOfColumnsAllocatesNothing", "internal/adapter/infer", ByMutation},
	{"the index of neighbours leaves out the columns the walk would end on within the first records, keeps every other, and speaks for no identifier column with a blank near its top", "TestOnlyColumnsThatCanAgreeAreWalked", "internal/adapter/infer", ByMutation},
	{"a swap that breaks a book number and leaves an article number whole is put back, so no rejection is promised that an article number system never makes", "TestASwapAWiderSchemeStillAcceptsIsNotKept", "internal/adapter/mutators", ByMutation},
	{"a swap into a form a wider scheme does not check is put back as well, because not checked is not rejected", "TestASwapAWiderSchemeDoesNotCheckIsNotKeptEither", "internal/adapter/mutators", ByMutation},
	{"every control a pointer can touch draws each of its states differently, so a person can tell hovering from pressing", "TestEveryControlAPointerCanTouchDrawsEachOfItsStatesDifferently", "internal/gui/parts", ByMutation},
	{"a control forgets a press the pointer carried away, because the driver delivers the release to whatever is under the pointer by then and never to the control that was pressed", "TestAControlForgetsAPressThePointerCarriedAway", "internal/gui/parts", ByMutation},
	{"a control nobody can use draws nothing for a pointer, however much of one the driver hands it", "TestADisabledControlDrawsNothingForAPointer", "internal/gui/parts", ByMutation},
	{"a kind with several forms says each length once and the noun once, so the payment card reads 15 or 16 digits", "TestAKindWithSeveralFormsSaysEachLengthOnce", "internal/gui/quickview", ByMutation},
	{"every value is written the way its own scheme says it is, so the shape a window shows beside ten values cannot drift from the values themselves", "TestEveryValueIsWrittenTheWayItsSchemeSays", "internal/adapter/values", ByMutation},
	{"every numbering scheme this build carries can say what its values look like, so no scheme reaches a surface with nothing to describe it", "TestEverySchemeSaysWhatItsValuesLookLike", "internal/adapter/values", ByMutation},
	{"the reader of a mask refuses what a mask forbids, so a mask that agrees with everything cannot pass for one that works", "TestAMaskRefusesTheThingsItIsMeantTo", "internal/adapter/values", ByMutation},
	{"every mask fits the panel it is drawn in, because canvas text does not wrap and a mask too long widens the whole panel instead of overflowing", "TestEveryMaskFitsThePanelItIsDrawnIn", "internal/gui/quickview", ByMutation},
	{"the note a run leaves beside the data says that every value in it is synthetic, so a folder of national identifiers says from the folder alone that they describe nobody", "TestTheNoteBesideTheDataSaysTheValuesAreSynthetic", "internal/produce", ByMutation},
	{"the note that file_per_entity changes nothing is said only when the recipe wrote that setting, so a recipe that never mentions it is not told about a line nobody wrote", "TestASettingNobodyWroteIsNotReportedAsChangingNothing", "internal/produce", ByMutation},
	{"moving the directory of a run records both the old and the new place and leaves the recipe pointing where it was moved", "TestMovingTheDirectoryRecordsBothSidesAndMovesTheRecipe", "internal/produce", ByMutation},
	{"a directory given on the command line is where the run writes, and the note beside the data records it as a move from the directory the recipe named", "TestADirectoryFromTheCommandLineIsWhereTheRunWritesAndTheNoteSaysSo", "internal/cli", ByMutation},
	{"every setting of a run reaches it from the recipe screen, the seed and the output form through the recipe and the extra form of the manifest through its own control, which is the parity with generate", "TestEverySettingOfARunReachesItFromTheScreen", "internal/gui/recipeview", ByMutation},
	{"the line for a terminal on the recipe screen carries the extra form of the manifest that was picked, and no form when none was", "TestTheLineForATerminalCarriesTheFormThatWasPicked", "internal/gui/recipeview", ByMutation},
	{"the note a run from the recipe screen leaves records the directory from its field as a move from the recipe, the way the note from a terminal records --out", "TestTheNoteSaysTheDirectoryCameFromTheScreenTheWayItSaysItOfTheFlag", "internal/gui/recipeview", ByMutation},
	{"a control without a label stands level with one that has a label beside it, because the room left for the missing label is the height of the label itself", "TestAControlWithoutALabelStandsLevelWithOneThatHasIt", "internal/gui/paint", ByMutation},
	{"a text control out of reach looks like a chooser out of reach, answers neither the pointer nor the keyboard, and comes back the way they left it", "TestATextControlOutOfReachLooksLikeAChooserOutOfReach", "internal/gui/parts", ByMutation},
	{"a text control disabled before a screen first draws it is drawn out of reach from the start, rather than wearing the edge of a control that can be used", "TestATextControlDisabledBeforeItIsDressedIsDressedOutOfReach", "internal/gui/parts", ByMutation},
	{"the catalogue of controls shows every control a screen builds, each once, and no control that no screen builds", "TestTheCatalogueShowsEveryControlTheScreensBuildAndNothingElse", "internal/gui/catalogue", ByMutation},
	{"every control in the catalogue is shown in exactly the states its drawing answers to, so no column repeats the one before it and no state that changes the picture is left out", "TestEveryControlIsShownInExactlyTheStatesItsDrawingAnswersTo", "internal/gui/catalogue", ByMutation},
	{"every variant on a page of the catalogue builds the control its entry names, so the name the register is closed by is the name of what is really on the page", "TestEveryVariantBuildsTheControlItsEntryNames", "internal/gui/catalogue", ByMutation},
	{"every page of the catalogue of controls has a picture of its own and fits the window it opens in, so no control is left where no picture shows it", "TestEveryPageOfTheCatalogueHasAPictureAndFitsTheWindowItOpensIn", "internal/gui/screens", ByMutation},
	{"the window opens what its command line asks for, the catalogue for its flag and help for -h, and refuses with no window any flag it does not know and any word left over", "TestTheWindowOpensWhatItsCommandLineAsksForAndRefusesTheRest", "cmd/qadg-gui", ByMutation},
	{"help for the window names the flag that opens the catalogue of controls and writes nothing to the error channel", "TestHelpNamesTheFlagOfTheCatalogue", "cmd/qadg-gui", ByMutation},
	{"a check box is ticked and unticked by a tap, by the space bar and by Enter, tells its callback each time, and draws its mark only while it is ticked", "TestACheckBoxTogglesByTapAndByKeyAndDrawsItsMark", "internal/gui/parts", ByMutation},
	{"a check box set from code tells its callback once and only when its state really changes, and draws its mark", "TestSettingACheckBoxFromCodeTellsTheCallbackOnceAndOnlyOnAChange", "internal/gui/parts", ByMutation},
	{"a check box out of reach keeps its mark, so the layers a run in flight was ordered with can still be read, and ignores a tap", "TestADisabledCheckBoxKeepsItsMarkAndIgnoresATap", "internal/gui/parts", ByMutation},
	{"a grouped panel of choices opens with the keyboard in its search field and on the row already chosen, so a person can start typing at once", "TestOpeningAGroupedPanelPutsTheKeyboardInTheSearchField", "internal/gui/parts", ByMutation},
	{"typing in the panel of choices leaves the rows whose name, written form or heading holds the typed words, shows only their headings, and the foot counts how many match of how many there are", "TestTypingFiltersTheRowsAndTheFootCountsThem", "internal/gui/parts", ByMutation},
	{"a filter that matches nothing hides the rows, says in a sentence that nothing matches, counts nought in the foot, and Enter then neither picks nor closes", "TestAFilterThatMatchesNothingSaysSoAndEnterDoesNothing", "internal/gui/parts", ByMutation},
	{"the arrow keys walk the rows of the panel of choices up and down, and Enter picks the row the keyboard is on, tells the screen and closes the panel", "TestTheArrowsWalkTheRowsAndEnterPicksOne", "internal/gui/parts", ByMutation},
	{"the arrow keys held down in the panel of choices arrive at the first and the last row and stay there, rather than wrapping round or stopping short of either end", "TestTheArrowsStopAtTheEndsRatherThanWrapping", "internal/gui/parts", ByMutation},
	{"Escape and a tap outside the panel of choices both close it and leave the choice as it was", "TestEscapeAndATapOutsideCloseWithoutChoosing", "internal/gui/parts", ByMutation},
	{"a tap on a row of the panel of choices picks that row and closes the panel", "TestATapOnARowPicksIt", "internal/gui/parts", ByMutation},
	{"a flat list of choices opens without a search field or a foot, as wide as the chooser it opens under, and still answers the keyboard by first letter and Enter", "TestAFlatListOpensWithoutAFieldAndStillAnswersTheKeyboard", "internal/gui/parts", ByMutation},
	{"a chooser asked to open its panel while it is already open keeps the one panel rather than stacking a second over it", "TestOpeningTwiceOpensOnce", "internal/gui/parts", ByMutation},
	{"the panel of choices stays inside the window, slid left near the right edge, opened above near the foot, and no taller than the room it has", "TestThePanelStaysInsideTheCanvas", "internal/gui/parts", ByMutation},
	{"in the panel of choices the chosen row carries the check on the ground of the chosen and the row the keyboard is on carries the ground a pointer gives a row, each on its own row, and no row is outlined", "TestTheChosenRowIsMarkedAndTheKeyboardRowIsLit", "internal/gui/parts", ByMutation},
	{"a text field dressed twice is dressed once, so the ring of focus is drawn on the dressing a screen put on itself", "TestAFieldDressedTwiceKeepsTheMarksOnTheFirstDressing", "internal/gui/parts", ByMutation},
	{"a text field under the pointer stands on the ground a quiet button stands on, keeps the ring of focus on it when the keyboard arrives too, and comes back to rest when both leave", "TestAFieldAnswersThePointerTheWayAQuietButtonDoes", "internal/gui/parts", ByMutation},
	{"every heading the list of kinds on the Quick screen stands a kind under is a sentence of the catalogue in every locale, never a key or the code of a country", "TestEveryGroupHeadingIsWordedInTheCatalogue", "internal/gui/quickview", ByMutation},
	{"the list of kinds on the Quick screen opens with the country of the locale, then the other countries, then what belongs to no country, then people and places", "TestTheHeadingsStartWithTheCountryOfTheLocale", "internal/gui/quickview", ByMutation},
	{"every kind on the list of the Quick screen with a mask says at its end how long a value of it is, in the words of the card beside the values, and a kind without one says nothing there", "TestAKindWithAMaskSaysHowLongItIs", "internal/gui/quickview", ByMutation},
	{"every kind on the list of the Quick screen stands beside the name a person calls it, a sentence of the catalogue in every locale and never a key", "TestEveryKindIsNamedInTheCatalogue", "internal/gui/quickview", ByMutation},
	{"at every stop of the keyboard on every screen exactly one ring of focus is mounted on the screen, so no control believes it drew a mark that nobody sees", "TestEveryStopOfTheKeyboardDrawsExactlyOneFocusRing", "internal/gui/screens", ByMutation},
	{"the outcome column of the manifest browser shows the kind of failure for damage that did not happen whatever an application said, the verdict of the application for damage that did, and applied until then, and the card carries the rest", "TestTheOutcomeColumnAndTheCardDivideAnEntryBetweenThem", "internal/gui/manifestview", ByMutation},
	{"the card under the table of the manifest browser reads out the entry the keyboard stands on and follows it from row to row", "TestTheCardReadsOutTheEntryTheKeyboardStandsOn", "internal/gui/manifestview", ByMutation},
	{"the terminal line of the manifest browser names both files once both are open, and until then says which name is an example and keeps its copy out of reach", "TestTheTerminalLineNamesBothFilesOrSaysWhichIsAnExample", "internal/gui/manifestview", ByMutation},
	{"a manifest the browser could not read is named in the band beside the word that it could not be read, and the card under the table says what went wrong", "TestADocumentThatCouldNotBeReadIsNamedInTheBandAndExplainedInTheCard", "internal/gui/manifestview", ByMutation},
	{"every word the toolkit asks for is handed to it or left out with a reason, and every word handed to it is one it still asks for", "TestTheWordsHandedToTheToolkitAreTheWordsItAsksFor", "internal/gui/fynetheme", ByMutation},
	{"a run undone while replacing gives the earlier file back rather than leaving nothing under its name", "TestAnUndoneReplacementGivesTheEarlierFileBack", "internal/adapter/store", ByMutation},
	{"a replacement that succeeds keeps no hidden copy of what it replaced", "TestAReplacementThatSucceedsLeavesNoCopyBehind", "internal/adapter/store", ByMutation},
	{"a directory standing where a file of the dataset goes is refused before any work, with replacing allowed or not", "TestADirectoryUnderTheNameIsRefusedBeforeAnythingIsWritten", "internal/adapter/store", ByMutation},
	{"a source named after a file the run leaves beside the data is never written over, whatever form of the manifest was asked for", "TestASourceNamedAfterTheManifestIsNotWrittenOver", "internal/take", ByMutation},
	{"two names for one file that text cannot tell apart are seen as one once the file exists", "TestALinkToTheSourceIsSeenAsTheSource", "internal/take", ByMutation},
	{"a record of one blank field is written so that a reader still reads a record", "TestABlankRecordOfOneColumnIsStillARecord", "internal/adapter/format", ByMutation},
	{"a separator is one character other than a quote, a line end or a zero byte", "TestASeparatorIsOneCharacterTheFormatDoesNotAlreadyUse", "internal/domain/dataset", ByMutation},
	{"a field described by type alone that cannot hold the shortest value of its type is refused before the run", "TestALimitTooShortForTheTypeIsRefusedBeforeTheRun", "internal/adapter/values", ByMutation},
	{"the tree in one file never gives one object two members of the same name", "TestATreeRefusesAParentFieldNamedAfterAChild", "internal/adapter/format", ByMutation},
	{"a value in a copied command reaches the shell as one argument and as nothing else", "TestAValueReachesTheShellAsOneArgumentAndNothingElse", "internal/commandline", ByMutation},
	{"a bracket inside a string or a comment closes nothing, so it buys no room for nesting past the limit", "TestABracketInsideAStringOrACommentBuysNoRoom", "internal/adapter/recipeyaml", ByMutation},
	{"a recipe naming no seed is read with a fixed one, so asking for damage leaves a manifest rather than an internal defect", "TestARecipeWithoutASeedIsDamagedWithTheSeedItIsReadWith", "internal/produce", ByMutation},
	{"the record beside the data names every setting the reader of the recipe carried without reading", "TestTheNoteNamesWhatTheReaderCarriedWithoutReading", "internal/produce", ByMutation},
	{"a unique column drawing from a list walks to an entry still free, so a table as long as the list uses the list up whole", "TestAUniqueListIsUsedUpWholeAndTheSameEveryTime", "internal/adapter/format", ByMutation},
	{"a unique column whose values are known walks to a free one when drawing runs out of luck, so a hundred rows of a hundred values all get one", "TestAUniqueNumberFillsItsWholeDomain", "internal/adapter/format", ByMutation},
	{"the run holds the primary key whole, so no two rows share a key while its parts repeat on their own", "TestAKeyHeldWholeNeverRepeatsWhileItsPartsDo", "internal/adapter/format", ByMutation},
	{"the plan refuses a table with more rows than the fields of its key can make different keys, before a row is written", "TestAKeyIsRefusedWhenItsFieldsCannotMakeEnoughKeys", "internal/app/generate", ByMutation},
	{"the plan refuses a unique field whose known values are fewer than its rows, with the arithmetic", "TestAUniqueFieldWithTooFewKnownValuesIsRefusedBeforeTheRun", "internal/app/generate", ByMutation},
	{"an entry under values that does not fit the type or the length beside it is refused by its place in the list", "TestAListEntryTheFieldCannotHoldIsRefusedByItsPlace", "internal/app/generate", ByMutation},
	{"every value a request draws lies in the domain the same request reports, and the domain counts each value once", "TestEveryDrawnValueLiesInsideTheDomainItsRequestReports", "internal/adapter/values", ByMutation},
	// 🔴 B-348, 2026-11-03: every row that walked asked for the domain again, so a
	// unique list of 28000 entries took 2.8 s where the table without unique
	// took 0.077 s. The three below hold the answer kept, kept per field, and
	// never kept for a field whose domain moves with its row.
	{"a domain is the same at every address, so a field no rule binds can be asked once and the answer kept for every row", "TestADomainIsTheSameAtEveryAddress", "internal/adapter/values", ByMutation},
	{"the run asks once for the domain of a unique field that is the same in every row, however many rows walk it, and each field keeps its own answer", "TestADomainTheSameInEveryRowIsAskedForOnce", "internal/adapter/format", ByMutation},
	{"a unique field bound to the gender walks the domain of the gender of its row, so a table that uses up the names of women and then of men gives every row a name of its own gender", "TestAUniqueFieldBoundToTheGenderWalksTheDomainOfItsRow", "internal/adapter/format", ByMutation},
	{"a unique column whose values are not counted says what happened when drawing runs out and claims nothing about how many values there are", "TestAColumnThatCannotBeCountedSaysWhatHappenedAndNoMore", "internal/adapter/format", ByMutation},
	{"a list under values beside a generator or an expression is refused, because the field would name two places its value comes from", "TestAListUnderValuesIsRefusedBesideAnotherSource", "internal/adapter/recipeyaml", ByMutation},
	{"a part of the primary key asked to be empty in some rows is refused by the reader", "TestAPartOfTheKeyAskedToBeEmptyIsRefused", "internal/adapter/recipeyaml", ByMutation},
	{"a list under values is where a value comes from on its own or with a type beside it", "TestAListUnderValuesIsASourceOnItsOwnOrWithAType", "internal/adapter/recipeyaml", ByMutation},
	{"a field with a list under values draws from the list and from nothing else", "TestAFieldWithAListDrawsFromItAlone", "internal/adapter/values", ByMutation},
	{"a row built again for its key gives back what its unique columns took the first time", "TestARowBuiltAgainForItsKeyGivesBackWhatItTook", "internal/adapter/format", ByMutation},
	{"the domain of a number described by its type is counted with the digits the draw uses", "TestTheDomainOfANumberIsCounted", "internal/adapter/values", ByMutation},
	{"an entry written twice in a list under values is refused at the place of the second one", "TestAnEntryWrittenTwiceInTheListIsRefused", "internal/adapter/recipeyaml", ByMutation},
	{"an empty entry in a list under values is refused, because blank rows are what the share under empty says", "TestAnEmptyListOrAnEmptyEntryIsRefused", "internal/adapter/recipeyaml", ByMutation},
	{"the plan leaves a field a rule binds to the run, because its domain depends on the row", "TestAFieldARuleBindsIsLeftToTheRun", "internal/app/generate", ByMutation},
	{"two domains of one column, a name for a woman and a name for anybody, carry different names", "TestTwoDomainsOfOneColumnAreToldApart", "internal/adapter/values", ByMutation},
	{"an entry under values beside a type of date is held to a real day written as year-month-day", "TestAnEntryUnderValuesIsHeldAgainstItsTypeAndLength", "internal/adapter/values", ByMutation},
	{"a unique column bound to a rule says when every value it can take in a row is used rather than repeating one", "TestAUniqueFieldBoundToARuleSaysWhenEveryValueIsUsed", "internal/adapter/format", ByMutation},
	{"a walk goes round the end of its domain to the one place still free and says so when nothing is left", "TestAWalkGoesRoundToTheLastFreePlace", "internal/app/generate", ByMutation},
	{"a row of a dataset of several tables is judged by the answer about its own table rather than by the first answer for its number", "TestARowIsJudgedByTheAnswerAboutItsOwnTable", "internal/app/verify", ByMutation},
	{"records nobody reacted to are counted table by table, so rows of one table never make up for rows of another", "TestRecordsNobodyReactedToAreCountedTableByTable", "internal/app/verify", ByMutation},
	{"a row past the end of its table refuses the record, since it is usually every row shifted by one", "TestARowThatCannotBePlacedRefusesTheRecord", "internal/app/verify", ByMutation},
	{"rows refused for one reason are one sentence naming a few of them, so a record of every row without an entity is not a message of every row", "TestRowsRefusedForOneReasonAreOneSentence", "internal/app/verify", ByMutation},
	{"a manifest of 0.1 with entries in several tables is read as several tables held to the dataset as a whole, and the report says so", "TestAManifestOfTheEarlierVersionIsReadByWhatItCanSay", "internal/app/verify", ByMutation},
	{"a response record of a newer minor version is read and the report says among its limits that the newer part was not read", "TestANewerDocumentIsSaidAmongTheLimits", "internal/app/verify", ByMutation},
	{"the window is answered from the places the comparison judged with, so an entry of one table never shows the answer about another", "TestAnEntryIsAnsweredFromItsOwnTable", "internal/app/verify", ByMutation},
	{"a member this build does not know is refused in a version it knows and skipped in a newer minor one", "TestAnUnknownMemberIsRefusedOrSkippedByTheVersion", "internal/adapter/responsejson", ByMutation},
	{"the entity a row of a response record names is read with the row", "TestAnEntityIsReadWithItsRow", "internal/adapter/responsejson", ByMutation},
	{"the same number under two entities is two rows of the record, and only one place named twice is a row answered twice", "TestTheSameNumberUnderTwoEntitiesIsTwoRows", "internal/domain/response", ByMutation},
	{"rows of a response record wrong the same way are one sentence with a few of their numbers, so four hundred thousand repeats are not four hundred thousand sentences", "TestRowsWrongTheSameWayAreSaidOnce", "internal/domain/response", ByMutation},
	{"problems joined into one refusal begin every later sentence with a capital, and a fragment led by something from a file is left exactly as written", "TestEverySentenceAfterTheFirstBeginsWithACapitalAndADataValueIsLeftAlone", "internal/fault", ByMutation},
	{"rows refused for one reason name three of their numbers and count the rest, with the verb agreeing", "TestRowsRefusedForOneReasonAreSaidOnce", "internal/fault", ByMutation},
	{"two problems of a manifest read back from disk are two sentences", "TestTwoProblemsReadBackAreTwoSentences", "internal/adapter/manifestjson", ByMutation},
	{"a response record that cannot be compared is refused with every reason, each a sentence of its own", "TestARecordThatCannotBeComparedIsRefusedWithAReason", "internal/adapter/responsejson", ByMutation},
	{"a request a verb cannot meet sends it to what it was given, the recipe, the file, the documents or the flags", "TestADataFailureSendsEachVerbToWhatItWasGiven", "internal/cli", ByMutation},
	{"a list of entities that disagrees with itself, with the count beside it or with the entries under it is refused by the model", "TestAListOfEntitiesThatDisagreesIsRefused", "internal/domain/manifest", ByMutation},
	{"a manifest of 0.2 or later without its list of entities is refused, and one of 0.1 is read without it", "TestTheListOfEntitiesIsRequiredFromTheVersionThatAddedIt", "internal/adapter/manifestjson", ByMutation},
	{"the writer of a manifest refuses an envelope that lists no entity or leaves out the table of an entry", "TestTheEntitiesAreWrittenWholeOrNotAtAll", "internal/adapter/manifestjson", ByMutation},
	{"verify names the table of a finding when the dataset has several, and refuses a row naming none with the code for an invalid input", "TestARecordAboutTwoTablesIsHeldToEachOfThem", "internal/cli", ByMutation},
	{"the version of the response format this build reads is the one its specification states", "TestResponseVersionMatchesTheResponseSpecification", "internal/version", ByMutation},
	// 🔴 B-338, DEC-271: an empty list of rows is an answer, every row going
	// missing or, for a dataset of no rows, full agreement. A list left out or
	// written as null is not one, and was the same refusal until 2026-11-03.
	{"a response record listing no row is a record, and one leaving the list out is refused", "TestAnEmptyListOfRowsIsAnAnswerAndNoListIsNot", "internal/domain/response", ByMutation},
	{"the reader tells an empty list of rows from a list left out or written as null", "TestAnEmptyListOfRowsIsReadAndNoListIsRefused", "internal/adapter/responsejson", ByMutation},
	{"a record listing no row is compared like any other, agreeing with a dataset of no rows without limits about rows it does not have", "TestARecordListingNoRowIsComparedLikeAnyOther", "internal/app/verify", ByMutation},
	{"verify on the command line ends an empty list of rows with 0 for a dataset of no rows and with 4 for one with rows, and refuses a list left out with 3", "TestAnEmptyListOfRowsEndsAsWhatItSays", "internal/cli", ByMutation},
	// 🔴 B-97, DEC-272: a verb reads its flags wherever they stand after it.
	// Four verbs of five stopped at the first path until 2026-11-03 and
	// refused a flag after it as a path too many.
	{"the flags of a verb are read wherever they stand, a flag taking a value takes the next argument, a switch takes none, and a double dash ends the flags", "TestTheFlagsOfAVerbAreReadWhereverTheyStand", "internal/cli", ByMutation},
	{"every verb answers --help written after its path, help included", "TestEveryVerbAnswersHelpAskedForAfterItsPath", "internal/cli", ByMutation},
	{"the same flags on the other side of the path give the same run on both channels, for every verb", "TestAFlagAfterThePathMeansWhatItMeansBeforeIt", "internal/cli", ByMutation},
	{"a flag written last without its value is told it takes one rather than that it does not exist", "TestAFlagWrittenLastWithoutItsValueIsToldItTakesOne", "internal/cli", ByMutation},
	// 🔴 B-359: help read every argument after it as the name of a command,
	// so each flag its own help lists as accepted after the command was
	// refused there, --help included, until 2026-11-03.
	{"every flag of the tool written after help gives the run it gives before help, and --help after help prints the help of help", "TestTheFlagsOfTheToolMeanTheSameAfterHelpAsBeforeIt", "internal/cli", ByMutation},
	{"a command written as a flag after help is refused as a flag and the refusal still names the command", "TestACommandWrittenAsAFlagAfterHelpIsStillSuggested", "internal/cli", ByMutation},
	// 🔴 B-336: an identifier refuses a bound value it cannot read without a
	// kind, which is right only while no value a person writes reaches it.
	{"every value a generator playing a role draws is taken by every generator reading that role, in every locale", "TestEveryValueABoundGeneratorDrawsIsOneItsReadersTake", "internal/adapter/values", ByMutation},
	{"the window attaches a finding to the entry of its own table, matched on the entity as well as the row, the column and the mutator", "TestAnEntryOfOneTableIsNotGivenTheFindingOfAnother", "internal/gui/manifestview", ByMutation},
	{"a run on one screen ending does not give the window back while a run on another is still writing", "TestTheWindowIsHeldUntilTheLastRunLetsGo", "internal/gui/background", ByMutation},
	{"asking for the window stops every run going on and the window goes once the last of them has stopped", "TestAskingForTheWindowStopsEveryRunAndGoesAfterTheLast", "internal/gui/background", ByMutation},
	{"a run begun after somebody asked to leave is stopped at once rather than waited for", "TestARunBegunAfterTheAskToLeaveIsStoppedAtOnce", "internal/gui/background", ByMutation},
	{"a signal that arrives once nothing is held gives the window up at once and leaves nothing set for the next run", "TestASignalAfterTheLastRunGivesTheWindowUpAndArmsNothing", "internal/gui/background", ByMutation},
	{"a run letting go twice ends no other run holding the window", "TestLettingGoTwiceEndsNoOtherHolding", "internal/gui/background", ByMutation},
	{"one question is in flight at a time and of those asked meanwhile only the newest is asked next", "TestOnlyTheNewestOfTheQuestionsAskedMeanwhileIsAsked", "internal/gui/background", ByMutation},
	{"a read that comes apart frees the queue for the question waiting behind it", "TestAReadThatComesApartFreesTheQueue", "internal/gui/background", ByMutation},
	{"a read with nothing to do about it still frees the queue", "TestAReadWithNothingToDoStillFreesTheQueue", "internal/gui/background", ByMutation},
	{"the queue is free again once a read done where it was asked for has answered", "TestTheQueueIsFreeOnceAReadHereHasAnswered", "internal/gui/background", ByMutation},
	{"a separator typed while the file is being read waits for that read rather than reading the file again beside it", "TestASeparatorTypedWhileTheFileIsReadWaitsForTheRead", "internal/gui/takeview", ByMutation},
	{"letters typed during a check of the recipe wait for it and only the last of them is checked", "TestLettersTypedDuringACheckWaitForIt", "internal/gui/recipeview", ByMutation},
	{"the comparison is worked out where the response record is read, on the manifest taken when the read was asked for", "TestTheComparisonIsWorkedOutWhereTheRecordIsRead", "internal/gui/manifestview", ByMutation},
	{"a comparison worked out against a manifest that has since been replaced is not drawn beside the one now open", "TestAComparisonWithAManifestNoLongerOpenIsNotDrawn", "internal/gui/manifestview", ByMutation},
	{"every screen is handed every place of work, the closing of the window among them", "TestEveryScreenIsHandedEveryPlaceOfWork", "cmd/qadg-gui", ByMutation},
	{"every row of a child table belongs to the parent the plan shared it out to, parents without children included", "TestEveryRowBelongsToTheParentAParentPerChildGaveIt", "internal/domain/dataset", ByMutation},
	{"a preview cut to its first rows keeps the parent of every row it keeps and holds nothing past them", "TestACutKeepsTheParentOfEveryRowItKeeps", "internal/domain/dataset", ByMutation},
	{"a child of an empty table is a mapping of no rows, and no row of it has a parent", "TestNoParentsIsAMappingOfNoRows", "internal/domain/dataset", ByMutation},
	{"planning a relation holds a word for every parent and nothing for every child", "TestAPlanHoldsAWordPerParentRatherThanPerChild", "internal/app/generate", ByMutation},
	{"a plan asked to stop before a table it has not started stops there", "TestAStopBeforeATableEndsThePlan", "internal/app/generate", ByMutation},
	{"a plan asked to stop while it shares children out between the parents stops there", "TestAStopWhileChildrenAreSharedOutEndsThePlan", "internal/app/generate", ByMutation},
	{"a plan asked to stop while it repairs a count stops within a round of the parents", "TestAStopWhileACountIsRepairedEndsThePlan", "internal/app/generate", ByMutation},
	{"the file of a table is closed when the table ends, before the next one opens", "TestTheFileOfATableIsClosedBeforeTheNextOneOpens", "internal/adapter/format", ByMutation},
	{"a file that could not be closed is never reported as written", "TestAFileThatCouldNotBeClosedIsNeverReportedAsWritten", "internal/adapter/format", ByMutation},
	{"a writer that streams keeps nothing of the rows it has already written", "TestAWriterThatStreamsKeepsNothingOfTheRowsItWrote", "internal/adapter/format", ByMutation},
	{"the tree puts every child under the parent it names, in table order, whatever order the parents come in", "TestTheTreePutsEveryChildUnderItsParentWhateverOrderTheyArriveIn", "internal/adapter/format", ByMutation},
	{"a file its writer closed before the commit is committed like any other", "TestAFileClosedByItsWriterIsStillCommitted", "internal/adapter/store", ByMutation},
	{"a close that failed when the writer asked still stops the commit", "TestACloseThatFailedStopsTheCommit", "internal/adapter/store", ByMutation},
	{"a plan asked to stop while it draws the order a repair walks the parents in stops there", "TestAStopWhileTheOrderOfARepairIsDrawnEndsThePlan", "internal/app/generate", ByMutation},
	{"a record of exactly the limit is read and one a byte longer is refused, naming its line", "TestARecordOfTheLimitIsReadAndOneByteLongerIsRefused", "internal/adapter/sourcecsv", ByMutation},
	{"a quote left open is refused once the record passes the limit, before the rest of the file is read into memory", "TestAnUnclosedQuoteIsRefusedAtTheLimitRatherThanAtTheEnd", "internal/adapter/sourcecsv", ByMutation},
	{"the refusal of a record too long names the line it starts on, counting records that span lines", "TestTheLineOfARecordTooLongCountsTheLinesBeforeIt", "internal/adapter/sourcecsv", ByMutation},
	{"a file as wide as a spreadsheet is read and one column wider is refused before anything is kept for its columns", "TestAFileWiderThanASpreadsheetIsRefused", "internal/adapter/sourcecsv", ByMutation},
	{"a directory, a device or a pipe is refused as a source by name", "TestASourceThatIsNotAFileIsRefused", "internal/adapter/sourcecsv", ByMutation},
	{"a column keeps the values that fit its share and nothing after the first that does not", "TestAColumnKeepsItsFirstValuesUpToItsShare", "internal/adapter/sourcecsv", ByMutation},
	{"a value kept in the sample does not keep the record it was cut from", "TestAKeptValueDoesNotKeepItsRecord", "internal/adapter/sourcecsv", ByMutation},
	{"a stop between records ends the first pass over a file and lets the file go", "TestAStopBetweenRecordsEndsTheSurveyAndLetsTheFileGo", "internal/adapter/sourcecsv", ByMutation},
	{"a question asked again is told to stop, and what it answers is dropped", "TestAQuestionAskedAgainIsStoppedAndItsAnswerDropped", "internal/gui/background", ByMutation},
	{"a check of a recipe asked again is told to stop", "TestACheckAskedAgainIsToldToStop", "internal/gui/recipeview", ByMutation},
	{"a read of a file asked again is told to stop", "TestAReadAskedAgainIsToldToStop", "internal/gui/takeview", ByMutation},
	{"the stop of a preview reaches the plan, which shares the children of every parent out", "TestTheStopOfAPreviewReachesThePlan", "internal/produce", ByMutation},
	{"a separator typed while a newly chosen file is read goes to that file", "TestASeparatorTypedWhileAChosenFileIsReadGoesToThatFile", "internal/gui/takeview", ByMutation},
	{"the stop handed to take.Open reaches the reading of the file", "TestAReadingToldToStopEndsBeforeTheWholeFileIsRead", "internal/take", ByMutation},
	// 🔴 B-356: the working out that comes after the reading asks as well,
	// once a column and once a column of identifiers, so a read nobody wants
	// any more ends within one column of a wide file rather than after all.
	{"working out a table asks whether to stop before every column and before every column of identifiers it pairs, and a stop at any of those ends it with no shape", "TestWorkingOutATableAsksToStopBeforeEveryColumnAndEveryPairing", "internal/adapter/infer", ByMutation},
	{"the stop handed to take.Open reaches the working out of the columns, and a read stopped there lets the file go", "TestAReadStoppedWhileItsColumnsAreWorkedOutEndsAndLetsTheFileGo", "internal/take", ByMutation},
	{"the real check of a recipe hands its stop to the preview", "TestTheRealCheckHandsItsStopToThePreview", "internal/gui/recipeview", ByMutation},
	{"a forgotten question is told to stop, its answer is dropped and nothing waiting behind it is asked", "TestAForgottenQuestionIsStoppedAndNothingWaitingIsAsked", "internal/gui/background", ByMutation},
	{"an answer that arrives after the recipe screen was emptied is not drawn over it", "TestAnAnswerArrivingAfterTheRecipeWasClearedIsNotDrawn", "internal/gui/recipeview", ByMutation},
	{"the first error the receiver of quick values returns ends the drawing, so a count with no ceiling stops where its reader stops", "TestEachHandsOutAHugeCountOneValueAtATime", "internal/quick", ByMutation},
	{"an error of the engine ends the drawing of quick values before the receiver is handed anything", "TestEachEndsWhereTheEngineRefuses", "internal/quick", ByMutation},
	{"a batch of data lines is written only when the next line would not fit, so it fills the size a pipe takes whole exactly and never splits a line", "TestABatchIsWholeLinesAndAsLargeAsAPipeTakesWhole", "internal/cli", ByMutation},
	{"every value quick draws reaches the data channel in order, the last batch that nothing later pushes out included", "TestQuickWritesEveryValueItDrewInOrder", "internal/cli", ByMutation},
	{"quick stops drawing at the first failed write to standard output and writes nothing after it", "TestQuickStopsDrawingAtTheFirstWriteThatFails", "internal/cli", ByMutation},
	{"a quick run that drew its own seed names it on the log, and that seed gives the same values again", "TestADrawnSeedIsSaidAndBringsTheSameValuesBack", "internal/cli", ByMutation},
	{"a quick run whose engine refused the first value names no seed, because nothing was drawn for it to bring back", "TestNoSeedIsSaidWhenTheEngineRefusedTheFirstValue", "internal/cli", ByMutation},
	{"the entries of a word list remembered under a limit on length are keyed by the list, the limit and the unit", "TestAListUnderALimitIsTheSameWhicheverLimitWasAskedFirst", "internal/adapter/values", ByMutation},
	{"a value of a word list under a limit on length is drawn from the remembered entries rather than measuring the list again", "TestAValueUnderALimitIsDrawnWithoutMeasuringTheListAgain", "internal/adapter/values", ByMutation},
	{"a failed write to standard output does not send the reader to check a path that does not exist", "TestAFailedWriteToTheOutputDoesNotSendTheReaderToAPath", "internal/cli", ByMutation},
	{"a date of birth a PESEL cannot carry is refused as a request that cannot be met, naming the date and the years it could have been", "TestADateOfBirthThePeselCannotCarryIsARequestThatCannotBeMet", "internal/cli", ByMutation},
	{"a date the PESEL scheme cannot express is refused with the date named and a kind saying the request cannot be met", "TestADateOutsideTheSchemeIsRefused", "internal/adapter/provider/pl", ByMutation},
}
