package manifest_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
)

// worked is an envelope carrying a schema nobody declared, in the shape the
// mode that reads somebody else's file produces.
func worked() manifest.Run {
	envelope := run()
	envelope.Source = "contacts.csv"
	envelope.Inference = &manifest.Inference{
		Method: "values-validated",
		Columns: []manifest.InferredColumn{
			{Field: "pesel", Type: "identifier", Scheme: "PL.pesel",
				Basis: "every one of 40 values was accepted by the PL.pesel validator"},
			{Field: "note", Type: "string",
				Basis: "no numbering scheme this build carries accepted 40 values"},
		},
		NotInferred: []string{"maximum length", "required columns"},
	}
	return envelope
}

func TestAnEnvelopeCarryingAWorkedOutSchemaIsAccepted(t *testing.T) {
	if problems := worked().Problems(); len(problems) > 0 {
		t.Fatalf("a complete inference was refused: %v", problems)
	}
}

// An envelope from a recipe carries none, because nothing was guessed. Making
// this the ordinary case rather than an omission is what keeps the two modes
// apart in the document.
func TestAnEnvelopeWithNoInferenceIsAccepted(t *testing.T) {
	if problems := run().Problems(); len(problems) > 0 {
		t.Fatalf("an envelope from a recipe was refused for not guessing: %v", problems)
	}
}

// 🔴 The rule worth defending: an inference that lists what it recognised and
// stops reads as a complete account of the file. A reader then learns nothing
// about the columns where the damage they ordered had nothing to work with,
// which is the silence this product exists to find, one level up.
func TestAnInferenceThatDoesNotSayWhatItMissesIsRefused(t *testing.T) {
	envelope := worked()
	envelope.Inference.NotInferred = nil
	problems := envelope.Problems()
	if len(problems) == 0 {
		t.Fatal("an inference claiming no limits was accepted, so it reads as complete")
	}
	if !strings.Contains(strings.Join(problems, " "), "does not infer") {
		t.Errorf("the refusal has to say what is missing, and says %v", problems)
	}
}

// A conclusion with no evidence is a conclusion nobody can argue with, and the
// person who has to argue with it is the one whose file it is.
func TestAColumnWithNoBasisIsRefused(t *testing.T) {
	envelope := worked()
	envelope.Inference.Columns[0].Basis = "   "
	problems := envelope.Problems()
	if len(problems) == 0 {
		t.Fatal("a conclusion with no evidence behind it was accepted")
	}
	if !strings.Contains(strings.Join(problems, " "), "pesel") {
		t.Errorf("the refusal has to name the column, and says %v", problems)
	}
}

func TestAnInferenceMissingItsOtherParticularsIsRefused(t *testing.T) {
	for _, given := range []struct {
		what   string
		breaks func(*manifest.Inference)
		says   string
	}{
		{"no method", func(i *manifest.Inference) { i.Method = "" }, "without saying how"},
		{"no columns", func(i *manifest.Inference) { i.Columns = nil }, "names no column"},
		{"a column with no name", func(i *manifest.Inference) {
			i.Columns[1].Field = ""
		}, "has no name"},
	} {
		envelope := worked()
		given.breaks(envelope.Inference)
		problems := envelope.Problems()
		if len(problems) == 0 {
			t.Errorf("an inference with %s was accepted", given.what)
			continue
		}
		if !strings.Contains(strings.Join(problems, " "), given.says) {
			t.Errorf("an inference with %s was refused with %v, which does not say %q",
				given.what, problems, given.says)
		}
	}
}

// Everything wrong at once comes back at once, because whoever is fixing this
// is fixing code of ours rather than a recipe of theirs, and a list of one
// costs a second run per problem.
func TestEveryProblemWithAnInferenceIsReportedTogether(t *testing.T) {
	envelope := worked()
	envelope.Inference.Method = ""
	envelope.Inference.NotInferred = nil
	envelope.Inference.Columns[0].Basis = ""
	if problems := envelope.Problems(); len(problems) < 3 {
		t.Errorf("3 things were wrong and %d were reported: %v", len(problems), problems)
	}
}

// A manifest whose envelope would be refused must not reach a disk, and the
// streaming writer asks this before it writes anything for exactly that reason.
func TestAWholeManifestRefusesAnUnusableInference(t *testing.T) {
	envelope := worked()
	envelope.Inference.NotInferred = nil
	_, problems := manifest.New("0.1", envelope, []manifest.Defect{applied()})
	if len(problems) == 0 {
		t.Fatal("a manifest carrying an inference the envelope refuses was built anyway")
	}
}
