package parts

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// Choice is one thing a chooser offers, together with what the panel shows
// beside it and the heading it stands under.
type Choice struct {
	// Value is what is chosen, and what the chooser then shows.
	Value string
	// Detail stands beside the value in the panel, and may be empty. Typing in
	// the search field finds it, so a kind is found by what a person calls it
	// as well as by its code.
	Detail string
	// Note stands at the far end of the row, and may be empty.
	Note string
	// Group is the heading the value stands under. A list whose choices all
	// leave it empty is flat, and opens without a search field or a foot.
	Group string
}

// PanelWords are the sentences the panel says, in the language of whoever
// built it. The panel holds no words of its own, for the reason no control
// here does: text a person reads is a key of the catalogue, never a literal.
type PanelWords struct {
	// Prompt is what the empty search field says.
	Prompt string
	// Counted is the noun after the count of choices, such as kinds.
	Counted string
	// Matches replaces the count while a filter is on, with two counts in
	// order: how many match, then how many there are.
	Matches string
	// Enter says what Enter does.
	Enter string
	// Nothing stands where the rows were when nothing matches.
	Nothing string
}

// picker is the panel a chooser opens: a search field, the choices under their
// headings, and a foot saying how many there are.
//
// # Why the toolkit's pop up menu could not be this
//
// widget.Select opens a PopUpMenu, and the menu is a column of items and
// nothing else: it cannot take a field to filter by, cannot put a heading
// between two items, and shows one column where this shows two. The owner
// said what the chooser has to become in one sentence on 2026-09-08, "with
// grouping and searching", and gave the reason with it: there will be dozens
// of kinds, and in time hundreds, and nobody scrolls two hundred rows looking
// for one. The pop up is also private to the toolkit, so it could not have
// been extended either.
//
// # What stays with the rows and the field
//
// The rows are TapRow, which already answers the pointer, and the field is
// FieldEntry, which already looks like every other field. This widget holds
// what neither of them can: which row the keyboard is on, which rows the
// filter left, and the keys that move between them.
//
// # 🔴 Why the panel takes the keyboard when there is no field
//
// A key goes to whatever the top overlay has focused, and to nothing under it,
// so a panel with no focusable thing inside it is a panel the keyboard cannot
// reach at all. The field takes the keyboard when there is one, and forwards
// the keys that mean move, pick and close. Without a field the panel itself
// does, which is what PopUpMenu does too, read on the source of v2.8.1.
type picker struct {
	widget.BaseWidget

	choices []Choice
	words   PanelWords
	chosen  string
	onPick  func(string)
	onClose func()
	// compact is the panel of an editor's suggestions: rows as tall as their
	// words and a tight margin (newSuggestionPanel). natural is how wide such a
	// panel is with every detail in full, kept from before any was cut.
	compact bool
	natural float32
	// nameWidth is the floor under the column of names, the same on every row.
	nameWidth float32
	// onShape is told when the rows the filter left changed, so that
	// whoever placed the panel can size it to them again.
	//
	// 🔴 The panel follows its rows rather than keeping the height it opened
	// at, and the render decided it: with three rows left of seventeen, a
	// panel of fixed height was three rows over half a screen of nothing,
	// which reads as a panel that failed to fill rather than as room.
	onShape func()

	// search is nil for a flat list.
	search *searchEntry
	rows   []*pickRow
	// current is the row the keyboard is on, as an index into rows, or -1
	// when the filter left nothing.
	current int
	scroll  *Scroll
	list    *fyne.Container
	nothing fyne.CanvasObject
	foot    *canvas.Text
	body    fyne.CanvasObject
}

// pickRow is one choice, its heading if it opens a group, and whether the
// filter left it on screen. The choice is what the row shows, which in a
// compact panel may carry its detail cut to the panel's width (fitDetails), and
// full is the choice as it was given.
type pickRow struct {
	choice  Choice
	full    Choice
	tap     *TapRow
	heading fyne.CanvasObject
	shown   bool
}

var (
	_ fyne.Widget    = (*picker)(nil)
	_ fyne.Focusable = (*picker)(nil)
	_ fyne.Tappable  = (*picker)(nil)
)

