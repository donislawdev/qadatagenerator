package parts

import "fyne.io/fyne/v2"

// rowScroll is the scrolling column of a panel of rows, whose wheel stops with
// a whole row at the top of the view.
//
// # Why the wheel does not stop where it lands
//
// A notch moves three lines of text out of the box (Scroll), and a row of a
// panel is not a line of text, so a notch left the panel standing in the middle
// of a row: the top row cut at its head and the bottom one at its foot. Seen
// under Xvfb on 2026-11-25 on the list of an editor's suggestions, two notches
// over the generators (B-429). The panel is cut to end on a whole row
// (picker.Fitted) and the arrows keep it there (picker.reveal), so the wheel was
// the one way into half a row. Both panels a picker makes scroll through this
// type, the chooser's and the editor's.
//
// # How it stops
//
// The wheel moves an aim by the system's distance, the way every area of the
// window moves, and the view stands on the stop nearest the aim. The aim is kept
// apart from the offset, so that turns shorter than half a row add up rather
// than each being rounded away. A touchpad sends a fraction of a notch at a time,
// and without the aim it would never get past the first row. The end of the list
// is a stop too, so the last row is reached whole also where the rows above it
// do not divide the view.
//
// The bar is left as the toolkit has it: a drag puts the view where the pointer
// is, and the next notch starts from there.
type rowScroll struct {
	Scroll
	rows *fyne.Container
	// aim is where the wheel has taken the view, before the stop on a row.
	aim float32
	// stood is the offset this type last set. An offset that differs from it
	// was set by something else, the arrows or the bar, and the aim starts
	// over from there.
	stood float32
}

// newRowScroll makes a column of rows scrollable up and down.
func newRowScroll(rows *fyne.Container) *rowScroll {
	area := &rowScroll{rows: rows}
	area.setUp(rows)
	// Before the toolkit's own constructor would, for the reason NewScroll
	// gives: the wheel has to reach this type and not the scroll it embeds.
	area.ExtendBaseWidget(area)
	return area
}

// Scrolled moves the aim by the system's distance for the notches turned and
// stands the view on the stop nearest it.
func (s *rowScroll) Scrolled(event *fyne.ScrollEvent) {
	if s.Offset.Y != s.stood {
		s.aim = s.Offset.Y
	}
	moved := wheelTurned(event, s.Size().Height)
	s.aim = min(max(s.aim-moved.Scrolled.DY, 0), s.end())
	// The toolkit moves first, because it is what shows the bars while the
	// wheel turns. Both moves are made before the next frame is drawn.
	s.Scroll.Scroll.Scrolled(moved)
	s.ScrollToOffset(fyne.NewPos(s.Offset.X, s.StopNear(s.aim)))
	s.stood = s.Offset.Y
}

// StopNear is where the view stands once the wheel has taken it to aim: the top
// of the row nearest it, or the end of the list.
func (s *rowScroll) StopNear(aim float32) float32 {
	return stopNear(aim, s.end(), s.tops())
}

// end is the furthest the view can go down.
func (s *rowScroll) end() float32 {
	return max(s.Content.MinSize().Height-s.Size().Height, 0)
}

// tops are where the rows on screen begin, down the column. A heading is a
// stop as well as a row, since a group shown from its heading reads whole.
func (s *rowScroll) tops() []float32 {
	tops := make([]float32, 0, len(s.rows.Objects))
	for _, item := range s.rows.Objects {
		if item.Visible() {
			tops = append(tops, item.Position().Y)
		}
	}
	return tops
}

// stopNear is the stop nearest aim: the top of a row, or end, the furthest the
// view can go. A row that begins past end stops at end, because the view cannot
// stand on it. Of two stops as near, the one higher up is taken.
func stopNear(aim, end float32, tops []float32) float32 {
	best := end
	for _, top := range tops {
		top = min(top, end)
		if distance(top, aim) < distance(best, aim) || (distance(top, aim) == distance(best, aim) && top < best) {
			best = top
		}
	}
	return best
}

// distance is how far apart two places down the column are.
func distance(a, b float32) float32 {
	if a > b {
		return a - b
	}
	return b - a
}
