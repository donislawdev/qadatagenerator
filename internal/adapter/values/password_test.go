package values_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
)

// The characters the owner chose on 2026-11-25 (DEC-340), written here a second
// time on purpose: a character slipping into or out of the generator's alphabet
// has to turn this red.
const (
	chosenSymbols = "!#$%&*+-=?@^_"
	chosenLetters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	chosenDigits  = "0123456789"
)

// Every password holds a lowercase letter, an uppercase letter, a digit and a
// symbol, only the chosen characters, twelve to sixteen of them, never begins
// with what a spreadsheet reads as a formula, and is the same at the same place.
func TestEveryPasswordHoldsEveryClassAndOnlyTheChosenCharacters(t *testing.T) {
	source := values.New("en-US", "passwords")
	lengths := map[int]int{}
	startsWithSymbol := 0
	for row := range int64(2000) {
		req := ask(values.Password, "password", row)
		value, err := source.Produce(req)
		if err != nil {
			t.Fatalf("row %d: %v", row, err)
		}
		if again, _ := source.Produce(req); again != value {
			t.Fatalf("row %d gave %q and then %q at the same place", row, value, again)
		}
		lengths[len(value)]++
		if strings.ContainsAny(value[:1], "=+-@") {
			t.Fatalf("row %d: %q begins with a character a spreadsheet takes for a formula", row, value)
		}
		if strings.ContainsAny(value[:1], chosenSymbols) {
			startsWithSymbol++
		}
		for _, class := range []string{"abcdefghijklmnopqrstuvwxyz", "ABCDEFGHIJKLMNOPQRSTUVWXYZ", chosenDigits, chosenSymbols} {
			if !strings.ContainsAny(value, class) {
				t.Fatalf("row %d: %q holds nothing of %q", row, value, class)
			}
		}
		for _, r := range value {
			if !strings.ContainsRune(chosenLetters+chosenDigits+chosenSymbols, r) {
				t.Fatalf("row %d: %q holds %q, which is not among the chosen characters", row, value, r)
			}
		}
	}
	for length := 12; length <= 16; length++ {
		if lengths[length] == 0 {
			t.Errorf("no password of %d characters in 2000 rows: %v", length, lengths)
		}
	}
	if len(lengths) != 5 {
		t.Errorf("the lengths drawn are %v, and 12 to 16 were wanted", lengths)
	}
	// The symbols that are not a formula may still begin a password, so the
	// rule above moved four characters and not the whole class.
	if startsWithSymbol == 0 {
		t.Error("no password in 2000 begins with a symbol, so the rule about formulas moved every symbol")
	}
}

// max_length shortens a password down to the four characters that hold one of
// each class, and below that the field is refused before a row is written, with
// the sentence the draw would give.
func TestMaxLengthShortensAPasswordAndBelowFourItIsRefused(t *testing.T) {
	source := values.New("en-US", "passwords")
	for _, limit := range []int{4, 8, 13} {
		for row := range int64(200) {
			req := ask(values.Password, "password", row)
			req.HasMaxLength, req.MaxLength, req.Unit = true, limit, "characters"
			value, err := source.Produce(req)
			if err != nil {
				t.Fatalf("max_length %d, row %d: %v", limit, row, err)
			}
			if len(value) > limit || len(value) < min(12, limit) {
				t.Fatalf("max_length %d, row %d: %q is %d characters", limit, row, value, len(value))
			}
		}
	}

	short := ask(values.Password, "password", 0)
	short.HasMaxLength, short.MaxLength, short.Unit = true, 3, "characters"
	if _, err := source.Produce(short); err == nil || !strings.Contains(err.Error(), "at least 4") {
		t.Errorf("max_length 3 was answered with %v, which does not say a password needs 4", err)
	}
	domain, known := source.DomainOf(short)
	if !known || domain.Size != 0 {
		t.Errorf("below four the plan is told known=%v and %d values, and it has to be told there are none", known, domain.Size)
	}
	if _, known := source.DomainOf(ask(values.Password, "password", 0)); known {
		t.Error("an unlimited password claims a domain the plan could count, and it is far larger than any table")
	}
	if _, err := source.Produce(generate.Request{Generator: values.Password, At: at("password", 1)}); err != nil {
		t.Errorf("a password with no limit was refused: %v", err)
	}
}
