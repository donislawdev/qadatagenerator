package paint

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"

	theme "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// TestACardOfRowsNeverGrowsPastTheRoomItWasGiven holds the one thing a layout
// in this toolkit must never do.
//
// # 🔴 What went wrong, and why nothing said so
//
// Fyne does not clip children. A layout that resizes its child taller than the
// height it was itself given produces a child that draws OUTSIDE its parent,
// over whatever is underneath, and nothing fails and nothing warns.
//
// fitHeight did exactly that. It divides the height it is offered between its
// rows and then raises a row that came out too short to a floor, so that a row
// stays readable. Raising the row raises the total, and when the window is
// short the total lands above the height that was offered. The card then draws
// over the band of chrome below it and the last row is sliced in half.
//
// # 🔴 Why it took a person to find it
//
// Every render this project makes comes from a probe whose canvas scale is one,
// and the owner's machine draws at one and a half. So a render asked for at
// 1670 by 1210 shows a window with 1670 by 1210 LOGICAL pixels, while the owner
// at the same number of real pixels has 1099 by 800 of them - two and a third
// times the area, in every picture any decision was made from. Everything fit
// in the pictures. The owner put the program beside the drawing and asked why
// things were overlapping.
//
// The screen offers up to fifty values. At the floor those need over two
// thousand pixels, so this was never about one row: it was a card that could
// ask for five times the window and be handed it without a word.
func TestACardOfRowsNeverGrowsPastTheRoomItWasGiven(t *testing.T) {
	// The heights a person really has, rather than the ones a probe is asked
	// for. 800 is the owner's window at the scale their machine draws at, and
	// 420 is roughly what is left for the card once the chrome above and below
	// it has taken its share.
	for _, room := range []float32{800, 420, 300, theme.RowHeightMin} {
		for _, rows := range []int{1, 10, 25, 50} {
			card := canvas.NewRectangle(nil)
			holder := FitHeight(rows, theme.NameStepTight, card)
			holder.Resize(fyne.NewSize(600, room))

			if card.Size().Height > room {
				t.Errorf("%d rows in %v of room were laid out %v tall, so the card draws %v "+
					"pixels outside the space it was given, onto whatever is under it",
					rows, room, card.Size().Height, card.Size().Height-room)
			}
		}
	}
}

// TestACardOfRowsStillGrowsWithTheWindow holds the other half, so that the fix
// above cannot be had by never growing at all.
//
// A card that always took its minimum would pass the test above and lose the
// thing FitHeight exists for: spending spare height on the rows a person is
// reading rather than leaving it blank underneath them.
func TestACardOfRowsStillGrowsWithTheWindow(t *testing.T) {
	const rows = 10
	card := canvas.NewRectangle(nil)
	holder := FitHeight(rows, theme.NameStepTight, card)

	holder.Resize(fyne.NewSize(600, theme.RowHeightMin*rows))
	tight := card.Size().Height
	holder.Resize(fyne.NewSize(600, theme.RowHeightMax*rows*2))
	roomy := card.Size().Height

	if roomy <= tight {
		t.Errorf("the card is %v tall in a short window and %v in a tall one, so it stopped "+
			"spending spare height on its rows", tight, roomy)
	}
	// And it stops growing where the ceiling says, or a list of three values in
	// a tall window becomes three bands the height of a hand.
	if want := theme.RowHeightMax*rows + theme.StepOf(theme.NameStepTight)*2; roomy > want {
		t.Errorf("the card grew to %v where its rows cap out at %v", roomy, want)
	}
}
