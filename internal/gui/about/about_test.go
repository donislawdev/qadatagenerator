package about

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/fynetheme"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	"github.com/donislawdev/qadatagenerator/internal/license"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

// TestMain starts an application, because a control laid out without one is a
// nil dereference rather than a quiet no operation. This screen carried none
// until 2026-09-08, when it gained its first control: everything on it before
// that was a label, and a label does not ask the running application how wide
// it is.
//
// The theme is the product's, as in every other screen's tests: the lists on
// this screen measure their columns on the text they draw, and the toolkit's
// test theme draws it at other sizes.
func TestMain(m *testing.M) {
	application := test.NewApp()
	application.Settings().SetTheme(fynetheme.Current())
	os.Exit(m.Run())
}

// texts walks the built screen and collects every string a person would read.
//
// The screen is checked by what it says rather than by how it looks. A test
// comparing rendered pixels to a stored image goes red when a padding changes
// and says nothing about whether the licence is still on screen, which is the
// promise this screen exists to keep.
//
// 🔴 Text of the canvas as well as labels. The values in the lists and the
// card are drawn as canvas text since 2026-09-23, and a walk that knew only
// labels would have reported every module missing.
func texts(object fyne.CanvasObject) []string {
	switch typed := object.(type) {
	case *widget.Label:
		return []string{typed.Text}
	case *canvas.Text:
		return []string{typed.Text}
	case *fyne.Container:
		var found []string
		for _, child := range typed.Objects {
			found = append(found, texts(child)...)
		}
		return found
	case *container.Scroll:
		return texts(typed.Content)
	// The scrolling area of this screen is the keyboard's rather than the
	// toolkit's own, and it is a type of its own, so a walk that knew only the
	// toolkit's stopped at it and reported an empty screen.
	case *parts.KeyScroll:
		return texts(typed.Content)
	}
	return nil
}

// The screen says the window draws with the software renderer when it does
// (B-414), and says nothing of it when the graphics driver draws, which is
// almost everywhere. The second half is the canary of the first: a screen that
// always said it would pass the first half alone.
func TestTheScreenSaysWhenTheWindowDrawsInSoftware(t *testing.T) {
	sentence := i18n.English.T(i18n.WindowSoftwareDrawing)
	if !has(texts(Content(i18n.English, license.Current(), version.Current(), true, nil, nil)), sentence) {
		t.Errorf("a window drawn by the software renderer does not say so on the About screen: %q", sentence)
	}
	if has(screen(t), sentence) {
		t.Errorf("a window the graphics driver draws says it is drawn by the software renderer")
	}
}

func screen(t *testing.T) []string {
	t.Helper()
	return texts(Content(i18n.English, license.Current(), version.Current(), false, nil, nil))
}

func has(all []string, want string) bool {
	for _, one := range all {
		if strings.Contains(one, want) {
			return true
		}
	}
	return false
}

// The screen has to say the same things the two flags say, because that is the
// parity rule made real rather than declared. A window that shows fewer facts
// than --license is a second answer to one question.
func TestTheScreenCarriesEveryFactTheLicenceFlagPrints(t *testing.T) {
	all := screen(t)
	report := license.Current()

	for _, want := range []string{report.Author, report.Code.License, report.Spec.License, report.Spec.Scope} {
		if !has(all, want) {
			t.Errorf("the screen does not carry %q, which --license prints", want)
		}
	}
	if !has(all, report.Notices.GeneratedData) {
		t.Error("the screen does not carry the notice that generated data belongs to the user")
	}
	if !has(all, report.Notices.SyntheticData) {
		t.Error("the screen does not carry the notice that every value is synthetic")
	}
}

// Four versions at once, because a compliance answer is about one build and
// three of the four are contracts somebody else depends on.
//
// # Why the label is checked and not only the value
//
// Three of the four versions read 0.1 today. A test looking for the values
// alone would stay green with one of them missing from the screen, because a
// sibling still carries the same string. A mutation removing the recipe row
// proved exactly that, which is what mutations are for.
func TestTheScreenCarriesAllFourVersions(t *testing.T) {
	all := screen(t)
	versions := version.Current()
	for _, one := range []struct {
		label i18n.Key
		value string
	}{
		{i18n.AboutVersionTool, versions.Tool},
		{i18n.AboutVersionCatalog, versions.Catalog},
		{i18n.AboutVersionManifest, versions.Manifest},
		{i18n.AboutVersionRecipe, versions.Recipe},
	} {
		label := i18n.English.T(one.label)
		if !has(all, label) {
			t.Errorf("the row %q is not on the screen", label)
		}
		if !has(all, one.value) {
			t.Errorf("the version %q for %q is not on the screen", one.value, label)
		}
	}
}

