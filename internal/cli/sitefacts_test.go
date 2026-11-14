package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/mutators"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

// The project website shows what this program says about itself: every
// command with its flags, every exit code, every generator and every kind of
// damage this build carries out. It reads them from web/data/facts/cli.json
// rather than from words written a second time, and this test writes that file
// and holds it to the program (DEC-305).
//
// Run with -update after a deliberate change, then read the diff of the file:
// it is the list of what every page showing these facts now says differently.
//
// What it does not check: the sentences the site writes around these facts,
// and the README, which TestTheReadmeShowsWhatTheProgramDoes walks.
func TestTheSiteFactsAreWhatTheProgramSays(t *testing.T) {
	now := encodeForSite(t, gatherSiteFacts(t))
	if *updateDataset {
		if err := os.WriteFile(siteFactsFile, now, 0o644); err != nil {
			t.Fatalf("cannot write %s: %v", siteFactsFile, err)
		}
		t.Logf("wrote %s. Read its diff before committing it", siteFactsFile)
		return
	}
	committed, err := os.ReadFile(siteFactsFile)
	if err != nil {
		t.Fatalf("cannot read %s, which the website is built from: %v\nWrite it with\n  %s",
			siteFactsFile, err, siteFactsRedo)
	}
	if problem := siteFactsProblem(committed, now); problem != "" {
		t.Error(problem)
	}
}

// The comparison above notices a program that gained a generator the file does
// not name, so a website that fell behind the program cannot pass.
func TestTheSiteFactsCheckNoticesAProgramThatMovedOn(t *testing.T) {
	facts := gatherSiteFacts(t)
	before := encodeForSite(t, facts)
	facts.Generators = append(facts.Generators, siteGenerator{Name: "planted", Produces: "string"})
	after := encodeForSite(t, facts)
	if siteFactsProblem(before, after) == "" {
		t.Fatal("facts naming one more generator read as the same facts, so the website " +
			"could show a list the program no longer has")
	}
	if siteFactsProblem(before, before) != "" {
		t.Fatal("the same facts were reported as different, so the check would fail every build")
	}
}

const (
	siteFactsFile = "../../web/data/facts/cli.json"
	siteFactsRedo = "go test ./internal/cli -run TestTheSiteFactsAreWhatTheProgramSays -update"
)

// siteFacts is the whole file. Every list keeps the order the program keeps,
// so the site shows them in the order the help prints them.
type siteFacts struct {
	Version    string          `json:"version"`
	Commands   []siteCommand   `json:"commands"`
	Flags      []siteFlag      `json:"flags"`
	ExitCodes  []siteExitCode  `json:"exitCodes"`
	Locales    []string        `json:"locales"`
	Generators []siteGenerator `json:"generators"`
	Damage     []siteDamage    `json:"damage"`
}

type siteCommand struct {
	Name      string     `json:"name"`
	Arguments string     `json:"arguments"`
	Summary   string     `json:"summary"`
	What      string     `json:"what"`
	Example   string     `json:"example"`
	Flags     []siteFlag `json:"flags"`
}

type siteFlag struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
}

type siteExitCode struct {
	Code    int    `json:"code"`
	Name    string `json:"name"`
	Meaning string `json:"meaning"`
}

// siteGenerator says where a generator can be filled. A locale it cannot be
// filled in carries the reason help prints for it.
type siteGenerator struct {
	Name     string            `json:"name"`
	Produces string            `json:"produces"`
	Missing  map[string]string `json:"missing,omitempty"`
}

// siteDamage is one kind of damage this build carries out, as the catalog
// describes it. The example pair is the one the catalog holds to the engine.
type siteDamage struct {
	ID          string `json:"id"`
	Group       string `json:"group"`
	Summary     string `json:"summary"`
	Expectation string `json:"expectation"`
	Confidence  string `json:"confidence"`
	Before      string `json:"before"`
	After       string `json:"after"`
}

func gatherSiteFacts(t *testing.T) siteFacts {
	t.Helper()
	facts := siteFacts{Version: version.Current().Tool, Locales: values.Locales()}

	for _, doc := range commandDocs {
		command := siteCommand{
			Name:      doc.name,
			Arguments: doc.arguments,
			Summary:   doc.summary,
			What:      unwrapped(doc.what),
			Example:   doc.example,
			Flags:     siteFlagsOf(doc.flags),
		}
		facts.Commands = append(facts.Commands, command)
	}
	facts.Flags = siteFlagsOf(flagDocs)
	for _, entry := range exitTable {
		facts.ExitCodes = append(facts.ExitCodes, siteExitCode{int(entry.code), entry.name, entry.meaning})
	}

	source := values.New("", "site")
	for _, name := range source.Names() {
		generator := siteGenerator{Name: name, Produces: producedBy(source, name)}
		for _, locale := range facts.Locales {
			if fine, why := source.CanProduce(name, locale); !fine {
				if generator.Missing == nil {
					generator.Missing = map[string]string{}
				}
				generator.Missing[locale] = why
			}
		}
		facts.Generators = append(facts.Generators, generator)
	}

	loaded, err := catalogyaml.Load()
	if err != nil {
		t.Fatalf("cannot load the catalog: %v", err)
	}
	for _, id := range mutators.IDs() {
		entry, described := loaded.Lookup(id)
		if !described {
			t.Fatalf("%s is carried out by this build and the catalog does not describe it", id)
		}
		group, _, _ := strings.Cut(string(id), ".")
		facts.Damage = append(facts.Damage, siteDamage{
			ID:          string(id),
			Group:       group,
			Summary:     unwrapped(entry.Summary),
			Expectation: string(entry.Expectation),
			Confidence:  string(entry.Confidence),
			Before:      entry.Example.Before,
			After:       entry.Example.After,
		})
	}
	return facts
}

func siteFlagsOf(docs []flagDoc) []siteFlag {
	flags := []siteFlag{}
	for _, doc := range docs {
		flags = append(flags, siteFlag{Name: doc.shown, Summary: doc.what})
	}
	return flags
}

// unwrapped joins the lines a terminal needs into the sentences a web page
// wraps by itself.
func unwrapped(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// siteFactsProblem compares the committed file with what the program says now,
// and names the first line where they part.
func siteFactsProblem(committed, now []byte) string {
	committed = bytes.ReplaceAll(committed, []byte("\r\n"), []byte("\n"))
	if bytes.Equal(committed, now) {
		return ""
	}
	return firstDifference(filepath.ToSlash(siteFactsFile), committed, now, siteFactsRedo)
}

// firstDifference says where a file of the site first parts from what the
// program makes now, and how to write it again. Both the facts and the runs
// of the site are held this way.
func firstDifference(name string, committed, now []byte, redo string) string {
	was, is := strings.Split(string(committed), "\n"), strings.Split(string(now), "\n")
	at := 0
	for at < len(was) && at < len(is) && was[at] == is[at] {
		at++
	}
	return "the website shows what the program no longer does, first at line " +
		strconv.Itoa(at+1) + " of " + name + ":\n  file:    " +
		lineOr(was, at) + "\n  program: " + lineOr(is, at) +
		"\nWrite it again with\n  " + redo + "\nthen read the diff and commit it."
}

func lineOr(lines []string, at int) string {
	if at < len(lines) {
		return strings.TrimSpace(lines[at])
	}
	return "(the file ends here)"
}
