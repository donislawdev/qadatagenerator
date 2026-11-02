// Package pl carries the providers of Polish values and identifiers.
//
// They live under the adapters rather than in the domain because they hold
// knowledge of one country, which is exactly the thing this project expects to
// receive as contributions. The contract they satisfy is in the domain, where
// the corruption engine can see it.
package pl

import (
	"errors"
	"fmt"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// PESEL is the Polish national identification number.
//
// # Where the rules came from
//
// The structure, the century table, the position of the gender digit and the
// check digit weights were read from the government description of the number
// rather than recalled. This project has already been bitten once by an
// identifier whose rules were written from memory and turned out not to exist,
// so the rule here is that an algorithm arrives with a source or does not
// arrive.
//
// The number is eleven digits, RRMMDDPPPPK. The first six are the date of
// birth, the century is carried by an offset added to the month, the tenth
// digit is even for women and odd for men, and the last is a check digit over
// the first ten with the weights 1 3 7 9 1 3 7 9 1 3.
//
// A second source confirms all four of those, and it took a day to find. The
// government page states the rules in words and works no example, so for that
// day the weights here rested on prose with nothing to check them against. The
// country sheet the Ministry of Finance publishes on the European TIN Portal,
// under its own copyright notice, takes one number apart digit by digit. It
// agrees on the weights, on the century table and on the gender digit, and its
// example is a test in this package.
//
// # Why this one matters more than the others
//
// It is the bridge between the two pillars. Pillar one has to make it agree
// with the date of birth and the gender in the same record. Pillar two gets
// two different corruptions out of that: a number that is wrong in itself, and
// a number that is faultless and contradicts the rest of the row. The second
// kind is the one applications almost never check.
//
// # Ethics
//
// PESEL has no reserved range. There is no block of numbers set aside for
// documentation, so any well formed PESEL could be somebody's. Nothing here
// can fix that, and pretending otherwise would be worse than saying it: a
// dataset carrying these has to be marked synthetic in its own metadata, which
// is a duty of whatever writes the output rather than of this file.
type PESEL struct{}

// Contract checks, so that a mismatch between what this provider claims and
// what it implements fails to compile rather than at run time.
var (
	_ provider.Provider      = PESEL{}
	_ provider.CheckDigited  = PESEL{}
	_ provider.DateEncoder   = PESEL{}
	_ provider.GenderEncoder = PESEL{}
)

const (
	// length is the number of digits, and checkPosition is the index of the
	// check digit. genderPosition is the digit that carries the gender.
	length         = 11
	checkPosition  = 10
	genderPosition = 9
)

// weights are applied to the first ten digits. Read from the government
// description, not derived.
var weights = [10]int{1, 3, 7, 9, 1, 3, 7, 9, 1, 3}

// centuries maps the first year of a century to the number added to the month.
//
// The table is the whole reason a PESEL can carry a date beyond nineteen
// ninety nine in two digits of year. It is written out rather than computed,
// because it is a decision of the numbering scheme and not an arithmetic
// pattern to be rediscovered.
var centuries = []struct {
	firstYear int
	offset    int
}{
	{1800, 80},
	{1900, 0},
	{2000, 20},
	{2100, 40},
	{2200, 60},
}

// ErrOutOfRange is the cause of every refusal of a date the numbering scheme
// cannot express, reachable through errors.Is.
var ErrOutOfRange = errors.New("the scheme covers the years 1800 to 2299 only")

// outOfRange refuses a real day the scheme has no century for, naming it.
//
// # Why it carries a kind when the other refusals of this file do not
//
// It is the one a person reaches: a date of birth asked for by hand is checked
// for its shape and not against this scheme, so a well formed date from 1700
// arrives here. Until 2026-11-02 it arrived at the surface as a bare error,
// which every surface reports as a defect of this tool with the advice to
// report it, for a request that was simply one the scheme cannot meet. The
// refusals of a date or a gender that cannot be read are stopped before they
// get this far, so nothing reaches them with a value a person typed.
func outOfRange(date provider.Date) error {
	return fault.Wrap(fault.Data, ErrOutOfRange,
		fmt.Sprintf("pesel: %04d-%02d-%02d cannot be written", date.Year, date.Month, date.Day)).
		DoInstead("give a date of birth in those years")
}

// ID names this provider. The country prefix is ISO 3166-1 alpha-2, because a
// provider belongs to a country rather than to a language.
func (PESEL) ID() string { return "PL.pesel" }

// Capabilities are what this provider brings.
//
// identifier.reserved-range is deliberately absent. PESEL has no reserved
// block, and claiming one would be a promise this scheme cannot keep.
// Mask is eleven digits, which is the form the regulation prints as
// RRMMDDPPPPK. The letters of that form are not carried here: a mask says where
// a character varies, and what the varying characters MEAN is the business of
// the date and gender readers below, which can be asked.
func (PESEL) Mask() string { return strings.Repeat(string(provider.MaskDigit), length) }

func (PESEL) Capabilities() []capability.Name {
	return []capability.Name{
		capability.IdentifierCheckDigit,
		capability.IdentifierEncodesDate,
		capability.IdentifierEncodesGender,
	}
}

// Generate produces one PESEL for the place described by the request.
//
// The date of birth and the gender come from the agreement when the recipe
// bound them, so that the number agrees with the rest of the record. When it
// bound neither, both are derived from the same coordinate, which keeps the
// answer a function of the place rather than of a clock or a counter.
func (p PESEL) Generate(req provider.Request) (string, error) {
	date, err := p.dateFor(req)
	if err != nil {
		return "", err
	}
	gender, err := p.genderFor(req)
	if err != nil {
		return "", err
	}

	month, err := encodeMonth(date)
	if err != nil {
		return "", err
	}
	if !validDate(date) {
		return "", fmt.Errorf("pesel: %04d-%02d-%02d is not a date", date.Year, date.Month, date.Day)
	}

	// The ordinal number is drawn from the coordinate, then its last digit is
	// forced to carry the gender. Drawing the whole of it and correcting one
	// digit keeps the draw a pure function of the place.
	serial := req.Source.IntN(seed.Coordinate{
		Entity:  req.At.Entity,
		Row:     req.At.Row,
		Field:   req.At.Field,
		Purpose: req.At.Purpose + ":pesel-serial",
	}, 1000)

	digits := make([]int, 0, length)
	digits = appendPair(digits, date.Year%100)
	digits = appendPair(digits, month)
	digits = appendPair(digits, date.Day)
	digits = appendTriple(digits, int(serial))
	digits = append(digits, genderDigit(req, gender))
	digits = append(digits, checkDigit(digits))

	return digitsToString(digits), nil
}

// Validate answers whether a value is a correct PESEL.
//
// It is written from the rules of the scheme rather than from what Generate
// happens to produce. Checking a generator with its own arithmetic would make
// one mistake twice and let the two agree with each other.
func (PESEL) Validate(value string) provider.Verdict {
	if len(value) != length {
		return provider.Invalid(fmt.Sprintf("a PESEL has %d digits, this one has %d", length, len(value)))
	}
	digits, ok := parseDigits(value)
	if !ok {
		return provider.Invalid("a PESEL is digits only")
	}
	date, ok := decodeDate(digits)
	if !ok {
		return provider.Invalid("the month does not belong to any century the scheme covers")
	}
	if !validDate(date) {
		return provider.Invalid(fmt.Sprintf("the encoded date %04d-%02d-%02d does not exist", date.Year, date.Month, date.Day))
	}
	if want := checkDigit(digits[:checkPosition]); digits[checkPosition] != want {
		return provider.Invalid(fmt.Sprintf("the check digit is %d and should be %d", digits[checkPosition], want))
	}
	return provider.Valid()
}

// CheckDigits says where the check digit sits.
//
// The value is checked for being digits and not only for its length. Length
// alone answered yes for eleven characters of anything, handing a mutator an
// offset into a value the scheme does not describe, which is how a corruption
// lands somewhere nobody chose. The statistical number already asked the
// stricter question and this is the two agreeing.
func (PESEL) CheckDigits(value string) ([]int, bool) {
	if digits, ok := parseDigits(value); !ok || len(digits) != length {
		return nil, false
	}
	return []int{checkPosition}, true
}

// WithCheckDigits returns the value with its check digit recomputed.
func (PESEL) WithCheckDigits(value string) (string, error) {
	digits, ok := parseDigits(value)
	if !ok || len(digits) != length {
		return "", fmt.Errorf("pesel: %s is not eleven digits", provider.Show(value))
	}
	digits[checkPosition] = checkDigit(digits[:checkPosition])
	return digitsToString(digits), nil
}

// EncodedDate reads the date of birth out of the value.
func (PESEL) EncodedDate(value string) (provider.Date, bool) {
	digits, ok := parseDigits(value)
	if !ok || len(digits) != length {
		return provider.Date{}, false
	}
	date, ok := decodeDate(digits)
	if !ok || !validDate(date) {
		return provider.Date{}, false
	}
	return date, true
}

// WithDate returns the value carrying a different date of birth, check digit
// included.
//
// This is what lets a mutator produce a number that is well formed on its own
// and disagrees with the rest of the record, which is the corruption an
// application is least likely to catch.
func (p PESEL) WithDate(value string, date provider.Date) (string, error) {
	digits, ok := parseDigits(value)
	if !ok || len(digits) != length {
		return "", fmt.Errorf("pesel: %s is not eleven digits", provider.Show(value))
	}
	month, err := encodeMonth(date)
	if err != nil {
		return "", err
	}
	// A date that does not exist is refused here rather than written. The
	// mutator that wants an impossible date asks for it by name and gets it
	// from its own path, so that this one cannot produce one by accident.
	if !validDate(date) {
		return "", fmt.Errorf("pesel: %04d-%02d-%02d is not a date", date.Year, date.Month, date.Day)
	}

	rebuilt := make([]int, 0, length)
	rebuilt = appendPair(rebuilt, date.Year%100)
	rebuilt = appendPair(rebuilt, month)
	rebuilt = appendPair(rebuilt, date.Day)
	rebuilt = append(rebuilt, digits[6:checkPosition]...)
	rebuilt = append(rebuilt, checkDigit(rebuilt))
	return digitsToString(rebuilt), nil
}

// EncodedGender reads the gender out of the value.
func (PESEL) EncodedGender(value string) (provider.Gender, bool) {
	digits, ok := parseDigits(value)
	if !ok || len(digits) != length {
		return provider.Unknown, false
	}
	if digits[genderPosition]%2 == 0 {
		return provider.Female, true
	}
	return provider.Male, true
}

// WithGender returns the value carrying a different gender, check digit
// included.
func (PESEL) WithGender(value string, gender provider.Gender) (string, error) {
	digits, ok := parseDigits(value)
	if !ok || len(digits) != length {
		return "", fmt.Errorf("pesel: %s is not eleven digits", provider.Show(value))
	}
	switch gender {
	case provider.Female:
		digits[genderPosition] &^= 1
	case provider.Male:
		digits[genderPosition] |= 1
	default:
		return "", fmt.Errorf("pesel: the scheme encodes female and male, not %s", provider.Show(string(gender)))
	}
	digits[checkPosition] = checkDigit(digits[:checkPosition])
	return digitsToString(digits), nil
}

// --- the arithmetic, kept in one place ---------------------------------------

// checkDigit computes the eleventh digit over the first ten.
//
// The government description says to add only the last digit whenever a
// product runs to two, and to subtract only the last digit of the total. Both
// are the same as working modulo ten, and the final modulo is what turns a
// result of ten into nought. That last step is the case a straight subtraction
// gets wrong, and it happens whenever the weighted total ends in nought.
func checkDigit(first []int) int {
	sum := 0
	for i, digit := range first {
		sum += digit * weights[i]
	}
	return (10 - sum%10) % 10
}

func encodeMonth(date provider.Date) (int, error) {
	if date.Month < 1 || date.Month > 12 {
		return 0, fmt.Errorf("pesel: %d is not a month", date.Month)
	}
	for _, century := range centuries {
		if date.Year >= century.firstYear && date.Year < century.firstYear+100 {
			return date.Month + century.offset, nil
		}
	}
	return 0, outOfRange(date)
}

// decodeDate turns the first six digits back into a date, or reports that the
// month belongs to no century the scheme covers.
func decodeDate(digits []int) (provider.Date, bool) {
	year := digits[0]*10 + digits[1]
	month := digits[2]*10 + digits[3]
	day := digits[4]*10 + digits[5]

	for _, century := range centuries {
		if month > century.offset && month <= century.offset+12 {
			return provider.Date{
				Year:  century.firstYear + year,
				Month: month - century.offset,
				Day:   day,
			}, true
		}
	}
	return provider.Date{}, false
}

// validDate answers whether a date exists, leap years included.
//
// It is written here rather than taken from the standard library on purpose.
// The library normalises out of range values instead of refusing them, so the
// thirtieth of February would silently become the first or the second of
// March, and a mutator whose whole point is an impossible date would appear to
// have produced a possible one.
func validDate(date provider.Date) bool {
	if date.Month < 1 || date.Month > 12 || date.Day < 1 {
		return false
	}
	return date.Day <= daysIn(date.Year, date.Month)
}

func daysIn(year, month int) int {
	switch month {
	case 4, 6, 9, 11:
		return 30
	case 2:
		if isLeap(year) {
			return 29
		}
		return 28
	default:
		return 31
	}
}

func isLeap(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// --- turning a request into a date and a gender ------------------------------

// dateFor takes the bound date of birth, or derives one when the recipe bound
// none.
func (p PESEL) dateFor(req provider.Request) (provider.Date, error) {
	if bound, ok := req.Agrees.Value(provider.RoleBirthDate); ok {
		date, ok := parseISODate(bound)
		if !ok {
			return provider.Date{}, fmt.Errorf("pesel: %s is not a date written as YYYY-MM-DD", provider.Show(bound))
		}
		return date, nil
	}
	return p.deriveDate(req), nil
}

// deriveDate produces a date of birth from the coordinate.
//
// The span is a plausible range of living adults rather than the whole span
// the scheme can express. A dataset of people born in 1804 is technically
// valid and useless for testing, and the recipe can still ask for a bound date
// when it cares.
func (PESEL) deriveDate(req provider.Request) provider.Date {
	at := func(purpose string, bound int64) int {
		return int(req.Source.IntN(seed.Coordinate{
			Entity:  req.At.Entity,
			Row:     req.At.Row,
			Field:   req.At.Field,
			Purpose: req.At.Purpose + purpose,
		}, bound))
	}
	year := 1940 + at(":pesel-year", 70)
	month := 1 + at(":pesel-month", 12)
	day := 1 + at(":pesel-day", int64(daysIn(year, month)))
	return provider.Date{Year: year, Month: month, Day: day}
}

// genderSpellings maps every way a bound gender may be written onto what the
// scheme encodes.
//
// # Why the list is this wide, decided by the owner on 2026-08-03
//
// The column comes from somebody else's recipe or somebody else's file, and the
// question "what will a Polish tester write" has no answer: English or Polish,
// a full word or a single letter, all of them are likely. Guessing narrowly
// would refuse ordinary data, and the alternative to a wide list is not a
// narrow list, it is the silent draw this used to do.
//
// / genderFor takes the bound gender, or derives one.
//
// # Why an unreadable value is a failure rather than a draw
//
// Nothing bound means draw, and that is right: defaulting would make every
// unbound record the same sex. A binding this file cannot read is a different
// thing entirely, and until 2026-08-03 the two returned the same answer.
//
// The consequence was the quietest defect in this project. A recipe declaring
// pesel matches: [birth_date, gender] with a value such as kobieta or other
// received a number carrying a drawn gender, disagreeing with the column beside
// it, with nothing said anywhere. The tool sells the ability to produce an
// identifier that is well formed and contradicts its record as a deliberate
// corruption, so producing one by accident and reporting nothing is the exact
// shape of silence the manifest exists to prevent.
//
// The date half of the same agreement already failed loudly for an unreadable
// value. This is the two halves answering alike.
func (PESEL) genderFor(req provider.Request) (provider.Gender, error) {
	bound, ok := req.Agrees.Value(provider.RoleGender)
	if !ok {
		return provider.Unknown, nil
	}
	// The vocabulary is asked of the domain rather than held here. It lived in
	// this file until 2026-09-06, and being here is what let the generator of
	// first names disagree with it silently: a recipe binding f produced a
	// PESEL of a woman beside a name of nobody in particular.
	if gender, known := provider.ParseGender(bound); known {
		return gender, nil
	}
	return provider.Unknown, fmt.Errorf("pesel: %s is not a gender this scheme can encode, which are %s",
		provider.Show(bound), strings.Join(provider.GenderSpellings(), ", "))
}

// genderDigit picks the tenth digit, carrying the gender.
//
// An unbound gender is drawn rather than defaulted, because defaulting would
// make every unbound record the same sex and that is a dataset nobody wants to
// test with.
func genderDigit(req provider.Request, gender provider.Gender) int {
	drawn := int(req.Source.IntN(seed.Coordinate{
		Entity:  req.At.Entity,
		Row:     req.At.Row,
		Field:   req.At.Field,
		Purpose: req.At.Purpose + ":pesel-gender",
	}, 10))
	switch gender {
	case provider.Female:
		return drawn &^ 1
	case provider.Male:
		return drawn | 1
	default:
		return drawn
	}
}

// --- small helpers ------------------------------------------------------------

func parseDigits(value string) ([]int, bool) {
	digits := make([]int, 0, len(value))
	for _, symbol := range value {
		if symbol < '0' || symbol > '9' {
			return nil, false
		}
		digits = append(digits, int(symbol-'0'))
	}
	return digits, true
}

func digitsToString(digits []int) string {
	var built strings.Builder
	built.Grow(len(digits))
	for _, digit := range digits {
		built.WriteByte(byte('0' + digit))
	}
	return built.String()
}

func appendPair(digits []int, value int) []int {
	return appendFixed(digits, value, 2)
}

func appendTriple(digits []int, value int) []int {
	return appendFixed(digits, value, 3)
}

// appendFixed writes a value out as a fixed number of digits, most significant
// first, dropping anything above the width.
func appendFixed(digits []int, value, width int) []int {
	divisor := 1
	for remaining := 1; remaining < width; remaining++ {
		divisor *= 10
	}
	for ; divisor > 0; divisor /= 10 {
		digits = append(digits, value/divisor%10)
	}
	return digits
}

// parseISODate reads a date written as YYYY-MM-DD.
//
// The format is fixed rather than taken from the machine, because a date
// parsed by regional settings would give two different datasets on two
// machines from one seed.
func parseISODate(value string) (provider.Date, bool) {
	if len(value) != 10 || value[4] != '-' || value[7] != '-' {
		return provider.Date{}, false
	}
	digits, ok := parseDigits(value[0:4] + value[5:7] + value[8:10])
	if !ok {
		return provider.Date{}, false
	}
	date := provider.Date{
		Year:  digits[0]*1000 + digits[1]*100 + digits[2]*10 + digits[3],
		Month: digits[4]*10 + digits[5],
		Day:   digits[6]*10 + digits[7],
	}
	if !validDate(date) {
		return provider.Date{}, false
	}
	return date, true
}