// newPicker builds the panel for these choices, with this one chosen.
func newPicker(choices []Choice, chosen string, words PanelWords,
	pick func(string), dismiss func()) *picker {
	p := &picker{choices: choices, words: words, chosen: chosen, onPick: pick, onClose: dismiss}
	p.ExtendBaseWidget(p)
	p.buildRows()
	if p.grouped() {
		p.buildGrouped()
		return p
	}
	p.body = paint.On(paint.PanelGround(),
		paint.PadAll(palette.NameStepRelated, p.scroll))
	return p
}

// suggestionMargin is the margin of the panel of an editor's suggestions,
// tighter than a chooser's, because it opens among the lines of a text.
const suggestionMargin = palette.NameStepTight

// newSuggestionPanel is the panel of a chooser made for the list under an
// editor's caret (suggest.go): flat, nothing chosen, its rows as tall as their
// words and its margin the tight step. A code editor's list is read a line to a
// row, and it opens inside a field, where every row taller than a line is a row
// fewer in sight. Measured on the render of the recipe screen on 2026-11-24:
// with the rows of a chooser the list under the fourth line of the field held
// two rows.
//
// pointer is told when the pointer arrives on a row and leaves it, because the
// rows take the pointer from the field the panel stands in.
func newSuggestionPanel(choices []Choice, pick func(string), dismiss func(), pointer func(bool)) *picker {
	p := &picker{choices: choices, onPick: pick, onClose: dismiss, compact: true}
	p.ExtendBaseWidget(p)
	p.buildRows()
	for _, row := range p.rows {
		row.tap.OnPointer = pointer
	}
	p.body = paint.On(paint.PanelGround(), paint.PadAll(suggestionMargin, p.scroll))
	p.natural = p.body.MinSize().Width
	return p
}

// nameInset is how far from the panel's edge the names in its rows begin.
func (p *picker) nameInset() float32 {
	return palette.StepOf(suggestionMargin) + paint.OptionInset()
}

// fitDetails cuts the detail of every row of a compact panel to what a panel
// of that width leaves beside the name, or gives it back in full where it fits.
//
// # Why cut rather than let the panel run wider
//
// The panel stands inside the field, which is what keeps the keyboard in the
// field (suggest.go), so it can be no wider than the field. A field at the size
// the window opens in is about 335 wide at the recipe screen, measured in the
// live window on 2026-11-24, where the longest meaning of a key ran under the
// panel's edge and stopped in the middle of a word. The name stays whole,
// because it is what gets written.
func (p *picker) fitDetails(width float32) {
	if !p.compact {
		return
	}
	margin := palette.StepOf(suggestionMargin)
	room := width - margin - margin
	for _, row := range p.rows {
		bare := paint.OptionWidth(paint.OptionLook{Name: row.full.Value, NameWidth: p.nameWidth, Compact: true})
		cut := paint.CutToWidth(row.full.Detail, palette.TypeBody, room-bare)
		if cut != row.choice.Detail {
			row.choice.Detail = cut
			row.tap.Refresh()
		}
	}
}

// grouped says whether any choice stands under a heading, which is what
// decides between the two shapes of the panel.
func (p *picker) grouped() bool {
	for _, choice := range p.choices {
		if choice.Group != "" {
			return true
		}
	}
	return false
}

// buildRows makes one row per choice, a heading where a group begins, and the
// scrolling column they stand in.
func (p *picker) buildRows() {
	names := make([]string, 0, len(p.choices))
	for _, choice := range p.choices {
		names = append(names, choice.Value)
	}
	p.nameWidth = paint.OptionNameWidth(names...)

	items := make([]fyne.CanvasObject, 0, len(p.choices))
	group := ""
	for index, choice := range p.choices {
		row := &pickRow{choice: choice, full: choice, shown: true}
		if choice.Group != "" && choice.Group != group {
			group = choice.Group
			row.heading = paint.OptionHeading(choice.Group)
			items = append(items, row.heading)
		}
		at := index
		row.tap = NewTapRow(func(hovered bool) fyne.CanvasObject {
			this := p.rows[at].choice
			return paint.OptionFace(paint.OptionLook{
				Name: this.Value, Detail: this.Detail, Note: this.Note, NameWidth: p.nameWidth,
				Chosen: this.Value == p.chosen, Current: p.current == at, Hovered: hovered,
				Compact: p.compact,
			})
		}, func() { p.pick(at) })
		p.rows = append(p.rows, row)
		items = append(items, row.tap)
	}
	p.list = paint.Column(palette.NameStepHairline, items...)
	p.scroll = NewScroll(p.list)
	p.current = p.indexOf(p.chosen)
	if p.current < 0 && len(p.rows) > 0 {
		p.current = 0
	}
}

