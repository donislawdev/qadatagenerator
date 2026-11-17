package guard

import (
	"sort"
	"strings"
)

// networkAllowed lists the packages under net that cannot open a connection.
// Parsing an address is not talking to one.
var networkAllowed = map[string]bool{
	"net/url": true,
}

// reachesOut are the packages outside the net tree that can leave this machine
// or bring in code from elsewhere.
//
// syscall is here because it can do anything the operating system can, sockets
// included, so importing it directly would route around every other name on
// this list. It arrives in the build graph anyway through os, which is why this
// answers about what our own files import and not about what the graph
// contains. Those are two different questions and the second one is asked in
// CI, where a shell can run go list.
var reachesOut = map[string]bool{
	"os/exec": true,
	"syscall": true,
	"plugin":  true,
}

// OpensTheWorld reports whether an import can reach outside this machine or
// start another program.
//
// The tool has no server, no account, no telemetry and no update check. That
// is a selling point in every bank, and it is worth nothing as a promise, so
// it is a check instead.
//
// # Why a segment rather than a prefix
//
// This tested strings.HasPrefix(importPath, "net/") until 2026-08-03, which
// answers for the standard library and for nothing else. golang.org/x/net/http2
// does not begin with net/ and would have passed, so the guard covered the one
// source of network code this project is least likely to add by accident and
// missed the one it is most likely to add on purpose.
//
// Matching a whole path segment named net catches both spellings and cannot be
// fooled by a name that merely starts the same way, so netlink and networkd
// stay clean.
func OpensTheWorld(importPath string) bool {
	if networkAllowed[importPath] {
		return false
	}
	if reachesOut[importPath] {
		return true
	}
	for _, segment := range strings.Split(importPath, "/") {
		if segment == "net" {
			return true
		}
	}
	return false
}

// Network reports every package that imports something able to reach outside.
//
// # What this cannot see, said out loud
//
// It reads the imports written in our own files. A dependency that opens a
// socket somewhere inside itself is invisible here, because the scan never
// leaves this repository, and the promise being kept is about the whole binary
// rather than about our half of it.
//
// The other half is checked in CI, where go list can walk the build graph
// including everything the modules pull in. Two mechanisms because they need
// two different tools, and naming the split here is what stops the next reader
// from believing this function answers the whole question.
// # The one exception, and where the rest of it lives
//
// A package holding a file named in guard.SyscallAllowed may import syscall.
// That is coarser than the allowance itself, which is a file and a list of
// identifiers, so this half is deliberately not the whole check: SyscallUse
// answers whether the file took only what it may. Neither is enough alone, and
// splitting them this way is what keeps this function answering one question.
func Network(pkgs []Package) []string {
	var findings []string
	for _, pkg := range pkgs {
		for _, importPath := range pkg.Imports {
			if importPath == syscallPackage && packageMayNameSyscall(pkg.Path) {
				continue
			}
			// The same split for starting a process: SpawnUse answers what the
			// allowed file starts, and this answers that no other package may.
			if importPath == execPackage && packageMaySpawn(pkg.Path) {
				continue
			}
			if OpensTheWorld(importPath) {
				findings = append(findings, pkg.Path+" imports "+importPath+", which can reach outside this machine")
			}
		}
	}
	sort.Strings(findings)
	return findings
}

// layerRule says which parts of the tree a package may not import.
type layerRule struct {
	// pkg is the prefix of the importing package.
	pkg string
	// forbidden are the prefixes it may not reach. An import that stays
	// inside pkg is always allowed.
	forbidden []string
	// why is printed with the finding, because a rule whose reason is not
	// at hand gets deleted by whoever hits it first.
	why string
}

var layerRules = []layerRule{
	{
		pkg:       "internal/fault",
		forbidden: []string{"internal/", "cmd/"},
		why:       "the failure kind sits under everything else, so it can depend on nothing of ours",
	},
	{
		pkg: "internal/version",
		forbidden: []string{"internal/app", "internal/adapter", "internal/cli", "internal/gui",
			"internal/damage", "internal/take"},
		why: "the version numbers are read by every layer, so they may not know any of them",
	},
	{
		pkg: "internal/domain",
		forbidden: []string{"internal/app", "internal/adapter", "internal/cli", "internal/gui",
			"internal/damage", "internal/take"},
		why: "the domain knows nothing about files, formats, the command line or the window",
	},
	{
		pkg: "internal/app",
		forbidden: []string{"internal/adapter", "internal/cli", "internal/gui",
			"internal/damage", "internal/take"},
		why: "a use case takes what it needs as a port and does not know where the data comes from",
	},
	{
		pkg:       "internal/damage",
		forbidden: []string{"internal/cli", "internal/gui", "internal/take"},
		why: "the assembly that breaks records is what both surfaces share, so it may " +
			"know neither of them, and it sits below the mode that uses it",
	},
	{
		pkg:       "internal/take",
		forbidden: []string{"internal/cli", "internal/gui"},
		why: "the take and corrupt mode exists so that both surfaces run one of it, " +
			"which it cannot do while knowing either",
	},
	{
		pkg:       "internal/produce",
		forbidden: []string{"internal/cli", "internal/gui"},
		why: "answering what a recipe would produce is the same answer on both surfaces, " +
			"and the window is only the first to ask",
	},
	{
		pkg:       "internal/cli",
		forbidden: []string{"internal/gui"},
		why:       "the two surfaces run the same logic and never each other",
	},
	{
		pkg:       "internal/gui",
		forbidden: []string{"internal/cli"},
		why:       "the two surfaces run the same logic and never each other",
	},
}

