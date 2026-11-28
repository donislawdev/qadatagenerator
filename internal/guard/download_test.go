package guard

import (
	"maps"
	"sort"
	"strings"
	"testing"
)

// # What this defends (B-394)
//
// The README and the download page of the website are where a person reads
// which file to take, before anything is downloaded. They name the archives
// the signing script makes, the installer and the checksum file, and nothing
// else reads them on the day of a release, so an archive renamed in the
// release would leave both naming a file no release has. The page shows the
// README's table rather than a copy of it, which is asked here too, so the
// README is the one place the names are written for a reader.
//
// Adapted on 2026-11-28 from the README guard of Cert Doctor, a project of the
// same owner under the same licence.
const (
	readmeFile      = "README.md"
	downloadPage    = "web/content/en/download.md"
	downloadHeading = "Download and run"
	checksumFile    = "`verify-SHA256SUMS.txt`"
)

// archiveName is the name build_release.sh gives the archive of one program
// for one target, with version standing in for the version: Go's darwin is
// labelled macos, and Windows gets a zip.
func archiveName(program, goos, goarch, version string) string {
	system, ending := goos, ".tar.gz"
	switch goos {
	case "darwin":
		system = "macos"
	case "windows":
		ending = ".zip"
	}
	return program + "_" + version + "_" + system + "_" + goarch + ending
}

// TestTheReadmeAndTheDownloadPageNameWhatAReleaseShips holds the README's
// section on downloading to the release: every archive the signing script
// makes or passes on, the installer and the checksum file, by the names a
// release gives them. The download page has to show that section's table.
func TestTheReadmeAndTheDownloadPageNameWhatAReleaseShips(t *testing.T) {
	root := rootOrFail(t)
	files := map[string]string{}
	for _, rel := range []string{readmeFile, downloadPage, signingScript} {
		files[rel] = readRepoFile(t, root, rel)
	}
	broken := maps.Clone(files)
	broken[readmeFile] = planted(t, readmeFile, files[readmeFile], "`qadg_<version>_linux_arm64.tar.gz`", "`qadg_<version>_linux_aarch64.tar.gz`")
	broken[readmeFile] = planted(t, readmeFile, broken[readmeFile], checksumFile+" on the release page", "`SHA256SUMS` on the release page")
	broken[downloadPage] = planted(t, downloadPage, files[downloadPage], `{{< readme-table heading="`+downloadHeading+`" >}}`, "")
	found := checkDownloadNames(broken)
	requireFinding(t, found, "never names `qadg_<version>_linux_arm64.tar.gz`")
	requireFinding(t, found, "never names "+checksumFile)
	requireFinding(t, found, "does not show the README's table under "+downloadHeading)

	section := "\n## " + downloadHeading + "\n"
	moved := map[string]string{readmeFile: planted(t, readmeFile, files[readmeFile], section, "\n## Downloads\n"),
		downloadPage: files[downloadPage], signingScript: files[signingScript]}
	requireFinding(t, checkDownloadNames(moved), "has no section ## "+downloadHeading)

	requireNone(t, checkDownloadNames(files), "what the README and the download page say a release ships")
}

// readmeSection is the text under "## heading" up to the next heading of the
// same level, and whether there is such a heading.
func readmeSection(readme, heading string) (string, bool) {
	_, after, found := strings.Cut(strings.ReplaceAll(readme, "\r\n", "\n"), "\n## "+heading+"\n")
	if !found {
		return "", false
	}
	if end := strings.Index(after, "\n## "); end >= 0 {
		after = after[:end]
	}
	return after, true
}

func checkDownloadNames(files map[string]string) []string {
	var problems []string
	section, found := readmeSection(files[readmeFile], downloadHeading)
	if !found {
		problems = append(problems, readmeFile+" has no section ## "+downloadHeading+", which the download page shows")
	}
	names := []string{installerFile, checksumFile}
	rows, err := signingTable(files[signingScript])
	if err != nil {
		problems = append(problems, signingScript+": "+err.Error())
	}
	for _, row := range rows {
		names = append(names, "`"+archiveName(row[0], row[1], row[2], "<version>")+"`")
	}
	for _, name := range names {
		if found && !strings.Contains(section, name) {
			problems = append(problems, readmeFile+": "+downloadHeading+" never names "+name+", which a release ships")
		}
	}
	if !strings.Contains(files[downloadPage], `{{< readme-table heading="`+downloadHeading+`" >}}`) {
		problems = append(problems, downloadPage+" does not show the README's table under "+downloadHeading+
			", so it can name other archives than the README does")
	}
	sort.Strings(problems)
	return problems
}
