// Package catalogue is the view qadg-gui opens with -catalogue: every control
// this window builds its screens from, in every state it can be seen in, with
// very long text and extreme values (DEC-253, rule 4 of the rules for the
// window).
//
// # Why a second frame rather than a hidden screen of the first
//
// The pages are screens of a second shell.Shell, the same frame the program
// wears, so the catalogue has a column of pages with icons and a title above
// each page and costs no layout of its own. The ceiling on hand made layouts is
// full, and a hidden screen in the first frame would have needed changes to the
// frame and to the guard over its column (DEC-253, points 1 and 5).
//
// # Why every state is put on the control directly
//
// A state of a page is reached by calling the control the way the driver calls
// it: MouseIn for the pointer, MouseDown on top of it for a press, FocusGained
// for the keyboard, Disable for out of reach. The call is made on a fresh
// control for every cell and before the page is on a canvas, so no cell
// answers to what happened in another one. The price is written here rather
// than discovered: a cell that holds the keyboard is not the control the
// canvas believes has it, so in a live window every focused cell keeps its mark
// until the keyboard really arrives and leaves.
//
// # What is left out on purpose
//
//   - internal/gui/damageparts. It builds sections out of the controls shown
//     here, and those sections are photographed where they stand, in the
//     states of the recipe policy and of the take and corrupt screen.
//   - the open panel of a chooser. It is an overlay, one per window, and it is
//     photographed by the states of the quick screen that open it.
//   - the open list of suggestions of the editor. It is the same panel drawn
//     inside the field, it opens only while somebody types there, and it is
//     photographed by the state of the recipe screen that opens it,
//     recipe-suggest (DEC-338).
//   - parts.Standing, which is a floor under a height rather than a control.
package catalogue

import (
	"fyne.io/fyne/v2"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	"github.com/donislawdev/qadatagenerator/internal/gui/shell"
)

// Samples is what stands on the controls of the catalogue.
//
// It is handed in rather than written here, because a specimen is data and
// nobody translates it (DEC-253, point 2). It lives beside the sample recipes
// in internal/gui/screens, and both the program and the stored pictures take it
// from there, so the window and the pictures of it cannot show two samples.
type Samples struct {
	// Word is one word, the length most labels and values in the window are.
	Word string
	// Long is a sentence longer than any label in the window.
	Long string
	// Extreme is a value at the edge of what a field is ever given: wide
	// letters, and more of them than the field is sized for.
	Extreme string
	// Lines is text in several lines, some longer than the control is wide.
	Lines string
	// Values are what a chooser offers and what a row of values shows. The
	// first one is the value chosen.
	Values []string
	// Table is what a table shows: its first row is the heading.
	Table [][]string
}

// The pages in the order the column shows them, named so that a state of the
// stored pictures says WHICH page it wants rather than a position that moves
// under it. The constants in internal/gui/screens say why that is measured
// rather than tidy.
//
// # Why eight pages, and what would change the number
//
// A page holds what fits the window at the size it opens in, StartWidth by
// StartHeight, because a page that runs past the foot of the window is a page
// whose foot no picture shows. Measured on the render of 2026-09-25: a chooser
// and a check box with a very long label each take about 470 of the 710 units
// a page has under its title, so they cannot share one. The seventh came on
// 2026-11-11 with the row of an empty field with a hint: the text page holding
// the editor as well asked for 855 units of height against the 780 the window
// opens at, and the page of areas given the editor instead asked for 995. The
// eighth came on 2026-11-12 with the list of values (B-370): the page of lists
// holding it as well asked for 1077 units, and the page of areas 1096, so the
// row of a value and the list it stands in became a page of their own. A
// page is a family of controls a person would look for together. What would
// change the number: a control added to a family, or a change to the size the
// window opens in.
const (
	PageButtons = iota
	PageChoosers
	PageChecks
	PageText
	PageLines
	PageLists
	PageValues
	PageAreas
	// PageCount is how many pages there are, so that a caller can ask for a
	// picture of every one of them.
	PageCount
)

// Frame is the catalogue of controls, with the brand of the program over it.
//
// The brand is handed in rather than built, because the program and the stored
// pictures already build it in one place each. Only the line under the name
// changes, so that nobody takes this frame for the window proper.
func Frame(text i18n.Catalogue, brand shell.Brand, samples Samples) *shell.Shell {
	brand.Promise = text.T(i18n.CataloguePromise)
	// Under a heading of their own, because a page naming no group goes to
	// the foot of the column, which the window keeps for the one screen that
	// is not work.
	controls := text.T(i18n.CatalogueGroup)
	pages := make([]shell.Screen, PageCount)
	pages[PageButtons] = shell.Screen{Name: text.T(i18n.CataloguePageButtons), Group: controls,
		Icon: paint.IconPlay, Content: page(text, samples, buttons())}
	pages[PageChoosers] = shell.Screen{Name: text.T(i18n.CataloguePageChoosers), Group: controls,
		Icon: paint.IconSwap, Content: page(text, samples, choosers())}
	pages[PageChecks] = shell.Screen{Name: text.T(i18n.CataloguePageChecks), Group: controls,
		Icon: paint.IconCheck, Content: page(text, samples, checks())}
	pages[PageText] = shell.Screen{Name: text.T(i18n.CataloguePageText), Group: controls,
		Icon: paint.IconDoc, Content: page(text, samples, texts())}
	pages[PageLines] = shell.Screen{Name: text.T(i18n.CataloguePageLines), Group: controls,
		Icon: paint.IconRecipe, Content: page(text, samples, editors())}
	pages[PageLists] = shell.Screen{Name: text.T(i18n.CataloguePageLists), Group: controls,
		Icon: paint.IconChecklist, Content: page(text, samples, lists())}
	pages[PageValues] = shell.Screen{Name: text.T(i18n.CataloguePageValues), Group: controls,
		Icon: paint.IconCopy, Content: page(text, samples, values())}
	pages[PageAreas] = shell.Screen{Name: text.T(i18n.CataloguePageAreas), Group: controls,
		Icon: paint.IconChevron, Content: page(text, samples, areas())}
	return shell.New(pages, brand)
}

// Content is the catalogue opened on one page, for a caller that wants only
// what is drawn.
func Content(text i18n.Catalogue, brand shell.Brand, samples Samples, page int) fyne.CanvasObject {
	frame := Frame(text, brand, samples)
	frame.Open(page)
	return frame.Content()
}
