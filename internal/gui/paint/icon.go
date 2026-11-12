package paint

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"

	"github.com/donislawdev/qadatagenerator/internal/gui/assets"
	theme "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// The names of the icons this window draws.
//
// # Why constants rather than the file names typed at each call site
//
// assets.Icon panics on a name it does not hold, which is right, and these
// constants are what makes that panic unreachable from a typo. The test beside
// this file closes the register in both directions: a constant naming a file
// that is not embedded, and a file nothing draws, are both findings. The second
// half matters as much as the first, since an icon nothing uses is a kilobyte
// in the binary and a promise in the directory listing.
const (
	IconAgain     = "refresh-cw"
	IconAlert     = "circle-alert"
	IconArrow     = "arrow-right"
	IconBolt      = "zap"
	IconBug       = "bug"
	IconBreak     = "file-x"
	IconCheck     = "check"
	IconChecklist = "clipboard-check"
	IconChevron   = "chevron-down"
	// 🔴 IconChevronUp is the one file in this set that was not taken from the
	// icon set as published. It is chevron-down mirrored about the middle of
	// the grid: the path of the shipped file is "m6 9 6 6 6-6", and reflecting
	// it across y=12 gives "m6 15 6-6 6 6", which is what the file holds.
	//
	// Derived rather than fetched because a path typed from memory is a guess
	// that draws something plausible and slightly wrong, and nothing in this
	// project would catch it. The shape is confirmed by a second file already
	// in the set: chevrons-up-down carries the same upward stroke as
	// "m7 9 5-5 5 5", on a shorter span.
	//
	// The set is ISC, which permits a modified copy, and the notice beside the
	// files covers it. This sentence is here so that nobody later reads the
	// file as one of theirs and looks for it in their catalogue.
	IconChevronUp = "chevron-up"
	IconCode      = "code-xml"
	IconCopy      = "copy"
	IconDoc       = "file-code"
	IconFolder    = "folder-open"
	IconHeart     = "heart"
	IconInfo      = "info"
	IconPlay      = "play"
	IconRecipe    = "file-cog"
	IconSearch    = "search"
	IconShield    = "shield-check"
	IconSwap      = "chevrons-up-down"
	IconWarn      = "triangle-alert"
	IconClose     = "x"
)

// IconNames is every constant above, so a test can close the register from the
// side the constants are on.
var IconNames = []string{
	IconAgain, IconAlert, IconArrow, IconBolt, IconBug, IconBreak,
	IconCheck, IconChecklist, IconChevron, IconChevronUp, IconCode, IconCopy, IconDoc, IconFolder,
	IconHeart, IconInfo, IconPlay, IconRecipe, IconSearch, IconShield, IconSwap,
	IconWarn, IconClose,
}

// The two attributes the loader rewrites, as they are written in every file of
// the set.
//
// They are constants because the substitution is silent when it misses: an icon
// whose colour was not replaced draws black on a dark window, and one whose
// stroke was not replaced is the wrong weight beside its neighbours. Neither
// raises anything. The test beside this file reads every embedded file and
// fails if either string is absent, which turns a silent miss into a loud one.
const (
	sourceColour = "currentColor"
	sourceStroke = `stroke-width="2"`
)

// Icon is one embedded icon, drawn in the colour of a token at the given size.
//
// # Why the colour is substituted here rather than by the toolkit
//
// 🔴 Measured on the source of Fyne 2.8.1, and it is the trap that would have
// cost a whole round of rework. theme.NewColoredResource rewrites FILL and
// never touches STROKE: internal/svg/svg.go:297 walks Paths, Rects, Circles,
// Ellipses, Polygons and Groups by their Fill attribute alone. This icon set,
// like every other outline set in use today, is drawn entirely in stroke with
// fill="none". Sent through that route, the icons would take our colour into an
// attribute that paints nothing, and the decoder would turn the leftover
// currentColor into black: internal/svg/svg.go:54 reads the stream replacing it
// with #000000. On a dark window that is an invisible shape, and nothing
// anywhere would report it.
//
// Substituting both attributes ourselves is not a workaround but the better
// route: it is what lets the stroke width follow the size the icon is drawn at,
// which the toolkit's own path cannot do at all.
//
// # Why the resource name is derived rather than counted
//
// A counter would name the same icon differently depending on the order the
// screen happened to build its children in, and the stored trees that guard
// these screens record resource names. A screen would then differ from its
// stored copy for no reason a person could see. Deriving the name from what the
// icon IS keeps two identical icons identical, which is rule 7 of this project
// applied to a file name.
func Icon(name, token string, size float32) *canvas.Image {
	image := canvas.NewImageFromResource(IconResource(name, token, size))
	image.FillMode = canvas.ImageFillContain
	image.SetMinSize(fyne.NewSize(size, size))
	return image
}

// IconResource is the same icon as a resource rather than as something drawn,
// which is the form anything asking the toolkit for an icon has to hand it.
//
// # Why this is separate from Icon
//
// Two callers want the drawing and one wants the file. The controls of this
// window take an image and place it themselves, but a control this window does
// NOT draw asks the theme for its icon by name and takes a resource: a chooser
// reads theme.IconNameArrowDropDown at widget/select.go:441 and draws whatever
// comes back inside itself. That is the only route to the arrow of a chooser
// short of writing a renderer for one, and a renderer is the expensive answer
// to a question the toolkit already asks out loud.
//
// Everything that makes Icon careful applies here unchanged, which is the
// reason there is one body and not two: the substitution of both attributes,
// and the name derived from what the icon is rather than counted.
func IconResource(name, token string, size float32) fyne.Resource {
	ink := theme.Ink(token)
	hex := fmt.Sprintf("#%02x%02x%02x", ink.R, ink.G, ink.B)

	body := string(assets.Icon(name))
	body = strings.ReplaceAll(body, sourceColour, hex)
	body = strings.ReplaceAll(body, sourceStroke,
		fmt.Sprintf(`stroke-width="%.2f"`, StrokeFor(size)))

	return fyne.NewStaticResource(
		fmt.Sprintf("%s-%s-%.0f.svg", name, strings.TrimPrefix(hex, "#"), size), []byte(body))
}

// StrokeFor is how thick the line of an icon has to be written in the file so
// that it arrives at the intended width on screen.
//
// # Why this is arithmetic rather than a number per size
//
// The set is drawn with a stroke of 2 on a grid of 24, and the toolkit scales
// the whole drawing. So an icon at 13 arrives with a line half as thick as the
// same icon at 26, and a small one beside a large one looks like it came from
// another set. Scaling the stroke by the inverse of the size holds the width on
// screen steady instead, which is what makes a row of icons at three sizes read
// as one family.
//
// The bounds are why the result is not simply the inverse: below the floor a
// large icon becomes a hairline that disappears at the display scales a desktop
// applies, and above the ceiling a small icon becomes a blob with no shape left
// inside it. Both ends are reached by the sizes this window uses.
func StrokeFor(size float32) float32 {
	if size <= 0 {
		return theme.IconStrokeCeiling
	}
	stroke := theme.IconStrokeTarget * theme.IconGrid / size
	if stroke < theme.IconStrokeFloor {
		return theme.IconStrokeFloor
	}
	if stroke > theme.IconStrokeCeiling {
		return theme.IconStrokeCeiling
	}
	return stroke
}
