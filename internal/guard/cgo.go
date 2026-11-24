package guard

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// CgoAllowed names the files that may call C, together with the names each of
// them may take from it.
//
// # Why C is gated like syscall
//
// A file that imports C can call any function the system's C library offers,
// sockets included, and can write a function of its own in the comment above
// the import. It is the same door as syscall and a wider one, and the window
// went through it on Windows precisely because syscall was refused. So the
// network rule refuses C as well, and an allowance is a file and the C names it
// may use, read as a pair, exactly as in SyscallAllowed.
//
// # What may stand in the comment above import "C"
//
// Only system headers (#include <...>) and #cgo lines passing the linker the
// flags in CgoLinkerFlags. C written there would be a function with an innocent
// name doing anything, and the list below would then name what is called rather
// than what runs. A header of our own is refused for the same reason. C files
// beside the Go ones are refused by BesideGo.
//
// # What it does not do
//
// A name reached through a function value is invisible, as it is to the
// syscall guard.
var CgoAllowed = map[string][]string{
	// Borrowing the console of the program that started the window and letting
	// it go again, asking whether a person can see this desktop, and the
	// dialog that says why the window did not open, on Windows (DEC-315,
	// DEC-316). The rest are the types and constants those calls take.
	"cmd/qadg-gui/console_windows.go": {
		"AttachConsole", "FreeConsole", "GetProcessWindowStation", "GetUserObjectInformationW", "MessageBoxW",
		"ATTACH_PARENT_PROCESS", "MB_ICONERROR", "MB_OK", "MB_SETFOREGROUND", "UOI_FLAGS", "WSF_VISIBLE",
		"DWORD", "HANDLE", "PVOID", "USEROBJECTFLAGS", "WCHAR",
	},
	// How many lines Windows is set to scroll for one notch of the mouse
	// wheel, so the window moves as far as every other program does (B-420).
	// Allowed by the owner on 2026-11-23. The rest are the type and the two
	// constants the one call takes and answers with.
	"cmd/qadg-gui/wheel_windows.go": {
		"SystemParametersInfoW", "SPI_GETWHEELSCROLLLINES", "WHEEL_PAGESCROLL", "PVOID", "UINT",
	},
	// Showing the folder a run wrote into in File Explorer, when a person
	// presses Open folder, with COM set up around the call as its
	// documentation asks (DEC-336). Allowed by the owner on 2026-11-24. The
	// rest are the constants and the type the three calls take.
	"cmd/qadg-gui/folder_windows.go": {
		"ShellExecuteW", "CoInitializeEx", "CoUninitialize",
		"COINIT_APARTMENTTHREADED", "COINIT_DISABLE_OLE1DDE", "SW_SHOWNORMAL", "WCHAR",
	},
}

// CgoLinkerFlags are the only flags a #cgo line may pass, and only to the
// linker.
//
// A flag for the C compiler can change what an allowed name means. -D can make
// it stand for another function and -include can bring in a header of our own,
// and the go command lets both through, because its list of allowed flags
// (cmd/go/internal/work/security.go) protects the machine that builds, not what
// the program calls. A linker flag can join a library or let a definition
// replace another, so each one is named here.
var CgoLinkerFlags = map[string]bool{
	// Makes the window a GUI program on Windows, so no console opens beside
	// it (DEC-315).
	"-Wl,--subsystem,windows": true,
	// Joins the library of COM, set up around the call that shows a folder in
	// File Explorer (DEC-336). MinGW does not join it by itself.
	"-lole32": true,
}

// cgoPackage is the import being controlled. It is not a package but the
// switch that hands the comment above it to a C compiler.
const cgoPackage = "C"

// cgoGate holds the product to CgoAllowed, and an allowed file to C written by
// nobody but the system.
var cgoGate = namedGate{importPath: cgoPackage, list: "CgoAllowed", beyondNames: cWrittenHere}

// packageMayCallC reports whether a package holds a file allowed to call C.
func packageMayCallC(pkg string) bool {
	return holdsAnAllowedFile(CgoAllowed, pkg)
}