// Every linked module has to be named with the day its licence was read. A
// licence is a property of a version rather than of a project, so a name
// without a date is a fact somebody remembered.
func TestEveryLinkedModuleIsNamedWithTheDayItsLicenceWasRead(t *testing.T) {
	all := screen(t)
	report := license.Current()
	if len(report.Components) == 0 {
		t.Fatal("no components in the register, so this test would pass by checking nothing")
	}
	for _, component := range report.Components {
		for _, want := range []string{component.Module, component.Version, component.License, component.Checked} {
			if !has(all, want) {
				t.Errorf("the screen does not carry %q for %s", want, component.Module)
			}
		}
	}
}

// The reference data is on the screen as the license flag prints it: each
// source named with its version, its terms, the day they were read and every
// list taken from it.
//
// 🔴 The screen went without this section from its first commit until
// 2026-09-23, while --license printed it and the heading waited in the
// catalogue, and every test here was green because none asked.
func TestEveryReferenceDataSourceIsNamedWithItsLists(t *testing.T) {
	all := screen(t)
	sources := license.BySource(license.Current().Data)
	if len(sources) == 0 {
		t.Fatal("no reference data in the register, so this test would pass by checking nothing")
	}
	for _, source := range sources {
		for _, want := range []string{source.Project, source.Version, source.License, source.Checked, source.Source} {
			if !has(all, want) {
				t.Errorf("the screen does not carry %q for %s", want, source.Project)
			}
		}
		for _, list := range source.Lists {
			if !has(all, list) {
				t.Errorf("the screen does not name the list %s read from %s", list, source.Project)
			}
		}
	}
}

// The typeface and the icon set are named with their terms, as the license
// flag names them.
func TestEveryEmbeddedAssetIsNamedWithItsTerms(t *testing.T) {
	all := screen(t)
	report := license.Current()
	if len(report.Embedded) == 0 {
		t.Fatal("no embedded asset in the register, so this test would pass by checking nothing")
	}
	for _, asset := range report.Embedded {
		for _, want := range []string{asset.Asset, asset.Version, asset.License, asset.Checked, asset.Source} {
			if !has(all, want) {
				t.Errorf("the screen does not carry %q for %s", want, asset.Asset)
			}
		}
	}
}

// No screen holds its own sentences, and no label renders as a key.
//
// # Why the shape is checked rather than the list of known keys
//
// The first version compared each label against i18n.Keys, which answers only
// for keys somebody declared. A mutation asking for a key that is on no list
// went unnoticed: it rendered as a dotted identifier on screen and the test
// stayed green, because the identifier was not one it knew to look for. What
// matters to a person reading the window is the shape, so that is what this
// looks at.
//
// 🔴 Every part after a dot starts with a letter, because every key does and a
// version does not. On 2026-09-23 the versions of the modules became texts of
// their own, and "v0.0.1" matched the looser shape thirty six times. No key in
// the catalogue has a part starting with a digit, which was counted before the
// shape was narrowed, so nothing the looser shape caught is let through.
func TestNoLabelRendersAsItsOwnKey(t *testing.T) {
	looksLikeAKey := regexp.MustCompile(`^[a-z][a-z0-9]*(\.[a-z][a-z0-9-]*)+$`)
	for _, one := range screen(t) {
		if looksLikeAKey.MatchString(one) {
			t.Errorf("the label %q reads as a translation key rather than as a sentence", one)
		}
	}
}

// A screen built against a build carrying nothing has to say so rather than
// show an empty space, which reads as a layout fault.
//
// All three lists, because each has its own sentence and a list forgotten here
// is the one that would show a head over nothing.
func TestAnEmptyRegisterIsStatedRatherThanLeftBlank(t *testing.T) {
	empty := license.Report{}
	all := texts(Content(i18n.English, empty, version.Current(), false, nil, nil))
	for _, key := range []i18n.Key{i18n.AboutEmptyComponents, i18n.AboutEmptyData, i18n.AboutEmptyEmbedded} {
		if !has(all, i18n.English.T(key)) {
			t.Errorf("a build with an empty register shows nothing where %q belongs", i18n.English.T(key))
		}
	}
	// A count of nought beside a heading over that sentence would say it twice.
	for _, nought := range []string{"0 modules", "0 lists", "0 assets"} {
		if has(all, nought) {
			t.Errorf("an empty list is counted as %q beside a sentence already saying it is empty", nought)
		}
	}
}

