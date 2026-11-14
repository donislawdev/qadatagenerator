package socialcard_test

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/software"
	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
	"github.com/donislawdev/qadatagenerator/internal/adapter/mutators"
	"github.com/donislawdev/qadatagenerator/internal/app/corrupt"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/gui/fynetheme"
	"github.com/donislawdev/qadatagenerator/internal/gui/theme"
	"github.com/donislawdev/qadatagenerator/internal/quick"
	"github.com/donislawdev/qadatagenerator/internal/take"
)

var update = flag.Bool("update", false, "draw the social preview again and write it with the list of what it says")

const (
	pictureFile = "../../../.github/social-preview/social-preview.png"
	recordFile  = "../../../.github/social-preview/says.txt"
	iconFile    = "../appicon/lentils.ico"

	// redraw is the command every message here ends with.
	redraw = "go test ./internal/gui/socialcard -run TestTheSocialPreviewSaysWhatTheProgramDoes -update"
	upload = "upload it on GitHub: Settings, General, Social preview, Edit. GitHub keeps a copy of " +
		"its own, so a new picture here changes nothing there until somebody uploads it"
)

// recordHeader opens says.txt. Lines starting with "# " are not texts of the
// picture, and no text drawn on it starts that way.
var recordHeader = []string{
	"# Every text drawn on social-preview.png, one to a line, in the order drawn.",
	"# Written by " + redraw,
}

// The picture. 1280 by 640 is what GitHub asks for, and a 2:1 picture is what
// most sites show whole. Sites that cut to 1.91:1 take about 30 pixels off each
// side, so nothing sits closer than the margin to an edge. A feed shows it about
// 500 pixels wide, so there are few things on it and none of them small.
const (
	width, height = 1280, 640
	margin        = 64
	// leftRoom is the width of the left column, and the right column starts
	// at rightX and runs to the margin.
	leftRoom = 600
	rightX   = 716
	// edgeRoom is the least a block may leave above and below itself.
	edgeRoom = 36
)

// The type, in pixels of the picture.
const (
	nameSize     = 32
	pillSize     = 17
	headlineSize = 58
	headlineStep = 68
	subSize      = 22
	subStep      = 30
	bulletSize   = 21
	bulletStep   = 29
	footSize     = 18
	headSize     = 16
	labelSize    = 15
	valueSize    = 21
	tagSize      = 17
	captionSize  = 15
)

// The distances between things, also in pixels of the picture.
const (
	iconSize     = 64
	brandGap     = 18
	pillPadX     = 12
	pillPadY     = 4
	headlineGap  = 30
	subGap       = 14
	bulletsGap   = 16
	bulletGap    = 8
	bulletIndent = 24
	markerSize   = 10
	markerAt     = 11
	footGap      = 22
	cardPad      = 20
	cardRadius   = 14
	cardGap      = 16
	headerHeight = 44
	goodHeight   = 52
	labelColumn  = 82
	badHeight    = 72
	badValueAt   = 36
	valueGap     = 10
	tagPadX      = 13
	tagPadY      = 5
	tagGap       = 16
	captionGap   = 14
	// washAlpha is how strongly a character is marked. The badge wash of the
	// window is fainter, but under one digit it is not seen at feed size.
	washAlpha = 0x55
)

// The words of the card that are not read from the program. Changing one of
// these is changing the picture, and says.txt shows it in the diff.
const (
	productName = "QA Data Generator"
	surfaces    = "GUI + CLI"
	headline1   = "Good data. Bad data."
	headline2   = "And the right answer."
	sub         = "A test data generator for forms, APIs and file imports. It makes whole datasets, breaks them on purpose, and tells your tests what your app should do with each value."
	recipeLead  = "Recipes"
	recipeRest  = "describe your own dataset in YAML, get thousands of rows in CSV or JSON"
	goodLead    = "Good data"
	goodRest    = "%s and %d more"
	badLead     = "Bad data"
	badRest     = "%d ways to break a value"
	answerLead  = "The right answer"
	answerRest  = "accept or reject, in a file your CI reads"
	leadSep     = " - "
	foot        = "Free, GPL-3.0  ·  works offline  ·  Windows, macOS, Linux"
	goodHead    = "good data"
	badHead     = "bad and tricky data"
	verdictHead = "your app should"
	manifestFor = "expectation-manifest.json"
	aboutSep    = "  ·  "
	arrow       = "→"
	caption     = "From real runs of qadg quick and qadg corrupt."
)

