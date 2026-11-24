package recipeview

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestexport"
	"github.com/donislawdev/qadatagenerator/internal/commandline"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	"github.com/donislawdev/qadatagenerator/internal/gui/outward"
)

// folderless is the Open folder button for a screen built by a test inside the
// package, which opens nothing and has nothing to refuse.
var folderless = outward.NewFolders(i18n.English, outward.Nowhere{}, background.ReadHere, nil)

// The line for a terminal carries the extra form of the manifest that was
// picked, and nothing when none was.
//
// Inside the package because the line is cut with an ellipsis where it is
// drawn, so the flag that matters here is past the end of what a picture of
// the screen holds. What a person copies is the whole line, and the whole line
// is what is asked for.
func TestTheLineForATerminalCarriesTheFormThatWasPicked(t *testing.T) {
	view := New(i18n.English, nil, nil, background.Work{Read: background.ReadHere}, folderless, nil)
	view.Content()

	if strings.Contains(view.commandText.Text, commandline.Manifest) {
		t.Errorf("no form was picked and the line carries one: %q", view.commandText.Text)
	}

	form := manifestexport.Forms()[0]
	view.ChooseForm(form)
	if !strings.Contains(view.commandText.Text, commandline.Manifest+" "+form) {
		t.Errorf("the form %q was picked and the line does not carry it: %q",
			form, view.commandText.Text)
	}

	view.ChooseForm(i18n.English.T(i18n.TakeSettingNone))
	if strings.Contains(view.commandText.Text, commandline.Manifest) {
		t.Errorf("the form was set back to none and the line still carries one: %q",
			view.commandText.Text)
	}
}
