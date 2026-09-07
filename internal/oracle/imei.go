package oracle

import "strings"

// The rule this file implements.
//
// ETSI TS 123 003 version 19.7.0, the transposition of 3GPP TS 23.003,
// "Numbering, addressing and identification". Clause 6.2.1 states the shape and
// annex B, which is normative, states the arithmetic in three numbered steps
// and then works an example of them.
//
// The newer text of version 20.0.0 was read too and says the same thing word
// for word in that annex. It is not the one cited, because it states that it
// shall not be implemented and points at the publications offices instead, and
// ETSI is one of those. The provider carries the longer account of that.
//
// This is one document rather than two either way. Two readings of one text is
// a second reading and not a second source, and the register of anchors says so
// rather than counting it twice.
//
// Nothing here rests on a measurement.

// imeiStepOneAsPublished, imeiStepTwoAsPublished and imeiStepThreeAsPublished
// are the three steps of annex B.2, in the words the specification uses.
//
// # Why the steps and not just the answer
//
// The provider says the same thing as a walk from the right hand end and a
// remainder folded twice, which is compact and is not what the specification
// says. This file follows the sentences literally, labelling the digits the way
// annex B labels them, so that a fold written the wrong way round in one is not
// written the same wrong way round in the other.
const (
	imeiStepOneAsPublished   = "Double the values of the odd labelled digits D1, D3, D5 ... D13 of the IMEI."
	imeiStepTwoAsPublished   = "Add together the individual digits of all the seven numbers obtained in Step 1, and then add this sum to the sum of all the even labelled digits D2, D4, D6 ... D14 of the IMEI."
	imeiStepThreeAsPublished = "If the number obtained in Step 2 ends in 0, then set CD to be 0. If the number obtained in Step 2 does not end in 0, then set CD to be that number subtracted from the next higher number which does end in 0."
)

// imeiAllocationCodeAsPublished and imeiSerialNumberAsPublished are the two
// elements clause 6.2.1 lists, each written as the specification writes it.
//
// # Why two constants rather than one
//
// The clause prints them as separate items of a list. Joining them into one
// string would have made a sentence the source does not contain, which is a
// paraphrase wearing the clothes of a quotation, and it would have carried the
// list separator into prose that may not hold one.
//
// Held because the eight and the six are the shape of the value and the fold
// below does not read them. A shape typed into a provider and never compared
// with anything is a shape nobody checked.
const (
	imeiAllocationCodeAsPublished = "Type Allocation Code (TAC). Its length is 8 digits"
	imeiSerialNumberAsPublished   = "Serial Number (SNR) is an individual serial number uniquely identifying each equipment within the TAC. Its length is 6 digits"
)

// imeiOddLabelCount is the count annex B.2 states out loud when it calls the
// results of step 1 "the seven numbers".
//
// It is a structural fact about the labelling rather than a free parameter: the
// odd labels from D1 to D13 are seven of the fourteen. Held so that a change to
// the labelling below disagrees with the sentence instead of quietly weighing a
// different set of digits.
const imeiOddLabelCount = 7

// IMEIFinding is one reason a value is not a valid fifteen digit equipment
// identity. An empty list means the value is valid.
type IMEIFinding string

// CheckIMEI reports every rule a value breaks.
//
// # What it deliberately does not answer
//
// It asks the fifteen digit question only. The sixteen digit form carrying a
// software version number has no check digit at all, so there is no arithmetic
// here that could speak about it, and the provider answers that one with a
// verdict about applicability rather than about correctness. An oracle that
// invented an opinion there would disagree with the provider about a value
// neither of them calls wrong.
//
// It also says nothing about where the value sits. This scheme has no safe
// harbour and the provider does not refuse values outside the block it
// generates in, so neither does this.
func CheckIMEI(value string) []IMEIFinding {
	if !imeiPublishedStepsStillSayWhatTheFoldDoes() {
		// The arithmetic below is written from those three sentences. If a
		// sentence stops saying what the code does, the code is no longer
		// supported by anything and should say so rather than carry on being
		// right by accident.
		return []IMEIFinding{"the published steps no longer describe the calculation this file performs"}
	}
	if body, serial, ok := imeiPublishedLengths(); !ok || body+serial != 14 {
		// Loud rather than truncated. Lengths that stopped adding up to the
		// body would otherwise leave the labelling below reading past the value.
		return []IMEIFinding{"the published element lengths no longer add up to the fourteen digits the check digit covers"}
	}
	if len(value) != 15 {
		return []IMEIFinding{"an equipment identity is fifteen digits"}
	}
	for at := 0; at < len(value); at++ {
		if value[at] < '0' || value[at] > '9' {
			return []IMEIFinding{"the value carries a character that is not a digit"}
		}
	}

	total, counted := imeiPublishedTotal(value[:14])
	if counted != imeiOddLabelCount {
		return []IMEIFinding{"step 1 did not produce the seven numbers the specification names"}
	}

	var findings []IMEIFinding
	if want := imeiNextNumberEndingInNought(total) - total; value[14] != byte('0'+want) {
		findings = append(findings, "the check digit does not agree with the fourteen digits before it")
	}
	return findings
}

