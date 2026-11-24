package guard

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ToolkitNetworkNames are the calls of the graphical toolkit that leave this
// machine.
//
// # Why this guard exists at all, and what it is not
//
// The import scan answers about our own packages, so it sees a network package
// we import and nothing else. The toolkit is a single import that carries
// net/http inside it: measured on 2026-08-12, the window closes over 328
// packages against 152 for the command line binary, and the root package of
// the toolkit imports net/http directly to serve resources over it.
//
// The owner settled what the unbreakable rule means here: it is a rule about
// the behaviour of the product, not about the dependency closure. A promise
// about behaviour needs a guard about behaviour, which is what this is. It
// does not claim the binary cannot reach the network. It claims that nothing
// we wrote asks it to, and that a line which starts asking has to be a
// deliberate act rather than an autocompletion.
//
// This is the same shape as the one allowance for reaching the operating
// system: a list of names rather than a blanket rule about a file, because a
// blanket allowance would let everything through and nobody would remember a
// year later why the file was exempt.
var ToolkitNetworkNames = map[string]string{
	"LoadResourceFromURLString": "fetches a resource over the network",
	"LoadResourceFromURL":       "fetches a resource over the network",
	"OpenURL":                   "hands a URL to the desktop, which opens it outside this process",
}

// WindowProgram is the program that opens the window. It links everything the
// window does, so a call it made would be the window's call as much as one in
// ViewRoot.
const WindowProgram = "cmd/qadg-gui"

// ToolkitAllowance is one file allowed one of the calls above, and the only
// addresses that file may name.
type ToolkitAllowance struct {
	Call      string
	Addresses []string
	// Scheme is set for a file that opens what a person chose rather than a
	// page named in it, and is then the one scheme an address built there may
	// carry. Such a file takes a path from its caller, so the rule against
	// taking one does not hold for it, and this rule stands in its place: each
	// address is built from parts, as a url.URL with this scheme written out,
	// and never parsed from text or changed after it is built. A path handed
	// in can then become a file address and nothing else.
	Scheme string
	Why    string
}

// ToolkitNetworkAllowed are the places in the window allowed to hand an
// address to the desktop: two pages fixed in one file (DEC-276), and the
// folder a run wrote into, on the systems where the toolkit opens it
// (DEC-336).
//
// # Why this is an exception and not a hole
//
// Untouchable rule 8 is about what the program does. A button that opens a page
// in the browser connects to nothing: the address goes to the system, which
// opens it in another program, and only when a person presses the button. The
// owner chose the two buttons on 2026-11-12, and what makes them safe to allow
// is that the addresses are fixed. So the allowance is for one call in one
// file, and that file may name these two addresses and no other, which the
// guard checks in both directions: a third address in the file is a finding,
// and an address listed here that the file no longer names is one as well.
//
// The file is also refused an exported function taking a string or a URL,
// because that is the shape of a way for a caller to hand in an address of its
// own, which is the one thing the allowance must not become.
var ToolkitNetworkAllowed = map[string]ToolkitAllowance{
	"internal/gui/outward/outward.go": {
		Call: "OpenURL",
		Addresses: []string{
			"https://github.com/donislawdev/qadatagenerator",
			"https://donislawdev.com/support/",
		},
		Why: "the source of this program and the page supporting its author, opened in the browser " +
			"when a person presses the button for one (DEC-276)",
	},
	// The folder a run wrote into, shown in the file manager when a person
	// presses Open folder, on macOS and Linux. Windows takes another way,
	// because there the toolkit did not open a folder named in letters outside
	// ASCII (measured 2026-11-24, DEC-336), so the file for Windows is in
	// CgoAllowed instead. Allowed by the owner on 2026-11-24.
	"cmd/qadg-gui/folder_other.go": {
		Call:   "OpenURL",
		Scheme: "file",
		Why: "the folder a run wrote into, opened in the file manager when a person presses " +
			"Open folder, on macOS and Linux (DEC-336)",
	},
}

