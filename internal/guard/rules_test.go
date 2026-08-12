package guard

import (
	"strings"
	"testing"
)

const module = "github.com/donislawdev/qadatagenerator"

// Every check below is proved twice: once against a graph built to break it,
// so that we know the check can fail, and once against the real tree, so that
// we know the tree is clean. A check that has only ever passed tells us
// nothing about either.

func TestNetworkCatchesAnImportThatReachesOutside(t *testing.T) {
	broken := []Package{
		{Path: "internal/adapter/store", Imports: []string{"os", "net/http"}},
	}
	findings := Network(broken)
	if len(findings) != 1 || !strings.Contains(findings[0], "net/http") {
		t.Fatalf("expected one finding about net/http, got %v", findings)
	}
}

func TestNetworkAllowsAddressParsing(t *testing.T) {
	if OpensTheWorld("net/url") {
		t.Fatal("net/url only parses, it opens nothing")
	}
	for _, banned := range []string{"net", "net/http", "net/smtp", "os/exec"} {
		if !OpensTheWorld(banned) {
			t.Fatalf("%s should be banned", banned)
		}
	}
}

func TestLayersCatchesTheDomainReachingUpwards(t *testing.T) {
	broken := []Package{
		{Path: "internal/domain/seed", Imports: []string{module + "/internal/cli"}},
	}
	findings := Layers(broken, module)
	if len(findings) != 1 || !strings.Contains(findings[0], "internal/cli") {
		t.Fatalf("expected the domain to be caught reaching the command line, got %v", findings)
	}
}

// A use case takes what it needs as a port. Reaching an adapter directly is
// the same breach as reaching the surface, and it is the easier one to make by
// accident, because the adapter is right there and already does the job.
func TestLayersCatchesAUseCaseReachingAnAdapter(t *testing.T) {
	broken := []Package{
		{Path: "internal/app/validate", Imports: []string{module + "/internal/adapter/recipeyaml"}},
	}
	findings := Layers(broken, module)
	if len(findings) != 1 || !strings.Contains(findings[0], "internal/adapter") {
		t.Fatalf("expected the use case to be caught reaching an adapter, got %v", findings)
	}
}

func TestLayersAllowsAPackageToImportItsOwnSubpackage(t *testing.T) {
	fine := []Package{
		{Path: "internal/domain", Imports: []string{module + "/internal/domain/seed"}},
	}
	if findings := Layers(fine, module); len(findings) != 0 {
		t.Fatalf("a package may reach inside itself, got %v", findings)
	}
}

func TestLayersCatchesTheBasePackageDependingOnUs(t *testing.T) {
	broken := []Package{
		{Path: "internal/fault", Imports: []string{module + "/internal/version"}},
	}
	if findings := Layers(broken, module); len(findings) != 1 {
		t.Fatalf("expected one finding, got %v", findings)
	}
}

func TestReachFindsAnIndirectPath(t *testing.T) {
	broken := []Package{
		{Path: "cmd/qadg", Imports: []string{module + "/internal/cli"}},
		{Path: "internal/cli", Imports: []string{module + "/internal/app"}},
		{Path: "internal/app", Imports: []string{module + "/internal/gui"}},
	}
	chain := Reach(broken, module, "cmd/qadg", "internal/gui")
	if len(chain) != 4 {
		t.Fatalf("expected the whole chain, got %v", chain)
	}
	if chain[0] != "cmd/qadg" || chain[len(chain)-1] != "internal/gui" {
		t.Fatalf("chain should run from one end to the other, got %v", chain)
	}
}

func TestTestOnlyIgnoresTestFiles(t *testing.T) {
	fine := []Package{
		{Path: "internal/cli", TestImports: []string{module + "/internal/guard"}},
	}
	if findings := TestOnly(fine, module, "internal/guard"); len(findings) != 0 {
		t.Fatalf("a test may import the guards, got %v", findings)
	}
	broken := []Package{
		{Path: "internal/cli", Imports: []string{module + "/internal/guard"}},
	}
	if findings := TestOnly(broken, module, "internal/guard"); len(findings) != 1 {
		t.Fatalf("product code may not import the guards, got %v", findings)
	}
}

// The checks above prove the rules work. The ones below apply them to the
// repository as it stands.

