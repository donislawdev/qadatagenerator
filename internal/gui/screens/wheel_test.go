//go:build !darwin

package screens

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// TestEveryScrollingAreaMovesAsFarAsTheSystemSays turns the wheel over every
// scrolling area of every state, where a person would put the pointer, and asks
// each one to move as far as the system says a notch moves: three lines of text
// out of the box, five, a whole view, or nothing (B-420).
//
// # Why every state, and why the wheel goes in from outside
//
// A screen built around the toolkit's own scroll, or a control of the toolkit
// that keeps its scrolling inside it, would scroll at the toolkit's step and
// look no different in any picture or stored tree. So the question is put to
// the effect. The wheel goes through test.Scroll, which finds the object under
// the pointer with the same function the driver of a real window uses
// (internal/driver/util.go of v2.8.2), so an area is reached the way the
// pointer reaches it: past fields and nested areas, through the pane laid over
// a table or an editor, or not at all.
//
// # What it does not see
//
// What the wheel does over a field of one line or anything else that is not a
// scrolling area, the size the wheel is turned at being the one the trees are
// stored at, and macOS, where the toolkit's own wheel is kept on purpose
// (parts/wheel_darwin.go).
func TestEveryScrollingAreaMovesAsFarAsTheSystemSays(t *testing.T) {
	was := parts.WheelLines
	t.Cleanup(func() { parts.WheelLines = was })

	data := sampleWithSource(t)
	checked := map[string]int{}
	canaryRun := false
	for _, state := range All() {
		window := openedAt(data, state)
		if !canaryRun {
			theToolkitsOwnAreasAreCaught(t)
			canaryRun = true
		}
		for _, finding := range wheelFindings(window.Canvas(), checked) {
			t.Errorf("%s: %s", state.Name, finding)
		}
		window.Close()
	}

	t.Logf("areas turned, by kind, over %d states: %v", len(All()), checked)
	// What the window is made of, so a walk that found nothing cannot pass:
	// each kind of area has to have been turned at least once.
	for _, kind := range []string{"Scroll", "KeyScroll", "ScrolledRows", "KeyTable", "Editor"} {
		if checked[kind] == 0 {
			t.Errorf("no state has a %s with anything below its fold, so the wheel was never tried on one (checked %v)", kind, checked)
		}
	}
}

// theToolkitsOwnAreasAreCaught is the canary: the toolkit's own scroll and its
// own multi line field, put on a screen and turned the same way, have to come
// back as findings, or the test above passes with every area back on the
// toolkit's step.
func theToolkitsOwnAreasAreCaught(t *testing.T) {
	t.Helper()
	field := widget.NewMultiLineEntry()
	field.SetText(strings.Repeat("a line of a recipe\n", 200))
	bare := container.NewVScroll(container.NewGridWrap(fyne.NewSize(10, 5000)))
	window := test.NewWindow(container.NewGridWithColumns(2, bare, field))
	defer window.Close()
	window.Resize(fyne.NewSize(size.Width-1, size.Height))
	window.Resize(size)

	findings := wheelFindings(window.Canvas(), map[string]int{})
	if len(findings) != 2 {
		t.Fatalf("canary: the toolkit's own scroll and field gave %d finding(s) rather than two: %v", len(findings), findings)
	}
}

// openedAt builds a state in a window the way the stored trees are, at their
// size and laid out twice (render says why).
func openedAt(data Data, state State) fyne.Window {
	window := test.NewWindow(Assemble(i18n.English, data, state))
	window.Resize(size)
	Walk(window.Canvas(), state)
	window.Resize(fyne.NewSize(size.Width-1, size.Height))
	window.Resize(size)
	return window
}

