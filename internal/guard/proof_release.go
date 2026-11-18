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
}
