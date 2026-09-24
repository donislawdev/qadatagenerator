package fynetheme

import (
	"encoding/json"
	"fmt"
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
)

// toolkitWords is every word the toolkit localises that this window can put in
// front of somebody, by the identifier the toolkit asks for it with.
//
// # How the list was made, and what is left out on purpose
//
// Read on the source of fyne v2.8.1: every call of lang.L and lang.X under
// dialog, widget and the desktop driver. Left out are the colour picker
// (Advanced, Confirm) and the calendar (the names of the days), which this
// window never opens. The names of the places at the side of the file dialog
// (Home, Documents, Downloads) are drawn by the toolkit without asking, so they
// are English on every desktop and are not here either.
//
// A word missing from this list falls back to the English written in the
// toolkit's own source, which is the language of this window today. It stops
// being harmless the day a second interface language arrives, and that is the
// day this list has to be complete rather than sufficient.
var toolkitWords = map[string]i18n.Key{
	"Open":              i18n.ToolkitOpen,
	"Save":              i18n.ToolkitSave,
	"Cancel":            i18n.ToolkitCancel,
	"File":              i18n.ToolkitFile,
	"Folder":            i18n.ToolkitFolder,
	"New Folder":        i18n.ToolkitNewFolder,
	"Create Folder":     i18n.ToolkitCreateFolder,
	"file.name":         i18n.ToolkitName,
	"file.parent":       i18n.ToolkitParent,
	"Show Hidden Files": i18n.ToolkitShowHiddenFiles,
	"Enter filename":    i18n.ToolkitEnterFilename,
	"OK":                i18n.ToolkitOK,
	"Error":             i18n.ToolkitError,
	"Yes":               i18n.ToolkitYes,
	"No":                i18n.ToolkitNo,
	"Cut":               i18n.ToolkitCut,
	"Copy":              i18n.ToolkitCopy,
	"Paste":             i18n.ToolkitPaste,
	"Select all":        i18n.ToolkitSelectAll,
	"Undo":              i18n.ToolkitUndo,
	"Redo":              i18n.ToolkitRedo,
	"Quit":              i18n.ToolkitQuit,
}

// Speak makes the toolkit say its own words in the language of this window
// rather than in the language of the desktop.
//
// # Why this is needed at all
//
// The toolkit picks its language from the system, and on Windows from
// GetUserDefaultLocaleName alone, with no variable to set instead (read on
// go-locale, which fyne v2.8.1 uses for it). So on a Polish desktop the file
// dialog of an English window gave its title, its buttons and its check box in
// Polish, and so did the menu of a text field (B-290). Rule fourteen of
// the project separates the language of the data from the language of the
// interface, and here the interface spoke two.
//
// # How
//
// lang.AddTranslationsForLocale is the toolkit's public way in. Messages added
// for a language replace the ones it carried for the same language, so these
// words are added for the locale of the system and for its language alone,
// which are the two the toolkit can settle on for this desktop. Whichever it
// picks then holds the words of this catalogue.
//
// Called once at start, before the first window is built, because the toolkit
// reads a word when it builds the control carrying it.
func Speak(text i18n.Catalogue) error {
	messages := make(map[string]string, len(toolkitWords))
	for id, key := range toolkitWords {
		messages[id] = text.T(key)
	}
	data, err := json.Marshal(messages)
	if err != nil {
		return fmt.Errorf("the words handed to the toolkit could not be written: %w", err)
	}
	for _, locale := range desktopLocales() {
		if err := lang.AddTranslationsForLocale(data, locale); err != nil {
			return fmt.Errorf("the toolkit refused the words for %s: %w", locale, err)
		}
	}
	return nil
}

// desktopLocales is the locale of this desktop and its language alone, the
// second only when it differs, in a fixed order.
func desktopLocales() []fyne.Locale {
	system := lang.SystemLocale()
	locales := map[fyne.Locale]bool{system: true, fyne.Locale(system.LanguageString()): true}
	ordered := make([]fyne.Locale, 0, len(locales))
	for locale := range locales {
		ordered = append(ordered, locale)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	return ordered
}
