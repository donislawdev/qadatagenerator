package screens

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/catalogue"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// Every page of the catalogue of controls has a picture, and fits the window
// at the size it opens in, with the samples the window really shows.
//
// # Why both in one place
//
// A page that runs past the foot of the window is a page whose foot no picture
// shows, which is the reason the catalogue is cut into pages at all (DEC-253,
// point 5). And a page with no state here is a page no picture shows at all.
// Both questions need the samples, which live in this package, and the list of
// states, which is this package's.
//
// # 🔴 Why the window is laid out before it is asked
//
// A grid that wraps counts its rows from the width it was last given, so asked
// before it has been laid out it answers for one row whatever it holds. It is
// laid out at another size first for the reason render gives: a window given
// the size it already has does nothing.
func TestEveryPageOfTheCatalogueHasAPictureAndFitsTheWindowItOpensIn(t *testing.T) {
	opens := fyne.NewSize(palette.StartWidth, palette.StartHeight)
	pictured := map[int]bool{}
	for _, state := range All() {
		if !state.Catalogue {
			continue
		}
		pictured[state.Screen] = true
		content := Assemble(i18n.English, Data{}, state)
		window := test.NewWindow(content)
		window.Resize(fyne.NewSize(opens.Width-1, opens.Height))
		window.Resize(opens)
		if needs := content.MinSize(); needs.Height > opens.Height || needs.Width > opens.Width {
			t.Errorf("%s needs %v and the window opens at %v, so the foot of the page is past the "+
				"foot of the window and no picture shows it. Move a control to a page of its own",
				state.Name, needs, opens)
		}
		window.Close()
	}
	for page := 0; page < catalogue.PageCount; page++ {
		if !pictured[page] {
			t.Errorf("page %d of the catalogue has no state on the list, so no picture shows it", page)
		}
	}
}
