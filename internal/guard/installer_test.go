package guard

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// # What this defends (B-419 (e), DEC-331)
//
// The Windows installer: the template WiX builds it from, the script that
// fills and builds it, the test that installs it on a runner, and the signing
// step that builds it from the signed archive. What the installer DOES is
// asked by check_msi.ps1, on a runner for every build of a release and on
// Windows Server 2025 before these files were committed. These guards hold
// what neither of those sees coming: a line those measurements rest on, a
// value that must never change, one name written in several places, and one
// rule for a release candidate. They read the files as text and run nothing,
// so they run on every system, WiX or not.
//
// Adapted on 2026-11-21 from the installer guards of Cert Doctor, a project of
// the same owner under the same licence.
const (
	installerTemplate = ".github/msi/qadg-setup.wxs.in"
	installerScript   = ".github/scripts/build_msi.py"
	installerCheck    = ".github/scripts/check_msi.ps1"
	commandResources  = "cmd/qadg/qadg.rc"
	windowWords       = "internal/adapter/i18n/locales/en.yaml"
)

// installerFile is the installer as the release note names it, with <version>
// where the release puts its version.
const installerFile = "`qadg-setup_<version>_windows_amd64.msi`"

// installerUpgradeCode is the product's identity in Windows Installer, for
// good (DEC-331 decision 5). Every machine that has the program finds the
// version it has through it, so a new one would leave the old install beside
// the new one on every one of them. Written here a second time on purpose:
// this is the copy that does not move when the script does.
const installerUpgradeCode = "929C63C8-F396-4781-B4F3-33B8A485BAA1"

const wixNamespace = "http://wixtoolset.org/schemas/v4/wxs"

// wxsElement is one element of the installer source: its name, its namespace,
// its attributes and the element it sits in. Comments are not elements, so a
// word in the explanation of a line never counts as the line.
type wxsElement struct {
	name, space string
	attrs       map[string]string
	parent      int
}

func wxsElements(source string) ([]wxsElement, error) {
	dec := xml.NewDecoder(strings.NewReader(source))
	var found []wxsElement
	open := []int{-1}
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return found, nil
		}
		if err != nil {
			return nil, err
		}
		switch el := tok.(type) {
		case xml.StartElement:
			attrs := map[string]string{}
			for _, a := range el.Attr {
				key := a.Name.Local
				if a.Name.Space != "" {
					key = a.Name.Space + ":" + a.Name.Local
				}
				attrs[key] = a.Value
			}
			found = append(found, wxsElement{el.Name.Local, el.Name.Space, attrs, open[len(open)-1]})
			open = append(open, len(found)-1)
		case xml.EndElement:
			open = open[:len(open)-1]
		}
	}
}

func elementsNamed(els []wxsElement, name string) []wxsElement {
	var out []wxsElement
	for _, e := range els {
		if e.name == name {
			out = append(out, e)
		}
	}
	return out
}

