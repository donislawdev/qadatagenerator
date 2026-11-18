package appicon_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	imagedraw "image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/software"
	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/gui/appicon"
	"github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

var update = flag.Bool("update", false, "write lentils.ico and lentils.icns again from the two drawings")

const (
	iconFile  = "lentils.ico"
	smallFile = "lentils-small.svg"

	// drawnAt is the size both drawings are made at, and the only size the
	// toolkit's rasteriser draws them correctly at. See the package comment.
	drawnAt = 256

	// smallUpTo is the largest size taken from the small drawing.
	smallUpTo = 32
)

// sizes are the pictures the icon file holds, largest first.
//
// Windows picks one per place it draws an icon and scales it only when the
// size it wants is missing. 16, 24 and 32 are the title bar, the taskbar and
// File Explorer at a scale of 100 and 150 per cent, 20 and 40 are the same
// places at 125 per cent, and 256 is the largest view in File Explorer.
var sizes = []int{256, 128, 64, 48, 40, 32, 24, 20, 16}

// The two executables and the resource compiled into each, from this
// directory.
var executables = []struct{ name, resource string }{
	{"qadg.exe", filepath.Join("..", "..", "..", "cmd", "qadg", "rsrc_windows_amd64.syso")},
	{"qadg-gui.exe", filepath.Join("..", "..", "..", "cmd", "qadg-gui", "rsrc_windows_amd64.syso")},
}

func TestMain(m *testing.M) {
	test.NewApp()
	os.Exit(m.Run())
}

