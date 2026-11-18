package license_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/guard"
	"github.com/donislawdev/qadatagenerator/internal/license"
)

// noticesFile is the notices of the command line in the root of the module,
// which is the one source. The copy beside this package exists only because an
// embed pattern cannot walk up out of its own package (DEC-134).
const noticesFile = "THIRD-PARTY-NOTICES.md"

// TestTheEmbeddedNoticesAreTheOnesInTheRoot is the whole reason the copy is
// allowed to exist. Two files holding the same thing drift silently: the
// binary would print yesterday's notices while the archive carried today's,
// and the guard in internal/guard reads only the root one. Comparing the bytes
// turns that into a red test in the same minute it happens.
func TestTheEmbeddedNoticesAreTheOnesInTheRoot(t *testing.T) {
	// The comparison is asked for both of its answers first, so that a
	// comparison that always agrees cannot pass for the check.
	if why := noticesDiffer([]byte("same\n"), []byte("same\n")); why != "" {
		t.Fatalf("two equal files were reported as different: %s", why)
	}
	if noticesDiffer([]byte("a licence\n"), []byte("a licence, edited\n")) == "" {
		t.Fatal("two different files were reported as equal")
	}

	root, err := guard.ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root, so the copy cannot be checked: %v", err)
	}
	published, err := os.ReadFile(filepath.Join(root, noticesFile))
	if err != nil {
		t.Fatalf("cannot read %s, so the copy is unchecked rather than correct: %v", noticesFile, err)
	}
	if why := noticesDiffer(published, []byte(license.CommandLineNotices)); why != "" {
		t.Errorf("the notices compiled into this build are not %s any more: %s. "+
			"The root file is the one source. Copy it over internal/license/%s and run the tests again.",
			noticesFile, why, noticesFile)
	}
	// An empty or truncated embed would still be compared, so the copy also has
	// to open the way the file does.
	if !strings.HasPrefix(license.CommandLineNotices, "# Third party notices\n") {
		t.Errorf("the embedded notices do not open with their heading: %.60q", license.CommandLineNotices)
	}
}

// noticesDiffer says where two files stop agreeing, or nothing when they are
// the same bytes. Exact rather than normalised, because line endings are held
// to LF across the tree and a difference in them is a real difference.
func noticesDiffer(published, embedded []byte) string {
	if bytes.Equal(published, embedded) {
		return ""
	}
	at := 0
	for at < len(published) && at < len(embedded) && published[at] == embedded[at] {
		at++
	}
	return fmt.Sprintf("%d bytes in the root, %d embedded, first difference at byte %d", len(published), len(embedded), at)
}
