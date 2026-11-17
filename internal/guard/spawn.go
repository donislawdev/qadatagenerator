package guard

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
)

// SpawnAllowed names the files that may start a process, together with the one
// start each of them may make.
//
// # Why there is any
//
// The network guard refuses os/exec, because a program this tool starts can
// reach whatever that program reaches, and the command line binary still
// starts nothing (DEC-169). The window on Windows has to start one program:
// itself, again, with the software renderer, when the graphics driver gave it
// no window (B-414, DEC-317).
//
// # Why a call and where its program comes from, rather than a file
//
// A file allowed os/exec could start anything, and a year from now nobody would
// remember it is an exception. So an entry names the one function of os/exec
// the file calls to start a process, which it calls exactly once, and the
// function whose answer the program has to be, bound once in the function
// that starts it. Every other name taken from os/exec, a second start, a
// program from anywhere else and a literal exec.Cmd are findings. So is
// os.StartProcess in any file, which starts a process without importing
// os/exec at all and which the network guard cannot see.
//
// # What it does not do
//
// It reads the tree as written, like the syscall guard beside it. A program
// passed through a variable of another function, or a function value holding
// exec.Command, is invisible here.
var SpawnAllowed = map[string]Spawn{
	// The window, starting itself again with the software renderer. The other
	// name is the type of the error a process that ran and failed answers with,
	// which starts nothing.
	"cmd/qadg-gui/again.go": {Call: "Command", ProgramFrom: "os.Executable", Also: []string{"ExitError"}},
}

// Spawn is the one start a file may make.
type Spawn struct {
	// Call is the function of os/exec that starts the process.
	Call string
	// ProgramFrom is the package and function whose answer the program is, as
	// a file writes it with the package's own name.
	ProgramFrom string
	// Also are the other names the file may take from os/exec.
	Also []string
}

// execPackage is the import being controlled.
const execPackage = "os/exec"

// packageMaySpawn reports whether a package holds a file allowed to start a
// process.
func packageMaySpawn(pkg string) bool {
	return holdsAnAllowedFile(SpawnAllowed, pkg)
}

// SpawnUse reports every way the product starts a process outside what
// allowed lets it, in both directions: a start nobody allowed, an allowed file
// doing more than its entry says, and an entry for a file that no longer
// starts anything.
func SpawnUse(root string, allowed map[string]Spawn) ([]string, error) {
	var findings []string
	seen := map[string]bool{}
	err := eachProductFile(root, ".", func(path, name string) error {
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		allowance, isAllowed := allowed[name]
		found, imports, err := spawnsIn(parsed, name, allowance, isAllowed)
		seen[name] = imports
		findings = append(findings, found...)
		return err
	})
	if err != nil {
		return nil, err
	}
	for file := range allowed {
		if !seen[file] {
			findings = append(findings, file+" is allowed to start a process and imports no "+execPackage+
				", so the allowance is stale")
		}
	}
	sort.Strings(findings)
	return findings, nil
}

// spawnsIn reports what one file does to start a process, and whether it
// imports os/exec.
func spawnsIn(parsed *ast.File, name string, allowance Spawn, allowed bool) ([]string, bool, error) {
	findings, err := startProcessIn(parsed, name)
	if err != nil {
		return nil, false, err
	}
	execName, err := importedAs(parsed, execPackage)
	if err != nil || execName == "" {
		return findings, false, err
	}
	switch {
	case !allowed:
		return append(findings, name+" imports "+execPackage+", and only the files in guard.SpawnAllowed may"), true, nil
	case execName == ".":
		return append(findings, name+" imports "+execPackage+" with a dot, which hides what it calls"), true, nil
	}
	found, err := startsIn(parsed, name, execName, allowance)
	return append(findings, found...), true, err
}