// What this defends. The pictures File Explorer and the taskbar show for both
// executables are the drawings in this directory, at every size Windows asks
// for, rather than an older drawing or a blank file.
//
// With -update it writes lentils.ico from the drawings instead. Without it, it
// draws them again and compares. The comparison allows a little difference,
// because the rasteriser is floating point and arm64 fuses a multiply and an
// add that amd64 does in two steps, so the same drawing need not give the same
// bytes on every machine this test runs on. A different drawing is far beyond
// that allowance, which is what the second test below makes sure of.
func TestTheIconFileIsDrawnFromTheDrawings(t *testing.T) {
	drawn := pictures(t)

	if *update {
		if err := os.WriteFile(iconFile, encodeIcon(t, drawn), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s. Compile it into both executables again with the windres commands in appicon.go", iconFile)
		return
	}

	raw, err := os.ReadFile(iconFile)
	if err != nil {
		t.Fatalf("%v\nWrite it with: go test ./internal/gui/appicon -run %s -update", err, t.Name())
	}
	entries, err := readIcon(raw)
	if err != nil {
		t.Fatal(err)
	}

	held := map[int]bool{}
	for _, entry := range entries {
		held[entry.size] = true
		want, ok := drawn[entry.size]
		if !ok {
			t.Errorf("%s holds a %d px picture that no drawing here is reduced to", iconFile, entry.size)
			continue
		}
		got, err := png.Decode(bytes.NewReader(entry.payload))
		if err != nil {
			t.Errorf("the %d px picture in %s is not a PNG: %v", entry.size, iconFile, err)
			continue
		}
		count := differing(want, got)
		t.Logf("%3d px: %d pixels differ by more than a few levels", entry.size, count)
		if allowed := tolerance(entry.size); count > allowed {
			t.Errorf("the %d px picture in %s differs from the drawing in %d pixels, more than the %d "+
				"allowed, so it is older than the drawing. Write it again with -update and rebuild both .syso "+
				"files", entry.size, iconFile, count, allowed)
		}
	}
	for _, size := range sizes {
		if !held[size] {
			t.Errorf("%s holds no %d px picture, so Windows scales one of the others to that size itself",
				iconFile, size)
		}
	}
}

// What this defends. The icon the Dock and the Finder show for the window's
// bundle on macOS is the same spoon on its plate, at every size the file holds,
// rather than an older drawing or a blank page.
//
// With -update it writes lentils.icns as well, and the release packs it into
// the bundle as it is (.github/scripts/make_app_bundle.sh). The comparison is
// the one the icon file for Windows gets, with the same allowance.
func TestTheMacIconFileIsDrawnFromTheDrawings(t *testing.T) {
	drawn := macPictures(t)

	if *update {
		if err := os.WriteFile(macIconFile, encodeMacIcon(t, drawn), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s", macIconFile)
		return
	}

	raw, err := os.ReadFile(macIconFile)
	if err != nil {
		t.Fatalf("%v\nWrite it with: go test ./internal/gui/appicon -run %s -update", err, t.Name())
	}
	chunks, err := readMacIcon(raw)
	if err != nil {
		t.Fatal(err)
	}
	held := map[string]bool{}
	for _, chunk := range chunks {
		size, known := macTypeSize(chunk.name)
		switch {
		case !known:
			t.Errorf("%s holds a %q picture, a type this icon is not drawn for", macIconFile, chunk.name)
			continue
		case held[chunk.name]:
			t.Errorf("%s holds the %q picture twice", macIconFile, chunk.name)
			continue
		}
		held[chunk.name] = true
		got, err := png.Decode(bytes.NewReader(chunk.payload))
		if err != nil {
			t.Errorf("the %q picture in %s is not a PNG: %v", chunk.name, macIconFile, err)
			continue
		}
		if got.Bounds().Dx() != size || got.Bounds().Dy() != size {
			t.Errorf("the %q picture in %s is %v, and macOS reads that type as %d px", chunk.name, macIconFile, got.Bounds().Size(), size)
			continue
		}
		count := differing(drawn[size], got)
		t.Logf("%s %3d px: %d pixels differ by more than a few levels", chunk.name, size, count)
		if allowed := tolerance(size); count > allowed {
			t.Errorf("the %q picture in %s differs from the drawing on its plate in %d pixels, more than the %d "+
				"allowed, so it is older than the drawing or the plate. Write it again with -update", chunk.name, macIconFile, count, allowed)
		}
	}
	for _, each := range macTypes {
		if !held[each.name] {
			t.Errorf("%s holds no %q picture, so macOS scales another one to %d px itself", macIconFile, each.name, each.size)
		}
	}
}

// The reader above refuses a file that is not an icon file for macOS, rather
// than reading whatever it finds as no pictures at all.
func TestAMacIconFileThatIsNotOneIsRefused(t *testing.T) {
	good := []byte("icns\x00\x00\x00\x10ic07\x00\x00\x00\x08")
	if _, err := readMacIcon(good); err != nil {
		t.Fatalf("a well formed file with one empty picture is refused: %v", err)
	}
	for name, raw := range map[string][]byte{
		"another magic":               []byte("icnx\x00\x00\x00\x08"),
		"a length past the end":       []byte("icns\x00\x00\x00\x20ic07\x00\x00\x00\x08"),
		"a chunk past the end":        []byte("icns\x00\x00\x00\x10ic07\x00\x00\x00\x40"),
		"a chunk shorter than itself": []byte("icns\x00\x00\x00\x10ic07\x00\x00\x00\x04"),
		"too short for a header":      []byte("icns"),
	} {
		if _, err := readMacIcon(raw); err == nil {
			t.Errorf("%s is read as an icon file", name)
		}
	}
}

// tolerance is how many pixels of a picture may differ by more than a few
// levels between the stored picture and a new drawing of it: one in a
// thousand, and never fewer than two.
//
// Measured on 2026-11-13 against an icon file written on Windows: Linux on
// amd64 drew every size identically, and macOS on arm64 differed in one pixel
// at 256 px and one at 128 px, and in none below. One spilled lentil, in full
// view, recoloured in the master changed 345 pixels at 256 px and failed five
// sizes. One at the back of the heap changed 3 and passed, which is a change
// nobody looking at the icon would see either.
func tolerance(size int) int {
	return max(2, size*size/1000)
}

// The comparison above is only worth something if it can fail, and it fails
// on the case it exists for: a picture made from another drawing. The two
// drawings here are the same spoon at the same size, which is as close as two
// different drawings come.
func TestTheComparisonNoticesAnotherDrawing(t *testing.T) {
	small, err := os.ReadFile(smallFile)
	if err != nil {
		t.Fatal(err)
	}
	master, little := reduce(draw(t, appicon.SVG), smallUpTo), reduce(draw(t, small), smallUpTo)
	if count := differing(master, little); count <= tolerance(smallUpTo) {
		t.Errorf("the master and the small drawing differ in only %d pixels at %d px, "+
			"so the comparison would not notice an icon file made from the wrong drawing", count, smallUpTo)
	}
}

// What this defends. Each drawing actually draws something. The rasteriser
// meets a construct it cannot read with an empty picture and no error, which
// is what happened to one of the icons inside the window in 2026, and an empty
// drawing would sail through the comparison above once it was written into
// the icon file as well.
func TestBothDrawingsDrawASpoonRatherThanNothing(t *testing.T) {
	small, err := os.ReadFile(smallFile)
	if err != nil {
		t.Fatal(err)
	}
	for name, source := range map[string][]byte{appicon.Name: appicon.SVG, smallFile: small} {
		if problem := notADrawing(draw(t, source)); problem != "" {
			t.Errorf("%s: %s", name, problem)
		}
	}
	empty := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" width="256" height="256"></svg>`)
	if notADrawing(draw(t, empty)) == "" {
		t.Error("an empty drawing passes as a spoon, so this check proves nothing")
	}
}

// notADrawing says what is wrong with a picture that should show the spoon,
// or nothing when it shows something of the right weight with clear corners.
func notADrawing(picture image.Image) string {
	b := picture.Bounds()
	opaque := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := picture.At(x, y).RGBA(); a >= 0x8000 {
				opaque++
			}
		}
	}
	// The spoon covers about a third of its square. Nothing, or nearly all of
	// it, is a drawing the rasteriser could not read or a plate behind it.
	if share := float64(opaque) / float64(b.Dx()*b.Dy()); share < 0.15 || share > 0.85 {
		return fmt.Sprintf("%.0f%% of the picture is solid, which is not a spoon on a clear ground", share*100)
	}
	for _, corner := range []image.Point{
		{X: b.Min.X, Y: b.Min.Y}, {X: b.Max.X - 1, Y: b.Min.Y},
		{X: b.Min.X, Y: b.Max.Y - 1}, {X: b.Max.X - 1, Y: b.Max.Y - 1},
	} {
		if _, _, _, a := picture.At(corner.X, corner.Y).RGBA(); a != 0 {
			return fmt.Sprintf("the corner at %d,%d is not clear, so the icon sits on a plate", corner.X, corner.Y)
		}
	}
	return ""
}

