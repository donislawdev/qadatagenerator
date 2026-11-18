package license

import _ "embed"

// CommandLineNotices is the text of THIRD-PARTY-NOTICES.md: the licences and copyright
// notices of the work by other people that the command line links and carries
// (DEC-327). --license prints it at the end of its report, and the machine
// readable form carries it as third_party_notices.
//
// # In the binary and not only beside it
//
// MIT and BSD ask for their notice in every copy, a binary copy included, and
// a binary travels on its own. Copied into a container image or onto a build
// agent, it leaves the archive and the file beside it behind.
//
// # Why this embeds a copy of the file in the root
//
// go:embed cannot reach out of its own package, and a package in the root of
// the module would be public API, which DEC-134 rejected for the catalog for
// that reason. So the file is embedded from a copy beside this one, and
// TestTheEmbeddedNoticesAreTheOnesInTheRoot holds the copy to the root file
// byte for byte. tools/noty/rozwin.ps1 writes both.
//
// Only the command line names this variable, and Current leaves it out of the
// report, because the window links this package for the rest of the register
// and carries other code than the one these notices describe.
//
//go:embed THIRD-PARTY-NOTICES.md
var CommandLineNotices string
