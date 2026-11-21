package guard

// The part of the register about what a release carries beside the binaries:
// the notices of other people's work today, and the pieces of the release
// pipeline that B-419 adds after them.
//
// It is a part of its own from its first entry rather than a cut made later.
// The release pipeline is one subject with several pieces still to come, and
// proof.go stands at the ceiling on the size of a file. Every tool that reads
// the register reads every proof*.go of this package.
var releaseGuards = []Guard{
	// --- the notices of other people's work (DEC-327) -----------------------
	//
	// Probes, not mutations. The guard reads the real build through go list
	// and is loud when the gathering reaches nothing: it stops when the walk
	// never arrives at the licence of the Go distribution, and it fails on an
	// exception no walk reached or a landmark of the window's roads it lost.
	// Its checks are proved by a canary that plants one gap per question. On
	// 2026-11-18 a version and a notice taken out of the command line's file,
	// and four faults put into the window's, were each caught once by hand.
	{"every module, licence file and copyright statement each binary links stands in its notices", "TestTheNoticesCoverWhatTheBinaryLinks", "internal/guard", ByProbe},
	{"the notices check sees a missing module, a wrong version, a missing text and a section for nothing", "TestTheNoticesGuardCanFail", "internal/guard", ByProbe},
	{"an #include is followed to where the compiler finds the file, and no further", "TestAnIncludeIsFollowedTheWayTheCompilerLooks", "internal/guard", ByProbe},
	{"a copyright statement in a source comment is read off it, and a sentence about copyright is not", "TestACopyrightStatementIsReadOffAComment", "internal/guard", ByProbe},

	// The code the C compiler links into the window of its own, which no walk
	// of go list reaches. The pin is loud when it is empty, because it then
	// says nothing about each of the three targets, and the walk of .github
	// stops when it never arrives at ci.yml. On 2026-11-18 a version of
	// MinGW-w64 changed in the pin, the reader's call taken out of ci.yml and
	// the compiler's sections taken out of the window's row were each caught
	// once by hand. The reader itself is a shell script on the runner, and
	// tools/noty/atrapa-mapy.ps1 runs it on three real maps and their spoiled
	// copies.
	{"the compiler's own code in the window is the code its notices answer for, and a build reads the linker's map", "TestTheCompilersOwnCodeInTheWindowIsTheCodeItsNoticesAnswerFor", "internal/guard", ByProbe},
	{"the toolchain check sees a library nobody read, a gap in the pin, a section at another version and a build that skips the reader", "TestTheToolchainGuardCanFail", "internal/guard", ByProbe},

	// The bill of materials (DEC-328). Each test below runs the command's
	// checks on faults planted in fixtures. The tests that read the real
	// notices and the real .github/opengl run the same parser, and on
	// 2026-11-18 a section back at "Public domain." in the window's copy, NCSA
	// taken out of the renderer's sentence and the icons taken out of the
	// table of works were each caught once by hand. The document they render
	// is held to two golden files that pyspdxtools from spdx-tools 0.8.5
	// accepted that day, and the same validator rejected four documents
	// broken on purpose.
	{"the bill of materials refuses a build that links a module, a version or a toolchain the notices of its program do not name", "TestEveryDisagreementIsReported", "cmd/sbom", ByProbe},
	{"the bill of materials refuses notices it cannot read whole: an identifier nobody read on the SPDX lists, a text without its words, a work in the wrong place", "TestNoticesThatCannotDescribeABuildAreRefused", "cmd/sbom", ByProbe},
	{"the software renderer is described with the build of the window it travels beside, and with no other", "TestTheRendererGoesWithTheBuildItTravelsBeside", "cmd/sbom", ByProbe},
	// The three that read the real tree first ask their check about the tree
	// with one fault planted in it, and then about the tree as it is.
	{"the notices this tree ships can describe a build, with every text without an SPDX identifier and every work where the table says", "TestTheNoticesInThisTreeDescribeABuild", "cmd/sbom", ByProbe},
	{"the software renderer's notices this tree keeps are read whole, NCSA among their licences", "TestTheRenderersNoticesInThisTreeAreRead", "cmd/sbom", ByProbe},
	{"builds that link what the shipped notices name are described, and the same builds with the toolkit at another version are refused", "TestTheShippedNoticesDescribeABuildThatLinksWhatTheyName", "cmd/sbom", ByProbe},

	// The owner's decision of 2026-11-18 that --license ends with those
	// notices. Both tests ask their check for a planted fault first, and on
	// the same day a report without the notices and a copy with one changed
	// byte were each caught once by hand.
	{"the notices compiled into each binary are its notices file in the root, byte for byte", "TestTheEmbeddedNoticesAreTheOnesInTheRoot", "internal/license", ByProbe},
	{"--license ends with those notices, whole, and its machine readable form carries them", "TestLicenseEndsWithTheThirdPartyNotices", "internal/cli", ByProbe},

	// The same for the window, whose About screen is its --license. Both
	// tests ask their check about a planted fault first, and on 2026-11-18 a
	// report built without the notices failed both once by hand.
	{"the About screen saves the window's notices, whole, with a section for every embedded asset", "TestTheScreenWritesTheNoticesOfWhatItEmbeds", "internal/gui/about", ByProbe},
	{"the report the About screen saves carries the window's notices", "TestTheScreenOffersTheReportInTheFormAMachineReads", "internal/gui/about", ByProbe},

	// The release workflow and its scripts (DEC-329). Every test below plants
	// one fault in a copy of the real files first, and stops when the text it
	// plants in is no longer there, so a check that reads nothing or a file
	// that moved away from what it reads fails rather than passes. On
	// 2026-11-18 the call of the map's reader taken out of build_window.sh
	// was caught once by hand in the real tree. What the scripts DO is asked
	// of them by running them, on the three systems, and on the runners by
	// the workflow itself on every push that changes them.
	{"the release builds and packs the targets the notices are checked for, and the window job builds the window's", "TestTheReleaseBuildsTheTargetsTheNoticesAreCheckedFor", "internal/guard", ByProbe},
	{"the windows the window job builds reach the archives, and the renderer's notices in them are the ones the bill of materials reads", "TestTheWindowsAndTheirRendererReachTheArchives", "internal/guard", ByProbe},
	{"every workflow that can write names each action by a commit with its release beside it", "TestEveryWorkflowThatCanWritePinsItsActionsToACommit", "internal/guard", ByProbe},
	{"a script a workflow or a release script runs is handed to its interpreter and is in the tree", "TestAScriptAWorkflowRunsIsHandedToItsInterpreter", "internal/guard", ByProbe},
	{"the release opens an empty draft behind the tag and its three checks, with a note that says how to check a download", "TestTheReleaseWorkflowOpensADraftAndPublishesNothing", "internal/guard", ByProbe},
	{"the bill of materials is read by SPDX's own validator, installed from files pinned by their bytes", "TestTheBillOfMaterialsIsReadBySPDXsOwnValidator", "internal/guard", ByProbe},
	{"the bundle on macOS names the author the Windows resources name, and its icon is in the tree", "TestTheMacBundleSaysWhatTheWindowsResourcesSay", "internal/guard", ByProbe},

	// The signing of a release (DEC-330). The same shape: each test plants
	// one fault in a copy of the real files first and stops when the text it
	// plants in is gone. What the signing script DOES is asked of it by
	// running it: on 2026-11-18 its checks of a signed file ran on the two
	// libraries of the software renderer as the same card signed them for
	// another project's release, and refused six faults planted in them, and
	// the Mac's script refused its own wrong inputs on the Mac.
	{"each certificate a release is signed with is pinned once, by a digest with a date, and the signing script reads that copy", "TestThePinnedCertificatesAreDigestsWithADate", "internal/guard", ByProbe},
	{"the signing script signs the archives the release builds, reads the renderer's files from their pin and names this repository", "TestTheSigningScriptSignsTheArchivesTheReleaseBuilds", "internal/guard", ByProbe},
	{"the note, the attestation workflow and the bill of materials name one SPDX version", "TestEveryFileNamesThePredicateTheDocumentIs", "internal/guard", ByProbe},
	{"nothing after the build publishes a release, and both steps that add files ask whether it is still a draft", "TestTheSigningLeavesTheReleaseADraft", "internal/guard", ByProbe},

	// The Windows installer (DEC-331). The same shape again. What the
	// installer DOES is asked of Windows by check_msi.ps1, which ran on
	// Windows Server 2025 on 2026-11-21 before these were written, and failed
	// there on a package built without the line the shape guard holds first.
	{"the installer template is the shape that was measured: for every account, upgrades after the new files, nothing running closed, one PATH entry and one shortcut", "TestTheInstallerIsTheShapeThatWasMeasured", "internal/guard", ByProbe},
	{"the installer's upgrade code, file name, product name, publisher and WiX are one value each wherever they are written", "TestTheInstallerIsNamedTheSameEverywhere", "internal/guard", ByProbe},
	{"every place that builds, installs or signs the installer asks the same question of a release candidate", "TestEveryPlaceAsksTheSameQuestionOfACandidate", "internal/guard", ByProbe},
	{"the signing builds the installer after the card and before the checksums, and the draft waits for the job that installs it", "TestTheSigningBuildsTheInstallerAfterTheCardAndBeforeTheSums", "internal/guard", ByProbe},
}
