package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
)

// # The second start (B-414, DEC-317)
//
// On a computer whose graphics driver gives no OpenGL window, such as a
// virtual machine without 3D acceleration, the toolkit makes no window and
// failure.go catches that. On Windows this window then starts itself again
// with -software-gl, and the second process loads Mesa's software renderer
// from a folder named opengl beside the executable before the toolkit exists.
// The toolkit's later request for opengl32.dll by name is then answered by the
// library already loaded under that name, which is Mesa's.
//
// A second process rather than a second try in this one, because the toolkit
// keeps what it learned about the failed window in state its public API cannot
// reset, and because the first attempt has already loaded the system's
// opengl32.dll here. Only after a refusal, so a computer with a working driver
// keeps drawing with it. Only on Windows, because a Linux desktop has Mesa of
// its own and macOS always has OpenGL, and nothing is carried beside the
// window there.
//
// The second process never starts a third. When the toolkit refuses there too,
// the answer is the refusal, with a sentence saying the renderer was tried.

// softwareFlag is the flag that asks for the software renderer, the same name
// the two sibling projects of this one give it. Public: it is how somebody
// sees the window as a computer without a driver sees it.
const softwareFlag = "software-gl"

// The renderer's files and what it is told before it loads.
//
// The order is not a preference. Mesa's opengl32.dll imports
// libgallium_wgl.dll by name (its import table, read on 2026-11-17 in Mesa
// 26.2.5 from pal1000/mesa-dist-win), and a library loaded by path has its own
// dependencies searched for the ordinary way rather than beside it, so loaded
// first it fails with "module not found". Loaded second, it finds the renderer
// already loaded under that name.
//
// The driver is named rather than left to Mesa, which otherwise picks one
// itself. Measured by Testing Files Generator on 2026-09-17 on a computer with
// a graphics card: Mesa chose its driver over Direct3D 12, which needs a
// dxil.dll nobody ships, and the process died within three seconds with no
// word, two times out of two.
//
// In a folder of their own rather than beside the executable, because a file
// named opengl32.dll beside a program is the shape of a library hijack, and
// these two are only ever loaded by the absolute path worked out here.
const (
	softwareDir      = "opengl"
	softwareRenderer = "libgallium_wgl.dll"
	softwareLoader   = "opengl32.dll"
	softwareDriver   = "GALLIUM_DRIVER"
	softwareLLVMPipe = "llvmpipe"
)

// softwareFiles are the renderer's files under exeDir, in the order they have
// to be loaded. Everything that names them takes them from here.
func softwareFiles(exeDir string) []string {
	return []string{
		filepath.Join(exeDir, softwareDir, softwareRenderer),
		filepath.Join(exeDir, softwareDir, softwareLoader),
	}
}

// executableDir is the folder the running executable is in.
func executableDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}

// rendererBeside answers the first of the renderer's files that is missing
// under exeDir, or what stood in the way of looking. A file that cannot be
// looked at is not a file that is missing, because the advice differs: put the
// folder back, against check who may read it. The question to the file system
// is handed in so that a test can answer it with a refusal no disk arranges
// the same way on every system.
func rendererBeside(exeDir string, stat func(string) (fs.FileInfo, error)) (missing string, err error) {
	for _, path := range softwareFiles(exeDir) {
		_, err := stat(path)
		switch {
		case err == nil:
			continue
		case errors.Is(err, fs.ErrNotExist):
			return path, nil
		default:
			return "", err
		}
	}
	return "", nil
}

// renderer is how this window draws: whether the launch asked for the software
// renderer, and what happened when it was loaded.
type renderer struct {
	asked  bool
	loaded error
}

// software reports whether this window draws with the software renderer, which
// the About screen says.
func (r renderer) software() bool { return r.asked && r.loaded == nil }

// rendererFor loads the software renderer when the launch asked for it, and
// says on the error channel how that went. It is called before the toolkit
// exists, which is the whole point of it.
func rendererFor(asked launch, text i18n.Catalogue, channel io.Writer, load func() error) renderer {
	if !asked.softwareGL {
		return renderer{}
	}
	drawing := renderer{asked: true, loaded: load()}
	_, _ = fmt.Fprintln(channel, loadingSentence(text, drawing.loaded))
	return drawing
}

// secondAttempt is the second start, with every part that touches the world
// a field, so that the decision can be pressed by a test on any system without
// a window, a renderer or a second process. That is also why the name of the
// system is a field rather than read.
type secondAttempt struct {
	asked launch
	goos  string
	// executableDir is where the executable is, or what stood in the way of
	// knowing.
	executableDir func() (string, error)
	// beside is the first of the renderer's files missing beside the
	// executable, or what stood in the way of looking.
	beside func(exeDir string) (missing string, err error)
	// start starts this program again with args and answers its exit code.
	start func(args []string) (code int, err error)
	// note is where the line said before the second start goes.
	note io.Writer
	text i18n.Catalogue
}