// wantAttrs adds a finding for every attribute of e that is not the value the
// measurement rests on.
func wantAttrs(found []string, what string, e wxsElement, want map[string]string, why string) []string {
	keys := make([]string, 0, len(want))
	for k := range want {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, attr := range keys {
		if got := e.attrs[attr]; got != want[attr] {
			found = append(found, fmt.Sprintf("%s has %s=%q, want %q: %s", what, attr, got, want[attr], why))
		}
	}
	return found
}

// TestTheInstallerIsTheShapeThatWasMeasured holds the template to the lines
// the measurements rest on. Each one changed would still build a working
// installer, which is why it needs a guard rather than a build: what breaks is
// an upgrade, an uninstall or another account's Start menu, on somebody else's
// machine.
func TestTheInstallerIsTheShapeThatWasMeasured(t *testing.T) {
	root := rootOrFail(t)
	template := readRepoFile(t, root, installerTemplate)
	for _, c := range []struct{ old, replacement, want string }{
		{`<Property Id="MSIRESTARTMANAGERCONTROL" Value="Disable" />`, ``, "MSIRESTARTMANAGERCONTROL"},
		{`Scope="perMachine"`, `Scope="perUser"`, `Scope="perUser"`},
		{`Schedule="afterInstallExecute"`, `Schedule="afterInstallValidate"`, "the major upgrade has Schedule"},
		{`System="yes" Permanent="no"`, `System="yes" Permanent="yes"`, "the PATH entry has Permanent"},
		{`<MediaTemplate EmbedCab="yes" />`, `<MediaTemplate EmbedCab="yes" /><CustomAction Id="x" />`, "<CustomAction>"},
		{`Target="[INSTALLFOLDER]{{WINDOW_EXE}}"`, `Target="[INSTALLFOLDER]qadg.exe"`, "the shortcut has Target"},
		{`UpgradeCode="{{UPGRADE_CODE}}"`, `UpgradeCode="` + installerUpgradeCode + `"`, "the package has UpgradeCode"},
		{`<Wix xmlns="http://wixtoolset.org/schemas/v4/wxs">`,
			`<Wix xmlns="http://wixtoolset.org/schemas/v4/wxs" xmlns:util="http://wixtoolset.org/schemas/v4/wxs/util">`,
			"brings in the namespace"},
	} {
		requireFinding(t, checkInstallerShape(planted(t, installerTemplate, template, c.old, c.replacement)), c.want)
	}
	requireNone(t, checkInstallerShape(template), "the installer template")
}

func checkInstallerShape(source string) []string {
	els, err := wxsElements(source)
	if err != nil {
		return []string{installerTemplate + " is not XML: " + err.Error()}
	}
	var found []string
	if len(els) < 10 {
		found = append(found, fmt.Sprintf("read %d element(s), so this is not reading the installer source", len(els)))
	}
	// No extension, no custom action, nothing that ends a program, no dialogs
	// but Windows Installer's own (DEC-331 decision 3): a run in progress is cut
	// off by nothing, and the package holds only our files.
	for _, e := range els {
		if e.space != wixNamespace {
			found = append(found, fmt.Sprintf("<%s> is from %q, an extension of WiX, and the installer carries none", e.name, e.space))
		}
		for key := range e.attrs {
			if strings.HasPrefix(key, "xmlns:") {
				found = append(found, fmt.Sprintf("<%s> brings in the namespace %s, and the installer uses no extension", e.name, key))
			}
		}
		switch e.name {
		case "CustomAction", "CloseApplication", "InstallExecuteSequence", "UI", "UIRef", "Binary":
			found = append(found, fmt.Sprintf("the installer has a <%s>. It runs no action of its own, ends nothing that is "+
				"running, and its only dialogs are Windows Installer's", e.name))
		}
	}

	pkg := elementsNamed(els, "Package")
	if len(pkg) != 1 {
		return append(found, fmt.Sprintf("the template has %d <Package>, and it is one package", len(pkg)))
	}
	found = wantAttrs(found, "the package", pkg[0], map[string]string{
		"Name": "{{APP_NAME}}", "UpgradeCode": "{{UPGRADE_CODE}}", "Scope": "perMachine",
	}, "the name and the UpgradeCode come from build_msi.py, where a guard pins them, and the package installs "+
		"for the machine, so qadg is on PATH for a build agent and the window in every account's Start menu")

	upgrade := elementsNamed(els, "MajorUpgrade")
	if len(upgrade) != 1 {
		found = append(found, fmt.Sprintf("the template has %d <MajorUpgrade>, and the next release replaces this one", len(upgrade)))
	} else {
		found = wantAttrs(found, "the major upgrade", upgrade[0], map[string]string{
			"Schedule": "afterInstallExecute", "AllowSameVersionUpgrades": "yes",
		}, "the old version goes after the new files are in place, and a rebuild of one version replaces the first build")
	}

	// Measured on Windows Server 2025 on 2026-11-21: without this line an
	// install over a running qadg ends with 1601. The OLD package's properties
	// decide an upgrade, so this is in the package from the first one.
	var manager []wxsElement
	for _, e := range elementsNamed(els, "Property") {
		if e.attrs["Id"] == "MSIRESTARTMANAGERCONTROL" {
			manager = append(manager, e)
		}
	}
	if len(manager) != 1 || manager[0].attrs["Value"] != "Disable" {
		found = append(found, fmt.Sprintf("the package sets MSIRESTARTMANAGERCONTROL %d time(s), and it sets it once, "+
			"to Disable, or an install over a running qadg fails", len(manager)))
	}

	env := elementsNamed(els, "Environment")
	if len(env) != 1 {
		found = append(found, fmt.Sprintf("the template has %d <Environment>, and it sets one PATH entry", len(env)))
	} else {
		found = wantAttrs(found, "the PATH entry", env[0], map[string]string{
			"Name": "PATH", "System": "yes", "Part": "last", "Value": "[INSTALLFOLDER]", "Permanent": "no", "Action": "set",
		}, "the folder goes on the machine's PATH after everything already there, and comes off again at uninstall")
	}

	shortcuts := elementsNamed(els, "Shortcut")
	if len(shortcuts) != 1 {
		found = append(found, fmt.Sprintf("the template has %d shortcut(s). The window has one, and the command line none", len(shortcuts)))
	} else {
		found = wantAttrs(found, "the shortcut", shortcuts[0], map[string]string{
			"Target": "[INSTALLFOLDER]{{WINDOW_EXE}}", "WorkingDirectory": "INSTALLFOLDER", "Name": "{{APP_NAME}}",
		}, "it starts the window, by the name build_msi.py gives it, in the folder it is installed in")
	}

	folder := -1
	for i, e := range els {
		if e.name == "Directory" && e.attrs["Id"] == "INSTALLFOLDER" {
			folder = i
		}
	}
	if folder < 0 || els[folder].parent < 0 || els[els[folder].parent].attrs["Id"] != "ProgramFiles64Folder" ||
		els[folder].attrs["Name"] != "{{APP_NAME}}" {
		found = append(found, `the install folder is not Program Files\{{APP_NAME}}, where check_msi.ps1 looks for its files`)
	}
	files := elementsNamed(els, "Files")
	if len(files) != 1 || files[0].attrs["Include"] != `$(PayloadDir)\**` {
		found = append(found, `the template does not take the window's archive whole with one <Files Include="$(PayloadDir)\**">`)
	}
	sort.Strings(found)
	return found
}

var (
	pythonConstant  = `(?m)^%s = "([^"]*)"$`
	resourceValue   = `VALUE "%s", "([^"]*)"`
	windowTitleWord = regexp.MustCompile(`(?m)^window\.title: (.+)$`)
	releaseWixPin   = regexp.MustCompile(`dotnet tool install --global wix --version (\S+)`)
)

// pyConst is the value of NAME = "..." at the top of a Python script, or "".
func pyConst(script, name string) string {
	m := regexp.MustCompile(fmt.Sprintf(pythonConstant, regexp.QuoteMeta(name))).FindStringSubmatch(script)
	if m == nil {
		return ""
	}
	return m[1]
}

// resourceString is one string of the version block of a resource script, or
// "".
func resourceString(script, key string) string {
	m := regexp.MustCompile(fmt.Sprintf(resourceValue, regexp.QuoteMeta(key))).FindStringSubmatch(script)
	if m == nil {
		return ""
	}
	return m[1]
}

func installerFiles(t *testing.T) map[string]string {
	t.Helper()
	root := rootOrFail(t)
	files := map[string]string{}
	for _, rel := range []string{installerScript, installerCheck, signingScript, releaseWorkflow,
		commandResources, windowResources, windowWords, draftNotes, verifyWorkflow} {
		files[rel] = readRepoFile(t, root, rel)
	}
	return files
}

// TestTheInstallerIsNamedTheSameEverywhere holds every place that writes one
// of the installer's names to the one place it lives (DEC-331): the upgrade
// code, the file a person downloads, the product's name and its publisher,
// and the WiX it is built with.
func TestTheInstallerIsNamedTheSameEverywhere(t *testing.T) {
	files := installerFiles(t)
	for _, c := range []struct{ file, old, replacement, want string }{
		{installerScript, `UPGRADE_CODE = "` + installerUpgradeCode + `"`, `UPGRADE_CODE = "00000000-0000-0000-0000-000000000000"`, "UPGRADE_CODE in"},
		{windowWords, "window.title: QA Data Generator", "window.title: QA Data Gen", "the window's title"},
		{windowResources, `VALUE "ProductName", "QA Data Generator"`, `VALUE "ProductName", "qadg-gui"`, "ProductName in " + windowResources},
		{commandResources, `VALUE "CompanyName", "DonislawDev"`, `VALUE "CompanyName", "donislawdev"`, "CompanyName in " + commandResources},
		{signingScript, `INSTALLER = "qadg-setup_%s_windows_amd64.msi"`, `INSTALLER = "qadg_%s_windows_amd64.msi"`, "INSTALLER in"},
		{releaseWorkflow, "--global wix --version 5.0.2", "--global wix --version 7.0.1", "the WiX " + releaseWorkflow},
		{installerCheck, "$name = 'QA Data Generator'", "$name = 'QA Data Gen'", installerCheck + " does not hold"},
		{draftNotes, installerFile, "`qadg_<version>_windows_amd64.msi`", draftNotes + " does not hold"},
		{verifyWorkflow, `'PFiles64\QA Data Generator'`, `'PFiles64\QA Data Gen'`, verifyWorkflow + " does not unpack"},
		{verifyWorkflow, `expected+=("qadg-setup_${version}_windows_amd64.msi")`, `expected+=("qadg_setup_${version}_windows_amd64.msi")`, verifyWorkflow + " does not hold"},
	} {
		broken := map[string]string{}
		for k, v := range files {
			broken[k] = v
		}
		broken[c.file] = planted(t, c.file, files[c.file], c.old, c.replacement)
		requireFinding(t, checkInstallerNames(broken), c.want)
	}
	requireNone(t, checkInstallerNames(files), "the installer's names")
}

func checkInstallerNames(files map[string]string) []string {
	var found []string
	script := files[installerScript]
	same := func(what, got, want string) {
		if got != want {
			found = append(found, fmt.Sprintf("%s is %q, and it has to be %q", what, got, want))
		}
	}
	same("UPGRADE_CODE in "+installerScript, pyConst(script, "UPGRADE_CODE"), installerUpgradeCode)
	same("NAME in "+installerScript, pyConst(script, "NAME"), "qadg-setup_{version}_windows_amd64.msi")
	same("INSTALLER in "+signingScript, pyConst(files[signingScript], "INSTALLER"), "qadg-setup_%s_windows_amd64.msi")

	// The product's name (DEC-331 decision 1): what Windows calls the program,
	// the window's title, the product name both programs carry, and where the
	// installer test looks for the installed files.
	app, publisher := pyConst(script, "APP_NAME"), pyConst(script, "PUBLISHER")
	if app == "" || publisher == "" {
		found = append(found, "APP_NAME or PUBLISHER is not read out of "+installerScript)
	}
	title := ""
	if m := windowTitleWord.FindStringSubmatch(files[windowWords]); m != nil {
		title = strings.TrimSpace(m[1])
	}
	same("the window's title in "+windowWords, title, app)
	for _, res := range []string{commandResources, windowResources} {
		same("ProductName in "+res, resourceString(files[res], "ProductName"), app)
		same("CompanyName in "+res, resourceString(files[res], "CompanyName"), publisher)
	}
	for path, text := range map[string]string{
		installerCheck:  "$name = '" + app + "'",
		releaseWorkflow: `$msi = "qadg-setup_$($env:VERSION)_windows_amd64.msi"`,
		draftNotes:      installerFile,
		verifyWorkflow:  `expected+=("qadg-setup_${version}_windows_amd64.msi")`,
	} {
		if !strings.Contains(files[path], text) {
			found = append(found, fmt.Sprintf("%s does not hold %q, so it names the installer otherwise", path, text))
		}
	}
	// Where the verifier finds the installed files in an administrative
	// install of the published installer: the folder the template names after
	// the product, under the source name of Program Files.
	if unpacked := `$inside = Join-Path $image 'PFiles64\` + app + `'`; !strings.Contains(files[verifyWorkflow], unpacked) {
		found = append(found, fmt.Sprintf("%s does not unpack the installer into %q, the folder named after the product", verifyWorkflow, unpacked))
	}
	if !strings.Contains(files[installerCheck], "$publisher = '"+publisher+"'") {
		found = append(found, installerCheck+" does not hold the publisher "+publisher)
	}

	wix := ""
	if m := releaseWixPin.FindStringSubmatch(files[releaseWorkflow]); m != nil {
		wix = m[1]
	}
	same("the WiX "+releaseWorkflow+" installs", wix, pyConst(script, "WIX_VERSION"))
	sort.Strings(found)
	return found
}

