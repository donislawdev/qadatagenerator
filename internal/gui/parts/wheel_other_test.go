//go:build !darwin

package parts

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// The window divides what the toolkit hands over by the toolkit's step to count
// the notches turned. That step is a constant inside the toolkit rather than
// something it exports, so it is read from the toolkit's own source: a toolkit
// that changes it turns this red, where otherwise every notch would quietly move
// a different distance from the one the system is set to.
func TestTheWheelStepIsTheToolkitsStep(t *testing.T) {
	// The canary: the reading has to find a number that is not 25, or it could
	// be agreeing with the constant by never reading anything.
	if n, ok := scrollSpeedIn("const (\n\tscrollSpeed            = float32(31)\n)"); !ok || n != 31 {
		t.Fatalf("canary: a declared step of 31 was read as %v (%v)", n, ok)
	}

	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "fyne.io/fyne/v2").Output()
	if err != nil {
		t.Fatalf("finding the toolkit's source, which needs the go command go test puts on PATH: %v", err)
	}
	path := filepath.Join(strings.TrimSpace(string(out)), "internal", "driver", "glfw", "scroll_speed_default.go")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the toolkit's wheel step: %v", err)
	}
	n, ok := scrollSpeedIn(string(source))
	if !ok {
		t.Fatalf("%s no longer declares scrollSpeed, so the step the window divides by is unknown", path)
	}
	if said := float64(palette.ToolkitWheelStep); n != said {
		t.Errorf("the toolkit's wheel step is %v and palette.ToolkitWheelStep says %v, so a notch moves %.2f times what the system is set to",
			n, said, n/said)
	}
}

var scrollSpeedDeclared = regexp.MustCompile(`scrollSpeed\s*=\s*float32\(([0-9.]+)\)`)

// scrollSpeedIn reads the step out of the toolkit's declaration.
func scrollSpeedIn(source string) (float64, bool) {
	match := scrollSpeedDeclared.FindStringSubmatch(source)
	if match == nil {
		return 0, false
	}
	n, err := strconv.ParseFloat(match[1], 64)
	return n, err == nil
}

// Every kind of scrolling area this package makes moves, under a notch turned
// the way the driver turns it, as far as the system says: three lines out of
// the box, five, a whole view, or nothing. The wheel goes through test.Scroll,
// which finds the object under the pointer with the same function the driver
// of a real window uses (internal/driver/util.go of v2.8.2), so the table and
// the editor are reached through the pane laid over them or not at all.
//
// The canaries are the toolkit's own scroll and the toolkit's own multi line
// field, turned the same way. Both move the toolkit's step, and the check has
// to tell that apart from what the system asks for, or every area here could be
// back on the toolkit's step with this test green.
func TestEveryKindOfAreaMovesAsFarAsTheSystemSays(t *testing.T) {
	was := WheelLines
	t.Cleanup(func() { WheelLines = was })

	kinds := []struct {
		name  string
		build func() (fyne.CanvasObject, func() *container.Scroll)
	}{
		{"Scroll", func() (fyne.CanvasObject, func() *container.Scroll) {
			area := NewScroll(tallColumn())
			return area, func() *container.Scroll { return &area.Scroll }
		}},
		{"KeyScroll", func() (fyne.CanvasObject, func() *container.Scroll) {
			area := NewKeyScroll(tallColumn())
			return area.Framed(), func() *container.Scroll { return &area.Scroll.Scroll }
		}},
		{"KeyTable", func() (fyne.CanvasObject, func() *container.Scroll) {
			table := NewKeyTable(
				func() (int, int) { return 400, 2 },
				func() fyne.CanvasObject { return widget.NewLabel("cell") },
				func(widget.TableCellID, fyne.CanvasObject) {},
				func(bool) {},
			)
			return table.Framed(), func() *container.Scroll { return innerArea(table) }
		}},
		{"Editor", func() (fyne.CanvasObject, func() *container.Scroll) {
			editor := NewEditor(func(string) {})
			editor.SetText(strings.Repeat("a line of a recipe\n", 200))
			return editor.Framed(), func() *container.Scroll { return innerArea(editor) }
		}},
	}

	line := float32(0)
	for _, kind := range kinds {
		t.Run(kind.name, func(t *testing.T) {
			object, area := kind.build()
			window := test.NewWindow(object)
			defer window.Close()
			window.Resize(fyne.NewSize(480, 300))
			if line == 0 {
				line = wheelLine()
				bareAreasAreCaught(t, line)
			}
			scrolled := area()
			if scrolled == nil {
				t.Fatal("the area inside could not be found, so the wheel was not tried")
			}
			for _, lines := range []int{3, 5, -1, 0} {
				WheelLines = func() int { return lines }
				scrolled.ScrollToOffset(fyne.Position{})
				test.Scroll(window.Canvas(), centreOf(object), 0, -palette.ToolkitWheelStep)
				want := notchMoves(lines, line, scrolled.Size().Height)
				if !movedBy(scrolled.Offset.Y, want, roomIn(scrolled)) {
					t.Errorf("a notch with the system set to %d line(s) moved %.1f, and %.1f was asked for",
						lines, scrolled.Offset.Y, min(want, roomIn(scrolled)))
				}
			}
		})
	}
}