var (
	// The card speaks to the whole world, so it leads with identifiers used
	// everywhere and shows one national one, the Polish PESEL, as an example.
	//
	// named are the identifiers the point about good data names. Each has to be
	// one this build generates. IBAN is left out because this build makes
	// Polish ones only, and a reader elsewhere would expect their own.
	named = []string{"ssn", "vin", "isbn", "imei", "pesel"}
	// goodShown are the generators whose values the first card shows. SSN is
	// left out because its values are drawn from numbers never issued, on
	// purpose, and a card saying such a value should be accepted would be
	// arguing with the ethics of the tool.
	goodShown = []string{"vin", "imei", "pesel"}
	goodSeed  = "card"
	goodPlace = "en-US"

	// The run whose manifest the second card quotes: book numbers and
	// surnames. The rows are the first applied entry of each damage named
	// here, in this order, so the card shows two refusals of a broken check
	// digit beside a real surname that has to be accepted.
	shownLayers = []string{"L3", "REV"}
	shown       = []catalog.ID{
		"L3.checksum.off_by_one",
		"L3.checksum.transposed_digits",
		"REV.name.null_surname",
	}
	fileGenerator = "isbn"
	fileRows      = 12
	surnames      = []string{"Smith", "Johnson"}
)

// phrases say each quoted kind of damage in plain words, because a feed is no
// place to teach the identifiers of the catalog.
var phrases = map[catalog.ID]string{
	"L3.checksum.off_by_one":        "check digit off by one",
	"L3.checksum.transposed_digits": "two digits swapped",
	"REV.name.null_surname":         "a real surname, Null",
}

// verdicts are the words and colours of what the application should do. A value
// the domain gains and this list lacks is a problem of the card, not a blank.
var verdicts = map[catalog.Expectation]struct{ words, ink string }{
	catalog.Rejection:  {"reject", theme.StateDanger},
	catalog.Acceptance: {"accept", theme.StateSuccess},
	catalog.Warning:    {"accept and mark", theme.StateWarning},
}

func TestMain(m *testing.M) {
	flag.Parse()
	test.NewApp().Settings().SetTheme(fynetheme.Current())
	os.Exit(m.Run())
}

// 🔴 The picture GitHub shows says what this build does, and looks the way this
// file draws it.
//
// Two checks, because they fail for different reasons. says.txt against the
// texts drawn now is exact and the same on every machine: a new identifier or a
// new kind of damage is a changed line. The pixels are compared only where the
// picture is drawn, Windows on amd64, because the rasteriser works in floating
// point and arm64 fuses operations amd64 does not, which moved the icon by a
// pixel when it was measured on 2026-11-14. A changed colour or distance is a
// changed picture there.
func TestTheSocialPreviewSaysWhatTheProgramDoes(t *testing.T) {
	drawn := draw(gather(t), iconPicture(t))
	if len(drawn.problems) > 0 {
		t.Fatalf("the card no longer fits or says something it has no words for:\n  %s",
			strings.Join(drawn.problems, "\n  "))
	}
	picture := render(drawn)

	if *update {
		writePicture(t, picture)
		writeRecord(t, drawn.says)
		t.Logf("wrote %s and %s. Look at the picture, commit both, and %s", pictureFile, recordFile, upload)
		return
	}

	if found := stale(readRecord(t), drawn.says); len(found) > 0 {
		t.Errorf("the picture GitHub shows no longer says what this build does:\n  %s\n"+
			"Draw it again with\n  %s\nthen look at it, commit it with says.txt, and %s",
			strings.Join(found, "\n  "), redraw, upload)
	}

	if !drawnHere() {
		t.Logf("the pixels are compared on windows/amd64, where the picture is drawn, and this is %s/%s",
			runtime.GOOS, runtime.GOARCH)
		return
	}
	if n := differing(readPicture(t), picture); n > 0 {
		t.Errorf("the picture differs from the card as this file draws it now in %d pixels, so a "+
			"colour, a distance or the icon changed. Draw it again with\n  %s\nthen look at it, "+
			"commit it, and %s", n, redraw, upload)
	}
}

