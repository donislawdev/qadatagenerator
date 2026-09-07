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
	"strconv"
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
	VIN        = "vin"
	EAN        = "ean"
	IMEI       = "imei"
	ISBN       = "isbn"
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

// The field types the catalog uses in applies_to. They are constants here
// because the corruption engine matches on them, so a typo would silently make
// every mutator inapplicable to a column rather than fail to build.
const (
	TypeString     = "string"
	TypeIdentifier = "identifier"
	TypeDate       = "date"
	TypePostalCode = "postal_code"
	TypeInteger    = "integer"
	TypeDecimal    = "decimal"
)

// types says what kind of value each generator produces.
//
// # Why this exists, and it is a measurement rather than tidiness
//
// The catalog aims damage at field types and a recipe declares generators, and
// until 2026-08-10 nothing joined the two. A field with a generator carried no
// type at all, because a recipe field names exactly one of type, generator or
// expression. The engine asks the entry whether it covers the field type, an
// empty type matches nothing except the reserved word any, and only two of the
// fifty two entries carry that. So a recipe written the ordinary way, with
// generators, left fifty of the fifty two entries inapplicable for a reason
// nobody had chosen. That is the same shape as the quiet mismatch the audit
// found in PL.pesel: two halves, each correct about its own half.
//
// # Why gender is a string rather than an enum
//
// It draws from two values and looks like the enumeration the catalog means.
// But the entry that puts a value outside a set also requires the set itself,
// and a set is something a recipe declares on the field. Calling this an enum
// would hand a mutator a capability nothing supplies, which is the failure
// DEC-98 already paid for once.
var types = map[string]string{
	FirstName:  TypeString,
	LastName:   TypeString,
	City:       TypeString,
	Region:     TypeString,
	Gender:     TypeString,
	BirthDate:  TypeDate,
	PostalCode: TypePostalCode,
	Pesel:      TypeIdentifier,
	Nip:        TypeIdentifier,
	Regon:      TypeIdentifier,
	SSN:        TypeIdentifier,
	ABA:        TypeIdentifier,
	IBAN:       TypeIdentifier,
	VIN:        TypeIdentifier,
	EAN:        TypeIdentifier,
	IMEI:       TypeIdentifier,
	ISBN:       TypeIdentifier,
}

// Source produces values for one locale.
type Source struct {
	locale string
	seed   seed.Source
}

// Contract check, so that a mismatch fails to compile rather than at run time.
var _ generate.Values = Source{}

// defaultLocale is what a recipe naming none is read as, in one place because
// the source and the answer about what it can produce have to agree on it.
const defaultLocale = "en-US"

// New returns a source of values for a locale and a seed.
func New(locale, from string) Source {
	if locale == "" {
		locale = defaultLocale
	}
	return Source{locale: locale, seed: seed.New(from)}
}

// everywhere is a generator that reads the same roles in every locale.
func everywhere(roles ...string) func(string) []string {
	return func(string) []string { return roles }
}

// whereSurnamesInflect is the gender, in the locales whose surnames carry it.
//
// Measured on 2026-09-05 rather than reasoned: binding two genders at one
// address moves a Polish surname and leaves an English one alone, because the
// Polish lists are split by inflection and the English one is not. The test of
// the register does that measurement for every generator and every locale.
func whereSurnamesInflect(locale string) []string {
	if strings.HasPrefix(locale, "pl") {
		return []string{provider.RoleGender}
	}
	return nil
}

// RolesRead are the consistency roles that generator takes account of, sorted.
//
// An unknown generator reads nothing, which is the same answer as a generator
// that reads nothing, and that is deliberate: a caller asking this has already
// been told whether the name exists, and answering twice would put the refusal
// in two places.
func (s Source) RolesRead(generator string) []string {
	known, found := producers[generator]
	if !found || known.reads == nil {
		return nil
	}
	roles := append([]string(nil), known.reads(s.locale)...)
	sort.Strings(roles)
	return roles
}

