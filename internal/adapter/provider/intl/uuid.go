package intl

import (
	"fmt"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// UUID is the universally unique identifier of RFC 9562, drawn in the version
// made of random bits.
//
// # Where the rules come from
//
// RFC 9562 of May 2024, which replaced RFC 4122, read at rfc-editor.org on
// 2026-11-25. Section 4 gives the written form: thirty two hexadecimal digits in
// groups of eight, four, four, four and twelve, joined by hyphens, in either
// case. Section 4.1 puts the variant in the most significant bits of octet 8 and
// section 4.2 the version in the most significant four bits of octet 6. Section
// 5.4 says a version 4 value is random bits with those two fields overwritten,
// and Appendix A.3 works one value through from its random bytes to the final
// text. That example and the example values of the other versions are held by
// the tests beside this file.
//
// # Why version 4 and nothing else
//
// It is the version that carries nothing but chance: no moment, no machine and
// no name. A time ordered version would write a moment into every value, and
// to keep a run reproducible that moment would have to come from the seed, a
// time at which nothing happened. Validate accepts every version the RFC
// defines, because a column somebody else wrote is read as UUIDs and not as
// the UUIDs of this build.
//
// # Why lowercase
//
// The RFC takes either case. Its own examples of version 4 and 5 and the one in
// section 4 are written in lowercase, so a value from here reads like the text
// of the standard.
//
// # What this scheme does not claim
//
// No check digit and no reserved range. A UUID has no character computed from
// the others, so one wrong character cannot be caught by the value itself, and
// there is no block of values set aside for tests: the RFC reserves the Nil and
// Max values for meanings of their own (sections 5.9 and 5.10), not for data
// nobody owns. Being random, a value from here belongs to nobody in the sense
// the identifiers of people do.
type UUID struct{}

// Contract checks.
var (
	_ provider.Provider = UUID{}
	_ provider.Unissued = UUID{}
)

const (
	// uuidLength is the written form of section 4: thirty two digits and four
	// hyphens.
	uuidLength = 36
	// uuidOctets is the size of a UUID, sixteen octets.
	uuidOctets = 16
	// uuidVersionOctet and uuidVariantOctet are where section 4.2 and section
	// 4.1 put the two fields.
	uuidVersionOctet = 6
	uuidVariantOctet = 8
	// uuidVersionAt and uuidVariantAt are the characters of the written form
	// those octets begin at: the first digit of the third group and of the
	// fourth.
	uuidVersionAt = 14
	uuidVariantAt = 19
	// uuidVersion4 is version 4 in the high half of octet 6, and uuidLowHalf
	// keeps the half below it.
	uuidVersion4 = 0x40
	uuidLowHalf  = 0x0f
	// uuidVariant10 is the variant of this RFC, the bits 1 and 0 at the top of
	// octet 8, and uuidBelowVariant keeps the six bits below them.
	uuidVariant10    = 0x80
	uuidBelowVariant = 0x3f
	// uuidHex is the alphabet of the written form in the case this build
	// writes.
	uuidHex = "0123456789abcdef"
	// uuidMaxVersion is the highest version section 4.2 defines. Nought is
	// unused and nine to fifteen are reserved for future definition.
	uuidMaxVersion = 8
	// The first digit of the fourth group says the variant, from Table 1 of
	// section 4.1: below 8 the old Network Computing System, 8 to b this RFC,
	// c and d Microsoft's older identifiers, e and f reserved for the future.
	uuidVariantThisRFC   = 0x8
	uuidVariantMicrosoft = 0xc
	uuidVariantFuture    = 0xe
)

// uuidGroups are the lengths of the five groups of digits of section 4.
var uuidGroups = [...]int{8, 4, 4, 4, 12}

// ID names this provider. A UUID belongs to no country, so the name carries no
// prefix.
func (UUID) ID() string { return "uuid" }

// Capabilities are what this provider brings, and it brings none: nothing in
// the registry describes a part of this value.
func (UUID) Capabilities() []capability.Name { return nil }

// Unissued is true: nobody hands out UUIDs. Whoever needs one draws it, and a
// version 4 value is 122 random bits (section 5.4), so the panel beside the
// values says that rather than telling a person to treat one as real data.
func (UUID) Unissued() bool { return true }

// Masks are thirty two places that each take a digit or a letter, in the five
// groups of section 4.
//
// The version 4 every drawn value carries is a place like the others here,
// although it never varies. Written as a fixed 4, the panel beside the values
// counted 31 letters or digits on 2026-11-25, and a person counting the 32 of a
// value would take the panel for wrong. The mask does not say either that the
// letters are a to f: a mask carrying that would be a second copy of Validate.
func (UUID) Masks() []string {
	groups := make([]string, 0, len(uuidGroups))
	for _, length := range uuidGroups {
		groups = append(groups, strings.Repeat(string(provider.MaskAlnum), length))
	}
	return []string{strings.Join(groups, "-")}
}

// Generate produces one version 4 UUID for the place described by the request.
//
// The agreement is not read. A UUID is chance and nothing else, so there is no
// column of the row it could follow.
func (UUID) Generate(req provider.Request) (string, error) {
	digest := req.Source.Bytes(seed.Coordinate{
		Entity: req.At.Entity, Row: req.At.Row, Field: req.At.Field,
		Purpose: req.At.Purpose + ":uuid",
	})
	var random [uuidOctets]byte
	copy(random[:], digest[:uuidOctets])
	return uuidFromRandom(random), nil
}

// uuidFromRandom is section 5.4 applied to sixteen random octets: the version
// and the variant overwrite their bits and the rest is written as section 4
// writes it.
func uuidFromRandom(random [uuidOctets]byte) string {
	random[uuidVersionOctet] = random[uuidVersionOctet]&uuidLowHalf | uuidVersion4
	random[uuidVariantOctet] = random[uuidVariantOctet]&uuidBelowVariant | uuidVariant10
	written := make([]byte, 0, uuidLength)
	for _, octet := range random {
		if isUUIDHyphen(len(written)) {
			written = append(written, '-')
		}
		written = append(written, uuidHex[octet>>4], uuidHex[octet&uuidLowHalf])
	}
	return string(written)
}

// Validate answers whether a value is a UUID of RFC 9562 in its written form.
//
// It is written from the sections of the RFC rather than from what Generate
// produces. Every version the RFC defines is valid, and so are the Nil and Max
// values of sections 5.9 and 5.10. A value of another variant is a UUID this
// build recognises and does not check, because section 4.1 leaves those layouts
// to the systems that reserved them.
func (UUID) Validate(value string) provider.Verdict {
	if len(value) != uuidLength {
		return provider.Invalid(fmt.Sprintf(
			"a UUID is 36 characters, 32 hexadecimal digits and 4 hyphens, and %s is %d",
			provider.Show(value), len(value)))
	}
	for at := range len(value) {
		if isUUIDHyphen(at) {
			if value[at] != '-' {
				return provider.Invalid(fmt.Sprintf(
					"character %d of %s should be a hyphen, which splits a UUID into groups of 8, 4, 4, 4 and 12 digits",
					at+1, provider.Show(value)))
			}
			continue
		}
		if hexValue(value[at]) < 0 {
			return provider.Invalid(fmt.Sprintf(
				"character %d of %s is not a hexadecimal digit, 0 to 9 or a to f",
				at+1, provider.Show(value)))
		}
	}
	digits := strings.ReplaceAll(strings.ToLower(value), "-", "")
	if digits == strings.Repeat("0", 2*uuidOctets) || digits == strings.Repeat("f", 2*uuidOctets) {
		return provider.Valid()
	}
	variant := hexValue(value[uuidVariantAt])
	switch {
	case variant < uuidVariantThisRFC:
		return provider.Unsupported("the variant of " + provider.Show(value) +
			" is the one kept for the old Network Computing System, which this build does not check")
	case variant >= uuidVariantMicrosoft && variant < uuidVariantFuture:
		return provider.Unsupported("the variant of " + provider.Show(value) +
			" is the one kept for Microsoft's older identifiers, which this build does not check")
	case variant >= uuidVariantFuture:
		return provider.Unsupported("the variant of " + provider.Show(value) +
			" is reserved for future definition, so this build does not check it")
	}
	if version := hexValue(value[uuidVersionAt]); version < 1 || version > uuidMaxVersion {
		return provider.Invalid(fmt.Sprintf(
			"%s says version %d, and the versions of a UUID are 1 to 8", provider.Show(value), version))
	}
	return provider.Valid()
}

// isUUIDHyphen says whether a place of the written form holds a hyphen: the
// one after each group but the last.
func isUUIDHyphen(at int) bool {
	end := 0
	for _, length := range uuidGroups[:len(uuidGroups)-1] {
		end += length
		if at == end {
			return true
		}
		end++
	}
	return false
}

// hexValue is the value of one hexadecimal digit in either case, or -1.
func hexValue(digit byte) int {
	switch {
	case digit >= '0' && digit <= '9':
		return int(digit - '0')
	case digit >= 'a' && digit <= 'f':
		return int(digit-'a') + 10
	case digit >= 'A' && digit <= 'F':
		return int(digit-'A') + 10
	}
	return -1
}
