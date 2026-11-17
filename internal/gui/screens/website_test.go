package screens

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"html"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/software"
	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
	"github.com/donislawdev/qadatagenerator/internal/adapter/recipeyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/responsejson"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/corrupt"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/domain/response"
	"github.com/donislawdev/qadatagenerator/internal/produce"
	"github.com/donislawdev/qadatagenerator/internal/take"
)

const (
	// websitePictures is where the site takes its pictures of the window from.
	websitePictures = "../../../web/assets/window"
	// readmeFile is the page the data of those pictures comes from.
	readmeFile = "../../../README.md"
	// websiteScale is twice the size the window is laid out at, so the pictures
	// stay sharp on a screen that draws two pixels for one, and the site shows
	// them at half their width.
	websiteScale = 2
	// redrawWebsite is the command that draws the pictures again.
	redrawWebsite = "go test ./internal/gui/screens -run TestTheWebsiteShowsTheWindowAsItIs -update"
	// recordHead opens the list of texts beside each picture.
	recordHead = "# Every text the picture beside this file shows, in the order the window draws them.\n" +
		"# Written by " + redrawWebsite + ". Do not edit.\n"
)

// websiteRun is the moment the run behind these pictures is dated. Fixed, so
// that nothing the window shows of it changes from one day to the next.
var websiteRun = time.Date(2026, time.November, 14, 12, 0, 0, 0, time.UTC)

// 🔴 Every picture of the window on the website is the window as this build
// draws it.
//
// The pictures are drawn from what the README shows, by the program itself: a
// file of customers is generated from the recipe the README saves as shop.yaml,
// damaged the way the README damages a file of your own, and the manifest that
// run writes is opened beside the record of what an application did that the
// README prints.
//
// Two checks, because they fail for different reasons. The texts are compared
// on every system, and a screen that says something else, or a program that
// produces something else, is a changed line. The pixels are compared where the
// pictures are drawn, Windows on amd64, for the reason the social preview gives:
// the rasteriser works in floating point and arm64 fuses operations amd64 does
// not. A changed colour, distance or icon is a changed picture there.
func TestTheWebsiteShowsTheWindowAsItIs(t *testing.T) {
	pictures, err := filepath.Abs(websitePictures)
	if err != nil {
		t.Fatal(err)
	}
	data, recipe := websiteData(t)
	for _, state := range Website(recipe) {
		t.Run(state.Name, func(t *testing.T) {
			if buildable, missing := state.Buildable(data); !buildable {
				t.Fatalf("the data made from the README cannot build this state: %s", missing)
			}
			picture, says := photograph(data, state)
			if len(says) == 0 {
				t.Fatal("the picture shows no text at all, so this check would pass whatever it shows")
			}
			stored := filepath.Join(pictures, state.Name)
			if *update {
				writeWebsitePicture(t, stored, picture, says)
				return
			}
			if found := stale(readSays(t, stored+".txt"), says); len(found) > 0 {
				t.Errorf(behindTheWindow+":\n  %s\n"+
					"Draw it again with\n  %s\nthen look at it and commit it with its list of texts",
					strings.Join(found, "\n  "), redrawWebsite)
			}
			if !picturesDrawnHere() {
				t.Logf("the pixels are compared on windows/amd64, where the pictures are drawn, and this is %s/%s",
					runtime.GOOS, runtime.GOARCH)
				return
			}
			if n := differing(readPicture(t, stored+".png"), picture); n > 0 {
				t.Errorf("the picture on the website differs from the window in %d pixels, so a colour, a "+
					"distance or an icon changed. Draw it again with\n  %s\nthen look at it and commit it",
					n, redrawWebsite)
			}
		})
	}
}

// The check above notices a window that shows other values, both in the
// texts and in the pixels.
func TestAPictureOfAnotherWindowIsNoticed(t *testing.T) {
	data, recipe := websiteData(t)
	shown := Website(recipe)[0]
	other := shown
	other.Quick.Seed = websiteSeed + "-other"
	before, saidBefore := photograph(data, shown)
	after, saidAfter := photograph(data, other)
	if len(stale(saidBefore, saidAfter)) == 0 {
		t.Error("a window showing values from another seed read as the same window, so a picture " +
			"that fell behind the program would pass")
	}
	if differing(before, after) == 0 {
		t.Error("a window showing values from another seed has the same pixels, so a picture that " +
			"fell behind the program would pass on the machine that compares them")
	}
	if found := stale(saidBefore, slices.Clone(saidBefore)); len(found) > 0 {
		t.Errorf("the same texts read as different: %v", found)
	}
}

