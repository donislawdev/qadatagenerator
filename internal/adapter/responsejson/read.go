// Package responsejson reads what the application under test did, as JSON.
//
// This is the only reader in this project pointed at a file the project did
// not write, so it is the only one where every rule about untrusted input
// applies at once: a bounded read, a refusal that names what was wrong, and a
// model that gets the chance to refuse before anything is compared.
package responsejson

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/domain/response"
	"github.com/donislawdev/qadatagenerator/internal/fault"
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
// so this holds a run of roughly four hundred thousand rows. Beyond that the
// answer is not a bigger number, it is reading the file a row at a time, and
// that is recorded as B-145 with the measurement that would call for it.
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

func read(from io.Reader, path string) (response.Record, error) {
	// One byte past the limit, so that a file exactly at it is accepted and a
	// file over it is known to be over rather than guessed at.
	raw, err := io.ReadAll(io.LimitReader(from, MaxSize+1))
	if err != nil {
		return response.Record{}, fault.Wrap(fault.IO, err, "cannot read the response record "+path)
	}
	if len(raw) > MaxSize {
		return response.Record{}, fault.New(fault.Data, path+" is larger than "+
			strconv.Itoa(MaxSize/(1024*1024))+" MB, which is as much of a response record as this "+
			"build reads at once. Split the run into smaller datasets, or report the rows that "+
			"matter rather than every row")
	}

	var read document
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	// An unknown member is refused rather than skipped. This file is written by
	// hand or by somebody's script, so a misspelled member is the likely
	// mistake, and silently ignoring it would mean comparing against evidence
	// the producer believes they supplied.
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&read); err != nil {
		return response.Record{}, fault.Wrap(fault.Data, err, path+
			" is not a response record this build can read. See spec/application-response.md "+
			"for the shape, and check the member names")
	}
	if err := trailing(decoder); err != nil {
		return response.Record{}, fault.Wrap(fault.Data, err, path+
			" carries more than one document, and a run is one record")
	}

	built := asRecord(read)
	if problems := built.Problems(); len(problems) > 0 {
		return response.Record{}, fault.New(fault.Data, path+" cannot be compared: "+
			strings.Join(problems, ". "))
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
