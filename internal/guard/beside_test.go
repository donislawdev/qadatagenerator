package guard

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The rule applied to the repository: the go command compiles nothing of ours
// beside the Go files but the .syso of each binary, and those carry nothing
// but Windows resources. The scan has to list both, or the rule would pass by
// reading nothing.
func TestTheProductCompilesNothingBesideGoButResources(t *testing.T) {
	root, err := ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}
	files, err := CompiledBesideGo(root)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	for _, linked := range []string{"cmd/qadg/rsrc_windows_amd64.syso", "cmd/qadg-gui/rsrc_windows_amd64.syso"} {
		if !slices.Contains(files, linked) {
			t.Fatalf("the scan did not list %s, which the go command links into its binary, so it is not "+
				"reaching the product: %v", linked, files)
		}
	}
	if findings := BesideGo(root, files); len(findings) != 0 {
		t.Fatalf("something beside the Go files goes into a binary unread:\n%s\n\n"+
			"The guards read Go, so a file the go command compiles beside it is code nothing checks. "+
			"Write it in Go, or carry it outside our directories as third_party is carried.",
			strings.Join(findings, "\n"))
	}
}

// What the go command compiles is listed, for any system, and what it skips is
// not. Of the .syso files only the one carrying nothing but resources passes.
func TestTheScanBesideGoSeesWhatTheGoCommandCompiles(t *testing.T) {
	const (
		resources = pe.IMAGE_SCN_CNT_INITIALIZED_DATA | pe.IMAGE_SCN_MEM_READ
		code      = pe.IMAGE_SCN_CNT_CODE | pe.IMAGE_SCN_MEM_EXECUTE | pe.IMAGE_SCN_MEM_READ
	)
	rsrc := []pe.SectionHeader32{section(".rsrc", resources)}
	dir := t.TempDir()
	const assembly = "TEXT call(SB),$0\n"
	planted := map[string][]byte{
		"pkg/call.s":               []byte(assembly),
		"pkg/call_linux_arm64.s":   []byte(assembly),
		"pkg/helper.c":             []byte("/* a C file */\n"),
		"pkg/helper.h":             []byte("/* a C header */\n"),
		"pkg/resources.syso":       coff(t, rsrc, []pe.COFFSymbol{symbol(".rsrc", 1)}),
		"pkg/split.syso":           coff(t, []pe.SectionHeader32{section(".rsrc$01", resources), section(".rsrc$02", resources)}, nil),
		"pkg/code.syso":            coff(t, []pe.SectionHeader32{section(".text", code)}, nil),
		"pkg/hidden.syso":          coff(t, []pe.SectionHeader32{section(".rsrc", resources|pe.IMAGE_SCN_MEM_EXECUTE)}, nil),
		"pkg/borrowed.syso":        coff(t, rsrc, []pe.COFFSymbol{symbol("socket", 0)}),
		"pkg/empty.syso":           coff(t, nil, nil),
		"pkg/text.syso":            []byte("not an object\n"),
		"pkg/_left_out.s":          []byte(assembly),
		"pkg/notes.txt":            []byte("data, not code\n"),
		"pkg/code.go":              []byte("package pkg\n"),
		"pkg/testdata/inside.s":    []byte(assembly),
		"third_party/carried/x.c":  []byte("/* carried */\n"),
		"pkg/call_windows_amd64.S": []byte(assembly),
	}
	for name, content := range planted {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := CompiledBesideGo(dir)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	listed := []string{
		"pkg/borrowed.syso", "pkg/call.s", "pkg/call_linux_arm64.s", "pkg/call_windows_amd64.S", "pkg/code.syso",
		"pkg/empty.syso", "pkg/helper.c", "pkg/helper.h", "pkg/hidden.syso", "pkg/resources.syso",
		"pkg/split.syso", "pkg/text.syso",
	}
	if !slices.Equal(files, listed) {
		t.Fatalf("listed\n%v\nexpected\n%v", files, listed)
	}

	findings := strings.Join(BesideGo(dir, files), "\n")
	for _, refused := range []string{
		"pkg/call.s is compiled", "pkg/call_linux_arm64.s is compiled", "pkg/call_windows_amd64.S is compiled",
		"pkg/helper.c is compiled", "pkg/helper.h is compiled",
		`pkg/code.syso carries the section ".text"`, `pkg/code.syso marks its section ".text" as code`,
		`pkg/hidden.syso marks its section ".rsrc" as code`,
		`pkg/borrowed.syso names the symbol "socket"`,
		"pkg/empty.syso carries no section",
		"pkg/text.syso cannot be read",
	} {
		if !strings.Contains(findings, refused) {
			t.Errorf("no finding saying %q in:\n%s", refused, findings)
		}
	}
	for _, passed := range []string{"pkg/resources.syso", "pkg/split.syso"} {
		if strings.Contains(findings, passed) {
			t.Errorf("%s carries nothing but resources and was refused:\n%s", passed, findings)
		}
	}
}

// coff is a COFF object as windres writes one, without the bytes of its
// sections: a file header, the section headers, the symbols and an empty
// string table.
func coff(t *testing.T, sections []pe.SectionHeader32, symbols []pe.COFFSymbol) []byte {
	t.Helper()
	header := pe.FileHeader{
		Machine:              pe.IMAGE_FILE_MACHINE_AMD64,
		NumberOfSections:     uint16(len(sections)),
		PointerToSymbolTable: uint32(binary.Size(pe.FileHeader{}) + len(sections)*binary.Size(pe.SectionHeader32{})),
		NumberOfSymbols:      uint32(len(symbols)),
	}
	var object bytes.Buffer
	for _, part := range []any{header, sections, symbols, uint32(4)} {
		if err := binary.Write(&object, binary.LittleEndian, part); err != nil {
			t.Fatalf("cannot write the object: %v", err)
		}
	}
	// debug/pe reads the first 96 bytes of any file before it asks what the
	// file is, so a smaller object is refused as cut short. What follows the
	// string table is never read.
	object.Write(make([]byte, max(0, 96-object.Len())))
	return object.Bytes()
}

func section(name string, characteristics uint32) pe.SectionHeader32 {
	header := pe.SectionHeader32{Characteristics: characteristics}
	copy(header.Name[:], name)
	return header
}

// symbol is a symbol defined in the section numbered from one, or named and
// left for the linker to find elsewhere when the number is zero.
func symbol(name string, sectionNumber int16) pe.COFFSymbol {
	const static, external = 3, 2
	one := pe.COFFSymbol{SectionNumber: sectionNumber, StorageClass: static}
	if sectionNumber == 0 {
		one.StorageClass = external
	}
	copy(one.Name[:], name)
	return one
}