// The check above notices a card that says one more kind of damage, both in
// the list of texts and in the pixels.
func TestACardWithAnotherNumberIsNoticed(t *testing.T) {
	facts, icon := gather(t), iconPicture(t)
	now := draw(facts, icon)
	more := facts
	more.damage++
	later := draw(more, icon)
	if len(stale(now.says, later.says)) == 0 {
		t.Fatalf("a card saying %d kinds of damage read as the same card as one saying %d, so "+
			"a picture that fell behind the program would pass", more.damage, facts.damage)
	}
	if drawnHere() && differing(render(now), render(later)) == 0 {
		t.Fatal("a card saying another number drew the same pixels, so the comparison of " +
			"pixels would pass a picture that fell behind the program")
	}
}

// A card that does not fit, or that would have to say something it has no
// words for, is refused rather than drawn.
func TestACardThatDoesNotFitIsRefused(t *testing.T) {
	facts, icon := gather(t), iconPicture(t)

	wide := facts
	wide.rows = slices.Clone(facts.rows)
	wide.rows[0].after = strings.Repeat("9", 80)
	if problems := draw(wide, icon).problems; len(problems) == 0 {
		t.Error("a value of eighty digits fitted its row, so a longer value would run under its verdict unnoticed")
	}

	unnamed := facts
	unnamed.rows = slices.Clone(facts.rows)
	unnamed.rows[0].mutator = "L3.checksum.never_written"
	if problems := draw(unnamed, icon).problems; len(problems) == 0 {
		t.Error("a kind of damage with no plain words was drawn, so the card would show a blank")
	}

	fewer := facts
	fewer.identifiers = []string{"pesel"}
	if problems := draw(fewer, icon).problems; len(problems) == 0 {
		t.Error("the card named identifiers this build no longer generates and drew it anyway")
	}
}

// facts are everything the card says that the program decides.
type facts struct {
	damage      int
	identifiers []string
	good        []sample
	rows        []row
}

// sample is one value a generator produced.
type sample struct{ generator, value string }

// row is one entry of the manifest the card quotes.
type row struct {
	field, before, after string
	mutator              catalog.ID
	expectation          catalog.Expectation
}

// gather asks the program: which of its generators are identifiers, how many
// kinds of damage it implements, what three of its generators produce, and what
// a run of the damage over a file of book numbers and surnames wrote in its
// manifest.
func gather(t *testing.T) facts {
	t.Helper()
	identifiers := map[string]bool{}
	for _, locale := range quick.Locales() {
		for _, generator := range quick.Generators(locale) {
			if quick.Describe(generator, locale).Produces == "identifier" {
				identifiers[generator] = true
			}
		}
	}
	found := facts{
		damage:      len(mutators.IDs()),
		identifiers: sortedKeys(identifiers),
		rows:        corrupted(t),
	}
	for _, generator := range goodShown {
		values, err := quick.Run(quick.Request{Generator: generator, Locale: goodPlace, Seed: goodSeed, Count: 1})
		if err != nil || len(values) != 1 {
			t.Fatalf("the generator %s gave %d values and %v, so the card would show something it did not make",
				generator, len(values), err)
		}
		found.good = append(found.good, sample{generator, values[0]})
	}
	if found.damage == 0 || len(found.identifiers) == 0 {
		t.Fatalf("the program reported %d kinds of damage and %d identifiers, and a card about "+
			"nothing would be drawn without complaint", found.damage, len(found.identifiers))
	}
	return found
}

