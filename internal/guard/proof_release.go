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
	// exception no walk reached. Its checks are proved by a canary that plants
	// one gap per question, and on 2026-11-18 a version and a notice taken out
	// of the real file were caught once by hand.
	{"every module, licence file and copyright statement the command line links stands in its notices", "TestTheNoticesCoverWhatTheBinaryLinks", "internal/guard", ByProbe},
	{"the notices check sees a missing module, a wrong version, a missing text and a section for nothing", "TestTheNoticesGuardCanFail", "internal/guard", ByProbe},
	{"an #include is followed to where the compiler finds the file, and no further", "TestAnIncludeIsFollowedTheWayTheCompilerLooks", "internal/guard", ByProbe},
	{"a copyright statement in a source comment is read off it, and a sentence about copyright is not", "TestACopyrightStatementIsReadOffAComment", "internal/guard", ByProbe},
}
