package parts

import (
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// ShowIcon puts one of this window's own icons on a button, and keeps it
// readable while the keyboard is on it.
//
// # Why SetIcon on its own would have been a defect
//
// 🔴 Measured 2026-09-09, before a single icon was placed. An icon of this
// window carries its colour INSIDE the file: paint substitutes currentColor
// for a real value and hands the toolkit a finished resource, which is what
// lets the stroke width follow the size and what avoids the toolkit's own
// colouring route, which paints fill and never stroke.
//
// A finished resource cannot change colour, and the toolkit FLOODS a focused
// button with the focus colour. So an icon set once and left would be drawn in
// text.primary on mark.current, which measures 2.15 where a mark needs 3.0 -
// invisible at exactly the moment a person using the keyboard is looking at
// it, and in a state nobody photographs by reflex.
//
// This is the same fault parts.Flooded was written for, arriving through the
// picture instead of through the word. The repair is the same shape: the
// control already knows where the keyboard is, so the icon is redrawn from
// that, in the ink the label beside it is using.
//
// # Why the name is kept rather than the resource
//
// A resource is a colour already decided. Keeping the NAME means the colour is
// decided at the moment of drawing, from the state the control is in, which is
// the only way one control can carry two of them.
func (b *Button) ShowIcon(name string) {
	b.icon = name
	b.paintIcon()
}

// paintIcon draws the icon in the ink the label is drawn in.
//
// Both branches are needed and neither is the other's default: a button at
// rest is read on the ground it stands on, and a focused one is read on the
// flood the toolkit paints over that ground.
func (b *Button) paintIcon() {
	if b.icon == "" {
		return
	}
	ink := palette.TextPrimary
	if b.keyboardIsHere {
		ink = palette.TextOnMark
	}
	b.SetIcon(paint.IconResource(b.icon, ink, palette.IconSmall))
}
