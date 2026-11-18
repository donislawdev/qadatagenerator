package main

import (
	"strings"
	"testing"
)

// realListing is what go version -m printed on 2026-11-18 for qadg built for
// linux/arm64 with -trimpath from a clean checkout of commit edf3d17, with two
// changes: lines end in CRLF, as a pipe on Windows may leave them, and the file
// is a Windows path with a second ": " in it, which is what holds the rule
// that the toolchain follows the LAST one.
const realListing = "C:\\dist: two\\qadg: go1.27.4\r\n" +
	"\tpath\tgithub.com/donislawdev/qadatagenerator/cmd/qadg\r\n" +
	"\tmod\tgithub.com/donislawdev/qadatagenerator\tv0.0.0-20261118133242-edf3d17bad3e\t\r\n" +
	"\tdep\tgithub.com/goccy/go-yaml\tv1.19.2\th1:PmFC1S6h8ljIz6gMRBopkjP1TVT7xuwrButHID66PoM=\r\n" +
	"\tdep\tgolang.org/x/text\tv0.43.0\th1:1QivrlhwAsnMOcqgOdl9my7s9OyZl8O5/UGUTJVXoWc=\r\n" +
	"\tbuild\t-buildmode=exe\r\n" +
	"\tbuild\t-compiler=gc\r\n" +
	"\tbuild\t-trimpath=true\r\n" +
	"\tbuild\tCGO_ENABLED=0\r\n" +
	"\tbuild\tGOARCH=arm64\r\n" +
	"\tbuild\tGOOS=linux\r\n" +
	"\tbuild\tGOARM64=v8.0\r\n" +
	"\tbuild\tvcs=git\r\n" +
	"\tbuild\tvcs.revision=edf3d17bad3e9ab89e7265101fcd1b1e42dc5ef4\r\n" +
	"\tbuild\tvcs.time=2026-11-18T13:32:42Z\r\n" +
	"\tbuild\tvcs.modified=false\r\n"

func TestWhatGoVersionPrintsIsReadAsABuild(t *testing.T) {
	builds, err := parseBuilds(realListing)
	if err != nil {
		t.Fatal(err)
	}
	if len(builds) != 1 {
		t.Fatalf("%d builds read from one listing", len(builds))
	}
	b := builds[0]
	for _, check := range []struct{ field, got, want string }{
		{"file", b.file, `C:\dist: two\qadg`},
		{"program", b.program, programPath},
		{"target", b.target, "linux/arm64"},
		{"toolchain", b.goVersion, "go1.27.4"},
		{"revision", b.revision, "edf3d17bad3e9ab89e7265101fcd1b1e42dc5ef4"},
		{"time", b.time, "2026-11-18T13:32:42Z"},
		{"yaml", b.deps["github.com/goccy/go-yaml"], "v1.19.2"},
	} {
		if check.got != check.want {
			t.Errorf("%s is %q, want %q", check.field, check.got, check.want)
		}
	}
	if len(b.deps) != 2 {
		t.Errorf("%d modules read, the listing names two", len(b.deps))
	}
}

func TestSeveralBuildsOfOneCommitAreReadInTheirOrder(t *testing.T) {
	builds, err := parseBuilds(fixtureListing())
	if err != nil {
		t.Fatal(err)
	}
	if len(builds) != 2 || builds[0].target != "linux/amd64" || builds[1].target != "windows/arm64" {
		t.Fatalf("read %+v", builds)
	}
}

// The window is the second program a release builds, with the OpenGL binding
// it links from a directory of this repository, and a build of it for a target
// is not the command line's build for the same target.
func TestABuildOfTheWindowIsReadWithItsReplacement(t *testing.T) {
	listing := listingFor("dist/a", "linux/amd64", "go1.27.4", fixtureDeps) +
		windowListingFor("dist/w", "linux/amd64", windowDeps(nil))
	builds, err := parseBuilds(listing)
	if err != nil {
		t.Fatal(err)
	}
	if len(builds) != 2 {
		t.Fatalf("%d builds read from two listings", len(builds))
	}
	w := builds[1]
	if w.program != windowPath || w.label() != "qadg-gui linux/amd64" || builds[0].label() != "linux/amd64" {
		t.Errorf("read %s as %q and %s as %q", w.program, w.label(), builds[0].program, builds[0].label())
	}
	if w.replaced["github.com/go-gl/gl"] != "./third_party/go-gl-gl" || len(w.replaced) != 1 {
		t.Errorf("replacements read: %v", w.replaced)
	}
	if w.deps["github.com/go-gl/gl"] != "v0.0.0-20260331235117-4566fea9a276" {
		t.Errorf("the binding is at %q, where the notices name the version it replaces", w.deps["github.com/go-gl/gl"])
	}
}

