package capability_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
)

// wordsProblems says what is wrong with the words of the capabilities given,
// one sentence per problem.
func wordsProblems(names []capability.Name) []string {
	var problems []string
	for _, name := range names {
		words := name.Words()
		scope, _ := capability.ScopeOf(name)
		switch {
		case words == string(name):
			problems = append(problems, fmt.Sprintf("%s has no words and would reach a refusal as its name", name))
			continue
		case scope == capability.Field && name.ColumnWords() == "":
			problems = append(problems, fmt.Sprintf("%s belongs to a column and has no column words", name))
		case scope == capability.Run && name.ColumnWords() != "":
			problems = append(problems, fmt.Sprintf("%s belongs to a run and has column words", name))
		}
		for _, said := range []string{words, name.ColumnWords()} {
			if problem := unreadable(said); problem != "" {
				problems = append(problems, fmt.Sprintf("%s: %q %s", name, said, problem))
			}
		}
	}
	return problems
}

// bracketed is an example in brackets, which may hold anything.
var bracketed = regexp.MustCompile(`\([^)]*\)`)

// unreadable says why a phrase would not read inside a refusal, or nothing.
func unreadable(phrase string) string {
	if phrase == "" {
		return ""
	}
	if first := []rune(phrase)[0]; unicode.IsUpper(first) {
		return "starts with a capital, and it goes into the middle of a sentence"
	}
	// The joining mark is written as its code point, because the repository
	// keeps the character itself out of its prose and its sources alike.
	if strings.HasSuffix(phrase, ".") || strings.ContainsRune(phrase, rune(0x3b)) {
		return "ends a sentence or joins two, and it is one part of a sentence"
	}
	// Several phrases are joined with commas and a final and, so either one
	// outside the brackets of an example would split a phrase in the list.
	if outside := bracketed.ReplaceAllString(phrase, ""); strings.Contains(outside, ",") ||
		strings.Contains(outside, " and ") {
		return "holds a comma or an and outside brackets, and would split in a list of several"
	}
	for _, other := range capability.All() {
		if strings.Contains(phrase, string(other)) {
			return "names " + string(other) + ", which is what the words replace"
		}
	}
	return ""
}

// 🔴 Every capability in the registry has words, so none reaches a refusal as a
// bare name. The registry is read rather than listed here, so a capability added
// tomorrow is held to this without anybody remembering to add it.
func TestEveryCapabilityCanBeSaidInWords(t *testing.T) {
	for _, problem := range wordsProblems(capability.All()) {
		t.Error(problem)
	}
	// The canary is a capability nobody wrote words for.
	if len(wordsProblems([]capability.Name{"made.up"})) == 0 {
		t.Fatal("the check does not notice a capability without words")
	}
	if unreadable("knows the schema.surname of it") == "" {
		t.Fatal("the check does not notice a phrase that names a capability")
	}
	// The first wording, which ran into the phrase after it in a list.
	if unreadable("an identifier with a check digit, such as a PESEL") == "" {
		t.Fatal("the check does not notice a comma outside brackets")
	}
}

// A capability of a column reads the same in both sentences, as the thing a
// run lacks and as what the column it landed on is not.
func TestAColumnCapabilityIsAColumnThatIsSomething(t *testing.T) {
	name := capability.SchemaSurname
	if got := name.Words(); got != "a column "+name.ColumnWords() {
		t.Errorf("%s reads %q as what a run lacks and %q as what a column is", name, got, name.ColumnWords())
	}
}