// corrupted runs the damage over a file of valid identifiers and surnames, the
// way qadg corrupt and the window both do, and returns the rows the card
// quotes, read back from the manifest the run wrote.
func corrupted(t *testing.T) []row {
	t.Helper()
	dir := t.TempDir()
	identifiers, err := quick.Run(quick.Request{Generator: fileGenerator, Locale: goodPlace, Seed: goodSeed, Count: fileRows})
	if err != nil {
		t.Fatalf("the generator did not give the values of the file: %v", err)
	}
	lines := []string{fileGenerator + ",last_name"}
	for i, value := range identifiers {
		lines = append(lines, value+","+surnames[i%len(surnames)])
	}
	source := filepath.Join(dir, "people.csv")
	if err := os.WriteFile(source, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatalf("cannot write the file the run damages: %v", err)
	}

	settings := take.Defaults()
	settings.Layers = shownLayers
	settings.Rows = "100%"
	settings.Out = filepath.Join(dir, "out")
	described, err := catalogyaml.Load()
	if err != nil {
		t.Fatalf("the catalog inside this build could not be read: %v", err)
	}
	opened, err := take.Open(source, settings, corrupt.NeverStops{})
	if err != nil {
		t.Fatalf("the file could not be read: %v", err)
	}
	defer func() { _ = opened.Close() }()
	if _, err := opened.Corrupt(settings, described, corrupt.NeverStops{}, time.Now()); err != nil {
		t.Fatalf("the run did not finish: %v", err)
	}
	record, err := manifestjson.Load(filepath.Join(settings.Out, manifestjson.Name))
	if err != nil {
		t.Fatalf("the manifest could not be read back: %v", err)
	}

	var rows []row
	for _, wanted := range shown {
		for _, defect := range record.Defects() {
			if defect.Mutator == wanted && defect.Applied {
				rows = append(rows, row{
					field: defect.Location.Field, before: defect.ValueBefore, after: defect.ValueAfter,
					mutator: defect.Mutator, expectation: defect.Expectation,
				})
				break
			}
		}
	}
	if len(rows) != len(shown) {
		t.Fatalf("the run applied %d of the %d kinds of damage the card quotes, so the card would "+
			"show something no run did", len(rows), len(shown))
	}
	return rows
}

// card is a drawing being made: what it draws, every text it says, and every
// reason it cannot be drawn.
type card struct {
	objects  []fyne.CanvasObject
	says     []string
	problems []string
}

func (c *card) text(words string, at fyne.Position, size float32, style fyne.TextStyle, ink string) fyne.Size {
	drawn := canvas.NewText(words, theme.Ink(ink))
	drawn.TextSize = size
	drawn.TextStyle = style
	measured := drawn.MinSize()
	drawn.Resize(measured)
	drawn.Move(at)
	c.objects = append(c.objects, drawn)
	c.says = append(c.says, words)
	return measured
}

func (c *card) box(at fyne.Position, size fyne.Size, radius float32, fill, edge color.Color) {
	drawn := canvas.NewRectangle(fill)
	drawn.CornerRadius = radius
	if edge != nil {
		drawn.StrokeColor = edge
		drawn.StrokeWidth = 1
	}
	drawn.Resize(size)
	drawn.Move(at)
	c.objects = append(c.objects, drawn)
}

func (c *card) fits(what string, used, room float32) {
	if used > room {
		c.problems = append(c.problems, fmt.Sprintf("%s takes %.0f px where %.0f px are free", what, used, room))
	}
}

// centre moves everything drawn since the first index down so that a block of
// that height sits in the middle of the picture.
func (c *card) centre(first int, tall float32, what string) {
	c.fits(what+" in height", tall, height-2*edgeRoom)
	down := float32(math.Round(float64(height-tall) / 2))
	for _, object := range c.objects[first:] {
		object.Move(object.Position().AddXY(0, down))
	}
}

func draw(f facts, icon image.Image) *card {
	c := &card{}
	ground := canvas.NewImageFromImage(groundPicture())
	ground.Resize(fyne.NewSize(width, height))
	c.objects = append(c.objects, ground)
	c.left(f, icon)
	c.right(f)
	return c
}