// buildGrouped puts the field above the rows and the foot below them.
func (p *picker) buildGrouped() {
	p.search = newSearchEntry(p)
	p.search.SetPlaceHolder(p.words.Prompt)
	p.search.OnChanged = func(typed string) { p.filter(typed) }
	p.nothing = paint.Pad(palette.NameStepSection, palette.NameStepSection,
		palette.NameStepRelated, palette.NameStepRelated,
		paint.Text(p.words.Nothing, palette.TypeBody, palette.TextSecondary))
	p.nothing.Hide()
	p.foot = paint.Text("", palette.TypeLabel, palette.TextSecondary)
	p.say()

	// The magnifier stands before the text, inside the edge, as the accepted
	// drawing has it: it is what says the field filters rather than takes a
	// value. The gap after it is what the chooser puts before its arrow.
	magnifier := paint.Pad(palette.NameStepHairline, palette.NameStepHairline,
		palette.NameStepHairline, palette.NameStepTight,
		paint.Middle(paint.Icon(paint.IconSearch, palette.TextSecondary, palette.IconSmall)))
	top := paint.Pad(palette.NameStepHairline, palette.NameStepTight,
		palette.NameStepHairline, palette.NameStepHairline,
		fieldWith(&p.search.FieldEntry, p.search, magnifier))
	enter := paint.Text(p.words.Enter, palette.TypeLabel, palette.TextSecondary)
	counts := container.New(layout.NewBorderLayout(nil, nil, p.foot, enter), p.foot, enter)
	rule := paint.Hairline(palette.LineQuiet)
	footer := container.New(layout.NewBorderLayout(rule, nil, nil, nil), rule,
		paint.Pad(palette.NameStepTight, palette.NameStepHairline,
			palette.NameStepRelated, palette.NameStepRelated, counts))
	// 🔴 No shadow along the edge the rows run under, since 2026-10-21: it lay
	// over the half row at the foot of the panel and the owner read the pair as
	// a list cut off. The panel now ends on a whole row (Fitted) and the bar at
	// the side is what says there are more. The shadow is gone from the whole
	// window (fynetheme, unpainted) rather than from this list alone, because a
	// theme of the list's own would be a new scope, and a new set of fonts, every
	// time the panel opens (B-370).
	middle := container.NewStack(p.scroll, p.nothing)
	// 🔴 The rows are the MIDDLE of the border and the field and the foot are
	// its edges, and that is the rule this project paid for once: only what
	// cannot grow may stand in an edge. An open list put in an edge gave the
	// middle a height of minus 171 and a panic in the rasteriser.
	inside := container.New(layout.NewBorderLayout(top, footer, nil, nil), top, footer, middle)
	p.body = paint.On(paint.PanelGround(), paint.PadAll(palette.NameStepRelated, inside))
}

// indexOf is where a value stands in the rows, or -1.
func (p *picker) indexOf(value string) int {
	for index, row := range p.rows {
		if row.choice.Value == value {
			return index
		}
	}
	return -1
}

// filter leaves on screen the rows whose value, detail or heading contains
// what was typed, and puts the keyboard on the first of them.
//
// Rows are shown and hidden rather than rebuilt, so a row keeps its picture
// under the pointer while somebody types, and so that a list of hundreds costs
// a walk rather than a rebuild on every character.
func (p *picker) filter(typed string) {
	wanted := strings.ToLower(strings.TrimSpace(typed))
	first, groupsShown := -1, map[string]bool{}
	for index, row := range p.rows {
		row.shown = wanted == "" || matches(row.choice, wanted)
		show(row.tap, row.shown)
		if row.shown {
			groupsShown[row.choice.Group] = true
			if first < 0 {
				first = index
			}
		}
	}
	for _, row := range p.rows {
		if row.heading != nil {
			show(row.heading, groupsShown[row.choice.Group])
		}
	}
	p.current = first
	if chosen := p.indexOf(p.chosen); wanted == "" && chosen >= 0 {
		p.current = chosen
	}
	show(p.scroll, first >= 0)
	show(p.nothing, first < 0)
	p.list.Refresh()
	p.redrawRows()
	p.say()
	if p.onShape != nil {
		p.onShape()
	}
	p.reveal()
}