// ReadsRole reports whether that generator takes account of that role.
func (s Source) ReadsRole(generator, role string) bool {
	for _, read := range s.RolesRead(generator) {
		if read == role {
			return true
		}
	}
	return false
}

// Known reports whether this build has a generator of that name.
func (s Source) Known(generator string) bool {
	_, found := producers[generator]
	return found
}

// Locales are the locales this build carries word lists for, sorted.
//
// It is asked of this package rather than of the corpus directly, so that a
// caller describing what this build can produce has one door to knock on. What
// the answer means, and what it does not, is written where it comes from.
func Locales() []string { return corpus.Locales() }

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

// TypeOf says what kind of value a generator produces, and whether this build
// knows the generator at all.
//
// The second result is not decoration. An unknown name answering with an empty
// type would be indistinguishable from a generator that genuinely produces
// something untyped, and the engine would then aim damage at a column on the
// strength of a guess.
func (s Source) TypeOf(generator string) (string, bool) {
	kind, known := types[generator]
	return kind, known
}

// ProviderOf says which numbering scheme a generator draws on, and whether it
// draws on one at all.
//
// The answer comes from the registry rather than from a table beside it, for
// the same reason the capabilities do: a second list is a place for the two to
// disagree, and a generator added without its scheme would then be a column the
// engine believes belongs to nothing.
func (s Source) ProviderOf(generator string) (string, bool) {
	known, found := producers[generator]
	if !found || known.behind == nil {
		return "", false
	}
	return known.behind.ID(), true
}

// ProviderNamed returns the scheme with that identifier and whether this build
// carries it.
//
// It walks the registry rather than holding a second map keyed by identifier.
// The registry is small, this is asked once per damaged value, and a second map
// is a second place for a scheme to be missing from.
func (s Source) ProviderNamed(scheme string) (provider.Provider, bool) {
	for _, known := range producers {
		if known.behind != nil && known.behind.ID() == scheme {
			return known.behind, true
		}
	}
	return nil, false
}

