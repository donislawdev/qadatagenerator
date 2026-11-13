package guard

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/donislawdev/qadatagenerator/internal/license"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

// productName is what both executables call the product in their properties.
// It is the window title too, which lives in a translation file, and the two
// are allowed to part: a title may one day say more than the name.
const productName = "QA Data Generator"

// exeResources are the compiled Windows resources of the two executables.
var exeResources = []struct{ file, resource string }{
	{"qadg.exe", "cmd/qadg/rsrc_windows_amd64.syso"},
	{"qadg-gui.exe", "cmd/qadg-gui/rsrc_windows_amd64.syso"},
}

// What this defends. Both executables say who made them and which version they
// are, in the Details tab of their properties, and the version they say is the
// version they are.
//
// The resource scripts carry a COPY of version.Tool and of the author, because
// a resource script cannot ask a Go constant. A copy drifts, and the day it
// does the program reports one version in its manifests and another in its
// file properties. So this reads the compiled resource rather than the script:
// what ships is what is asked, which also catches a .syso nobody rebuilt after
// the script changed.
//
// When version.Tool changes this fails until both scripts say the same and
// both .syso files are compiled again. The command is in each script.
func TestBothExecutablesSayWhoMadeThemAndWhichVersion(t *testing.T) {
	root, err := ModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	for _, exe := range exeResources {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(exe.resource)))
		if err != nil {
			t.Errorf("no compiled resource for %s: %v", exe.file, err)
			continue
		}
		for _, finding := range exePropertiesFindings(raw, exe.file) {
			t.Errorf("%s: %s.\nIf its .rc script says otherwise, the .syso was not compiled again after the script "+
				"changed. The windres command is in the script.", exe.file, finding)
		}
	}
}

// The check above can fail, on the case it exists for: a resource still
// carrying the version before the last one.
func TestAnExecutableWithAnOlderVersionIsNoticed(t *testing.T) {
	root, err := ModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(exeResources[0].resource)))
	if err != nil {
		t.Fatal(err)
	}
	if found := exePropertiesFindings(raw, exeResources[0].file); len(found) > 0 {
		t.Fatalf("the real resource already fails: %v", found)
	}

	// The version string replaced by another of the same length, and the
	// numbers by others, which is what a script left behind looks like.
	older := bytes.ReplaceAll(raw, utf16Bytes(version.Tool), utf16Bytes(strings.Repeat("9", len(version.Tool))))
	if len(exePropertiesFindings(older, exeResources[0].file)) == 0 {
		t.Error("a resource carrying another version string passes")
	}
	numbers := bytes.Clone(raw)
	at := bytes.Index(numbers, fixedInfoSignature)
	if at < 0 {
		t.Fatal("the resource has no fixed file information")
	}
	binary.LittleEndian.PutUint32(numbers[at+8:], 0xFFFF_FFFF)
	if len(exePropertiesFindings(numbers, exeResources[0].file)) == 0 {
		t.Error("a resource carrying another FILEVERSION number passes")
	}
}

// fixedInfoSignature opens the binary block of version numbers that Windows
// reads before any of the strings.
var fixedInfoSignature = []byte{0xBD, 0x04, 0xEF, 0xFE}

// exePropertiesFindings says what a compiled resource gets wrong about the
// executable it is compiled into, or nothing.
func exePropertiesFindings(raw []byte, file string) []string {
	var findings []string
	// A resource holds its strings as UTF-16, so a search for the ASCII text
	// finds nothing and would pass an empty resource. Searched as Windows
	// stores them.
	for what, want := range map[string]string{
		"the version of this build (version.Tool)": version.Tool,
		"the author (license.Author)":              license.Author,
		"the licence of the code (license.Code)":   license.Code,
		"the product name":                         productName,
		"its own file name":                        file,
	} {
		if !bytes.Contains(raw, utf16Bytes(want)) {
			findings = append(findings, fmt.Sprintf("it does not carry %s, %q", what, want))
		}
	}

	at := bytes.Index(raw, fixedInfoSignature)
	if at < 0 || at+24 > len(raw) {
		return append(findings, "it has no fixed file information, so FILEVERSION is missing")
	}
	want, err := versionNumbers(version.Tool)
	if err != nil {
		return append(findings, err.Error())
	}
	// After the signature and the structure version: the file version in two
	// halves, then the product version in two halves.
	for _, half := range []struct {
		name   string
		offset int
	}{{"FILEVERSION", 8}, {"PRODUCTVERSION", 16}} {
		high := binary.LittleEndian.Uint32(raw[at+half.offset:])
		low := binary.LittleEndian.Uint32(raw[at+half.offset+4:])
		got := [4]uint32{high >> 16, high & 0xFFFF, low >> 16, low & 0xFFFF}
		if got != want {
			findings = append(findings, fmt.Sprintf("its %s is %d,%d,%d,%d and version.Tool %q makes it %d,%d,%d,%d",
				half.name, got[0], got[1], got[2], got[3], version.Tool, want[0], want[1], want[2], want[3]))
		}
	}
	return findings
}

// versionNumbers turns a version such as 0.1.0, or 0.2.0-rc.1, into the four
// numbers of FILEVERSION. The fourth is always zero.
func versionNumbers(v string) ([4]uint32, error) {
	var out [4]uint32
	core, _, _ := strings.Cut(v, "-")
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return out, fmt.Errorf("version.Tool %q is not three numbers", v)
	}
	for i, part := range parts {
		n, err := strconv.ParseUint(part, 10, 16)
		if err != nil {
			return out, fmt.Errorf("version.Tool %q: %w", v, err)
		}
		out[i] = uint32(n)
	}
	return out, nil
}

// utf16Bytes is a string the way a Windows resource stores it, little endian.
func utf16Bytes(s string) []byte {
	var b bytes.Buffer
	for _, unit := range utf16.Encode([]rune(s)) {
		b.WriteByte(byte(unit))
		b.WriteByte(byte(unit >> 8))
	}
	return b.Bytes()
}

// What this defends. The window wears the program's icon rather than the
// toolkit's own logo.
//
// The call is looked for through the parser rather than in the file text, so
// that a comment mentioning it cannot stand in for it. What it cannot see is a
// running window, which needs a desktop.
func TestTheWindowIsHandedTheProgramIcon(t *testing.T) {
	root, err := ModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "cmd", "qadg-gui", "main.go")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	handed, err := handsTheIconToTheToolkit(source)
	if err != nil {
		t.Fatal(err)
	}
	if !handed {
		t.Error("nothing in cmd/qadg-gui/main.go passes appicon.SVG to SetIcon, so the window keeps the " +
			"toolkit's logo")
	}

	// And it can fail: the same file with the call turned into a comment.
	commented := bytes.Replace(source, []byte("application.SetIcon("), []byte("// application.SetIcon("), 1)
	if bytes.Equal(commented, source) {
		t.Fatal("the call is not written the way this check expects, so the check below proves nothing")
	}
	if handed, err := handsTheIconToTheToolkit(commented); err != nil || handed {
		t.Errorf("main.go with the call commented out still counts as handing the icon over (err %v)", err)
	}
}

func handsTheIconToTheToolkit(source []byte) (bool, error) {
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, "main.go", source, 0)
	if err != nil {
		return false, err
	}
	found := false
	ast.Inspect(parsed, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "SetIcon" {
			return true
		}
		for _, arg := range call.Args {
			var written bytes.Buffer
			if printer.Fprint(&written, fset, arg) == nil && strings.Contains(written.String(), "appicon.SVG") {
				found = true
			}
		}
		return true
	})
	return found, nil
}