// ToolkitNetwork reports every call in the window that would leave this
// machine, outside the files the register allows it in.
//
// It reads the syntax tree, so the name inside a comment or a message is not a
// finding. It matches on the selected name rather than on the package, because
// the toolkit is imported under several names across its own packages and a
// list of import aliases is a register somebody has to keep in step.
//
// It reads ViewRoot and WindowProgram both. Until 2026-11-12 it read the first
// alone, so the program opening the window could have made any of these calls
// unseen, and the one place allowed to make one would not have been the only
// place able to.
//
// # What it does not cover
//
// A call reached through a variable of an interface type, or through
// reflection. Both are possible and neither is something a screen does by
// accident, which is the failure this guard is aimed at. Inside the allowed
// file it reads which addresses are written, not where the value handed to the
// call comes from: the package beside that file proves each button opens its
// own address.
func ToolkitNetwork(root string, allowed map[string]ToolkitAllowance) ([]string, error) {
	var findings []string
	seen := map[string]bool{}
	for _, where := range []string{ViewRoot, WindowProgram} {
		found, err := toolkitNetworkUnder(root, where, allowed, seen)
		if err != nil {
			return nil, err
		}
		findings = append(findings, found...)
	}
	for file, allowance := range allowed {
		if !seen[file] {
			findings = append(findings, fmt.Sprintf("%s is allowed %s and is not there, so the "+
				"exception describes nothing. It moved, and the register has to move with it, or "+
				"it went, and the entry has to go", file, allowance.Call))
		}
	}
	sort.Strings(findings)
	return findings, nil
}

// toolkitNetworkUnder reads one of the two trees, noting every allowed file it
// meets so that an allowance for a file nobody can find is reported.
func toolkitNetworkUnder(root, where string, allowed map[string]ToolkitAllowance,
	seen map[string]bool) ([]string, error) {
	dir := filepath.Join(root, filepath.FromSlash(where))
	if _, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("%s is not there, so this guard would check nothing there: %w", where, err)
	}

	var findings []string
	walkErr := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if path != dir && (skippedDirs[name] || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(relative)
		allowance, isAllowed := allowed[name]
		seen[name] = seen[name] || isAllowed
		found, err := toolkitNetworkIn(path, name, allowance, isAllowed)
		if err != nil {
			return err
		}
		findings = append(findings, found...)
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return findings, nil
}

func toolkitNetworkIn(path, name string, allowance ToolkitAllowance, isAllowed bool) ([]string, error) {
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	var findings []string
	called := false
	named := map[string]bool{}
	ast.Inspect(parsed, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.SelectorExpr:
			why, banned := ToolkitNetworkNames[typed.Sel.Name]
			switch {
			case !banned:
			case isAllowed && typed.Sel.Name == allowance.Call:
				called = true
			default:
				findings = append(findings, fmt.Sprintf("%s:%d calls %s, which %s",
					name, fileSet.Position(node.Pos()).Line, typed.Sel.Name, why))
			}
		case *ast.BasicLit:
			if address, err := strconv.Unquote(typed.Value); err == nil && typed.Kind == token.STRING &&
				strings.Contains(address, "://") {
				named[address] = true
			}
		}
		return true
	})
	if !isAllowed {
		return findings, nil
	}
	findings = append(findings, allowedFileFindings(parsed, fileSet, name, allowance, called, named)...)
	if allowance.Scheme != "" {
		findings = append(findings, schemeFindings(parsed, fileSet, name, allowance.Scheme)...)
	}
	return findings, nil
}

// allowedFileFindings is what an allowed file must keep true: it calls what it
// is allowed, it names exactly the addresses listed, and, unless it is bound to
// a scheme instead, nothing it exports takes an address from a caller.
func allowedFileFindings(parsed *ast.File, fileSet *token.FileSet, name string, allowance ToolkitAllowance,
	called bool, named map[string]bool) []string {
	var findings []string
	if !called {
		findings = append(findings, fmt.Sprintf("%s is allowed %s and does not call it, so the "+
			"exception has outlived its use", name, allowance.Call))
	}
	listed := map[string]bool{}
	for _, address := range allowance.Addresses {
		listed[address] = true
		if !named[address] {
			findings = append(findings, fmt.Sprintf("%s no longer names %s, which the register lists, so "+
				"the two have stopped describing each other", name, address))
		}
	}
	for address := range named {
		if !listed[address] {
			findings = append(findings, fmt.Sprintf("%s names %s, which the register does not list. "+
				"A page this window may open is a decision (DEC-276), so it goes into the register first",
				name, address))
		}
	}
	if allowance.Scheme != "" {
		return findings
	}
	for _, declared := range parsed.Decls {
		function, ok := declared.(*ast.FuncDecl)
		if !ok || !function.Name.IsExported() {
			continue
		}
		for _, parameter := range function.Type.Params.List {
			if takesAnAddress(parameter.Type) {
				findings = append(findings, fmt.Sprintf("%s:%d: %s takes a string or a URL, so a caller "+
					"could hand in an address of its own", name,
					fileSet.Position(function.Pos()).Line, function.Name.Name))
			}
		}
	}
	return findings
}