// buttons walks the built screen and collects every control on it.
func buttons(object fyne.CanvasObject) []*parts.Button {
	switch typed := object.(type) {
	case *parts.Button:
		return []*parts.Button{typed}
	case *fyne.Container:
		var found []*parts.Button
		for _, child := range typed.Objects {
			found = append(found, buttons(child)...)
		}
		return found
	case *container.Scroll:
		return buttons(typed.Content)
	case *parts.KeyScroll:
		return buttons(typed.Content)
	}
	return nil
}

// The screen offers the report in the form a machine reads.
//
// 🔴 This is the test the parity register names for --json, and naming one is
// the register's own condition: a claim that the window can do something, with
// no test behind it, is a sentence rather than a guarantee. DEC-30 names the
// reader and it is a compliance department, whose tool reads documents rather
// than paragraphs, so showing the same facts on screen answers nobody.
//
// What is checked is that the bytes are the report, not merely that a button
// exists. A button writing an empty file would satisfy every weaker version of
// this and hand a scanner nothing.
func TestTheScreenOffersTheReportInTheFormAMachineReads(t *testing.T) {
	var written []byte
	screen := Content(i18n.English, Report(), version.Current(), false,
		func(contents []byte) { written = contents }, nil)

	var save *parts.Button
	for _, button := range buttons(screen) {
		if button.Text == i18n.English.T(i18n.AboutSaveJSON) {
			save = button
		}
	}
	if save == nil {
		t.Fatalf("no control on this screen offers the report as JSON, so the reader named in "+
			"DEC-30 has nothing to read. The label looked for was %q",
			i18n.English.T(i18n.AboutSaveJSON))
	}
	save.OnTapped()

	var read license.Report
	if err := json.Unmarshal(written, &read); err != nil {
		t.Fatalf("what the button wrote is not JSON: %v", err)
	}
	if read.Tool != license.Current().Tool || read.Code.License != license.Current().Code.License {
		t.Errorf("the file says tool %q under %q and this build is %q under %q, so the report "+
			"describes something other than what produced it",
			read.Tool, read.Code.License, license.Current().Tool, license.Current().Code.License)
	}
	// The command line's report carries its notices in this field, and the
	// window's carries the window's. Without them the file a compliance
	// department reads would say less than the one the command line writes.
	// The check is asked for both of its answers first.
	if why := reportNoticesProblem(license.Report{ThirdPartyNotices: "notices"}, "notices"); why != "" {
		t.Fatalf("a report carrying the notices was refused: %s", why)
	}
	if reportNoticesProblem(license.Report{}, "notices") == "" {
		t.Fatal("a report carrying no notices passed for one that carries them")
	}
	if why := reportNoticesProblem(read, license.WindowNotices); why != "" {
		t.Error(why)
	}
}

// reportNoticesProblem says why a report does not carry the window's notices,
// or nothing when it carries them whole.
func reportNoticesProblem(read license.Report, notices string) string {
	if notices != "" && read.ThirdPartyNotices == notices {
		return ""
	}
	return fmt.Sprintf("the report this screen writes carries %d characters of notices and the window's "+
		"notices are %d, so the machine readable form leaves out what the window carries",
		len(read.ThirdPartyNotices), len(notices))
}

// A screen with no way to write says so by being unable to be pressed.
//
// A control that does nothing is a promise, and this one is handed nothing to
// do whenever the screen is being rendered rather than shown.
func TestWithNoWayToWriteTheControlIsNotPressable(t *testing.T) {
	for _, button := range buttons(Content(i18n.English, license.Current(), version.Current(), false, nil, nil)) {
		if button.Text == i18n.English.T(i18n.AboutSaveJSON) && !button.Disabled() {
			t.Error("the control offering the report can be pressed on a screen with no way to " +
				"write, so pressing it does nothing and says nothing")
		}
	}
}

