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
// It stands at one, and the one is honest rather than lazy: the rule that the
// command line binary never reaches the window cannot be broken on purpose
// while there is no window to reach. A mutation adding that import would not
// compile, so it would report a broken tree instead of a caught fault. This
// drops to nought on the day the graphical half exists.
const UnprovenCeiling = 1

// Guards is the register.
//
// It is deliberately not every test in the project. A test that checks one
// function does its arithmetic is not a guard. A guard is a test standing
// between the product and a promise it makes to somebody outside it.
var Guards = []Guard{
	// --- the architecture guards, all of them in this package ---------------
	{"no package can reach outside this machine", "TestRepositoryOpensNothing", "internal/guard", ByMutation},
	{"the scanner sees an import that reaches outside", "TestNetworkCatchesAnImportThatReachesOutside", "internal/guard", ByProbe},
	{"parsing an address is not talking to one", "TestNetworkAllowsAddressParsing", "internal/guard", ByProbe},
	{"the dependency rule holds across the whole tree", "TestRepositoryRespectsTheLayers", "internal/guard", ByMutation},
	{"the scanner sees the domain reaching upwards", "TestLayersCatchesTheDomainReachingUpwards", "internal/guard", ByMutation},
	{"the scanner sees a use case reaching an adapter", "TestLayersCatchesAUseCaseReachingAnAdapter", "internal/guard", ByMutation},
	{"the scanner sees the base package depending on ours", "TestLayersCatchesTheBasePackageDependingOnUs", "internal/guard", ByProbe},
	{"a package may be split into files without the rule noticing", "TestLayersAllowsAPackageToImportItsOwnSubpackage", "internal/guard", ByProbe},
	{"an indirect import is found, not only a direct one", "TestReachFindsAnIndirectPath", "internal/guard", ByProbe},
	{"the command line binary never pulls in the window", "TestCommandLineBinaryNeverReachesTheWindow", "internal/guard", Unproven},
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
}

// CountByProof returns how many guards are in each state.
func CountByProof() map[ProofState]int {
	counts := make(map[ProofState]int, 3)
	for _, guard := range Guards {
		counts[guard.Proof]++
	}
	return counts
}