// TestEveryPlaceAsksTheSameQuestionOfACandidate holds one rule for a release
// candidate in every place that asks: a hyphen in the version. Windows
// Installer reads only the three numbers, so a candidate's installer would
// take the release's own version, and the release would then not replace it.
func TestEveryPlaceAsksTheSameQuestionOfACandidate(t *testing.T) {
	files := installerFiles(t)
	for _, c := range []struct{ file, old, replacement, want string }{
		{signingScript, `return "-" in version`, `return "-rc" in version`, signingScript + " no longer builds"},
		{installerScript, `if "-" in tag:`, `if "-rc" in tag:`, installerScript + " no longer refuses"},
		{releaseWorkflow, `$env:VERSION.Contains('-')`, `$env:VERSION.Contains('-rc')`, releaseWorkflow + " no longer installs"},
		{verifyWorkflow, `if [[ "$TAG" == *-* ]]`, `if [[ "$TAG" == *-rc* ]]`, verifyWorkflow + " no longer expects"},
	} {
		broken := map[string]string{}
		for k, v := range files {
			broken[k] = v
		}
		broken[c.file] = planted(t, c.file, files[c.file], c.old, c.replacement)
		requireFinding(t, checkCandidateRule(broken), c.want)
	}
	requireNone(t, checkCandidateRule(files), "the candidate rule")
}

