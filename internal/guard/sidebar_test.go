package guard_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/guard"
)

// Every screen in the sidebar carries an icon, in both copies of that list.
//
// 🔴 This came out of the owner putting the program beside the accepted drawing
// on 2026-09-09 and asking why the program looked worse. Most of the answer was
// here: the list of screens exists twice, once in cmd/qadg-gui and once in
// internal/gui/screens, and only the second had icons. The second is what the
// probe renders and what every test reads, so every picture anybody looked at
// showed a column of five names with an icon each, while the program drew five
// names with a gap beside them - the layout reserves that space either way.
//
// The fault was an absence on the one surface nothing photographs, which is why
// it lived from the day the icons were drawn until somebody ran the binary.
func TestEverySidebarScreenCarriesAnIcon(t *testing.T) {
	root, err := guard.ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}
	findings, err := guard.SidebarScreens(root)
	if err != nil {
		t.Fatalf("the sidebar could not be read: %v", err)
	}
	for _, finding := range findings {
		t.Error(finding)
	}
}

// The guard sees a screen built without an icon, and is loud when there is
// nothing to read.
//
// Both halves matter, and the second one more. A guard that only objects to
// what it finds is green when it finds nothing at all, which is the state a
// renamed type or a moved package leaves it in - and that is the same class of
// silence that let the missing icons live.
func TestTheSidebarGuardSeesABareScreenAndIsLoudWhenThereIsNothingToRead(t *testing.T) {
	root := t.TempDir()
	build := func(where, source string) {
		dir := filepath.Join(root, filepath.FromSlash(where))
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	// The probe's copy, written the way that one writes it: a slice sized
	// first, then an entry per index, qualified by the package.
	build("internal/gui", `package gui

func build() {
	column := make([]shell.Screen, 1)
	column[0] = shell.Screen{Name: text.T(i18n.ScreenQuick), Icon: paint.IconBolt}
	_ = column
}
`)
	// The program's copy, written the way that one writes it, and bare.
	build("cmd/qadg-gui", `package main

func build() {
	_ = shell.New([]shell.Screen{
		{Name: text.T(i18n.ScreenTake)},
	})
}
`)

	findings, err := guard.SidebarScreens(root)
	if err != nil {
		t.Fatalf("a tree carrying both copies could not be read: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("one bare screen beside one that carries an icon produced %d findings: %v",
			len(findings), findings)
	}
	// The message names the screen, because a finding a person cannot act on
	// without opening the file is half a finding.
	if want := "ScreenTake"; !strings.Contains(findings[0], want) {
		t.Errorf("the finding does not name the screen it is about (%q): %s", want, findings[0])
	}

	// 🔴 The probe's copy must not be reported, and it is written in the other
	// of the two shapes this guard has to read: a literal naming its type in
	// full, where the program's copy writes entries inside a slice literal that
	// name no type at all. The first version of this guard read only the first
	// shape and therefore found nothing wrong with the copy that was wrong.
	if len(findings) > 0 && strings.Contains(findings[0], "internal/gui") {
		t.Errorf("the guard objected to the probe's copy, which names an icon: %s", findings[0])
	}

	// Nothing to read at all is a finding rather than a pass.
	empty := t.TempDir()
	if _, err := guard.SidebarScreens(empty); err == nil {
		t.Error("a tree with neither copy in it was read without complaint, so the guard would " +
			"pass by checking nothing")
	}
}
