package intl

import (
	"strings"

	"fmt"

	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// VIN is the vehicle identification number.
//
// # Where the rules come from, and why not from the standard
//
// The shape of a VIN is described by ISO 3779, which is sold rather than
// published. The United States wrote the same requirements into 49 CFR 565 for
// every vehicle subject to its safety standards, and a work of the federal
// government carries no copyright under 17 U.S.C. 105. So the regulation is
// both readable and quotable, and it is what this file implements.
//
// That route is the one this project has taken before. The arithmetic of the
// IBAN check number reached us the same way, through a regulation of the Polish
// central bank rather than through the ISO standard it repeats.
//
// # What the regulation gives that most sources do not
//
// The shape and the arithmetic, in one document, with a worked example beside
// them. That is worth naming because it is rare here: four official documents
// describe the shape of a REGON and not one of them states the arithmetic, so
// the weights there rest on a measurement instead of a quotation. Here there is
// no such gap, and the example of Table VI is held as an anchor.
//
// # The remainder of ten is answered a third way
//
// Two schemes in this build already fold the eleventh residue of a modulus
// eleven sum, and they answer differently: the Polish tax number calls such a
// body impossible and refuses it, while the statistical number writes nought.
// This one writes the letter X, which is a character the rest of the value
// cannot contain. Three schemes, three answers, so there is no general rule to
// carry to the fourth.
//
// # What this does not check, and why
//
// The check digit is a requirement of this regulation. A vehicle built outside
// its reach may carry a VIN with no check digit at all, and position nine will
// then hold whatever the manufacturer put there. Such a value cannot be told
// apart from a damaged one by looking at it, because nothing in the value says
// which regime it came from. Answering Unsupported would need that recognition,
// and recognising it would need a table of regions this file has no source for.
//
// So a value whose ninth position disagrees with the arithmetic is called
// invalid, and the message names the rule it was measured against rather than
// pronouncing on the vehicle. The limit is written down in docs/05 as well,
// because a limit known only to the code is a limit nobody reads.
type VIN struct{}

// Contract checks.
var (
	_ provider.Provider     = VIN{}
	_ provider.CheckDigited = VIN{}
)

const (
	// vinLength is seventeen characters, from 49 CFR 565.13(b).
	vinLength = 17
	// vinCheckPosition is position nine of the regulation, counted from zero.
	vinCheckPosition = 8
	// vinModulus and vinFoldedRemainder are the divisor of 49 CFR 565.15(c)(3)
	// and the one residue that does not fit in a digit.
	vinModulus         = 11
	vinFoldedRemainder = 10
	// vinFolded is what the regulation writes for that residue, from Table V.
	vinFolded = 'X'
)

// vinAlphabet is the set of 49 CFR 565.13(g), digits first.
//
// The letters I, O and Q are absent. The regulation states the set rather than
// the exclusion, so this is a quotation and not an inference, and the reason
// those three are missing is that a reader cannot tell them from one and zero.
const vinAlphabet = "0123456789ABCDEFGHJKLMNPRSTUVWXYZ"

// vinYearCodes are the model year codes of Table VII, as a set.
//
// The table runs from 2005 to 2039 and repeats after thirty, so the distinct
// codes are what matters here rather than the years. Three characters of the
// alphabet never appear as a year: nought, U and Z.
const vinYearCodes = "123456789ABCDEFGHJKLMNPRSTVWXY"

// vinWeights are the position weight factors of Table IV.
//
// The ninth entry is nought, which is how the regulation keeps the check digit
// out of its own sum. Nothing here has to remember to skip it.
var vinWeights = [vinLength]int{8, 7, 6, 5, 4, 3, 2, 10, 0, 9, 8, 7, 6, 5, 4, 3, 2}

// vinLetterValues is Table III, indexed by letter.
//
// A zero means the letter has no assigned value, which is unambiguous because
// the regulation assigns nothing lower than one. The three zeroes are I, O and
// Q, and they are the same three the alphabet leaves out, so a value carrying
// one of them is refused by the alphabet before it ever reaches this table.
var vinLetterValues = [26]int{
	1, 2, 3, 4, 5, 6, 7, 8, 0, // A B C D E F G H I
	1, 2, 3, 4, 5, 0, // J K L M N O
	7, 0, 9, // P Q R
	2, 3, 4, 5, 6, 7, 8, 9, // S T U V W X Y Z
}

// vinDigits is the alphabet of the tail, which the regulation requires to be
// numeric.
const vinDigits = "0123456789"

// The drawn blocks, each one a section the regulation names.
//
// Sixteen characters are drawn and one is computed. They are taken in four
// draws rather than sixteen, because a draw needs a coordinate and building
// sixteen of them for one value would cost more than the value is worth. Every
// bound is inside what an int64 holds, the largest being thirty three to the
// eighth, about 1.41e12.
//
// 🔴 The tail is drawn from digits and not from the alphabet, because 49 CFR
// 565.15(d) requires the last five characters to be numeric for passenger cars
// and the last four for every other vehicle. Five satisfies both, and the
// example the regulation works in Table VI ends in five digits.
//
// This was drawn from the whole alphabet until the mutator that adds one to the
// final digit reported that it had nothing to change. The guard was about
// damage and the defect was in generation, which is the more useful direction
// for a guard to fail in.
const (
	vinFrontLength  = 8
	vinFrontSpace   = 1_406_408_618_241
	vinMiddleLength = 2
	vinMiddleSpace  = 1089
	vinTailLength   = 5
	vinTailSpace    = 100_000
)

// ID names this provider.
//
// It carries no country prefix, because the scheme is international even though
// the regulation this file quotes is not. The catalog published the name before
// this file existed, in the applies_to of both checksum entries.
func (VIN) ID() string { return "vin" }

// Capabilities are what this provider brings.
//
// One. There is deliberately no claim to encode a date: position ten carries a
// model year, reading it needs position seven as well, and a year is not the
// day and month a Date carries. Squeezing one into the other would be a name
// that lies, which costs more than the capability is worth.
//
// There is no reserved range either. 49 CFR 565.16(a) has manufacturer
// identifiers assigned by SAE International, which checks a request against
// what is assigned and against blocks already reserved. Reserved there means
// held back for assignment, not left free, so this scheme has no safe harbour
// in the sense docs/05 uses the word.
// Mask is seventeen places that each take a digit or a letter, which is the
// one scheme here that is not purely numeric.
//
// It does not say which letters: I, O and Q are excluded by the standard and
// the last five places are digits for a passenger car. A mask carrying those
// rules would be a second copy of Validate, and the two would part company.
func (VIN) Mask() string {
	return strings.Repeat(string(provider.MaskAlnum), vinLength)
}

func (VIN) Capabilities() []capability.Name {
	return []capability.Name{capability.IdentifierCheckDigit}
}

// Generate produces one vehicle identification number for the place described
// by the request.
//
// The agreement is not read. Nothing in this scheme is bound to another column
// of the row: the sections identify a manufacturer, a set of vehicle attributes
// and a serial number, and none of those is a value a recipe can hold elsewhere.
//
// The model year is drawn from Table VII rather than from the whole alphabet,
// so the value can be decoded by whoever receives it. That makes this generator
// satisfy more rules than Validate enforces, which is the same shape the social
// security number has: it is generated only from ranges nobody is issued from,
// and it validates every well formed number.
func (v VIN) Generate(req provider.Request) (string, error) {
	value := make([]byte, 0, vinLength)
	value = appendFrom(value, vinAlphabet,
		req.Source.IntN(vinAt(req, "vin-front"), vinFrontSpace), vinFrontLength)
	// The check digit is computed last and its weight is nought, so what
	// stands here until then cannot change the sum. It is a character of the
	// alphabet all the same, because a value half built is still a value the
	// arithmetic will read.
	value = append(value, '0')
	value = append(value, vinYearCodes[req.Source.IntN(vinAt(req, "vin-year"),
		int64(len(vinYearCodes)))])
	value = appendFrom(value, vinAlphabet,
		req.Source.IntN(vinAt(req, "vin-middle"), vinMiddleSpace), vinMiddleLength)
	value = appendFrom(value, vinDigits,
		req.Source.IntN(vinAt(req, "vin-tail"), vinTailSpace), vinTailLength)

	digit, ok := vinCheckDigit(value)
	if !ok {
		// Unreachable while the draws come from the alphabet, and loud rather
		// than silent if that ever stops being true.
		return "", fmt.Errorf("vin: the drawn characters do not form a value this scheme can complete")
	}
	value[vinCheckPosition] = digit
	return string(value), nil
}

// Validate answers whether a value is a correct vehicle identification number
// under 49 CFR 565.
//
// It is written from the rules of the regulation rather than from what Generate
// happens to produce. Checking a generator with its own arithmetic would make
// one mistake twice and let the two halves agree with each other.
func (VIN) Validate(value string) provider.Verdict {
	if len(value) != vinLength {
		return provider.Invalid(fmt.Sprintf(
			"a vehicle identification number is seventeen characters and %s is %d",
			provider.Show(value), len(value)))
	}
	if at, ok := vinStrangeCharacter(value); ok {
		return provider.Invalid(fmt.Sprintf(
			"position %d of %s is not one of the characters this scheme uses. The letters I, O and Q are left out so that nothing can be read as a one or a nought",
			at+1, provider.Show(value)))
	}
	want, ok := vinCheckDigit([]byte(value))
	if !ok {
		return provider.Invalid(fmt.Sprintf(
			"no check digit completes %s", provider.Show(value)))
	}
	if got := value[vinCheckPosition]; got != want {
		return provider.Invalid(fmt.Sprintf(
			"the check digit is %s and should be %s",
			provider.Show(string(got)), provider.Show(string(want))))
	}
	return provider.Valid()
}

// CheckDigits says where the check digit sits.
func (VIN) CheckDigits(value string) ([]int, bool) {
	if len(value) != vinLength {
		return nil, false
	}
	return []int{vinCheckPosition}, true
}

// WithCheckDigits returns the value with its check digit recomputed.
//
// It never refuses for arithmetic reasons, and the contrast with the Polish tax
// number is the whole of this scheme's choice: the eleventh residue is written
// as a letter rather than called impossible, so every well formed body has a
// completion and a mutator that moved a character always gets a consistent
// value back.
func (VIN) WithCheckDigits(value string) (string, error) {
	if len(value) != vinLength {
		return "", fmt.Errorf("vin: %s is not seventeen characters", provider.Show(value))
	}
	rebuilt := []byte(value)
	digit, ok := vinCheckDigit(rebuilt)
	if !ok {
		return "", fmt.Errorf(
			"vin: %s carries a character this scheme does not use, so no check digit completes it",
			provider.Show(value))
	}
	rebuilt[vinCheckPosition] = digit
	return string(rebuilt), nil
}

// --- the arithmetic, kept in one place ---------------------------------------

// vinCheckDigit computes the ninth character over the other sixteen.
//
// The four steps are those of 49 CFR 565.15(c): transliterate, multiply by the
// weight of the position, add, divide by eleven. The regulation describes the
// last step by rounding a decimal, which is how a person with a calculator
// reads a table. This takes the integer remainder, which is the same number
// said without a rounding rule to get wrong.
func vinCheckDigit(value []byte) (byte, bool) {
	if len(value) != vinLength {
		return 0, false
	}
	sum := 0
	for at, weight := range vinWeights {
		assigned, ok := vinValueOf(value[at])
		if !ok {
			return 0, false
		}
		sum += assigned * weight
	}
	remainder := sum % vinModulus
	if remainder == vinFoldedRemainder {
		return vinFolded, true
	}
	return byte('0' + remainder), true
}

// vinValueOf is the assigned value of one character, from Table III.
func vinValueOf(symbol byte) (int, bool) {
	if isDigit(symbol) {
		return int(symbol - '0'), true
	}
	if !isUpperLetter(symbol) {
		return 0, false
	}
	if assigned := vinLetterValues[symbol-'A']; assigned != 0 {
		return assigned, true
	}
	return 0, false
}

// vinStrangeCharacter returns the first position holding a character this
// scheme does not use, and whether there is one.
func vinStrangeCharacter(value string) (int, bool) {
	for at := 0; at < len(value); at++ {
		if _, ok := vinValueOf(value[at]); !ok {
			return at, true
		}
	}
	return 0, false
}

// appendFrom writes a drawn number as characters of the given alphabet, widest
// first, in a fixed width.
func appendFrom(into []byte, alphabet string, drawn int64, width int) []byte {
	start := len(into)
	into = append(into, make([]byte, width)...)
	base := int64(len(alphabet))
	for at := width - 1; at >= 0; at-- {
		into[start+at] = alphabet[drawn%base]
		drawn /= base
	}
	return into
}

// vinAt is the address one draw is derived from.
//
// Each draw carries its own purpose, so the two blocks of one value cannot come
// out of the same number, and the value of a row does not depend on the order
// the rows were asked for.
func vinAt(req provider.Request, purpose string) seed.Coordinate {
	return seed.Coordinate{
		Entity:  req.At.Entity,
		Row:     req.At.Row,
		Field:   req.At.Field,
		Purpose: req.At.Purpose + ":" + purpose,
	}
}
