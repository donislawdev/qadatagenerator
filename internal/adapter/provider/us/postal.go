package us

import (
	"fmt"
	"sort"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/corpus"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// ZIP is the United States postal code together with the place it belongs to.
//
// # What the reference data actually is, stated before anything is built on it
//
// The registry was built from two files of the United States census, a
// relationship file pairing ZIP Code Tabulation Areas with places and a
// gazetteer giving those places their names and states.
//
// 🔴 A tabulation area is not a ZIP Code and this file will not pretend
// otherwise. The census calls them "generalized areal representations of the
// geographic extent and distribution of the point-based ZIP Codes" and states
// that "Not all valid ZIP Codes are represented by a 2020 ZCTA". ZIP Code is a
// trademark of the postal service, which publishes no list this project may
// take. So the codes here are a statistical approximation of the postal
// scheme, close enough to make a coherent record and not a postal register.
//
// The consequence is in Validate: this provider checks the shape of a code and
// deliberately does not rule that a code absent from the registry is wrong.
// A real ZIP Code can be absent, so refusing it would report a defect in data
// that has none, which is the failure this project has already avoided
// carefully once and would be reintroducing here for the sake of a stricter
// looking check.
//
// # Which place a code belongs to is our arithmetic, not the source's
//
// The relationship file lists every intersection, and one area can cross
// seventy-seven places. The census does not say which of them is the place of
// the code, because it is not answering that question. The extraction picks the
// one sharing the largest land area, breaking a tie on the lower identifier so
// that the answer cannot depend on the order of rows. That choice is ours and
// the file header says so, so that nobody later quotes it as a federal fact.
//
// # Ethics, and this scheme is the easy case for once
//
// Every code here is real and belongs to the place beside it, which is the whole
// point: the capability exists so pillar one can put a coherent pair in a record
// and pillar two can break it on purpose. Unlike a personal number a postal code
// identifies an area rather than a person, so there is no harbour to generate
// inside and nothing here can be traced to anybody.
type Postal struct{}

// Contract checks, so that a mismatch between what this provider claims and
// what it implements fails to compile rather than at run time.
var (
	_ provider.Provider     = Postal{}
	_ provider.PostalLinked = Postal{}
)

const (
	// postalCodeLength is the digits of the code itself, and postalPlusFourLength is the
	// longer written form, five digits, a hyphen and four more.
	postalCodeLength     = 5
	postalPlusFourLength = postalCodeLength + 1 + 4
	// postalFields is how many fields a registry line carries: code, place,
	// region.
	postalFields = 3
)

const (
	postalLocale = "en-US"
	postalList   = "postal-places"
)

// postalEntry is one line of the registry, already split.
type postalEntry struct {
	code  string
	place provider.Place
}

// postalRegistry is the whole registry, in the order the file holds it, which
// is sorted by code.
//
// It is built once when the package is loaded rather than on first use. A lazy
// build would need a guard the day this tool grows a second goroutine, and a
// lazy build without one is a race that appears years later and looks like
// corrupted data rather than like a race. The concurrency rule of this project
// keeps that list empty on purpose, so the cheap way to honour it is to do the
// work before anything can be concurrent at all.
var postalRegistry = loadPostalRegistry()

// postalByCode indexes the registry for lookup by code.
var postalByCode = indexPostalByCode(postalRegistry)

func loadPostalRegistry() []postalEntry {
	lines, ok := corpus.Load(postalLocale, postalList)
	if !ok {
		// The list is embedded in this binary, so it is either there or the
		// build is broken. Carrying on with an empty registry would answer
		// every lookup with "no such code", which reads exactly like a
		// legitimate answer and would be wrong every time.
		panic("us: the postal registry is missing from the binary")
	}
	entries := make([]postalEntry, 0, len(lines))
	for _, line := range lines {
		parts := strings.Split(line, "|")
		if len(parts) != postalFields {
			panic("us: the postal registry has a line that is not code, place and region: " + line)
		}
		entries = append(entries, postalEntry{
			code:  parts[0],
			place: provider.Place{Name: parts[1], Region: parts[2]},
		})
	}
	return entries
}

func indexPostalByCode(entries []postalEntry) map[string]provider.Place {
	byCode := make(map[string]provider.Place, len(entries))
	for _, entry := range entries {
		byCode[entry.code] = entry.place
	}
	return byCode
}

// ID names this provider. The country prefix is ISO 3166-1 alpha-2, because a
// provider belongs to a country rather than to a language.
func (Postal) ID() string { return "US.postal" }

// Capabilities are what this provider brings.
//
// One. It carries no check digit, encodes neither a date nor a gender, and has
// no block that is never assigned, so declaring anything else would be claiming
// a question this scheme cannot answer.
func (Postal) Capabilities() []capability.Name {
	return []capability.Name{capability.ReferencePostalToPlace}
}

// Generate produces one postal code for the place described by the request.
//
// The agreement is not read. Binding a postal code to a birth date or a gender
// is meaningless, and the binding that would matter, a code agreeing with a
// place already in the record, is the other direction: the caller draws the code
// and asks PlaceFor for the place that goes with it. Doing it the other way
// would need the place to be drawn first by somebody who does not have the
// registry.
func (p Postal) Generate(req provider.Request) (string, error) {
	index := req.Source.IntN(seed.Coordinate{
		Entity:  req.At.Entity,
		Row:     req.At.Row,
		Field:   req.At.Field,
		Purpose: req.At.Purpose + ":postal-code",
	}, int64(len(postalRegistry)))
	return postalRegistry[index].code, nil
}

// Validate answers whether a value is a correctly written postal code.
//
// It answers about shape and nothing else, and the omission is deliberate rather
// than lazy. See the type comment: the registry approximates the postal scheme,
// so a code it does not carry may still be a real one, and calling that invalid
// would fault data that is correct.
func (Postal) Validate(value string) provider.Verdict {
	if _, ok := parsePostalCode(value); !ok {
		return provider.Invalid("a postal code is five digits, written 12345 or 12345-6789")
	}
	return provider.Valid()
}

// PlaceFor returns the place a code belongs to.
//
// The longer written form is accepted and only its first five digits are looked
// up, because the four that follow narrow a delivery route within the same
// place and change nothing about which place it is.
func (Postal) PlaceFor(code string) (provider.Place, bool) {
	parsed, ok := parsePostalCode(code)
	if !ok {
		return provider.Place{}, false
	}
	place, found := postalByCode[parsed]
	return place, found
}

// CodesFor returns the codes belonging to a place, sorted.
//
// Sorted because the answer reaches a manifest and a report, and Go randomises
// map iteration on purpose. The registry is already held in code order, so this
// walk produces that order without sorting again.
func (Postal) CodesFor(place provider.Place) []string {
	var codes []string
	for _, entry := range postalRegistry {
		if entry.place == place {
			codes = append(codes, entry.code)
		}
	}
	return codes
}

// WithDifferentPlace returns a real code belonging to some other place.
//
// It walks forward from the code it was given and wraps around, returning the
// first entry whose place differs. Walking rather than drawing keeps it a pure
// function of its input, which is what lets one seed reproduce a dataset byte
// for byte, and wrapping is what stops the last code in the registry from being
// the one case that fails.
//
// It refuses a code the registry does not carry. That refusal is the
// applicability answer, and the caller has to record it in the manifest rather
// than pass the value through unchanged and silently.
func (p Postal) WithDifferentPlace(code string) (string, error) {
	parsed, ok := parsePostalCode(code)
	if !ok {
		return "", fmt.Errorf("postal: the value is not a postal code")
	}
	start := sort.Search(len(postalRegistry), func(i int) bool {
		return postalRegistry[i].code >= parsed
	})
	if start == len(postalRegistry) || postalRegistry[start].code != parsed {
		return "", fmt.Errorf("postal: the postal registry does not carry this code")
	}

	here := postalRegistry[start].place
	for step := 1; step < len(postalRegistry); step++ {
		candidate := postalRegistry[(start+step)%len(postalRegistry)]
		if candidate.place != here {
			return candidate.code, nil
		}
	}
	// Only reachable if every code in the registry shares one place, which
	// would mean the extraction produced something nobody intended.
	return "", fmt.Errorf("postal: the postal registry holds only one place")
}

// parsePostalCode accepts the two written forms and returns the five digits.
//
// The value is not trimmed and not case folded. A code arriving with a space
// around it is a defect in whatever produced it, and quietly accepting it here
// would hide that from the very report this tool exists to produce.
func parsePostalCode(value string) (string, bool) {
	switch len(value) {
	case postalCodeLength:
	case postalPlusFourLength:
		if value[postalCodeLength] != '-' || !allDigits(value[postalCodeLength+1:]) {
			return "", false
		}
	default:
		return "", false
	}
	head := value[:postalCodeLength]
	if !allDigits(head) {
		return "", false
	}
	return head, true
}

func allDigits(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return len(value) > 0
}
