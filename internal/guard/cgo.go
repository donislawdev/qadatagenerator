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
// Only build lines (#cgo) and system headers (#include <...>). C written there
// would be a function with an innocent name doing anything, and the list below
// would then name what is called rather than what runs. A header of our own
// is refused for the same reason.
//
// # What it does not do
//
// It reads Go files. C, C++ and assembly files beside them, which the Go tool
// compiles into the package with no import at all, and object files such as
// the rsrc_windows_amd64.syso of each binary are not read. A #cgo line can pass
// a flag that changes what a name means, and that is left to whoever reads the
// line. A name reached through a function value is invisible, as it is to the
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
// neither a #cgo line nor a system header.
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
				if line == "" || strings.HasPrefix(line, "#cgo ") || systemHeader(line) {
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

// systemHeader reports whether a line includes a header from the system's
// include path, written between angle brackets.
func systemHeader(line string) bool {
	header, isInclude := strings.CutPrefix(line, "#include <")
	header, closed := strings.CutSuffix(header, ">")
	return isInclude && closed && header != "" && !strings.ContainsAny(header, "<>\"")
}
