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

// embeddedNotices pairs each notices file in the root of the module, which is
// the one source, with the text a binary compiles in, and the heading that
// text opens with. The copies beside this package exist only because an embed
// pattern cannot walk up out of its own package (DEC-134).
var embeddedNotices = []struct {
	file, embedded, heading string
}{
	{"THIRD-PARTY-NOTICES.md", license.CommandLineNotices, "# Third party notices\n"},
	{"THIRD-PARTY-NOTICES-GUI.md", license.WindowNotices, "# Third party notices of the window\n"},
}

// TestTheEmbeddedNoticesAreTheOnesInTheRoot is the whole reason the copies are
// allowed to exist. Two files holding the same thing drift silently: a binary
// would hand out yesterday's notices while the archive carried today's, and
// the guard in internal/guard reads only the root ones. Comparing the bytes
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
		t.Fatalf("cannot find the module root, so the copies cannot be checked: %v", err)
	}
	for _, pair := range embeddedNotices {
		published, err := os.ReadFile(filepath.Join(root, pair.file))
		if err != nil {
			t.Errorf("cannot read %s, so its copy is unchecked rather than correct: %v", pair.file, err)
			continue
		}
		if why := noticesDiffer(published, []byte(pair.embedded)); why != "" {
			t.Errorf("the notices compiled in are not %s any more: %s. "+
				"The root file is the one source. Copy it over internal/license/%s and run the tests again.",
				pair.file, why, pair.file)
		}
		// An empty or truncated embed would still be compared, so the copy also
		// has to open the way the file does.
		if !strings.HasPrefix(pair.embedded, pair.heading) {
			t.Errorf("the embedded copy of %s does not open with its heading: %.60q", pair.file, pair.embedded)
		}
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
