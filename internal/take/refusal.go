package take

import (
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestexport"
	"github.com/donislawdev/qadatagenerator/internal/adapter/mutators"
	"github.com/donislawdev/qadatagenerator/internal/app/corrupt"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
)

// RefusalKind is a request this mode will not carry out, named rather than
// worded.
//
// # Why a kind and not the sentence itself
//
// The sentence cannot live here. The command line is English always, by an
// unbreakable rule, while the window shows whatever it was translated into and
// takes its words from a catalogue of keys. A shared sentence would force one
// of those two to be wrong, and the one it would break is the window, quietly,
// by putting an untranslated string on a translated screen.
//
// What both surfaces do share is which requests are refused and why, and that
// is exactly what a kind carries. Each surface then owns its own four parts:
// what happened, why, what would have been accepted, and what to do instead.
type RefusalKind int

const (
	// NotRefused is the zero value, so that a refusal nobody filled in cannot
	// be mistaken for a real one.
	NotRefused RefusalKind = iota
	// NoLayers is a run that never said what kind of damage it wants.
	NoLayers
	// UnknownLayer names a layer this build describes no damage in.
	UnknownLayer
	// UnreadableDelimiter is a field separator the format cannot be split on:
	// not one character, or a quote, a line end or a zero byte.
	UnreadableDelimiter
	// UnknownManifestForm asks for an export this build does not write.
	UnknownManifestForm
	// UnreadableShare is a share of rows that cannot be read as one.
	UnreadableShare
	// UnknownPerRow is a setting for how many defects one row carries that this
	// build has no meaning for.
	UnknownPerRow
	// WouldOverwriteSource is a run that would write over the file it reads.
	WouldOverwriteSource
)

// Refusal is why a run will not start, with what it needs to be said properly.
//
// Value and Allowed are here because every refusal in this project has to name
// the offending value and what would have been taken instead. Leaving those to
// the surface would mean each one working them out again, and the catalog is
// the only thing that knows some of them.
type Refusal struct {
	// Kind is what was wrong.
	Kind RefusalKind
	// Value is the offending value, empty where the fault is an absence.
	Value string
	// Allowed is what this build would have accepted, where that is a list it
	// can produce. It is empty where the answer is a sentence rather than a
	// set, and the surface supplies the sentence.
	Allowed []string
}

// Kinds is every refusal this mode can produce.
//
// 🔴 It exists so that a surface can be held to answering for all of them. A
// surface turns a kind into words with a switch, and a switch has a last
// branch: a kind added here and forgotten there would fall into that branch and
// be described as some other refusal entirely, in a complete and confident four
// part message about the wrong thing. That is a failure no compiler reports and
// no reader of the switch notices.
func Kinds() []RefusalKind {
	return []RefusalKind{
		NoLayers, UnknownLayer, UnreadableDelimiter, UnknownManifestForm,
		UnreadableShare, UnknownPerRow, WouldOverwriteSource,
	}
}

// Name is what this kind is called, for a guard that has to say which one it
// is unhappy about.
func (k RefusalKind) Name() string {
	switch k {
	case NoLayers:
		return "NoLayers"
	case UnknownLayer:
		return "UnknownLayer"
	case UnreadableDelimiter:
		return "UnreadableDelimiter"
	case UnknownManifestForm:
		return "UnknownManifestForm"
	case UnreadableShare:
		return "UnreadableShare"
	case UnknownPerRow:
		return "UnknownPerRow"
	case WouldOverwriteSource:
		return "WouldOverwriteSource"
	default:
		return "NotRefused"
	}
}

// Refused reports whether there is anything here to show.
func (r Refusal) Refused() bool { return r.Kind != NotRefused }

// Check holds the settings of a run against this build, and returns the first
// thing that makes the run impossible.
//
// 🔴 Everything here is asked before the file is opened, and that is the whole
// reason this is one function called early rather than checks scattered where
// each value is used: the file is somebody else's and they chose its size, so
// a pass over it must not be the price of a typo.
func Check(settings Settings, described catalog.Catalog) Refusal {
	if refusal := checkLayers(settings, described); refusal.Refused() {
		return refusal
	}
	return checkValues(settings)
}

// checkLayers refuses a run that named no layer, or named one this build
// carries no damage in.
//
// It is separate from the values below because it asks a different kind of
// question. A layer is answered against the catalog, and the answer changes as
// the catalog grows. The values below are answered against this build's own
// vocabulary and do not.
func checkLayers(settings Settings, described catalog.Catalog) Refusal {
	// The same answer validate gives about a recipe, from the same place. Two
	// lists of which layers carry damage would tell somebody their policy is
	// fine while the verb that runs it did nothing.
	carried := mutators.Coverage{Described: described}.Layers()
	if len(settings.Layers) == 0 {
		return Refusal{Kind: NoLayers, Allowed: carried}
	}
	carries := map[string]bool{}
	for _, layer := range carried {
		carries[layer] = true
	}
	for _, layer := range settings.Layers {
		if !carries[layer] {
			return Refusal{Kind: UnknownLayer, Value: layer, Allowed: carried}
		}
	}
	return Refusal{}
}

// checkValues refuses a value this build cannot read, one setting at a time.
func checkValues(settings Settings) Refusal {
	if _, usable := dataset.UsableSeparator(settings.Delimiter); !usable {
		return Refusal{Kind: UnreadableDelimiter, Value: settings.Delimiter}
	}
	if !manifestexport.Usable(settings.Manifest) {
		return Refusal{Kind: UnknownManifestForm, Value: settings.Manifest,
			Allowed: manifestexport.Forms()}
	}
	if err := corrupt.UsableShare(settings.Rows); err != nil {
		return Refusal{Kind: UnreadableShare, Value: settings.Rows}
	}
	if err := corrupt.UsablePerRow(settings.PerRow); err != nil {
		return Refusal{Kind: UnknownPerRow, Value: settings.PerRow,
			Allowed: corrupt.PerRowChoices()}
	}
	return Refusal{}
}

// The lists a surface builds its controls from.
//
// # Why these are here and not read off the adapters directly
//
// A screen offering these as tick boxes and lists needs the same three sets the
// checks above answer against. Reading them straight from the adapters would
// work and would put a second opinion about them in the window: the day a list
// here grows, a control built from another list stops offering what this build
// accepts, and nothing says so. One place answers both questions instead.
//
// It also keeps the window from importing three adapters to draw one screen.

// LayersIn is every kind of damage this build describes, in the order a person
// reads them.
func LayersIn(described catalog.Catalog) []string {
	return mutators.Coverage{Described: described}.Layers()
}

// ManifestForms is every extra form of the manifest this build writes.
func ManifestForms() []string { return manifestexport.Forms() }

// PerRowChoices is every setting for how many defects one record may carry.
func PerRowChoices() []string { return corrupt.PerRowChoices() }

// NamesIn splits a comma separated list, dropping what is only whitespace.
//
// It is here rather than at each surface because both of them take these lists
// from a person: the command line from a flag and the window from a field, and
// a second reading of the same syntax is a second set of rules about what a
// trailing comma means.
func NamesIn(written string) []string {
	var names []string
	for _, part := range strings.Split(written, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			names = append(names, trimmed)
		}
	}
	return names
}
