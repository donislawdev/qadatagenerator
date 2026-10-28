package manifestjson_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/fault"
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
		Entities: []manifest.Entity{{Name: "customer", Rows: 10}},
	}
}

// 🔴 The list of entities is written whole, and an envelope that has none or
// leaves out a table an entry stands in is refused before the document closes.
//
// The list is what a response record about two tables is held to (DEC-266). A
// manifest without it is one verify could only read as the sum of its tables,
// and the entries are on a disk by the time the envelope arrives, so the close
// is the last moment a refusal still means something.
func TestTheEntitiesAreWrittenWholeOrNotAtAll(t *testing.T) {
	var buffer bytes.Buffer
	writer := manifestjson.New(&buffer, "0.2")
	writer.Add(anEntry("one"))
	envelope := anEnvelope()
	envelope.Entities = []manifest.Entity{{Name: "customer", Rows: 4}, {Name: "order", Rows: 6}}
	if err := writer.Finish(envelope, 1, 1); err != nil {
		t.Fatal(err)
	}
	var document struct {
		Run struct {
			Entities []struct {
				Entity    string `json:"entity"`
				RowsTotal int    `json:"rows_total"`
			} `json:"entities"`
		} `json:"run"`
	}
	if err := json.Unmarshal(buffer.Bytes(), &document); err != nil {
		t.Fatalf("the manifest does not parse: %v\n%s", err, buffer.String())
	}
	if got := document.Run.Entities; len(got) != 2 || got[0].Entity != "customer" ||
		got[0].RowsTotal != 4 || got[1].Entity != "order" || got[1].RowsTotal != 6 {
		t.Errorf("the entities reached the file as %+v", got)
	}

	for _, given := range []struct {
		what   string
		listed []manifest.Entity
		says   string
	}{
		{"no entity at all", nil, "lists no entity"},
		{"the table of an entry left out", []manifest.Entity{{Name: "order", Rows: 10}}, "does not list"},
	} {
		t.Run(given.what, func(t *testing.T) {
			writer := manifestjson.New(&bytes.Buffer{}, "0.2")
			writer.Add(anEntry("one"))
			envelope := anEnvelope()
			envelope.Entities = given.listed
			err := writer.Finish(envelope, 1, 1)
			if err == nil || !strings.Contains(err.Error(), given.says) {
				t.Errorf("an envelope with %s closed with %v", given.what, err)
			}
		})
	}
}

// The document parses, and it is the shape the specification describes.
func TestTheDocumentIsReadableBySomebodyElsesParser(t *testing.T) {
	var buffer bytes.Buffer
	writer := manifestjson.New(&buffer, "0.2")
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
	writer := manifestjson.New(&buffer, "0.2")
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
	writer := manifestjson.New(&buffer, "0.2")

	broken := anEntry("no-field")
	broken.Location.Field = ""
	// The refusal is answered at once rather than held for the close, so that
	// the run ends at this entry instead of producing the rest of its dataset.
	if err := writer.Add(broken); err == nil {
		t.Error("an entry the manifest model refuses was taken without a word")
	}

	err := writer.Finish(anEnvelope(), 1, 1)
	if err == nil {
		t.Fatal("an entry the manifest model refuses was written out")
	}
	if strings.Contains(buffer.String(), "no-field") {
		t.Error("the refused entry reached the document anyway")
	}
}

// counted is a file that counts the writes it is handed.
//
// Write is all it has, like the file of a run. A fake that also took a string
// would be handed large pieces past the buffer and past this count.
type counted struct {
	kept   bytes.Buffer
	writes int
}

func (c *counted) Write(p []byte) (int, error) {
	c.writes++
	return c.kept.Write(p)
}

// takesUpTo is a file with room for so many bytes and no more.
type takesUpTo struct {
	room  int
	tried int
	kept  bytes.Buffer
}

var errNoRoom = errors.New("there is no room left on the disk")

func (f *takesUpTo) Write(p []byte) (int, error) {
	f.tried++
	if len(p) > f.room {
		taken := f.room
		f.kept.Write(p[:taken])
		f.room = 0
		return taken, errNoRoom
	}
	f.room -= len(p)
	f.kept.Write(p)
	return len(p), nil
}

