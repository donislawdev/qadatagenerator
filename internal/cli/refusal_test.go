package cli

import (
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/take"
)

// TestEveryRefusalOfTheModeHasWordsOfItsOwn holds this surface to answering for
// every refusal the take and corrupt mode can produce, and holds the register
// here to naming nothing the mode no longer refuses.
//
// # The failure this exists for, which is not the obvious one
//
// The obvious one is a refusal with no words at all. The compiler does not
// report it, because turning a kind into words is a lookup and a lookup that
// misses returns something.
//
// Before this guard the lookup was a switch, and its last branch caught every
// kind nobody had thought about. What came out was not silence, which somebody
// would have noticed. It was a complete, confident, four part message about a
// different refusal entirely: a person whose delimiter was wrong being told
// their run would write over the file it reads. They would then go and fix a
// path that was never the problem.
//
// So this checks both directions, which is the whole reason the switch became a
// register: a kind with no words fails, and words for a kind that no longer
// exists fail too. A register answering only the first question answers half of
// what was asked, which this project has paid for before.
func TestEveryRefusalOfTheModeHasWordsOfItsOwn(t *testing.T) {
	kinds := take.Kinds()
	if len(kinds) == 0 {
		t.Fatal("the mode reports no refusals at all, so this guard would check nothing")
	}

	named := map[take.RefusalKind]bool{}
	saidBy := map[string]string{}
	for _, kind := range kinds {
		named[kind] = true
		words, known := refusalSays[kind]
		if !known {
			t.Errorf("the mode refuses a run as %s and this surface has no words for "+
				"it, so somebody would be refused without being told why", kind.Name())
			continue
		}
		for part, text := range map[string]string{
			"what": words.what, "why": words.why, "instead": words.instead,
		} {
			if text == "" {
				t.Errorf("the refusal %s has no %s, so a person reading it is left "+
					"without one of the four parts every refusal here owes them",
					kind.Name(), part)
			}
		}
		if earlier, taken := saidBy[words.what]; taken {
			t.Errorf("the refusals %s and %s open with the same line %q, so one of "+
				"them is being described as the other", earlier, kind.Name(), words.what)
			continue
		}
		saidBy[words.what] = kind.Name()
	}

	for kind := range refusalSays {
		if !named[kind] {
			t.Errorf("this surface carries words for %s, which the mode no longer "+
				"refuses. Remove them rather than leaving a message about something "+
				"nobody can be refused for", kind.Name())
		}
	}
}

// TestARefusalWithNoWordsSaysSoRatherThanBorrowingSomeoneElses is the probe.
//
// It plants the fault the guard above exists to catch, which is an ADDITION: a
// kind the register has never heard of. What matters is not that something
// comes out, but that what comes out does not read as one of the real refusals.
// A person told the wrong cause acts on the wrong cause.
func TestARefusalWithNoWordsSaysSoRatherThanBorrowingSomeoneElses(t *testing.T) {
	unknown := take.RefusalKind(9999)
	said := refusalOf(take.Refusal{Kind: unknown})
	if said.what == "" || said.why == "" || said.instead == "" {
		t.Fatal("a refusal this surface has no words for came out incomplete, which " +
			"is the silence this tool refuses everywhere else")
	}
	for _, kind := range take.Kinds() {
		if words := refusalSays[kind]; words.what != "" && said.what == words.what {
			t.Errorf("a refusal nobody gave words to is described as %s, so a person "+
				"would be sent to fix something that was never wrong", kind.Name())
		}
	}
}

// TestARefusalNobodyFilledInIsNotMistakenForOne guards the zero value.
//
// A refusal is returned by value, so every path that does not refuse returns an
// empty one. If that empty value read as a refusal, a run that was fine would be
// stopped by a message about whatever the lookup happened to find.
func TestARefusalNobodyFilledInIsNotMistakenForOne(t *testing.T) {
	if (take.Refusal{}).Refused() {
		t.Error("an empty refusal reports itself as a refusal, so every path that " +
			"passes its checks would be stopped by one")
	}
	for _, kind := range take.Kinds() {
		if !(take.Refusal{Kind: kind}).Refused() {
			t.Errorf("the refusal %s does not report itself as one, so it would be "+
				"passed over and the run would carry on as though nothing was wrong",
				kind.Name())
		}
	}
}
