package takeview

import (
	"os"
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/fynetheme"
	"github.com/donislawdev/qadatagenerator/internal/take"
)

// TestMain puts the tests under the theme of this product rather than the one
// the toolkit tests with.
//
// 🔴 It is not a nicety, since 2026-09-15. The sentences of this screen are
// labels asking for a size by the name of a rung of the type ladder, and the
// toolkit's test theme answers nought to a name it does not know. A label of
// size nought has a character width of nought, the rich text divides the
// width by it, and int(+Inf) is a slice index that panics deep in the toolkit
// on the first layout. The quick screen's tests found the same thing the day
// they began drawing with the ladder.
func TestMain(m *testing.M) {
	application := test.NewApp()
	application.Settings().SetTheme(fynetheme.Current())
	os.Exit(m.Run())
}

// TestEveryRefusalOfTheModeHasWordsOnThisScreen holds this surface to answering
// for every refusal the take and corrupt mode can produce.
//
// # Why all of them, when this screen can only produce three
//
// The kinds of damage are tick boxes and two of the settings are lists, so
// nothing a person clicks here can name a value that does not exist. Four of
// these refusals are unreachable from this screen today.
//
// The day somebody swaps a list for a text field they become reachable, and a
// screen that had skipped them would answer with silence. This guard cannot
// tell an unreachable refusal from one that just became reachable, and it
// should not have to: the cost of translating four sentences nobody sees is a
// few lines, and the cost of the other choice is a person refused with a blank
// panel.
//
// It closes in both directions, so words for a kind the mode no longer refuses
// fail too. A register answering half of that question is the failure this
// project has already paid for once.
func TestEveryRefusalOfTheModeHasWordsOnThisScreen(t *testing.T) {
	kinds := take.Kinds()
	if len(kinds) == 0 {
		t.Fatal("the mode reports no refusals at all, so this guard would check nothing")
	}

	named := map[take.RefusalKind]bool{}
	for _, kind := range kinds {
		named[kind] = true
		words, known := refusalSays[kind]
		if !known {
			t.Errorf("the mode refuses a run as %s and this screen has no words for it, "+
				"so somebody would be refused by a blank panel", kind.Name())
			continue
		}
		for part, key := range map[string]i18n.Key{
			"what": words.what, "why": words.why, "instead": words.instead,
		} {
			if key == "" {
				t.Errorf("the refusal %s has no %s on this screen, so a person reading "+
					"it is left without one of the four parts every refusal owes them",
					kind.Name(), part)
			}
		}
	}
	for kind := range refusalSays {
		if !named[kind] {
			t.Errorf("this screen carries words for %s, which the mode no longer refuses. "+
				"Remove them rather than leaving a message about something nobody can be "+
				"refused for", kind.Name())
		}
	}
}

// TestEveryWordThisScreenAsksForIsInTheCatalogue closes the other half.
//
// A key with no entry renders as the key itself, which is a label reading
// take.refusal.share on a screen somebody is trying to use. The catalogue is
// already closed against the constants, so what is left to check is that the
// keys this screen reaches for are constants rather than typed in place.
func TestEveryWordThisScreenAsksForIsInTheCatalogue(t *testing.T) {
	text := i18n.English
	for kind, words := range refusalSays {
		for _, key := range []i18n.Key{words.what, words.why, words.allowed, words.instead} {
			if key == "" {
				continue
			}
			if text.T(key) == string(key) {
				t.Errorf("the refusal %s asks for %q and the catalogue has no such entry, "+
					"so the screen would show the key itself", kind.Name(), key)
			}
		}
	}
}

// TestNothingChosenDeliberatelyIsOfferedAsOneTickBoxAmongOthers is a guard
// standing in front of a rule this screen cannot break yet.
//
// # What it protects and why it is here before it is needed
//
// The security module is never a default, by an unbreakable rule, and docs/17
// puts it in pixels: it must not be a tick box in a row of tick boxes, because
// that is exactly what makes it look like one more ordinary choice.
//
// The kinds of damage on this screen are a row of tick boxes built from the
// catalog. The catalog carries no security module today, so this guard passes
// with nothing to say, and the day one is added it fails. That is the whole
// point. Written afterwards, it would be written by somebody looking at a
// screen that already offers it, and the rule would already have been broken by
// a build nobody stopped.
func TestNothingChosenDeliberatelyIsOfferedAsOneTickBoxAmongOthers(t *testing.T) {
	described, err := catalogyaml.Load()
	if err != nil {
		t.Fatalf("the catalog inside this binary could not be read: %v", err)
	}
	offered := take.LayersIn(described)
	if len(offered) == 0 {
		t.Fatal("the catalog describes no damage at all, so this guard would check nothing")
	}
	if offersDeliberateModule(offered) {
		t.Errorf("the %s module is offered as one tick box among the others in %v. It is "+
			"never a default, and a row of identical boxes is what makes it read as one "+
			"more ordinary choice. Give it a deliberate act of its own before it reaches "+
			"this screen", deliberateModule, offered)
	}
}

// offersDeliberateModule reports whether a row of tick boxes would include the
// one module that may never be picked by accident.
//
// It is a function of its own so that the guard above can be shown to fail. On
// the real catalog it has nothing to find today, and a check that has never
// been seen to fire is a check nobody should trust.
func offersDeliberateModule(layers []string) bool {
	for _, layer := range layers {
		if strings.EqualFold(layer, deliberateModule) {
			return true
		}
	}
	return false
}

// TestTheGuardAboveWouldNoticeTheModuleArriving is the probe for it.
//
// The real catalog carries no security module, so the guard above passes with
// nothing to look at. That is the ordinary state of a rule written before it is
// needed, and it is also how such a rule quietly stops working. Here the fault
// is planted instead of waited for.
func TestTheGuardAboveWouldNoticeTheModuleArriving(t *testing.T) {
	if !offersDeliberateModule([]string{"L0", "L2", deliberateModule, "UNI"}) {
		t.Error("a row of tick boxes carrying the security module was not noticed, so " +
			"the rule against offering it as an ordinary choice is checking nothing")
	}
	if offersDeliberateModule([]string{"L0", "L2", "UNI"}) {
		t.Error("an ordinary row of tick boxes was reported as carrying the security " +
			"module, which would make this guard fire on every build and be turned off")
	}
}

// deliberateModule is the module that may never be picked by accident.
//
// Named here rather than left as a string in the guard, because the name is the
// whole subject of the rule it guards and a rule pointing at a bare literal is
// a rule whose subject somebody changes without noticing.
const deliberateModule = "SEC"

// Every setting for how many defects a record may carry has a name too.
//
// The same shape as the guard above and for the same reason: the engine takes
// exactly_one and any, and a list offering those two spellings is a settings
// file shown to somebody who opened a window to avoid one.
func TestEverySettingForDefectsPerRecordCarriesAName(t *testing.T) {
	offered := take.PerRowChoices()
	if len(offered) == 0 {
		t.Fatal("this build offers no setting at all, so this guard would check nothing")
	}
	carried := make(map[string]bool, len(offered))
	for _, value := range offered {
		carried[value] = true
		if _, named := nameOfPerRow[value]; !named {
			t.Errorf("this build offers %q and nothing names it, so the list shows the value the "+
				"engine takes rather than a choice a person can make", value)
		}
	}
	for value := range nameOfPerRow {
		if !carried[value] {
			t.Errorf("%q is named here and this build offers no such setting, so the name is a "+
				"rule about nothing", value)
		}
	}
}