// IMEIPublishedTotal is the number annex B.2 calls the result of step 2, which
// annex B.3 prints for its own example before reaching a check digit.
//
// Exported so that a test can pin the working and not only the answer. A worked
// example compared on its final digit alone lets a wrong sum and a wrong fold
// cancel each other out, and this specification publishes the number in the
// middle precisely so that they cannot.
func IMEIPublishedTotal(body string) (int, bool) {
	if len(body) != 14 {
		return 0, false
	}
	for at := 0; at < len(body); at++ {
		if body[at] < '0' || body[at] > '9' {
			return 0, false
		}
	}
	total, counted := imeiPublishedTotal(body)
	return total, counted == imeiOddLabelCount
}

// imeiPublishedTotal performs steps 1 and 2 on the fourteen digits, using the
// labels the specification uses rather than a position counted from an end.
//
// The digits are labelled D14 first down to D1 last, so the digit at index
// position carries the label fourteen less that index. The odd labelled ones
// are doubled, the individual digits of each doubled result are added, and that
// sum is added to the plain sum of the even labelled ones. It returns how many
// numbers step 1 produced as well, because the specification says how many
// there should be.
func imeiPublishedTotal(body string) (total int, doubled int) {
	for index := 0; index < len(body); index++ {
		label := len(body) - index
		digit := int(body[index] - '0')
		if label%2 == 0 {
			total += digit
			continue
		}
		doubled++
		total += imeiSumOfIndividualDigits(digit * 2)
	}
	return total, doubled
}

// imeiSumOfIndividualDigits adds the digits of a number, which is what step 2
// asks for and is not the same operation as subtracting nine.
//
// Subtracting nine gives the same answer for every number step 1 can produce
// and is what the provider does. Written out here so the two files do not share
// a shortcut whose correctness depends on the range of its input.
func imeiSumOfIndividualDigits(number int) int {
	sum := 0
	for number > 0 {
		sum += number % 10
		number /= 10
	}
	return sum
}

// imeiNextNumberEndingInNought is step 3 taken literally.
//
// A total already ending in nought stays where it is, which is the first
// sentence of the step and is how a check digit of nought arises. The provider
// reaches the same number by folding a remainder twice, and the two shapes
// disagree if either is wrong.
func imeiNextNumberEndingInNought(total int) int {
	return ((total + 9) / 10) * 10
}

// imeiPublishedStepsStillSayWhatTheFoldDoes reports whether the three held
// sentences still carry the parts the arithmetic above depends on.
//
// It reads the sentences rather than merely storing them, which is the
// difference between a constant that documents and a constant that holds.
func imeiPublishedStepsStillSayWhatTheFoldDoes() bool {
	for sentence, parts := range map[string][]string{
		imeiStepOneAsPublished:   {"double the values", "odd labelled digits", "d1, d3, d5 ... d13"},
		imeiStepTwoAsPublished:   {"individual digits", "seven numbers", "even labelled digits", "d2, d4, d6 ... d14"},
		imeiStepThreeAsPublished: {"ends in 0", "set cd to be 0", "next higher number which does end in 0"},
	} {
		lowered := strings.ToLower(sentence)
		for _, part := range parts {
			if !strings.Contains(lowered, part) {
				return false
			}
		}
	}
	return true
}

// imeiPublishedLengths reads the two element lengths out of the held sentences,
// so the fourteen the check digit covers is taken from the source rather than
// typed here as a number.
func imeiPublishedLengths() (allocation int, serial int, ok bool) {
	first := imeiDigitRunsFollowing(imeiAllocationCodeAsPublished, "its length is ")
	second := imeiDigitRunsFollowing(imeiSerialNumberAsPublished, "its length is ")
	if len(first) != 1 || len(second) != 1 {
		return 0, 0, false
	}
	return first[0], second[0], true
}

// imeiDigitRunsFollowing collects the numbers that follow each occurrence of a
// marker in a sentence.
func imeiDigitRunsFollowing(sentence string, marker string) []int {
	lowered := strings.ToLower(sentence)
	var found []int
	for at := 0; ; {
		next := strings.Index(lowered[at:], marker)
		if next < 0 {
			return found
		}
		at += next + len(marker)
		number, digits := 0, 0
		for at+digits < len(lowered) && lowered[at+digits] >= '0' && lowered[at+digits] <= '9' {
			number = number*10 + int(lowered[at+digits]-'0')
			digits++
		}
		if digits == 0 {
			return nil
		}
		found = append(found, number)
		at += digits
	}
}
