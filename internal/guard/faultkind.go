package guard

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The register of which refusals carry which kind of failure.
//
// # Why this exists
//
// Two exit codes sit next to each other and mean different things. Code 3 says
// a document handed to the run is not valid. Code 5 says a document was read
// without trouble and what it asks for cannot be done. The difference decides
// what a person does next, and somebody else's pipeline branches on it.
//
// 🔴 The boundary was drawn on 2026-08-17 by walking the call sites once, by
// hand, and it was written into docs/09 as prose. Nothing held it. The bridge
// in the audit script compares the numbers and the names of the codes between
// the document and the table in the code, so it answers whether a code exists
// and never which ending gets which one. That is exactly how eight endings of
// one verb came to return the code for a recipe that cannot be satisfied while
// meaning that a file could not be read, with every guard green.
//
// So the sites are read from the tree, and every one of them has to be in the
// register below with the reason it carries the kind it carries. A new refusal
// is red until somebody says which side of the boundary it is on, and moving an
// existing one from one kind to the other is a visible edit rather than a
// character.

// RefusalKinds says, for every refusal either side of the boundary, why it
// carries the kind it does.
//
// 🔴 The reason is the whole entry. A register holding only the kind would say
// what the code already says, and would go on agreeing with it after somebody
// changed both. What it has to hold is the ARGUMENT, so that the next reader
// can tell a decision from a habit.
var RefusalKinds = map[string]string{
	// --- Config: a document handed to the run is not valid --------------------
	"internal/adapter/format.parentColumn Config":            "the recipe gives one entity a field and a relation of the same name, so the recipe is what has to change",
	"internal/adapter/format.usableDelimiter Config":         "the recipe gives a delimiter that is not one character. Measured 2026-08-17: the same value given as a FLAG never gets here, because the command line refuses it earlier as a wrong invocation, so this side of the boundary is the recipe alone",
	"internal/adapter/format.usableEncoding Config":          "the recipe asks for an encoding this build does not write, which is a setting rather than a limit of the data",
	"internal/adapter/manifestjson.Load Config":              "the manifest is not a manifest, is one this build would not have written, or names a major version it cannot read. All three are the document being wrong",
	"internal/adapter/responsejson.read Config":              "the response record cannot be read as one, carries two documents, fails the model or names a major version this build does not speak",
	"internal/adapter/store.usableName Config":               "the name comes from the recipe, and a name that cannot become a file is the recipe asking for something impossible before any data exists",
	"internal/adapter/store.Set.ownName Config":              "an entity is named after a file the run itself leaves, so the recipe asks for two things under one name. The names this tool reserves are fixed, which makes the recipe the only end anybody can move",
	"internal/adapter/values.Source.Produce Config":          "the recipe names a generator this build does not have, which is a spelling in a document rather than a limit of the data",
	"internal/adapter/values.Source.draw Config":             "the recipe asks for a list this build carries for no locale, so the pair of locale and field is what has to change",
	"internal/adapter/values.Source.drawEither Config":       "the same as draw, for a field drawing from two lists at once",
	"internal/adapter/values.Source.fromType Config":         "the recipe declares a type this build produces no values for, and the answer is to give the field a generator",
	"internal/adapter/values.Source.postalCode Config":       "the recipe asks for postal codes in a locale this build has no register for, which is a setting of the recipe",
	"internal/app/corrupt.Selector.somethingToDamage Config": "the policy excludes every field there is, so the policy asks for damage it has itself made impossible",
	"internal/app/corrupt.UsablePerRow Config":               "per_row names something the engine does not know, which is a value in the policy",
	"internal/app/corrupt.shareOf Config":                    "policy.rows is not a share this build can read, or is outside nought to a hundred. Both are the policy being wrong",
	"internal/app/generate.agreementFor Config":              "a consistency rule points at a field that is itself bound by a rule, and this build resolves one level. The recipe has to be written differently",
	"internal/app/generate.checkGenerators Config":           "the recipe names a generator this build does not have",
	"internal/app/generate.inDependencyOrder Config":         "the relations form a loop, so the recipe describes an order that cannot exist",
	"internal/app/generate.planEntity Config":                "an entity belongs to one this run does not produce, so the recipe names something that is not there",
	"internal/app/generate.refuseSecondParent Config":        "an entity belongs to more than one, and this build produces one relation per entity",
	"internal/app/generate.tooManyRows Config":               "the counts add up to more rows than this build can count, which is arithmetic in the recipe rather than a limit met while producing",
	"internal/app/validate.Run Config":                       "this is the verb whose whole answer is whether the recipe is valid",
	"internal/cli.recipeAt Config":                           "the recipe carries problems, and this verb was asked to produce data rather than to report on them",
	"internal/cli.writerFor Config":                          "the recipe asks for a format this build does not write",
	"internal/cli.usableFormat Config":                       "the run asks for a format this build does not write, from the recipe or from the flag that overrides it, and either way it is a document that cannot be acted on rather than one that cannot be met",
	"internal/take.Open Config":                              "the field separator is not one character. It is a setting somebody gave rather than data, and it is answered here a second time because a screen puts it under a cursor, where the empty value between two keystrokes would otherwise be a panic",

	// --- Data: a valid document asks for something that cannot be done --------
	"internal/adapter/manifestjson.bytesOf Data":             "the manifest is larger than this build reads back at once. The document is valid and this build cannot hold it, which is the other side of the boundary",
	"internal/adapter/responsejson.read Data":                "the response record is larger than this build reads at once, for the same reason as the manifest above",
	"internal/adapter/sourcecsv.File.changedUnderneath Data": "the source file changed while it was being read. 🔴 The file is the DATA being worked on rather than a document telling this tool what to do, which is where DEC-172 draws the line",
	"internal/adapter/sourcecsv.File.openAndReadHeader Data": "the source file is empty, and an empty file is data this mode cannot work with rather than an invalid instruction",
	"internal/adapter/sourcecsv.File.readFailure Data":       "the source file cannot be read as comma separated data. Still the payload rather than a document of ours, so it stays on this side",
	"internal/adapter/sourcecsv.File.survey Data":            "the source file has a header and no records, so there is nothing to damage",
	"internal/adapter/sourcecsv.usableHeader Data":           "a column of the source file has no name or repeats one, which the manifest could not then point at",
	"internal/adapter/values.Source.Produce Data":            "the value a generator produced is longer than the field accepts. The recipe is valid and cannot be satisfied by what this generator draws",
	"internal/adapter/values.Source.fromType Data":           "the field accepts fewer characters than a decimal needs for a point and two places",
	"internal/adapter/values.Source.pick Data":               "no entry of the list is short enough for the length this field accepts",
	"internal/app/generate.reachable Data":                   "the count of a child cannot be reached by any deal of its parents, so the recipe is valid and its arithmetic does not work out",
	"internal/app/generate.uniqueValue Data":                 "a unique field ran out of values while drawing, which is the definition of a valid recipe that cannot be satisfied",
}

