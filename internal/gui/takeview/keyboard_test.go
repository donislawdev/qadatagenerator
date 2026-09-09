package takeview

import (
	"image"
	"image/color"
	"math"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/fynetheme"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
)

// onACanvas is this screen in a window, with a file open and damage ticked.
//
// The window is a real one of the toolkit's test driver rather than nothing,
// because everything below is about the keyboard and focus without a canvas has
// nothing to land on.
func onACanvas(t *testing.T) (*View, *held, fyne.Canvas) {
	t.Helper()
	dir := t.TempDir()
	window := test.NewWindow(nil)
	t.Cleanup(window.Close)
	view, runs := opened(t, window, sourceFile(t, dir), dir)
	window.SetContent(view.Content())
	window.Resize(fyne.NewSize(1342, 900))
	return view, runs, window.Canvas()
}

// standOn walks the keyboard to a control the way a person does, by pressing
// Tab until it arrives.
//
// It presses rather than calling Focus, because pressing is what proves the
// control is reachable at all. A test that put the keyboard where it wanted it
// would pass on a screen no keyboard can walk.
func standOn(t *testing.T, canvas fyne.Canvas, wanted fyne.Focusable) {
	t.Helper()
	for pressed := 1; pressed <= walkLength; pressed++ {
		canvas.FocusNext()
		if canvas.Focused() == wanted {
			return
		}
	}
	t.Fatalf("the keyboard never reached %v in %d presses of Tab, so it cannot be "+
		"reached by keyboard at all", wanted, walkLength)
}

// walkLength is more presses of Tab than this screen has controls, so a walk
// that does not find something has been all the way round.
const walkLength = 60

// The keyboard meets the file before the settings, and the run last of all.
//
// 🔴 This is the guard for a comment that was false the day it was written.
// Measured on 2026-08-24 on a real canvas: the button that chooses a file
// stands at the very top of the screen and was the nineteenth and LAST thing
// the keyboard reached, after every setting and after the button that damages
// the file. The cause is that the toolkit's convenient border appends its edges
// after its middle, so the pane of settings walked in front of the bar above it.
//
// Nothing about the drawing said so, because it draws identically either way.
// That is the same failure the manifest browser paid for on 2026-08-18, on the
// screen next door, with a comment on this one already warning about it.
func TestTheKeyboardMeetsTheFileBeforeEverythingItIsUsedFor(t *testing.T) {
	view, _, canvas := onACanvas(t)

	canvas.FocusNext()
	first, isButton := canvas.Focused().(*parts.Button)
	if !isButton || first != view.chooseFile {
		t.Fatalf("the first thing the keyboard reaches is %s rather than the button that "+
			"chooses a file, which is the control at the top of the screen",
			describe(canvas.Focused()))
	}

	// From there the walk has to arrive at the settings and only then at the run,
	// which is the order the three parts are read in.
	reached := map[fyne.Focusable]int{}
	for pressed := 1; pressed <= walkLength; pressed++ {
		canvas.FocusNext()
		if canvas.Focused() == fyne.Focusable(view.chooseFile) {
			break
		}
		if _, seen := reached[canvas.Focused()]; !seen {
			reached[canvas.Focused()] = pressed
		}
	}
	seed, damage := reached[view.seed], reached[view.damage]
	if seed == 0 || damage == 0 {
		t.Fatalf("one press of Tab per control did not reach both a setting and the run: "+
			"the seed at %d and the button that damages the file at %d", seed, damage)
	}
	if damage < seed {
		t.Errorf("the keyboard reaches the button that damages the file at press %d and "+
			"a setting of that run at press %d, so it offers the act before what the act "+
			"is made of", damage, seed)
	}
}

// The keyboard follows the two buttons of a run as they take each other's place.
//
// 🔴 Measured on 2026-08-24 before this was true. Starting a run from the
// keyboard left the keyboard on a button no longer drawn, so nothing on the
// screen said where a person was standing. The way out was not where anybody
// would look for it either: the toolkit skips a disabled control BEFORE it
// compares it to the one holding focus, so it never finds its place in the walk
// and the next press of Tab answers with the first control on the screen. A
// person who stopped a run and pressed Tab was thrown to the top of the form.
func TestTheKeyboardFollowsTheButtonsOfARun(t *testing.T) {
	view, runs, canvas := onACanvas(t)
	standOn(t, canvas, view.damage)

	canvas.Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if canvas.Focused() != fyne.Focusable(view.stop) {
		t.Fatalf("a run was started from the keyboard and the keyboard is on %s. The "+
			"button it was standing on has left the screen, so there is nothing showing "+
			"where a person is", describe(canvas.Focused()))
	}

	runs.carryOut(t)
	if canvas.Focused() != fyne.Focusable(view.damage) {
		t.Errorf("the run ended and the keyboard is on %s rather than on the button that "+
			"has taken the place of the one it was standing on", describe(canvas.Focused()))
	}
}

