package kept

import "fyne.io/fyne/v2"

// Display is the monitor a window opens on and the part of it that windows may
// use, which is the monitor less the taskbar or the dock. All four are in the
// window's own units, the ones Resize takes. It is not called Screen because
// that is what an entry of the sidebar is called, and a guard reads that name.
type Display struct {
	Monitor   fyne.Size
	MonitorAt fyne.Position
	Work      fyne.Size
	WorkAt    fyne.Position
}

// Fit is the size to open a window of the given size at on this screen, and
// whether that is a different size (B-421). The window is then put in the
// middle of the monitor, which is what the room is worked out for.
//
// # Why this exists
//
// Two faults, both measured on 2026-11-17 on a monitor of 3840 by 2160 pixels
// with a taskbar of 72 along the bottom. A size kept on a large monitor and
// opened on a smaller one is larger than the screen, and the system makes the
// window smaller before the toolkit listens for its size: the toolkit laid the
// content out for the size it asked for and drew it into the smaller window,
// with the top and the right of every screen cut off, and closing the window
// wrote the size back. And a large size that the screen could hold was put
// where the system puts a new window, a little down and to the right of the
// last one: a window 2016 pixels tall opened 235 down, 91 of them below the
// bottom of the screen, and one closed maximised opened 463 down with 357
// below. The toolkit says neither where its window is nor how to move it,
// except to the middle.
//
// # The room
//
// The largest size that, in the middle of the monitor the way the toolkit
// centres a window, stays inside the work area: the monitor less twice the
// wider of the two strips the system keeps for itself on each axis, because
// the middle of the monitor is not the middle of the work area when the
// taskbar is on one side only. Each side of the size is cut on its own, so a
// window too tall keeps its width. A window closed maximised comes back a
// little shorter, by the height of the taskbar, and all of it on the screen.
//
// A screen the system describes with a nought in it, or one whose work area
// leaves no room in the middle, leaves the size as it was rather than
// shrinking the window to nothing.
func Fit(size fyne.Size, screen Display) (fyne.Size, bool) {
	if !Worth(screen.Monitor) || !Worth(screen.Work) {
		return size, false
	}
	room := centred(screen)
	if !Worth(room) || (size.Width <= room.Width && size.Height <= room.Height) {
		return size, false
	}
	return fyne.NewSize(fyne.Min(size.Width, room.Width), fyne.Min(size.Height, room.Height)), true
}

// centred is the largest size that stays inside the work area when it stands
// in the middle of the monitor.
func centred(s Display) fyne.Size {
	left := s.WorkAt.X - s.MonitorAt.X
	right := s.MonitorAt.X + s.Monitor.Width - (s.WorkAt.X + s.Work.Width)
	top := s.WorkAt.Y - s.MonitorAt.Y
	bottom := s.MonitorAt.Y + s.Monitor.Height - (s.WorkAt.Y + s.Work.Height)
	across, down := fyne.Max(left, right), fyne.Max(top, bottom)
	// The wider strip comes off both sides of the middle, so it is taken
	// away twice.
	return fyne.NewSize(s.Monitor.Width-across-across, s.Monitor.Height-down-down)
}
