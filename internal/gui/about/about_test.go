package about

import (
	"regexp"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	"github.com/donislawdev/qadatagenerator/internal/license"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

// texts walks the built screen and collects every string a person would read.
//
// The screen is checked by what it says rather than by how it looks. A test
// comparing rendered pixels to a stored image goes red when a padding changes
// and says nothing about whether the licence is still on screen, which is the
// promise this screen exists to keep.
func texts(object fyne.CanvasObject) []string {
	switch typed := object.(type) {
	case *widget.Label:
		return []string{typed.Text}
	case *fyne.Container:
		var found []string
		for _, child := range typed.Objects {
			found = append(found, texts(child)...)
		}
		return found
	case *container.Scroll:
		return texts(typed.Content)
	// The scrolling area of this screen is the keyboard's rather than the
	// toolkit's own, and it is a type of its own, so a walk that knew only the
	// toolkit's stopped at it and reported an empty screen.
	case *parts.KeyScroll:
		return texts(typed.Content)
	}
	return nil
}

func screen(t *testing.T) []string {
	t.Helper()
	return texts(Content(i18n.English, license.Current(), version.Current()))
}

func has(all []string, want string) bool {
	for _, one := range all {
		if strings.Contains(one, want) {
			return true
		}
	}
	return false
}

// The screen has to say the same things the two flags say, because that is the
// parity rule made real rather than declared. A window that shows fewer facts
// than --license is a second answer to one question.
func TestTheScreenCarriesEveryFactTheLicenceFlagPrints(t *testing.T) {
	all := screen(t)
	report := license.Current()

	for _, want := range []string{report.Code.License, report.Spec.License, report.Spec.Scope} {
		if !has(all, want) {
			t.Errorf("the screen does not carry %q, which --license prints", want)
		}
	}
	if !has(all, report.Notices.GeneratedData) {
		t.Error("the screen does not carry the notice that generated data belongs to the user")
	}
	if !has(all, report.Notices.SyntheticData) {
		t.Error("the screen does not carry the notice that every value is synthetic")
	}
}

// Four versions at once, because a compliance answer is about one build and
// three of the four are contracts somebody else depends on.
//
// # Why the label is checked and not only the value
//
// Three of the four versions read 0.1 today. A test looking for the values
// alone would stay green with one of them missing from the screen, because a
// sibling still carries the same string. A mutation removing the recipe row
// proved exactly that, which is what mutations are for.
func TestTheScreenCarriesAllFourVersions(t *testing.T) {
	all := screen(t)
	versions := version.Current()
	for _, one := range []struct {
		label i18n.Key
		value string
	}{
		{i18n.AboutVersionTool, versions.Tool},
		{i18n.AboutVersionCatalog, versions.Catalog},
		{i18n.AboutVersionManifest, versions.Manifest},
		{i18n.AboutVersionRecipe, versions.Recipe},
	} {
		label := i18n.English.T(one.label)
		if !has(all, label) {
			t.Errorf("the row %q is not on the screen", label)
		}
		if !has(all, one.value) {
			t.Errorf("the version %q for %q is not on the screen", one.value, label)
		}
	}
}

// Every linked module has to be named with the day its licence was read. A
// licence is a property of a version rather than of a project, so a name
// without a date is a fact somebody remembered.
func TestEveryLinkedModuleIsNamedWithTheDayItsLicenceWasRead(t *testing.T) {
	all := screen(t)
	report := license.Current()
	if len(report.Components) == 0 {
		t.Fatal("no components in the register, so this test would pass by checking nothing")
	}
	for _, component := range report.Components {
		for _, want := range []string{component.Module, component.Version, component.License, component.Checked} {
			if !has(all, want) {
				t.Errorf("the screen does not carry %q for %s", want, component.Module)
			}
		}
	}
}

// No screen holds its own sentences, and no label renders as a key.
//
// # Why the shape is checked rather than the list of known keys
//
// The first version compared each label against i18n.Keys, which answers only
// for keys somebody declared. A mutation asking for a key that is on no list
// went unnoticed: it rendered as a dotted identifier on screen and the test
// stayed green, because the identifier was not one it knew to look for. What
// matters to a person reading the window is the shape, so that is what this
// looks at.
func TestNoLabelRendersAsItsOwnKey(t *testing.T) {
	looksLikeAKey := regexp.MustCompile(`^[a-z][a-z0-9]*(\.[a-z0-9-]+)+$`)
	for _, one := range screen(t) {
		if looksLikeAKey.MatchString(one) {
			t.Errorf("the label %q reads as a translation key rather than as a sentence", one)
		}
	}
}

// A screen built against a build carrying nothing has to say so rather than
// show an empty space, which reads as a layout fault.
func TestAnEmptyRegisterIsStatedRatherThanLeftBlank(t *testing.T) {
	empty := license.Report{}
	all := texts(Content(i18n.English, empty, version.Current()))
	if !has(all, i18n.English.T(i18n.AboutEmptyComponents)) {
		t.Error("a build with no linked module shows nothing where the sentence saying so belongs")
	}
}