// show is Show or Hide by a flag, in one place rather than four.
func show(object fyne.CanvasObject, shown bool) {
	if shown {
		object.Show()
		return
	}
	object.Hide()
}

// matches says whether a choice is left on screen by what was typed. Every
// word a person can see on the row counts, so the filter never hides a row
// somebody is looking straight at.
func matches(choice Choice, wanted string) bool {
	for _, visible := range []string{choice.Value, choice.Detail, choice.Group} {
		if strings.Contains(strings.ToLower(visible), wanted) {
			return true
		}
	}
	return false
}

// shownCount is how many rows the filter left.
func (p *picker) shownCount() int {
	count := 0
	for _, row := range p.rows {
		if row.shown {
			count++
		}
	}
	return count
}

// say fills the foot: the count of choices, or how many of them match.
func (p *picker) say() {
	if p.foot == nil {
		return
	}
	shown, all := p.shownCount(), len(p.rows)
	if shown == all {
		p.foot.Text = strconv.Itoa(all) + " " + p.words.Counted
	} else {
		p.foot.Text = fmt.Sprintf(p.words.Matches, shown, all)
	}
	p.foot.Refresh()
}

// redrawRows asks every row for its picture again, because the row the
// keyboard is on is part of the picture and moving it changes two rows.
func (p *picker) redrawRows() {
	for _, row := range p.rows {
		row.tap.Refresh()
	}
}

// move puts the keyboard on the next shown row in that direction, and stops at
// either end rather than wrapping, so that a person holding an arrow down
// arrives at the last row and stays there.
func (p *picker) move(step int) {
	for index := p.current + step; index >= 0 && index < len(p.rows); index += step {
		if p.rows[index].shown {
			p.current = index
			p.redrawRows()
			p.reveal()
			return
		}
	}
}

// reveal scrolls the row the keyboard is on into view, by the least it can.
func (p *picker) reveal() {
	if p.current < 0 || p.current >= len(p.rows) {
		return
	}
	row := p.rows[p.current].tap
	top, bottom := row.Position().Y, row.Position().Y+row.Size().Height
	window := p.scroll.Size().Height
	offset := p.scroll.Offset.Y
	switch {
	case window <= 0:
		return
	case top < offset:
		offset = top
	case bottom > offset+window:
		offset = bottom - window
	default:
		return
	}
	p.scroll.ScrollToOffset(fyne.NewPos(0, offset))
}

// pick hands the chosen value up and closes.
func (p *picker) pick(index int) {
	if index < 0 || index >= len(p.rows) || !p.rows[index].shown {
		return
	}
	value := p.rows[index].choice.Value
	if p.onPick != nil {
		p.onPick(value)
	}
	p.close()
}

func (p *picker) close() {
	if p.onClose != nil {
		p.onClose()
	}
}

// keyReceiver is what the canvas is asked to focus when the panel opens: the
// field when there is one, otherwise the panel itself.
func (p *picker) keyReceiver() fyne.Focusable {
	if p.search != nil {
		return p.search
	}
	return p
}

// FocusGained and FocusLost are what make the panel a place the keyboard can
// be. The row it is on is drawn by the rows rather than here.
func (p *picker) FocusGained() {}
func (p *picker) FocusLost()   {}

// TypedKey moves, picks and closes. The field forwards these same keys here,
// so there is one reading of them whichever thing holds the keyboard.
func (p *picker) TypedKey(event *fyne.KeyEvent) {
	if event == nil {
		return
	}
	switch event.Name {
	case fyne.KeyDown:
		p.move(1)
	case fyne.KeyUp:
		p.move(-1)
	case fyne.KeyReturn, fyne.KeyEnter:
		p.pick(p.current)
	case fyne.KeyEscape:
		p.close()
	}
}

// TypedRune puts the keyboard on the first row whose value starts with that
// character, which is how a flat list is searched without a field.
func (p *picker) TypedRune(r rune) {
	if p.search != nil {
		return
	}
	typed := strings.ToLower(string(r))
	for index, row := range p.rows {
		if row.shown && strings.HasPrefix(strings.ToLower(row.choice.Value), typed) {
			p.current = index
			p.redrawRows()
			p.reveal()
			return
		}
	}
}

