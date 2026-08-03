package guard

import (
	"strings"
	"testing"
)

// # The guard covered the standard library and nothing else
//
// It tested whether an import began with net/, which answers for the standard
// library alone. golang.org/x/net/http2 does not begin with net/ and passed, so
// the check covered the source of network code this project is least likely to
// add by accident and missed the one it is most likely to add on purpose.
//
// Matching a whole path segment catches both spellings.
func TestTheNetworkGuardSeesNetworkPackagesOutsideTheStandardLibrary(t *testing.T) {
	banned := []string{
		"net",
		"net/http",
		"net/smtp",
		"os/exec",
		"golang.org/x/net/http2",
		"golang.org/x/net/proxy",
		"github.com/somebody/client/net/dial",
		"syscall",
		"plugin",
	}
	for _, importPath := range banned {
		if !OpensTheWorld(importPath) {
			t.Errorf("%s can leave this machine and was allowed", importPath)
		}
	}
}

// A name that merely starts the same way is not a network package, or the guard
// fires on prose and gets switched off by whoever hits it first.
func TestTheNetworkGuardDoesNotFireOnNamesThatMerelyLookAlike(t *testing.T) {
	allowed := []string{
		"net/url",
		"github.com/somebody/netlink",
		"github.com/somebody/networkd",
		"internal/adapter/provider/corpus",
		"encoding/json",
		"os",
	}
	for _, importPath := range allowed {
		if OpensTheWorld(importPath) {
			t.Errorf("%s opens nothing and was refused", importPath)
		}
	}
}

// The scanner has to report the package outside the standard library as well,
// not only answer about it.
func TestNetworkReportsADependencyThatReachesOutside(t *testing.T) {
	broken := []Package{
		{Path: "internal/adapter/store", Imports: []string{"os", "golang.org/x/net/http2"}},
	}
	findings := Network(broken)
	if len(findings) != 1 || !strings.Contains(findings[0], "golang.org/x/net/http2") {
		t.Fatalf("expected one finding about the dependency, got %v", findings)
	}
}

// # What this guard cannot see
//
// It reads the imports written in this repository. A dependency opening a
// socket inside itself is invisible to it, and the promise is about the whole
// binary rather than about our half of it. The other half is checked in CI,
// where go list can walk the build graph.
//
// This test exists so that the limit is stated somewhere that runs, rather than
// only in a comment. It asserts the shape of the gap: a package we do not
// import cannot be reported, however obviously it reaches outside.
func TestTheNetworkGuardOnlySeesImportsWrittenHere(t *testing.T) {
	// A dependency that imports net inside itself. Nothing in our tree names
	// it, so the scan has nothing to walk to.
	ours := []Package{
		{Path: "internal/adapter/recipeyaml", Imports: []string{"github.com/goccy/go-yaml"}},
	}
	if findings := Network(ours); len(findings) != 0 {
		t.Fatalf("expected the scan to be silent about what it cannot see, got %v", findings)
	}
}
