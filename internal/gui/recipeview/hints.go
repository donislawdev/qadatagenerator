package recipeview

import (
	"fyne.io/fyne/v2"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// What this screen says to somebody who has never seen a recipe.
//
// # Why it is here at all
//
// The screen was ordered so that a person could produce test data without
// learning a file format first, and until 2026-09-03 it opened as an empty box
// saying "A recipe, in YAML". The register carried thirteen generators then and
// not one of their names appeared anywhere in the window, so the only way to
// learn them was to guess one wrong and read the refusal. A vocabulary
// reachable only by making a mistake is a vocabulary this tool keeps to itself.
//
// The count is read from the register at every draw, so a sentence claiming a
// number today would be a sentence going stale on its own.
//
// # Why this is not the form DEC-13 refused
//
// Nothing here composes a recipe out of controls. The field stays the document:
// one button types a whole example into it once, and the list beside it types
// nothing ever. A form would have to grow a control per key of the format and
// would start quietly simplifying it the day the format grew, which is what
// that decision was about.

// exampleRecipe is what the button puts in an empty field.
//
// # Why these five fields and not two
//
// Every line of it answers a question somebody meeting this screen cannot
// answer by guessing: the shape of the document, that an entity is a key rather
// than a list item, where the row count goes, what a generator is called, and
// that identifiers can be made to agree with the columns beside them. The last
// one is the whole product, so leaving it out of the first thing anybody sees
// would sell the tool short.
//
// Ten rows because the preview shows the first ten, so the sample under it is
// full rather than half empty. Measured with the real binary on 2026-09-03:
// validate answers that it can be used and would write 10 rows in 1 table, and
// a run of it writes ten rows of five columns.
//
// The output section names the form and nothing more. file_per_entity changes
// nothing in the comma separated form, and an example carrying it invites
// somebody to turn it off, see nothing change and conclude it is broken.
const exampleRecipe = `version: "0.1"
seed: "example"
locale: pl-PL
entities:
  customer:
    count: 10
    primary_key: [pesel]
    fields:
      first_name: { generator: first_name }
      last_name:  { generator: last_name }
      gender:     { generator: gender }
      birth_date: { generator: birth_date }
      pesel:      { generator: pesel }
    consistency:
      - pesel matches: [birth_date, gender]
output:
  format: csv
`

// buildHints makes the two controls that answer what can be written here.
//
// Built once, like the card above them and for the same reason: both can take
// the keyboard, and the list remembers whether somebody opened it. Rebuilding
// it on every answer would shut it again on the next key press, which is the
// kind of defect that only shows up when a person is actually using the thing.
func (v *View) buildHints() {
	v.example = parts.NewButton(v.text.T(i18n.RecipeExample), v.startFromExample)
	v.example.ShowIcon(paint.IconRecipe)
	v.generators = parts.NewDisclosure(v.generatorsTitle(), v.generatorsList())
}

// ShowGenerators opens the list, for a caller with no keyboard.
//
// Exported for the reason SetRecipe above it is: the guard that holds what
// every screen draws and the probe that renders each state to a picture both
// have to reach a state, and a list nobody can photograph is a list nobody
// reviews.
func (v *View) ShowGenerators() { v.generators.Open() }

// startFromExample fills an empty field with a recipe that runs.
//
// The keyboard is put in the field afterwards, because the button it was on is
// no longer on the screen and because editing what just arrived is the only
// thing anybody wants to do next. The toolkit clears focus on a tap of its own
// accord, so this is not taking the keyboard from somebody, it is giving it
// somewhere to be.
func (v *View) startFromExample() {
	v.SetRecipe(exampleRecipe)
	// No application means nothing is running this screen, which happens to a
	// screen built to be read rather than shown. Asking the driver for a canvas
	// then would end the process inside a courtesy.
	app := fyne.CurrentApp()
	if app == nil {
		return
	}
	if canvas := app.Driver().CanvasForObject(v.editor); canvas != nil {
		canvas.Focus(v.editor)
	}
}

// generatorsTitle names the list and says how many are in it.
//
// Both forms of the word from the catalogue, the way the counters beside it
// take theirs: a plural made by adding a letter was drawn on this screen once
// and read back as "2 entitys".
func (v *View) generatorsTitle() string {
	return v.text.T(i18n.RecipeCanGenerate) + ", " +
		v.text.Counted(len(values.New("", "").Names()), i18n.RecipeGenerator, i18n.RecipeGenerators)
}

// generatorsList is every generator this build carries, with what it produces,
// and under them the places one of them cannot be produced in.
//
// # Why the locales are a paragraph here and columns on the other surface
//
// A terminal aligns columns and a column of widgets does not, so the same three
// facts are laid out the way each surface can carry them. What must not differ
// is the facts, and the guard beside this holds the list against the register
// rather than against the other surface.
//
// # Why every generator is listed for every locale
//
// The list describes the build rather than the recipe on screen. Reading the
// locale off a recipe that has not been written yet would mean answering about
// en-US to somebody who is about to write pl-PL, and getting two of the answers
// wrong for as long as the field is empty.
func (v *View) generatorsList() fyne.CanvasObject {
	source := values.New("", "")
	rows := []fyne.CanvasObject{}
	var missing []string
	for _, name := range source.Names() {
		rows = append(rows, sentence(name+" - "+producedBy(source, name)))
		for _, locale := range values.Locales() {
			if fine, why := source.CanProduce(name, locale); !fine {
				missing = append(missing,
					name+" "+v.text.T(i18n.RecipeNotIn)+" "+locale+": "+why)
			}
		}
	}
	if len(missing) > 0 {
		rows = append(rows, paint.Caption(v.text.T(i18n.RecipeNothingToDrawFrom)))
		for _, line := range missing {
			rows = append(rows, v.paragraph(line))
		}
	}
	return paint.Column(palette.NameStepHairline, rows...)
}

// producedBy says what one generator makes, naming the numbering scheme behind
// it when there is one.
//
// The words are the vocabulary of the recipe format rather than of this window,
// so they are not translated. It is the same rule the preview follows for the
// name of a column: what a person pastes into a report reads the same in every
// language.
func producedBy(source values.Source, generator string) string {
	kind, _ := source.TypeOf(generator)
	if scheme, behind := source.ProviderOf(generator); behind {
		return kind + " (" + scheme + ")"
	}
	return kind
}
