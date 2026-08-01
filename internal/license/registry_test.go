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
		// A module the build pulls in only through another one is not ours
		// to describe, so the indirect marker takes it out.
		if strings.Contains(trimmed, "// indirect") {
			continue
		}
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
	}
}

func TestProjectLicensesAreNamed(t *testing.T) {
	if license.Code == "" || license.Spec == "" {
		t.Fatal("both licenses have to be named, because the specification is not under the license of the code")
	}
	if license.GeneratedDataNotice == "" {
		t.Fatal("the notice about generated data is the sentence that removes a real barrier to adoption")
	}
}