// What this defends. Both executables carry the pictures in lentils.ico, so
// File Explorer shows the same spoon for both, and the one the window wears.
//
// A .syso is compiled from the icon file by hand, so it can be older than the
// icon file without anything else noticing. It is checked by bytes: windres
// copies every picture of an icon file into the resource unchanged, so each
// one must appear inside it whole.
func TestBothExecutablesCarryTheIconFile(t *testing.T) {
	raw, err := os.ReadFile(iconFile)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := readIcon(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, exe := range executables {
		resource, err := os.ReadFile(exe.resource)
		if err != nil {
			t.Errorf("no resource for %s: %v\nBuild it with the windres command in appicon.go", exe.name, err)
			continue
		}
		if missing := missingFrom(entries, resource); len(missing) > 0 {
			t.Errorf("the resource compiled into %s lacks the %v px pictures of %s, so it is older than the "+
				"icon file. Rebuild it with the windres command in appicon.go", exe.name, missing, iconFile)
		}
	}
}

// The check above can fail, on the case it exists for: a resource built from
// an icon file that has since changed.
func TestAnOlderResourceIsNoticed(t *testing.T) {
	raw, err := os.ReadFile(iconFile)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := readIcon(raw)
	if err != nil {
		t.Fatal(err)
	}
	resource, err := os.ReadFile(executables[0].resource)
	if err != nil {
		t.Fatal(err)
	}
	older := bytes.Clone(resource)
	at := bytes.Index(older, entries[0].payload)
	if at < 0 {
		t.Fatalf("the resource does not hold the %d px picture in the first place", entries[0].size)
	}
	// One byte in the middle of the largest picture, which is what a redrawn
	// icon looks like to a resource that was not rebuilt.
	older[at+len(entries[0].payload)/2] ^= 0xFF
	if missing := missingFrom(entries, older); len(missing) != 1 || missing[0] != entries[0].size {
		t.Errorf("a resource with one picture changed reports %v missing, want [%d]", missing, entries[0].size)
	}
}

// pictures draws every size the icon file holds, each from the drawing meant
// for it.
func pictures(t *testing.T) map[int]image.Image {
	t.Helper()
	small, err := os.ReadFile(smallFile)
	if err != nil {
		t.Fatal(err)
	}
	master, little := draw(t, appicon.SVG), draw(t, small)
	drawn := make(map[int]image.Image, len(sizes))
	for _, size := range sizes {
		from := master
		if size <= smallUpTo {
			from = little
		}
		drawn[size] = reduce(from, size)
	}
	return drawn
}

// draw is the toolkit's own rasteriser at the size the drawing is made at,
// which is how the window draws its icon.
func draw(t *testing.T, source []byte) image.Image {
	t.Helper()
	surface := software.NewTransparentCanvas()
	surface.SetPadded(false)
	surface.SetScale(1)
	picture := canvas.NewImageFromResource(fyne.NewStaticResource(appicon.Name, source))
	picture.FillMode = canvas.ImageFillContain
	surface.SetContent(picture)
	surface.Resize(fyne.NewSize(drawnAt, drawnAt))
	return surface.Capture()
}

// reduce averages the pixels: each pixel of the result is the weighted mean
// of the area it covers in the source, taken on colour multiplied by opacity
// so that a clear ground does not darken the edges.
func reduce(source image.Image, size int) image.Image {
	b := source.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, size, size))
	scale := float64(b.Dx()) / float64(size)
	for dy := 0; dy < size; dy++ {
		y0, y1 := float64(dy)*scale, float64(dy+1)*scale
		for dx := 0; dx < size; dx++ {
			x0, x1 := float64(dx)*scale, float64(dx+1)*scale
			var r, g, bl, a, area float64
			for sy := int(y0); float64(sy) < y1; sy++ {
				wy := min(y1, float64(sy+1)) - max(y0, float64(sy))
				for sx := int(x0); float64(sx) < x1; sx++ {
					w := wy * (min(x1, float64(sx+1)) - max(x0, float64(sx)))
					cr, cg, cb, ca := source.At(b.Min.X+sx, b.Min.Y+sy).RGBA()
					r += float64(cr) * w
					g += float64(cg) * w
					bl += float64(cb) * w
					a += float64(ca) * w
					area += w
				}
			}
			if a == 0 {
				continue
			}
			out.SetNRGBA(dx, dy, color.NRGBA{
				R: uint8(r/a*0xFF + 0.5),
				G: uint8(g/a*0xFF + 0.5),
				B: uint8(bl/a*0xFF + 0.5),
				A: uint8(a/area/0xFFFF*0xFF + 0.5),
			})
		}
	}
	return out
}

