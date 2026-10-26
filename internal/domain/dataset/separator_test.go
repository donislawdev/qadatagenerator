package dataset_test

import (
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
)

// A separator is one character that does not already mean something in the
// format, and the reader of this format is what decides which those are.
func TestASeparatorIsOneCharacterTheFormatDoesNotAlreadyUse(t *testing.T) {
	for _, given := range []struct {
		text   string
		usable bool
	}{
		{",", true}, {";", true}, {"\t", true}, {"|", true}, {"ż", true}, // punctuation: test data
		{"", false}, {"ab", false}, {"\n", false}, {"\r", false}, {`"`, false},
		{"\x00", false}, {"�", false}, {"\xff", false},
	} {
		separator, usable := dataset.UsableSeparator(given.text)
		if usable != given.usable {
			t.Errorf("%q was taken as usable=%v, and it is %v", given.text, usable, given.usable)
			continue
		}
		if usable && string(separator) != given.text {
			t.Errorf("%q came back as %q", given.text, string(separator))
		}
	}
}