// startProcessIn reports os.StartProcess, which starts a process with no
// import of os/exec.
func startProcessIn(parsed *ast.File, name string) ([]string, error) {
	osName, err := importedAs(parsed, "os")
	if err != nil || osName == "" {
		return nil, err
	}
	var findings []string
	ast.Inspect(parsed, func(node ast.Node) bool {
		if selectorOf(node, osName, "StartProcess") {
			findings = append(findings, name+" calls os.StartProcess, which starts a process no allowance names")
		}
		return true
	})
	return findings, nil
}

// startsIn holds an allowed file to its entry: the names it takes from
// os/exec, how many times it starts a process, and where each program comes
// from.
func startsIn(parsed *ast.File, name, execName string, allowance Spawn) ([]string, error) {
	permitted := map[string]bool{allowance.Call: true}
	for _, also := range allowance.Also {
		permitted[also] = true
	}
	var findings []string
	ast.Inspect(parsed, func(node ast.Node) bool {
		if selector, ok := node.(*ast.SelectorExpr); ok && isIdent(selector.X, execName) && !permitted[selector.Sel.Name] {
			findings = append(findings, fmt.Sprintf("%s uses exec.%s, and its allowance is exec.%s",
				name, selector.Sel.Name, allowance.Call))
		}
		return true
	})
	starts := 0
	for _, declaration := range parsed.Decls {
		function, isFunction := declaration.(*ast.FuncDecl)
		ast.Inspect(declaration, func(node ast.Node) bool {
			call, isCall := node.(*ast.CallExpr)
			if !isCall || !selectorOf(call.Fun, execName, allowance.Call) {
				return true
			}
			starts++
			if !isFunction || len(call.Args) == 0 || !programFrom(parsed, function, call.Args[0], allowance.ProgramFrom) {
				findings = append(findings, fmt.Sprintf("%s starts a program that is not what %s answered "+
					"in the function that starts it", name, allowance.ProgramFrom))
			}
			return true
		})
	}
	if starts != 1 {
		findings = append(findings, fmt.Sprintf("%s calls exec.%s %d times, and its allowance is one start",
			name, allowance.Call, starts))
	}
	return findings, nil
}

// programFrom reports whether program is a name bound exactly once in
// function, and to what the function named by from answered. A parameter, a
// second binding or anything other than that call is not.
func programFrom(parsed *ast.File, function *ast.FuncDecl, program ast.Expr, from string) bool {
	named, isName := program.(*ast.Ident)
	pkg, call, found := strings.Cut(from, ".")
	if !isName || !found || function.Body == nil {
		return false
	}
	pkgName, err := importedAs(parsed, pkg)
	if err != nil || pkgName == "" {
		return false
	}
	for _, field := range function.Type.Params.List {
		for _, param := range field.Names {
			if param.Name == named.Name {
				return false
			}
		}
	}
	bindings, fromCall := 0, false
	ast.Inspect(function.Body, func(node ast.Node) bool {
		switch bound := node.(type) {
		case *ast.AssignStmt:
			for _, left := range bound.Lhs {
				if isIdent(left, named.Name) {
					bindings++
					fromCall = len(bound.Rhs) == 1 && isCallOf(bound.Rhs[0], pkgName, call)
				}
			}
		case *ast.ValueSpec:
			for _, left := range bound.Names {
				if left.Name == named.Name {
					bindings++
				}
			}
		case *ast.RangeStmt:
			if isIdent(bound.Key, named.Name) || isIdent(bound.Value, named.Name) {
				bindings++
			}
		}
		return true
	})
	return bindings == 1 && fromCall
}

// selectorOf reports whether node is pkg.name written with that package name.
func selectorOf(node ast.Node, pkg, name string) bool {
	selector, ok := node.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == name && isIdent(selector.X, pkg)
}

// isCallOf reports whether expression calls pkg.name.
func isCallOf(expression ast.Expr, pkg, name string) bool {
	call, ok := expression.(*ast.CallExpr)
	return ok && selectorOf(call.Fun, pkg, name)
}

// isIdent reports whether expression is the bare name given.
func isIdent(expression ast.Expr, name string) bool {
	ident, ok := expression.(*ast.Ident)
	return ok && ident.Name == name
}