// wheelFindings turns the wheel over every scrolling area a person can reach on
// the canvas and says which did not move as far as the system says, counting
// in checked the kinds of area turned.
func wheelFindings(canvas fyne.Canvas, checked map[string]int) []string {
	// The driver looks in the top overlay alone when one is open, so an area
	// under a list that is open is out of reach of the pointer too.
	root := canvas.Overlays().Top()
	if root == nil {
		root = canvas.Content()
	}
	line := fyne.MeasureText("Ag", palette.TypeSize(palette.TypeBody), fyne.TextStyle{}).Height
	defer func() { parts.WheelLines = func() int { return palette.WheelLinesDefault } }()

	var findings []string
	for _, found := range scrollingAreas(root) {
		points := gridIn(found.at, found.size, 7)
		var wrong []string
		for _, lines := range []int{palette.WheelLinesDefault, 5, -1, 0} {
			parts.WheelLines = func() int { return lines }
			if lines == 0 {
				// Nothing may move wherever the wheel is turned, so every
				// point is turned and what moved is added up.
				var moved float32
				for _, point := range points {
					step, _ := turnOnce(canvas, found.area, point)
					moved += step
				}
				if moved > 0.5 {
					wrong = append(wrong, fmt.Sprintf("%.1f rather than nothing at 0 lines", moved))
				}
				continue
			}
			moved, room, reached := turnWhereItReaches(canvas, found.area, points)
			if !reached {
				wrong = append(wrong, fmt.Sprintf("nothing at %d line(s), at every point tried, between %v and %v",
					lines, found.at, found.at.Add(fyne.NewPos(found.size.Width, found.size.Height))))
				continue
			}
			if lines == palette.WheelLinesDefault {
				checked[found.kind]++
			}
			want := min(notchMoves(lines, line, found.area.Size().Height), room)
			if moved < want-0.5 || moved > want+0.5 {
				wrong = append(wrong, fmt.Sprintf("%.1f rather than %.1f at %d line(s)", moved, want, lines))
			}
		}
		if len(wrong) > 0 {
			findings = append(findings, fmt.Sprintf("the %s moved %s for a notch", found.kind, strings.Join(wrong, ", ")))
		}
	}
	return findings
}

// turnWhereItReaches turns the wheel one notch at the first of points from
// which the area moves, and says how far it moved and how far it could have.
// A point over a field or a nested area hands the wheel to that instead, and
// which point that is changes as the area moves under the pointer, so the
// search starts again at every turn.
func turnWhereItReaches(canvas fyne.Canvas, area *container.Scroll, points []fyne.Position) (moved, room float32, reached bool) {
	for _, point := range points {
		moved, room := turnOnce(canvas, area, point)
		if moved > 0.01 {
			return moved, room, true
		}
	}
	return 0, 0, false
}

// turnOnce turns the wheel one notch at point, down when the area has room
// below and up when it is at its end, and says how far the area moved and how
// far it could have.
func turnOnce(canvas fyne.Canvas, area *container.Scroll, point fyne.Position) (moved, room float32) {
	before := area.Offset.Y
	end := area.Content.MinSize().Height - area.Size().Height
	if before < end-0.5 {
		test.Scroll(canvas, point, 0, -palette.ToolkitWheelStep)
		return area.Offset.Y - before, end - before
	}
	test.Scroll(canvas, point, 0, palette.ToolkitWheelStep)
	return before - area.Offset.Y, before
}

// notchMoves is how far one notch should move an area showing view units with
// the system set to lines: that many lines of text, a whole view for a negative
// number, nothing for nought.
func notchMoves(lines int, line, view float32) float32 {
	if lines < 0 {
		return view
	}
	return float32(lines) * line
}

// scrollingArea is an area that moves up and down, with the part of it a person
// can see, in the coordinates of the canvas.
type scrollingArea struct {
	area *container.Scroll
	kind string
	at   fyne.Position
	size fyne.Size
}