// takesAnAddress is whether a parameter of this type could carry an address.
func takesAnAddress(expression ast.Expr) bool {
	switch typed := expression.(type) {
	case *ast.Ident:
		return typed.Name == "string"
	case *ast.StarExpr:
		return takesAnAddress(typed.X)
	case *ast.ArrayType:
		return takesAnAddress(typed.Elt)
	case *ast.Ellipsis:
		return takesAnAddress(typed.Elt)
	case *ast.SelectorExpr:
		return typed.Sel.Name == "URL"
	}
	return false
}

// urlPackage is the parser of addresses, the one package under net the network
// rule lets through.
const urlPackage = "net/url"

// partsOfAnAddress are the parts of a url.URL that could carry it somewhere
// other than a file, which a file bound to a scheme may not set or change.
var partsOfAnAddress = map[string]bool{
	"Scheme": true, "Opaque": true, "User": true, "Host": true, "RawPath": true, "RawQuery": true,
	"Fragment": true, "RawFragment": true, "OmitHost": true, "ForceQuery": true,
}

// schemeFindings is what a file bound to a scheme must keep true: every address
// it builds is a url.URL literal holding that scheme and a path, it parses none
// from text, and it changes none after building it.
//
// It follows the name net/url is imported under, so an alias changes nothing,
// and refuses a dot import, under which the parser would carry no name to
// follow.
func schemeFindings(parsed *ast.File, fileSet *token.FileSet, name, scheme string) []string {
	var findings []string
	at := func(node ast.Node) string { return fmt.Sprintf("%s:%d", name, fileSet.Position(node.Pos()).Line) }
	local := ""
	for _, imported := range parsed.Imports {
		if path, err := strconv.Unquote(imported.Path.Value); err != nil || path != urlPackage {
			continue
		}
		local = "url"
		if imported.Name != nil {
			local = imported.Name.Name
		}
		if local == "." {
			findings = append(findings, fmt.Sprintf("%s imports %s with a dot, so its parser carries no "+
				"name this guard can follow", at(imported), urlPackage))
		}
	}
	ofURL := func(expression ast.Expr, member string) bool {
		selector, isSelector := expression.(*ast.SelectorExpr)
		if !isSelector || selector.Sel.Name != member {
			return false
		}
		owner, isIdent := selector.X.(*ast.Ident)
		return isIdent && local != "" && owner.Name == local
	}
	ast.Inspect(parsed, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.CallExpr:
			if ofURL(typed.Fun, "Parse") || ofURL(typed.Fun, "ParseRequestURI") {
				findings = append(findings, fmt.Sprintf("%s parses an address from text, which could carry "+
					"any scheme, where this file may build only %s addresses", at(typed), scheme))
			}
		case *ast.CompositeLit:
			if ofURL(typed.Type, "URL") {
				findings = append(findings, builtAddressFindings(typed, at, scheme)...)
			}
		case *ast.SelectorExpr:
			if partsOfAnAddress[typed.Sel.Name] {
				findings = append(findings, fmt.Sprintf("%s reaches the %s of an address after it is built, "+
					"where this file may build only %s addresses", at(typed), typed.Sel.Name, scheme))
			}
		}
		return true
	})
	return findings
}

// builtAddressFindings is what is wrong with one url.URL literal in a file
// bound to a scheme: a scheme other than the one written out, or a part beside
// the scheme and the path.
func builtAddressFindings(literal *ast.CompositeLit, at func(ast.Node) string, scheme string) []string {
	var findings []string
	holdsScheme := false
	for _, element := range literal.Elts {
		pair, isPair := element.(*ast.KeyValueExpr)
		if !isPair {
			// Once for the literal: parts in order are all in order or none.
			return []string{at(element) + " builds an address from parts given in order, so " +
				"which part is the scheme is not written"}
		}
		key, _ := pair.Key.(*ast.Ident)
		switch {
		case key == nil:
		case key.Name == "Scheme":
			value, isText := pair.Value.(*ast.BasicLit)
			written, err := "", error(nil)
			if isText && value.Kind == token.STRING {
				written, err = strconv.Unquote(value.Value)
			}
			if !isText || err != nil || written != scheme {
				findings = append(findings, fmt.Sprintf("%s builds an address whose scheme is not %q "+
					"written out", at(pair), scheme))
				continue
			}
			holdsScheme = true
		case key.Name != "Path":
			findings = append(findings, fmt.Sprintf("%s sets the %s of an address, where this file may set "+
				"only the scheme and the path", at(pair), key.Name))
		}
	}
	if !holdsScheme && len(findings) == 0 {
		findings = append(findings, fmt.Sprintf("%s builds an address with no scheme, which the system "+
			"may read as a page rather than a %s", at(literal), scheme))
	}
	return findings
}
