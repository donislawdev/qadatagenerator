package main

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
)

// attemptWith is a second start in which every part that touches the world
// answers as told, and which records what it was asked to start.
type attemptWith struct {
	asked      launch
	goos       string
	dirErr     error
	missing    string
	besideErr  error
	code       int
	startErr   error
	started    [][]string
	note       bytes.Buffer
	besideDirs []string
}

func (w *attemptWith) attempt() secondAttempt {
	return secondAttempt{
		asked: w.asked,
		goos:  w.goos,
		executableDir: func() (string, error) {
			return "exe-dir", w.dirErr
		},
		beside: func(dir string) (string, error) {
			w.besideDirs = append(w.besideDirs, dir)
			return w.missing, w.besideErr
		},
		start: func(args []string) (int, error) {
			w.started = append(w.started, args)
			return w.code, w.startErr
		},
		note: &w.note,
		text: i18n.English,
	}
}

// The second start is taken only on Windows, only by a first process, and only
// with the renderer's files beside the program, and it answers the second
// process's exit code. Every other case answers why it was not taken and
// starts nothing, which is what keeps a refusal from becoming a chain of
// processes.
func TestTheSecondStartIsTakenOnlyWhereItCanHelp(t *testing.T) {
	refused := errors.New("refused")
	cases := []struct {
		name  string
		world attemptWith
		want  error
	}{
		{"a second process", attemptWith{asked: launch{softwareGL: true}, goos: "windows"}, alreadySoftware{}},
		{"a system nothing comes for", attemptWith{goos: "linux"}, notShipped{"linux"}},
		{"no executable to start", attemptWith{goos: "windows", dirErr: refused}, startFailed{refused}},
		{"a file that cannot be looked at", attemptWith{goos: "windows", besideErr: refused}, notReadable{refused}},
		{"a file missing", attemptWith{goos: "windows", missing: "a path"}, notBeside{"a path"}},
		{"a second process that would not start", attemptWith{goos: "windows", startErr: refused}, startFailed{refused}},
	}
	for _, each := range cases {
		world := each.world
		_, why := world.attempt().try()
		if !errors.Is(why, each.want) {
			t.Errorf("%s answered %#v and should answer %#v", each.name, why, each.want)
		}
		if each.name != "a second process that would not start" && len(world.started) != 0 {
			t.Errorf("%s started %v, and nothing should have started", each.name, world.started)
		}
	}

	world := attemptWith{goos: "windows", code: 7, asked: launch{catalogue: true, given: []string{"-catalogue"}}}
	code, why := world.attempt().try()
	if why != nil || code != 7 {
		t.Fatalf("the second start answered %d and %v, and should answer the second process's 7", code, why)
	}
	if len(world.started) != 1 || strings.Join(world.started[0], " ") != "-catalogue -software-gl" {
		t.Errorf("the second process was started with %q, and should get what this one was given plus the flag",
			world.started)
	}
	if len(world.besideDirs) != 1 || world.besideDirs[0] != "exe-dir" {
		t.Errorf("the renderer was looked for in %q, and belongs beside the executable", world.besideDirs)
	}
	if !strings.Contains(world.note.String(), i18n.English.T(i18n.WindowSoftwareAgain)) {
		t.Errorf("nothing was said before the second start: %q", world.note.String())
	}
}