// readmeVIN finds the values the README prints for the vin generator.
var readmeVIN = regexp.MustCompile(`^[A-HJ-NPR-Z0-9]{17}$`)

// The picture of Quick value shows first the three values the README prints,
// which is what its caption on the website says.
//
// The README block is the output under the command that asks for them, found
// by the command rather than by its place on the page, so moving the example
// moves nothing here.
func TestTheQuickPictureStartsWithTheValuesTheReadmePrints(t *testing.T) {
	readme := readmeText(t)
	const asked = "qadg quick vin --count 3 --seed " + websiteSeed
	at := strings.Index(readme, "\n"+asked+"\n")
	if at < 0 {
		t.Fatalf("the README no longer runs %q, which the caption of this picture refers to", asked)
	}
	printed := strings.Fields(readmeBlock(t, readme[at:], "output"))
	_, says := photograph(Data{}, Website("")[0])
	var shown []string
	for _, words := range says {
		if readmeVIN.MatchString(words) && len(shown) < len(printed) {
			shown = append(shown, words)
		}
	}
	if len(printed) == 0 || !slices.Equal(shown, printed) {
		t.Errorf("the README prints %v and the picture of Quick value starts with %v, so its caption on the "+
			"website is no longer true", printed, shown)
	}
}

// websiteData is what the README shows, produced by the program, and the
// recipe the README saves as shop.yaml with the policy it adds later.
//
// The test moves into the directory the file of customers is in, for the
// reason sampleWithSource gives: the screen writes the path into its command
// line, and a path of the machine running this would end up in the picture.
// The run that writes the manifest leaves its files in another directory, so
// the screen opening the file finds nothing of an earlier run beside it.
func websiteData(t *testing.T) (Data, string) {
	t.Helper()
	readme := readmeText(t)
	base := readmeBlock(t, readme, "file shop.yaml")
	policy := readmeBlock(t, readme, "append shop.yaml")
	response := readmeBlock(t, readme, "response response.json customers.corrupted.csv")

	root := t.TempDir()
	source, ran := filepath.Join(root, "yours"), filepath.Join(root, "run")
	// From the recipe as the README saves it, before the policy is added,
	// because that is the recipe the README makes its file of customers from.
	customers, rows := customersFrom(t, base, filepath.Join(root, "made"), source)
	promised := damaged(t, readme, customers, ran)
	actual := answeredFor(t, response, rows, filepath.Join(root, "response.json"))

	t.Chdir(source)
	return Data{
		Promised: promised, Actual: actual, HasActual: true,
		SourcePath: filepath.Base(customers),
	}, base + policy
}

// readmeCustomersColumns are the columns of the file the README damages as the
// reader's own, with the card of a payment beside them.
//
// 🔴 The same file TestTheReadmeShowsWhatTheProgramDoes in internal/cli makes,
// written out again because that helper lives in another package's tests. A
// drift between the two would leave these pictures true to the program and out
// of step with the README, and damaged below says so by comparing the counts
// of the run with the ones the README prints.
var readmeCustomersColumns = []string{"first_name", "last_name", "birth_date", "city", "zip"}