// differing counts the pixels in which any channel differs by more than a few
// levels. The colour of a clear pixel is not compared, because it is never
// seen. Pictures of different sizes differ everywhere.
func differing(want, got image.Image) int {
	if want.Bounds().Size() != got.Bounds().Size() {
		return want.Bounds().Dx() * want.Bounds().Dy()
	}
	const levels = 8
	far := func(p, q uint8) bool { return max(p, q)-min(p, q) > levels }
	wb, gb := want.Bounds(), got.Bounds()
	count := 0
	for y := 0; y < wb.Dy(); y++ {
		for x := 0; x < wb.Dx(); x++ {
			p := color.NRGBAModel.Convert(want.At(wb.Min.X+x, wb.Min.Y+y)).(color.NRGBA)
			q := color.NRGBAModel.Convert(got.At(gb.Min.X+x, gb.Min.Y+y)).(color.NRGBA)
			if far(p.A, q.A) || (p.A > 0 && q.A > 0 && (far(p.R, q.R) || far(p.G, q.G) || far(p.B, q.B))) {
				count++
			}
		}
	}
	return count
}

type entry struct {
	size    int
	payload []byte
}

// encodeIcon writes an icon file: a directory of the pictures, then each
// picture as a PNG.
//
// PNG at every size rather than the older bitmap form below 256. Windows has
// read PNG pictures in an icon file since Vista, and the toolchain this
// project builds with runs on nothing older than Windows 10.
func encodeIcon(t *testing.T, drawn map[int]image.Image) []byte {
	t.Helper()
	const header, directoryEntry = 6, 16
	payloads := make([][]byte, len(sizes))
	for i, size := range sizes {
		var b bytes.Buffer
		if err := png.Encode(&b, drawn[size]); err != nil {
			t.Fatal(err)
		}
		payloads[i] = b.Bytes()
	}

	var out bytes.Buffer
	put := func(v any) {
		if err := binary.Write(&out, binary.LittleEndian, v); err != nil {
			t.Fatal(err)
		}
	}
	put(uint16(0)) // reserved
	put(uint16(1)) // an icon, not a cursor
	put(uint16(len(sizes)))
	offset := header + directoryEntry*len(sizes)
	for i, size := range sizes {
		// A byte holds the side, and 256 is written as 0.
		side := uint8(size % 256)
		put([]uint8{side, side, 0, 0})
		put(uint16(1))  // colour planes
		put(uint16(32)) // bits per pixel
		put(uint32(len(payloads[i])))
		put(uint32(offset))
		offset += len(payloads[i])
	}
	for _, payload := range payloads {
		out.Write(payload)
	}
	return out.Bytes()
}