// bareAreasAreCaught is the canary of the test above, run once the window's
// theme is in place: the toolkit's own scroll and its own multi line field,
// turned one notch, have to be told apart from three lines.
func bareAreasAreCaught(t *testing.T, line float32) {
	t.Helper()
	want := notchMoves(palette.WheelLinesDefault, line, 1000)
	if want == palette.ToolkitWheelStep {
		t.Fatalf("three lines of %.1f are the toolkit's own step, so nothing here can tell the two apart", line)
	}

	bare := container.NewVScroll(tallColumn())
	window := test.NewWindow(bare)
	defer window.Close()
	window.Resize(fyne.NewSize(480, 300))
	test.Scroll(window.Canvas(), centreOf(bare), 0, -palette.ToolkitWheelStep)
	if movedBy(bare.Offset.Y, want, roomIn(bare)) {
		t.Fatalf("canary: the toolkit's own scroll moved %.1f and the check took it for %.1f", bare.Offset.Y, want)
	}

	field := widget.NewMultiLineEntry()
	field.SetText(strings.Repeat("a line of a recipe\n", 200))
	fieldWindow := test.NewWindow(field)
	defer fieldWindow.Close()
	fieldWindow.Resize(fyne.NewSize(480, 300))
	inner := innerArea(field)
	if inner == nil {
		t.Fatal("canary: the toolkit's own field has no area inside to measure")
	}
	test.Scroll(fieldWindow.Canvas(), centreOf(field), 0, -palette.ToolkitWheelStep)
	if movedBy(inner.Offset.Y, want, roomIn(inner)) {
		t.Fatalf("canary: the toolkit's own field moved %.1f and the check took it for %.1f", inner.Offset.Y, want)
	}
}

// The pane over a control of the toolkit hides itself rather than swallowing
// the wheel when the control's renderer does not hold exactly one scrolling
// area, finds the area again when the renderer hands back a different list,
// and costs a frame nothing when it does not.
func TestThePaneOverAToolkitControlFindsItsAreaOrStepsAside(t *testing.T) {
	area := container.NewVScroll(tallColumn())
	other := container.NewVScroll(tallColumn())

	none := withWheel(widget.NewSimpleRenderer(widget.NewLabel("no area")))
	if none.layer.Visible() || none.layer.area != nil {
		t.Error("a renderer with no scrolling area left the pane up, where it would swallow the wheel")
	}

	two := &changing{objects: []fyne.CanvasObject{area, other}}
	both := withWheel(two)
	if both.layer.Visible() || both.layer.area != nil {
		t.Error("a renderer with two scrolling areas left the pane up, handing the wheel to one of them by chance")
	}

	two.objects = []fyne.CanvasObject{widget.NewLabel("ground"), area}
	objects := both.Objects()
	if !both.layer.Visible() || both.layer.area != area {
		t.Error("the pane did not find the area after the renderer handed back a new list")
	}
	if len(objects) != 3 || objects[2] != both.layer {
		t.Errorf("the pane is not the last object, so the driver would find the area under it first: %v", objects)
	}

	if allocations := testing.AllocsPerRun(100, func() { both.Objects() }); allocations != 0 {
		t.Errorf("asking for the objects of an unchanged renderer allocated %.0f times, and a frame asks several times", allocations)
	}
}

// changing is a renderer whose list of objects the test sets.
type changing struct {
	objects []fyne.CanvasObject
}

func (c *changing) Destroy()                     {}
func (c *changing) Layout(fyne.Size)             {}
func (c *changing) MinSize() fyne.Size           { return fyne.Size{} }
func (c *changing) Objects() []fyne.CanvasObject { return c.objects }
func (c *changing) Refresh()                     {}

// tallColumn is content many screens tall.
func tallColumn() fyne.CanvasObject {
	return container.NewGridWrap(fyne.NewSize(10, 5000))
}

// innerArea is the one scrolling area of the toolkit among the objects a
// widget's renderer holds, which is where the table and the field keep theirs.
func innerArea(subject fyne.Widget) *container.Scroll {
	for _, object := range test.WidgetRenderer(subject).Objects() {
		if area, ok := object.(*container.Scroll); ok {
			return area
		}
	}
	return nil
}

// centreOf is the middle of an object, where a person puts the pointer.
func centreOf(object fyne.CanvasObject) fyne.Position {
	return fyne.NewPos(object.Position().X+object.Size().Width/2, object.Position().Y+object.Size().Height/2)
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

// roomIn is how far an area can move from the top.
func roomIn(area *container.Scroll) float32 {
	return area.Content.MinSize().Height - area.Size().Height
}

// movedBy says an area turned one notch from the top moved by want, or to its
// end when want is further than there is room for.
func movedBy(got, want, room float32) bool {
	want = min(want, room)
	return got > want-0.5 && got < want+0.5
}