// FaultSite is one place in the product that refuses with a kind, named the way
// a register can name it.
type FaultSite struct {
	// Package is the directory relative to the module root.
	Package string
	// Function is where the refusal is, with the receiver type when it is a
	// method, so that two functions of one package stay apart.
	Function string
	// Kind is Config or Data, spelled as it is in the source.
	Kind string
}

// Key is how the register names this site. It leaves the line number out on
// purpose: a line moves whenever anything above it changes, and a register
// keyed by line would need editing for reasons that have nothing to do with
// refusals.
func (s FaultSite) Key() string {
	return s.Package + "." + s.Function + " " + s.Kind
}

// BoundaryKinds are the two kinds this guard is about. The others are left
// alone deliberately: nobody confuses a read that failed with a recipe that
// cannot be satisfied, and a register covering everything would be a register
// nobody keeps.
var BoundaryKinds = map[string]bool{"Config": true, "Data": true}

// FaultSites reads the tree and returns every refusal carrying one of the two
// kinds either side of the boundary.
//
// # What it does not cover
//
// A kind held in a variable or handed in as a parameter, and a refusal built
// outside a function. Both are possible, neither is how any refusal in this
// tree is written today, and a guard that quietly covered less than it looks
// like is worse than one that says where it stops.
func FaultSites(root string) ([]FaultSite, error) {
	var sites []FaultSite
	walkErr := filepath.WalkDir(filepath.Join(root, "internal"), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if skippedDirs[name] || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		relative, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		found, err := faultSitesIn(path, filepath.ToSlash(relative))
		if err != nil {
			return err
		}
		sites = append(sites, found...)
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	sort.Slice(sites, func(i, j int) bool { return sites[i].Key() < sites[j].Key() })
	return unique(sites), nil
}

func faultSitesIn(path, pkg string) ([]FaultSite, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	var sites []FaultSite
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		name := functionName(function)
		ast.Inspect(function.Body, func(node ast.Node) bool {
			if kind, found := boundaryKindOf(node); found {
				sites = append(sites, FaultSite{Package: pkg, Function: name, Kind: kind})
			}
			return true
		})
	}
	return sites, nil
}

// boundaryKindOf answers whether this node is a refusal carrying one of the two
// kinds, and which one.
func boundaryKindOf(node ast.Node) (string, bool) {
	call, ok := node.(*ast.CallExpr)
	if !ok || len(call.Args) == 0 {
		return "", false
	}
	called, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || called.Sel.Name != "New" && called.Sel.Name != "Wrap" {
		return "", false
	}
	if pkg, ok := called.X.(*ast.Ident); !ok || pkg.Name != "fault" {
		return "", false
	}
	kind, ok := call.Args[0].(*ast.SelectorExpr)
	if !ok || !BoundaryKinds[kind.Sel.Name] {
		return "", false
	}
	return kind.Sel.Name, true
}

func functionName(function *ast.FuncDecl) string {
	if function.Recv == nil || len(function.Recv.List) == 0 {
		return function.Name.Name
	}
	return receiverName(function.Recv.List[0].Type) + "." + function.Name.Name
}

func receiverName(expr ast.Expr) string {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return "?"
}

// unique collapses the several refusals one function may carry of one kind into
// a single entry, because the register answers about a decision and one
// function makes that decision once.
func unique(sites []FaultSite) []FaultSite {
	var kept []FaultSite
	seen := map[string]bool{}
	for _, site := range sites {
		if seen[site.Key()] {
			continue
		}
		seen[site.Key()] = true
		kept = append(kept, site)
	}
	return kept
}

// FaultBoundary holds the sites found in the tree against the register and
// reports what does not line up, in both directions.
func FaultBoundary(sites []FaultSite) []string {
	var findings []string
	present := map[string]bool{}
	for _, site := range sites {
		present[site.Key()] = true
		if _, listed := RefusalKinds[site.Key()]; !listed {
			findings = append(findings, fmt.Sprintf(
				"%s refuses with fault.%s and the register does not say why. Exit code 3 says a "+
					"document is not valid and code 5 says a valid one cannot be satisfied, so this "+
					"needs an entry saying which of the two it is",
				site.Package+"."+site.Function, site.Kind))
		}
	}
	for key := range RefusalKinds {
		if !present[key] {
			findings = append(findings, fmt.Sprintf(
				"the register describes %q, which no longer refuses that way. Remove it rather "+
					"than leaving an answer about something that is not there", key))
		}
	}
	sort.Strings(findings)
	return findings
}
