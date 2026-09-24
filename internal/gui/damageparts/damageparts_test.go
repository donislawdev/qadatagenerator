package damageparts

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestexport"
	"github.com/donislawdev/qadatagenerator/internal/adapter/mutators"
)

// Every kind of damage this build offers has a name a person can read.
//
// 🔴 WRITTEN WITH THE NAMES THEMSELVES, on 2026-09-08, and it is the half that
// makes them safe. A kind of damage missing from the map falls back to its bare
// code, which is exactly what the map was written to remove and which nothing
// else would notice: the tick box appears, it works, and it is labelled L4.
// Since 2026-09-24 the names serve two screens, so a missing one is missing on
// both.
//
// Closed from both sides on purpose. A name for a kind the catalogue does not
// carry is a rule about nothing, and it would sit here looking like cover.
func TestEveryKindOfDamageOfferedCarriesAName(t *testing.T) {
	described, err := catalogyaml.Load()
	if err != nil {
		t.Fatalf("the catalog inside this binary could not be read: %v", err)
	}
	offered := mutators.Coverage{Described: described}.Layers()
	if len(offered) == 0 {
		t.Fatal("the catalog describes no damage at all, so this guard would check nothing")
	}
	carried := make(map[string]bool, len(offered))
	for _, code := range offered {
		carried[code] = true
		if _, named := nameOfLayer[code]; !named {
			t.Errorf("this build offers the kind of damage %q and nothing names it, so the tick "+
				"box is labelled with the code itself, which is what naming them was for", code)
		}
	}
	for code := range nameOfLayer {
		if !carried[code] {
			t.Errorf("%q is named here and this build offers no such kind of damage, so the name "+
				"is a rule about nothing sitting where a rule should be", code)
		}
	}
}

// The list of extra forms names the absence of one, and picking that name
// gives a run no form rather than a form called "none".
//
// Both screens that offer the list take it from here, so a run started from
// either of them with the first entry picked must export nothing.
func TestTheFirstFormOnTheListIsNoFormAtAll(t *testing.T) {
	choices := FormChoices(i18n.English)
	if len(choices) != len(manifestexport.Forms())+1 {
		t.Fatalf("the list offers %v, which is not every form this build writes and the absence of one",
			choices)
	}
	if FormNamed(i18n.English, choices[0]) != "" {
		t.Errorf("picking %q gives a run the form %q rather than none", choices[0],
			FormNamed(i18n.English, choices[0]))
	}
	for _, form := range manifestexport.Forms() {
		if FormNamed(i18n.English, form) != form {
			t.Errorf("picking %q gives a run %q", form, FormNamed(i18n.English, form))
		}
	}
	if !strings.Contains(LayerLabel(i18n.English, "L3"), "L3") {
		t.Errorf("a kind of damage named on screen lost its code: %q", LayerLabel(i18n.English, "L3"))
	}
}
