package oracle

import (
	"encoding/hex"
	"regexp"
	"strings"
)

// The rule this file implements.
//
// RFC 9562, Universally Unique IDentifiers, May 2024, read at rfc-editor.org on
// 2026-11-25. Section 4 gives the written form as ABNF, section 4.1 the variant
// in Table 1 and section 4.2 the version in Table 2, and sections 5.9 and 5.10
// the Nil and Max values. Appendix A works an example value for most versions,
// and the tests of the provider hold every one of them.
//
// The provider walks the characters one at a time and reads two of them as the
// version and the variant. This reads the written form with a pattern, decodes
// it into octets and reads the bits of the two octets the sections name, so a
// place counted wrong on one side is not counted wrong on the other.

// uuidWritten is the ABNF of section 4: groups of 4, 2, 2, 2 and 6 hex octets
// joined by single hyphens, the hex digits in either case.
var uuidWritten = regexp.MustCompile(`^[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}$`)

// UUIDFinding is one reason a value is not a UUID of RFC 9562 in its written
// form, in the layout that RFC defines. An empty list means it is one.
type UUIDFinding string

// The findings this oracle can return.
const (
	UUIDNotWritten UUIDFinding = "not the written form of section 4"
	UUIDNotVariant UUIDFinding = "not the variant of section 4.1 with the bits 1 and 0"
	UUIDNoVersion  UUIDFinding = "not a version defined in section 4.2"
)

// CheckUUID returns every reason value is not a UUID this RFC lays out. The
// Nil and Max values are UUIDs of their own (sections 5.9 and 5.10), although
// their variant bits are not 1 and 0.
func CheckUUID(value string) []UUIDFinding {
	if !uuidWritten.MatchString(value) {
		return []UUIDFinding{UUIDNotWritten}
	}
	octets, err := hex.DecodeString(strings.ReplaceAll(value, "-", ""))
	if err != nil {
		return []UUIDFinding{UUIDNotWritten}
	}
	if allOctets(octets, 0x00) || allOctets(octets, 0xff) {
		return nil
	}
	var findings []UUIDFinding
	// Bits 64 and 65, the top two of octet 8, are 1 and 0 (section 4.1).
	if octets[8]>>6 != 0b10 {
		findings = append(findings, UUIDNotVariant)
	}
	// Bits 48 to 51, the top four of octet 6 (section 4.2). Table 2 defines 1
	// to 8, calls 0 unused and keeps 9 to 15 for future definition.
	if version := octets[6] >> 4; version < 1 || version > 8 {
		findings = append(findings, UUIDNoVersion)
	}
	return findings
}

// allOctets says whether every octet is the same one.
func allOctets(octets []byte, every byte) bool {
	for _, octet := range octets {
		if octet != every {
			return false
		}
	}
	return true
}
