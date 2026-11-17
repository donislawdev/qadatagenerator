package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The rule this guard exists for, applied to the repository as it stands: the
// one process the product starts is the window starting itself again (B-414),
// and nothing else starts anything.
func TestTheProductStartsOnlyWhatSpawnAllowedNames(t *testing.T) {
	findings, err := SpawnUse(rootOrFail(t), SpawnAllowed)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	for _, finding := range findings {
		t.Error(finding + "\n        guard.SpawnAllowed says which file may start a process and what it may start")
	}
}

// With no allowance at all, the file that does start a process has to be
// reported, which proves the walk reaches the product rather than nothing.
func TestTheSpawnScanReachesTheProduct(t *testing.T) {
	findings, err := SpawnUse(rootOrFail(t), nil)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if !strings.Contains(strings.Join(findings, "\n"), "cmd/qadg-gui/again.go imports os/exec") {
		t.Fatalf("scanning with no allowance did not report the window's second start, so the walk "+
			"is not reaching it: %q", findings)
	}
}

// plantedStart is an allowed file that does exactly what its allowance says.
// Every case below changes one thing in it.
const plantedStart = `package sample

import (
	"os"
	"os/exec"
)

func start(args []string) (int, error) {
	self, err := os.Executable()
	if err != nil {
		return 0, err
	}
	var ended *exec.ExitError
	_ = ended
	return 0, exec.Command(self, args...).Run()
}
`

// The canary: one start of what os.Executable answered passes, and every other
// way to start a process is named.
func TestTheSpawnScanNamesEveryOtherStart(t *testing.T) {
	allowance := Spawn{Call: "Command", ProgramFrom: "os.Executable", Also: []string{"ExitError"}}
	cases := []struct {
		name, source, other, want string
	}{
		{"the allowed start", plantedStart, "", ""},
		{"another program", strings.Replace(plantedStart, "Command(self,", `Command("cmd",`, 1), "", "not what os.Executable answered"},
		{"the program bound twice", strings.Replace(plantedStart, "var ended", "self = \"cmd\"\n\tvar ended", 1), "",
			"not what os.Executable answered"},
		{"the program a parameter", strings.Replace(strings.Replace(plantedStart, "self, err := os.Executable()", "err := error(nil)", 1),
			"start(args []string)", "start(self string, args []string)", 1), "", "not what os.Executable answered"},
		{"two starts", strings.Replace(plantedStart, "_ = ended", "_ = exec.Command(self)", 1), "", "2 times"},
		{"no start", strings.Replace(plantedStart, "exec.Command(self, args...).Run()", "nil", 1), "", "0 times"},
		{"a start with a context", strings.Replace(plantedStart, "_ = ended", "_ = exec.CommandContext", 1), "", "uses exec.CommandContext"},
		{"a search of the path", strings.Replace(plantedStart, "_ = ended", "_ = exec.LookPath", 1), "", "uses exec.LookPath"},
		{"a literal command", strings.Replace(plantedStart, "_ = ended", "_ = exec.Cmd{Path: self}", 1), "", "uses exec.Cmd"},
		{"an alias that starts the same", strings.ReplaceAll(strings.Replace(plantedStart, `"os/exec"`, `run "os/exec"`, 1), "exec.", "run."), "", ""},
		{"an alias that searches the path", strings.Replace(strings.ReplaceAll(strings.Replace(plantedStart, `"os/exec"`, `run "os/exec"`, 1), "exec.", "run."),
			"_ = ended", "_ = run.LookPath", 1), "", "uses exec.LookPath"},
		{"a dot import", strings.Replace(plantedStart, `"os/exec"`, `. "os/exec"`, 1), "", "with a dot"},
		{"a file nobody allowed", plantedStart, plantedStart, "only the files in guard.SpawnAllowed may"},
		{"a process started through os", plantedStart, "package sample\n\nimport \"os\"\n\nvar _, _ = os.StartProcess(\"x\", nil, nil)\n",
			"calls os.StartProcess"},
	}
	for _, each := range cases {
		dir := sampleDir(t)
		writeGo(t, filepath.Join(dir, "sample", "again.go"), each.source)
		if each.other != "" {
			writeGo(t, filepath.Join(dir, "sample", "other.go"), each.other)
		}
		findings, err := SpawnUse(dir, map[string]Spawn{"sample/again.go": allowance})
		if err != nil {
			t.Fatalf("%s: scan failed: %v", each.name, err)
		}
		said := strings.Join(findings, "\n")
		switch {
		case each.want == "" && said != "":
			t.Errorf("%s was refused: %s", each.name, said)
		case each.want != "" && !strings.Contains(said, each.want):
			t.Errorf("%s was not named as %q, the scan said: %q", each.name, each.want, said)
		}
	}
}

// An allowance for a file that starts nothing is an open door nobody looks at.
func TestASpawnAllowanceForAFileThatStartsNothingIsStale(t *testing.T) {
	dir := sampleDir(t)
	writeGo(t, filepath.Join(dir, "sample", "again.go"), "package sample\n")
	findings, err := SpawnUse(dir, map[string]Spawn{"sample/again.go": {Call: "Command", ProgramFrom: "os.Executable"}})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if !strings.Contains(strings.Join(findings, "\n"), "stale") {
		t.Fatalf("an allowance for a file with no os/exec was not called stale: %q", findings)
	}
}

// sampleDir is a module root of its own with a package directory named sample.
func sampleDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "sample"), 0o700); err != nil {
		t.Fatal(err)
	}
	return dir
}

// The exception is one package wide in the network rule and no wider.
func TestTheNetworkRuleStillRefusesExecOutsideTheAllowedPackage(t *testing.T) {
	allowed := Package{Path: "cmd/qadg-gui", Imports: []string{"os/exec"}}
	other := Package{Path: "internal/adapter/store", Imports: []string{"os/exec"}}
	findings := Network([]Package{allowed, other})
	if len(findings) != 1 || !strings.Contains(findings[0], "internal/adapter/store") {
		t.Fatalf("expected the package with no allowance to be the only finding, got %v", findings)
	}
}