// A run does not leave the keyboard standing on a setting it has put out of
// reach.
//
// 🔴 The tick boxes are the case worth the test. They are the children of a
// group, so this package never holds them and could not name them in a list if
// it wanted to. What reaches them is asking the control the keyboard is on
// whether it is still usable, rather than looking it up.
func TestARunDoesNotLeaveTheKeyboardOnSomethingItPutOutOfReach(t *testing.T) {
	for _, where := range []struct {
		what  string
		reach func(*View) fyne.Focusable
	}{
		{"a text setting", func(v *View) fyne.Focusable { return v.seed }},
		{"a list setting", func(v *View) fyne.Focusable { return v.perRow }},
		{"a tick box for a kind of damage", nil},
	} {
		t.Run(where.what, func(t *testing.T) {
			view, _, canvas := onACanvas(t)
			if where.reach == nil {
				// The first tick box, reached the way a person reaches it: past
				// the button that chooses a file and past the two panes, which
				// are places to stand as well as places to read.
				for presses := 0; presses < 4; presses++ {
					canvas.FocusNext()
				}
				if _, isTick := canvas.Focused().(*widget.Check); !isTick {
					t.Fatalf("the fourth thing the keyboard reaches is %s rather than a "+
						"tick box, so this case is testing something else",
						describe(canvas.Focused()))
				}
			} else {
				standOn(t, canvas, where.reach(view))
			}

			view.Start()
			if canvas.Focused() != fyne.Focusable(view.stop) {
				t.Errorf("a run began while the keyboard stood on %s, which the run has "+
					"disabled, and the keyboard is still on %s", where.what,
					describe(canvas.Focused()))
			}
		})
	}
}

// The keyboard is given back to somebody who started a run with it and then
// reached for the mouse.
//
// 🔴 This guard used to reach the same branch a different way, and the way it
// used is gone. Until the two panes of this screen became reachable without a
// mouse, the moment between asking a run to stop and the run answering was the
// one moment here with nothing at all the keyboard could hold: every setting
// out of reach and the way to stop disabled so that a second press means
// nothing. A Tab pressed there was answered with nothing, and the person's
// place was gone. Measured on 2026-08-24, and on 2026-09-01 the two scrolling
// areas took that moment away by being somewhere to stand.
//
// The branch it protected is still reachable and still worth protecting,
// because the mouse also leaves the keyboard nowhere: the toolkit clears focus
// on every tap. Somebody who begins a run from the keyboard and then stops it
// with the pointer is standing nowhere and is owed a place, and telling them
// apart from somebody who only ever used the pointer is what the screen
// remembers rather than looks at.
func TestTheKeyboardIsGivenBackToSomebodyWhoReachedForTheMouse(t *testing.T) {
	view, runs, canvas := onACanvas(t)
	standOn(t, canvas, view.damage)
	canvas.Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if canvas.Focused() != fyne.Focusable(view.stop) {
		t.Fatalf("the run did not begin from the keyboard, which left it on %s",
			describe(canvas.Focused()))
	}

	test.Tap(view.stop)
	if canvas.Focused() != nil {
		t.Skipf("tapping a button no longer clears the keyboard, it left it on %s, so the "+
			"moment this guard is about no longer exists", describe(canvas.Focused()))
	}

	runs.carryOut(t)
	if canvas.Focused() != fyne.Focusable(view.damage) {
		t.Errorf("a run was started from the keyboard and stopped with the mouse, which "+
			"left the keyboard nowhere, and the end of the run put it on %s",
			describe(canvas.Focused()))
	}
}

// A run started with the mouse leaves the keyboard where the mouse left it,
// which is nowhere.
//
// 🔴 This is the guard against the rule above over reaching, and it is the
// reason the screen remembers whether it had the keyboard rather than looking at
// the end. The toolkit clears focus on every tap of a button, so a run started
// with the mouse and a Tab pressed into nothing look identical afterwards: both
// are a keyboard standing on nothing. Giving a focus ring to somebody who has
// only ever used the mouse is not a fix, it is a mark appearing for no reason.
func TestARunStartedWithTheMouseLeavesTheKeyboardAlone(t *testing.T) {
	view, runs, canvas := onACanvas(t)

	test.Tap(view.damage)
	if canvas.Focused() != nil {
		t.Skipf("tapping a button no longer clears the keyboard, it left it on %s, so the "+
			"two cases this guard tells apart are no longer alike", describe(canvas.Focused()))
	}
	if view.stage != stageRunning {
		t.Fatalf("tapping the button left the screen in stage %d rather than in a run", view.stage)
	}

	runs.carryOut(t)
	if canvas.Focused() != nil {
		t.Errorf("a run was begun and ended with the mouse and the keyboard is now on %s. "+
			"Nobody asked for it to be anywhere", describe(canvas.Focused()))
	}
}

