package screens

// The catalogue of controls as states of this list, and the samples that stand
// on its controls (DEC-253).
//
// # Why the samples are here
//
// They are data, the same kind of thing the sample recipes and the sample
// manifest of this package are, and nobody translates a specimen. The program
// takes them from here as well, so the window opened with -catalogue and the
// stored pictures of it show the same words.

import (
	"github.com/donislawdev/qadatagenerator/internal/gui/catalogue"
)

// catalogueStates are the pages of the catalogue, one picture each.
func catalogueStates() []State {
	return []State{
		{
			Name:      "catalogue-buttons",
			Purpose:   "every kind of button the screens build, in every state it can be seen in",
			Screen:    catalogue.PageButtons,
			Catalogue: true,
		},
		{
			Name:      "catalogue-choosers",
			Purpose:   "the chooser with nothing chosen, a value chosen and a value too long for it",
			Screen:    catalogue.PageChoosers,
			Catalogue: true,
		},
		{
			Name:      "catalogue-checks",
			Purpose:   "the check box ticked, not ticked and with a very long label",
			Screen:    catalogue.PageChecks,
			Catalogue: true,
		},
		{
			Name:      "catalogue-text",
			Purpose:   "the text field empty, empty with a hint, short, too long and too wide",
			Screen:    catalogue.PageText,
			Catalogue: true,
		},
		{
			Name:      "catalogue-lines",
			Purpose:   "the text of several lines, empty and with lines longer than it is wide",
			Screen:    catalogue.PageLines,
			Catalogue: true,
		},
		{
			Name:      "catalogue-lists",
			Purpose:   "the row of a value under the pointer, which no other picture shows, and the table",
			Screen:    catalogue.PageLists,
			Catalogue: true,
		},
		{
			Name:      "catalogue-areas",
			Purpose:   "the scrolling area holding more than it shows and the section that opens",
			Screen:    catalogue.PageAreas,
			Catalogue: true,
		},
	}
}

// CatalogueSamples is what stands on the controls of the catalogue.
//
// Each value is chosen for what it does to a control rather than for what it
// says: a word as long as most labels, a sentence past any label the window
// has, letters as wide as the font draws them and more of them than a field is
// sized for, a line longer than any area is wide, and letters of this
// product's first locale that are missing from many fonts.
func CatalogueSamples() catalogue.Samples {
	return catalogue.Samples{
		Word:    "Generate",
		Long:    "Write every table of this recipe into the chosen directory and keep the manifest beside it",
		Extreme: "WWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWW",
		Lines: "entities:\n" +
			"  customer:\n" +
			"    count: 10\n" +
			"    fields:\n" +
			"      pesel: {generator: PL.pesel}\n" +
			"      last_name: {generator: last_name}\n" +
			"      city: {generator: city}\n" +
			"      note: {generator: text, length: 400, note: a line longer than any area that shows it}\n" +
			"corruption:\n" +
			"  layers: [L2, L3]\n" +
			"  share: 5",
		Values: []string{"pl-PL", "en-US", "Łódź"},
		Table: [][]string{
			{"Row", "Field", "Damage"},
			{"1", "pesel", "L3.checksum.off_by_one"},
			{"2", "last_name", "L2.string.empty"},
			{"3", "gender", "L2.coercion.text_to_number"},
			{"4", "quantity", "L2.type.string_in_numeric"},
			{"5", "birth_date", "L2.date.impossible"},
			{"6", "ssn", "L3.sentinel.placeholder_value"},
		},
	}
}