// customersFrom generates the dataset of the recipe into one directory and
// writes, into another, the file of customers the README damages: its columns
// of the customers, and the card of the payment in the same row. It returns
// that file and how many records it holds.
func customersFrom(t *testing.T, recipe, made, into string) (string, int) {
	t.Helper()
	decoded := recipeyaml.Decode([]byte(recipe))
	if len(decoded.Problems) > 0 {
		t.Fatalf("the recipe the README saves as shop.yaml has problems: %v", decoded.Problems)
	}
	built := decoded.Recipe
	prepared, err := produce.Prepare(built, values.New(built.Locale, built.Seed), generate.NeverStops{})
	if err != nil {
		t.Fatalf("the recipe of the README cannot be prepared: %v", err)
	}
	if _, err := prepared.Write(produce.Settings{Into: made}, generate.NeverStops{}, websiteRun); err != nil {
		t.Fatalf("the dataset of the README cannot be written: %v", err)
	}
	people, cards := readCSV(t, filepath.Join(made, "customer.csv")), readCSV(t, filepath.Join(made, "payment.csv"))
	rows := min(len(people), len(cards)) - 1
	var out bytes.Buffer
	writer := csv.NewWriter(&out)
	_ = writer.Write(append(slices.Clone(readmeCustomersColumns), "card"))
	for row := 1; row <= rows; row++ {
		var record []string
		for _, column := range readmeCustomersColumns {
			record = append(record, csvCell(t, people, row, column))
		}
		_ = writer.Write(append(record, csvCell(t, cards, row, "card")))
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(into, 0o755); err != nil {
		t.Fatal(err)
	}
	named := filepath.Join(into, "customers.csv")
	if err := os.WriteFile(named, out.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return named, rows
}

// readCSV is a file the run wrote, header first.
func readCSV(t *testing.T, path string) [][]string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("the run of the README recipe wrote no %s: %v", filepath.Base(path), err)
	}
	defer func() { _ = file.Close() }()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil || len(rows) < 2 {
		t.Fatalf("%s cannot be read as a table with records: %v", filepath.Base(path), err)
	}
	return rows
}

// csvCell is the value of the named column in the given row.
func csvCell(t *testing.T, rows [][]string, row int, column string) string {
	t.Helper()
	at := slices.Index(rows[0], column)
	if at < 0 {
		t.Fatalf("the run of the README recipe wrote no column %s", column)
	}
	return rows[row][at]
}

// readmeCounts finds the line in which the README says how many defects its
// run over a file of your own ordered and applied.
var readmeCounts = regexp.MustCompile(`(\d+) defects ordered, (\d+) applied`)

// damaged runs the damage the README orders for a file of your own over the
// file given and returns the manifest the run wrote. The counts of the run are
// held to the ones the README prints, so a picture of this manifest is a picture
// of the run the README describes.
func damaged(t *testing.T, readme, path, into string) manifest.Manifest {
	t.Helper()
	settings := take.Defaults()
	settings.Layers = websiteLayers()
	settings.Seed = websiteSeed
	settings.Out = into
	opened, err := take.Open(path, settings, corrupt.NeverStops{})
	if err != nil {
		t.Fatalf("the file of customers cannot be opened for damage: %v", err)
	}
	defer func() { _ = opened.Close() }()
	result, err := opened.Corrupt(settings, describedDamage(), corrupt.NeverStops{}, websiteRun)
	if err != nil {
		t.Fatalf("the file of customers cannot be damaged: %v", err)
	}
	printed := readmeCounts.FindStringSubmatch(readme)
	if printed == nil {
		t.Fatal("the README no longer says how many defects its run over a file of your own ordered and applied")
	}
	ran := []string{strconv.Itoa(result.Counts.Requested), strconv.Itoa(result.Counts.Applied)}
	if !slices.Equal(printed[1:], ran) {
		t.Fatalf("the README prints %s defects ordered and %s applied, and the same run here gives %s and %s, "+
			"so the file made here is not the one the README damages", printed[1], printed[2], ran[0], ran[1])
	}
	promised, err := manifestjson.Load(filepath.Join(into, manifestjson.Name))
	if err != nil {
		t.Fatalf("the manifest of the run cannot be read back: %v", err)
	}
	return promised
}

// answeredFor is the record of what an application did with every row of the
// file, the way the README asks for one: the entries it prints, and every other
// row taken, which is what TestTheReadmeShowsWhatTheProgramDoes writes too.
func answeredFor(t *testing.T, printed string, rows int, path string) response.Record {
	t.Helper()
	var shown struct {
		SpecVersion string           `json:"spec_version"`
		Rows        []map[string]any `json:"rows"`
	}
	if err := json.Unmarshal([]byte(printed), &shown); err != nil {
		t.Fatalf("the record the README prints is not a response record: %v", err)
	}
	byRow := map[int]map[string]any{}
	for _, entry := range shown.Rows {
		number, numbered := entry["row"].(float64)
		if !numbered || int(number) < 1 || int(number) > rows {
			t.Fatalf("the README answers about %v, which is no row of a file of %d", entry["row"], rows)
		}
		byRow[int(number)] = entry
	}
	every := make([]map[string]any, 0, rows)
	for number := 1; number <= rows; number++ {
		entry, given := byRow[number]
		if !given {
			entry = map[string]any{"row": number, "verdict": "accepted"}
		}
		every = append(every, entry)
	}
	data, err := json.Marshal(map[string]any{"spec_version": shown.SpecVersion, "rows": every})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	actual, err := responsejson.Load(path)
	if err != nil {
		t.Fatalf("the record made from the README cannot be read: %v", err)
	}
	return actual
}