// CgoUse reports every place that calls C outside what the list allows, in both
// directions, and every line above an allowed import "C" that is neither a
// build line nor a system header.
func CgoUse(root string, allowed map[string][]string) ([]string, error) {
	return cgoGate.use(root, allowed)
}

// cWrittenHere reports every line of the comment above import "C" that is
// neither a system header nor a #cgo line passing only CgoLinkerFlags.
//
// The comment is read the way cgo reads it, its markers taken off and nothing
// else, because the directives the standard comment text leaves out are lines
// cgo still hands to the C compiler.
func cWrittenHere(parsed *ast.File, name string) []string {
	var findings []string
	for _, declaration := range parsed.Decls {
		imports, isImport := declaration.(*ast.GenDecl)
		if !isImport || imports.Tok != token.IMPORT {
			continue
		}
		for _, spec := range imports.Specs {
			one, isSpec := spec.(*ast.ImportSpec)
			if !isSpec {
				continue
			}
			if imported, err := strconv.Unquote(one.Path.Value); err != nil || imported != cgoPackage {
				continue
			}
			// The comment cgo takes, by the rule in cmd/cgo/ast.go: the one above
			// the spec, or above the whole import when it holds only this spec.
			doc := one.Doc
			if doc == nil && len(imports.Specs) == 1 {
				doc = imports.Doc
			}
			for _, line := range strings.Split(asCgoReadsIt(doc), "\n") {
				line = strings.TrimSpace(line)
				if line == "" || systemHeader(line) {
					continue
				}
				if cgoLine(line) {
					findings = append(findings, flagsBeyond(line, name)...)
					continue
				}
				findings = append(findings, fmt.Sprintf("%s writes %q above import \"C\", and only #cgo lines "+
					"and system headers may stand there", name, line))
			}
		}
	}
	return findings
}

// asCgoReadsIt is the text of a comment as cgo hands it on: a line comment
// without its two slashes and with its line ending, a block comment without its
// markers.
func asCgoReadsIt(doc *ast.CommentGroup) string {
	if doc == nil {
		return ""
	}
	var text strings.Builder
	for _, comment := range doc.List {
		body := comment.Text
		if strings.HasPrefix(body, "//") {
			text.WriteString(body[2:] + "\n")
			continue
		}
		text.WriteString(strings.TrimSuffix(strings.TrimPrefix(body, "/*"), "*/"))
	}
	return text.String()
}

// cgoLine reports whether the go command reads a line as a #cgo line, by its
// rule in go/build (saveCgo): #cgo followed by a space or a tab.
func cgoLine(line string) bool {
	return len(line) >= 5 && line[:4] == "#cgo" && (line[4] == ' ' || line[4] == '\t')
}

// flagsBeyond reports what a #cgo line does beyond passing the linker the
// flags in CgoLinkerFlags.
//
// The line is split as go/build splits it: what stands before the colon ends
// with the setting, and anything before the setting only says on which systems
// it applies. A flag is compared whole, so one in quotes or holding ${SRCDIR}
// is refused rather than read the way the go command would read it.
func flagsBeyond(line, name string) []string {
	settings, flags, hasColon := strings.Cut(strings.TrimSpace(line[len("#cgo"):]), ":")
	words := strings.Fields(settings)
	if !hasColon || len(words) == 0 || words[len(words)-1] != "LDFLAGS" {
		return []string{fmt.Sprintf("%s writes %q above import \"C\", and a #cgo line may set only LDFLAGS",
			name, line)}
	}
	var findings []string
	for _, flag := range strings.Fields(flags) {
		if !CgoLinkerFlags[flag] {
			findings = append(findings, fmt.Sprintf("%s passes %s to the linker in a #cgo line, and only the "+
				"flags in guard.CgoLinkerFlags may be passed", name, flag))
		}
	}
	return findings
}

// systemHeader reports whether a line includes a header from the system's
// include path, written between angle brackets.
func systemHeader(line string) bool {
	header, isInclude := strings.CutPrefix(line, "#include <")
	header, closed := strings.CutSuffix(header, ">")
	return isInclude && closed && header != "" && !strings.ContainsAny(header, "<>\"")
}