// readIcon reads the directory of an icon file and the bytes of each picture.
func readIcon(raw []byte) ([]entry, error) {
	const header, directoryEntry = 6, 16
	if len(raw) < header || binary.LittleEndian.Uint16(raw[2:]) != 1 {
		return nil, errors.New("not an icon file: the header does not say icon")
	}
	count := int(binary.LittleEndian.Uint16(raw[4:]))
	entries := make([]entry, 0, count)
	for i := 0; i < count; i++ {
		at := header + i*directoryEntry
		if at+directoryEntry > len(raw) {
			return nil, fmt.Errorf("the icon file claims %d pictures and is too short for them", count)
		}
		size := int(raw[at])
		if size == 0 {
			size = 256
		}
		length := int(binary.LittleEndian.Uint32(raw[at+8:]))
		offset := int(binary.LittleEndian.Uint32(raw[at+12:]))
		if offset < 0 || length < 0 || offset+length > len(raw) {
			return nil, fmt.Errorf("the %d px picture points past the end of the icon file", size)
		}
		entries = append(entries, entry{size: size, payload: raw[offset : offset+length]})
	}
	return entries, nil
}

// macIconFile is the icon the window's bundle carries on macOS.
const macIconFile = "lentils.icns"

// The plate the icon stands on in the Dock, where every icon is a rounded
// square of one size and a shape on a clear ground reads as smaller and
// foreign. The owner chose it on 2026-11-18, after the icon of Testing Files
// Generator, and its geometry is that icon's: a plate the size of the picture,
// a corner of 22 per cent of the side, the spoon at 82 per cent of the side in
// the middle. Apple's page on app icons, read the same day, gives no measures
// for an icon of this kind. The colour is the ground the palette sets apart
// from the window. Windows and Linux keep the spoon on a clear ground.
const (
	plateShade  = "neutral.2"
	plateCorner = 0.22
	plateInset  = 0.82

	// plateSupersample is how many times larger the plate is drawn before it
	// is reduced to drawnAt, so its corners come out smooth.
	plateSupersample = 4
)

// macTypes are the pictures lentils.icns holds, by the name macOS reads each
// one under. Apple's names, as iconutil writes them: macOS asks for a size in
// points and a scale, so a 32 px picture answers both for 16 points on a Retina
// screen (ic11) and for 32 points on any other (icp5), and is in the file
// twice.
//
// Nothing above 256 px. The drawing is correct at 256 px and below and nowhere
// else (see the package comment), so a 512 px picture would be the 256 px one
// enlarged, which macOS does itself when it needs one.
var macTypes = []struct {
	name string
	size int
}{
	{"icp4", 16}, {"ic11", 32}, {"icp5", 32}, {"ic12", 64},
	{"ic07", 128}, {"ic13", 256}, {"ic08", 256},
}

func macTypeSize(name string) (int, bool) {
	for _, each := range macTypes {
		if each.name == name {
			return each.size, true
		}
	}
	return 0, false
}

// macPictures draws every size lentils.icns holds, each from the drawing meant
// for it, on the plate.
func macPictures(t *testing.T) map[int]image.Image {
	t.Helper()
	small, err := os.ReadFile(smallFile)
	if err != nil {
		t.Fatal(err)
	}
	master, little := onPlate(t, draw(t, appicon.SVG)), onPlate(t, draw(t, small))
	drawn := map[int]image.Image{}
	for _, each := range macTypes {
		from := master
		if each.size <= smallUpTo {
			from = little
		}
		drawn[each.size] = reduce(from, each.size)
	}
	return drawn
}