// photograph draws one state at the size the stored trees are taken at and
// twice its density, and lists every text it shows.
func photograph(data Data, state State) (image.Image, []string) {
	if os.Getenv(staleClockAsked) == "" {
		setToolkitClock()
	}
	window := test.NewWindow(Assemble(i18n.English, data, state))
	defer window.Close()
	window.Canvas().(software.WindowlessCanvas).SetScale(websiteScale)
	window.Resize(size)
	Walk(window.Canvas(), state)
	// Laid out a second time at another size first, for the reason render
	// gives: a wrapping label sizes itself for the width it knew on the first
	// pass.
	window.Resize(fyne.NewSize(size.Width-1, size.Height))
	window.Resize(size)
	return window.Canvas().Capture(), textsOf(test.RenderToMarkup(window.Canvas()))
}

// setToolkitClock moves the clock the toolkit dates what it keeps by to now.
//
// # 🔴 Why a picture needs it
//
// The toolkit keeps the renderer of a widget for a minute after it was last
// used and throws it away when it cleans its caches. It dates "used" by a clock
// it reads only when it cleans, which a real window does every frame. A window
// with no screen cleans only in Capture. So a test process that runs for more
// than a minute before its first picture builds every widget of that picture
// with a date already a minute old, and Capture, which cleans before it paints,
// throws all of them away and paints new ones nobody laid out. A wrapping label
// that was never laid out has no width and draws its sentence on one line.
// Read on 2026-11-17 in fyne v2.8.2, internal/cache/base.go,
// driver/software/canvas.go:91 and widget/label.go:99. It first showed under
// -race on a slower runner, where this test began 97 seconds after the package.
//
// Capturing a blank surface is how a test can make the toolkit read its clock,
// so every widget the picture then builds is dated now and stays for the minute
// the picture takes a few seconds of.
func setToolkitClock() {
	software.NewCanvas().Capture()
}

// staleClockAsked is set in the environment of a run of this test binary that
// takes its pictures without setting the toolkit's clock first, which is how the
// check below shows that its run would notice the fault.
const staleClockAsked = "QADG_PICTURE_WITH_A_STALE_CLOCK"

// behindTheWindow opens the finding of a picture whose texts the window no
// longer shows.
const behindTheWindow = "the picture on the website no longer shows what the window shows"

// 🔴 A picture of the window is the same however long the package ran before it
// was taken.
//
// The fault this holds shut is described at setToolkitClock. Waiting for it would
// take a minute of tests before the picture, so the test binary runs the first
// picture again with the toolkit told to keep what it builds for a millisecond,
// which it reads from FYNE_CACHE when it starts. Twice: as the pictures are
// taken, which must pass, and without the clock set first, which must fail the
// way it failed on the runner. Without the second run the first would pass
// whatever the pictures do, for example once the toolkit stops reading
// FYNE_CACHE.
//
// The first run passes with a millisecond because the toolkit cleans at most
// once in ten seconds (internal/cache/base.go, Clean), and from setting the
// clock to the capture the first picture took under four of them with the race
// detector in a container on 2026-11-17. A picture that one day takes ten
// seconds turns this red here before it turns the pictures red elsewhere.
func TestAPictureIsTheSameHoweverLongThePackageRanBeforeIt(t *testing.T) {
	if *update {
		t.Skip("drawing the pictures again is not a check of them")
	}
	if output, err := pictureAgain(false); err != nil {
		t.Errorf("a picture taken a millisecond after the toolkit started keeping what it builds no "+
			"longer shows the window:\n%s", output)
	}
	if output, err := pictureAgain(true); err == nil || !strings.Contains(output, behindTheWindow) {
		t.Errorf("a picture taken without setting the toolkit's clock first did not fail the way it "+
			"failed on the runner, so the run above would pass whatever the pictures do:\n%s", output)
	}
}

