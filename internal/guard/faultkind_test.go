package guard_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/guard"
)

// Every refusal either side of the boundary between exit code 3 and exit code 5
// says why it is on the side it is on.
//
// 🔴 The boundary was walked once by hand on 2026-08-17 and written into
// docs/09, and nothing held it there. The bridge in the audit script compares
// the numbers and the names of the codes between the document and the table in
// the code, so it answers whether a code exists and never which ending gets
// which one. That is how eight endings of one verb came to return the code
// meaning a recipe cannot be satisfied while meaning that a file could not be
// read, with every guard green.
func TestEveryRefusalEitherSideOfTheBoundarySaysWhy(t *testing.T) {
	root, err := guard.ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}
	sites, err := guard.FaultSites(root)
	if err != nil {
		t.Fatalf("the refusals could not be read: %v", err)
	}
	if len(sites) == 0 {
		t.Fatal("no refusal of either kind was found, so this guard would check nothing")
	}

	for _, finding := range guard.FaultBoundary(sites) {
		t.Error(finding)
	}

	config, data := 0, 0
	for _, site := range sites {
		switch site.Kind {
		case "Config":
			config++
		case "Data":
			data++
		}
	}
	t.Logf("%d refusals say a document is not valid and %d say a valid one cannot be satisfied",
		config, data)
}

// A reason is a sentence rather than a restatement of the kind.
//
// Without this the register could be filled with the word Config next to every
// entry naming Config, which would pass the guard above while saying nothing.
// The register exists to carry the ARGUMENT, because that is the part a reader
// cannot recover from the code.
func TestEveryReasonInTheRegisterSaysSomething(t *testing.T) {
	for key, why := range guard.RefusalKinds {
		if len(strings.Fields(why)) < 6 {
			t.Errorf("the register answers for %q with %q, which is too short to be an argument",
				key, why)
		}
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(why)), "config") ||
			strings.HasPrefix(strings.ToLower(strings.TrimSpace(why)), "data") {
			t.Errorf("the register answers for %q by naming the kind again rather than saying why", key)
		}
	}
}
