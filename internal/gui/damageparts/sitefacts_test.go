package damageparts

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
)

var update = flag.Bool("update", false, "write the website's facts about the window again")

// The project website calls each kind of damage what this window calls it, so a
// person who read the site finds the same words on the tick boxes. The names
// live here, in a package of the window, which the command line cannot import,
// so they get a file of facts of their own beside the one internal/cli writes
// (DEC-305).
func TestTheWindowFactsAreWhatTheWindowSays(t *testing.T) {
	now := encodeWindowFacts(t, gatherWindowFacts())
	if *update {
		if err := os.WriteFile(windowFactsFile, now, 0o644); err != nil {
			t.Fatalf("cannot write %s: %v", windowFactsFile, err)
		}
		t.Logf("wrote %s. Read its diff before committing it", windowFactsFile)
		return
	}
	committed, err := os.ReadFile(windowFactsFile)
	if err != nil {
		t.Fatalf("cannot read %s, which the website is built from: %v\nWrite it with\n  %s",
			windowFactsFile, err, windowFactsRedo)
	}
	committed = bytes.ReplaceAll(committed, []byte("\r\n"), []byte("\n"))
	if !bytes.Equal(committed, now) {
		t.Errorf("the website names the kinds of damage differently from this window.\n"+
			"  file:   %s\n  window: %s\nWrite it again with\n  %s\nthen read the diff and commit it",
			strings.TrimSpace(string(committed)), strings.TrimSpace(string(now)), windowFactsRedo)
	}
}

// A kind of damage renamed in the window gives other facts, so the comparison
// above cannot pass a site that still shows the old name.
func TestARenamedKindOfDamageChangesTheWindowFacts(t *testing.T) {
	facts := gatherWindowFacts()
	before := encodeWindowFacts(t, facts)
	facts.Groups[0].Name += " renamed"
	if bytes.Equal(before, encodeWindowFacts(t, facts)) {
		t.Fatal("a renamed kind of damage gave the same facts, so the website could keep a name " +
			"the window no longer shows")
	}
}

const (
	windowFactsFile = "../../../web/data/facts/window.json"
	windowFactsRedo = "go test ./internal/gui/damageparts -run TestTheWindowFactsAreWhatTheWindowSays -update"
)

type windowFacts struct {
	Groups []windowGroup `json:"groups"`
}

// windowGroup is one kind of damage as the tick box names it. The site shows
// only the kinds the command line facts carry damage in.
type windowGroup struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

func gatherWindowFacts() windowFacts {
	codes := make([]string, 0, len(nameOfLayer))
	for code := range nameOfLayer {
		codes = append(codes, code)
	}
	slices.Sort(codes)
	facts := windowFacts{}
	for _, code := range codes {
		facts.Groups = append(facts.Groups, windowGroup{Code: code, Name: i18n.English.T(nameOfLayer[code])})
	}
	return facts
}

func encodeWindowFacts(t *testing.T, facts windowFacts) []byte {
	t.Helper()
	var b bytes.Buffer
	encoder := json.NewEncoder(&b)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(facts); err != nil {
		t.Fatalf("cannot encode the window facts: %v", err)
	}
	return b.Bytes()
}