func checkCandidateRule(files map[string]string) []string {
	var found []string
	for _, c := range []struct{ path, text, what string }{
		{installerScript, `if "-" in tag:`, "refuses a candidate's installer"},
		{signingScript, `return "-" in version`, "builds no installer for a candidate"},
		{releaseWorkflow, `$env:VERSION.Contains('-')`, "installs no installer for a candidate"},
		{releaseWorkflow, `if [[ "${GITHUB_REF_NAME}" == *-* ]]`, "marks a candidate as a pre-release"},
		{verifyWorkflow, `if [[ "$TAG" == *-* ]]`, "expects no installer for a candidate"},
	} {
		if !strings.Contains(files[c.path], c.text) {
			found = append(found, fmt.Sprintf("%s no longer %s by %q", c.path, c.what, c.text))
		}
	}
	return found
}

// TestTheSigningBuildsTheInstallerAfterTheCardAndBeforeTheSums holds the
// order the signing script builds the installer in: asked whether it can be
// built before anything is signed, built after the programs are signed, because
// an installer made before would carry unsigned copies, and before the
// checksums, which have to list it. It is signed with the product's name and
// built from the tree of the tag. The release waits for the job that installs
// it on a runner, and that job asks for 3010 while qadg runs, which is what
// the line the shape guard holds is for.
func TestTheSigningBuildsTheInstallerAfterTheCardAndBeforeTheSums(t *testing.T) {
	files := installerFiles(t)
	signing, release, check := files[signingScript], files[releaseWorkflow], files[installerCheck]

	early := "        print(\"\\n[7/%d] building and signing the Windows installer\" % steps)\n" +
		"        build_installer(tree, work, version, thumbprint, windows_pin, signtool, args.dry_run)\n"
	card := "        print(\"\\n[6/%d] signing the Windows programs and libraries with the card\" % steps)\n"
	moved := planted(t, signingScript, signing, early, "")
	moved = planted(t, signingScript, moved, card, early+card)
	requireFinding(t, checkInstallerSigning(moved, release, check), "out of order")
	requireFinding(t, checkInstallerSigning(planted(t, signingScript, signing, `"/d", name`, `"/v"`), release, check),
		`no longer holds "/d", name`)
	requireFinding(t, checkInstallerSigning(signing, planted(t, releaseWorkflow, release,
		"needs: [check, build, installer]", "needs: [check, build]"), check), "without waiting")
	requireFinding(t, checkInstallerSigning(signing, release, planted(t, installerCheck, check,
		"Check ($code -eq 3010)", "Check ($code -eq 3010 -or $code -eq 1601)")), "3010")

	requireNone(t, checkInstallerSigning(signing, release, check), "the installer's signing")
}