// left is who, the line, the three things it gives, and where it runs.
func (c *card) left(f facts, icon image.Image) {
	first := len(c.objects)
	x, y := float32(margin), float32(0)
	bold := fyne.TextStyle{Bold: true}

	picture := canvas.NewImageFromImage(icon)
	picture.FillMode = canvas.ImageFillContain
	picture.Resize(fyne.NewSize(iconSize, iconSize))
	picture.Move(fyne.NewPos(x, y))
	c.objects = append(c.objects, picture)

	nameAt := x + iconSize + brandGap
	name := fyne.MeasureText(productName, nameSize, bold)
	c.text(productName, fyne.NewPos(nameAt, y+(iconSize-name.Height)/2), nameSize, bold, theme.TextPrimary)
	pill := fyne.MeasureText(surfaces, pillSize, bold)
	pillAt := nameAt + name.Width + brandGap
	pillTall := pill.Height + 2*pillPadY
	c.box(fyne.NewPos(pillAt, y+(iconSize-pillTall)/2), fyne.NewSize(pill.Width+2*pillPadX, pillTall),
		pillTall/2, theme.Ink(theme.BgCurrent), theme.Ink(theme.MarkCurrent))
	c.text(surfaces, fyne.NewPos(pillAt+pillPadX, y+(iconSize-pill.Height)/2), pillSize, bold, theme.TextAccent)
	c.fits("the name and "+surfaces, pillAt+pill.Width+2*pillPadX-x, leftRoom)

	y += iconSize + headlineGap
	for i, line := range []string{headline1, headline2} {
		ink := theme.TextPrimary
		if i == 1 {
			ink = theme.TextAccent
		}
		used := c.text(line, fyne.NewPos(x, y), headlineSize, bold, ink)
		c.fits("the headline", used.Width, leftRoom)
		y += headlineStep
	}

	y += subGap
	for _, line := range wrap(sub, subSize, fyne.TextStyle{}, leftRoom, leftRoom) {
		c.text(line, fyne.NewPos(x, y), subSize, fyne.TextStyle{}, theme.TextSecondary)
		y += subStep
	}

	var spoken []string
	for _, id := range named {
		if !slices.Contains(f.identifiers, id) {
			c.problems = append(c.problems, "the card names "+id+", which this build no longer generates")
		}
		spoken = append(spoken, strings.ToUpper(id))
	}
	y += bulletsGap
	for _, point := range []struct{ lead, rest string }{
		{recipeLead, recipeRest},
		{goodLead, fmt.Sprintf(goodRest, strings.Join(spoken, ", "), len(f.identifiers)-len(named))},
		{badLead, fmt.Sprintf(badRest, f.damage)},
		{answerLead, answerRest},
	} {
		y = c.bullet(point.lead, point.rest, x, y) + bulletGap
	}

	y += footGap - bulletGap
	last := c.text(foot, fyne.NewPos(x, y), footSize, fyne.TextStyle{}, theme.TextSecondary)
	c.fits("the foot", last.Width, leftRoom)
	c.centre(first, y+last.Height, "the left column")
}

// bullet draws one point, its lead in bold and the rest wrapped after it, and
// returns where the next one starts.
func (c *card) bullet(lead, rest string, x, y float32) float32 {
	bold := fyne.TextStyle{Bold: true}
	c.box(fyne.NewPos(x, y+markerAt), fyne.NewSize(markerSize, markerSize), markerSize/4, theme.Ink(theme.TextAccent), nil)
	at := x + bulletIndent
	room := float32(leftRoom - bulletIndent)
	led := c.text(lead, fyne.NewPos(at, y), bulletSize, bold, theme.TextPrimary)
	lines := wrap(leadSep+rest, bulletSize, fyne.TextStyle{}, room-led.Width, room)
	for i, line := range lines {
		lineAt := at
		if i == 0 {
			lineAt += led.Width
		}
		c.text(line, fyne.NewPos(lineAt, y), bulletSize, fyne.TextStyle{}, theme.TextSecondary)
		y += bulletStep
	}
	return y
}

