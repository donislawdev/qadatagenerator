package manifestjson

import (
	"encoding/json"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/fault"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

// MaxSize is how large a manifest this build reads back.
//
// The writer streams and has no limit, because it writes what a run produced.
// The reader holds the entries as a value while a comparison runs, and it is
// pointed at a path somebody typed, so it is bounded like every other read of
// something this process did not just create.
const MaxSize = 48 * 1024 * 1024

// Load reads a manifest written by an earlier run.
//
// # Why reading our own format back is allowed here and not in the exports
//
// The exports refuse to read this file and are fed the same entries instead,
// because a writer and a reader disagreeing inside one run would be a new
// class of defect in a report that must not lie. That argument is about one
// run. This is a later run, given a path by somebody who wants last week's
// manifest compared against what their application did today, and there is no
// second source of those entries to be fed from. The entries have to come off
// the disk, so what matters is that the reader refuses anything it cannot
// read rather than filling in a default.
func Load(path string) (manifest.Manifest, error) {
	handle, err := os.Open(path)
	if err != nil {
		return manifest.Manifest{}, fault.Wrap(fault.IO, err, "cannot read the manifest "+path)
	}
	defer func() { _ = handle.Close() }()

	raw, err := io.ReadAll(io.LimitReader(handle, MaxSize+1))
	if err != nil {
		return manifest.Manifest{}, fault.Wrap(fault.IO, err, "cannot read the manifest "+path)
	}
	if len(raw) > MaxSize {
		return manifest.Manifest{}, fault.New(fault.Data, path+" is larger than "+
			strconv.Itoa(MaxSize/(1024*1024))+" MB, which is as much of a manifest as this build "+
			"reads back at once")
	}

	var read manifestDocument
	if err := json.Unmarshal(raw, &read); err != nil {
		return manifest.Manifest{}, fault.Wrap(fault.Config, err, path+
			" is not a manifest this build can read. It has to be the JSON manifest a run wrote, "+
			"rather than one of the forms exported beside it")
	}
	if len(read.Defects) == 0 && read.SpecVersion == "" {
		return manifest.Manifest{}, fault.New(fault.Config, path+
			" carries neither a spec version nor a single entry, so it is not a manifest")
	}
	built, problems := asManifest(read)
	if len(problems) > 0 {
		return manifest.Manifest{}, fault.New(fault.Config, path+
			" is a manifest this build would not have written: "+strings.Join(problems, ". "))
	}
	if version.Compatibility(version.Manifest, built.SpecVersion()) == version.Unreadable {
		return manifest.Manifest{}, fault.New(fault.Config, path+" is written to version "+
			fault.Shorten(built.SpecVersion())+" of the manifest format, and this build speaks "+
			version.Manifest+". A different major version is one where what a field means has "+
			"changed, so reading it here would be guesswork. Use a build that speaks it, or "+
			"produce the manifest again with this one")
	}
	return built, nil
}

// manifestDocument is the reading side of the shape, and it is deliberately
// tolerant of members it does not know: a manifest written by a later build
// carries what this one cannot use, and refusing it would make every upgrade a
// reason to keep old files unreadable.
type manifestDocument struct {
	SpecVersion string           `json:"spec_version"`
	Defects     []defectDocument `json:"defects"`
	// The envelope is read whole rather than in the two members a comparison
	// happens to use. It goes back through the model, and the model refuses an
	// envelope that cannot say when it ran, from which seed and against which
	// catalog. Reading half of it would have meant our own manifests failing
	// their own rules on the way back in, which is how this was found.
	Run runDocument `json:"run"`
}

func asManifest(read manifestDocument) (manifest.Manifest, []string) {
	var defects []manifest.Defect
	for _, entry := range read.Defects {
		defect := manifest.Defect{
			ID:            entry.ID,
			Mutator:       catalog.ID(entry.Mutator),
			Layer:         catalog.Layer(entry.Layer),
			Applied:       entry.Applied,
			Expectation:   catalog.Expectation(entry.Expectation),
			Confidence:    catalog.Confidence(entry.Confidence),
			Rationale:     entry.Rationale,
			FailureReason: entry.FailureReason,
			FailureKind:   manifest.FailureKind(entry.FailureKind),
		}
		defect.Location = locationFrom(entry.Location)
		if entry.ValueBefore != nil {
			defect.ValueBefore, defect.HasValueBefore = *entry.ValueBefore, true
		}
		if entry.ValueAfter != nil {
			defect.ValueAfter, defect.HasValueAfter = *entry.ValueAfter, true
		}
		defects = append(defects, defect)
	}
	// 🔴 Built through the model rather than assembled as a value, so that a
	// manifest somebody edited by hand is refused by the same rules that
	// refused it at writing time. A comparison run against an entry the model
	// would not have written is a comparison against something this tool never
	// promised.
	return manifest.New(read.SpecVersion, manifest.Run{
		GeneratedAt:    read.Run.GeneratedAt,
		Seed:           read.Run.Seed,
		Locale:         read.Run.Locale,
		CatalogVersion: read.Run.CatalogVersion,
		Recipe:         read.Run.Recipe,
		Policy:         read.Run.Policy,
		Tag:            read.Run.Tag,
		Source:         read.Run.Source,
		RowsTotal:      read.Run.RowsTotal,
		RowsMutated:    read.Run.RowsMutated,
	}, defects)
}

// locationFrom turns a place on disk back into a place in the model, whole.
//
// # Why every member and not the three a row usually has
//
// It read the entity, the row and the field until 2026-08-17, which is what an
// entry damaging a value carries and what every test here happened to use. The
// layers name places differently, so the other three members are not decoration:
// a container defect may name only a path, set integrity damage names the other
// records it spans, and damage to the bytes names an offset. Dropping them cost
// two different failures, and only one of them was loud.
//
// Loud: an entry that came back without its path or without the records it
// spans is one the model refuses, so a manifest this tool wrote from its own
// full policy could not be opened by this tool at all, and verify said the input
// document was invalid while naming our own file.
//
// Quiet, and the worse of the two: damage to the bytes of a file legitimately
// names no place, so an offset dropped on the way in raises no complaint
// anywhere. The document opens, the run succeeds, and the manifest has simply
// stopped saying where in the file the damage was. That is this project's own
// ban on silence, broken by the reader.
func locationFrom(document locationDocument) manifest.Location {
	place := manifest.Location{
		Entity: document.Entity,
		Row:    document.Row,
		Field:  document.Field,
		Path:   document.Path,
	}
	if document.ByteOffset != nil {
		place.ByteOffset, place.HasByteOffset = *document.ByteOffset, true
	}
	for _, related := range document.Related {
		place.Related = append(place.Related, locationFrom(related))
	}
	return place
}
