package guard

// The part of the register about the window: its values, its visual system,
// what it may ask the toolkit, and what it says and how the keyboard moves
// through it.
//
// It is a continuation of proof.go and not a register of its own. It was cut
// out on 2026-10-28, when the half about the tool reached 509 lines of code
// against a ceiling of 500 that may be lowered but never raised. The window is
// one surface with its own sections, which makes it a seam rather than an
// arbitrary line, and every tool that reads the register reads every proof*.go
// of this package.
var windowGuards = []Guard{
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
	{"the scanner sees a sidebar entry built without an icon, in either of the two shapes the two copies write", "TestTheSidebarGuardSeesABareScreenAndIsLoudWhenThereIsNothingToRead", "internal/guard", ByProbe},
	{"a window that is gone is reported rather than read as clean", "TestTheLiteralGuardIsLoudWhenTheWindowIsGone", "internal/guard", ByProbe},
	{"the theme is exempt by the list and not by its name", "TestTheThemeIsExemptByTheListAndNotByItsName", "internal/guard", ByProbe},
	{"a spacing step survives the scaling factors a desktop applies", "TestEveryStepSurvivesTheCommonScalingFactors", "internal/gui/theme", ByMutation},
	{"the scale never gives one distance two names", "TestEveryStepOfTheScaleIsUsedOnce", "internal/gui/theme", ByMutation},
	{"a colour of the palette is never translucent, so its measured contrast is about what is shown", "TestEveryColourIsFullyOpaque", "internal/gui/theme", ByMutation},

	// --- the visual system: the ladders, the tokens and the icon loader -----
	//
	// The register above guards the palette measured for the first window.
	// These guard the ladders that DEC-232 added beside it, and the thing they
	// really guard is the pair a token carries: the threshold it answers to and
	// the grounds it stands on. Every one below was proved by putting a
	// measured fault of the accepted drawing back in and watching it fail.
	{"every token reaches its threshold on every ground it is drawn on", "TestEveryTokenReachesItsThresholdOnEveryGroundItStandsOn", "internal/gui/theme", ByMutation},
	{"a ladder rises step by step, which is what makes a step one step", "TestEveryLadderRisesStepByStep", "internal/gui/theme", ByMutation},
	{"a token constant and a token entry answer to each other", "TestEveryTokenNameAndEntryAnswerToEachOther", "internal/gui/theme", ByMutation},
	{"a token is either a ground or is measured against one", "TestATokenIsEitherAGroundOrIsMeasuredAgainstOne", "internal/gui/theme", ByMutation},
	{"no rung of the type ladder is large text, so one threshold answers for all of them", "TestNoRungOfTheTypeLadderIsLargeTextByWCAG", "internal/gui/theme", ByMutation},
	{"an icon carries our colour rather than the black the decoder would leave", "TestTheLoaderLeavesNeitherSourceValueBehind", "internal/gui/paint", ByMutation},
	{"the same icon twice is the same resource, so a stored tree does not move on its own", "TestTheSameIconTwiceIsTheSameResource", "internal/gui/paint", ByMutation},
	{"an icon holds its stroke width on screen rather than scaling it away", "TestTheStrokeRisesAsTheIconGetsSmaller", "internal/gui/paint", ByMutation},
	{"an icon name and an embedded file answer to each other", "TestEveryIconNameAndFileAnswerToEachOther", "internal/gui/paint", ByMutation},
	{"every embedded icon carries what the loader rewrites, so no substitution misses in silence", "TestEveryEmbeddedIconCarriesWhatTheLoaderRewrites", "internal/gui/paint", ByMutation},

	// --- the window never asks the toolkit to leave this machine ------------
	//
	// The import scan answers about our own packages. The toolkit is one
	// import carrying net/http inside it, so that scan cannot see this. The
	// owner settled on 2026-08-12 that the unbreakable rule is about what the
	// product does rather than about what its closure contains, and a promise
	// about behaviour needs a guard about behaviour.
	{"nothing we wrote asks the toolkit to leave this machine", "TestNothingInTheWindowAsksTheToolkitToLeaveThisMachine", "internal/guard", ByMutation},
	{"the scanner sees a call that leaves the machine", "TestTheToolkitGuardSeesACallThatLeavesTheMachine", "internal/guard", ByProbe},
	{"the one file allowed to open a page names only its own addresses, takes none from a caller and is reported when it moves", "TestTheOneFileAllowedToOpenAPageMayOpenOnlyItsOwnPages", "internal/guard", ByProbe},
	{"a window that is gone is reported rather than read as clean", "TestTheToolkitGuardIsLoudWhenTheWindowIsGone", "internal/guard", ByProbe},
	{"every banned call records why it is banned", "TestEveryBannedToolkitCallSaysWhyItIsBanned", "internal/guard", ByProbe},
	{"nothing in the window gives itself a theme of its own, which would cost a set of fonts on every return to a screen", "TestNothingInTheWindowGivesItselfAThemeOfItsOwn", "internal/guard", ByMutation},
	{"the scanner sees every way of naming a theme of its own and nothing else", "TestTheThemeScopeGuardSeesEveryWayToNameOneAndNothingElse", "internal/guard", ByProbe},
	{"a window that is gone is reported rather than read as having no theme of its own", "TestTheThemeScopeGuardIsLoudWhenTheWindowIsGone", "internal/guard", ByProbe},

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
	// The program's icon and the properties of both executables. Each is
	// proven by a fault planted in the test file itself: an older version in
	// the resource, the call turned into a comment, a picture from the other
	// drawing, a resource with one picture changed, an empty drawing.
	{"both executables say who made them and which version they are, and the version is the one this build is", "TestBothExecutablesSayWhoMadeThemAndWhichVersion", "internal/guard", ByProbe},
	{"a resource still carrying another version is told apart", "TestAnExecutableWithAnOlderVersionIsNoticed", "internal/guard", ByProbe},
	{"the window wears the program's icon rather than the toolkit's logo", "TestTheWindowIsHandedTheProgramIcon", "internal/guard", ByProbe},
	{"the icon file both executables carry is the drawing at every size Windows asks for", "TestTheIconFileIsDrawnFromTheDrawings", "internal/gui/appicon", ByProbe},
	{"both executables carry the icon file as it is now", "TestBothExecutablesCarryTheIconFile", "internal/gui/appicon", ByProbe},
	{"both drawings of the icon draw something the rasteriser could read", "TestBothDrawingsDrawASpoonRatherThanNothing", "internal/gui/appicon", ByProbe},
	{"the picture GitHub shows says what this build does and looks the way it is drawn", "TestTheSocialPreviewSaysWhatTheProgramDoes", "internal/gui/socialcard", ByProbe},

	// The README, the other page a stranger reads first. Its faults are planted
	// on small pages in the test file: a changed line, an unrun command, a block
	// of nothing but elisions, a wrong exit code, a defect the manifest lacks,
	// a generator or group left out or made up, and a Go version other than
	// go.mod's.
	{"the commands the README shows run, and print and write what it shows", "TestTheReadmeShowsWhatTheProgramDoes", "internal/cli", ByProbe},

	// The project website reads what the program says about itself from two
	// files of facts, one for each side of the program, so a page cannot show a
	// command, a generator or a name of damage the program no longer has. The
	// canary beside each test plants a program that moved on.
	{"the website's facts about the command line are what the command line says", "TestTheSiteFactsAreWhatTheProgramSays", "internal/cli", ByProbe},
	{"the website names every kind of damage the way the window does", "TestTheWindowFactsAreWhatTheWindowSays", "internal/gui/damageparts", ByProbe},

	// The pictures of the window on the website and in the README are drawn by
	// the program from the data of the README and compared with the window as it
	// draws now, texts everywhere and pixels on windows/amd64. The canary draws
	// the same screen from another seed.
	{"every picture of the window on the website is the window as this build draws it", "TestTheWebsiteShowsTheWindowAsItIs", "internal/gui/screens", ByProbe},
	{"the picture of Quick value starts with the values the README prints, as its caption says", "TestTheQuickPictureStartsWithTheValuesTheReadmePrints", "internal/gui/screens", ByProbe},
}
