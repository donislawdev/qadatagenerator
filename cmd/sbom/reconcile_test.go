package main

import (
	"reflect"
	"testing"
)

// fixtureNotes is the notices of both programs of the fixture tree.
func fixtureNotes(t *testing.T) map[string]notices {
	t.Helper()
	notes, err := parseTree(fixtureTree)
	if err != nil {
		t.Fatal(err)
	}
	return notes
}

// fixtureNotesWithRenderer is the same, with the software renderer's notices
// read from fixtureRenderer.
func fixtureNotesWithRenderer(t *testing.T) map[string]notices {
	t.Helper()
	notes := fixtureNotes(t)
	if err := addRenderer(notes, fixtureRenderer(), "opengl"); err != nil {
		t.Fatal(err)
	}
	return notes
}

func readBuilds(t *testing.T, listing string) []build {
	t.Helper()
	builds, err := parseBuilds(listing)
	if err != nil {
		t.Fatal(err)
	}
	return builds
}

func TestBuildsThatLinkWhatTheNoticesNameAreAccepted(t *testing.T) {
	for name, c := range map[string]struct {
		listing  string
		renderer bool
	}{
		"the command line":                    {fixtureListing(), false},
		"both programs, with the renderer":    {fixtureListing() + fixtureWindowListing(), true},
		"the window alone, one build":         {windowListingFor("dist/w", "linux/amd64", windowDeps(fixturePortal)), false},
		"the window for Windows alone":        {windowListingFor("dist/w.exe", "windows/amd64", windowDeps(fixturePortal)), true},
		"the command line for Windows, alone": {listingFor("dist/a.exe", "windows/amd64", "go1.27.4", fixtureDeps), false},
	} {
		notes := fixtureNotes(t)
		if c.renderer {
			notes = fixtureNotesWithRenderer(t)
		}
		if problems := reconcile(readBuilds(t, c.listing), notes); len(problems) != 0 {
			t.Errorf("%s: problems with builds that match: %v", name, problems)
		}
	}
}

// Every disagreement is reported, each naming the build it is about, so one run
// lists everything a dependency update has to touch.
func TestEveryDisagreementIsReported(t *testing.T) {
	deps := map[string]string{
		"example.com/alpha": "v1.2.4",
		"example.com/gamma": "v0.1.0",
	}
	builds := readBuilds(t, listingFor("dist/a", "darwin/arm64", "go1.27.5", deps))
	want := []string{
		"darwin/arm64 was built with go1.27.5, and the notices name go1.27.4",
		"darwin/arm64 links example.com/alpha v1.2.4, and the notices name v1.2.3",
		"darwin/arm64 links example.com/gamma v0.1.0, which the notices do not name",
		"the notices name example.com/beta/v2, which none of the 1 build(s) of qadg given links",
	}
	if got := reconcile(builds, fixtureNotes(t)); !reflect.DeepEqual(got, want) {
		t.Errorf("reported\n%q\nwant\n%q", got, want)
	}
}

// A build of the window is held to the window's notices and to no other: a
// module only the command line's notices name is one the window does not
// link, and one only the window's name is one the command line does not.
func TestEachProgramIsHeldToItsOwnNotices(t *testing.T) {
	window := windowDeps(map[string]string{"example.com/alpha": "v1.2.3"})
	delete(window, "example.com/glfw")
	listing := listingFor("dist/a", "linux/amd64", "go1.27.4", fixtureDeps) +
		windowListingFor("dist/w", "linux/amd64", window)
	want := []string{
		"qadg-gui linux/amd64 links example.com/alpha v1.2.3, which the window's notices do not name",
		"the window's notices name example.com/glfw, which none of the 1 build(s) of qadg-gui given links",
		"the window's notices name example.com/portal, which none of the 1 build(s) of qadg-gui given links",
	}
	if got := reconcile(readBuilds(t, listing), fixtureNotes(t)); !reflect.DeepEqual(got, want) {
		t.Errorf("reported\n%q\nwant\n%q", got, want)
	}
}

// A module one build links is accounted for by that build, so a module linked
// on one system only is not reported as named and missing.
func TestAModuleIsLinkedWhenAnyBuildLinksIt(t *testing.T) {
	alphaOnly := map[string]string{"example.com/alpha": "v1.2.3"}
	text := listingFor("dist/a", "linux/amd64", "go1.27.4", alphaOnly) +
		listingFor("dist/b", "linux/arm64", "go1.27.4", fixtureDeps)
	if problems := reconcile(readBuilds(t, text), fixtureNotes(t)); len(problems) != 0 {
		t.Errorf("problems: %v", problems)
	}
}

// The software renderer goes with the build of the window for Windows and with
// nothing else: that build without its notices leaves two files of its
// archive out, and its notices without that build describe an archive nobody
// built, the command line's build for the same target included.
func TestTheRendererGoesWithTheBuildItTravelsBeside(t *testing.T) {
	// Each build alone, so each links every module the window's notices name.
	windows := windowListingFor("dist/w.exe", "windows/amd64", windowDeps(fixturePortal))
	linux := windowListingFor("dist/w", "linux/amd64", windowDeps(fixturePortal))
	cliWindows := listingFor("dist/a.exe", "windows/amd64", "go1.27.4", fixtureDeps)
	for _, c := range []struct {
		name     string
		listing  string
		renderer bool
		want     []string
	}{
		{"the build without the renderer", windows, false,
			[]string{"qadg-gui windows/amd64 travels with Mesa 3D, and no -renderer folder with its notices was given"}},
		{"the renderer without the build", linux, true,
			[]string{"the notices of Mesa 3D were given, and no build of qadg-gui for windows/amd64, which it travels beside"}},
		{"the renderer beside the command line", cliWindows, true,
			[]string{"the notices of Mesa 3D were given, and no build of qadg-gui for windows/amd64, which it travels beside"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			notes := fixtureNotes(t)
			if c.renderer {
				notes = fixtureNotesWithRenderer(t)
			}
			if got := reconcile(readBuilds(t, c.listing), notes); !reflect.DeepEqual(got, c.want) {
				t.Errorf("reported\n%q\nwant\n%q", got, c.want)
			}
		})
	}
}
