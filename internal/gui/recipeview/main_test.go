package recipeview_test

import (
	"os"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	"github.com/donislawdev/qadatagenerator/internal/gui/fynetheme"
	"github.com/donislawdev/qadatagenerator/internal/gui/outward"
)

// TestMain gives the test application this window's own theme, so that what a
// render reads back is what a person sees.
//
// 🔴 The real theme rather than the default test one, and not only for looks:
// this screen is built out of paint, which asks the theme for its type sizes by
// name, and the default test theme answers a name it does not know with nought.
// A label of size nought divides by zero in the text renderer three layouts on,
// which is the same crash the take screen met with test.NewTempApp and is why
// nothing here uses that.
// nowhere is the Open folder button for a screen built by a test, which opens
// nothing and has nothing to refuse.
var nowhere = outward.NewFolders(i18n.English, outward.Nowhere{}, background.ReadHere, nil)

func TestMain(m *testing.M) {
	application := test.NewApp()
	application.Settings().SetTheme(fynetheme.Current())
	os.Exit(m.Run())
}
