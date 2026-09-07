package intl_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/intl"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
	"github.com/donislawdev/qadatagenerator/internal/oracle"
)

// theManualsOwnNumber is the calculation the ISBN Users' Manual works in
// appendix A1.1 of its seventh edition, with the check digit it arrives at put
// back in last place.
//
// The manual writes it as 978-92-95055-12-4. It is the manual's own ISBN, printed
// on its imprint page, so the worked example and the document it appears in
// check each other without anything of ours in between.
const theManualsOwnNumber = "9789295055124"

// theManualsPrintedProducts are the twelve weighted products the manual prints
// in the table of that appendix, in the order it prints them.
//
// This is the reason this anchor is the firmest in the build. A source that
// gives an answer lets a wrong sum and a wrong fold cancel out. A source that
// gives the total as well pins both halves. This one gives every multiplication
// on the way to the total, so a single weight applied to the wrong digit cannot
// hide inside a total that still comes out right.
var theManualsPrintedProducts = []int{9, 21, 8, 27, 2, 27, 5, 0, 5, 15, 1, 6}

// theManualsPrintedSum is the total the manual prints at the end of that row,
// and theManualsPrintedRemainder is what it prints at step 2.
const (
	theManualsPrintedSum       = 126
	theManualsPrintedRemainder = 6
	theManualsPrintedDigit     = '4'
)

// generateISBN produces the number for one row, failing the test rather than
// handing back an error nobody looks at.
func generateISBN(t *testing.T, source seed.Source, row int64) string {
	t.Helper()
	value, err := (intl.ISBN{}).Generate(provider.Request{
		Source: source,
		At:     seed.Coordinate{Entity: "library", Row: row, Field: "isbn", Purpose: "value"},
	})
	if err != nil {
		t.Fatalf("row %d: the generator refused: %v", row, err)
	}
	return value
}

// The manual does the arithmetic itself and shows every link of the chain. This
// walks the same chain and stops at each printed number rather than only at the
// answer.
func TestTheExampleTheManualWorksComesOutTheSameHere(t *testing.T) {
	body := theManualsOwnNumber[:12]

	if len(theManualsPrintedProducts) != len(body) {
		t.Fatalf("the manual prints %d products for %d digits", len(theManualsPrintedProducts), len(body))
	}

	// Link one: every individual multiplication, taken as the manual counts the
	// alternation, which is from the left starting light.
	sum := 0
	for at := 0; at < len(body); at++ {
		weight := 1
		if at%2 == 1 {
			weight = 3
		}
		product := int(body[at]-'0') * weight
		if product != theManualsPrintedProducts[at] {
			t.Errorf("digit %d of the worked example: the manual prints the product %d and these weights give %d",
				at+1, theManualsPrintedProducts[at], product)
		}
		sum += product
	}

	// Link two: the total.
	if sum != theManualsPrintedSum {
		t.Errorf("the manual prints a sum of %d and the same weights give %d", theManualsPrintedSum, sum)
	}

	// Link three: the remainder it prints before folding it.
	if remainder := sum % 10; remainder != theManualsPrintedRemainder {
		t.Errorf("the manual prints a remainder of %d and this gives %d", theManualsPrintedRemainder, remainder)
	}

	// Link four: the digit, reached through the provider rather than repeated
	// here, so the thing under test is the thing that ships.
	rebuilt, err := (intl.ISBN{}).WithCheckDigits(theManualsOwnNumber)
	if err != nil {
		t.Fatalf("recomputing the check digit of the worked example failed: %v", err)
	}
	if rebuilt[12] != theManualsPrintedDigit {
		t.Errorf("the manual reaches the check digit %q and this reaches %q",
			theManualsPrintedDigit, rebuilt[12])
	}
	if rebuilt != theManualsOwnNumber {
		t.Errorf("rebuilding the worked example changed it from %s to %s", theManualsOwnNumber, rebuilt)
	}
	if verdict := (intl.ISBN{}).Validate(theManualsOwnNumber); !verdict.Valid {
		t.Errorf("the number the manual prints for itself was refused: %s", verdict.Reason)
	}
}

