package produce

import (
	"time"

	"github.com/donislawdev/qadatagenerator/internal/adapter/format"
	"github.com/donislawdev/qadatagenerator/internal/adapter/store"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/damage"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// Settings is what a run of a recipe was asked for beyond the recipe itself.
//
// Everything the recipe says stays in the recipe. What is here is what belongs
// to one run rather than to the recipe: where it goes, whether it may replace
// what is there, the extra form of the manifest (DEC-163), and the record of
// anything that moved a setting of the recipe from outside it.
type Settings struct {
	// Into is the directory the files are written to.
	Into string
	// Replace says whether files already there may be replaced.
	Replace bool
	// Manifest is the extra form the manifest is exported in, empty for none.
	// It changes nothing for a recipe that orders no damage, because such a
	// run writes no manifest to export, and FormUnused says so.
	Manifest string
	// Overrides are the settings of the recipe something outside it moved, in
	// the order they were moved. They are recorded, not applied: the recipe
	// handed in already carries the values that won.
	Overrides []Override
}

// FormUnused reports whether an extra form of the manifest was asked for by a
// run that writes no manifest at all, because its recipe orders no damage.
//
// # Why this is a fact both surfaces ask rather than a refusal
//
// The form is a correct setting that has nothing to act on, and a run that
// refused it would fail over a word that did no harm. A run that ignored it
// without saying so is the other failure, and the one that happened: the
// command line ended with nought and wrote no manifest, with not a word about
// the flag (B-301). So the run goes ahead and each surface says, in its own
// words, that no manifest was written and why (DEC-252).
func (s Settings) FormUnused(rec recipe.Recipe) bool {
	return s.Manifest != "" && !rec.Policy.Asks()
}

// Written is what a run left on a disk.
type Written struct {
	// Files are the names the run gave its files: the data in the order the
	// writer asked for them, then the manifest and its export when the recipe
	// orders damage, then the note beside them.
	//
	// 🔴 The data files are collected by watching the writer rather than
	// worked out from the recipe. Which files a form produces is the writer's
	// decision, and one form writes a file per table while another writes a
	// single file for the whole dataset. A caller listing them from the plan
	// would be right about today's forms and quietly wrong about the next one.
	Files []string
	// Rows is how many records the whole run wrote.
	Rows int
	// Plan is the plan the run followed, for a surface that reports it.
	Plan generate.Plan
	// Damaged says the recipe ordered damage, so that a manifest was written
	// and Counts says what became of the order.
	Damaged bool
	// Counts is what the damage did, as numbers. Each surface says it in its
	// own words, because one of them speaks English always and the other one
	// whatever it was translated into.
	Counts damage.Counts
	// Oversized are the manifests this run wrote that this build will not read
	// back, if any.
	Oversized []damage.Oversize
}

// Writes names every file a run leaves behind, given the files its writer asks
// for, whether its recipe orders damage, and the extra form of the manifest.
//
// It is one function rather than a list assembled at each caller, because a
// screen says what a run would write before it runs and the run says what it
// wrote afterwards, and the two have to be the same sentence. A manifest named
// by one and not by the other is how a person comes to look for a file nobody
// wrote.
func Writes(data []string, damaged bool, form string) []string {
	names := append([]string{}, data...)
	if damaged {
		names = append(names, damage.Files(form)...)
	}
	return append(names, RunNoteName)
}

// Prepared is a recipe planned and not yet run.
//
// # Why the plan is held beside the recipe rather than handed in with it
//
// The command line prints the plan before it runs and the run follows it, so
// the plan has to be made once and kept. Taking it as a parameter beside the
// recipe would let a caller hand in a plan made from a different recipe, and
// making it again inside the run costs a second pass that allocates the parent
// of every child row. Holding the two together makes the first impossible and
// the second unnecessary, which is what the take mode does with a file it has
// opened.
type Prepared struct {
	rec    recipe.Recipe
	source values.Source
	plan   generate.Plan
}

// Prepare plans a run of the recipe and writes nothing.
func Prepare(rec recipe.Recipe, source values.Source) (Prepared, error) {
	plan, err := planOf(rec, source)
	if err != nil {
		return Prepared{}, err
	}
	return Prepared{rec: rec, source: source, plan: plan}, nil
}

// Plan is what the run will follow, with what it will not do written into it.
func (p Prepared) Plan() generate.Plan { return p.plan }

// Write produces the dataset a recipe describes and leaves it in a directory,
// with the damage its policy orders done on the way, the manifest saying what
// was done, and the note saying what the run was.
//
// # What this shares with the preview beside it, and why that matters
//
// The same plan, the same engine, the same writer chosen the same way. That is
// not tidiness: it is what makes the rows somebody looked at before pressing a
// button the rows they get after pressing it. Two assemblies would be two
// answers, and the second one would only be found by somebody comparing a
// screen with a file.
//
// # Why both surfaces call this rather than each assembling a run
//
// 🔴 They did, and they disagreed about the one thing this product is for. The
// command line carried out the policy of a recipe and the recipe screen did
// not, so one recipe gave damaged data and a manifest in a terminal and clean
// data in a window, with nothing saying so on either side (B-300). One
// assembly makes that impossible rather than unlikely.
//
// # Why nothing appears under its real name until the whole run has succeeded
//
// The set of files renames at the end, so a run that fails or is stopped leaves
// nothing that looks like a finished dataset, and no manifest describing
// records nobody will find. A half written folder is worse than no folder,
// because it is the kind of thing that gets committed to somebody's test
// repository.
//
// The halt is handed in rather than made here, because what can stop a run is a
// property of the surface: a terminal is interrupted by a signal and a window by
// somebody pressing a button.
func (p Prepared) Write(settings Settings, stop generate.Halt, at time.Time) (Written, error) {
	rec, source, plan := p.rec, p.source, p.plan
	set := store.New(settings.Into, settings.Replace)
	committed := false
	defer func() {
		if !committed {
			set.Discard()
		}
	}()

	files := &named{set: set}
	writer, err := format.For(files, rec, stop)
	if err != nil {
		return Written{}, err
	}
	// A recipe with a policy has its damage done on the way to the disk, and
	// the manifest joins this same set of files. The first pillar is untouched
	// by this: a recipe without a policy gets the writer it always got.
	writer, broken, err := withDamage(rec, source, writer, set, settings.Manifest)
	if err != nil {
		return Written{}, err
	}
	if err := generate.Run(plan, rec, source, writer, stop); err != nil {
		return Written{}, err
	}
	written := Written{Rows: plan.TotalRows(), Plan: plan, Damaged: broken != nil}
	if broken != nil {
		if err := closeManifest(broken, rec, plan, at); err != nil {
			return Written{}, err
		}
		written.Counts = broken.Counts()
	}
	if err := writeRunNote(set, rec, plan, settings.Overrides); err != nil {
		return Written{}, err
	}
	// Nothing is asked about a halt from here on. Everything is written and
	// what remains is giving the files their real names, and a rename stopped
	// half way through the set would leave exactly the half finished dataset
	// this whole arrangement exists to prevent.
	if err := set.Commit(); err != nil {
		return Written{}, err
	}
	committed = true
	written.Files = Writes(files.names, written.Damaged, settings.Manifest)
	if broken != nil {
		written.Oversized = damage.Oversized(settings.Into, settings.Manifest)
	}
	return written, nil
}

// named is the set of files, remembering what the writer asked of it.
type named struct {
	set   *store.Set
	names []string
}

func (n *named) Create(name string) (format.WriteCloser, error) {
	file, err := n.set.Create(name)
	if err != nil {
		return nil, err
	}
	n.names = append(n.names, name)
	return file, nil
}