// The reasons the second start was not taken. Each becomes a sentence through
// rendererSentence, except notShipped, which becomes none: on a system nothing
// is carried for there is nothing missing to name. Their Error text is the
// fact and no words, because the words a person reads come from the catalogue.
type (
	// alreadySoftware: this IS the second process, and it starts no third.
	alreadySoftware struct{}
	// notShipped: no renderer comes with the window on this system.
	notShipped struct{ goos string }
	// notBeside: a file of the renderer is not where it belongs.
	notBeside struct{ path string }
	// notReadable: a file of the renderer could not be looked at, in the
	// system's words, which name the file.
	notReadable struct{ err error }
	// startFailed: the second process could not be started.
	startFailed struct{ err error }
)

func (alreadySoftware) Error() string { return softwareFlag }
func (e notShipped) Error() string    { return e.goos }
func (e notBeside) Error() string     { return e.path }
func (e notReadable) Error() string   { return e.err.Error() }
func (e notReadable) Unwrap() error   { return e.err }
func (e startFailed) Error() string   { return e.err.Error() }
func (e startFailed) Unwrap() error   { return e.err }

// secondAttemptFor is the second start of the real window: this executable,
// this system, a real second process.
func secondAttemptFor(asked launch, text i18n.Catalogue, note io.Writer) secondAttempt {
	return secondAttempt{
		asked:         asked,
		goos:          runtime.GOOS,
		executableDir: executableDir,
		beside:        func(dir string) (string, error) { return rendererBeside(dir, os.Stat) },
		start:         startAgain,
		note:          note,
		text:          text,
	}
}

// try starts this program again with the same arguments and the flag, waits
// for it, and answers its exit code. When it does not start it, it answers
// why, and the refusal carries that reason.
func (a secondAttempt) try() (int, error) {
	if a.asked.softwareGL {
		return 0, alreadySoftware{}
	}
	if a.goos != "windows" {
		return 0, notShipped{a.goos}
	}
	dir, err := a.executableDir()
	if err != nil {
		return 0, startFailed{err}
	}
	missing, err := a.beside(dir)
	if err != nil {
		return 0, notReadable{err}
	}
	if missing != "" {
		return 0, notBeside{missing}
	}
	// Said before the second process starts, on the channel whatever started
	// this one reads, and as a note rather than a diagnosis, because all this
	// program knows is that the first attempt gave no window.
	_, _ = fmt.Fprintln(a.note, a.text.T(i18n.WindowSoftwareAgain))
	code, err := a.start(append(append([]string{}, a.asked.given...), "-"+softwareFlag))
	if err != nil {
		return 0, startFailed{err}
	}
	return code, nil
}

// rendererSentence is what a refusal adds about the software renderer: why it
// was not tried, or that it was and did not help. Nothing on a system no
// renderer comes for, where a sentence about a missing file would be about a
// file nobody promised.
//
// why is what try answered. loaded is what this process found when it loaded
// the renderer: nil when it loaded, and nil as well in a first process, which
// never tried.
func rendererSentence(text i18n.Catalogue, why, loaded error) string {
	var beside notBeside
	var unreadable notReadable
	var failed startFailed
	var shipsNot notShipped
	switch {
	case errors.As(why, &beside):
		return fmt.Sprintf(text.T(i18n.WindowSoftwareNotBeside), beside.path)
	case errors.As(why, &unreadable):
		return fmt.Sprintf(text.T(i18n.WindowSoftwareNotReadable), withoutFullStop(unreadable.err))
	case errors.As(why, &failed):
		return fmt.Sprintf(text.T(i18n.WindowSoftwareStartFailed), withoutFullStop(failed.err))
	case !errors.Is(why, alreadySoftware{}) || errors.As(loaded, &shipsNot):
		// Not tried on a system nothing comes for, with the flag or without.
		return ""
	case loaded != nil:
		return fmt.Sprintf(text.T(i18n.WindowSoftwareNotLoaded), withoutFullStop(loaded))
	default:
		return text.T(i18n.WindowSoftwareDidNotHelp)
	}
}

// loadingSentence is what a window asked for the software renderer says about
// loading it: that it draws with it, that none comes for this system, or what
// stood in the way.
func loadingSentence(text i18n.Catalogue, loaded error) string {
	var shipsNot notShipped
	switch {
	case errors.As(loaded, &shipsNot):
		return text.T(i18n.WindowSoftwareNotShipped)
	case loaded != nil:
		return fmt.Sprintf(text.T(i18n.WindowSoftwareNotLoaded), withoutFullStop(loaded))
	default:
		return text.T(i18n.WindowSoftwareDrawing)
	}
}

// withoutFullStop is an error's words without the full stop Windows ends its
// own with, for a sentence that brings its own.
func withoutFullStop(err error) string {
	return strings.TrimRight(strings.TrimSpace(err.Error()), ".")
}
