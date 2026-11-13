package values_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
)

// exampleSchemes maps a scheme as the words of a capability name it to the
// provider this build carries for it.
var exampleSchemes = map[string]string{
	"PESEL": "PL.pesel",
	"EAN":   "ean",
	"ABA":   "US.aba",
	"SSN":   "US.ssn",
}

// anExample finds the scheme a phrase gives as its example.
var anExample = regexp.MustCompile(`such as an? ([A-Z]{2,})\b`)

// 🔴 An example in the words of a capability is a scheme of this build that
// supplies it and carries a check digit.
//
// The words reach a refusal, and "an identifier that encodes a date, such as a
// PESEL" tells somebody what their data lacks. A scheme there that does not
// supply the capability would send them to add a column that changes nothing.
// A scheme without a check digit is never read out of a file, so in the mode
// that damages one it would send somebody with that very column looking for it.
// Measured on the binary: the first wording gave a social security number as
// the example, and the file it was refused on had a column of them. The
// capabilities live in the domain and the schemes here, so this is the one
// place both can be seen.
func TestAnExampleInTheWordsOfACapabilitySuppliesIt(t *testing.T) {
	supplied := map[string]map[capability.Name]bool{}
	for _, scheme := range values.New("en-US", "").Schemes() {
		held := map[capability.Name]bool{}
		for _, name := range scheme.Capabilities() {
			held[name] = true
		}
		supplied[scheme.ID()] = held
	}
	checked := 0
	for _, name := range capability.All() {
		problems, examples := exampleProblems(name, name.Words(), supplied)
		for _, problem := range problems {
			t.Error(problem)
		}
		checked += examples
	}
	// Without examples this test would pass on nothing at all.
	if checked == 0 {
		t.Fatal("no capability gives a scheme as an example, so nothing was checked")
	}
	// The canary is the first wording, which gave a scheme that is never read
	// out of a file.
	if problems, _ := exampleProblems(capability.IdentifierReservedRange,
		"an identifier with a block that is never issued (such as an SSN)", supplied); len(problems) == 0 {
		t.Fatal("the check does not notice an example without a check digit")
	}
}

// exampleProblems says what is wrong with the examples in one phrase, and how
// many examples it found.
func exampleProblems(name capability.Name, phrase string,
	supplied map[string]map[capability.Name]bool) ([]string, int) {
	var problems []string
	found := anExample.FindAllStringSubmatch(phrase, -1)
	for _, example := range found {
		id, named := exampleSchemes[example[1]]
		if !named {
			problems = append(problems, fmt.Sprintf("%s gives %s as an example, and this test "+
				"does not know which provider that is", name, example[1]))
			continue
		}
		held, carried := supplied[id]
		if !carried {
			problems = append(problems, fmt.Sprintf("%s gives %s as an example, and this build "+
				"carries no %s", name, example[1], id))
			continue
		}
		if !held[name] {
			problems = append(problems, fmt.Sprintf("%s gives %s as an example, and %s does not "+
				"supply it", name, example[1], id))
		}
		if !held[capability.IdentifierCheckDigit] {
			problems = append(problems, fmt.Sprintf("%s gives %s as an example, and %s has no "+
				"check digit, so a file is never read as holding it", name, example[1], id))
		}
	}
	return problems, len(found)
}
