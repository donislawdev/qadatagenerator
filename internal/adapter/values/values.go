// Package values is where a generated value actually comes from.
//
// It is the adapter behind the port the generate use case takes, and it is the
// only place that knows a name in a recipe means a word list here, an
// identifier provider there and a piece of arithmetic somewhere else. The use
// case has no business knowing which, and keeping it that way is what lets a
// new generator arrive without touching the use case.
//
// # Names are a public surface
//
// A generator name goes into somebody's recipe and stays there. The names below
// are therefore a contract in the same sense as the identifiers of the
// mutators, and they need a pass before the repository becomes public along
// with the rest.
package values

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/corpus"
	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/intl"
	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/pl"
	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/us"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// The generator names this build knows.
const (
	FirstName  = "first_name"
	LastName   = "last_name"
	Gender     = "gender"
	BirthDate  = "birth_date"
	City       = "city"
	Region     = "region"
	PostalCode = "postal_code"
	Pesel      = "pesel"
	Nip        = "nip"
	Regon      = "regon"
	SSN        = "ssn"
	ABA        = "aba"
	IBAN       = "iban"
)

// roles maps a generator onto the role it plays in a consistency rule.
//
// Only two generators play one today, and both are read by an identifier. The
// role comes from here rather than from the name of the column, because a
// recipe may call a column anything and half of them will not be in English.
var roles = map[string]string{
	BirthDate: provider.RoleBirthDate,
	Gender:    provider.RoleGender,
}

// Source produces values for one locale.
type Source struct {
	locale string
	seed   seed.Source
}

// Contract check, so that a mismatch fails to compile rather than at run time.
var _ generate.Values = Source{}

// New returns a source of values for a locale and a seed.
func New(locale, from string) Source {
	if locale == "" {
		locale = "en-US"
	}
	return Source{locale: locale, seed: seed.New(from)}
}

// Known reports whether this build has a generator of that name.
func (s Source) Known(generator string) bool {
	_, found := producers[generator]
	return found
}

