package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The driver is named before the first file loads, and the renderer is the
// first file asked for. Both are read off a load that fails, which is the
// only load a test can make without carrying Mesa: with nothing in the folder
// the error has to name the renderer and the variable has to be set already.
// A file that is not a library is refused in the system's words, with its
// path, rather than loaded or passed over.
func TestTheDriverIsNamedBeforeTheRendererLoads(t *testing.T) {
	t.Setenv(softwareDriver, "")
	dir := t.TempDir()
	renderer := filepath.Join(dir, softwareDir, softwareRenderer)

	err := loadSoftwareRenderer(dir)
	if err == nil || !strings.Contains(err.Error(), renderer) {
		t.Errorf("with an empty folder loading answered %v, and should name %s, the first file it loads", err, renderer)
	}
	if got := os.Getenv(softwareDriver); got != softwareLLVMPipe {
		t.Errorf("%s is %q after loading began, and Mesa reads it as it starts, so it has to be %q first",
			softwareDriver, got, softwareLLVMPipe)
	}

	if err := os.MkdirAll(filepath.Dir(renderer), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(renderer, []byte("not a library"), 0o600); err != nil {
		t.Fatal(err)
	}
	err = loadSoftwareRenderer(dir)
	if err == nil || !strings.Contains(err.Error(), renderer) {
		t.Errorf("a renderer that is not a library answered %v, and should be refused with its path", err)
	}
}
