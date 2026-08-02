package us

import (
	"strings"
	"testing"
)

// The writer is reached from inside this package only, so it is tested from
// inside it. The alternative would be a test in the outer package pretending
// to drive it through a door that does not exist, which passes without
// checking anything.

// Silent truncation is the failure worth guarding here. A group of one hundred
// written into two digits would come out as 00, which is one of the two the
// scheme calls invalid, so the tool would produce a broken number with every
// line of it reading correctly.
func TestAPartTooLargeForItsWidthIsRefusedRatherThanTruncated(t *testing.T) {
	cases := []struct {
		name  string
		value int
		width int
	}{
		{"a group of one hundred", 100, groupDigits},
		{"an area of one thousand", 1000, areaDigits},
		{"a serial of ten thousand", 10000, serialDigits},
		{"a negative part", -1, groupDigits},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered == nil {
					t.Fatalf("%d was written into %d digits instead of refused", test.value, test.width)
				}
			}()
			var built strings.Builder
			writeDigits(&built, test.value, test.width)
			t.Fatalf("wrote %q", built.String())
		})
	}
}

// The padding itself, since the guard above only says what does not happen.
func TestAPartIsPaddedToItsFullWidth(t *testing.T) {
	cases := map[string]struct {
		value int
		width int
	}{
		"000":  {0, areaDigits},
		"007":  {7, areaDigits},
		"666":  {666, areaDigits},
		"01":   {1, groupDigits},
		"99":   {99, groupDigits},
		"0001": {1, serialDigits},
		"9999": {9999, serialDigits},
	}

	for want, test := range cases {
		var built strings.Builder
		writeDigits(&built, test.value, test.width)
		if got := built.String(); got != want {
			t.Errorf("%d in %d digits became %q, expected %q", test.value, test.width, got, want)
		}
	}
}

// Reading a value and writing it back has to return the same text, or the
// corruption would report a change it did not make. Both spellings, since both
// are accepted.
func TestReadingAndWritingANumberReturnsTheSameText(t *testing.T) {
	for _, value := range []string{
		"666-12-3456", "666123456", "000-00-0000", "000000000", "999-99-9999",
	} {
		parsed, ok := parse(value)
		if !ok {
			t.Errorf("%q was not read at all", value)
			continue
		}
		if got := parsed.String(); got != value {
			t.Errorf("%q was read and written back as %q", value, got)
		}
	}
}