// right tells the headline again with values: what the generators made, what
// the damage did to such values, and what the application should do with each.
func (c *card) right(f facts) {
	first := len(c.objects)
	x, room := float32(rightX), float32(width-margin-rightX)
	mono := fyne.TextStyle{Monospace: true}
	monoBold := fyne.TextStyle{Monospace: true, Bold: true}

	// Good data.
	tall := float32(headerHeight + goodHeight*len(f.good))
	c.panel(0, tall, goodHead, verdictHead, theme.TextAccent, fyne.TextStyle{})
	accept := verdicts[catalog.Acceptance]
	for i, made := range f.good {
		at := headerHeight + float32(i*goodHeight)
		c.rule(at)
		name := strings.ToUpper(made.generator)
		label := fyne.MeasureText(name, labelSize, fyne.TextStyle{})
		value := fyne.MeasureText(made.value, valueSize, monoBold)
		c.text(name, fyne.NewPos(x+cardPad, at+(goodHeight-label.Height)/2), labelSize, fyne.TextStyle{}, theme.TextSecondary)
		c.text(made.value, fyne.NewPos(x+cardPad+labelColumn, at+(goodHeight-value.Height)/2), valueSize, monoBold, theme.TextPrimary)
		tagWide := c.tag(accept.words, accept.ink, at, goodHeight)
		c.fits("the value of "+made.generator, labelColumn+value.Width+tagGap+tagWide, room-2*cardPad)
	}

	// Bad data, from the manifest.
	top := tall + cardGap
	tall = float32(headerHeight + badHeight*len(f.rows))
	c.panel(top, tall, badHead, manifestFor, theme.TextSecondary, mono)
	for i, entry := range f.rows {
		at := top + headerHeight + float32(i*badHeight)
		c.rule(at)
		phrase, known := phrases[entry.mutator]
		if !known {
			c.problems = append(c.problems, fmt.Sprintf("the card has no plain words for %s", entry.mutator))
		}
		about := c.text(entry.field+aboutSep+phrase, fyne.NewPos(x+cardPad, at+cardPad/2+2), labelSize,
			fyne.TextStyle{}, theme.TextSecondary)

		vx := x + cardPad
		before := c.text(entry.before, fyne.NewPos(vx, at+badValueAt), valueSize, mono, theme.TextSecondary)
		vx += before.Width + valueGap
		pointer := c.text(arrow, fyne.NewPos(vx, at+badValueAt), valueSize, mono, theme.TextAccent)
		vx += pointer.Width + valueGap
		c.marks(entry.before, entry.after, fyne.NewPos(vx, at+badValueAt))
		after := c.text(entry.after, fyne.NewPos(vx, at+badValueAt), valueSize, monoBold, theme.TextPrimary)
		valueWide := vx + after.Width - (x + cardPad)

		answer, known := verdicts[entry.expectation]
		if !known {
			c.problems = append(c.problems, fmt.Sprintf("the card has no words for the expectation %q", entry.expectation))
			continue
		}
		tagWide := c.tag(answer.words, answer.ink, at, badHeight)
		c.fits("row "+fmt.Sprint(i+1)+" of the manifest", max(about.Width, valueWide)+tagGap+tagWide, room-2*cardPad)
	}

	below := top + tall + captionGap
	written := c.text(caption, fyne.NewPos(x+2, below), captionSize, fyne.TextStyle{}, theme.TextSecondary)
	c.fits("the caption", written.Width, room)
	c.centre(first, below+written.Height, "the right column")
}

// panel draws a card of the right column with its head: what it holds on the
// left and what the column on the right means.
func (c *card) panel(top, tall float32, holds, right, rightInk string, rightStyle fyne.TextStyle) {
	x, room := float32(rightX), float32(width-margin-rightX)
	bold := fyne.TextStyle{Bold: true}
	c.box(fyne.NewPos(x, top), fyne.NewSize(room, tall), cardRadius, theme.Ink(theme.BgSurface), theme.Ink(theme.LineQuiet))
	c.box(fyne.NewPos(x+1, top+1), fyne.NewSize(room-2, headerHeight-1), cardRadius-1, theme.Ink(theme.BgHeader), nil)
	c.box(fyne.NewPos(x+1, top+headerHeight-cardRadius), fyne.NewSize(room-2, cardRadius-1), 0, theme.Ink(theme.BgHeader), nil)
	left := fyne.MeasureText(holds, headSize, bold)
	c.text(holds, fyne.NewPos(x+cardPad, top+(headerHeight-left.Height)/2), headSize, bold, theme.TextPrimary)
	words := fyne.MeasureText(right, headSize, rightStyle)
	c.text(right, fyne.NewPos(x+room-cardPad-words.Width, top+(headerHeight-words.Height)/2), headSize, rightStyle, rightInk)
	c.fits("the head of "+holds, left.Width+tagGap+words.Width, room-2*cardPad)
}