// The example the previous edition works has to come out the same, because that
// is the claim which lets this build keep citing it for the article number.
//
// If the two editions ever disagreed, the older citation would be a statement
// about a superseded rule and would have to be withdrawn rather than kept.
func TestTheEarlierEditionsExampleStillAgreesWithTheCurrentRule(t *testing.T) {
	const earlier = "9780110002224"
	const earlierPrintedSum = 56

	sum := 0
	for at := 0; at < 12; at++ {
		weight := 1
		if at%2 == 1 {
			weight = 3
		}
		sum += int(earlier[at]-'0') * weight
	}
	if sum != earlierPrintedSum {
		t.Errorf("the sixth edition prints a sum of %d and the current rule gives %d", earlierPrintedSum, sum)
	}
	if verdict := (intl.ISBN{}).Validate(earlier); !verdict.Valid {
		t.Errorf("the example of the earlier edition was refused: %s", verdict.Reason)
	}
}

// Every value this build produces has to satisfy the validator written from the
// manual's other sentence rather than from the generator.
func TestEveryGeneratedBookNumberSatisfiesTheIndependentValidator(t *testing.T) {
	source := seed.New("book-numbers")
	for row := int64(0); row < 3000; row++ {
		value := generateISBN(t, source, row)
		if findings := oracle.CheckISBN(value); len(findings) != 0 {
			t.Fatalf("row %d produced %s and the independent validator refused it: %v", row, value, findings)
		}
		if verdict := (intl.ISBN{}).Validate(value); !verdict.Valid {
			t.Fatalf("row %d produced %s and the provider refused its own value: %s", row, value, verdict.Reason)
		}
	}
}

// The narrowing the type describes is the only promise generation makes, so it
// is the one thing a test has to hold it to.
func TestEveryGeneratedBookNumberSitsInTheUnallocatedWindow(t *testing.T) {
	source := seed.New("window")
	for row := int64(0); row < 3000; row++ {
		value := generateISBN(t, source, row)
		if !strings.HasPrefix(value, "978") {
			t.Fatalf("row %d produced %s, which is outside the prefix this scheme generates in", row, value)
		}
		window := value[3:10]
		if window < "6700000" || window > "6998999" {
			t.Fatalf("row %d produced %s, whose registration window %s is outside the range the agency leaves undefined",
				row, value, window)
		}
	}
}

// The narrowing is not a harbour and the provider must not say that it is.
//
// This test is named for what the provider does not claim, because a reader who
// sees every generated value sharing a prefix will assume the opposite. The
// equipment identity carries the same guard for the same reason.
func TestTheBookNumberClaimsNoSafeHarbour(t *testing.T) {
	for _, declared := range (intl.ISBN{}).Capabilities() {
		if declared == capability.IdentifierReservedRange {
			t.Fatal("this provider declares a reserved range.\n" +
				"It generates inside a window the agency has not allocated, which is a narrowing and not a harbour.\n" +
				"The agency allocates into that space over time, so nothing here can promise that a value cannot\n" +
				"belong to a real book. Declaring the range would put that promise into a manifest.")
		}
	}
	if _, claims := interface{}((intl.ISBN{})).(provider.RangeAware); claims {
		t.Fatal("this provider satisfies RangeAware, which is the interface a scheme with a harbour implements")
	}
}

// A music number is a real identifier of something and is not a book number.
// Saying which of the two it is beats saying only that it is not a book.
func TestAMusicNumberIsRefusedAndSaidToBeAMusicNumber(t *testing.T) {
	// Built rather than quoted, so this test carries no claim about any real
	// publication. The arithmetic is satisfied deliberately, because a value
	// refused for its check digit would prove nothing about the prefix.
	body := "979012345678"
	sum := 0
	for at := 0; at < len(body); at++ {
		weight := 1
		if at%2 == 1 {
			weight = 3
		}
		sum += int(body[at]-'0') * weight
	}
	value := body + string(byte('0'+(10-sum%10)%10))

	if findings := oracle.CheckISBN(value); len(findings) == 0 {
		t.Error("the independent validator accepted a number in the music prefix")
	}
	verdict := (intl.ISBN{}).Validate(value)
	if verdict.Valid {
		t.Fatal("a number beginning 9790 was accepted as a book number")
	}
	if !strings.Contains(strings.ToLower(verdict.Reason), "music") {
		t.Errorf("the refusal does not say the value is a music number: %s", verdict.Reason)
	}
}

// The prefix the two agencies share is read out of the sentence the music
// agency publishes, rather than typed twice.
func TestTheMusicPrefixIsTheOneItsAgencyPublishes(t *testing.T) {
	if got := oracle.ISMNPublishedPrefix(); got != "9790" {
		t.Errorf("the published sentence yields the prefix %q rather than 9790", got)
	}
	if sentence := oracle.ISBNPublishedLengthSentence(); !strings.Contains(sentence, "13 digits") {
		t.Errorf("the agency's own sentence about length no longer names thirteen digits: %s", sentence)
	}
}

