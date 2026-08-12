// Package i18n answers what the window should say, by key rather than by
// literal.
//
// # Why the window never holds its own text
//
// The command line is English and stays English, because a flag and an exit
// code are a contract with somebody's pipeline. The window is translated, and
// the rule that makes translation possible is that no screen holds a sentence:
// a new language has to be a new file beside the catalogue and no change in Go
// at all. A literal in a view is the one thing that makes that impossible,
// because it is invisible to whoever adds a language.
//
// # Why the keys are constants and not bare strings
//
// A key typed as a string turns a typo into a label that renders as its own
// key, and nobody notices until somebody reads the screen. Declared as
// constants, a typo does not compile. A test closes the constants against the
// catalogue in both directions, so a key with no entry and an entry no screen
// asks for are both failures.
package i18n

import (
	_ "embed"
	"fmt"
	"sort"

	"github.com/goccy/go-yaml"
)

// Key is the name a screen asks for text by.
type Key string

// The keys of the About screen.
const (
	AboutWindowTitle Key = "about.window.title"

	AboutHeadingVersions   Key = "about.heading.versions"
	AboutHeadingLicenses   Key = "about.heading.licenses"
	AboutHeadingComponents Key = "about.heading.components"
	AboutHeadingData       Key = "about.heading.data"
	AboutHeadingNotices    Key = "about.heading.notices"

	AboutVersionTool     Key = "about.version.tool"
	AboutVersionCatalog  Key = "about.version.catalog"
	AboutVersionManifest Key = "about.version.manifest"
	AboutVersionRecipe   Key = "about.version.recipe"

	AboutLicenseCode Key = "about.license.code"
	AboutLicenseSpec Key = "about.license.spec"

	// Only the date carries a label. The module name, its version and its
	// license read as themselves on the line, and a label in front of each
	// turned thirty seven modules into a wall nobody could scan.
	AboutColumnChecked Key = "about.column.checked"

	AboutEmptyComponents Key = "about.empty.components"
	AboutEmptyData       Key = "about.empty.data"
)

// Keys is every key a screen may ask for, so a test can close the register
// from the side the code is on.
var Keys = []Key{
	AboutWindowTitle,
	AboutHeadingVersions, AboutHeadingLicenses, AboutHeadingComponents,
	AboutHeadingData, AboutHeadingNotices,
	AboutVersionTool, AboutVersionCatalog, AboutVersionManifest, AboutVersionRecipe,
	AboutLicenseCode, AboutLicenseSpec,
	AboutColumnChecked,
	AboutEmptyComponents, AboutEmptyData,
}

// The catalogue travels inside the binary, because a window that needs a file
// beside it to name its own buttons is a window that breaks when somebody
// moves the executable.
//
//go:embed locales/en.yaml
var englishBytes []byte

// Catalogue is one language.
type Catalogue struct {
	language string
	text     map[Key]string
}

// English is the catalogue every build carries.
//
// It is parsed once at start rather than on each lookup. Parsing here can fail
// only if the embedded file stops being the file this package ships, which a
// test rules out, so the failure is turned into a catalogue that answers with
// its own keys rather than into a panic that takes the window down.
var English = func() Catalogue {
	loaded, err := Parse("en", englishBytes)
	if err != nil {
		return Catalogue{language: "en", text: map[Key]string{}}
	}
	return loaded
}()

// Parse reads a catalogue from the bytes of a locale file.
func Parse(language string, raw []byte) (Catalogue, error) {
	var flat map[string]string
	if err := yaml.Unmarshal(raw, &flat); err != nil {
		return Catalogue{}, fmt.Errorf("the %s catalogue is not readable: %w", language, err)
	}
	text := make(map[Key]string, len(flat))
	for name, value := range flat {
		text[Key(name)] = value
	}
	return Catalogue{language: language, text: text}, nil
}

// Language is the code of this catalogue.
func (c Catalogue) Language() string { return c.language }

// T is the text for a key.
//
// A missing key answers with the key itself. That is deliberate and it is the
// least bad of the three options: an empty string leaves a control with no
// label and looks like a layout fault, a panic takes down a window over a
// missing sentence, and the key at least names what is missing to whoever sees
// it. The pair of tests that close the register makes it unreachable in this
// build.
func (c Catalogue) T(key Key) string {
	if value, found := c.text[key]; found {
		return value
	}
	return string(key)
}

// Names lists every key the catalogue carries, sorted, so a test comparing it
// against the constants is not also comparing two orderings.
func (c Catalogue) Names() []Key {
	names := make([]Key, 0, len(c.text))
	for name := range c.text {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool { return names[i] < names[j] })
	return names
}
