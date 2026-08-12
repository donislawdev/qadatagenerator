package manifestjson_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
)

func anEntry(id string) manifest.Defect {
	return manifest.Defect{
		ID:             id,
		Location:       manifest.Location{Entity: "customer", Row: 1, Field: "amount"},
		Mutator:        "L2.type.string_in_numeric",
		Layer:          catalog.FieldShape,
		Applied:        true,
		ValueBefore:    "1250",
		HasValueBefore: true,
		ValueAfter:     "n/a",
		HasValueAfter:  true,
		Expectation:    catalog.Rejection,
		Confidence:     catalog.Hard,
		Rationale:      "Puts text where the schema declares a number.",
	}
}

func anEnvelope() manifest.Run {
	return manifest.Run{
		GeneratedAt: "2026-08-11T00:00:00Z", Seed: "s", Locale: "pl-PL",
		CatalogVersion: "0.1", RowsTotal: 10, RowsMutated: 1,
	}
}

// The document parses, and it is the shape the specification describes.
func TestTheDocumentIsReadableBySomebodyElsesParser(t *testing.T) {
	var buffer bytes.Buffer
	writer := manifestjson.New(&buffer, "0.1")
	writer.Add(anEntry("one"))
	writer.Add(anEntry("two"))
	if err := writer.Finish(anEnvelope(), 2, 2); err != nil {
		t.Fatal(err)
	}

	var document map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &document); err != nil {
		t.Fatalf("the manifest is not a JSON document: %v\n%s", err, buffer.String())
	}
	for _, member := range []string{"spec_version", "defects", "run"} {
		if _, carried := document[member]; !carried {
			t.Errorf("the document has no %s", member)
		}
	}
	if entries, ok := document["defects"].([]any); !ok || len(entries) != 2 {
		t.Errorf("the document carries %v entries, want 2", document["defects"])
	}
}

// An empty run is still a document. A policy that ordered nothing, or a table of
// no rows, has to produce a manifest somebody can open rather than an empty file.
func TestARunWithNoEntriesIsStillADocument(t *testing.T) {
	var buffer bytes.Buffer
	writer := manifestjson.New(&buffer, "0.1")
	if err := writer.Finish(anEnvelope(), 0, 0); err != nil {
		t.Fatal(err)
	}
	var document struct {
		Defects []json.RawMessage `json:"defects"`
		Run     struct {
			Requested int `json:"defects_requested"`
		} `json:"run"`
	}
	if err := json.Unmarshal(buffer.Bytes(), &document); err != nil {
		t.Fatalf("an empty manifest does not parse: %v\n%s", err, buffer.String())
	}
	if len(document.Defects) != 0 || document.Run.Requested != 0 {
		t.Errorf("an empty run produced %d entries and a count of %d",
			len(document.Defects), document.Run.Requested)
	}
}

// 🔴 An entry the model would refuse never reaches the file.
//
// A manifest built as a value is checked as a whole, and by then a streaming
// writer has already put its entries on a disk. So the check moves to the last
// moment where refusing still means something. Found by running the binary: an
// order aimed at no column produced an entry with no field, which the model
// refuses for a defect in a value.
func TestAnEntryTheModelWouldRefuseNeverReachesTheFile(t *testing.T) {
	var buffer bytes.Buffer
	writer := manifestjson.New(&buffer, "0.1")

	broken := anEntry("no-field")
	broken.Location.Field = ""
	writer.Add(broken)

	err := writer.Finish(anEnvelope(), 1, 1)
	if err == nil {
		t.Fatal("an entry the manifest model refuses was written out")
	}
	if strings.Contains(buffer.String(), "no-field") {
		t.Error("the refused entry reached the document anyway")
	}
}

// A value the mutator produced on purpose survives, including an empty one. One
// catalog entry exists to replace a field with nothing, so empty and absent are
// different answers.
func TestAValueThatIsEmptyOnPurposeIsWrittenRatherThanDropped(t *testing.T) {
	var buffer bytes.Buffer
	writer := manifestjson.New(&buffer, "0.1")
	emptied := anEntry("emptied")
	emptied.ValueAfter = ""
	writer.Add(emptied)
	if err := writer.Finish(anEnvelope(), 1, 1); err != nil {
		t.Fatal(err)
	}

	var document struct {
		Defects []map[string]any `json:"defects"`
	}
	if err := json.Unmarshal(buffer.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	after, carried := document.Defects[0]["value_after"]
	if !carried {
		t.Fatal("a value emptied on purpose disappeared from the manifest")
	}
	if after != "" {
		t.Errorf("the emptied value came out as %v", after)
	}
}

// The counts in the envelope are what the caller measured, not what this writer
// happened to see. Two sources for one number is how a summary comes to
// disagree with the list under it.
func TestTheEnvelopeCarriesTheCountsItWasGiven(t *testing.T) {
	var buffer bytes.Buffer
	writer := manifestjson.New(&buffer, "0.1")
	writer.Add(anEntry("one"))
	if err := writer.Finish(anEnvelope(), 60, 1); err != nil {
		t.Fatal(err)
	}
	var document struct {
		Run struct {
			Requested int `json:"defects_requested"`
			Applied   int `json:"defects_applied"`
		} `json:"run"`
	}
	if err := json.Unmarshal(buffer.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if document.Run.Requested != 60 || document.Run.Applied != 1 {
		t.Errorf("the envelope says %d ordered and %d applied, want 60 and 1",
			document.Run.Requested, document.Run.Applied)
	}
}

// 🔴 Every part of an inference the model carries reaches the file.
//
// This is a guard against a shape that has bitten twice: the model gains a
// member, the writer keeps its own document type, and both halves are correct
// on their own while the file quietly says less than the run knew. It happened
// here with relations, which the envelope carried and the document dropped, and
// nothing failed because nothing compared the two.
//
// The relation is what damage contradicting a neighbouring column stands on, so
// a manifest missing it asks the reader to take that damage on trust.
func TestAnInferredRelationReachesTheFileWithItsBasis(t *testing.T) {
	var buffer bytes.Buffer
	writer := manifestjson.New(&buffer, "0.1")
	envelope := anEnvelope()
	envelope.Inference = &manifest.Inference{
		Method:  "values-validated",
		Columns: []manifest.InferredColumn{{Field: "pesel", Type: "identifier", Basis: "every value"}},
		Relations: []manifest.InferredRelation{
			{Field: "pesel", With: "sex", Kind: "gender", Basis: "every record agrees"},
		},
		NotInferred: []string{"maximum length"},
	}
	writer.Add(anEntry("one"))
	if err := writer.Finish(envelope, 1, 1); err != nil {
		t.Fatalf("the manifest did not close: %v", err)
	}

	var document struct {
		Run struct {
			Inference struct {
				Relations []struct {
					Field, With, Kind, Basis string
				} `json:"relations"`
			} `json:"schema_inference"`
		} `json:"run"`
	}
	if err := json.Unmarshal(buffer.Bytes(), &document); err != nil {
		t.Fatalf("the document does not parse: %v", err)
	}
	written := document.Run.Inference.Relations
	if len(written) != 1 {
		t.Fatalf("the run proved one relation and the file carries %d:\n%s",
			len(written), buffer.String())
	}
	if written[0].Field != "pesel" || written[0].With != "sex" {
		t.Errorf("the relation names %s and %s", written[0].Field, written[0].With)
	}
	if written[0].Kind == "" || written[0].Basis == "" {
		t.Errorf("the relation reached the file without saying what agrees or why: %+v", written[0])
	}
}
