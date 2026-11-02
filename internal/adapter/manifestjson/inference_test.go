package manifestjson_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
)

// inferred is an envelope of a run that read somebody else's file, with every
// part of an inference filled in: a column with a scheme and one without, a
// relation, and a file wider than the columns described.
func inferred() manifest.Run {
	envelope := anEnvelope()
	envelope.Source = "customers.csv"
	envelope.Inference = &manifest.Inference{
		Method: "values-validated",
		Columns: []manifest.InferredColumn{
			{Field: "pesel", Type: "identifier", Scheme: "PL.pesel", Basis: "every value passes the scheme"},
			{Field: "sex", Type: "gender", Basis: "two spellings of a gender"},
		},
		Relations: []manifest.InferredRelation{
			{Field: "pesel", With: "sex", Kind: "gender", Basis: "every record agrees"},
		},
		NotInferred:  []string{"maximum length", "required"},
		ColumnsTotal: 3,
	}
	return envelope
}

// onDisk writes a manifest of one entry under that envelope and returns where.
func onDisk(t *testing.T, envelope manifest.Run) string {
	t.Helper()
	var buffer bytes.Buffer
	writer := manifestjson.New(&buffer, "0.2")
	if err := writer.Add(anEntry("one")); err != nil {
		t.Fatalf("the entry was refused: %v", err)
	}
	if err := writer.Finish(envelope, 1, 1); err != nil {
		t.Fatalf("the manifest did not close: %v", err)
	}
	path := filepath.Join(t.TempDir(), "expectation-manifest.json")
	if err := os.WriteFile(path, buffer.Bytes(), 0o600); err != nil {
		t.Fatalf("the manifest could not be put on a disk: %v", err)
	}
	return path
}

// 🔴 The inference a run wrote is the one read back.
//
// Until 2026-11-02 the reader parsed schema_inference and built the envelope
// without it, so every manifest from the mode that reads somebody else's file
// came back as if nothing had been inferred. Each half was correct on its own,
// which is the shape the test of relations beside this one was written against.
func TestAnInferenceSurvivesBeingWrittenAndRead(t *testing.T) {
	envelope := inferred()

	read, err := manifestjson.Load(onDisk(t, envelope))
	if err != nil {
		t.Fatalf("a manifest this build wrote was refused: %v", err)
	}
	if got := read.Run().Inference; !reflect.DeepEqual(got, envelope.Inference) {
		t.Errorf("the run inferred\n%+v\nand the manifest read back says\n%+v", envelope.Inference, got)
	}
}

// 🔴 An inference somebody edited is held to the rules it was written under.
//
// A conclusion about somebody else's column with its basis taken out is one
// nobody can argue with, and the writer refuses to write one. Reading it back
// used to accept it, because the inference never reached the model.
func TestAnInferenceEditedToLoseItsBasisIsRefusedOnReading(t *testing.T) {
	path := onDisk(t, inferred())
	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	edited := bytes.Replace(text, []byte(`"two spellings of a gender"`), []byte(`""`), 1)
	if bytes.Equal(edited, text) {
		t.Fatalf("the basis to take out is not in the document:\n%s", text)
	}
	if err := os.WriteFile(path, edited, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err = manifestjson.Load(path)
	if err == nil {
		t.Fatal("an inference about sex with no basis was read as a valid manifest")
	}
	if !strings.Contains(err.Error(), "the inference about sex carries no basis") {
		t.Errorf("the refusal does not say which conclusion lost its basis: %v", err)
	}
}