// A refusal says what became of the renderer: why it was not tried, that it
// could not be loaded, or that it was and did not help, and nothing on a
// system no renderer comes for.
func TestTheRefusalSaysWhatBecameOfTheRenderer(t *testing.T) {
	text := i18n.English
	unreadable := errors.New("Access is denied.")
	cases := []struct {
		name        string
		why, loaded error
		want        string
	}{
		{"a file missing", notBeside{`C:\x\opengl\libgallium_wgl.dll`}, nil, `C:\x\opengl\libgallium_wgl.dll is not there`},
		{"a file unreadable", notReadable{unreadable}, nil, "could not be read: Access is denied. Check"},
		{"no second process", startFailed{unreadable}, nil, "did not work: Access is denied."},
		{"a system nothing comes for", notShipped{"linux"}, nil, ""},
		{"the flag on a system nothing comes for", alreadySoftware{}, notShipped{"darwin"}, ""},
		{"a renderer that did not load", alreadySoftware{}, errors.New("Failed to load x: bad."), "could not be loaded: Failed to load x: bad."},
		{"a renderer that loaded and did not help", alreadySoftware{}, nil, text.T(i18n.WindowSoftwareDidNotHelp)},
	}
	for _, each := range cases {
		said := rendererSentence(text, each.why, each.loaded)
		if (each.want == "") != (said == "") || !strings.Contains(said, each.want) {
			t.Errorf("%s said %q and should say %q", each.name, said, each.want)
		}
		if strings.Contains(said, "..") {
			t.Errorf("%s ends the system's words with two full stops: %q", each.name, said)
		}
	}
}

// A window asked for the renderer says it draws with it, that none comes for
// this system, or why it could not be loaded, and says nothing when nobody
// asked.
func TestTheWindowSaysHowItDraws(t *testing.T) {
	text := i18n.English
	loads := 0
	var said bytes.Buffer
	if drawing := rendererFor(launch{}, text, &said, func() error { loads++; return nil }); drawing.software() || loads != 0 || said.Len() != 0 {
		t.Errorf("a window nobody asked to draw in software loaded %d times, said %q, and draws in software: %v",
			loads, said.String(), drawing.software())
	}
	cases := []struct {
		loaded   error
		want     string
		software bool
	}{
		{nil, text.T(i18n.WindowSoftwareDrawing), true},
		{notShipped{"linux"}, text.T(i18n.WindowSoftwareNotShipped), false},
		{errors.New("Failed to load x."), "could not be loaded: Failed to load x.", false},
	}
	for _, each := range cases {
		said.Reset()
		drawing := rendererFor(launch{softwareGL: true}, text, &said, func() error { return each.loaded })
		if !strings.Contains(said.String(), each.want) || drawing.software() != each.software {
			t.Errorf("loading that answered %v said %q and draws in software %v, and should say %q and %v",
				each.loaded, said.String(), drawing.software(), each.want, each.software)
		}
	}
}

// The renderer is looked for in the opengl folder beside the executable, the
// renderer before its loader, because that is the order they load in, and a
// file that cannot be looked at is told apart from one that is missing.
func TestTheRendererIsLookedForBesideTheProgram(t *testing.T) {
	dir := t.TempDir()
	files := softwareFiles(dir)
	if len(files) != 2 || filepath.Base(files[0]) != softwareRenderer || filepath.Base(files[1]) != softwareLoader ||
		filepath.Dir(files[0]) != filepath.Join(dir, softwareDir) {
		t.Fatalf("the renderer's files are %q, and should be %s then %s in %s", files, softwareRenderer,
			softwareLoader, filepath.Join(dir, softwareDir))
	}
	if missing, err := rendererBeside(dir, os.Stat); missing != files[0] || err != nil {
		t.Errorf("with nothing there the first missing file is %q (%v), and should be %q", missing, err, files[0])
	}
	if err := os.MkdirAll(filepath.Dir(files[0]), 0o700); err != nil {
		t.Fatal(err)
	}
	for i, file := range files {
		if err := os.WriteFile(file, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		want := ""
		if i == 0 {
			want = files[1]
		}
		if missing, err := rendererBeside(dir, os.Stat); missing != want || err != nil {
			t.Errorf("with %d of the files there the missing one is %q (%v), and should be %q", i+1, missing, err, want)
		}
	}
	denied := func(string) (fs.FileInfo, error) { return nil, fs.ErrPermission }
	if missing, err := rendererBeside(dir, denied); missing != "" || !errors.Is(err, fs.ErrPermission) {
		t.Errorf("a file that could not be looked at answered %q and %v, and is not a missing file", missing, err)
	}
}
