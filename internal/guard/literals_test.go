package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The window holds no value of its own.
//
// # Why this passes by checking nothing today, said out loud
//
// The only package under the window is the theme, and the theme is the one
// place allowed to hold values, so this walk finds no file to read. That is
// the honest state on the day the guard was written and it is why the probe
// below exists. The day the first screen lands, this stops being empty on its
// own, without anybody remembering to switch it on.
func TestTheWindowTakesEveryValueFromTheTheme(t *testing.T) {
	root, err := ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}
	findings, err := Literals(root, LiteralsAllowed)
	if err != nil {
		t.Fatalf("cannot read the window: %v", err)
	}
	if len(findings) > 0 {
		t.Errorf("%d value(s) in the window should have come from the theme:\n%s",
			len(findings), strings.Join(findings, "\n"))
	}
}

// The guard has to be able to fail, has to leave the theme alone, and has to
// stay quiet about prose. All three are here because each is a way this guard
// could be worthless without anybody noticing.
func TestTheLiteralGuardFiresOnTheViewAndNotOnTheTheme(t *testing.T) {
	root := t.TempDir()
	written := func(where, name, body string) {
		t.Helper()
		dir := filepath.Join(root, filepath.FromSlash(where))
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	written("internal/gui/screen", "colour.go", "package screen\n\nimport \"image/color\"\n\n"+
		"func c() color.NRGBA { return color.NRGBA{R: 0, G: 0, B: 0, A: 1} }\n")
	written("internal/gui/screen", "text.go", "package screen\n\nfunc s() string { return \"#1B1B1B\" }\n")
	written("internal/gui/screen", "gap.go", "package screen\n\nfunc g() int { return 12 }\n")
	// Nought and one are how a program says empty and single, so they are the
	// two numbers a view may still write.
	written("internal/gui/screen", "clean.go", "package screen\n\n"+
		"func k(xs []string) string {\n\tif len(xs) == 0 {\n\t\treturn \"\"\n\t}\n\treturn xs[1]\n}\n")
	// Prose is not a finding. A guard that fires on a comment gets switched
	// off, and a switched off guard is worse than an absent one.
	written("internal/gui/screen", "prose.go", "package screen\n\n"+
		"// The panel is #242424 and sits 12 away from the edge. See the theme.\nfunc p() {}\n")
	// The theme is the one place a value may live, so none of this is a
	// finding there.
	written("internal/gui/theme", "values.go", "package theme\n\nimport \"image/color\"\n\n"+
		"var Panel = color.NRGBA{R: 36, G: 36, B: 36, A: 255}\n\nconst Gap = 12\n")
	// A test may write a colour, because checking that a control shows the
	// right one needs the value written down to compare against.
	written("internal/gui/screen", "screen_test.go", "package screen\n\nfunc t() string { return \"#FFB900\" }\n")

	findings, err := Literals(root, LiteralsAllowed)
	if err != nil {
		t.Fatalf("cannot read the tree: %v", err)
	}
	if len(findings) != 3 {
		t.Fatalf("expected three findings, got %d:\n%s", len(findings), strings.Join(findings, "\n"))
	}
	for _, want := range []string{"screen/colour.go:", "screen/gap.go:", "screen/text.go:"} {
		if !containsPrefix(findings, "internal/gui/"+want) {
			t.Errorf("no finding names %s:\n%s", want, strings.Join(findings, "\n"))
		}
	}
}

// A guard that answers quietly when its subject is gone is the failure this
// project has paid for before, so the missing case is loud on purpose.
func TestTheLiteralGuardIsLoudWhenTheWindowIsGone(t *testing.T) {
	root := t.TempDir()
	if _, err := Literals(root, LiteralsAllowed); err == nil {
		t.Fatal("a missing window reported no findings, which reads exactly like a clean one")
	}
}

// The list of allowed packages has to be what exempts the theme. Passing an
// empty list has to make the theme a finding like anything else, otherwise the
// exemption is coming from somewhere nobody can see.
func TestTheThemeIsExemptByTheListAndNotByItsName(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, filepath.FromSlash("internal/gui/theme"))
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	body := "package theme\n\nconst Gap = 12\n"
	if err := os.WriteFile(filepath.Join(dir, "values.go"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	findings, err := Literals(root, map[string]bool{})
	if err != nil {
		t.Fatalf("cannot read the tree: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("the theme is exempt by something other than the list, got %d findings:\n%s",
			len(findings), strings.Join(findings, "\n"))
	}
}

func containsPrefix(findings []string, prefix string) bool {
	for _, one := range findings {
		if strings.HasPrefix(one, prefix) {
			return true
		}
	}
	return false
}