// Somebody who walks away from a run they started is not walked back.
//
// 🔴 The names in the sidebar stay in reach for the whole of a run, so leaving
// one screen while it works is something a person can do and this screen must
// not undo. The other name is stood in for by a control of the toolkit's own in
// the same window, because what matters is that it is reachable and usable while
// this screen is not, which is exactly what the sidebar is.
func TestSomebodyWhoWalksAwayFromARunIsNotWalkedBack(t *testing.T) {
	dir := t.TempDir()
	window := test.NewWindow(nil)
	defer window.Close()
	view, runs := opened(t, window, sourceFile(t, dir), dir)
	elsewhere := widget.NewButton("another screen", func() {})
	window.SetContent(container.NewBorder(elsewhere, nil, nil, nil, view.Content()))
	window.Resize(fyne.NewSize(1342, 900))
	canvas := window.Canvas()

	standOn(t, canvas, view.damage)
	canvas.Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if canvas.Focused() != fyne.Focusable(view.stop) {
		t.Fatalf("the run did not begin from the keyboard, which left it on %s",
			describe(canvas.Focused()))
	}

	canvas.Focus(elsewhere)
	runs.carryOut(t)
	if canvas.Focused() != fyne.Focusable(elsewhere) {
		t.Errorf("somebody left this screen while its run was going on and the end of the "+
			"run took the keyboard back to %s", describe(canvas.Focused()))
	}
}

// When the button that would take the keyboard back cannot have it, the
// keyboard goes to the one control this screen always has.
//
// 🔴 The toolkit answers a focus it cannot carry out with silence rather than
// with a failure, so without this the keyboard would simply stay on a button
// that is no longer drawn. The condition is planted rather than waited for: the
// settings are frozen during a run, so the only way to reach it from outside is
// to change them behind the freeze, which is what a future setting reachable
// mid run would do by itself.
func TestTheKeyboardHasSomewhereToGoWhenTheButtonCannotTakeIt(t *testing.T) {
	view, runs, canvas := onACanvas(t)
	standOn(t, canvas, view.damage)
	canvas.Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})

	// Nothing ordered means the settings would be refused, so the button that
	// damages the file is disabled when the run ends and cannot be stood on.
	view.chosen.Layers = nil

	runs.carryOut(t)
	if !view.damage.Disabled() {
		t.Fatal("the button that damages the file is not disabled, so this guard is not " +
			"testing what it says it is")
	}
	if canvas.Focused() != fyne.Focusable(view.chooseFile) {
		t.Errorf("the run ended, the button that would take the keyboard back cannot have "+
			"it, and the keyboard is on %s rather than on the button that chooses a file",
			describe(canvas.Focused()))
	}
}

// Every kind of control on this screen shows the keyboard in pixels.
//
// 🔴 This screen carries four kinds of control and the rule that focus must be
// visible is a threshold on a colour, which says nothing about whether the
// colour is ever painted. The failure it misses is the one measured next door on
// 2026-08-18: a control painted in the colour focus arrives in has nothing left
// to show that focus arrived, and the first press of Tab changed not one pixel.
// So this asks the pixels under the theme the product really wears.
func TestTheKeyboardCanBeSeenOnEveryKindOfControlHere(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, fynetheme.Current())

	for _, kind := range []struct {
		what string
		// floor is what this kind of control measures today. It is a ratchet in
		// the same sense as the ceilings in internal/guard: it may rise and it
		// may never fall.
		floor float64
		build func() (fyne.CanvasObject, fyne.Focusable)
	}{
		{"a tick box for a kind of damage", visibleMark, func() (fyne.CanvasObject, fyne.Focusable) {
			box := widget.NewCheck("L2", func(bool) {})
			return box, box
		}},
		{"a text setting", visibleMark, func() (fyne.CanvasObject, fyne.Focusable) {
			field := widget.NewEntry()
			return field, field
		}},
		{"a list setting", visibleMark, func() (fyne.CanvasObject, fyne.Focusable) {
			list := widget.NewSelect([]string{"one", "two"}, func(string) {})
			return list, list
		}},
		{"a button", visibleMark, func() (fyne.CanvasObject, fyne.Focusable) {
			button := parts.NewButton(i18n.English.T(i18n.TakeRun), func() {})
			return button, button
		}},
	} {
		drawn, takesFocus := kind.build()
		window := test.NewWindow(container.NewPadded(drawn))
		window.Resize(fyne.NewSize(300, 80))
		canvas := window.Canvas()

		before := frozenPicture(canvas.Capture())
		canvas.Focus(takesFocus)
		mark, covering := largestChange(before, canvas.Capture())
		window.Close()

		if covering == 0 {
			t.Errorf("the keyboard arriving on %s changes not one pixel, so there is "+
				"nothing on the screen saying where a person is standing", kind.what)
			continue
		}
		if ratio := contrast(mark.was, mark.now); ratio < kind.floor {
			t.Errorf("the keyboard arriving on %s paints %d pixels at a contrast of %.2f "+
				"against what they replaced, and the threshold is %.2f. A change this "+
				"faint is a mark in the picture and not on anybody's screen",
				kind.what, covering, ratio, kind.floor)
		}
	}
}