func scanRepository(t *testing.T) []Package {
	t.Helper()
	root, err := ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}
	pkgs, err := Scan(root)
	if err != nil {
		t.Fatalf("cannot read the package graph: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatal("the scan found no packages, so it is checking nothing")
	}
	return pkgs
}

func TestRepositoryOpensNothing(t *testing.T) {
	if findings := Network(scanRepository(t)); len(findings) != 0 {
		t.Fatalf("network capable imports found:\n%s", strings.Join(findings, "\n"))
	}
}

func TestRepositoryRespectsTheLayers(t *testing.T) {
	if findings := Layers(scanRepository(t), module); len(findings) != 0 {
		t.Fatalf("dependency rule broken:\n%s", strings.Join(findings, "\n"))
	}
}

// The command line binary must not reach the window, not even through three
// other packages. That import would pull in the graphical toolkit and with it
// the C toolchain, and the binary that has to cross compile from one machine
// would stop building.
func TestCommandLineBinaryNeverReachesTheWindow(t *testing.T) {
	pkgs := scanRepository(t)
	// Without this the rule would pass by looking for a package that is no
	// longer called that, which is the quietest way for a guard to retire.
	if !Has(pkgs, "cmd/qadg") {
		t.Fatal("cmd/qadg is not in the graph, so this rule would check nothing")
	}
	// The other end needs the same treatment and did not have it until the
	// window existed. While nothing lived under internal/gui this test passed
	// by searching for a destination that was not there, which reads exactly
	// like an isolation that holds.
	if !anyPackageUnder(pkgs, ViewRoot) {
		t.Fatalf("no package lives under %s, so this rule would check nothing", ViewRoot)
	}
	chain := Reach(pkgs, module, "cmd/qadg", "internal/gui")
	if chain != nil {
		t.Fatalf("cmd/qadg reaches internal/gui through %s", strings.Join(chain, " -> "))
	}
}

// anyPackageUnder says whether the graph holds a package at or below prefix.
//
// Has answers about one exact path, and the window is a tree of packages
// rather than a single one, so the two questions are not the same question.
func anyPackageUnder(pkgs []Package, prefix string) bool {
	for _, pkg := range pkgs {
		if within(pkg.Path, prefix) {
			return true
		}
	}
	return false
}

func TestTheWindowIsPresentEnoughForTheIsolationToMeanSomething(t *testing.T) {
	pkgs := []Package{{Path: "internal/gui/theme"}}
	if !anyPackageUnder(pkgs, ViewRoot) {
		t.Error("a package below the window counts as the window being there")
	}
	if anyPackageUnder([]Package{{Path: "internal/cli"}}, ViewRoot) {
		t.Error("a package outside the window does not count as the window being there")
	}
}

func TestHasSeesTheDifferenceBetweenAbsentAndClean(t *testing.T) {
	pkgs := []Package{{Path: "cmd/qadg"}}
	if !Has(pkgs, "cmd/qadg") {
		t.Error("the package is there")
	}
	if Has(pkgs, "cmd/qadg-gui") {
		t.Error("the package is not there")
	}
}

func TestGuardsAreActuallyLoadedFromDisk(t *testing.T) {
	pkgs := scanRepository(t)
	for _, expected := range []string{"internal/cli", "internal/fault", "internal/guard", "internal/oracle"} {
		if !Has(pkgs, expected) {
			t.Errorf("the scan missed %s, so every rule about it is passing by default", expected)
		}
	}
}

func TestGuardsShipWithNoBinary(t *testing.T) {
	if findings := TestOnly(scanRepository(t), module, "internal/guard"); len(findings) != 0 {
		t.Fatalf("the guards are linked into the product:\n%s", strings.Join(findings, "\n"))
	}
}

// The independent validators are a second telling of rules the product already
// implements. Linking them would put both tellings in the binary and invite
// somebody to call whichever is nearer, at which point the product would be
// checking itself against itself and the whole point would be gone.
func TestTheIndependentValidatorsShipWithNoBinary(t *testing.T) {
	if findings := TestOnly(scanRepository(t), module, "internal/oracle"); len(findings) != 0 {
		t.Fatalf("the independent validators are linked into the product:\n%s", strings.Join(findings, "\n"))
	}
}
