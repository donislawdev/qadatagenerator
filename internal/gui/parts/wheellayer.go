package parts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// wheelLayer is a pane of nothing laid over a control that keeps its scrolling
// area inside the toolkit, so that one notch moves that area as far as it moves
// every other area of this window (Scroll).
//
// # Why a layer, when Scroll is an extension
//
// The table and the recipe editor are the toolkit's own widgets, and each builds
// its scrolling area inside its renderer, as a type of the toolkit that nothing
// outside can extend: widget.Table at widget/table.go:164 and widget.Entry at
// widget/entry.go:191 of v2.8.2. The driver hands a wheel to the last object
// under the pointer that answers Scrolled, internal/driver/util.go:57, which
// is that inner area, so neither control ever hears the wheel. A pane that
// answers Scrolled and is drawn after everything else the control holds is the
// last such object, so the wheel reaches it instead, and it hands the inner
// area the event the driver would have, with the window's distance in it.
//
// # Why nothing else changes
//
// The driver looks for the object answering each kind of event on its own
// (window.go:686 for the wheel, other lookups for a press, a hover, a drag and
// the shape of the pointer), and this pane answers the wheel and nothing else.
// So a click, a hover, a drag on a bar and the caret all go on reaching the
// control under it, as they did. The pane covers the whole control rather than
// its area alone, so a wheel over the row of headings or the edge of the field
// scrolls the area too, where the toolkit did nothing.
//
// # What finds the area, and what holds it
//
// The area is the one toolkit scroll among the objects the toolkit's renderer
// hands back, the order and the count of which are the toolkit's choice and
// could change in any release. When none is found, or more than one, the pane
// hides and the wheel goes back to the toolkit's step, which is no worse than
// before. TestEveryScrollingAreaMovesAsFarAsTheSystemSays in
// internal/gui/screens turns the wheel over every area of every screen, so a
// release that moves the area out of reach turns that red rather than a
// control quietly scrolling at the toolkit's step again.
type wheelLayer struct {
	widget.BaseWidget
	area *container.Scroll
}

// newWheelLayer makes the pane for the area found among objects.
func newWheelLayer(objects []fyne.CanvasObject) *wheelLayer {
	layer := &wheelLayer{}
	layer.ExtendBaseWidget(layer)
	layer.find(objects)
	return layer
}

// find takes the area out of the renderer's objects, and hides the pane when
// there is not exactly one.
func (l *wheelLayer) find(objects []fyne.CanvasObject) {
	l.area = nil
	found := 0
	for _, object := range objects {
		if area, ok := object.(*container.Scroll); ok {
			l.area = area
			found++
		}
	}
	if found != 1 {
		l.area = nil
		l.Hide()
		return
	}
	l.Show()
}

// Scrolled hands the area the wheel with the window's distance in it.
func (l *wheelLayer) Scrolled(event *fyne.ScrollEvent) {
	if l.area == nil {
		return
	}
	l.area.Scrolled(wheelTurned(event, l.area.Size().Height))
}

// CreateRenderer draws nothing. The pane is there to be found under the
// pointer, not to be seen.
func (l *wheelLayer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(&fyne.Container{})
}

// wheeled is a renderer of the toolkit with a wheelLayer laid over all of it.
//
// Everything but the list of objects and where they go is the toolkit's
// renderer, unchanged.
type wheeled struct {
	fyne.WidgetRenderer
	layer *wheelLayer
	// objects is what the toolkit's renderer last handed back with the pane
	// after them, kept so that a frame asking several times allocates nothing.
	objects []fyne.CanvasObject
}

// withWheel lays the pane over a renderer of the toolkit.
func withWheel(toolkit fyne.WidgetRenderer) fyne.WidgetRenderer {
	inner := toolkit.Objects()
	wrapped := &wheeled{WidgetRenderer: toolkit, layer: newWheelLayer(inner)}
	wrapped.keep(inner)
	return wrapped
}

// Layout lets the toolkit place its objects, then spreads the pane over them.
func (w *wheeled) Layout(size fyne.Size) {
	w.WidgetRenderer.Layout(size)
	w.layer.Move(fyne.Position{})
	w.layer.Resize(size)
}

// Objects is the toolkit's objects with the pane last, so that the driver,
// which keeps the last object under the pointer that answers Scrolled, finds
// the pane rather than the area under it. When the toolkit's renderer hands back
// a different list, the area is looked for again.
func (w *wheeled) Objects() []fyne.CanvasObject {
	inner := w.WidgetRenderer.Objects()
	if !w.holds(inner) {
		w.layer.find(inner)
		w.keep(inner)
	}
	return w.objects
}

// keep stores the toolkit's objects with the pane after them.
func (w *wheeled) keep(inner []fyne.CanvasObject) {
	w.objects = append(append(make([]fyne.CanvasObject, 0, len(inner)+1), inner...), w.layer)
}

// holds says whether the list kept is the toolkit's current one with the pane
// after it.
func (w *wheeled) holds(inner []fyne.CanvasObject) bool {
	if len(w.objects) != len(inner)+1 {
		return false
	}
	for i, object := range inner {
		if w.objects[i] != object {
			return false
		}
	}
	return true
}