// Every build has a program, so a package nobody builds is a mistake in this
// command rather than in a build.
func TestOnlyThePackagesAReleaseBuildsHaveAProgram(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("a package that is not a program was given one")
		}
	}()
	programNamed(modulePath + "/cmd/sbom")
}

// A commit time in another zone is still the same instant, and SPDX takes it
// in UTC with a Z.
func TestTheCommitTimeIsWrittenInUTC(t *testing.T) {
	text := strings.Replace(fixtureListing(), "vcs.time=2026-11-18T08:01:06Z", "vcs.time=2026-11-18T09:01:06+01:00", 1)
	builds, err := parseBuilds(text)
	if err != nil {
		t.Fatal(err)
	}
	if builds[0].time != "2026-11-18T08:01:06Z" {
		t.Errorf("the time is %q", builds[0].time)
	}
}

func TestAListingThatIsNotOfReleaseBuildsIsRefused(t *testing.T) {
	one := listingFor("dist/a", "linux/amd64", "go1.27.4", fixtureDeps)
	other := listingFor("dist/b", "linux/arm64", "go1.27.4", fixtureDeps)
	window := windowListingFor("dist/w", "linux/amd64", windowDeps(nil))
	for _, c := range []struct{ name, text, want string }{
		{"nothing", "\n\n", "names no binary"},
		{"indented first", "\tpath\tx\n" + one, "line 1 is indented"},
		{"no toolchain", "dist/a\n", "names no binary and no toolchain"},
		{"not a toolchain", "dist/a: not a Go executable\n", "names no binary and no toolchain"},
		{"broken build line", one + "\tbuild\tnoequals\n", "build line missing '='"},
		{"another program", strings.Replace(one, "\tpath\t"+programPath, "\tpath\t"+modulePath+"/cmd/sbom", 1),
			"is a build of"},
		{"no commit", strings.Replace(one, "\tbuild\tvcs.revision="+fixtureRevision+"\n", "", 1), "records no vcs.revision"},
		{"no target", strings.Replace(one, "\tbuild\tGOOS=linux\n", "", 1), "records no GOOS"},
		{"edited tree", strings.Replace(one, "vcs.modified=false", "vcs.modified=true", 1), "not committed"},
		{"unknown edits", strings.Replace(one, "\tbuild\tvcs.modified=false\n", "", 1), "not committed"},
		{"time that is not one", strings.Replace(one, "2026-11-18T08:01:06Z", "yesterday", 1), "not a time"},
		{"replaced module", strings.Replace(one, "h1:AAAA=\n", "h1:AAAA=\n\t=>\texample.com/fork\tv9.9.9\th1:BBBB=\n", 1),
			"links example.com/fork in place of"},
		{"the binding replaced by a version", strings.Replace(window, "./third_party/go-gl-gl\t(devel)", "example.com/gl\tv1.0.0", 1),
			"links example.com/gl in place of github.com/go-gl/gl"},
		{"the binding from another directory", strings.Replace(window, "./third_party/go-gl-gl", "./elsewhere", 1),
			"links ./elsewhere in place of github.com/go-gl/gl"},
		{"the binding from its directory, at a version", strings.Replace(window, "./third_party/go-gl-gl\t(devel)", "./third_party/go-gl-gl\tv1.0.0", 1),
			"links ./third_party/go-gl-gl in place of github.com/go-gl/gl"},
		{"target twice", one + one, "are both builds for linux/amd64"},
		{"window target twice", window + window, "are both builds for qadg-gui linux/amd64"},
		{"two commits", one + strings.Replace(other, fixtureRevision, "0000000000000000000000000000000000000000", 1),
			"one document describes one commit"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := parseBuilds(c.text)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("err = %v, want one saying %q", err, c.want)
			}
		})
	}
}
