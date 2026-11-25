package intl

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
	"github.com/donislawdev/qadatagenerator/internal/oracle"
)

// rfcExampleValues are the example values RFC 9562 writes out, read at
// rfc-editor.org on 2026-11-25: Figure 1 of section 4, the final value of every
// example in Appendix A and Appendix B, and the Nil and Max values of sections
// 5.9 and 5.10. Their case is the RFC's.
var rfcExampleValues = map[string]string{
	"section 4, figure 1, version 1": "f81d4fae-7dec-11d0-a765-00a0c91e6bf6",
	"A.1, version 1":                 "C232AB00-9414-11EC-B3C8-9F6BDECED846",
	"A.2, version 3":                 "5df41881-3aed-3515-88a7-2f4a814cf09e",
	"A.3, version 4":                 "919108f7-52d1-4320-9bac-f847db4148a8",
	"A.4, version 5":                 "2ed6657d-e927-568b-95e1-2665a8aea6a2",
	"A.5, version 6":                 "1EC9414C-232A-6B00-B3C8-9F6BDECED846",
	"A.6, version 7":                 "017F22E2-79B0-7CC3-98C4-DC0C0C07398F",
	"B.1, version 8":                 "2489E9AD-2EE2-8E00-8EC9-32D5F69181C0",
	"B.2, version 8":                 "5c146b14-3c52-8afd-938a-375d0df1fbf6",
	"5.9, Nil":                       "00000000-0000-0000-0000-000000000000",
	"5.10, Max":                      "FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF",
}

// Appendix A.3 of RFC 9562 works a version 4 value through from sixteen random
// octets to its written form, overwriting the version and the variant. The same
// octets have to come out as the same text here. This is the anchor of the
// written form and of the two fields Generate overwrites.
func TestTheRFCExampleOfAVersionFourValueIsReproduced(t *testing.T) {
	octets, err := hex.DecodeString("919108F752D133205BACF847DB4148A8")
	if err != nil {
		t.Fatal(err)
	}
	var random [uuidOctets]byte
	copy(random[:], octets)
	if got, want := uuidFromRandom(random), "919108f7-52d1-4320-9bac-f847db4148a8"; got != want {
		t.Errorf("the random octets of A.3 were written as %s, and the RFC writes %s", got, want)
	}
}

// Every example value of the RFC is a valid UUID to this build and to the
// independent reading, whatever its version and its case. This is the anchor of
// the version and the variant as Validate reads them.
func TestEveryExampleValueOfTheRFCIsAccepted(t *testing.T) {
	for where, value := range rfcExampleValues {
		if verdict := (UUID{}).Validate(value); !verdict.Valid {
			t.Errorf("%s, %s: refused (%s)", where, value, verdict.Reason)
		}
		if findings := oracle.CheckUUID(value); len(findings) != 0 {
			t.Errorf("%s, %s: the independent reading finds %v", where, value, findings)
		}
	}
}

// What the RFC does not lay out is refused with a reason, a layout it leaves to
// others is recognised and not checked, and on every value the provider and the
// independent reading agree about whether it is a UUID of this RFC.
func TestValidateRefusesWhatTheRFCDoesNotLayOut(t *testing.T) {
	cases := []struct {
		value string
		want  string // valid, invalid or unsupported
	}{
		{"919108F7-52D1-4320-9BAC-F847DB4148A8", "valid"},
		{"919108f7-52d1-4320-9bac-f847db4148a", "invalid"},
		{"919108f7-52d1-4320-9bac-f847db4148a8a", "invalid"},
		{"919108f752d1-4320-9bac-f847db4148a8-", "invalid"},
		{"919108f7_52d1-4320-9bac-f847db4148a8", "invalid"},
		{"919108g7-52d1-4320-9bac-f847db4148a8", "invalid"},
		{"{919108f7-52d1-4320-9bac-f847db4148a8}", "invalid"},
		{"919108f752d143209bacf847db4148a8", "invalid"},
		{"919108f7-52d1-0320-9bac-f847db4148a8", "invalid"},
		{"919108f7-52d1-9320-9bac-f847db4148a8", "invalid"},
		{"919108f7-52d1-f320-9bac-f847db4148a8", "invalid"},
		{"919108f7-52d1-4320-7bac-f847db4148a8", "unsupported"},
		{"919108f7-52d1-4320-cbac-f847db4148a8", "unsupported"},
		{"919108f7-52d1-4320-ebac-f847db4148a8", "unsupported"},
		{"", "invalid"},
	}
	for _, each := range cases {
		verdict := (UUID{}).Validate(each.value)
		got := "invalid"
		switch {
		case verdict.Valid:
			got = "valid"
		case verdict.Unsupported:
			got = "unsupported"
		}
		if got != each.want {
			t.Errorf("%q came out %s (%s), and %s was wanted", each.value, got, verdict.Reason, each.want)
		}
		if !verdict.Valid && verdict.Reason == "" {
			t.Errorf("%q was turned away without a reason", each.value)
		}
		if lays := len(oracle.CheckUUID(each.value)) == 0; lays != verdict.Valid {
			t.Errorf("%q: the provider says valid=%v and the independent reading says %v",
				each.value, verdict.Valid, oracle.CheckUUID(each.value))
		}
	}
}

// Every value drawn is version 4 of this RFC in lowercase, the same at the same
// place and different at different places. Two thousand rows of 122 random
// bits cannot collide by chance, so a repeat here is a fault of the draw.
func TestEveryDrawnValueIsAVersionFourUUID(t *testing.T) {
	source := seed.New("uuid")
	seen := make(map[string]int64, 2000)
	for row := range int64(2000) {
		req := provider.Request{Source: source, At: seed.Coordinate{Entity: "e", Row: row, Field: "id", Purpose: "value"}}
		value, err := (UUID{}).Generate(req)
		if err != nil {
			t.Fatalf("row %d: %v", row, err)
		}
		again, _ := (UUID{}).Generate(req)
		if again != value {
			t.Fatalf("row %d gave %s and then %s at the same place", row, value, again)
		}
		if verdict := (UUID{}).Validate(value); !verdict.Valid {
			t.Fatalf("row %d: %s refused (%s)", row, value, verdict.Reason)
		}
		if findings := oracle.CheckUUID(value); len(findings) != 0 {
			t.Fatalf("row %d: %s, the independent reading finds %v", row, value, findings)
		}
		if value[uuidVersionAt] != '4' || !strings.ContainsRune("89ab", rune(value[uuidVariantAt])) {
			t.Fatalf("row %d: %s is not version 4 with the variant of the RFC", row, value)
		}
		if value != strings.ToLower(value) {
			t.Fatalf("row %d: %s is not lowercase", row, value)
		}
		if earlier, twice := seen[value]; twice {
			t.Fatalf("rows %d and %d both gave %s", earlier, row, value)
		}
		seen[value] = row
	}
}

// The independent reading has to refuse what it exists to refuse, or the
// agreement above could be two readings that accept everything.
func TestTheIndependentReadingOfAUUIDCanRefuse(t *testing.T) {
	for _, value := range []string{"not a uuid", "919108f7-52d1-0320-9bac-f847db4148a8", "919108f7-52d1-4320-cbac-f847db4148a8"} {
		if len(oracle.CheckUUID(value)) == 0 {
			t.Errorf("canary: the independent reading accepted %q", value)
		}
	}
}
