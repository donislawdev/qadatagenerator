package main

import (
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

// The folder of the renderer's notices is read for its version, its licence
// terms, the files it lists and the copyright lines of the licence files its
// table names, and the Apache template line that names nobody stays out.
func TestTheRenderersNoticesAreRead(t *testing.T) {
	mesa := fixtureNotesWithRenderer(t)[windowPath].works[rendererHeading]
	want := component{
		path:     rendererHeading,
		version:  "26.2.5",
		licences: []string{"MIT", "Apache-2.0 WITH LLVM-exception", "NCSA"},
		copyrights: []string{
			"Copyright (C) 1999-2007  Brian Paul   All Rights Reserved.",
			"Copyright (c) 2003-2019 University of Illinois at Urbana-Champaign.",
		},
		files: []listedFile{
			{"opengl/libgallium_wgl.dll", "61880320", "700e1fb7b9487d9d1c53752b03aeb040d22f3731b46aa254d88830b6279474de"},
			{"opengl/opengl32.dll", "139264", "c8511461ba71a0288750467e3be2f5c1c2bbbceaaf0ab5d8c496b0ba7302daa8"},
		},
	}
	if !reflect.DeepEqual(mesa, want) {
		t.Errorf("read\n%+v\nwant\n%+v", mesa, want)
	}
}

// rendererDir is the folder this repository keeps the renderer's notices in,
// from this package.
const rendererDir = "../../.github/opengl"

// The folder this repository keeps is read whole: its sentence, its two files
// and the copyright lines of its licence files, so a change to it that this
// command cannot read fails on the push.
//
// It asks its check for both answers: the same folder with NCSA taken out of
// its sentence, which LLVM's licence file needs since 2026-11-18, has to be
// reported, and the folder as it is has to pass.
func TestTheRenderersNoticesInThisTreeAreRead(t *testing.T) {
	kept := os.DirFS(rendererDir)
	planted := fstest.MapFS{}
	entries, err := fs.ReadDir(kept, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		body, err := fs.ReadFile(kept, e.Name())
		if err != nil {
			t.Fatal(err)
		}
		planted[e.Name()] = &fstest.MapFile{Data: body}
	}
	sentence := string(planted[rendererNotices].Data)
	without := strings.Replace(sentence, " AND NCSA", "", 1)
	if without == sentence {
		t.Fatal("NCSA is not in the renderer's sentence, so the planted case would read the folder as it is")
	}
	planted[rendererNotices] = &fstest.MapFile{Data: []byte(without)}
	if len(rendererProblems(planted)) == 0 {
		t.Error("the check found nothing wrong with the renderer's notices without NCSA")
	}
	for _, p := range rendererProblems(kept) {
		t.Error(p)
	}
}

// rendererProblems is everything wrong with a folder that should be the
// renderer's notices this tree keeps.
func rendererProblems(folder fs.FS) []string {
	notes := map[string]notices{}
	if err := addRenderer(notes, folder, rendererDir); err != nil {
		return []string{err.Error()}
	}
	var problems []string
	mesa := notes[windowPath].works[rendererHeading]
	if mesa.version == "" || len(mesa.files) != 2 {
		problems = append(problems, fmt.Sprintf("the renderer is read as version %q with %d files", mesa.version, len(mesa.files)))
	}
	for _, want := range []string{"MIT", "Apache-2.0 WITH LLVM-exception", "NCSA", "BSL-1.0", "BSD-3-Clause"} {
		if !slices.Contains(mesa.licences, want) {
			problems = append(problems, fmt.Sprintf("the renderer's terms are %q, without %s", mesa.licences, want))
		}
	}
	for _, holder := range []string{"VMware", "University of Illinois", "Microsoft"} {
		if !strings.Contains(strings.Join(mesa.copyrights, "\n"), holder) {
			problems = append(problems, "no copyright line of the renderer names "+holder)
		}
	}
	return problems
}

func TestRendererNoticesThatCannotDescribeTheFilesAreRefused(t *testing.T) {
	edit := func(old, replacement string) fstest.MapFS {
		folder := fixtureRenderer()
		file := *folder[rendererNotices]
		edited := strings.Replace(string(file.Data), old, replacement, 1)
		if edited == string(file.Data) {
			t.Fatalf("%q is not in the fixture, so the case would test the fixture as it is", old)
		}
		file.Data = []byte(edited)
		folder[rendererNotices] = &file
		return folder
	}
	without := func(name string) fstest.MapFS {
		folder := fixtureRenderer()
		delete(folder, name)
		return folder
	}
	for _, c := range []struct {
		name   string
		folder fstest.MapFS
		want   string
	}{
		{"no notices", without(rendererNotices), "the software renderer's notices cannot be read from opengl"},
		{"no sentence", edit("Mesa 3D, version", "Mesa, version"), "does not say \"Mesa 3D, version"},
		{"words for a licence", edit("MIT AND Apache-2.0 WITH LLVM-exception AND NCSA.", "Various licences."),
			"does not say \"Mesa 3D, version"},
		{"identifier nobody read", edit("AND NCSA.", "AND NCSA-1.0."), "names NCSA-1.0, which is not"},
		{"a text to carry", edit("AND NCSA.", "AND LicenseRef-Cephes."), "names LicenseRef-Cephes, and quotes no text"},
		{"no licence files", edit("| `LICENSE-Mesa.txt` | Mesa | MIT |\n| `LICENSE-LLVM.txt`", "| LLVM"),
			"names no licence file in its table"},
		{"a licence file missing", without("LICENSE-LLVM.txt"), "names LICENSE-LLVM.txt, which cannot be read"},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := addRenderer(map[string]notices{}, c.folder, "opengl")
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("err = %v, want one saying %q", err, c.want)
			}
		})
	}
	// A list with no file in it: both lines lose the backticks a file is
	// named in.
	noFiles := edit("- `opengl/libgallium_wgl.dll`", "- opengl/libgallium_wgl.dll")
	file := *noFiles[rendererNotices]
	file.Data = []byte(strings.Replace(string(file.Data), "- `opengl/opengl32.dll`", "- opengl/opengl32.dll", 1))
	noFiles[rendererNotices] = &file
	if err := addRenderer(map[string]notices{}, noFiles, "opengl"); err == nil ||
		!strings.Contains(err.Error(), `"Mesa 3D" is of files beside a binary, and lists none`) {
		t.Errorf("notices that list no file were let through: %v", err)
	}
	notes := fixtureNotesWithRenderer(t)
	if err := addRenderer(notes, fixtureRenderer(), "opengl"); err == nil || !strings.Contains(err.Error(), "described twice") {
		t.Errorf("the renderer given twice was let through: %v", err)
	}
}
