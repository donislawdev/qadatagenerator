// Package assets carries the typeface and the icon set the window is drawn
// with, compiled into the binary so that the product looks the same on a
// machine that has neither installed.
//
// # Why these files are here rather than asked of the system
//
// A window that asks the operating system for a font gets a different one on
// every platform, and a layout measured against one of them is wrong on the
// other two. Shipping the faces is the only way the same build looks the same
// everywhere, which is the same argument as the one for a pinned toolchain.
//
// # Why the files sit inside this package
//
// The embed directive cannot reach a directory above the package that holds
// it, which was measured and written down as DEC-134. Anything embedded is
// therefore a file in this directory, and the licence texts sit beside the
// files they cover rather than in one folder somewhere else.
//
// # Terms, and the day somebody read them
//
// IBM Plex Sans and IBM Plex Mono are under the SIL Open Font License, read at
// github.com/IBM/plex on 2026-09-08 for release 1.1.0. Lucide is under ISC,
// and the icons it inherited from Feather are additionally under MIT, both
// read in the project's own LICENSE file at tag 1.43.0 on the same day. Every
// one of those terms is compatible with GPL-3.0 and every one asks the same
// thing in return: keep the notice. The notices are the two LICENSE files in
// the directories below, and they are what makes keeping these files allowed.
//
// The npm packages of IBM Plex are deliberately not used. The release note for
// 1.1.0 records that telemetry was added to the package, which says nothing
// about the font files themselves but is exactly the kind of thing an offline
// product should not invite in. What is embedded here is the .ttf files out of
// the release archive.
package assets

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed fonts/*.ttf
var fonts embed.FS

//go:embed icons/*.svg
var icons embed.FS

// Where the embedded files sit and what an icon file is called.
//
// The embed directives above have to spell the same paths as literals, because
// the compiler reads them before any constant exists. That is a duplication of
// two words that cannot be removed, so the test beside this file closes it
// instead: it lists both directories and fails if either comes back empty.
const (
	fontDir       = "fonts"
	iconDir       = "icons"
	iconExtension = ".svg"
)

// Font names, as the files are called. They are the four weights the window
// asks for and no more, because every unused face is a quarter of a megabyte
// somebody downloads and never sees.
const (
	SansRegular    = "IBMPlexSans-Regular.ttf"
	SansSemiBold   = "IBMPlexSans-SemiBold.ttf"
	SansItalic     = "IBMPlexSans-Italic.ttf"
	SansSemiItalic = "IBMPlexSans-SemiBoldItalic.ttf"
	MonoRegular    = "IBMPlexMono-Regular.ttf"
	MonoSemiBold   = "IBMPlexMono-SemiBold.ttf"
)

// Font returns the bytes of one embedded typeface.
//
// It panics when the name is not there, and that is deliberate. The names are
// constants in this file, so a miss is a typo in our own source rather than
// anything a user did, and a window that silently falls back to a system font
// would look wrong in a way nobody would trace back to here.
func Font(name string) []byte {
	b, err := fonts.ReadFile(fontDir + "/" + name)
	if err != nil {
		panic(fmt.Sprintf("embedded font %q is missing: %v", name, err))
	}
	return b
}

// Icon returns one embedded icon as SVG source, with the colour still written
// as currentColor.
//
// # Why the colour is left in the source
//
// The toolkit offers to recolour an icon, and for this icon set it would ruin
// it. Its Colorize walks fill attributes only and never touches stroke, while
// the decoder turns a leftover currentColor into black. These icons are drawn
// entirely in stroke, so that route paints a black shape onto a dark window.
// Substituting the colour in the source, in both attributes at once, is both
// the fix and a gain: it lets one icon carry a different fill and stroke, and
// lets the stroke width follow the size the icon is drawn at.
func Icon(name string) []byte {
	b, err := icons.ReadFile(iconDir + "/" + name + iconExtension)
	if err != nil {
		panic(fmt.Sprintf("embedded icon %q is missing: %v", name, err))
	}
	return b
}

// IconNames lists every embedded icon, sorted, so a test can compare the set
// against what the window asks for in both directions.
func IconNames() []string {
	entries, err := icons.ReadDir(iconDir)
	if err != nil {
		panic(fmt.Sprintf("embedded icons are missing: %v", err))
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		// Trimmed rather than sliced by a length. The guard on literals
		// refuses a bare number anywhere under the window, and it is right
		// to: a four written here is the length of an extension, and every
		// other four in this half of the tree is a distance. Neither the
		// guard nor a reader can tell those apart, so the number goes and
		// the intent is said in words instead.
		name := entry.Name()
		if trimmed := strings.TrimSuffix(name, iconExtension); trimmed != name {
			names = append(names, trimmed)
		}
	}
	return names
}