// pictureAgain takes the first picture of the website in a new run of this test
// binary, whose toolkit keeps what it builds for a millisecond, with or without
// the clock set first.
func pictureAgain(stale bool) (string, error) {
	asked := ""
	if stale {
		asked = "yes"
	}
	again := exec.Command(os.Args[0], "-test.count=1",
		"-test.run=^TestTheWebsiteShowsTheWindowAsItIs$/^"+Website("")[0].Name+"$")
	again.Env = append(os.Environ(), "FYNE_CACHE=1ms", staleClockAsked+"="+asked)
	output, err := again.CombinedOutput()
	return string(output), err
}

// textOpens finds every text element of a stored tree and what it says.
var textOpens = regexp.MustCompile(`<text[^>]*>([^<]*)</text>`)

// textsOf lists what each text element of a tree says, in the order the tree
// has them, which is the order they are drawn.
func textsOf(tree string) []string {
	var says []string
	for _, found := range textOpens.FindAllStringSubmatch(tree, -1) {
		if words := html.UnescapeString(found[1]); strings.TrimSpace(words) != "" {
			says = append(says, words)
		}
	}
	return says
}

// readmeText is the README of this repository.
func readmeText(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(readmeFile)
	if err != nil {
		t.Fatalf("the README the pictures are drawn from cannot be read: %v", err)
	}
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}

// readmeBlock is the fenced block under the readme-check marker given, the
// marker TestTheReadmeShowsWhatTheProgramDoes in internal/cli acts on, so the
// block this reads is one that test has run.
func readmeBlock(t *testing.T, readme, marker string) string {
	t.Helper()
	lines := strings.Split(readme, "\n")
	at := slices.Index(lines, "<!-- readme-check: "+marker+" -->")
	if at < 0 {
		t.Fatalf("the README has no block marked %q, so the pictures have nothing to be drawn from", marker)
	}
	var block []string
	opened := false
	for _, line := range lines[at+1:] {
		fence := strings.HasPrefix(line, "```")
		switch {
		case fence && opened:
			return strings.Join(block, "\n") + "\n"
		case fence:
			opened = true
		case opened:
			block = append(block, line)
		}
	}
	t.Fatalf("the block marked %q in the README never closes", marker)
	return ""
}

// stale lists what one list of texts has and the other lacks. Texts in another
// order are a difference too, because the order is the order they are drawn.
func stale(was, now []string) []string {
	var found []string
	for _, line := range now {
		if !slices.Contains(was, line) {
			found = append(found, "the window now shows    "+line)
		}
	}
	for _, line := range was {
		if !slices.Contains(now, line) {
			found = append(found, "the picture shows       "+line)
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

// picturesDrawnHere says whether this is the system the pictures are drawn and
// compared on.
func picturesDrawnHere() bool { return runtime.GOOS == "windows" && runtime.GOARCH == "amd64" }

// writeWebsitePicture writes a picture and the list of its texts.
//
// 🔴 Refused on macOS and off amd64 rather than allowed to write a picture the
// comparing machine would call different: macOS draws no scroll bar and arm64
// rounds differently.
func writeWebsitePicture(t *testing.T, stored string, picture image.Image, says []string) {
	t.Helper()
	if runtime.GOOS == "darwin" || runtime.GOARCH != "amd64" {
		t.Fatalf("the pictures of the website cannot be drawn on %s/%s, because they would differ from "+
			"what windows/amd64 compares them with. Draw them on Windows or Linux on amd64",
			runtime.GOOS, runtime.GOARCH)
	}
	var encoded bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(&encoded, picture); err != nil {
		t.Fatalf("encoding the picture: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(stored), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stored+".png", encoded.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	record := recordHead + strings.Join(says, "\n") + "\n"
	if err := os.WriteFile(stored+".txt", []byte(record), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s.png and its list of texts. Look at it before committing it", stored)
}

// readSays is the list of texts stored beside a picture.
func readSays(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("nothing says what the picture shows, so a picture behind the window would pass: %v\n"+
			"Draw it with\n  %s", err, redrawWebsite)
	}
	var says []string
	for _, line := range strings.Split(strings.TrimRight(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n"), "\n") {
		if !strings.HasPrefix(line, "# ") {
			says = append(says, line)
		}
	}
	if len(says) == 0 {
		t.Fatalf("%s lists no texts, so this would pass whatever the picture shows", path)
	}
	return says
}

// readPicture is a picture the website carries.
func readPicture(t *testing.T, path string) image.Image {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the picture is missing: %v\nDraw it with\n  %s", err, redrawWebsite)
	}
	picture, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("%s is not a PNG: %v", path, err)
	}
	return picture
}