// Schemes returns every numbering scheme this build carries, sorted by
// identifier and each one once.
//
// # Why the whole list, when everything else here is asked by name
//
// The mode that reads somebody else's file has no name to ask by. Nothing
// declared what a column holds, so the only way to find out is to offer a value
// to every scheme and see which one recognises it. That is a different question
// from the one the rest of this file answers, which is always what a named
// generator produces.
//
// Sorted rather than in registry order, because whoever asks compares one
// answer against another and Go randomises map iteration on purpose. A scheme
// reached by two generators appears once, since the answer is about schemes and
// not about the ways of producing them.
func (s Source) Schemes() []provider.Provider {
	seen := make(map[string]provider.Provider, len(producers))
	for _, known := range producers {
		if known.behind == nil {
			continue
		}
		seen[known.behind.ID()] = known.behind
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	schemes := make([]provider.Provider, 0, len(ids))
	for _, id := range ids {
		schemes = append(schemes, seen[id])
	}
	return schemes
}

// ProducesType reports whether this build can fill a field described by that
// type alone.
//
// The list is short on purpose and is not the whole catalog vocabulary. Each
// remaining type is a decision about shape that belongs with the generator
// owning it, and inventing an email or a money format here would put an unowned
// format into somebody's data.
func (s Source) ProducesType(kind string) bool {
	return kind == TypeInteger || kind == TypeDecimal
}

// Produce returns one value.
//
// A request with no generator is a field the recipe described by type alone.
// Such a field produced nothing until 2026-08-10 and the plan said nothing
// about it either, so a declared column arrived empty and no part of the run
// mentioned it. That is silence in the first pillar, found while planning the
// second.
func (s Source) Produce(req generate.Request) (string, error) {
	if req.Generator == "" {
		return s.fromType(req)
	}
	known, found := producers[req.Generator]
	if !found {
		return "", fault.New(fault.Config, "no generator called "+req.Generator)
	}
	value, err := known.produce(s, req)
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

// Digits a number gets when the field declares no maximum length.
//
// # Why six, and what it is not
//
// It is a default, not a measurement, and the difference matters because this
// project treats a number that pretends to be a measurement as a defect. Six
// digits keeps a value readable, stays far inside every integer type a database
// offers, and is wide enough that a column of them does not look like a counter.
// A recipe that needs a particular range has nowhere to say so yet: field
// parameters are carried and unread, which the plan reports rather than hides.
const defaultDigits = 6

// fromType produces a value for a field described by type alone.
//
// Only the two numeric types are built. The rest of the catalog vocabulary,
// such as email or money, would each be a decision about shape that belongs
// with the generator that owns it rather than with a fallback, and inventing
// one here would put an unowned format into people's data.
func (s Source) fromType(req generate.Request) (string, error) {
	digits := defaultDigits
	if req.HasMaxLength && req.MaxLength < digits {
		// A number longer than the column is a value that fits by our
		// arithmetic and is rejected on arrival, which is the failure the unit
		// on max_length exists to prevent.
		digits = req.MaxLength
	}
	switch req.Type {
	case TypeInteger:
		return strconv.FormatInt(s.wholeNumber(req, digits), 10), nil
	case TypeDecimal:
		// Two places, because that is what money and every rate in a business
		// system carry, and because a decimal with a varying number of places
		// is a different value on every row for reasons the reader cannot see.
		const places = 2
		if digits <= places {
			return "", fault.New(fault.Data, fmt.Sprintf(
				"%s accepts %d %s, and a decimal needs room for a point and two places",
				req.At.Field, req.MaxLength, req.Unit))
		}
		whole := s.wholeNumber(req, digits-places-1)
		fraction := s.seed.IntN(coordinate(req, ":fraction"), 100)
		return fmt.Sprintf("%d.%02d", whole, fraction), nil
	default:
		return "", fault.New(fault.Config, fmt.Sprintf(
			"%s is declared as %s, and this build produces values for %s and %s only",
			req.At.Field, req.Type, TypeInteger, TypeDecimal)).
			DoInstead("give the field a generator, or change the type")
	}
}

// widestDrawn is the most digits a draw can be asked for.
//
// # Why a ceiling on something nothing can reach
//
// Nothing does reach it, and that is read rather than hoped: the one caller
// starts at defaultDigits, which is six, and only ever lowers it. The ceiling
// is here for the reason take.CheckOutput is, which is that the cost of being
// wrong is out of proportion to the cost of the check. The limit below is built
// by multiplying, and ten to the nineteenth does not fit in the integer holding
// it. It would wrap to a negative number and the draw would panic on a bound at
// or below zero, in the middle of writing somebody's dataset, one value at a
// time.
//
// Eighteen is arithmetic and not a taste: ten to the eighteenth is the largest
// power of ten an int64 holds. Asking for more gets the widest number this can
// draw rather than a panic, and how wide a field actually gets is decided by
// the caller above rather than here.
const widestDrawn = 18

// wholeNumber draws a number of at most the given number of digits.
func (s Source) wholeNumber(req generate.Request, digits int) int64 {
	if digits < 1 {
		return 0
	}
	if digits > widestDrawn {
		digits = widestDrawn
	}
	limit := int64(1)
	for i := 0; i < digits; i++ {
		limit *= 10
	}
	return s.seed.IntN(coordinate(req, ":number"), limit)
}

// producer is one way of making a value.
type producer func(Source, generate.Request) (string, error)

// generator is one name a recipe can use, together with what stands behind it.
//
// # Why what stands behind it is recorded rather than looked up twice
//
// The capabilities a run has depend on which providers and which word lists
// take part in it, and that is a second question about the same registry. A
// separate list answering it would be a copy, and copies part company. Keeping
// the provider beside the function means one entry answers both, and adding a
// generator cannot leave the second answer behind.
type generator struct {
	// produce makes the value.
	produce producer
	// behind is the identifier provider this draws on, when there is one. A
	// generator drawing from a word list or from arithmetic leaves it nil.
	behind provider.Provider
	// list is the corpus this draws from, empty when it draws from nothing.
	list string
	// needs answers whether this build can produce the value in a locale,
	// before a row exists. A generator that works everywhere leaves it nil.
	needs func(locale string) (bool, string)
	// reads answers which roles of a consistency rule this generator takes
	// account of, in a given locale. A generator that ignores every one of
	// them in every locale leaves it nil.
	//
	// # Why the registry has to say this, when nothing else asks
	//
	// A recipe binds a role and a provider reads it or does not, and until
	// 2026-09-05 nothing needed to know which. The verb that produces one
	// value does: asked for a city with a gender beside it, it would either
	// bind a value nothing reads and say nothing, or refuse. The first is
	// silence in the shape this product refuses everywhere else, and the
	// second needs this field.
	//
	// # Why a written list is honest here
	//
	// Nothing in a provider declares what it reads, so this cannot be derived
	// from the code. What it CAN be is measured: bind two different values for
	// a role at one address and see whether the produced value moves. A guard
	// beside this file does exactly that for every generator and every role,
	// so the list is closed from both sides by measurement rather than by
	// somebody remembering to extend it.
	//
	// 🔴 It takes a locale for the same reason `needs` above it does, and the
	// reason is measured rather than anticipated: a surname reads the gender
	// in Polish, because Polish surnames inflect, and reads nothing in
	// English. A register that answered without the locale would promise an
	// influence that half the runs do not have.
	reads func(locale string) []string
}

// producers is the registry. It is a map rather than a switch so that the list
// of names and the list of implementations cannot drift apart.
var producers = map[string]generator{
	FirstName: {produce: Source.firstName, list: "first-names",
		needs: fromLists("first-names-female", "first-names-male"),
		reads: everywhere(provider.RoleGender)},
	LastName: {produce: Source.lastName, list: "last-names", needs: surnameLists,
		reads: whereSurnamesInflect},
	Gender:     {produce: Source.gender},
	BirthDate:  {produce: Source.birthDate},
	City:       {produce: fromList("cities"), list: "cities", needs: fromLists("cities")},
	Region:     {produce: fromList("regions"), list: "regions", needs: fromLists("regions")},
	PostalCode: {produce: Source.postalCode, behind: us.Postal{}, needs: postalRegister},
	Pesel: {produce: fromProvider(pl.PESEL{}), behind: pl.PESEL{},
		reads: everywhere(provider.RoleBirthDate, provider.RoleGender)},
	Nip:   {produce: fromProvider(pl.NIP{}), behind: pl.NIP{}},
	Regon: {produce: fromProvider(pl.REGON{}), behind: pl.REGON{}},
	SSN:   {produce: fromProvider(us.SSN{}), behind: us.SSN{}},
	ABA:   {produce: fromProvider(us.ABA{}), behind: us.ABA{}},
	IBAN:  {produce: fromProvider(intl.IBAN{}), behind: intl.IBAN{}},
	VIN:   {produce: fromProvider(intl.VIN{}), behind: intl.VIN{}},
	EAN:   {produce: fromProvider(intl.EAN{}), behind: intl.EAN{}},
	IMEI:  {produce: fromProvider(intl.IMEI{}), behind: intl.IMEI{}},
	ISBN:  {produce: fromProvider(intl.ISBN{}), behind: intl.ISBN{}},
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
	// 🔴 The bound value is PARSED rather than compared. Until 2026-09-06 this
	// compared it against the word female itself, while the scheme behind a
	// PESEL accepted eleven spellings, so a recipe binding f got a PESEL of a
	// woman beside a name drawn from both lists AND NOTHING SAID SO. One row,
	// one declared rule, two answers.
	switch bound, _ := req.Agrees.Value(provider.RoleGender); genderOf(bound) {
	case provider.Female:
		return s.draw("first-names-female", req)
	case provider.Male:
		return s.draw("first-names-male", req)
	default:
		return s.drawEither("first-names-female", "first-names-male", req)
	}
}

// genderOf reads a bound gender the way the whole product reads one.
//
// An unreadable value answers Unknown rather than failing, and the difference
// from the scheme next door is deliberate: a name drawn without a gender is an
// ordinary name, while an identifier that cannot encode what it was told to
// encode is a value nobody asked for. The scheme refuses, a word list draws
// from both, and the run is refused by the scheme either way whenever both are
// in the same rule.
func genderOf(bound string) provider.Gender {
	if gender, known := provider.ParseGender(bound); known {
		return gender
	}
	return provider.Unknown
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
	if bound, _ := req.Agrees.Value(provider.RoleGender); genderOf(bound) == provider.Male {
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
	if fine, why := postalRegister(s.locale); !fine {
		return "", fault.New(fault.Config, why).DoInstead(postalInstead)
	}
	return us.Postal{}.Generate(provider.Request{Source: s.seed, At: req.At})
}

// noList is the sentence saying this build carries no such word list.
//
// 🔴 It stands in one place because two answers say it: the run, when it draws,
// and the plan, before anything is drawn. Until 2026-08-26 only the run could
// say it, so validate called a recipe usable and a dry run counted the rows of
// a dataset that could not be produced. Two sentences would be two places for
// the plan and the run to part company, which is the drift the plan exists to
// make impossible.
func noList(list, locale string) string {
	return "this build has no " + list + " for " + locale
}

// postalRegister answers whether a postal code can be drawn in a locale.
func postalRegister(locale string) (bool, string) {
	if !strings.HasPrefix(locale, "pl") {
		return true, ""
	}
	return false, "this build has no register of postal codes for " + locale +
		", so a postal code cannot be produced there"
}

// postalInstead is what to do about it, kept apart from the reason above.
//
// 🔴 The two used to be one string, which put them in two places at once: the
// listing of what this build cannot fill showed the advice as part of the
// reason, and the refusal took the advice into its message and then had the
// sentence for its kind added underneath. Two next moves, and the second one
// about a recipe that is not the matter here.
const postalInstead = "generate the dataset for a locale this build carries a register for, " +
	"or leave the field out"

// fromLists answers whether every named list is carried for a locale.
func fromLists(names ...string) func(string) (bool, string) {
	return func(locale string) (bool, string) {
		for _, name := range names {
			if _, found := corpus.Load(locale, name); !found {
				return false, noList(name, locale)
			}
		}
		return true, ""
	}
}

// surnameLists names the lists a surname is drawn from, which depend on the
// locale for the reason written at lastName.
func surnameLists(locale string) (bool, string) {
	if !strings.HasPrefix(locale, "pl") {
		return fromLists("last-names")(locale)
	}
	return fromLists("last-names-male", "last-names-unisex")(locale)
}

// CanProduce reports whether this build can fill a field with that generator in
// a locale, and says why not when it cannot.
//
// # Why the locale is an argument rather than the one this source carries
//
// The question belongs to the build rather than to a source, and both callers
// ask it about a recipe rather than about themselves. The plan holds the
// locale the recipe declared, and the validator is handed a source built with
// none at all, so a method reading its own field would answer the validator
// about the wrong locale every time.
func (s Source) CanProduce(generator, locale string) (bool, string) {
	if locale == "" {
		locale = defaultLocale
	}
	known, found := producers[generator]
	if !found || known.needs == nil {
		return true, ""
	}
	return known.needs(locale)
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
		return "", fault.New(fault.Config, noList(list, s.locale))
	}
	return s.pick(entries, list, req)
}

// drawEither draws from two lists as if they were one, without concatenating
// them on every call.
func (s Source) drawEither(first, second string, req generate.Request) (string, error) {
	entries, found := corpus.Joined(s.locale, first, second)
	if !found {
		// Named one at a time rather than as a pair, so that the sentence is
		// the one the plan gives for the same list and the reader is told which
		// of the two is missing rather than that one of them is.
		if _, carried := corpus.Load(s.locale, first); !carried {
			return "", fault.New(fault.Config, noList(first, s.locale))
		}
		return "", fault.New(fault.Config, noList(second, s.locale))
	}
	return s.pick(entries, first, req)
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
