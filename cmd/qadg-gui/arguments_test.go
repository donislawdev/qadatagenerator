package main

import (
	"bytes"
	"strings"
	"testing"
)

// The command line of the window answers each way of being called the way
// DEC-253 settled: the flag opens the catalogue, nothing opens the window, a
// request for help lists the flags, and anything else is refused with no
// window at all.
func TestTheWindowOpensWhatItsCommandLineAsksForAndRefusesTheRest(t *testing.T) {
	for _, each := range []struct {
		args       []string
		catalogue  bool
		help       bool
		softwareGL bool
		refused    string
	}{
		{args: nil},
		{args: []string{"-catalogue"}, catalogue: true},
		{args: []string{"--catalogue"}, catalogue: true},
		{args: []string{"-h"}, help: true},
		{args: []string{"-software-gl"}, softwareGL: true},
		{args: []string{"--software-gl", "-catalogue"}, softwareGL: true, catalogue: true},
		// The spelling this refusal exists for.
		{args: []string{"-catalog"}, refused: "-catalog"},
		{args: []string{"-catalogue", "recipe.yaml"}, refused: "recipe.yaml"},
		{args: []string{"-catalogue=maybe"}, refused: "maybe"},
	} {
		var out bytes.Buffer
		asked, err := arguments(each.args, &out)
		name := strings.Join(each.args, " ")
		if each.refused == "" {
			if err != nil {
				t.Errorf("%q is refused: %v", name, err)
			}
			if asked.catalogue != each.catalogue || asked.help != each.help || asked.softwareGL != each.softwareGL {
				t.Errorf("%q asks for the catalogue: %v, for help: %v and for the software renderer: %v",
					name, asked.catalogue, asked.help, asked.softwareGL)
			}
			// What a second start hands on is what came, not what was understood.
			if strings.Join(asked.given, " ") != name {
				t.Errorf("%q is kept as %q for a second start", name, asked.given)
			}
			continue
		}
		if err == nil {
			t.Errorf("%q opens a window, so a wrong spelling looks like a flag that does nothing", name)
			continue
		}
		for _, part := range []string{each.refused, "allowed: -catalogue, -h, -software-gl", "try: qadg-gui -h"} {
			if !strings.Contains(err.Error(), part) {
				t.Errorf("the refusal of %q does not say %q:\n%v", name, part, err)
			}
		}
	}
}

// Help names the flags the window has and writes nothing to the error channel,
// since asking for it is not a mistake.
func TestHelpNamesTheFlagOfTheCatalogue(t *testing.T) {
	var out bytes.Buffer
	if _, err := arguments([]string{"-h"}, &out); err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{"-catalogue", "-software-gl"} {
		if !strings.Contains(out.String(), flag) {
			t.Errorf("help does not name %s:\n%s", flag, out.String())
		}
	}
}
