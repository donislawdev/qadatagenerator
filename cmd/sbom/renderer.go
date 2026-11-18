package main

import (
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"strings"
)

// The software renderer's notices are NOTICES.md in the folder that travels
// beside the window, kept in .github/opengl of this repository (DEC-323). They
// open with one sentence that names the work, its version and its licence
// expression, list the files the folder holds with their sizes and sums, and
// name in a table the licence files beside them, which hold the texts.
const rendererNotices = "NOTICES.md"

var (
	rendererOpening = regexp.MustCompile("(?m)^" + regexp.QuoteMeta(rendererHeading) + ", version `([^`]+)`\\. " + expressionText)
	rendererLicence = regexp.MustCompile("(?m)^\\| `([^`]+)` \\|")
)

// addRenderer reads the software renderer's notices from a folder and makes
// them the work the window's builds for Windows travel with. dir is how
// messages name the folder.
//
// Its copyright lines come from the licence files its table names, read the
// way the copyright lines of a program's notices are read from its quoted
// texts. An identifier without a text on the SPDX list is refused, because
// the document would have to carry a text these notices do not quote.
func addRenderer(notes map[string]notices, folder fs.FS, dir string) error {
	body, err := fs.ReadFile(folder, rendererNotices)
	if err != nil {
		return fmt.Errorf("the software renderer's notices cannot be read from %s: %w", dir, err)
	}
	text := strings.ReplaceAll(string(body), "\r\n", "\n")
	m := rendererOpening.FindStringSubmatch(text)
	if m == nil {
		return fmt.Errorf("%s/%s does not say \"%s, version `<version>`. <SPDX expression>.\"", dir, rendererNotices, rendererHeading)
	}
	c := component{path: rendererHeading, version: m[1]}
	named, err := readExpression(&c, rendererHeading, m[2])
	if err != nil {
		return fmt.Errorf("%s/%s: %w", dir, rendererNotices, err)
	}
	for _, id := range named {
		if strings.HasPrefix(id, refPrefix) {
			return fmt.Errorf("%s/%s names %s, and quotes no text for the document to carry", dir, rendererNotices, id)
		}
	}
	for _, line := range strings.Split(text, "\n") {
		if f := listedFileLine.FindStringSubmatch(strings.TrimSpace(line)); f != nil {
			c.files = append(c.files, listedFile{path: f[1], size: f[2], sha256: f[3]})
		}
	}
	if err := works[rendererHeading].check(c); err != nil {
		return fmt.Errorf("%s/%s: %w", dir, rendererNotices, err)
	}
	licences := rendererLicence.FindAllStringSubmatch(text, -1)
	if len(licences) == 0 {
		return fmt.Errorf("%s/%s names no licence file in its table, so it quotes no text", dir, rendererNotices)
	}
	for _, l := range licences {
		licence, err := fs.ReadFile(folder, l[1])
		if err != nil {
			return fmt.Errorf("%s/%s names %s, which cannot be read: %w", dir, rendererNotices, l[1], err)
		}
		for _, line := range strings.Split(strings.ReplaceAll(string(licence), "\r\n", "\n"), "\n") {
			c.copyrights = appendCopyright(c.copyrights, line)
		}
	}
	w := notes[windowPath]
	if _, twice := w.works[rendererHeading]; twice {
		return errors.New("the software renderer is described twice")
	}
	if w.works == nil {
		w.works = map[string]component{}
	}
	w.works[rendererHeading] = c
	notes[windowPath] = w
	return nil
}
