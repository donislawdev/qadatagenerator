package parts

import (
	"fyne.io/fyne/v2"

	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// confirmation is a control saying, for a while, that it did something a
// person cannot see, such as filling the clipboard, and then saying its own
// word again. A button and a row of values wear one.
//
// # Why on the control and why for a while
//
// Until 2026-10-21 a sentence beside the button said so and stayed until the
// next copy, so a screen went on claiming the clipboard held what it held an
// hour ago. The owner asked for the button itself to say it for a moment.
//
// # Why the count back is an animation of the toolkit
//
// An animation ticks on the thread that draws, before a frame, so the control
// changes back where every other change to it happens and nothing here starts
// concurrent work: the register of files allowed to stays as it is. It runs
// forth and back, palette.ConfirmationLeg each way, and the word goes back on the last
// tick, which is the one at nought after the one at one -
// internal/animation/runner.go of v2.8.2. That shape is chosen for one more
// reason: the test driver ticks an animation that runs back once at nought and
// stops, test/driver.go:117, so a picture of a pressed control shows the word a
// person sees rather than one that went away before anybody could look.
type confirmation struct {
	on     bool
	settle *fyne.Animation
}

// start puts the word up and counts back from it, drawing through redraw at
// both ends. A second start before the end begins the count again, so a person
// pressing twice is told twice rather than half as long.
func (c *confirmation) start(redraw func()) {
	c.on = true
	redraw()
	if c.settle != nil {
		c.settle.Stop()
	}
	reachedEnd := false
	c.settle = fyne.NewAnimation(palette.ConfirmationLeg, func(done float32) {
		if done == 1 {
			reachedEnd = true
		}
		if reachedEnd && done == 0 {
			c.on = false
			redraw()
		}
	})
	c.settle.AutoReverse = true
	c.settle.Curve = fyne.AnimationLinear
	c.settle.Start()
}

// end takes the word down before its time, for a screen on which something
// else has since filled the clipboard: two controls both saying they did would
// say the clipboard holds two things.
func (c *confirmation) end(redraw func()) {
	if c.settle != nil {
		c.settle.Stop()
		c.settle = nil
	}
	if c.on {
		c.on = false
		redraw()
	}
}
