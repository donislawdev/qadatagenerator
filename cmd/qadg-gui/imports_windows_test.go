package main

import (
	"debug/pe"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// What this defends (B-414): the window names no opengl32.dll among the
// libraries Windows loads before any code of it runs.
//
// With that import Windows maps the system's opengl32.dll at start, and then
// hands it to every later request for the same name, so the software
// renderer the window starts again with on a computer without an OpenGL
// driver could never be the one it draws with. The published OpenGL binding
// links it, and the copy in third_party/go-gl-gl does not.
//
// It reads the import table of a binary it builds, because the import comes
// from a reference to a function rather than from a flag of the linker: GLFW
// still hands the linker -lopengl32 and creates no import, since it loads the
// library by name. Only the linked binary can say which of the two it is.
//
// The canary is a program of a few lines that does call into opengl32.dll,
// built the same way. The reader has to find the library there first, so a
// reader that sees nothing cannot pass the window.
func TestTheWindowDoesNotImportOpenGLWhenItStarts(t *testing.T) {
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("this test builds the window and needs the go command, which go test puts on PATH: %v", err)
	}
	if canary := importedLibraries(t, buildOpenGLImporter(t, goTool)); !canary["opengl32.dll"] {
		t.Fatalf("a program that calls into opengl32.dll read as importing only %v, so the reader is blind", canary)
	}
	built := filepath.Join(t.TempDir(), "qadg-gui.exe")
	if output, err := exec.Command(goTool, "build", "-o", built, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, output)
	}
	if importedLibraries(t, built)["opengl32.dll"] {
		t.Errorf("qadg-gui.exe imports opengl32.dll when it starts, so the software renderer can never be loaded " +
			"in its place. go.mod has to replace github.com/go-gl/gl with third_party/go-gl-gl")
	}
}

// openGLImporter is the canary: it calls one function of opengl32.dll, which
// is what puts a library into the import table.
const openGLImporter = `package main

// #cgo LDFLAGS: -lopengl32
// #include <windows.h>
import "C"

func main() { _ = C.wglGetProcAddress(nil) }
`

// buildOpenGLImporter builds the canary as a module of its own and returns
// the path of the executable.
func buildOpenGLImporter(t *testing.T, goTool string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range map[string]string{
		"go.mod":  "module canary\n\ngo 1.27\n",
		"main.go": openGLImporter,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	built := filepath.Join(dir, "canary.exe")
	command := exec.Command(goTool, "build", "-o", built, ".")
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("go build of the canary: %v\n%s", err, output)
	}
	return built
}

// importedLibraries are the libraries an executable imports, in lower case.
//
// They are read from the imported symbols, which name their library after a
// colon, because ImportedLibraries of debug/pe answers an empty list for the
// executables Go links: measured on 2026-11-17 with Go 1.27.4 on this window,
// none from ImportedLibraries against 244 symbols from ImportedSymbols.
func importedLibraries(t *testing.T, path string) map[string]bool {
	t.Helper()
	file, err := pe.Open(path)
	if err != nil {
		t.Fatalf("%s is not a readable executable: %v", path, err)
	}
	defer func() { _ = file.Close() }()
	symbols, err := file.ImportedSymbols()
	if err != nil {
		t.Fatalf("cannot read the imports of %s: %v", path, err)
	}
	libraries := map[string]bool{}
	for _, symbol := range symbols {
		if _, library, found := strings.Cut(symbol, ":"); found {
			libraries[strings.ToLower(library)] = true
		}
	}
	return libraries
}
