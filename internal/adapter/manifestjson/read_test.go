package manifestjson_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
)

// written puts entries through the writer and reads them back the way any other
// program would, off a file.
//
// # Why the pair is exercised rather than each half alone
//
// Both halves were checked before this file existed and both were green. The
// writer was held to producing the shape the specification describes, and the
// reader was held to refusing documents it should refuse. Nothing asked whether
// what one wrote was what the other read, and that is where the whole cost sat:
// a manifest this tool produced from its own recipe could not be opened by this
// tool, and the run said the document was invalid, naming our own file.
func written(t *testing.T, defects ...manifest.Defect) (manifest.Manifest, error) {
	t.Helper()
	var buffer bytes.Buffer
	writer := manifestjson.New(&buffer, "0.1")
	applied := 0
	for _, defect := range defects {
		writer.Add(defect)
		if defect.Applied {
			applied++
		}
	}
	run := manifest.Run{
		GeneratedAt: "2026-08-17T00:00:00Z", Seed: "s", Locale: "pl-PL",
		CatalogVersion: "0.1", RowsTotal: 60, RowsMutated: applied,
	}
	if err := writer.Finish(run, len(defects), applied); err != nil {
		t.Fatalf("the manifest could not be written: %v", err)
	}

	path := filepath.Join(t.TempDir(), "expectation-manifest.json")
	if err := os.WriteFile(path, buffer.Bytes(), 0o600); err != nil {
		t.Fatalf("the manifest could not be put on a disk: %v", err)
	}
	return manifestjson.Load(path)
}

// A defect spanning more than one record survives being written and read.
//
// 🔴 This is the case that was found by running the tool rather than by any
// test. Set integrity damage names the other records it involves, the model
// refuses an entry at that layer without them, and an entry that lost them on
// the way in made the whole document unreadable. So a manifest produced from
// this project's own full policy could not be opened by this project's own
// verify, which returned the code meaning the input document is invalid.
func TestADefectSpanningSeveralRecordsSurvivesBeingWrittenAndRead(t *testing.T) {
	spanning := manifest.Defect{
		ID: "customer-38-L5.pk.duplicate",
		Location: manifest.Location{
			Entity: "customer", Row: 38, Field: "pesel",
			Related: []manifest.Location{{Entity: "customer", Row: 39}},
		},
		Mutator: "L5.pk.duplicate", Layer: catalog.SetLevel,
		Expectation: catalog.Rejection, Confidence: catalog.Hard,
		Rationale:     "two records carry one primary key",
		FailureReason: "this build carries no implementation of it",
		FailureKind:   manifest.Unimplemented,
	}

	read, err := written(t, spanning)
	if err != nil {
		t.Fatalf("a manifest this tool wrote could not be read back: %v", err)
	}
	entries := read.Defects()
	if len(entries) != 1 {
		t.Fatalf("one entry was written and %d came back", len(entries))
	}
	related := entries[0].Location.Related
	if len(related) != 1 {
		t.Fatalf("the entry named one other record and %d came back", len(related))
	}
	if related[0].Row != 39 || related[0].Entity != "customer" {
		t.Errorf("the other record came back as %+v", related[0])
	}
}

// A place inside a nested structure survives, and its loss is not quiet.
//
// A container defect names a row or a path, so an entry carrying only a path
// becomes unreadable rather than merely poorer. The reader refusing the whole
// document is the loud half. The quiet half is below.
func TestAPlaceInsideANestedStructureSurvivesBeingWrittenAndRead(t *testing.T) {
	nested := manifest.Defect{
		ID:       "order-1-L1.json.truncated",
		Location: manifest.Location{Path: "$.orders[3].lines[0]"},
		Mutator:  "L1.json.truncated", Layer: catalog.Container,
		Applied:     true,
		Expectation: catalog.Rejection, Confidence: catalog.Hard,
		Rationale: "the document ends in the middle of a record",
	}

	read, err := written(t, nested)
	if err != nil {
		t.Fatalf("a manifest this tool wrote could not be read back: %v", err)
	}
	if got := read.Defects()[0].Location.Path; got != "$.orders[3].lines[0]" {
		t.Errorf("the path came back as %q", got)
	}
}

// A byte offset survives, and this is the one that would be lost in silence.
//
// 🔴 Damage to the bytes of a file names no place at all in the ordinary case,
// so the model accepts an entry at that layer with an empty location. An offset
// dropped on the way in therefore raises no complaint anywhere: the document
// opens, the run succeeds, and the manifest simply stops saying where in the
// file the damage was. That is the ban on silence broken by the reader rather
// than by the writer.
func TestAByteOffsetSurvivesBeingWrittenAndRead(t *testing.T) {
	inTheBytes := manifest.Defect{
		ID: "file-L0.encoding.mixed",
		Location: manifest.Location{
			ByteOffset: 4096, HasByteOffset: true,
		},
		Mutator: "L0.encoding.mixed", Layer: catalog.Bytes, Applied: true,
		Expectation: catalog.Rejection, Confidence: catalog.Hard,
		Rationale: "the file changes encoding part way through",
	}

	read, err := written(t, inTheBytes)
	if err != nil {
		t.Fatalf("a manifest this tool wrote could not be read back: %v", err)
	}
	place := read.Defects()[0].Location
	if !place.HasByteOffset {
		t.Fatal("the entry said where in the file the damage was, and what came back says nothing")
	}
	if place.ByteOffset != 4096 {
		t.Errorf("the offset came back as %d", place.ByteOffset)
	}
}

// The first byte of a file is a real place, so an offset of nought is not the
// same as no offset at all.
func TestAnOffsetOfNoughtIsNotTheSameAsNoOffset(t *testing.T) {
	atTheStart := manifest.Defect{
		ID:       "file-L0.encoding.bom_added",
		Location: manifest.Location{ByteOffset: 0, HasByteOffset: true},
		Mutator:  "L0.encoding.bom_added", Layer: catalog.Bytes, Applied: true,
		Expectation: catalog.Rejection, Confidence: catalog.Hard,
		Rationale: "a byte order mark was put in front of the document",
	}

	read, err := written(t, atTheStart)
	if err != nil {
		t.Fatalf("a manifest this tool wrote could not be read back: %v", err)
	}
	if !read.Defects()[0].Location.HasByteOffset {
		t.Error("damage at the first byte came back as damage with no place, and the first byte " +
			"of a file is a real place")
	}
}