// visibleMark is what WCAG 1.4.11 asks of a part of an interface that carries
// meaning, and docs/17 rule 4.9 asks of this one by name.
//
// 🔴 Counting changed pixels was the first shape of the check above and it is
// not enough, which was measured rather than reasoned: with the colour of focus
// set equal to the ground, the tick box still changed 77 pixels. They are the
// smoothed edges of a rounded rectangle painted in the colour that was already
// there, so something changed and nothing could be seen. A guard answering
// "did anything change" would have called that a mark.
const visibleMark = 3.0

// paint is a colour that arrived where another colour was.
type paint struct{ was, now color.Color }

// largestChange finds the colour the keyboard puts down over the widest area,
// and says what was underneath it.
//
// The widest area rather than the strongest difference, because the smoothed
// edge of any shape carries a handful of pixels of every colour in between. The
// body of the mark is what a person sees, and the body is what covers ground.
func largestChange(before, after image.Image) (paint, int) {
	counted := map[paint]int{}
	bounds := before.Bounds()
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			was, now := before.At(x, y), after.At(x, y)
			r1, g1, b1, a1 := was.RGBA()
			r2, g2, b2, a2 := now.RGBA()
			if r1 == r2 && g1 == g2 && b1 == b2 && a1 == a2 {
				continue
			}
			counted[paint{was: was, now: now}]++
		}
	}
	var widest paint
	most := 0
	for which, covering := range counted {
		if covering > most {
			widest, most = which, covering
		}
	}
	return widest, most
}

// contrast is the ratio WCAG defines between two colours.
//
// It is written here rather than taken from the contrast bridge in tools,
// because the two answer different questions. That one asks whether the palette
// this project wrote down meets its thresholds, and this one asks whether the
// ink that actually reached the screen does. A palette can be right while
// nothing paints it, which is the whole family of failure this screen has been
// paying for.
func contrast(first, second color.Color) float64 {
	lighter, darker := relativeLuminance(first), relativeLuminance(second)
	if lighter < darker {
		lighter, darker = darker, lighter
	}
	return (lighter + 0.05) / (darker + 0.05)
}

func relativeLuminance(of color.Color) float64 {
	r, g, b, _ := of.RGBA()
	return 0.2126*channel(r) + 0.7152*channel(g) + 0.0722*channel(b)
}

// channel undoes the transfer function of the sRGB colour space, which is what
// makes the sum above a measure of light rather than of stored numbers.
func channel(raw uint32) float64 {
	part := float64(raw) / 65535
	if part <= 0.04045 {
		return part / 12.92
	}
	return math.Pow((part+0.055)/1.055, 2.4)
}

// frozenPicture keeps a render, because the canvas paints over the one it
// returns.
func frozenPicture(source image.Image) image.Image {
	bounds := source.Bounds()
	kept := image.NewNRGBA(bounds)
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			kept.Set(x, y, source.At(x, y))
		}
	}
	return kept
}

// describe says what the keyboard is standing on, in words that can be matched
// to the screen.
func describe(standing fyne.Focusable) string {
	switch typed := standing.(type) {
	case nil:
		return "nothing at all"
	case *parts.Button:
		return "the button " + typed.Text
	case *widget.Check:
		return "the tick box " + typed.Text
	case *widget.Entry:
		return "a text setting holding " + typed.Text
	case *widget.Select:
		return "a list setting holding " + typed.Selected
	default:
		return "something this test cannot name"
	}
}