// Thirteen noughts satisfy the arithmetic and are still not a book number.
//
// The article number in this package accepts them, and correctly so, because
// nothing about a GTIN-13 says which prefixes exist. This scheme has a prefix
// rule, so the same value has to come out differently here. That contrast is
// the point of the test.
func TestThirteenNoughtsPassTheArithmeticAndAreStillNotABookNumber(t *testing.T) {
	const noughts = "0000000000000"

	if verdict := (intl.EAN{}).Validate(noughts); !verdict.Valid {
		t.Errorf("the article number scheme refused thirteen noughts, so this contrast no longer holds: %s", verdict.Reason)
	}
	verdict := (intl.ISBN{}).Validate(noughts)
	if verdict.Valid {
		t.Fatal("thirteen noughts were accepted as a book number")
	}
	if !strings.Contains(verdict.Reason, "978") {
		t.Errorf("the refusal does not name the prefixes a book number carries: %s", verdict.Reason)
	}
}

// Generation is narrow and validation is wide, which is the shape every scheme
// here with a narrowed generator has. A real book number from outside the
// window must not be called wrong.
func TestValidateAcceptsBookNumbersOutsideTheWindowThisSchemeUses(t *testing.T) {
	// Bodies chosen in allocated registration groups, completed by the
	// arithmetic rather than quoted from any catalogue.
	for _, body := range []string{"978030640615", "978014143960", "979101234567", "978999219999"} {
		sum := 0
		for at := 0; at < len(body); at++ {
			weight := 1
			if at%2 == 1 {
				weight = 3
			}
			sum += int(body[at]-'0') * weight
		}
		value := body + string(byte('0'+(10-sum%10)%10))

		if verdict := (intl.ISBN{}).Validate(value); !verdict.Valid {
			t.Errorf("%s is a well formed book number outside the generator's window and was refused: %s",
				value, verdict.Reason)
		}
		if findings := oracle.CheckISBN(value); len(findings) != 0 {
			t.Errorf("%s was refused by the independent validator: %v", value, findings)
		}
	}
}

// Moving one digit has to break the number, which is what the catalog's check
// digit corruption relies on.
func TestMovingASingleDigitBreaksTheBookCheckDigit(t *testing.T) {
	source := seed.New("moved")
	for row := int64(0); row < 200; row++ {
		value := generateISBN(t, source, row)
		for at := 0; at < len(value); at++ {
			moved := []byte(value)
			moved[at] = '0' + (moved[at]-'0'+1)%10
			if verdict := (intl.ISBN{}).Validate(string(moved)); verdict.Valid {
				t.Fatalf("moving position %d of %s on by one left it valid as %s", at+1, value, moved)
			}
		}
	}
}

// Recomputing must never refuse for arithmetic reasons, because a mutator that
// moved a digit needs a consistent value back every time.
//
// Two claims live here and the first draft of this test ran them together and
// failed, which was the test being wrong rather than the code. Repairing a
// value restores the arithmetic. It does not restore the prefix, and it must
// not, because a corruption that moved the first digit produced something that
// is genuinely no longer a book number and pretending otherwise would hide the
// corruption the mutator just made. So the wide claim is made where the prefix
// survived, and the narrow one everywhere.
func TestRecomputingTheBookCheckDigitNeverRefuses(t *testing.T) {
	source := seed.New("rebuild")
	for row := int64(0); row < 500; row++ {
		value := generateISBN(t, source, row)
		for at := 0; at < len(value); at++ {
			moved := []byte(value)
			moved[at] = '0' + (moved[at]-'0'+1)%10
			rebuilt, err := (intl.ISBN{}).WithCheckDigits(string(moved))
			if err != nil {
				t.Fatalf("recomputing %s failed: %v", moved, err)
			}
			if len(rebuilt) != len(value) {
				t.Fatalf("%s was rebuilt into %s, which is a different length", moved, rebuilt)
			}
			// The arithmetic has to close whatever the prefix now says, and
			// that is asked of the rebuilt value directly.
			total := 0
			for index := 0; index < len(rebuilt); index++ {
				weight := 1
				if index%2 == 1 {
					weight = 3
				}
				total += int(rebuilt[index]-'0') * weight
			}
			if total%10 != 0 {
				t.Fatalf("%s was rebuilt into %s, whose weighted sum %d does not divide by ten", moved, rebuilt, total)
			}
			if at < 3 {
				// The prefix was the thing that moved, so the value is not a
				// book number any more and the full validator is right to say so.
				continue
			}
			if findings := oracle.CheckISBN(rebuilt); len(findings) != 0 {
				t.Fatalf("%s was rebuilt into %s, which the independent validator refuses: %v", moved, rebuilt, findings)
			}
		}
	}
}

