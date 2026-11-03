package intl

import (
	"strings"

	"fmt"

	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// ISBN is the thirteen digit book number.
//
// # Where the rules come from
//
// The ISBN Users' Manual of the International ISBN Agency, seventh edition,
// London 2017, appendix A1.1. It was downloaded from the agency's own site on
// 2026-09-07, and the agency's site is where the manual says its most current
// version will be found, so the copy read here is both the current text and the
// text with standing. The previous scheme in this build had to separate those
// two questions because the newest copy of its specification disclaimed itself.
// Here they are the same document, and that was established rather than assumed.
//
// The sixth edition of 2012 was read as well, because this project already
// cites it as the second source for the article number. Its appendix works a
// different example and reaches the same rule, so the two editions agree. That
// makes the older citation corroborated rather than stale, which is worth
// knowing before somebody removes it.
//
// # Why this is a separate scheme and not a setting on the article number
//
// An ISBN-13 is a GTIN-13 carrying the prefix 978 or 979, so the arithmetic is
// shared and was already proven when the article number was written. Everything
// else differs. The prefix is fixed to two values rather than free, the
// registration structure below it belongs to a different register, and the
// catalog names isbn separately from ean in the applies_to of both check digit
// entries. A flag on the article number would have made one name in the catalog
// unreachable.
//
// # The arithmetic is stated here the way the manual states it
//
// "Each of the first 12 digits of the ISBN is alternately multiplied by 1 and
// 3", counted from the left. The article number in this package walks the same
// weights from the right hand end instead, because GS1 aligns its fields to the
// right edge and has longer ones. For a twelve digit body the two readings
// coincide, and they are still written differently on purpose: an alternation
// started on the wrong foot survives being copied and does not survive being
// restated from the other end.
//
// # The residue is not the fourth answer anybody is expecting
//
// Three schemes here fold the eleventh residue of a modulus eleven sum and each
// answers differently, so a reader arrives expecting a fourth. There is none.
// The modulus is ten, ten residues fit in ten digits, and the manual names the
// one wrinkle out loud rather than leaving it to be inferred: a computed ten is
// written as nought.
//
// # ISBN-10 is not here, and the source is what decided that
//
// The ten character form would have been the genuine fourth case of the modulus
// eleven family, with a letter X in the check position. It is not implemented,
// and the reason is not that it was thought unimportant.
//
// The current manual does not carry the algorithm. Its appendix ends with a
// note that the calculation method for ten digit ISBNs is available from the
// agency's website. The sixth edition does not carry it either, so this is not
// an artefact of one edition. The page on that website which describes the
// check digit gives only the thirteen digit rule, and says that ISBNs were ten
// digits up to the end of December 2006 but since 1 January 2007 always consist
// of thirteen. So the pointer leads to a page that does not carry what it
// points at, and the agency's current description of an ISBN has no ten digit
// form in it at all.
//
// The widely repeated answer, that the ten character form is modulus eleven
// with X, is probably correct. It cannot be anchored to the body that publishes
// the standard, and three schemes in this build have already shown that the
// fold of a modulus eleven residue is the exact thing that differs between
// schemes that otherwise look identical. Guessing the fourth from the
// neighbours is what the lesson from those three forbids, so the fourth case is
// left open rather than answered from memory.
//
// # There is no safe harbour, and the candidate for one was disqualified
//
// The agency publishes a range file listing every registration group. Two
// groups in it, 978-99902 and 978-99951, are held by a "Reserved Agency" with
// no registrant space assigned at all, which looks exactly like the
// demonstration prefix that gives the article number its harbour. It is not
// one. Both were previously allocated, 99902 to Gabon and 99951 to the
// Democratic Republic of the Congo, so book numbers may have been issued inside
// them. Reserved there means withdrawn from use, not set aside for examples.
//
// That is the same kind of question the article number asked about the prefix
// 952 and it came out the other way. GS1 annotates history where there is any,
// writes nothing of the kind against 952, and stands its own examples on it.
// Here the history exists and is on the record.
//
// One asymmetry is worth stating because it will come up again. Declaring a
// harbour needs a first party source. Declining to declare one needs only a
// credible indication that numbers were once issued there. The burden runs one
// way, so a weaker source is enough to refuse a claim than to make one.
//
// # What this generates instead, and what it does not promise
//
// Generation is narrowed to the single window under prefix 978 which the
// agency's range file leaves with no registration group defined at all, the
// seven digits after the prefix falling between 6700000 and 6998999. A value
// there cannot be decomposed into a registration group, a registrant and a
// publication, so no book carries it today.
//
// That is a narrowing and not a harbour, and identifier.reserved-range is
// deliberately not declared. The agency allocates into that space over time,
// and the neighbouring groups 978-65 and 978-66 are recent, so 978-67 is
// plausibly next. This is the third scheme in this build to take that shape,
// after the equipment identity, and like that one it is guarded by a test named
// for what the provider does not claim rather than for what it does.
//
// Nothing offline can notice the day the agency allocates there, because this
// tool makes no network connection by rule. The recheck condition is written
// into docs/05 instead, where a condition for reopening a decision is what this
// project writes when no guard can exist.
//
// # Why the narrowing stays inside prefix 978
//
// The range file marks windows under prefix 979 as having no registration group
// as well, and they do not all mean the same thing. The 979-0 window is not
// empty at all: it belongs to the International Standard Music Number, whose
// agency states that an ISMN consists of thirteen digits starting 979-0. The
// 979 space is shared between the two identifiers. So under 978 the mark means
// nothing is here, and under 979 it can mean this is somebody else's. One label
// with two meanings inside one file is what the previous scheme was caught by,
// and staying in 978 keeps the reading unambiguous.
type ISBN struct{}

// Contract checks.
var (
	_ provider.Provider     = ISBN{}
	_ provider.CheckDigited = ISBN{}
	_ provider.Narrower     = ISBN{}
)

const (
	// isbnLength is thirteen digits.
	isbnLength = 13
	// isbnBodyLength is what the check digit is computed over.
	isbnBodyLength = 12
	// isbnCheckPosition is the last position, counted from zero.
	isbnCheckPosition = 12
	// isbnModulus is the divisor named by the manual.
	isbnModulus = 10
	// isbnLightWeight and isbnHeavyWeight are the alternating multipliers, in
	// the order the manual gives them: the first of the twelve digits takes the
	// light one.
	isbnLightWeight = 1
	isbnHeavyWeight = 3
	// isbnPrefixLength is the length of the GS1 element of an ISBN.
	isbnPrefixLength = 3
)

// isbnDigits is the alphabet of this scheme, which is numeric throughout.
//
// Spelled out here rather than shared with the article number. The two are the
// same ten characters for unrelated reasons and binding them would let a change
// to one scheme's alphabet move the other silently.
const isbnDigits = "0123456789"

// The two GS1 elements an ISBN may carry.
//
// A thirteen digit number beginning with anything else is a well formed article
// number and is not a book number, which is a distinction this scheme is
// entitled to make because it is readable from the value itself.
const (
	isbnBookPrefix    = "978"
	isbnSharedPrefix  = "979"
	isbnMusicSubspace = "9790"
)

// The window under prefix 978 with no registration group defined, from the
// agency's range file as read on 2026-09-07.
//
// These are the seven digits following the prefix. They are two numbers taken
// from a register rather than any part of it reproduced, which is the
// difference between a fact and somebody else's table.
const (
	isbnUnallocatedFirst = 6_700_000
	isbnUnallocatedLast  = 6_998_999
	isbnUnallocatedWidth = 7
	isbnRemainderWidth   = 2
	isbnRemainderSpace   = 100
	isbnUnallocatedCount = isbnUnallocatedLast - isbnUnallocatedFirst + 1
	isbnDrawnSpace       = isbnUnallocatedCount * isbnRemainderSpace
)

// ID names this provider.
//
// No country prefix, because the scheme is international. The catalog published
// this name before this file existed, in the applies_to of both check digit
// entries, so it is read from there rather than chosen here.
func (ISBN) ID() string { return "isbn" }

// Masks are thirteen digits, unpunctuated. The hyphenated form a book prints
// splits the number in places that depend on the registration group, so there
// is no single punctuated mask and this build produces none.
func (ISBN) Masks() []string { return []string{strings.Repeat(string(provider.MaskDigit), isbnLength)} }

// Capabilities are what this provider brings.
//
// One only. The check digit is what the catalog asks of this scheme. There is
// no reserved range to declare, for the reason written on the type, and a
// declaration made anyway would be a promise to a manifest that nothing here
// could keep.
func (ISBN) Capabilities() []capability.Name {
	return []capability.Name{capability.IdentifierCheckDigit}
}

// Within is the article number, because an ISBN-13 is a GTIN-13 carrying the
// prefix 978 or 979 and the arithmetic of the two is one rule. The validator
// here refuses everything the article number refuses and some things it
// accepts, a wrong prefix above all, so the set of book numbers lies strictly
// inside the set of article numbers and not beside it.
func (ISBN) Within() []provider.Provider { return []provider.Provider{EAN{}} }

// Generate produces one book number.
//
// The agreement is not read. Nothing in this scheme is bound to another column:
// the digits below the prefix identify a registrant and a publication within
// it, and neither is a value a recipe holds elsewhere.
//
// One draw rather than two. The window and the digits below it are decomposed
// from a single number because a draw needs a coordinate of its own, and two
// coordinates for one value would cost more than the second one buys.
func (i ISBN) Generate(req provider.Request) (string, error) {
	drawn := req.Source.IntN(isbnAt(req, "isbn-body"), isbnDrawnSpace)
	window := isbnUnallocatedFirst + drawn/isbnRemainderSpace
	below := drawn % isbnRemainderSpace

	value := make([]byte, 0, isbnLength)
	value = append(value, isbnBookPrefix...)
	value = appendFrom(value, isbnDigits, window, isbnUnallocatedWidth)
	value = appendFrom(value, isbnDigits, below, isbnRemainderWidth)
	// A place for the check digit, so the value is thirteen characters before
	// the arithmetic reads it.
	value = append(value, '0')

	digit, ok := isbnCheckDigit(value)
	if !ok {
		// Unreachable while the draws come from digits, and loud rather than
		// silent if that ever stops being true.
		return "", fmt.Errorf("isbn: the drawn characters do not form a value this scheme can complete")
	}
	value[isbnCheckPosition] = digit
	return string(value), nil
}

// Validate answers whether a value is a correct thirteen digit book number.
//
// It is written from the rules of the manual rather than from what Generate
// happens to produce. Checking a generator with its own arithmetic would make
// one mistake twice and let the two halves agree with each other.
//
// It is wider than Generate on purpose, in the shape the social security
// number, the vehicle number and the article number already have: generate only
// where nothing can be claimed, but do not call somebody else's real value
// wrong. It is not wider than the scheme, though. The prefix is checked,
// because a value that is not a book number at all should not be called a
// correct one, and the music number is refused by name for the same reason.
//
// What it does not do is ask whether the registration group exists. That would
// need the agency's register, which changes and which this tool does not carry,
// and a check standing on a register we do not ship would go quietly wrong.
func (ISBN) Validate(value string) provider.Verdict {
	if len(value) != isbnLength {
		return provider.Invalid(fmt.Sprintf(
			"a book number is thirteen digits and %s is %d",
			provider.Show(value), len(value)))
	}
	if at, ok := isbnStrangeCharacter(value); ok {
		return provider.Invalid(fmt.Sprintf(
			"position %d of %s is not a digit, and this scheme is numeric throughout",
			at+1, provider.Show(value)))
	}
	if reason, wrong := isbnWrongPrefix(value); wrong {
		return provider.Invalid(reason)
	}
	want, ok := isbnCheckDigit([]byte(value))
	if !ok {
		return provider.Invalid(fmt.Sprintf(
			"no check digit completes %s", provider.Show(value)))
	}
	if got := value[isbnCheckPosition]; got != want {
		return provider.Invalid(fmt.Sprintf(
			"the check digit is %s and should be %s",
			provider.Show(string(got)), provider.Show(string(want))))
	}
	return provider.Valid()
}

// CheckDigits says where the check digit sits.
func (ISBN) CheckDigits(value string) ([]int, bool) {
	if len(value) != isbnLength {
		return nil, false
	}
	return []int{isbnCheckPosition}, true
}

// WithCheckDigits returns the value with its check digit recomputed.
//
// It never refuses for arithmetic reasons. Every twelve digit body has a
// completion, because ten residues fit in ten digits, so a mutator that moved a
// digit always gets a consistent value back.
//
// The prefix is not examined here. This is the repair a corruption asks for
// after it has moved a digit, and refusing to repair a value because the
// corruption moved it out of the prefix would leave the mutator holding
// something it cannot finish.
func (ISBN) WithCheckDigits(value string) (string, error) {
	if len(value) != isbnLength {
		return "", fmt.Errorf("isbn: %s is not thirteen digits", provider.Show(value))
	}
	rebuilt := []byte(value)
	digit, ok := isbnCheckDigit(rebuilt)
	if !ok {
		return "", fmt.Errorf(
			"isbn: %s carries a character that is not a digit, so no check digit completes it",
			provider.Show(value))
	}
	rebuilt[isbnCheckPosition] = digit
	return string(rebuilt), nil
}

// --- the arithmetic, kept in one place ---------------------------------------

// isbnCheckDigit computes the last digit over the twelve before it.
//
// Stated as the manual states it. Each of the first twelve digits is
// alternately multiplied by one and three, starting from the left with one. The
// check digit is ten minus the remainder of that sum divided by ten, with the
// single exception the manual names: a result of ten is written as nought.
func isbnCheckDigit(value []byte) (byte, bool) {
	if len(value) != isbnLength {
		return 0, false
	}
	sum := 0
	for at := 0; at < isbnBodyLength; at++ {
		symbol := value[at]
		if !isDigit(symbol) {
			return 0, false
		}
		weight := isbnLightWeight
		if at%2 == 1 {
			weight = isbnHeavyWeight
		}
		sum += int(symbol-'0') * weight
	}
	// The exception, said without a branch: a computed ten folds to nought.
	return byte('0' + (isbnModulus-sum%isbnModulus)%isbnModulus), true
}

// isbnWrongPrefix reports why a value's GS1 element is not one a book number
// carries, and whether it is wrong at all.
//
// The music number is separated from the merely unknown prefix because the two
// are different facts about the value and a reader deserves the one that
// applies. A number beginning 979-0 is a real identifier of something, and
// saying so is more useful than saying it is not a book.
func isbnWrongPrefix(value string) (string, bool) {
	if len(value) >= len(isbnMusicSubspace) && value[:len(isbnMusicSubspace)] == isbnMusicSubspace {
		return fmt.Sprintf(
			"%s begins 9790, which is the music number rather than the book number. "+
				"Both are thirteen digits and both use this check digit, so the prefix is what separates them",
			provider.Show(value)), true
	}
	prefix := value[:isbnPrefixLength]
	if prefix != isbnBookPrefix && prefix != isbnSharedPrefix {
		return fmt.Sprintf(
			"a book number begins %s or %s and %s begins %s",
			isbnBookPrefix, isbnSharedPrefix, provider.Show(value), prefix), true
	}
	return "", false
}

// isbnStrangeCharacter returns the first position holding something that is not
// a digit, and whether there is one.
func isbnStrangeCharacter(value string) (int, bool) {
	for at := 0; at < len(value); at++ {
		if !isDigit(value[at]) {
			return at, true
		}
	}
	return 0, false
}

// isbnAt is the address the draw is derived from.
//
// It carries its own purpose, so this value cannot come out of the same number
// as another field of the same row, and the value of a row does not depend on
// the order the rows were asked for.
func isbnAt(req provider.Request, purpose string) seed.Coordinate {
	return seed.Coordinate{
		Entity:  req.At.Entity,
		Row:     req.At.Row,
		Field:   req.At.Field,
		Purpose: req.At.Purpose + ":" + purpose,
	}
}