func checkInstallerSigning(signing, release, check string) []string {
	var found []string
	_, main, ok := strings.Cut(signing, "\ndef main(")
	if !ok {
		return []string{signingScript + " has no main"}
	}
	steps := []string{"check_installer(tree, version)", "signing_thumbprint(windows_pin)", "sign_macos(label",
		"sign_windows(work", "build_installer(tree, work", "write_checksums(work)"}
	at := -1
	for _, step := range steps {
		i := strings.Index(main, step)
		if i < 0 {
			found = append(found, fmt.Sprintf("main in %s never calls %s", signingScript, step))
			continue
		}
		if i < at {
			found = append(found, fmt.Sprintf("main in %s calls %s out of order: %s", signingScript, step, strings.Join(steps, ", then ")))
		}
		at = i
	}
	for _, text := range []string{`"/d", name`, `["git", "archive", "--format=tar", ref]`, `installer = INSTALLER % version`} {
		if !strings.Contains(signing, text) {
			found = append(found, fmt.Sprintf("%s no longer holds %s", signingScript, text))
		}
	}
	if !strings.Contains(release, "needs: [check, build, installer]") {
		found = append(found, releaseWorkflow+" opens the draft without waiting for the job that installs the installer")
	}
	if !strings.Contains(jobBlocks(release)["installer"], "& ./.github/scripts/check_msi.ps1") {
		found = append(found, releaseWorkflow+" has no installer job that runs "+installerCheck)
	}
	if strings.Count(check, "Check ($code -eq 3010)") != 1 {
		found = append(found, installerCheck+" no longer asks for exactly 3010 from the install over a running qadg")
	}
	return found
}
