package paint

import (
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"

	theme "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// The faces of the controls, drawn here and given behaviour next door.
//
// # Why a face is separate from the control at all
//
// A widget of the toolkit draws itself, and the drawing is the half this
// project needs to replace while the behaviour is the half it must not. Keeping
// the face as a function returning shapes means the same drawing serves a real
// control in parts and a still picture in a mock, and the two cannot drift,
// because there is one of them.
//
// This is also what makes the rule of DEC-228 checkable rather than aspirational:
// nothing here takes a callback, so nothing here can quietly become a control.

// ButtonLook is everything the face of a button depends on.
//
// # Why a struct and not five arguments
//
// 🔴 Because the face has to carry the STATES, and that is what makes it usable
// by the real control rather than only by a still picture.
//
// Until 2026-09-09 this took a label, an icon, a flag and a width, which is
// everything a drawing needs and not everything a control needs. So the mock
// drew buttons out of these shapes and the window drew them out of the
// toolkit's, and the two were free to look nothing alike - which is exactly
// what the owner saw when he put the two pictures side by side. The comment at
// the top of this file promised they could not drift because there was one
// drawing. There was one drawing and one of them was not using it.
//
// A caller that omits every state gets the resting look, so the mock says what
// it means without spelling out three falses.
type ButtonLook struct {
	// Label is the words on the button, and Icon the name of one of this
	// window's icons, empty for a button that shows none.
	Label string
	Icon  string

	// Filled marks the one action a screen is for.
	Filled bool

	// Width is a floor rather than a size. Zero means whatever the label needs.
	Width float32

	// Leading puts the label against the leading edge instead of centring it,
	// which is what a column of places to go needs and what a row of actions
	// does not.
	Leading bool

	// Plain drops the edge. A place to go in a column is not a slab to press,
	// and an edge around each of five names reads as a box around each rather
	// than as a list. The owner settled the same thing for the heading of the
	// generator list on 2026-09-05.
	Plain bool

	// Current marks a plain button naming the place the window is on, which
	// stands on the tinted ground of bg.current in every state rather than on
	// nothing.
	//
	// 🔴 A flag on the face since 2026-09-24, and before that a rectangle the
	// sidebar item drew UNDER the button. The two could not agree: the face
	// knew the pointer and not the place, the item knew the place and not the
	// pointer, so a pointer on the open screen covered its tint with the grey of
	// bg.hover. The live window showed it right after a click, which is when
	// the pointer is always there, and the state list had said in writing that
	// nothing photographed it.
	Current bool

	// Focused and Disabled are the states a person reaches with the keyboard,
	// Hovered and Pressed the two they reach with a pointer.
	//
	// 🔴 The last two arrived on 2026-09-09 and the sentence that used to stand
	// here called the first two "the two states a person can put a button
	// into". That was written while this face was drawing pictures, where the
	// only states are the ones somebody spells out. It became false the moment
	// a real control started wearing this face and nothing said so: the toolkit
	// had drawn a hover of its own since always - widget/button.go:382 blends
	// ColorNameHover - and taking its drawing away took that with it, silently,
	// because a state nobody draws leaves no mark to miss.
	//
	// Measured on the real binary before this: a window with the pointer over a
	// sidebar item was identical to one with the pointer in empty space, to the
	// last of 1,965,600 pixels.
	Focused  bool
	Disabled bool
	Hovered  bool
	Pressed  bool
}

// pointerGround is the ground a pointer puts an ordinary control on, and the
// edge that ground needs. The ground is empty when no pointer is on it, which
// is the resting look and is not the same as a ground of the window's colour.
//
// # Why the edge travels with the ground rather than staying put
//
// Because it cannot stay put, and that is measured rather than chosen.
// border.control clears WCAG 1.4.11 at 3.03 on the ground a pointer gives a
// control and falls to 2.68 one step later, which is where pressing puts it.
// The register carries the arithmetic beside bg.pressed.
//
// So this returns the pair rather than the ground, and every control asking for
// one asks for both. A caller free to take the ground and keep its own edge is
// a caller free to reintroduce that 2.68, in a place where nothing would fail.
func pointerGround(hovered, pressed bool) (ground, edge string) {
	switch {
	case pressed:
		return theme.BgPressed, theme.BorderPressed
	case hovered:
		return theme.BgHover, theme.BorderControl
	}
	return "", theme.BorderControl
}

// currentGround is the ground a plain button naming the open place stands on,
// in each state a pointer can put it in. The accent ladder carries the three
// as steps 3, 4 and 5, which is the job the method gives those steps.
func currentGround(hovered, pressed bool) string {
	switch {
	case pressed:
		return theme.BgCurrentPressed
	case hovered:
		return theme.BgCurrentHover
	}
	return theme.BgCurrent
}

// filledGround is the fill the one action of a screen takes, and the ink its
// label is read in on that fill.
//
// # Why it stopped being exported on 2026-09-24
//
// It had a second reader: a filled button that kept three shapes of its own and
// recoloured them, and so could not ask ButtonFace for this pair. That button
// is a flag on parts.Button now and wears this face, so the pair has one reader
// and the rule one copy, which is how a fill stays with the only ink it can
// carry.
//
// # 🔴 Why the ink does NOT move with the fill, corrected on 2026-09-09
//
// It used to. The button lightened under the pointer, and lightening past a
// point leaves white unreadable, so the label went dark to compensate. The
// owner looked at the sheet of states and said it plainly: nobody changes the
// colour of the text on a button, and it looks wrong.
//
// He is right, and the arithmetic that had been written here was answering the
// wrong question. It said there is no violet carrying both inks - true, and
// beside the point, because the fill does not have to go up the ladder. NN/g
// and every design system asked describe hover on a filled button as a slight
// DARKENING. Going down keeps one ink for every state. The steps are the ones
// the register names beside fill.hover and fill.pressed, re-measured on the
// blue of 2026-09-24:
//
//	rest     accent.9   white at 4.75
//	hover    accent.7   white at 7.49
//	pressed  accent.6   white at 9.38
//
// Until then this table named accent.8 and accent.7, one step off the
// register, which is why it now points there rather than repeating it. The
// old direction gave 4.23 on hover, under the 4.5 a label answers to, which is
// what forced the ink to move in the first place.
func filledGround(hovered, pressed bool) (fill, ink string) {
	switch {
	case pressed:
		return theme.FillPressed, theme.TextOnFill
	case hovered:
		return theme.FillHover, theme.TextOnFill
	}
	return theme.FillPrimary, theme.TextOnFill
}

// ButtonFace is the look of a button, in whatever state it is in.
//
// # Why filled and quiet rather than a list of importances
//
// There is exactly one action a screen is for, and everything else is either a
// way out or a way sideways. That was settled on the pixels: four screens of
// five drew every button identically, and a screen where nothing stands out is
// a screen where a person has to read every control to find the one they want.
//
// The filled one carries the accent. The quiet one carries an edge, because
// DEC-211 settled that a shape a person can press has to say so, and measured
// what happens when it does not: 240 buttons at a contrast of 1.07.
//
// # What each state looks like, and why
//
//   - DISABLED loses the edge and keeps a quiet label. An edge says a control
//     can be used, so leaving it on the one thing nobody can press would make
//     it the most findable thing on the screen. The filled one keeps a grey
//     fill instead of its accent since 2026-10-21 (DEC-261), so that the
//     action of a screen is still a shape before it can be pressed.
//   - FOCUSED turns the edge mark.focus, the shade every control in this
//     window marks focus with, and a filled button, which has no edge at rest,
//     gains one. Until 2026-09-09 this line described a ring inside the shape
//     in the ink of the label, which the owner saw on the sheet of states and
//     called awful, and the code below changed while this sentence did not
//     until 2026-09-24.
//
// The mark is drawn by this face rather than by the toolkit, which is the whole
// point of the control wearing it. The toolkit's own answer floods the control
// with the focus colour, and that flood was measured at 1.23 for the mark and
// 1.56 for the label under it.
func ButtonFace(look ButtonLook) fyne.CanvasObject {
	ink := theme.TextPrimary
	var ground fyne.CanvasObject
	under, edge := pointerGround(look.Hovered, look.Pressed)
	// 🔴 Focus is the EDGE of the control rather than a second shape inside it,
	// changed on 2026-09-09 after the owner looked at the ring in a filled
	// button and called it awful. The reckoning behind the shade, including the
	// two candidates it killed, is beside mark.focus in the register.
	if look.Focused && !look.Disabled {
		edge = theme.MarkFocus
	}
	switch {
	case look.Disabled:
		// First, and it stays first. A control nobody can press does not answer
		// a pointer, so a flag left set on it by a pointer that arrived before
		// it was disabled must not reach the drawing.
		//
		// 🔴 The FILLED one keeps its shape, in grey, rather than losing it.
		// Until 2026-10-21 it lost it, and the one action of a screen with
		// nothing to act on yet read as a bold word: a person looking for what
		// to press next could not find the button at all (DEC-261). A fill is
		// not an edge, so the rule that an edge says a control can be used
		// still holds, and every other button still goes without one.
		ink = theme.TextSecondary
		ground = Clear()
		if look.Filled {
			ground = Fill(theme.BgControl, theme.RadiusControl)
		}
	case look.Filled:
		var fill string
		fill, ink = filledGround(look.Hovered, look.Pressed)
		ground = Fill(fill, theme.RadiusControl)
		if look.Focused {
			// The one control with no edge at rest, so focus ADDS one rather
			// than recolouring it. Same shade as everywhere else: 3.84 against
			// the fill and 6.05 against the fill under the pointer.
			ground = inside(Bordered(fill, theme.MarkFocus, theme.RadiusControl))
		}
	case look.Plain:
		ground = plainGround(look, under)
	case under == "":
		// An edge around nothing rather than a fill of its own. A quiet
		// button stands on three grounds here, and any fill would be a
		// patch of the wrong colour on two of them. Seen on the render.
		ground = inside(Outline(edge, theme.RadiusControl))
	default:
		ground = inside(Bordered(under, edge, theme.RadiusControl))
	}

	return AtLeast(look.Width, theme.ControlHeight, container.NewStack(ground, buttonWords(look, ink)))
}

// plainGround is the ground of a plain button, the one kind with no edge at
// rest.
//
// Apart from ButtonFace because that function passed the limit on the length
// of a function, and this branch is the longest of its switch. under is the
// ground the pointer gives any button, which a button naming the place the
// window is on replaces with its own.
func plainGround(look ButtonLook, under string) fyne.CanvasObject {
	// No edge at rest, so a pointer has only the ground to show with.
	var ground fyne.CanvasObject = Clear()
	if look.Current {
		under = currentGround(look.Hovered, look.Pressed)
	}
	if under != "" {
		ground = Fill(under, theme.RadiusControl)
	}
	if look.Focused && !look.Disabled {
		// 🔴 Except for focus, which has to be visible on every control
		// that can hold the keyboard. Caught by the sheet of every state of
		// every control on 2026-09-09 and by nothing else: moving focus
		// from a ring to an edge left this branch drawing no mark at all,
		// since it is the one branch that draws no edge to recolour. The
		// tree still said the button was focused, so no test could see it.
		//
		// 🔴 An OUTLINE when this button has no ground of its own, and that
		// distinction cost a second render. Asking for a bordered shape
		// with no fill token named fills it with the fallback, so a plain
		// button in focus came out as a solid white block - which the sheet
		// showed at once and which the tree, again, called a focused button.
		if under == "" {
			ground = inside(Outline(theme.MarkFocus, theme.RadiusControl))
		} else {
			ground = inside(Bordered(under, theme.MarkFocus, theme.RadiusControl))
		}
	}
	return ground
}

// buttonWords is the icon and the label of a button, with the room either side
// of them, in the ink the ground of the button asked for.
//
// Apart from ButtonFace for the reason plainGround is: the ground and the words
// are two halves of one face, and the face passed the limit with both inside.
func buttonWords(look ButtonLook, ink string) fyne.CanvasObject {
	row := []fyne.CanvasObject{}
	if look.Icon != "" {
		// The icon takes the accent on every button that has no fill of its
		// own, which is where most of the colour of a screen comes from since
		// 2026-09-24. Not on the filled one, whose icon is read on the accent,
		// and not on a disabled one, which must not look like it can be used.
		iconInk := theme.InkIcon
		if look.Filled || look.Disabled {
			iconInk = ink
		}
		row = append(row, Middle(Icon(look.Icon, iconInk, theme.IconSmall)))
	}
	// The one action of a screen carries its label at the weight of a heading,
	// which is how the window drew it for as long as it was a control of its
	// own. Kept on 2026-09-24 when it came onto this face, so that removing a
	// copy of the drawing did not also change what a person had been looking at.
	label := Text(look.Label, theme.TypeBody, ink)
	if look.Filled {
		label = Strong(look.Label, theme.TypeBody, ink)
	}
	row = append(row, Middle(label))

	// The gap between the icon and the words comes from the scale here, which
	// is the difference between this face and the toolkit's button: the toolkit
	// takes that gap from its own padding, and this window answers nought to
	// that question on purpose, so an icon on a stock button touches its label.
	words := Row(theme.NameStepTight, row...)

	// 🔴 Room either side of the words, and it is not decoration. Centring alone
	// asks for exactly the width of the label, so the edge is drawn ON the first
	// and last letter and a button reads as a word somebody boxed too tightly.
	// Seen on the render the day the button began drawing itself: every button
	// with no width of its own came out clipped.
	var inner fyne.CanvasObject = Pad(theme.NameStepHairline, theme.NameStepHairline,
		theme.NameStepGroup, theme.NameStepGroup, container.NewCenter(words))
	if look.Leading {
		inner = Pad(theme.NameStepHairline, theme.NameStepHairline,
			theme.NameStepRelated, theme.NameStepRelated,
			container.New(layout.NewBorderLayout(nil, nil, words, nil), words))
	}
	return inner
}

// inside puts a shape far enough in from its own bounds that a stroke centred
// on its edge lands wholly within the control.
//
// # 🔴 Why a face needs this, measured on 2026-09-09
//
// The toolkit draws a stroke CENTRED on the path, so a rectangle at the full
// size of a control puts half its edge outside that control. At the thickness
// this window uses that is one pixel, drawn over whatever the control stands
// on and arriving as a blend rather than as the colour asked for.
//
// It went unnoticed until a chooser that draws itself stood in one row beside a
// text field that does not, and the two were compared in the stored tree. The
// toolkit's own field puts its border at 1,1 and one thickness in from each
// edge. Ours sat at 0,0. The same row, two controls, two different edges - and
// DEC-215 settled that this is a fault even when each passes on its own.
//
// parts.Outlined, the edge once laid over a stock button, did this correctly
// for as long as the window used it, which is why the distance already has a
// name in the theme. It was deleted with DEC-253, when nothing called it any
// more.
//
// Nothing of ours lays anything out here: the toolkit's padded layout takes the
// margin, which is the same reason Outlined nested two of its containers.
func inside(shape fyne.CanvasObject) fyne.CanvasObject {
	edge := theme.OutlineInset
	return container.New(layout.NewCustomPaddedLayout(edge, edge, edge, edge), shape)
}

// FieldFace is the look of something typed into: a ground a step off the
// window, an edge, and the text inside it.
//
// The face carries no caret and no selection. Those belong to the control that
// wears it, and the toolkit draws both.
func FieldFace(value, rung string, width float32) fyne.CanvasObject {
	ground := Bordered(theme.BgControl, theme.BorderControl, theme.RadiusControl)
	text := Middle(Mono(value, rung, theme.TextPrimary))
	return AtLeast(width, theme.ControlHeight,
		On(ground, Pad(theme.NameStepHairline, theme.NameStepHairline,
			theme.NameStepRelated, theme.NameStepRelated, text)))
}

// FieldGround is the ground and the edge of a text field, in whichever of its
// four states the keyboard and the pointer put it in.
//
// # Why a field needs one at all, when the toolkit draws its own
//
// Because the toolkit's edge is tied to the arithmetic that places the text:
// the renderer of widget.Entry puts its words at twice SizeNameInputBorder from
// the top and draws its edge at that same thickness. Drawing the edge here
// unties the two, and it also lets a field show focus the way every other
// control in this window shows it.
//
// # Why the pointer is answered from the rule the quiet button uses
//
// The toolkit draws nothing for a pointer on a field (B-282), so what it should
// look like is a decision, and the decision already made for a quiet button is
// the one thing that keeps a field and a button in one row looking like one
// program: the same ground under the pointer, the same edge, and focus as the
// edge in mark.focus whichever ground is under it. There is no pressed state,
// since pressing a field puts the caret in it and the toolkit draws that.
//
// # Why a field out of reach looks like a chooser out of reach
//
// Because the two stand in one row of every form in this window, and DEC-215
// made them one family. So it is the same ground, drawn by the same function,
// whoever is on the field: the pointer and the keyboard still arrive at a
// disabled field - a press focuses it so its text can be copied, at
// widget/entry.go:397 and internal/app/focus_manager.go:48 of v2.8.1 - and
// neither may make it look like something that can be changed (DEC-253).
//
// # Why a refused value is the edge and focus then moves inside it
//
// A field holding what its screen would refuse wears its edge in state.danger,
// which is the colour this window gives a refusal, and it does so whether or
// not the keyboard is in it - somebody typing the wrong number is exactly the
// person who has to see it (B-363). Focus is shown by the colour of that same
// edge, so on a refused field it is drawn as a second edge just inside the
// first rather than being lost under the red.
func FieldGround(look FieldLook) fyne.CanvasObject {
	if look.Disabled {
		return disabledGround()
	}
	under, edge := pointerGround(look.Hovered, false)
	if under == "" {
		under = theme.BgControl
	}
	if look.Invalid {
		refused := inside(Bordered(under, theme.StateDanger, theme.RadiusControl))
		if !look.Focused {
			return refused
		}
		return container.NewStack(refused, innerRing(theme.MarkFocus))
	}
	if look.Focused {
		edge = theme.MarkFocus
	}
	return inside(Bordered(under, edge, theme.RadiusControl))
}

// innerRing is an edge with nothing inside it, drawn one thickness in from the
// edge of the control, so that it touches the first edge and covers none of it.
func innerRing(token string) fyne.CanvasObject {
	ring := canvas.NewRectangle(color.Transparent)
	ring.CornerRadius = theme.RadiusControl - theme.BorderThickness
	ring.StrokeColor = theme.Ink(token)
	ring.StrokeWidth = theme.BorderThickness
	in := theme.InnerRingInset
	return container.New(layout.NewCustomPaddedLayout(in, in, in, in), ring)
}

// FieldLook is everything the ground of a text field depends on.
//
// A struct for the reason ChooserLook is one, and with its names: the states
// mean here exactly what they mean on the chooser beside it. Invalid is the
// field's own: it holds what its screen would refuse.
type FieldLook struct {
	Focused  bool
	Hovered  bool
	Disabled bool
	Invalid  bool
}

// disabledGround is what a control that holds a value wears when nobody can
// change it: the ground and no edge.
//
// An edge says a control can be used, so it goes. The ground stays, because a
// value nobody can touch still has to show WHERE it is, or it floats in the
// window. One function for the chooser and the field, so that the two cannot
// drift apart the way they did before DEC-215.
//
// Not inset like the edged grounds, and that is what keeps the size the same:
// an edge is centred on its path, so the edged ground one pixel in reaches the
// border of the control, and a ground with no edge has to reach it on its own.
func disabledGround() fyne.CanvasObject {
	return Fill(theme.BgControl, theme.RadiusControl)
}

// ChooserLook is everything the face of a chooser depends on.
//
// A struct for the reason ButtonLook is one, and it arrived the same way: the
// face took a value and a width, which is everything a picture needs and not
// enough for a control, so the mock drew choosers out of these shapes and the
// window drew them out of the toolkit's.
type ChooserLook struct {
	// Value is what is written on it.
	Value string

	// Width is a floor rather than a size. Zero means whatever the value needs,
	// which is what a still picture wants and what a real chooser must not have
	// - see ChooserWidth below.
	Width float32

	// Unchosen marks a value that is the placeholder rather than a choice, so
	// that the words standing in for an answer do not read as one.
	Unchosen bool

	// Focused and Disabled are the states a person reaches with the keyboard,
	// Hovered and Pressed the two they reach with a pointer. They mean here
	// exactly what they mean on a button, and they are drawn from the same two
	// tokens, because a chooser and a button standing in one row have to look
	// like one family - which DEC-215 settled for the resting look and settles
	// again for every other.
	//
	// 🔴 There is no Open, and its absence is a finding rather than an
	// omission. The toolkit never draws one either: selectRenderer.updateIcon
	// at widget/select.go:437 of v2.8.1 asks for IconNameArrowDropDown in every
	// state, because what opens is a pop up menu laid over the window rather
	// than anything about the chooser. Nor could this face be told: the pop up
	// is kept in a private field and dismissed through a callback assigned
	// inside a private method, so a flag set on tapping would have no way to
	// learn that the list had closed and would leave the mark stuck open.
	Focused  bool
	Disabled bool
	Hovered  bool
	Pressed  bool
}

// ChooserFace is the look of a control that opens a list: the field face with a
// mark at its trailing edge saying it opens.
//
// # Why the mark is drawn rather than left to the toolkit
//
// The toolkit draws its own arrow inside its chooser, in its own icon set and
// its own colour, and a screen carrying one stock arrow among nineteen icons of
// another family is the kind of detail nobody can name and everybody sees.
//
// # What each state looks like, and why it matches the button
//
//   - DISABLED keeps the ground and loses the edge, which is exactly what the
//     rectangle laid beside the toolkit's chooser used to do, and since DEC-253
//     what a text field out of reach wears too: both take disabledGround.
//   - FOCUSED gains a ring inside the shape, in the ink of the value, drawn by
//     the same helper the button uses. The toolkit's answer is to flood the
//     whole control with the focus colour, measured at 1.56 for the value
//     written on it, which is the entire reason parts.Flooded existed until
//     DEC-253 deleted it.
func ChooserFace(look ChooserLook) fyne.CanvasObject {
	ink := theme.TextPrimary
	under, edge := pointerGround(look.Hovered, look.Pressed)
	if look.Focused && !look.Disabled {
		edge = theme.MarkFocus
	}
	if under == "" {
		// A chooser has a ground of its own at rest, where a button has none,
		// so this is the one place the resting look is a colour rather than the
		// absence of one.
		under = theme.BgControl
	}
	ground := inside(Bordered(under, edge, theme.RadiusControl))
	switch {
	case look.Disabled:
		ink = theme.TextSecondary
		ground = disabledGround()
	case look.Unchosen:
		ink = theme.TextSecondary
	}

	// 🔴 The gap before the arrow is drawn here rather than left to the border
	// layout, and it is not decoration. That layout puts the theme's padding
	// between a side and the centre, and this window answers nought to that
	// question on purpose, so the value and the arrow TOUCH on any chooser
	// whose width is its content - which every chooser here now is.
	arrow := Pad(theme.NameStepHairline, theme.NameStepHairline,
		theme.NameStepTight, theme.NameStepHairline,
		Middle(Icon(IconChevron, theme.TextSecondary, theme.IconSmall)))
	inner := container.NewBorder(nil, nil, nil, arrow,
		Middle(Text(look.Value, theme.TypeBody, ink)))

	dressed := []fyne.CanvasObject{ground,
		Pad(theme.NameStepHairline, theme.NameStepHairline,
			theme.NameStepRelated, theme.NameStepTight, inner)}
	return AtLeast(look.Width, theme.ControlHeight, container.NewStack(dressed...))
}

// ChooserWidth is how wide a chooser has to be to show any of these values
// without changing size when one of them is picked.
//
// # Why a chooser may not be as wide as what it currently says
//
// Because the width would then change every time somebody chose something, and
// a row of four settings would shuffle sideways under the hand that was using
// it. The toolkit avoids that too, and by an accident: selectRenderer.MinSize
// throws away the width of the value and uses the width of the PLACEHOLDER, so
// every chooser in this window was the width of the words "(Select one)" no
// matter what it held or what it could hold. Stable, and about nothing.
//
// Measured from the widest thing the chooser can be asked to show, so the
// number means something and still cannot move.
//
// # 🔴 Why it ASKS the face rather than adding up what the face contains
//
// The first version added the padding either side of the value, the gap before
// the arrow and the arrow itself, which is what the face puts around a value
// and reads as obviously right. It was four pixels short, and the guard next
// door caught it on its first run: a floor four pixels under what the drawing
// asks for is not a floor at all, so the width went on moving and the whole
// point of this function was lost while it looked correct.
//
// The mistake is not worth finding again. Arithmetic written beside a drawing
// is a second copy of that drawing, and this one had been wrong from the moment
// it was typed, in a place where nothing but a picture would have shown it. So
// the number comes from the drawing itself, and the two cannot disagree.
//
// It also removes a difference that has nothing to do with us: the border
// layout inside the face takes its own gap from the theme, which this window
// answers with nought and a test harness answers with four. Adding up the parts
// meant getting one answer here and another one on screen.
//
// # Why one face rather than one per value
//
// Because the width of the face rises with the width of the value and nothing
// else, so the widest value makes the widest face. Measuring the strings is
// cheap and building the drawings is not, and a chooser will one day be offered
// hundreds of options.
func ChooserWidth(values ...string) float32 {
	widest, wideEnough := float32(0), ""
	for _, value := range values {
		measured := fyne.MeasureText(value, theme.TypeSize(theme.TypeBody), fyne.TextStyle{}).Width
		if measured > widest {
			widest, wideEnough = measured, value
		}
	}
	return ChooserFace(ChooserLook{Value: wideEnough}).MinSize().Width
}

// FieldWidth is how wide a field has to be to show that text without cutting it.
//
// # Why a field is given a width at all
//
// A field with none takes whatever the layout has left over, and that is how
// the seed on the quick screen came to run to the edge of the window: it stood
// as the CENTRE of a border layout, so it was handed every pixel the settings
// beside it did not want. The owner called it giga long, and he was describing
// arithmetic rather than taste - the field was as wide as the window minus four
// controls, whatever that happened to be.
//
// # Why measured rather than written down
//
// Same reason as ChooserWidth next door. A number written at the one place it
// is used is a number nobody can hold to anything, and it goes wrong the day
// the type ladder moves. This asks the face how wide it would be, so the answer
// follows the font and the padding of the day it is asked.
//
// The caller passes the WIDEST thing the field has to hold rather than the
// thing it holds right now: a seed of thirty two characters, not the empty
// string it starts as.
func FieldWidth(sample string) float32 {
	// 🔴 Rounded UP to a whole number, because the face measures a string and a
	// string measures fractionally. A width of 61.6 makes the layout round its
	// neighbour's position, and the gap beside it comes out 13 where the scale
	// says 12 - which tools/odstepy.py reports as a distance nobody chose,
	// correctly, and which nothing else would ever have noticed.
	return float32(math.Ceil(float64(FieldFace(sample, theme.TypeBody, 0).MinSize().Width)))
}

// Samples a field is measured from when more than one screen takes the same
// kind of value, so that the same value is the same width wherever it is typed.
//
// # Why these, added on 2026-10-21 (DEC-261)
//
// The owner, looking at the window: the fields are of different lengths, and
// the length of a field should fit the value it takes. A seed ran from the last
// setting to the button on the Quick screen and to the edge of the column on
// the damage screen, so the shortest value of a row had the longest field. A
// seed is a word or a date somebody types and has to type again, and a list of
// columns to spare is a few names. A path stays as wide as its column, because
// a path is the one value here that is genuinely long.
const (
	// SeedSample is a seed of the kind somebody types: a word with a date.
	// Fourteen characters, because the widest row of the Quick screen, with a
	// gender and a date of birth, put its button twelve units into the margin
	// of the window at eighteen, measured on the render of 2026-10-21.
	SeedSample = "sprint-2026-10"
	// NamesSample is a few column names, which is what is spared in practice.
	NamesSample = "first_name, last_name, email, phone"
)