// scrollingAreas walks what is drawn under root and returns every area with
// something below its fold and some of itself in sight.
func scrollingAreas(root fyne.CanvasObject) []scrollingArea {
	var found []scrollingArea
	var walk func(object fyne.CanvasObject, at, clipAt fyne.Position, clipSize fyne.Size, owner string)
	walk = func(object fyne.CanvasObject, at, clipAt fyne.Position, clipSize fyne.Size, owner string) {
		if object == nil || !object.Visible() {
			return
		}
		position := at.Add(object.Position())
		switch object.(type) {
		case *parts.KeyTable:
			owner = "KeyTable"
		case *parts.Editor:
			owner = "Editor"
		}
		if area := toolkitScroll(object); area != nil {
			clipAt, clipSize = overlap(clipAt, clipSize, position, object.Size())
			room := area.Content.MinSize().Height - area.Size().Height
			vertical := area.Direction == container.ScrollVerticalOnly || area.Direction == container.ScrollBoth
			if vertical && room > 0.5 && clipSize.Width > 2 && clipSize.Height > 2 {
				found = append(found, scrollingArea{area: area, kind: kindOf(object, owner), at: clipAt, size: clipSize})
			}
		}
		var children []fyne.CanvasObject
		switch typed := object.(type) {
		case *fyne.Container:
			children = typed.Objects
		case fyne.Widget:
			children = test.WidgetRenderer(typed).Objects()
		}
		for _, child := range children {
			walk(child, position, clipAt, clipSize, owner)
		}
	}
	walk(root, fyne.Position{}, fyne.Position{X: -1e6, Y: -1e6}, fyne.NewSize(2e6, 2e6), "")
	return found
}

// kindOf names an area by what built it: one of this window's types, or the
// control of the toolkit that keeps it inside, or the toolkit itself.
func kindOf(object fyne.CanvasObject, owner string) string {
	switch object.(type) {
	case *parts.ScrolledRows:
		return "ScrolledRows"
	case *parts.KeyScroll:
		return "KeyScroll"
	case *parts.Scroll:
		return "Scroll"
	}
	if owner != "" {
		return owner
	}
	return fmt.Sprintf("toolkit's own scroll (%T)", object)
}

// toolkitScroll is the toolkit's scroll an object is or embeds at any depth,
// found by its type rather than by a list of ours, so that a scrolling area of
// a type nobody has written yet is turned too.
func toolkitScroll(object fyne.CanvasObject) *container.Scroll {
	if area, ok := object.(*container.Scroll); ok {
		return area
	}
	value := reflect.ValueOf(object)
	if value.Kind() != reflect.Pointer || value.IsNil() || value.Elem().Kind() != reflect.Struct {
		return nil
	}
	return embeddedScroll(value.Elem())
}

var toolkitScrollType = reflect.TypeFor[container.Scroll]()

// embeddedScroll looks through the embedded fields of a struct for the
// toolkit's scroll.
func embeddedScroll(value reflect.Value) *container.Scroll {
	for i := range value.NumField() {
		field := value.Type().Field(i)
		if !field.Anonymous || field.Type.Kind() != reflect.Struct {
			continue
		}
		if field.Type == toolkitScrollType {
			if area, ok := value.Field(i).Addr().Interface().(*container.Scroll); ok {
				return area
			}
			continue
		}
		if found := embeddedScroll(value.Field(i)); found != nil {
			return found
		}
	}
	return nil
}

// overlap is the part of two rectangles they share.
func overlap(at fyne.Position, size fyne.Size, otherAt fyne.Position, otherSize fyne.Size) (fyne.Position, fyne.Size) {
	left, top := max(at.X, otherAt.X), max(at.Y, otherAt.Y)
	right := min(at.X+size.Width, otherAt.X+otherSize.Width)
	bottom := min(at.Y+size.Height, otherAt.Y+otherSize.Height)
	return fyne.NewPos(left, top), fyne.NewSize(max(right-left, 0), max(bottom-top, 0))
}

// gridIn is n by n points spread inside a rectangle, the middle one first,
// where a person puts the pointer.
func gridIn(at fyne.Position, size fyne.Size, n int) []fyne.Position {
	points := []fyne.Position{fyne.NewPos(at.X+size.Width/2, at.Y+size.Height/2)}
	for row := range n {
		for column := range n {
			x := at.X + size.Width*(float32(column)+0.5)/float32(n)
			y := at.Y + size.Height*(float32(row)+0.5)/float32(n)
			points = append(points, fyne.NewPos(x, y))
		}
	}
	return points
}