// rule is the line above a row of the right column.
func (c *card) rule(at float32) {
	c.box(fyne.NewPos(rightX+1, at), fyne.NewSize(width-margin-rightX-2, 1), 0, theme.Ink(theme.LineQuiet), nil)
}

// tag draws a verdict at the right end of a row and returns its width.
func (c *card) tag(words, ink string, at, rowTall float32) float32 {
	bold := fyne.TextStyle{Bold: true}
	measured := fyne.MeasureText(words, tagSize, bold)
	wide, tall := measured.Width+2*tagPadX, measured.Height+2*tagPadY
	place := fyne.NewPos(width-margin-cardPad-wide, at+(rowTall-tall)/2)
	shade := theme.Ink(ink)
	c.box(place, fyne.NewSize(wide, tall), tall/4, color.NRGBA{R: shade.R, G: shade.G, B: shade.B, A: theme.BadgeWashAlpha}, nil)
	c.text(words, place.AddXY(tagPadX, tagPadY), tagSize, bold, ink)
	return wide
}

// marks washes the characters the damage changed, so one wrong digit is seen
// at a glance. Values of different lengths were replaced whole and get none.
func (c *card) marks(before, after string, at fyne.Position) {
	was, now := []rune(before), []rune(after)
	if len(was) != len(now) {
		return
	}
	for i := range now {
		if was[i] != now[i] {
			c.wash(at, i, 1, theme.StateDanger)
		}
	}
}

// wash lays a faint colour under count characters of a value, starting at the
// character from.
func (c *card) wash(at fyne.Position, from, count int, ink string) {
	if count == 0 {
		return
	}
	cell := fyne.MeasureText("0", valueSize, fyne.TextStyle{Monospace: true, Bold: true})
	shade := theme.Ink(ink)
	c.box(at.AddXY(float32(from)*cell.Width, 0), fyne.NewSize(float32(count)*cell.Width, cell.Height), 3,
		color.NRGBA{R: shade.R, G: shade.G, B: shade.B, A: washAlpha}, nil)
}

// wrap breaks words into lines, the first no wider than firstRoom and the rest
// no wider than room.
func wrap(words string, size float32, style fyne.TextStyle, firstRoom, room float32) []string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(words) {
		longer := strings.TrimLeft(line+" "+word, " ")
		if len(lines) == 0 && strings.HasPrefix(words, " ") {
			longer = line + " " + word
		}
		free := room
		if len(lines) == 0 {
			free = firstRoom
		}
		if strings.TrimSpace(line) != "" && fyne.MeasureText(longer, size, style).Width > free {
			lines = append(lines, line)
			longer = word
		}
		line = longer
	}
	return append(lines, line)
}

// groundPicture is the window's colour with two faint lights, the accent from
// the top right and a warm one from the bottom left, the colour of the icon.
func groundPicture() image.Image {
	base := theme.Ink(theme.BgWindow)
	lights := []struct {
		x, y, reach, strength float64
		ink                   color.NRGBA
	}{
		{1150, 20, 760, 0.16, theme.Ink(theme.TextAccent)},
		{40, 680, 560, 0.08, theme.Ink(theme.StateWarning)},
	}
	out := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			r, g, b := float64(base.R), float64(base.G), float64(base.B)
			for _, light := range lights {
				near := 1 - math.Hypot(float64(x)-light.x, float64(y)-light.y)/light.reach
				if near <= 0 {
					continue
				}
				share := near * near * light.strength
				r += (float64(light.ink.R) - r) * share
				g += (float64(light.ink.G) - g) * share
				b += (float64(light.ink.B) - b) * share
			}
			out.SetNRGBA(x, y, color.NRGBA{R: uint8(r + 0.5), G: uint8(g + 0.5), B: uint8(b + 0.5), A: 0xFF})
		}
	}
	return out
}

func render(c *card) image.Image {
	surface := software.NewCanvas()
	surface.SetPadded(false)
	surface.SetScale(1)
	surface.SetContent(container.NewWithoutLayout(c.objects...))
	surface.Resize(fyne.NewSize(width, height))
	return surface.Capture()
}