// Tapped swallows a tap on the panel itself, so that a click on its margin
// does not fall through to the shield under it and close the panel a person
// is pointing at.
func (p *picker) Tapped(*fyne.PointEvent) {}

// Width is how wide the panel is over a chooser of that width.
//
// A grouped panel is the width the theme sets, because its rows carry a
// second column and a heading. A flat one is the width of its chooser, which
// is what every list under a chooser is, and no narrower than its own rows -
// seen on the real window on 2026-09-14, where the two locales stood in a
// panel four times wider than either of them.
func (p *picker) Width(chooser float32) float32 {
	if p.compact {
		// As wide as its rows with every detail in full, whatever was cut to
		// fit the last place it stood in.
		return max(chooser, p.natural)
	}
	if p.grouped() {
		return palette.PanelWidth
	}
	return max(chooser, p.body.MinSize().Width)
}

// ContentHeight is how tall the panel wants to be to show every row without
// scrolling, which the chooser caps at the ceiling the theme sets.
func (p *picker) ContentHeight() float32 {
	if !p.scroll.Visible() {
		// The sentence saying nothing matches stands where the rows were,
		// and it is as tall as it needs to be.
		return p.body.MinSize().Height
	}
	chrome := p.body.MinSize().Height - p.scroll.MinSize().Height
	return chrome + p.list.MinSize().Height
}

// Fitted is the height under this one at which the panel ends on the foot of a
// whole row, so that no row shows only its top half.
//
// The owner found the last row of the kinds cut in half on 2026-10-21. The
// panel is as tall as its rows up to a ceiling and up to the room on the
// screen, and either limit falls wherever it falls, so the rows the limit
// leaves are counted here and the panel gives back the part of the one it
// would have cut. It ends on a row and never on a heading, which with nothing
// under it reads as a group that is empty. A panel too short for even one row
// keeps the height it got, since a part of a row is still more than none.
func (p *picker) Fitted(height float32) float32 {
	if !p.scroll.Visible() {
		return height
	}
	chrome := p.body.MinSize().Height - p.scroll.MinSize().Height
	room := height - chrome
	gap := palette.StepOf(palette.NameStepHairline)
	reached, used := float32(0), float32(0)
	for _, item := range p.list.Objects {
		if !item.Visible() {
			continue
		}
		end := used + item.MinSize().Height
		if end > room {
			break
		}
		if _, row := item.(*TapRow); row {
			reached = end
		}
		used = end + gap
	}
	if reached == 0 {
		return height
	}
	return chrome + reached
}

// CreateRenderer draws the panel built once above.
func (p *picker) CreateRenderer() fyne.WidgetRenderer {
	return &pickerRenderer{panel: p}
}

// pickerRenderer holds the one tree the panel was built as. Nothing is
// swapped here: what changes between states lives on the rows, the foot and
// the visibility of the scrolling area, each of which refreshes itself.
type pickerRenderer struct {
	panel *picker
}

func (r *pickerRenderer) Layout(size fyne.Size) {
	r.panel.body.Resize(size)
	r.panel.body.Move(fyne.NewPos(0, 0))
}

func (r *pickerRenderer) MinSize() fyne.Size { return r.panel.body.MinSize() }

func (r *pickerRenderer) Refresh() { r.panel.body.Refresh() }

func (r *pickerRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.panel.body} }

func (r *pickerRenderer) Destroy() {}

// searchEntry is the field at the top of the panel: a FieldEntry that hands
// the keys meaning move, pick and close to the panel, and keeps the rest.
//
// It declares no renderer of its own, so the drawing of a field stays the one
// drawing every field here has.
type searchEntry struct {
	FieldEntry
	panel *picker
}

func newSearchEntry(panel *picker) *searchEntry {
	entry := &searchEntry{panel: panel}
	entry.ExtendBaseWidget(entry)
	return entry
}

// TypedKey forwards the navigation keys to the panel and types the rest.
func (e *searchEntry) TypedKey(event *fyne.KeyEvent) {
	if event == nil {
		return
	}
	switch event.Name {
	case fyne.KeyUp, fyne.KeyDown, fyne.KeyReturn, fyne.KeyEnter, fyne.KeyEscape:
		e.panel.TypedKey(event)
		return
	}
	e.FieldEntry.TypedKey(event)
}
