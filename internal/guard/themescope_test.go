package guard

import (
	"strings"
	"testing"
)

// Nothing in the window gives itself a theme of its own, which is what kept
// the window's memory growing every time somebody came back to a screen
// (B-370). ThemeScopeNames says why, with the measurement.
func TestNothingInTheWindowGivesItselfAThemeOfItsOwn(t *testing.T) {
	root, err := ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}
	findings, err := ThemeScopes(root)
	if err != nil {
		t.Fatalf("cannot read the window: %v", err)
	}
	if len(findings) > 0 {
		t.Errorf("%d theme(s) of their own in the window:\n%s", len(findings), strings.Join(findings, "\n"))
	}
}

// Every way of naming a theme of its own is seen, and nothing else is.
//
// The clean files are the half that keeps the guard switched on: one that
// fires on a comment, or on a type of the same spelling in a package of ours,
// is one somebody turns off on the first day it is in the way.
func TestTheThemeScopeGuardSeesEveryWayToNameOneAndNothingElse(t *testing.T) {
	root := t.TempDir()
	written := writtenUnder(t, root, "internal/gui/screen")
	const importContainer = "import (\n\t\"fyne.io/fyne/v2\"\n\t\"fyne.io/fyne/v2/container\"\n)\n\n"
	written("built.go", "package screen\n\n"+importContainer+
		"func b(o fyne.CanvasObject, t fyne.Theme) fyne.CanvasObject { return container.NewThemeOverride(o, t) }\n")
	written("literal.go", "package screen\n\n"+importContainer+
		"func l(o fyne.CanvasObject) fyne.CanvasObject { return &container.ThemeOverride{Content: o} }\n")
	written("aliased.go", "package screen\n\nimport (\n\t\"fyne.io/fyne/v2\"\n\tc \"fyne.io/fyne/v2/container\"\n)\n\n"+
		"func a(o fyne.CanvasObject, t fyne.Theme) fyne.CanvasObject {\n"+
		"\treturn wrap(c.NewThemeOverride(o, t)).Content\n}\n\n"+
		"func wrap(o fyne.CanvasObject) *c.Scroll { return c.NewVScroll(o) }\n")
	written("dotted.go", "package screen\n\nimport (\n\t\"fyne.io/fyne/v2\"\n\t. \"fyne.io/fyne/v2/container\"\n)\n\n"+
		"func d(o fyne.CanvasObject, t fyne.Theme) fyne.CanvasObject { return NewThemeOverride(o, t) }\n")
	// Clean: prose, a package of ours with the same spelling, the toolkit's
	// container used for something else, and a test file.
	written("prose.go", "package screen\n\n// Never container.NewThemeOverride here, see the guard.\nfunc p() {}\n")
	written("ours.go", "package screen\n\nimport \"example.invalid/look\"\n\n"+
		"func o() { _ = look.NewThemeOverride() }\n")
	written("stacked.go", "package screen\n\n"+importContainer+
		"func s(o fyne.CanvasObject) fyne.CanvasObject { return container.NewStack(o) }\n")
	written("built_test.go", "package screen\n\n"+importContainer+
		"func tb(o fyne.CanvasObject, t fyne.Theme) fyne.CanvasObject { return container.NewThemeOverride(o, t) }\n")
	writtenUnder(t, root, WindowProgram)("main.go", "package main\n\n"+importContainer+
		"func main() { _ = container.NewThemeOverride(nil, (fyne.Theme)(nil)) }\n")

	findings, err := ThemeScopes(root)
	if err != nil {
		t.Fatalf("cannot read the tree: %v", err)
	}
	want := []string{"internal/gui/screen/aliased.go:9", "internal/gui/screen/built.go:8",
		"internal/gui/screen/dotted.go:8", "internal/gui/screen/literal.go:8", WindowProgram + "/main.go:8"}
	if len(findings) != len(want) {
		t.Fatalf("expected %d findings, got %d:\n%s", len(want), len(findings), strings.Join(findings, "\n"))
	}
	for _, each := range want {
		if !containsPrefix(findings, each+" ") {
			t.Errorf("no finding at %s:\n%s", each, strings.Join(findings, "\n"))
		}
	}
}

// A window that is not where the guard looks is an error, not a clean window.
func TestTheThemeScopeGuardIsLoudWhenTheWindowIsGone(t *testing.T) {
	if _, err := ThemeScopes(t.TempDir()); err == nil {
		t.Error("a tree with no window was read as a window with no theme of its own")
	}
}
