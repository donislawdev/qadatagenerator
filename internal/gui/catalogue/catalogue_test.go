package catalogue

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/fynetheme"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

func TestMain(m *testing.M) {
	application := test.NewApp()
	// The window's own theme rather than the toolkit's test theme, because the
	// guard below compares drawings, and a drawing in colours this window does
	// not have would answer about a window nobody sees.
	application.Settings().SetTheme(fynetheme.Current())
	os.Exit(m.Run())
}

// The package of the controls and the packages the catalogue does not count as
// a screen using one: parts builds its own controls out of each other, and the
// catalogue builds every one of them on purpose.
const (
	partsImport = "github.com/donislawdev/qadatagenerator/internal/gui/parts"
	partsDir    = "../parts"
)

var notAScreen = []string{"parts", "catalogue"}

// samples are what the guards build the catalogue with. Any words will do,
// since what is asked is which controls it holds and how they are drawn.
var samples = Samples{
	Word:    "Word",
	Long:    "A sentence longer than any label the window has on any of its screens",
	Extreme: "WWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWW",
	Lines:   "first\nsecond\nthird\nfourth\nfifth\nsixth\nseventh\neighth",
	Values:  []string{"one", "two"},
	Table:   [][]string{{"a", "b"}, {"1", "2"}, {"3", "4"}},
}

// every is the whole register, page after page.
func every() []control {
	return slices.Concat(buttons(), choosers(), checks(), texts(), editors(), lists(), areas())
}

// TestTheCatalogueShowsEveryControlTheScreensBuildAndNothingElse closes the
// register of the catalogue against the code, in both directions (DEC-253).
//
// # Why the code rather than a list
//
// A list of the controls kept beside this register would be a second register
// somebody forgets to extend, which is the drift the rules for the window name
// when they say the catalogue may not show a control the window does not have,
// nor leave out one it has. So both halves are read from the source: which
// types internal/gui/parts builds, from its constructors, and which of those a
// screen asks for, from every call to one of them outside parts and outside
// this package.
func TestTheCatalogueShowsEveryControlTheScreensBuildAndNothingElse(t *testing.T) {
	built := constructors(t)
	used := usedByScreens(t, built)
	shown := map[string]bool{}
	for _, each := range every() {
		if shown[each.kind] {
			t.Errorf("the catalogue shows %s twice, so a change to one of its entries leaves the other "+
				"telling a different story", each.kind)
		}
		shown[each.kind] = true
	}
	for _, kind := range sortedKeys(used) {
		if !shown[kind] {
			t.Errorf("a screen builds parts.%s (%s) and the catalogue does not show it, so nobody can "+
				"see it in every state. Add it to a page in register.go", kind, used[kind])
		}
	}
	for _, kind := range sortedKeys(shown) {
		if _, isUsed := used[kind]; !isUsed {
			t.Errorf("the catalogue shows parts.%s and no screen builds one, so the catalogue shows a "+
				"control the window does not have. Remove it from register.go, or delete the control", kind)
		}
	}
}

// constructors is every exported New function of parts, with the type it
// builds.
func constructors(t *testing.T) map[string]string {
	t.Helper()
	built := map[string]string{}
	for _, file := range parsed(t, partsDir) {
		for _, declared := range file.Decls {
			function, isFunc := declared.(*ast.FuncDecl)
			if !isFunc || function.Recv != nil || !function.Name.IsExported() ||
				!strings.HasPrefix(function.Name.Name, "New") || function.Type.Results == nil {
				continue
			}
			if kind := pointedAt(function.Type.Results.List[0].Type); kind != "" {
				built[function.Name.Name] = kind
			}
		}
	}
	// The canary: the parser reading nothing would close the register over an
	// empty set and pass.
	if len(built) == 0 {
		t.Fatal("read no constructor out of internal/gui/parts, so nothing here was checked")
	}
	return built
}

// pointedAt is the name of the type a result points at, or nothing.
func pointedAt(result ast.Expr) string {
	star, isStar := result.(*ast.StarExpr)
	if !isStar {
		return ""
	}
	if name, isName := star.X.(*ast.Ident); isName {
		return name.Name
	}
	return ""
}

// usedByScreens is every type of parts some package of the window builds,
// with one place that builds it, for a message.
func usedByScreens(t *testing.T, built map[string]string) map[string]string {
	t.Helper()
	used := map[string]string{}
	read := 0
	for _, dir := range windowDirs(t) {
		for path, file := range parsed(t, dir) {
			read++
			alias := importedAs(file, partsImport)
			if alias == "" {
				continue
			}
			ast.Inspect(file, func(node ast.Node) bool {
				if kind := constructed(node, alias, built); kind != "" && used[kind] == "" {
					used[kind] = filepath.ToSlash(path)
				}
				return true
			})
		}
	}
	if read == 0 || len(used) == 0 {
		t.Fatalf("read %d files of the window and found %d controls built, so nothing here was checked",
			read, len(used))
	}
	return used
}

// constructed is the type a call to a constructor of parts builds, when the
// node is one.
func constructed(node ast.Node, alias string, built map[string]string) string {
	call, isCall := node.(*ast.CallExpr)
	if !isCall {
		return ""
	}
	selector, isSelector := call.Fun.(*ast.SelectorExpr)
	if !isSelector {
		return ""
	}
	if pkg, isName := selector.X.(*ast.Ident); !isName || pkg.Name != alias {
		return ""
	}
	return built[selector.Sel.Name]
}

