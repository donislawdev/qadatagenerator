package manifestview

import (
	"errors"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
)

// saying reports whether the words are anywhere on what is drawn.
//
// The tree is walked rather than rendered, because these guards are about what
// the screen says and not about how it looks. The scrolling areas of the
// toolkit are walked into, which is what reaches the sentence in the card.
func saying(drawn fyne.CanvasObject, words string) bool {
	for _, said := range texts(drawn) {
		if strings.Contains(said, words) {
			return true
		}
	}
	return false
}

func texts(object fyne.CanvasObject) []string {
	switch typed := object.(type) {
	case *widget.Label:
		return []string{typed.Text}
	case *canvas.Text:
		return []string{typed.Text}
	case *widget.RichText:
		return drawn(typed)
	case *container.Scroll:
		return texts(typed.Content)
	case *fyne.Container:
		var found []string
		for _, child := range typed.Objects {
			found = append(found, texts(child)...)
		}
		return found
	}
	return nil
}

// Reading a document says so on the screen and puts the two buttons out of
// reach until it answers.
//
// 🔴 Both halves were missing until 2026-09-01, and the reason they arrived
// together is that each one is useless without the other. The read happened
// where the file chooser answered, which is the thread the toolkit draws on, so
// for a manifest of the size this product warns about the window repainted
// nothing and answered no key. Saying "reading" there would have been a
// sentence nobody could see, because nothing was repainting to show it.
//
// The sentence is point eight of the GUI checklist read the other way round. A
// screen that has not answered yet and one with nothing to say look alike, and
// the one that is working is the one that has to say so.
func TestReadingADocumentSaysSoAndPutsTheControlsOutOfReach(t *testing.T) {
	waiting := []func() func(){}
	view := New(i18n.English, nil, Readers{
		Manifest: func(string) (manifest.Manifest, error) {
			return manifest.Manifest{}, errors.New("nothing was read, this test never answers")
		},
	}, func(read func() func(), _ func(error) func()) { waiting = append(waiting, read) }, nil)
	view.Content()

	view.startReading("/runs/expectation-manifest.json", theManifest, func() func() {
		_, _ = view.read.Manifest("whatever")
		return func() {}
	})

	if len(waiting) != 1 {
		t.Fatalf("a read was asked for %d times at the place of work rather than once, so "+
			"the document is being read on the thread that draws", len(waiting))
	}
	if !view.open.Disabled() {
		t.Error("the button that opens a manifest can be pressed while one is being read, " +
			"so a second read can be asked for over the first")
	}
	if !view.respond.Disabled() {
		t.Error("the button that opens a response record can be pressed while a document " +
			"is being read")
	}
	if !saying(view.detailBox, i18n.English.T(i18n.ManifestReading)) {
		t.Error("the screen does not say that it is reading, so a person waiting on a large " +
			"manifest cannot tell it from a screen with nothing to say")
	}
	if !saying(view.manifestFacts, i18n.English.T(i18n.ManifestFactsReading)) {
		t.Error("the band does not say beside the file that it is being read")
	}

	waiting[0]()()
	if view.open.Disabled() {
		t.Error("the read answered and the button that opens a manifest is still out of reach")
	}
	if view.Reading() {
		t.Error("the read answered and the screen still holds itself to be reading")
	}
}

// A read that failed leaves the band naming the file and the card saying why.
func TestADocumentThatCouldNotBeReadIsNamedInTheBandAndExplainedInTheCard(t *testing.T) {
	view := New(i18n.English, nil, Readers{}, func(read func() func(), _ func(error) func()) {
		read()()
	}, nil)
	view.Content()
	view.startReading("/runs/broken.json", theManifest, func() func() {
		return func() { view.unreadable(errors.New("this is not a manifest")) }
	})

	if !saying(view.manifestFacts, i18n.English.T(i18n.ManifestFactsUnreadable)) {
		t.Error("the band does not say that the file could not be read")
	}
	if !saying(view.detailBox, "this is not a manifest") {
		t.Error("the card does not carry what went wrong")
	}
	if view.Unread() == nil {
		t.Error("a caller with nobody looking at the screen is told nothing went wrong")
	}
}

// An answer about a document nobody is waiting for any more is not drawn.
//
// The two buttons are out of reach for the length of a read, so nobody clicking
// can ask for a second one today. This stands in front of the day something
// else opens a document, because the failure it prevents is silent: an answer
// about the document before last drawn over a newer one, with nothing on screen
// saying which is which.
func TestAnAnswerAboutADocumentNobodyIsWaitingForIsNotDrawn(t *testing.T) {
	waiting := []func() func(){}
	view := New(i18n.English, nil, Readers{}, func(read func() func(), _ func(error) func()) {
		waiting = append(waiting, read)
	}, nil)
	view.Content()

	drawn := 0
	view.startReading("/runs/first.json", theManifest, func() func() { return func() { drawn = 1 } })
	view.startReading("/runs/second.json", theManifest, func() func() { return func() { drawn = 2 } })
	if len(waiting) != 2 {
		t.Fatalf("the screen asked for %d reads rather than the two this guard walks",
			len(waiting))
	}

	waiting[1]()()
	waiting[0]()()
	if drawn != 2 {
		t.Errorf("the answer drawn last was the one from read %d, so an older answer was "+
			"drawn over a newer one", drawn)
	}
}
