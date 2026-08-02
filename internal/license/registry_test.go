package license_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/guard"
	"github.com/donislawdev/qadatagenerator/internal/license"
)

// Every dependency has to pass two questions before it enters: whether its
// license agrees with ours, and whether we are allowed to take a substantial
// part of whatever data it carries. Both are decisions for a person, and
// neither can be checked here.
//
// What can be checked is that no dependency slipped in without anybody writing
// its answers down. This test compares the registry with go.mod, so a module
// added without a line here fails the build instead of shipping unlisted.

var requireLine = regexp.MustCompile(`^\s*([\w.\-]+\.[\w.\-]+/[^\s]+)\s+(v[^\s]+)`)

func directRequirements(t *testing.T) map[string]string {
	t.Helper()
	root, err := guard.ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("cannot read go.mod: %v", err)
	}

	found := map[string]string{}
	inBlock := false
	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "//"):
			continue
		case trimmed == "require (":
			inBlock = true
			continue
		case inBlock && trimmed == ")":
			inBlock = false
			continue
		}
		// A module pulled in through another one is described too. It ends
		// up in the binary just the same, and whoever received the binary is
		// the person the licence of the code inside it binds. Listing one
		// module too many costs a line here, listing one too few is a claim
		// about somebody else's terms that nobody checked.
		trimmed = strings.TrimSuffix(trimmed, "// indirect")
		candidate := trimmed
		if !inBlock {
			if !strings.HasPrefix(trimmed, "require ") {
				continue
			}
			candidate = strings.TrimPrefix(trimmed, "require ")
		}
		if match := requireLine.FindStringSubmatch(candidate); match != nil {
			found[match[1]] = match[2]
		}
	}
	return found
}

func TestRegistryDescribesEveryDependency(t *testing.T) {
	required := directRequirements(t)
	listed := map[string]string{}
	for _, component := range license.ThirdParty {
		listed[component.Module] = component.Version
	}

	for module, wanted := range required {
		got, present := listed[module]
		if !present {
			t.Errorf("go.mod requires %s and the license registry does not describe it", module)
			continue
		}
		if got != wanted {
			t.Errorf("%s is %s in go.mod and %s in the license registry", module, wanted, got)
		}
	}
	for module := range listed {
		if _, present := required[module]; !present {
			t.Errorf("the license registry describes %s, which this build no longer uses", module)
		}
	}
}

func TestEveryListedComponentIsFullyDescribed(t *testing.T) {
	for _, component := range license.ThirdParty {
		if component.License == "" {
			t.Errorf("%s has no license, and the flag exists to state exactly that", component.Module)
		}
		if component.Source == "" {
			t.Errorf("%s has no source, so nobody can read its terms", component.Module)
		}
		if !isADay(component.Checked) {
			t.Errorf("%s does not say when its terms were read, so nobody can tell how old that fact is", component.Module)
		}
	}
}

// A license is a property of a version rather than of a project, and projects
// have moved between permissive and copyleft terms between releases. Without
// the day, the next session cannot tell a fact that was checked from one that
// was remembered, and a remembered one reads most convincingly when it has
// gone stale.
func TestEveryReferenceCollectionSaysWhenItsTermsWereRead(t *testing.T) {
	for _, source := range license.Data {
		if !isADay(source.Checked) {
			t.Errorf("%s does not say when the terms of %s were read", source.Corpus, source.Project)
		}
	}
}

// isADay accepts YYYY-MM-DD and nothing else. The format is fixed here rather
// than taken from the machine, for the same reason every other date in this
// project is.
func isADay(value string) bool {
	if len(value) != 10 || value[4] != '-' || value[7] != '-' {
		return false
	}
	for position, symbol := range value {
		if position == 4 || position == 7 {
			continue
		}
		if symbol < '0' || symbol > '9' {
			return false
		}
	}
	return true
}

func TestProjectLicensesAreNamed(t *testing.T) {
	if license.Code == "" || license.Spec == "" {
		t.Fatal("both licenses have to be named, because the specification is not under the license of the code")
	}
	if license.GeneratedDataNotice == "" {
		t.Fatal("the notice about generated data is the sentence that removes a real barrier to adoption")
	}
}
