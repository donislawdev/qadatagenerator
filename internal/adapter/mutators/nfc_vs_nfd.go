package mutators

import (
	"fmt"

	"golang.org/x/text/unicode/norm"

	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
)

// nfcVsNFDID is the catalog entry this implements.
const nfcVsNFDID catalog.ID = "UNI.normalization.nfc_vs_nfd"

// formParameter is the knob the entry names for the form written.
const formParameter = "form"

// decomposed is the only form this build writes.
const decomposed = "NFD"

// nfcVsNFD writes a text value in its canonically decomposed form, so that every
// accent that has a code point of its own after its letter gets one.
//
// # Why this is the same text, and what an application gets wrong about it
//
// Canonical decomposition is an equivalence the Unicode standard defines: the
// two forms are the same text, and an application that compares, indexes or
// counts them as different will also fail to find a person who typed their name
// on another device. Only the canonical forms carry that promise. The
// compatibility forms fold characters that mean different things, which is why
// the entry is about NFC and NFD and nothing else.
//
// # Why the normalisation comes from a library
//
// The tables are part of the Unicode standard and change with it. Writing the
// decomposition of a few dozen letters by hand would be wrong for every letter
// not on the list, the way a hand written segmentation was wrong for the scripts
// it left out (DEC-288). golang.org/x/text carries the tables of the standard
// it states.
type nfcVsNFD struct{}

// ID is the catalog identifier.
func (nfcVsNFD) ID() catalog.ID { return nfcVsNFDID }

// Damage writes the value in the form the entry names.
func (nfcVsNFD) Damage(target mutator.Target) mutator.Outcome {
	form, given := target.Params.Text(formParameter)
	if !given {
		return mutator.Skipped(fmt.Sprintf(
			"no %s was given, so nothing says which normalisation form to write", formParameter))
	}
	if form != decomposed {
		return mutator.Skipped(fmt.Sprintf(
			"%q is not a form this build writes. It writes %s, the decomposed form of the same text",
			form, decomposed))
	}
	if target.Value == "" {
		return mutator.Unfit("the value is empty, so there is nothing to write in another form")
	}
	if norm.NFD.IsNormalString(target.Value) {
		if norm.NFC.IsNormalString(target.Value) {
			return mutator.Unfit("no character of the value has a decomposed form, so it is " +
				"written the same in NFC and NFD, as plain Latin letters, digits and punctuation " +
				"always are")
		}
		return mutator.Unfit("the value is already decomposed, so writing it in NFD would " +
			"change nothing")
	}
	return mutator.Done(norm.NFD.String(target.Value))
}
