package values

import (
	"fmt"
	"strconv"
	"time"

	"github.com/donislawdev/qadatagenerator/internal/app/generate"
)

// What a field can take, asked before a row exists and again when a unique
// column has run out of luck.
//
// It stands apart from the registry because it answers a different question
// about the same generators: the registry makes one value, and this says how
// many there are and which is which. Both read the same entries, through the
// same function for a word list, so the two cannot part company.

// fromValues draws one entry out of the list the recipe wrote under the field.
func (s Source) fromValues(req generate.Request) (string, error) {
	return req.Values[s.seed.IntN(req.At, int64(len(req.Values)))], nil
}

// DomainOf returns every value a request can produce, after its limit on
// length and its agreement with the row, and whether this build knows them all.
//
// # What is known, and what is left unknown on purpose
//
// A list, a word list, a gender, a date of birth and a number described by its
// type alone are known, because each is drawn from a set this build holds or
// counts. An identifier is left unknown. Its set is the arithmetic of a scheme,
// often bound to a date and a gender, and counting it here would be a second
// implementation of every scheme that nothing checks against the first.
func (s Source) DomainOf(req generate.Request) (generate.Domain, bool) {
	switch {
	case len(req.Values) > 0:
		return listDomain("values", fitting(req.Values, req)), true
	case req.Generator == "":
		return typeDomain(req)
	}
	known, found := producers[req.Generator]
	switch {
	case !found:
		return generate.Domain{}, false
	case known.entries != nil:
		found, err := known.entries(s, req)
		if err != nil {
			// A list this build does not carry. The plan refuses that with its
			// own sentence before anything asks for a domain.
			return generate.Domain{}, false
		}
		return listDomain(found.key, s.entriesWithin(found, req)), true
	case req.Generator == Gender:
		female, male := genderWords(s.locale)
		return listDomain(Gender, fitting([]string{female, male}, req)), true
	case req.Generator == BirthDate:
		return birthDates(req), true
	}
	return generate.Domain{}, false
}

// listDomain is a domain read off a list, in the order of the list, with each
// value counted once.
//
// 🔴 A name both a woman and a man carry stands in both lists, and counting it
// twice would promise a unique column a value it cannot have, then name the
// wrong number when the column ran out. Measured on 2026-10-26 by the test
// beside this file: the two lists of first names for en-US join into 703
// entries and 690 names. The draw still takes such a name from either list,
// which only makes it more likely, and the domain still holds it.
func listDomain(key string, entries []string) generate.Domain {
	distinct := entries[:0:0]
	seen := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if !seen[entry] {
			seen[entry] = true
			distinct = append(distinct, entry)
		}
	}
	return generate.Domain{Key: key, Size: int64(len(distinct)), At: func(index int64) string {
		return distinct[index]
	}}
}

// typeDomain is the domain of a number described by its type alone, in the
// order of the numbers, the same digits fromType draws.
func typeDomain(req generate.Request) (generate.Domain, bool) {
	digits := defaultDigits
	if req.HasMaxLength && req.MaxLength < digits {
		digits = req.MaxLength
	}
	switch req.Type {
	case TypeInteger:
		if digits < 1 {
			return generate.Domain{Key: TypeInteger}, true
		}
		return generate.Domain{Key: TypeInteger, Size: power(digits), At: func(index int64) string {
			return strconv.FormatInt(index, 10)
		}}, true
	case TypeDecimal:
		if fine, _ := (Source{}).TypeFits(TypeDecimal, digits); !fine {
			return generate.Domain{Key: TypeDecimal}, true
		}
		const places = 100
		return generate.Domain{Key: TypeDecimal, Size: power(digits-decimalPlaces-1) * places,
			At: func(index int64) string {
				return fmt.Sprintf("%d.%02d", index/places, index%places)
			}}, true
	}
	return generate.Domain{}, false
}

// power is ten to the given number of digits, which the callers keep within
// what an int64 holds.
func power(digits int) int64 {
	limit := int64(1)
	for range digits {
		limit *= 10
	}
	return limit
}

// birthDates is every day a date of birth is drawn from, earliest first.
//
// Every one of them is ten characters long, so a limit shorter than that
// leaves none, which is the answer the draw gives one row later.
func birthDates(req generate.Request) generate.Domain {
	first := time.Date(earliestBirth, time.January, 1, 0, 0, 0, 0, time.UTC)
	after := time.Date(latestBirth+1, time.January, 1, 0, 0, 0, 0, time.UTC)
	days := int64(after.Sub(first).Hours() / 24)
	written := func(index int64) string {
		return first.AddDate(0, 0, int(index)).Format("2006-01-02")
	}
	if !fits(written(0), req) {
		return generate.Domain{Key: BirthDate}
	}
	return generate.Domain{Key: BirthDate, Size: days, At: written}
}

// Accepts reports whether a value written in the recipe under values can stand
// in the field the request describes, and says why not when it cannot.
//
// # Which types are checked
//
// The three this build can read for itself: a whole number, a decimal and a
// date, each in the one spelling the rest of the tool writes. Any other type is
// taken as the recipe wrote it (DEC-266), because checking an email or a money
// format here would put an unowned format into somebody's data, the same reason
// fromType makes only two of them.
func (s Source) Accepts(req generate.Request, value string) (bool, string) {
	switch {
	case req.Type == TypeInteger && !wholeDigits(value):
		return false, strconv.Quote(value) + " is not a whole number written in digits, such as 42 or -7"
	case req.Type == TypeDecimal && !decimalDigits(value):
		return false, strconv.Quote(value) + " is not a decimal written in digits with a point, such as 4.25"
	case req.Type == TypeDate && !isoDate(value):
		return false, strconv.Quote(value) + " is not a date written as year-month-day, such as 1984-02-29"
	case !fits(value, req):
		return false, fmt.Sprintf("%s is %d %s long, and the field accepts %d",
			strconv.Quote(value), measure(value, req.Unit), req.Unit, req.MaxLength)
	}
	return true, ""
}

// wholeDigits reports whether a value is a whole number in plain decimal
// digits, with a minus sign allowed in front.
func wholeDigits(value string) bool {
	if len(value) > 0 && value[0] == '-' {
		value = value[1:]
	}
	if value == "" {
		return false
	}
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}

// decimalDigits reports whether a value is a decimal in plain digits with a
// point, or a whole number, which a decimal column takes as well. A comma is
// refused, because the same file has to read the same in every country.
func decimalDigits(value string) bool {
	for at := range len(value) {
		if value[at] == '.' {
			return wholeDigits(value[:at]) && value[at+1:] != "" && wholeDigits(value[at+1:]) &&
				value[at+1] != '-'
		}
	}
	return wholeDigits(value)
}

// isoDate reports whether a value is a real day written as year-month-day.
func isoDate(value string) bool {
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}