// The document reaches its file in pieces of a few kilobytes rather than one
// write per entry. Until 2026-10-28 every entry was a call to the system of its
// own (found by an outside review on 2026-10-22), so a run of a hundred
// thousand defects made a hundred thousand of them.
func TestTheManifestReachesItsFileInPiecesRatherThanAnEntryAtATime(t *testing.T) {
	file := &counted{}
	writer := manifestjson.New(file, "0.2")
	const entries = 500
	for at := 0; at < entries; at++ {
		if err := writer.Add(anEntry(fmt.Sprintf("entry-%d", at))); err != nil {
			t.Fatalf("entry %d was refused: %v", at, err)
		}
	}
	if err := writer.Finish(anEnvelope(), entries, entries); err != nil {
		t.Fatal(err)
	}
	if most := file.kept.Len()/4096 + 1; file.writes > most {
		t.Errorf("%d entries, %d bytes, reached the file in %d writes, want at most %d",
			entries, file.kept.Len(), file.writes, most)
	}
	var document struct {
		Defects []json.RawMessage `json:"defects"`
	}
	if err := json.Unmarshal(file.kept.Bytes(), &document); err != nil {
		t.Fatalf("the buffered document does not parse: %v", err)
	}
	if len(document.Defects) != entries {
		t.Errorf("the buffered document carries %d of the %d entries", len(document.Defects), entries)
	}
}

// 🔴 A document whose last bytes cannot be written does not close.
//
// The buffer holds them until the close, and the set of files gives every file
// its real name right after it. A failure to send them that went unsaid would
// give a short document the name of a whole one.
func TestAManifestWhoseLastBytesCannotBeWrittenDoesNotClose(t *testing.T) {
	file := &takesUpTo{room: 64}
	writer := manifestjson.New(file, "0.2")
	if err := writer.Add(anEntry("one")); err != nil {
		t.Fatalf("an entry was refused before anything reached the file: %v", err)
	}
	err := writer.Finish(anEnvelope(), 1, 1)
	if err == nil {
		t.Fatalf("the manifest closed although %d bytes of it reached the file", file.kept.Len())
	}
	if kind := fault.KindOf(err); kind != fault.IO {
		t.Errorf("a file out of room was reported as %v rather than as a failure to write", kind)
	}
}

// 🔴 A document that cannot be written says so while entries still arrive,
// within one buffer of them, rather than when it closes, and it says so at the
// entry during which the disk refused rather than at the one after.
//
// It is the half of ending a run early that this package owns: the writer of
// records stops at the first entry that is refused, and an entry is refused
// only if the document says it was.
func TestAManifestThatCannotBeWrittenSaysSoWithinOneBuffer(t *testing.T) {
	file := &takesUpTo{}
	writer := manifestjson.New(file, "0.2")
	for at := 1; at <= 100; at++ {
		err := writer.Add(anEntry(fmt.Sprintf("entry-%d", at)))
		if file.tried == 0 {
			if err != nil {
				t.Fatalf("entry %d was refused before the disk was tried: %v", at, err)
			}
			continue
		}
		if err == nil {
			t.Fatalf("the disk refused during entry %d and the entry was taken", at)
		}
		if !errors.Is(err, errNoRoom) || fault.KindOf(err) != fault.IO {
			t.Errorf("a full disk was reported as %v", err)
		}
		if again := writer.Add(anEntry("after")); !errors.Is(again, errNoRoom) {
			t.Errorf("the entry after the failure was answered with %v", again)
		}
		return
	}
	t.Fatal("a hundred entries were taken by a document whose file takes nothing")
}

// A value the mutator produced on purpose survives, including an empty one. One
// catalog entry exists to replace a field with nothing, so empty and absent are
// different answers.
func TestAValueThatIsEmptyOnPurposeIsWrittenRatherThanDropped(t *testing.T) {
	var buffer bytes.Buffer
	writer := manifestjson.New(&buffer, "0.2")
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
	writer := manifestjson.New(&buffer, "0.2")
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
	writer := manifestjson.New(&buffer, "0.2")
	envelope := anEnvelope()
	envelope.Inference = &manifest.Inference{
		Method:  "values-validated",
		Columns: []manifest.InferredColumn{{Field: "pesel", Type: "identifier", Basis: "every value"}},
		Relations: []manifest.InferredRelation{
			{Field: "pesel", With: "sex", Kind: "gender", Basis: "every record agrees"},
		},
		NotInferred:  []string{"maximum length"},
		ColumnsTotal: 1,
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