// windowDirs is every directory of internal/gui but the two that are not a
// screen using a control, together with the binary of the window.
func windowDirs(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir("..")
	if err != nil {
		t.Fatal(err)
	}
	dirs := []string{filepath.Join("..", "..", "..", "cmd", "qadg-gui")}
	seen := map[string]bool{}
	for _, entry := range entries {
		seen[entry.Name()] = true
		if entry.IsDir() && !slices.Contains(notAScreen, entry.Name()) {
			dirs = append(dirs, filepath.Join("..", entry.Name()))
		}
	}
	// A name on the list that stopped naming a directory would be a hole
	// nobody sees, since leaving out nothing looks exactly like leaving out
	// the right thing.
	for _, name := range notAScreen {
		if !seen[name] {
			t.Errorf("%s is left out of the walk and there is no such directory under internal/gui", name)
		}
	}
	return dirs
}

// parsed is every Go file of a directory that is not a test.
func parsed(t *testing.T, dir string) map[string]*ast.File {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]*ast.File{}
	set := token.NewFileSet()
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(set, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("cannot read %s: %v", path, err)
		}
		files[path] = file
	}
	return files
}

// importedAs is the name a file gives an import, or nothing when it has none.
func importedAs(file *ast.File, path string) string {
	for _, spec := range file.Imports {
		imported, err := strconv.Unquote(spec.Path.Value)
		if err != nil || imported != path {
			continue
		}
		if spec.Name != nil {
			return spec.Name.Name
		}
		return filepath.Base(path)
	}
	return ""
}

func sortedKeys[V any](of map[string]V) []string {
	keys := make([]string, 0, len(of))
	for key := range of {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// Every variant builds the type its entry names, so the name the register is
// closed by is the name of what is really on the page.
func TestEveryVariantBuildsTheControlItsEntryNames(t *testing.T) {
	for _, each := range every() {
		for _, shown := range each.variants {
			made, drawn := shown.build(i18n.English, samples)
			kind := reflect.TypeOf(made)
			if kind.Kind() == reflect.Pointer {
				kind = kind.Elem()
			}
			if kind.Name() != each.kind || kind.PkgPath() != partsImport {
				t.Errorf("%s of %s builds %s.%s", i18n.English.T(shown.name), each.kind, kind.PkgPath(), kind.Name())
			}
			if drawn == nil {
				t.Errorf("%s of %s puts nothing on the page", i18n.English.T(shown.name), each.kind)
			}
		}
	}
}

// TestEveryControlIsShownInExactlyTheStatesItsDrawingAnswersTo derives the
// columns of the catalogue from the drawing (DEC-253, point 4d).
//
// # Why the drawing and not the methods
//
// A control carrying a method is not a control that draws what the method
// says. The text field has a MouseDown, promoted from the toolkit's entry, and
// draws no press. The chooser has a Disable it did not write, and draws being
// out of reach. A register built from methods would show a press nobody sees
// and a column that is the one before it again - and the rules for the window
// say an appearance is checked by its effect, never by the presence of code.
//
// So each state a control takes is drawn on a fresh control and compared with
// the state a person reaches it from: a press with the pointer it lands on,
// everything else with rest. A state that changes the picture has to be a
// column, and a column has to change the picture.
func TestEveryControlIsShownInExactlyTheStatesItsDrawingAnswersTo(t *testing.T) {
	for _, each := range every() {
		t.Run(each.kind, func(t *testing.T) {
			if len(each.states) == 0 || each.states[0] != rest {
				t.Fatalf("%s is not shown at rest first, and every other state is read against that one", each.kind)
			}
			for candidate := pointer; candidate < stateCount; candidate++ {
				checkState(t, each, candidate)
			}
		})
	}
}

// checkState holds one state of one control against its drawing.
func checkState(t *testing.T, each control, candidate state) {
	t.Helper()
	listed := slices.Contains(each.states, candidate)
	name := i18n.English.T(stateNames[candidate])
	takes, changes := false, false
	for _, shown := range each.variants {
		made, _ := shown.build(i18n.English, samples)
		if !candidate.apply(made) {
			continue
		}
		takes = true
		if !bytes.Equal(pixels(t, shown, candidate), pixels(t, shown, candidate.before())) {
			changes = true
		}
	}
	switch {
	case listed && !takes:
		t.Errorf("%s is shown %q and cannot be put in that state, so the page would lie about it",
			each.kind, name)
	case listed && !changes:
		t.Errorf("%s is shown %q and that state draws exactly what %q draws, so the column shows "+
			"nothing a person could tell apart", each.kind, name, i18n.English.T(stateNames[candidate.before()]))
	case changes && !listed:
		t.Errorf("%s draws %q differently and the catalogue does not show it, so nobody sees that state",
			each.kind, name)
	}
}

// pixels is a fresh control of this variant in this state, drawn on its own,
// with room around it for a mark drawn outside its edge.
func pixels(t *testing.T, shown variant, in state) []byte {
	t.Helper()
	made, drawn := shown.build(i18n.English, samples)
	in.apply(made)
	content := paint.PadAll(palette.NameStepGroup, drawn)
	window := test.NewWindow(content)
	defer window.Close()
	window.Resize(content.MinSize())
	captured := window.Canvas().Capture()
	bounds := captured.Bounds()
	rgba := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, captured.At(x, y))
		}
	}
	return append([]byte(bounds.String()), rgba.Pix...)
}