// Names returns every generator this build has, sorted.
func (s Source) Names() []string {
	names := make([]string, 0, len(producers))
	for name := range producers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// RoleOf returns the role a generator plays, and whether it plays one.
func (s Source) RoleOf(generator string) (string, bool) {
	role, plays := roles[generator]
	return role, plays
}

// Produce returns one value.
func (s Source) Produce(req generate.Request) (string, error) {
	produce, found := producers[req.Generator]
	if !found {
		return "", fault.New(fault.Config, "no generator called "+req.Generator)
	}
	value, err := produce(s, req)
	if err != nil {
		return "", err
	}
	if fits(value, req) {
		return value, nil
	}
	return "", fault.New(fault.Data, fmt.Sprintf(
		"%s produced a value of %d %s for %s, which accepts %d. Raise the limit or change the generator",
		req.Generator, measure(value, req.Unit), req.Unit, req.At.Field, req.MaxLength))
}

// producer is one way of making a value.
type producer func(Source, generate.Request) (string, error)

// producers is the registry. It is a map rather than a switch so that the list
// of names and the list of implementations cannot drift apart.
var producers = map[string]producer{
	FirstName:  Source.firstName,
	LastName:   Source.lastName,
	Gender:     Source.gender,
	BirthDate:  Source.birthDate,
	City:       fromList("cities"),
	Region:     fromList("regions"),
	PostalCode: Source.postalCode,
	Pesel:      fromProvider(pl.PESEL{}),
	Nip:        fromProvider(pl.NIP{}),
	Regon:      fromProvider(pl.REGON{}),
	SSN:        fromProvider(us.SSN{}),
	ABA:        fromProvider(us.ABA{}),
	IBAN:       fromProvider(intl.IBAN{}),
}

// fromProvider adapts an identifier provider to this registry.
func fromProvider(p provider.Provider) producer {
	return func(s Source, req generate.Request) (string, error) {
		return p.Generate(provider.Request{Source: s.seed, At: req.At, Agrees: req.Agrees})
	}
}

// fromList draws an entry out of a word list.
func fromList(name string) producer {
	return func(s Source, req generate.Request) (string, error) {
		return s.draw(name, req)
	}
}

// firstName draws a name of the gender the row already carries.
//
// A row whose gender says nothing draws from the two lists together rather than
// from one of them, because picking a side quietly would make every dataset
// without a gender column read as one sex.
func (s Source) firstName(req generate.Request) (string, error) {
	switch bound, _ := req.Agrees.Value(provider.RoleGender); provider.Gender(bound) {
	case provider.Female:
		return s.draw("first-names-female", req)
	case provider.Male:
		return s.draw("first-names-male", req)
	default:
		return s.drawEither("first-names-female", "first-names-male", req)
	}
}

// lastName draws a surname, and in Polish that depends on the gender of the
// bearer.
//
// Polish surnames inflect: a family name ending in ski is noted for a man and
// ska for a woman, so a list drawn without regard to the gender column produces
// records a Polish reader sees as wrong at a glance. The unisex list is the one
// that does not inflect and is correct for either.
func (s Source) lastName(req generate.Request) (string, error) {
	if !strings.HasPrefix(s.locale, "pl") {
		return s.draw("last-names", req)
	}
	if bound, _ := req.Agrees.Value(provider.RoleGender); provider.Gender(bound) == provider.Male {
		return s.drawEither("last-names-male", "last-names-unisex", req)
	}
	return s.draw("last-names-unisex", req)
}

// gender draws one of the two values a numbering scheme can encode.
func (s Source) gender(req generate.Request) (string, error) {
	if s.seed.IntN(req.At, 2) == 0 {
		return string(provider.Female), nil
	}
	return string(provider.Male), nil
}

// birthDate draws a date, written the one way that does not depend on the
// settings of whichever machine ran the tool.
//
// The span is fixed here rather than read from the recipe, because the age
// range is a generator setting the format documents and this build carries
// without reading. Reporting that is the business of the validator, which
// already does it.
func (s Source) birthDate(req generate.Request) (string, error) {
	const (
		earliest = 1930
		latest   = 2007
	)
	year := earliest + int(s.seed.IntN(coordinate(req, ":year"), latest-earliest+1))
	month := 1 + int(s.seed.IntN(coordinate(req, ":month"), 12))
	day := 1 + int(s.seed.IntN(coordinate(req, ":day"), int64(daysIn(year, month))))
	return fmt.Sprintf("%04d-%02d-%02d", year, month, day), nil
}

// postalCode draws a code, and only where this build carries a register of
// them.
func (s Source) postalCode(req generate.Request) (string, error) {
	if strings.HasPrefix(s.locale, "pl") {
		return "", fault.New(fault.Config, "this build has no register of postal codes for "+
			s.locale+", so a postal code cannot be produced there. Generate the dataset for en-US "+
			"or drop the field")
	}
	return us.Postal{}.Generate(provider.Request{Source: s.seed, At: req.At})
}

// draw takes one entry out of a named list, honouring a maximum length by
// choosing among the entries that fit.
//
// Choosing rather than truncating is the whole point. A truncated name is a
// value this tool invented, and inventing values is what the second pillar does
// on purpose and the first must never do by accident.
func (s Source) draw(list string, req generate.Request) (string, error) {
	entries, found := corpus.Load(s.locale, list)
	if !found {
		return "", fault.New(fault.Config, "this build has no "+list+" for "+s.locale)
	}
	return s.pick(entries, list, req)
}

// drawEither draws from two lists as if they were one, without concatenating
// them on every call.
func (s Source) drawEither(first, second string, req generate.Request) (string, error) {
	left, foundLeft := corpus.Load(s.locale, first)
	right, foundRight := corpus.Load(s.locale, second)
	if !foundLeft || !foundRight {
		return "", fault.New(fault.Config, "this build has no "+first+" or "+second+" for "+s.locale)
	}
	return s.pick(append(append([]string{}, left...), right...), first, req)
}

// pick chooses one entry, from those that fit when a limit is declared.
func (s Source) pick(entries []string, list string, req generate.Request) (string, error) {
	if req.HasMaxLength {
		fitting := entries[:0:0]
		for _, entry := range entries {
			if measure(entry, req.Unit) <= req.MaxLength {
				fitting = append(fitting, entry)
			}
		}
		entries = fitting
	}
	if len(entries) == 0 {
		return "", fault.New(fault.Data, fmt.Sprintf(
			"no entry of %s for %s is short enough for %s, which accepts %d %s",
			list, s.locale, req.At.Field, req.MaxLength, req.Unit))
	}
	return entries[s.seed.IntN(req.At, int64(len(entries)))], nil
}

// fits reports whether a value is within the declared maximum length.
func fits(value string, req generate.Request) bool {
	return !req.HasMaxLength || measure(value, req.Unit) <= req.MaxLength
}

// measure counts a value in the unit the field declared.
//
// Graphemes are counted as characters and this build says so rather than
// pretending otherwise: the standard library has no grapheme segmentation, and
// the package that does would be a second dependency. The two differ on emoji
// and on combining marks, which is exactly the case the unit exists for, so the
// gap is real and named rather than hidden.
func measure(value, unit string) int {
	if unit == string(recipe.Bytes) {
		return len(value)
	}
	return utf8.RuneCountInString(value)
}

// coordinate returns the address of one part of a composite value.
func coordinate(req generate.Request, part string) seed.Coordinate {
	return seed.Coordinate{
		Entity: req.At.Entity, Row: req.At.Row, Field: req.At.Field,
		Purpose: req.At.Purpose + part,
	}
}

// daysIn returns the length of a month, so that a drawn day is always a real
// one.
func daysIn(year, month int) int {
	switch month {
	case 4, 6, 9, 11:
		return 30
	case 2:
		if year%4 == 0 && (year%100 != 0 || year%400 == 0) {
			return 29
		}
		return 28
	default:
		return 31
	}
}