// onPlate puts a drawing made at drawnAt on the plate, at drawnAt. The drawing
// is reduced rather than drawn smaller, for the reason the package comment
// gives about lines below the size a drawing is made at.
func onPlate(t *testing.T, spoon image.Image) image.Image {
	t.Helper()
	shade, ok := theme.ShadeOf(plateShade)
	if !ok {
		t.Fatalf("the palette has no %s for the plate", plateShade)
	}
	side := float64(drawnAt * plateSupersample)
	radius := plateCorner * side
	big := image.NewNRGBA(image.Rect(0, 0, int(side), int(side)))
	for y := 0; y < int(side); y++ {
		for x := 0; x < int(side); x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			cx, cy := math.Min(math.Max(px, radius), side-radius), math.Min(math.Max(py, radius), side-radius)
			if (px-cx)*(px-cx)+(py-cy)*(py-cy) <= radius*radius {
				big.SetNRGBA(x, y, shade.NRGBA())
			}
		}
	}
	out := image.NewNRGBA(image.Rect(0, 0, drawnAt, drawnAt))
	imagedraw.Draw(out, out.Bounds(), reduce(big, drawnAt), image.Point{}, imagedraw.Src)
	inner := int(math.Round(plateInset * drawnAt))
	at := (drawnAt - inner) / 2
	imagedraw.Draw(out, image.Rect(at, at, at+inner, at+inner), reduce(spoon, inner), image.Point{}, imagedraw.Over)
	return out
}

type macChunk struct {
	name    string
	payload []byte
}

// encodeMacIcon writes an icon file for macOS: the four letters icns and the
// length of the whole file, then each picture as its type, its length with the
// eight bytes of its own header counted in, and a PNG, the form iconutil
// writes. No table of contents: it is optional, and a summary of the file kept
// inside the file is a second copy of it.
func encodeMacIcon(t *testing.T, drawn map[int]image.Image) []byte {
	t.Helper()
	var body bytes.Buffer
	for _, each := range macTypes {
		var picture bytes.Buffer
		if err := png.Encode(&picture, drawn[each.size]); err != nil {
			t.Fatal(err)
		}
		body.WriteString(each.name)
		if err := binary.Write(&body, binary.BigEndian, uint32(picture.Len()+8)); err != nil {
			t.Fatal(err)
		}
		body.Write(picture.Bytes())
	}
	out := bytes.NewBufferString("icns")
	if err := binary.Write(out, binary.BigEndian, uint32(body.Len()+8)); err != nil {
		t.Fatal(err)
	}
	out.Write(body.Bytes())
	return out.Bytes()
}

// readMacIcon reads the pictures of an icon file for macOS.
func readMacIcon(raw []byte) ([]macChunk, error) {
	const header = 8
	if len(raw) < header || string(raw[:4]) != "icns" {
		return nil, errors.New("not an icon file for macOS: it does not start with icns")
	}
	if total := int(binary.BigEndian.Uint32(raw[4:])); total != len(raw) {
		return nil, fmt.Errorf("the file says it is %d bytes and is %d", total, len(raw))
	}
	var chunks []macChunk
	for at := header; at < len(raw); {
		if at+header > len(raw) {
			return nil, fmt.Errorf("a picture at byte %d is cut off before its header ends", at)
		}
		length := int(binary.BigEndian.Uint32(raw[at+4:]))
		if length < header || at+length > len(raw) {
			return nil, fmt.Errorf("the picture %q at byte %d says it is %d bytes long", raw[at:at+4], at, length)
		}
		chunks = append(chunks, macChunk{name: string(raw[at : at+4]), payload: raw[at+header : at+length]})
		at += length
	}
	return chunks, nil
}

// missingFrom lists, smallest first, the sizes whose picture is not inside a
// compiled resource.
func missingFrom(entries []entry, resource []byte) []int {
	var missing []int
	for _, e := range entries {
		if !bytes.Contains(resource, e.payload) {
			missing = append(missing, e.size)
		}
	}
	sort.Ints(missing)
	return missing
}
