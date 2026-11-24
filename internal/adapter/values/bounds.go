package values

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// span is the range a number described by its type is drawn from, both ends
// included.
//
// It is counted in the smallest step of the type, ones for an integer and
// hundredths for a decimal, so that one draw covers every type and a bound of
// 99.99 is the whole number 9999 rather than a float that is almost it.
type span struct {
	kind      string
	low, high int64
	// bounded says the recipe gave min_value or max_value. A decimal with
	// neither keeps the two draws it has always been made with, a whole part
	// and then the places, so that a recipe written before the bounds existed
	// gives the same bytes from the same seed. An integer has always been one
	// draw from nought, which is the span with no bound, so it needs no second
	// path.
	bounded bool
}

// spanOf is the range a field described by type alone draws from, or why the
// field cannot be filled as it is declared, with what to do instead.
//
// # Where the ends come from
//
// An end the recipe gives is used as written. An end it leaves out is the one
// the field has without bounds: nought below, and above it the largest number
// of six digits, or of the maximum length when that is shorter. A field asking
// for numbers from five gets them up to 999999, and one asking for numbers up
// to fifty gets them from nought.
//
// # What is refused, and why each is not settled quietly
//
// Every refusal here is a range somebody wrote that the column would not
// follow. Rounding a bound to the places of the type, swapping two ends written
// the wrong way round or shortening a number to fit the field would each give
// a column that disagrees with its recipe, and the reader of the recipe would
// have no way to tell.
func spanOf(req generate.Request) (span, error) {
	digits := digitsOf(req)
	drawn := span{kind: req.Type, bounded: req.MinValue != "" || req.MaxValue != ""}
	switch req.Type {
	case TypeInteger:
		if !drawn.bounded && digits < 1 {
			return span{}, fault.New(fault.Config, fmt.Sprintf(
				"an integer needs at least one digit, and the field accepts %d", digits)).
				DoInstead("raise max_length")
		}
		drawn.high = power(digits) - 1
	case TypeDecimal:
		if !drawn.bounded && digits < decimalPlaces+2 {
			return span{}, fault.New(fault.Config, fmt.Sprintf(
				"a decimal needs at least %d characters, a digit, a point and %d places, and the field accepts %d",
				decimalPlaces+2, decimalPlaces, digits)).
				DoInstead("raise max_length, or describe the field with a generator")
		}
		drawn.high = power(digits-decimalPlaces-1)*hundredths - 1
	default:
		if drawn.bounded {
			return span{}, fault.New(fault.Config, fmt.Sprintf(
				"min_value and max_value bound a field of type %s or %s, and this one is declared as %s",
				TypeInteger, TypeDecimal, req.Type)).
				DoInstead("remove the bounds, or declare the field as " + TypeInteger + " or " + TypeDecimal)
		}
		return drawn, nil
	}
	if !drawn.bounded {
		return drawn, nil
	}
	return drawn.bound(req)
}

// hundredths is how many of the smallest step of a decimal make one, ten to
// the power of decimalPlaces.
const hundredths = 100

// bound puts the ends the recipe gives in place of the ones the field has
// without them, and asks whether the result is a range the field can hold.
func (s span) bound(req generate.Request) (span, error) {
	if req.MinValue != "" {
		low, err := s.units("min_value", req.MinValue)
		if err != nil {
			return span{}, err
		}
		if req.MaxValue == "" && low > s.high {
			return span{}, fault.New(fault.Config, fmt.Sprintf(
				"min_value %s is above %s, the largest number the field draws without max_value",
				req.MinValue, s.text(s.high))).
				DoInstead("add max_value")
		}
		s.low = low
	}
	if req.MaxValue != "" {
		high, err := s.units("max_value", req.MaxValue)
		if err != nil {
			return span{}, err
		}
		if req.MinValue == "" && high < s.low {
			return span{}, fault.New(fault.Config, fmt.Sprintf(
				"max_value %s is below %s, the smallest number the field draws without min_value",
				req.MaxValue, s.text(s.low))).
				DoInstead("add min_value")
		}
		s.high = high
	}
	if s.low > s.high {
		return span{}, fault.New(fault.Config, fmt.Sprintf(
			"min_value %s is above max_value %s", req.MinValue, req.MaxValue)).
			DoInstead("swap the two, or change one of them")
	}
	if !req.HasMaxLength {
		return s, nil
	}
	// The longest number of a range is at one of its ends, because a number
	// grows longer as it moves away from nought on either side. Every
	// character of a number is one byte and one grapheme, so the unit the
	// limit is counted in does not change the answer.
	for _, end := range []struct {
		name  string
		units int64
	}{{"smallest", s.low}, {"largest", s.high}} {
		if written := s.text(end.units); len(written) > req.MaxLength {
			return span{}, fault.New(fault.Config, fmt.Sprintf(
				"%s, the %s number the field draws, is %d characters long, and the field accepts %d",
				written, end.name, len(written), req.MaxLength)).
				DoInstead("raise max_length, or narrow min_value and max_value")
		}
	}
	return s, nil
}

// units reads a bound as a count of the smallest step of the type.
//
// The text is the one the reader of recipes made from the number written,
// such as -5 or 99.99. A bound reaching here from somewhere else is read the
// same way and refused when it is not that shape.
func (s span) units(name, text string) (int64, error) {
	places := 0
	if s.kind == TypeDecimal {
		places = decimalPlaces
	}
	unsigned, negative := strings.CutPrefix(text, "-")
	whole, fraction, _ := strings.Cut(unsigned, ".")
	switch {
	case len(fraction) > places && places == 0:
		return 0, fault.New(fault.Config, fmt.Sprintf(
			"%s %s is not a whole number, and the field is an %s", name, text, TypeInteger)).
			DoInstead("write a whole number, or declare the field as " + TypeDecimal)
	case len(fraction) > places:
		return 0, fault.New(fault.Config, fmt.Sprintf(
			"%s %s has %d decimal places, and a %s here has %d", name, text, len(fraction), TypeDecimal, places)).
			DoInstead(fmt.Sprintf("write the bound with %d places at most", places))
	case len(whole)+places > widestDrawn:
		return 0, fault.New(fault.Config, fmt.Sprintf(
			"%s %s has more digits than this build draws, which is %d", name, text, widestDrawn-places)).
			DoInstead("write a bound with fewer digits")
	}
	count, err := strconv.ParseInt(whole+fraction+strings.Repeat("0", places-len(fraction)), 10, 64)
	if err != nil || whole == "" {
		return 0, fault.New(fault.Config, fmt.Sprintf("%s %s is not a number", name, strconv.Quote(text))).
			DoInstead("write a number such as 1 or 99.99")
	}
	if negative {
		count = -count
	}
	return count, nil
}

// text writes a count of the smallest step the way the field shows it.
func (s span) text(units int64) string {
	if s.kind != TypeDecimal {
		return strconv.FormatInt(units, 10)
	}
	sign := ""
	if units < 0 {
		// Safe: every count here is within widestDrawn digits, far from the
		// one int64 that has no positive twin.
		sign, units = "-", -units
	}
	return fmt.Sprintf("%s%d.%0*d", sign, units/hundredths, decimalPlaces, units%hundredths)
}

// size is how many numbers the span holds.
//
// Both ends are within widestDrawn digits, so the difference stays far inside
// what an int64 holds.
func (s span) size() int64 { return s.high - s.low + 1 }
