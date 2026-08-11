package manifestjson

import "github.com/donislawdev/qadatagenerator/internal/domain/manifest"

// The shapes below are the published document, and the JSON names are the
// contract. They are written out here rather than put on the domain model,
// because the model answers what a manifest is and a tag answering how it is
// spelled on disk would tie the two together for good.
//
// The format grows by adding fields. An existing name never changes meaning,
// and a reader ignores what it does not know.

type defectDocument struct {
	ID            string           `json:"id"`
	Location      locationDocument `json:"location"`
	Mutator       string           `json:"mutator"`
	Layer         string           `json:"layer"`
	Applied       bool             `json:"applied"`
	ValueBefore   *string          `json:"value_before,omitempty"`
	ValueAfter    *string          `json:"value_after,omitempty"`
	Expectation   string           `json:"expectation"`
	Confidence    string           `json:"confidence"`
	Rationale     string           `json:"rationale"`
	FailureReason string           `json:"failure_reason,omitempty"`
	FailureKind   string           `json:"failure_kind,omitempty"`
	Profile       *profileDocument `json:"profile,omitempty"`
}

type locationDocument struct {
	Entity     string             `json:"entity,omitempty"`
	Row        int                `json:"row,omitempty"`
	Field      string             `json:"field,omitempty"`
	Path       string             `json:"path,omitempty"`
	ByteOffset *int               `json:"byte_offset,omitempty"`
	Related    []locationDocument `json:"related,omitempty"`
}

type profileDocument struct {
	Code       string `json:"code,omitempty"`
	HTTPStatus int    `json:"http_status,omitempty"`
	Field      string `json:"field,omitempty"`
}

type runDocument struct {
	GeneratedAt      string `json:"generated_at"`
	Seed             string `json:"seed"`
	Locale           string `json:"locale"`
	CatalogVersion   string `json:"catalog_version"`
	Recipe           string `json:"recipe,omitempty"`
	Policy           string `json:"policy,omitempty"`
	Tag              string `json:"tag,omitempty"`
	RowsTotal        int    `json:"rows_total"`
	RowsMutated      int    `json:"rows_mutated"`
	DefectsRequested int    `json:"defects_requested"`
	DefectsApplied   int    `json:"defects_applied"`
}

// defectOf turns one entry into the document shape.
//
// The two values use pointers so that a value the mutator produced on purpose,
// such as an empty string, is written as an empty string rather than
// disappearing. One entry of the catalog exists to replace a field with nothing,
// so the difference between empty and absent is a defect this format has to
// carry.
func defectOf(defect manifest.Defect) defectDocument {
	document := defectDocument{
		ID:            defect.ID,
		Location:      locationOf(defect.Location),
		Mutator:       string(defect.Mutator),
		Layer:         string(defect.Layer),
		Applied:       defect.Applied,
		Expectation:   string(defect.Expectation),
		Confidence:    string(defect.Confidence),
		Rationale:     defect.Rationale,
		FailureReason: defect.FailureReason,
		FailureKind:   string(defect.FailureKind),
	}
	if defect.HasValueBefore {
		before := defect.ValueBefore
		document.ValueBefore = &before
	}
	if defect.HasValueAfter {
		after := defect.ValueAfter
		document.ValueAfter = &after
	}
	if defect.Profile != nil {
		document.Profile = &profileDocument{
			Code: defect.Profile.Code, HTTPStatus: defect.Profile.HTTPStatus,
			Field: defect.Profile.Field,
		}
	}
	return document
}

func locationOf(location manifest.Location) locationDocument {
	document := locationDocument{
		Entity: location.Entity, Row: location.Row,
		Field: location.Field, Path: location.Path,
	}
	if location.HasByteOffset {
		// Nought is a real place in a file, so the offset is carried by pointer
		// rather than left to the rule that omits empty numbers.
		offset := location.ByteOffset
		document.ByteOffset = &offset
	}
	for _, related := range location.Related {
		document.Related = append(document.Related, locationOf(related))
	}
	return document
}

func runOf(run manifest.Run, requested, applied int) runDocument {
	return runDocument{
		GeneratedAt: run.GeneratedAt, Seed: run.Seed, Locale: run.Locale,
		CatalogVersion: run.CatalogVersion, Recipe: run.Recipe,
		Policy: run.Policy, Tag: run.Tag,
		RowsTotal: run.RowsTotal, RowsMutated: run.RowsMutated,
		DefectsRequested: requested, DefectsApplied: applied,
	}
}