// iconPicture is the 64 pixel picture from the icon file the executables carry,
// which is the drawing averaged down from the size it is made at.
func iconPicture(t *testing.T) image.Image {
	t.Helper()
	data, err := os.ReadFile(iconFile)
	if err != nil {
		t.Fatalf("the icon file is missing: %v", err)
	}
	const head, entry = 6, 16
	if len(data) < head {
		t.Fatalf("the icon file holds %d bytes, fewer than its own header", len(data))
	}
	count := int(binary.LittleEndian.Uint16(data[4:head]))
	for i := range count {
		start := head + i*entry
		if start+entry > len(data) {
			break
		}
		fields := data[start : start+entry]
		if int(fields[0]) != iconSize {
			continue
		}
		size := int(binary.LittleEndian.Uint32(fields[8:12]))
		offset := int(binary.LittleEndian.Uint32(fields[12:16]))
		if offset+size > len(data) {
			t.Fatalf("the %d pixel picture runs past the end of the icon file", iconSize)
		}
		picture, err := png.Decode(bytes.NewReader(data[offset : offset+size]))
		if err != nil {
			t.Fatalf("the %d pixel picture of the icon file is not a PNG: %v", iconSize, err)
		}
		return picture
	}
	t.Fatalf("the icon file has no %d pixel picture", iconSize)
	return nil
}

// stale lists what one list of texts has and the other lacks. Texts in another
// order are a difference too, because the order is the order they are drawn.
func stale(was, now []string) []string {
	var found []string
	for _, line := range now {
		if !slices.Contains(was, line) {
			found = append(found, "the card now says    "+line)
		}
	}
	for _, line := range was {
		if !slices.Contains(now, line) {
			found = append(found, "the picture says     "+line)
		}
	}
	if len(found) == 0 && !slices.Equal(was, now) {
		found = append(found, "the same texts, drawn in another order or another number of times")
	}
	return found
}

// differing counts the pixels in which any channel differs by more than a few
// levels. Pictures of different sizes differ everywhere.
func differing(want, got image.Image) int {
	if want.Bounds().Size() != got.Bounds().Size() {
		return want.Bounds().Dx() * want.Bounds().Dy()
	}
	const levels = 8 << 8
	wb, gb := want.Bounds(), got.Bounds()
	n := 0
	for y := range wb.Dy() {
		for x := range wb.Dx() {
			wr, wg, wbl, wa := want.At(wb.Min.X+x, wb.Min.Y+y).RGBA()
			gr, gg, gbl, ga := got.At(gb.Min.X+x, gb.Min.Y+y).RGBA()
			for _, pair := range [][2]uint32{{wr, gr}, {wg, gg}, {wbl, gbl}, {wa, ga}} {
				if max(pair[0], pair[1])-min(pair[0], pair[1]) > levels {
					n++
					break
				}
			}
		}
	}
	return n
}

func drawnHere() bool { return runtime.GOOS == "windows" && runtime.GOARCH == "amd64" }

func sortedKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func writePicture(t *testing.T, picture image.Image) {
	t.Helper()
	var encoded bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(&encoded, picture); err != nil {
		t.Fatalf("encoding the picture: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(pictureFile), 0o755); err != nil {
		t.Fatalf("making the directory of the picture: %v", err)
	}
	if err := os.WriteFile(pictureFile, encoded.Bytes(), 0o644); err != nil {
		t.Fatalf("writing the picture: %v", err)
	}
}

func writeRecord(t *testing.T, says []string) {
	t.Helper()
	lines := append(slices.Clone(recordHeader), says...)
	if err := os.WriteFile(recordFile, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("writing the list of what the picture says: %v", err)
	}
}

func readRecord(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(recordFile)
	if err != nil {
		t.Fatalf("nothing says what the picture says, so a picture behind the program would pass: %v", err)
	}
	var says []string
	for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if !strings.HasPrefix(line, "# ") {
			says = append(says, line)
		}
	}
	if len(says) == 0 {
		t.Fatalf("%s lists no texts, so this would pass whatever the card says", recordFile)
	}
	return says
}

func readPicture(t *testing.T) image.Image {
	t.Helper()
	data, err := os.ReadFile(pictureFile)
	if err != nil {
		t.Fatalf("the picture is missing: %v", err)
	}
	picture, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("the picture is not a PNG: %v", err)
	}
	return picture
}