// The corruption engine asks where the check digit is and has to be told the
// last position.
func TestTheBookCheckDigitIsReportedAtTheLastPosition(t *testing.T) {
	positions, ok := (intl.ISBN{}).CheckDigits(theManualsOwnNumber)
	if !ok {
		t.Fatal("the provider will not say where the check digit of a correct number is")
	}
	if len(positions) != 1 || positions[0] != 12 {
		t.Errorf("the check digit was reported at %v rather than at position twelve counted from zero", positions)
	}
	if _, ok := (intl.ISBN{}).CheckDigits("978"); ok {
		t.Error("a value of the wrong length was given a check digit position")
	}
}

// Shapes this scheme does not have have to be refused, each with the reason
// that applies to it.
func TestTheShapesABookNumberDoesNotHaveAreRefused(t *testing.T) {
	for _, test := range []struct {
		value string
		names string
	}{
		{"", "thirteen"},
		{"978929505512", "thirteen"},
		{"97892950551244", "thirteen"},
		{"97892950551X4", "not a digit"},
		{"1234567890128", "978"},
	} {
		verdict := (intl.ISBN{}).Validate(test.value)
		if verdict.Valid {
			t.Errorf("%q was accepted as a book number", test.value)
			continue
		}
		if !strings.Contains(verdict.Reason, test.names) {
			t.Errorf("refusing %q gave the reason %q, which does not mention %q",
				test.value, verdict.Reason, test.names)
		}
	}
}

// The same request has to give the same number, which is the promise the whole
// tool rests on.
func TestTheSameRequestProducesTheSameBookNumber(t *testing.T) {
	first := generateISBN(t, seed.New("stable"), 41)
	second := generateISBN(t, seed.New("stable"), 41)
	if first != second {
		t.Fatalf("the same request gave %s and then %s", first, second)
	}
	if other := generateISBN(t, seed.New("stable"), 42); other == first {
		t.Errorf("two different rows gave the same number %s", first)
	}
	if other := generateISBN(t, seed.New("different"), 41); other == first {
		t.Errorf("two different seeds gave the same number %s", first)
	}
}

// What this scheme cannot see, measured rather than reasoned about.
//
// The catalog promises a hard rejection for a transposition of two adjacent
// digits, and this arithmetic cannot deliver that promise for every pair. The
// number of blind pairs is counted here instead of being argued for, because
// the same claim was made about the article number from algebra and the backlog
// entry that records it asks for the count.
func TestTheAdjacentTranspositionsThisSchemeMissesAreThoseDifferingByFive(t *testing.T) {
	missed := map[string]bool{}
	for first := 0; first <= 9; first++ {
		for second := 0; second <= 9; second++ {
			if first == second {
				continue
			}
			// Two adjacent positions inside the body, completed properly, then
			// swapped. Position eight and nine are an arbitrary pair well away
			// from both ends.
			body := []byte("978670000000")
			body[8] = byte('0' + first)
			body[9] = byte('0' + second)

			sum := 0
			for at := 0; at < len(body); at++ {
				weight := 1
				if at%2 == 1 {
					weight = 3
				}
				sum += int(body[at]-'0') * weight
			}
			value := string(body) + string(byte('0'+(10-sum%10)%10))

			swapped := []byte(value)
			swapped[8], swapped[9] = swapped[9], swapped[8]
			if verdict := (intl.ISBN{}).Validate(string(swapped)); verdict.Valid {
				pair := fmt.Sprintf("%d%d", min(first, second), max(first, second))
				missed[pair] = true
			}
		}
	}

	want := map[string]bool{"05": true, "16": true, "27": true, "38": true, "49": true}
	if len(missed) != len(want) {
		t.Errorf("this scheme is blind to %d adjacent transpositions and the count expected was %d: %v",
			len(missed), len(want), missed)
	}
	for pair := range want {
		if !missed[pair] {
			t.Errorf("the pair %s was expected to pass unnoticed and did not", pair)
		}
	}
	for pair := range missed {
		if !want[pair] {
			t.Errorf("the pair %s passes unnoticed and was not expected to", pair)
		}
	}
}
