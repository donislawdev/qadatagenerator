package screens

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// sampleRows is how many records the sample file holds.
//
// Enough that a conclusion about a column rests on more than a coincidence, and
// few enough that the whole screen fits in a picture somebody looks at. The
// number is not a threshold anything depends on.
const sampleRows = 12

// SampleFile writes a file this tool did not make, into the directory given,
// and returns its path.
//
// # Why it is written rather than carried as a string
//
// The screen it feeds reads a file. Handing it records from memory would build
// a state by a road no person travels, and the point of this list is that every
// state in it is one somebody can reach.
//
// # Why the identifiers are produced rather than typed out
//
// A correct identifier written into a file here is a correct identifier nobody
// checked, and everything this screen says about a column turns on whether a
// validator recognised it. Produced by the same providers the product uses, they
// are correct for the same reason the product is.
func SampleFile(dir string) (string, error) {
	source := values.New("pl-PL", "screens-sample")
	lines := []string{"id,first_name,pesel,aba,note"}
	for row := 0; row < sampleRows; row++ {
		fields, err := sampleRow(source, row)
		if err != nil {
			return "", err
		}
		lines = append(lines, strings.Join(fields, ","))
	}
	path := filepath.Join(dir, "contacts.csv")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// sampleRow is one record, carrying a column of each kind the screen draws
// differently: one recognised as a numbering scheme, one recognised as ordinary
// text, and one that is neither.
func sampleRow(source values.Source, row int) ([]string, error) {
	pesel, err := produced(source, values.Pesel, row)
	if err != nil {
		return nil, err
	}
	name, err := produced(source, values.FirstName, row)
	if err != nil {
		return nil, err
	}
	aba, err := produced(source, values.ABA, row)
	if err != nil {
		return nil, err
	}
	return []string{pesel[:6], name, pesel, aba, "a note"}, nil
}

func produced(source values.Source, generator string, row int) (string, error) {
	return source.Produce(generate.Request{
		Generator: generator,
		At: seed.Coordinate{
			Entity: "contacts", Row: int64(row), Field: generator, Purpose: "value",
		},
	})
}

// The two recipes the editor is photographed with.
//
// # Why they are written out rather than produced
//
// The file the take and corrupt screen reads is produced, because everything
// that screen says about a column turns on whether a validator recognised it,
// and a correct identifier typed in here would be one nobody checked. Nothing
// of the sort is true of these: what this screen says about a recipe turns on
// what the READER says about it, and the reader is run for real. A recipe built
// by some generator here would only move the question of whether these are the
// recipes they claim to be.
//
// What holds that claim instead is a guard beside them, which runs the same
// check the screen runs and fails if the clean one stops being clean or the
// broken one stops being broken. Both were also put through qadg validate on
// 2026-09-02, which is how the schema version below is 0.1 rather than the 1
// this comment first carried.
const (
	// CleanRecipe is a recipe this build can use.
	CleanRecipe = `version: "0.1"
seed: "sample"
entities:
  customer:
    count: 3
    fields:
      name:
        generator: first_name
`

	// BrokenRecipe names a generator this build does not have, which is a
	// problem that carries a place AND a line AND a list of what would have
	// been accepted. That makes it the one broken recipe that exercises every
	// part of a drawn problem rather than only the first.
	BrokenRecipe = `version: "0.1"
seed: "sample"
entities:
  customer:
    count: 3
    fields:
      email:
        generator: person.email
`
)
