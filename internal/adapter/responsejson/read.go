// Package responsejson reads what the application under test did, as JSON.
//
// This is the only reader in this project pointed at a file the project did
// not write, so it is the only one where every rule about untrusted input
// applies at once: a bounded read, a refusal that names what was wrong, and a
// model that gets the chance to refuse before anything is compared.
package responsejson

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/adapter/boundedfile"
	"github.com/donislawdev/qadatagenerator/internal/domain/response"
	"github.com/donislawdev/qadatagenerator/internal/fault"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

// MaxSize is how large a response record this build reads.
//
// # Why there is a limit, and why it is this one
//
// The record is held as a value while the comparison runs, so its size is the
// memory the run costs. It also arrives from outside, which makes an unbounded
// read a way to end a run by handing it a large file.
//
// The number comes from a measurement rather than from a feeling: a record
// carrying a verdict, a code and a message for one row runs about 120 bytes,
// so this holds a run of roughly four hundred thousand rows.
//
// # What was measured about reading a row at a time, and why it is not done
//
// This comment used to say that beyond the limit the answer was to read the
// file a row at a time. That was written without a measurement and it is
// wrong. Measured on a 38.6 MB document, five runs each with the first
// discarded: reading it whole peaks at 46.6 MB, a json.Decoder walking the
// same file peaks at 136.6 MB, and walking it token by token to skip what is
// not wanted peaks at the same 136.6 MB. The decoder buffers the value it is
// working through, so streaming this shape of file costs three times what
// holding it costs. Raising the limit is therefore a question about memory,
// not a question waiting on a streaming reader.
const MaxSize = 48 * 1024 * 1024

// document is the shape on disk, kept apart from the model so that a change to
// one is a decision about the other rather than a side effect of it.
type document struct {
	SpecVersion string `json:"spec_version"`
	Dataset     *struct {
		RowsTotal *int `json:"rows_total"`
	} `json:"dataset"`
	Rows []struct {
		Row      *int              `json:"row"`
		Verdict  string            `json:"verdict"`
		Code     string            `json:"code"`
		Message  string            `json:"message"`
		Stored   map[string]string `json:"stored"`
		Warnings []string          `json:"warnings"`
	} `json:"rows"`
}

// Load reads a response record and refuses one that cannot be compared.
func Load(path string) (response.Record, error) {
	handle, err := os.Open(path)
	if err != nil {
		return response.Record{}, fault.Wrap(fault.IO, err, "cannot read the response record "+path)
	}
	defer func() { _ = handle.Close() }()
	return read(handle, path)
}

func read(from *os.File, path string) (response.Record, error) {
	raw, err := bytesOf(from, path)
	if err != nil {
		return response.Record{}, err
	}
	return decode(raw, path)
}

// bytesOf reads the record whole, within the limit this build reads at once.
//
// Split out of read for the same reason the manifest reader split its own: read
// says what a response record has to be, and this says how much of a file is
// taken in. Two subjects in one function is also what pushed it past the hard
// length limit the day each of its refusals gained a next move of its own.
func bytesOf(from *os.File, path string) ([]byte, error) {
	raw, err := boundedfile.Read(from, MaxSize)
	switch {
	case errors.Is(err, boundedfile.ErrTooLarge):
		// # Why the next move left the message
		//
		// It used to be the second sentence of the message, which put two next
		// moves in front of the reader: this one, and the one the command line
		// adds for a Data failure, telling them to change the file or the flags
		// describing it. The second contradicts the first, and no flag decides
		// how large a record somebody else's application writes. One move, in
		// the place the surfaces already look for it.
		return nil, fault.New(fault.Data, path+" is larger than "+
			strconv.Itoa(MaxSize/(1024*1024))+" MB, which is as much of a response record as this "+
			"build reads at once").
			DoInstead("split the run into smaller datasets, or report the rows that matter " +
				"rather than every row")
	case errors.Is(err, boundedfile.ErrChangedWhileRead):
		return nil, fault.New(fault.Data, path+" grew past "+
			strconv.Itoa(MaxSize/(1024*1024))+" MB while it was being read, so something else is "+
			"writing it").
			DoInstead("compare against a record the application has finished writing")
	case err != nil:
		return nil, fault.Wrap(fault.IO, err, "cannot read the response record "+path)
	}
	return raw, nil
}

// decode turns the bytes into a record this build can compare, or refuses them.
func decode(raw []byte, path string) (response.Record, error) {
	var read document
	// Read straight from the bytes. Turning them into a string and the string
	// back into a reader copied the whole record twice, which at the size this
	// build accepts is ninety six megabytes of work with no answer attached.
	decoder := json.NewDecoder(bytes.NewReader(raw))
	// An unknown member is refused rather than skipped. This file is written by
	// hand or by somebody's script, so a misspelled member is the likely
	// mistake, and silently ignoring it would mean comparing against evidence
	// the producer believes they supplied.
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&read); err != nil {
		return response.Record{}, fault.Wrap(fault.Config, err, path+
			" is not a response record this build can read. See spec/application-response.md "+
			"for the shape, and check the member names")
	}
	if err := trailing(decoder); err != nil {
		return response.Record{}, fault.Wrap(fault.Config, err, path+
			" carries more than one document, and a run is one record")
	}

	built := asRecord(read)
	if problems := built.Problems(); len(problems) > 0 {
		return response.Record{}, fault.New(fault.Config, path+" cannot be compared: "+
			strings.Join(problems, ". "))
	}
	if version.Compatibility(version.Response, built.SpecVersion) == version.Unreadable {
		return response.Record{}, fault.New(fault.Config, path+" is written to version "+
			fault.Shorten(built.SpecVersion)+" of the response format, and this build speaks "+
			version.Response+". A different major version is one where what a field means has "+
			"changed, so a comparison against it would be a verdict on your application drawn "+
			"from a document this build cannot read").
			DoInstead("write the record to a version this build speaks, or use a build that " +
				"speaks the one it is written to")
	}
	return built, nil
}

// trailing reports anything after the document, because a file holding two
// records is a file whose second half would go unread.
func trailing(decoder *json.Decoder) error {
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("there is more than one document in the file")
		}
		return err
	}
	return nil
}

func asRecord(read document) response.Record {
	built := response.Record{SpecVersion: read.SpecVersion}
	if read.Dataset != nil && read.Dataset.RowsTotal != nil {
		built.RowsTotal, built.HasRowsTotal = *read.Dataset.RowsTotal, true
	}
	for _, row := range read.Rows {
		made := response.Row{
			Verdict: response.Verdict(row.Verdict),
			Code:    row.Code,
			Message: row.Message,
			// 🔴 Present rather than non-empty. A producer that read the row
			// back and found it unchanged writes an empty object, and a
			// producer that read nothing writes no member at all. Those are
			// different answers and the whole value of the member is telling
			// them apart, which a nil check does and an emptiness check
			// does not.
			Stored:      row.Stored,
			HasStored:   row.Stored != nil,
			Warnings:    row.Warnings,
			HasWarnings: row.Warnings != nil,
		}
		if row.Row != nil {
			made.Number = *row.Row
		}
		built.Rows = append(built.Rows, made)
	}
	return built
}