// The notices are the one thing on this screen whose absence is a licence
// breach rather than a missing feature.
//
// 🔴 Why this asks for the BYTES rather than for a button
//
// Measured on 2026-09-08: the embed directive alone put nothing in the built
// binary, because the linker drops an embedded set that no reachable code
// reads, and the tests beside the asset package stayed green throughout since a
// test binary reads them. A test that only found a control would have stayed
// green as well. What makes this one hold the rule is that it presses the
// control and reads what came out, which is the same code path the built binary
// links.
//
// What has to come out is the whole of the window's notices, and under them a
// section for the project of every typeface and icon set the register names.
// That the licence text stands in full in that section is held by the guard in
// internal/guard, which reads the same file.
func TestTheScreenWritesTheNoticesOfWhatItEmbeds(t *testing.T) {
	// The check is asked first about a file that is right and two that are
	// not, so that a check that always agrees cannot pass for it.
	planted := "# Notices\n\n## someone/face\n\nSome terms.\n"
	face := []license.Asset{{Asset: "A face", Project: "someone/face"}}
	if problems := noticesFileProblems([]byte(planted), planted, face); problems != nil {
		t.Fatalf("a file equal to the notices, with a section for the asset, was refused: %v", problems)
	}
	if noticesFileProblems(nil, planted, face) == nil {
		t.Fatal("an empty file passed for the notices")
	}
	if noticesFileProblems([]byte("# Notices\n"), "# Notices\n", face) == nil {
		t.Fatal("notices with no section for an embedded asset passed")
	}

	var written []byte
	screen := Content(i18n.English, Report(), version.Current(), false,
		func(contents []byte) { written = contents }, nil)

	var save *parts.Button
	for _, button := range buttons(screen) {
		if button.Text == i18n.English.T(i18n.AboutSaveNotices) {
			save = button
		}
	}
	if save == nil {
		t.Fatalf("no control on this screen offers the licence notices, so a person who "+
			"received the binary cannot read the terms of what travelled inside it. "+
			"The label looked for was %q", i18n.English.T(i18n.AboutSaveNotices))
	}
	save.OnTapped()

	if len(license.Embedded) == 0 {
		t.Fatal("the register names no embedded asset, so this test proves nothing about a " +
			"build that carries a typeface and an icon set")
	}
	for _, problem := range noticesFileProblems(written, license.WindowNotices, license.Embedded) {
		t.Error(problem)
	}
}

// noticesFileProblems says what is wrong with the file the notice button
// wrote: not the window's notices, or no section in them for the project of an
// embedded asset.
func noticesFileProblems(written []byte, notices string, assets []license.Asset) []string {
	if notices == "" || string(written) != notices {
		return []string{fmt.Sprintf("the file this screen writes is %d bytes and the window's notices "+
			"are %d, so what reaches whoever received the binary is not THIRD-PARTY-NOTICES-GUI.md",
			len(written), len(notices))}
	}
	var problems []string
	for _, asset := range assets {
		if !bytes.Contains(written, []byte("\n## "+asset.Project+"\n")) {
			problems = append(problems, fmt.Sprintf("the file this screen writes has no section for %s, "+
				"the project of %s, so a reader cannot find the terms of the files it covers", asset.Project, asset.Asset))
		}
	}
	return problems
}

// With nothing to write, or nowhere to write it, the control says so by being
// unable to be pressed, which is the rule already stated for the report beside
// it. The report the register alone gives carries no notices, which is what a
// screen rendered against another build may be handed.
func TestWithNoWayToWriteTheNoticeControlIsNotPressable(t *testing.T) {
	cases := map[string]fyne.CanvasObject{
		"no way to write":    Content(i18n.English, Report(), version.Current(), false, nil, nil),
		"no notices to save": Content(i18n.English, license.Current(), version.Current(), false, func([]byte) {}, nil),
	}
	for name, screen := range cases {
		found := false
		for _, button := range buttons(screen) {
			if button.Text != i18n.English.T(i18n.AboutSaveNotices) {
				continue
			}
			found = true
			if !button.Disabled() {
				t.Errorf("%s: the control offering the notices can be pressed, so pressing it does "+
					"nothing and says nothing", name)
			}
		}
		if !found {
			t.Errorf("%s: no control offers the notices, so this test checked nothing", name)
		}
	}
}