// Layers reports every import that crosses the dependency rule the wrong way.
func Layers(pkgs []Package, module string) []string {
	var findings []string
	for _, pkg := range pkgs {
		for _, rule := range layerRules {
			if within(pkg.Path, rule.pkg) {
				findings = append(findings, rule.violatedBy(pkg, module)...)
			}
		}
	}
	sort.Strings(findings)
	return findings
}

// violatedBy returns every import of one package that this rule forbids.
//
// An import that stays inside the rule's own subtree is always allowed, which
// is what lets a package be split into files without the rule noticing.
func (rule layerRule) violatedBy(pkg Package, module string) []string {
	var found []string
	for _, importPath := range pkg.Imports {
		target, ok := moduleRelative(importPath, module)
		if !ok || within(target, rule.pkg) {
			continue
		}
		if rule.forbids(target) {
			found = append(found, pkg.Path+" imports "+target+": "+rule.why)
		}
	}
	return found
}

func (rule layerRule) forbids(target string) bool {
	for _, forbidden := range rule.forbidden {
		if within(target, forbidden) {
			return true
		}
	}
	return false
}

// Has reports whether the graph contains a package at path.
//
// It exists because of how a guard stops working without saying so. A rule
// about a package that is not there passes every time and checks nothing, so
// a renamed directory would retire the rule in silence.
func Has(pkgs []Package, path string) bool {
	for _, pkg := range pkgs {
		if pkg.Path == path {
			return true
		}
	}
	return false
}

// Reach returns the import chain from one package to another, or nil when
// there is none.
//
// The transitive form is the one that matters for the edge this protects. A
// direct import is easy to spot in review and an indirect one is not, and both
// pull the same code into the binary.
func Reach(pkgs []Package, module, from, to string) []string {
	graph := make(map[string][]string, len(pkgs))
	for _, pkg := range pkgs {
		for _, importPath := range pkg.Imports {
			if target, ok := moduleRelative(importPath, module); ok {
				graph[pkg.Path] = append(graph[pkg.Path], target)
			}
		}
	}

	cameFrom := map[string]string{from: ""}
	queue := []string{from}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, next := range graph[current] {
			if _, seen := cameFrom[next]; seen {
				continue
			}
			cameFrom[next] = current
			if within(next, to) {
				return chain(cameFrom, from, next)
			}
			queue = append(queue, next)
		}
	}
	return nil
}

func chain(cameFrom map[string]string, from, to string) []string {
	var reversed []string
	for step := to; step != ""; step = cameFrom[step] {
		reversed = append(reversed, step)
		if step == from {
			break
		}
	}
	ordered := make([]string, 0, len(reversed))
	for i := len(reversed) - 1; i >= 0; i-- {
		ordered = append(ordered, reversed[i])
	}
	return ordered
}

// TestOnly reports every ordinary source file that imports a package meant for
// tests. A guard linked into the product is a guard that ships.
func TestOnly(pkgs []Package, module, target string) []string {
	var findings []string
	for _, pkg := range pkgs {
		for _, importPath := range pkg.Imports {
			relative, ok := moduleRelative(importPath, module)
			if ok && within(relative, target) {
				findings = append(findings, pkg.Path+" imports "+relative+" outside a test file")
			}
		}
	}
	sort.Strings(findings)
	return findings
}

// moduleRelative turns an import path into a path relative to the module root
// and reports whether it belongs to this module at all.
func moduleRelative(importPath, module string) (string, bool) {
	if importPath == module {
		return ".", true
	}
	if strings.HasPrefix(importPath, module+"/") {
		return strings.TrimPrefix(importPath, module+"/"), true
	}
	return "", false
}

// within reports whether path is the prefix itself or lies under it. Comparing
// strings alone would make internal/climate match internal/cli.
func within(path, prefix string) bool {
	trimmed := strings.TrimSuffix(prefix, "/")
	return path == trimmed || strings.HasPrefix(path, trimmed+"/")
}
