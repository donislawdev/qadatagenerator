package parts

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// 🔴 TestEveryExtensionLaysOutItsChildren asks the one question that the
// toolkit's extension mechanism answers silently when it is got wrong.
//
// # What broke, and why nothing saw it
//
// parts.Select was built as &Select{Select: *widget.NewSelect(...)}. The
// toolkit's constructor ends in ExtendBaseWidget(itself), so the struct it
// returns already carries an impl naming ITSELF, and copying the struct copies
// that pointer. ExtendBaseWidget is `if w.super() != nil { return }`, so our
// call did nothing, and impl went on naming the discarded widget.
// BaseWidget.Resize sets w.size and then calls cache.Renderer(impl).Layout -
// so the SIZE lands on the widget that is drawn and the LAYOUT lands on the
// renderer of the widget that is not.
//
// The failure is total and completely quiet. The widget reports its size
// correctly, the tree shows it at 611x27, no test fails, nothing panics, and
// the value is simply never drawn: every child of the live renderer stays 0x0,
// so the text is truncated to an ellipsis and painted at -4,-4. It stood on 54
// choosers across 22 screen states, on both screens that have one, and it was
// found by looking at a picture rather than by any guard here.
//
// # Why this asks about children rather than about impl
//
// impl is unexported, so no test can name it. What it decides is observable in
// exactly one place: whether a resize reaches the renderer that is drawn. That
// is what a person sees, and it is what this asks.
//
// # Why the whole family rather than the chooser
//
// A guard written for the one member somebody noticed is not a guard for the
// family, and this package holds six extensions of toolkit widgets. Any of
// them can acquire the same fault the day someone reaches for a constructor
// the toolkit exports, which is the natural thing to reach for - it is what
// the chooser did. KeyTable already carries a sentence describing this exact
// trap, and the chooser two files away broke anyway.
func TestEveryExtensionLaysOutItsChildren(t *testing.T) {
	// A size no minimum can already equal, so Resize cannot return early on
	// the `size == w.Size()` line and hide the fault it is asked about.
	const w, h = 611, 44
	size := fyne.NewSize(w, h)

	family := []struct {
		name  string
		build func() fyne.Widget
	}{
		{"Button", func() fyne.Widget { return NewButton("Choose a file", func() {}) }},
		{"PrimaryButton", func() fyne.Widget {
			return NewPrimaryButton("Generate this dataset", func() {})
		}},
		{"Editor", func() fyne.Widget { return NewEditor(func(string) {}) }},
		{"KeyScroll", func() fyne.Widget {
			return NewKeyScroll(widget.NewLabel("a body long enough to scroll"))
		}},
		{"KeyTable", func() fyne.Widget {
			return NewKeyTable(
				func() (int, int) { return 3, 2 },
				func() fyne.CanvasObject { return widget.NewLabel("cell") },
				func(widget.TableCellID, fyne.CanvasObject) {},
				func(bool) {},
			)
		}},
		{"Select", func() fyne.Widget {
			chooser := NewSelect([]string{"pesel", "nip"}, func(string) {})
			chooser.SetSelected("pesel")
			return chooser
		}},
	}

	for _, member := range family {
		t.Run(member.name, func(t *testing.T) {
			subject := member.build()
			window := test.NewWindow(subject)
			defer window.Close()
			window.Resize(size)
			subject.Resize(size)

			if got := subject.Size(); got != size {
				t.Fatalf("the widget itself did not take the size: got %v, want %v", got, size)
			}

			children := test.WidgetRenderer(subject).Objects()
			if len(children) == 0 {
				t.Fatalf("the renderer has no objects at all, so there is nothing to draw")
			}

			laid := 0
			for _, child := range children {
				if child.Size().Width > 0 && child.Size().Height > 0 {
					laid++
				}
			}
			if laid == 0 {
				t.Errorf(
					"every one of the %d objects in the renderer is still 0x0 after a"+
						" resize to %v, so the widget is drawn empty while reporting the"+
						" right size.\nThis is the impl fault: the constructor copied a"+
						" toolkit widget by value, so ExtendBaseWidget was a no-op and"+
						" Layout went to the renderer of a discarded widget. Build the"+
						" struct empty and set its fields by hand, as KeyTable does.",
					len(children), size,
				)
			}
		})
	}
}

// 🔴 TestChooserDrawsItsValue is the other half, and it is here because the
// size question above would stay green for a chooser that lays its children
// out and still shows nothing.
//
// What a person lost when the impl fault was live was not geometry, it was the
// VALUE: the text was truncated to an ellipsis because it was measured against
// a width of zero. So this asks for the value itself, in the markup that the
// rasteriser draws from.
func TestChooserDrawsItsValue(t *testing.T) {
	chooser := NewSelect([]string{"pesel", "nip"}, func(string) {})
	chooser.SetSelected("pesel")

	window := test.NewWindow(chooser.OnScreen())
	defer window.Close()
	window.Resize(fyne.NewSize(611, 44))

	markup := test.RenderToMarkup(window.Canvas())
	if !contains(markup, "pesel") {
		t.Errorf(
			"the chosen value is not in the rendered markup, so the control is"+
				" drawn blank.\nmarkup:\n%s", markup,
		)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
