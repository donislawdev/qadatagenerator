// Package take is the mode where somebody else's file goes in, and the same
// file with damage in it comes out beside a manifest saying what was done.
//
// # What makes this different from the mode that generates
//
// There, a recipe declares what every column holds and five suppliers say what
// a run can do. Here nobody declared anything, so the shape is worked out by
// looking at the values, and the manifest carries the evidence for what was
// worked out. Everything downstream of that is the same engine, the same
// selector and the same writer, which is deliberate: damage that behaved
// differently depending on where the records came from would be a second
// product.
//
// # Why this is a package of its own rather than part of a surface
//
// It was part of one, the command line, for as long as the command line was
// the only caller. A screen doing the same work cannot import that surface,
// because the layer test forbids the two surfaces to know each other, and the
// rule is not bookkeeping: it is what keeps one of them from quietly becoming
// a reimplementation of the other.
//
// So what is here is everything both surfaces need to agree on, which is what
// a run refuses, what it works out, what it would do and what it did. What is
// not here is a single word meant for a person to read.
package take

import (
	"path/filepath"
	"time"

	"github.com/donislawdev/qadatagenerator/internal/adapter/format"
	"github.com/donislawdev/qadatagenerator/internal/adapter/infer"
	"github.com/donislawdev/qadatagenerator/internal/adapter/mutators"
	"github.com/donislawdev/qadatagenerator/internal/adapter/sourcecsv"
	"github.com/donislawdev/qadatagenerator/internal/adapter/store"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/corrupt"
	"github.com/donislawdev/qadatagenerator/internal/damage"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

const (
	// Suffix is what tells the damaged file from the one it came from.
	//
	// 🔴 It exists because this is the only mode that reads and writes in one
	// run, so it is the only one that could destroy what it was given. The
	// suffix makes that collision impossible by construction rather than by the
	// caller remembering to point the output somewhere else.
	Suffix = ".corrupted.csv"

	// DefaultSeed is what the draws come from when nobody said.
	//
	// A recipe carries its seed and a file does not, so this mode needs one of
	// its own. It is a fixed word rather than a clock reading, because two runs
	// of one command have to damage the same rows: a seed taken from the time
	// would make this the one mode whose output nobody can reproduce.
	DefaultSeed = "0"

	// DefaultLocale is what the envelope records when nobody said.
	//
	// It matches what the source of values defaults to, so the two cannot
	// disagree. In this mode it decides less than it looks: which numbering
	// scheme a column follows is worked out from the values and not from the
	// locale, and the manifest says so in its own section.
	DefaultLocale = "en-US"

	// DefaultDelimiter is the field separator assumed when nobody said.
	DefaultDelimiter = ","

	// DefaultOut is where files are written when nobody said.
	DefaultOut = "."

	// InferenceMethod names how a schema was worked out, for the manifest.
	//
	// It is a name rather than a sentence because a reader comparing two
	// manifests needs to tell one way of looking from another at a glance. What
	// the way of looking actually is belongs in the basis of each column, which
	// is where a person reads it.
	InferenceMethod = "values-validated"
)

// Settings is what a run of this mode was asked for, as values rather than as
// the flags or the fields somebody typed them into.
type Settings struct {
	// Out is the directory the files are written to.
	Out string
	// Seed is what makes the choice of damage repeatable.
	Seed string
	// Layers is which kinds of damage were ordered.
	//
	// 🔴 There is no default and that is a decision rather than an omission.
	// Choosing them would be deciding what "damage this file" means on somebody
	// else's behalf, and the cheapest default to defend, everything, would take
	// the bytes and the container apart on a file whose owner wanted to test an
	// import. Giving this a default later breaks nothing, because every run
	// without it is refused today. Changing a default later breaks every
	// pipeline that relied on it.
	Layers []string
	// Rows is the share of records to damage.
	Rows string
	// PerRow is how many defects one record may carry.
	PerRow string
	// ExcludeFields are the columns to leave alone.
	ExcludeFields []string
	// Delimiter is the character separating fields, both read and written.
	Delimiter string
	// Locale is what the envelope records.
	Locale string
	// Overwrite says whether files already there may be replaced.
	Overwrite bool
	// Manifest is the extra form the manifest is exported in, empty for none.
	Manifest string
}

// Defaults is the settings a run has before anybody changes anything.
//
// Layers is deliberately absent from it, for the reason written where the field
// is declared.
func Defaults() Settings {
	return Settings{
		Out: DefaultOut, Seed: DefaultSeed, Delimiter: DefaultDelimiter,
		Locale: DefaultLocale,
	}
}

// Policy turns the settings into the policy the engine takes.
//
// It builds the same value a recipe would have produced, so that the engine
// cannot tell where the policy came from. A second shape for the same thing
// would be a second set of rules about what a policy means.
func (s Settings) Policy() recipe.Policy {
	return recipe.Policy{
		Layers: s.Layers, Rows: s.Rows, PerRow: s.PerRow,
		ExcludeFields: s.ExcludeFields,
	}
}

// OutputName is what the damaged file is called.
func OutputName(source string) string {
	return sourcecsv.EntityOf(source) + Suffix
}

// CheckOutput refuses a run that would write over the file it is reading.
//
// # Why both the suffix and the check
//
// The suffix makes the collision impossible by construction, so this can never
// fire while the name is built that way. It stays because it guards something
// that cannot be taken back: this is the only mode that reads and writes at
// once, and the day somebody changes how the name is built, a redundant check
// is what stands between that change and a person losing the file they handed
// over. It has a mutation of its own for the same reason.
func CheckOutput(source string, settings Settings) Refusal {
	if sameFile(filepath.Join(settings.Out, OutputName(source)), source) {
		return Refusal{Kind: WouldOverwriteSource, Value: source}
	}
	return Refusal{}
}

// sameFile reports whether two paths name one place, as far as comparing text
// can tell.
//
// It compares cleaned absolute paths. That answers the case this guards, which
// is a directory and a name that meet, and it does not answer links or two
// mounts of one volume. Saying so here is cheaper than a reader assuming it
// covers more: what covers the rest is the suffix above, which keeps the two
// names apart whatever the file system thinks.
func sameFile(written, source string) bool {
	left, err := filepath.Abs(written)
	if err != nil {
		return false
	}
	right, err := filepath.Abs(source)
	if err != nil {
		return false
	}
	return filepath.Clean(left) == filepath.Clean(right)
}

// Opened is somebody else's file, read once and understood as far as looking at
// the values allows.
//
// Nothing has been written when this exists, and that matters to both surfaces:
// the command line shows what was worked out before it damages anything, and a
// screen shows it while somebody is still deciding.
type Opened struct {
	file  *sourcecsv.File
	shape infer.Schema
	from  values.Source
	path  string
}

// Open reads the file and works out what its columns hold.
//
// Whoever calls this owns closing it, which is why the handle is held rather
// than the contents: a screen keeps the file open while somebody chooses
// settings, and a run that never starts must not have read a large file into
// memory for nothing.
func Open(path string, settings Settings) (*Opened, error) {
	file, err := sourcecsv.Open(path, []rune(settings.Delimiter)[0])
	if err != nil {
		return nil, err
	}
	from := values.New(settings.Locale, settings.Seed)
	return &Opened{
		file: file, shape: infer.From(file.Table(), file.Samples(), from.Schemes()),
		from: from, path: path,
	}, nil
}

// Close releases the file.
func (o *Opened) Close() error { return o.file.Close() }

// Table is the entity, its columns and how many records it holds.
func (o *Opened) Table() dataset.Table { return o.file.Table() }

// Shape is what the columns were taken to hold, with the basis of each finding.
func (o *Opened) Shape() infer.Schema { return o.shape }

// HadByteOrderMark reports whether the file began with a byte order mark, which
// is worth telling somebody because the file written back does not have one.
func (o *Opened) HadByteOrderMark() bool { return o.file.HadByteOrderMark() }

// Plan is what a run would do, without doing any of it.
//
// It states what is certain rather than estimating. How many rows a share of
// them comes to is a draw, and a number here that the real run then missed
// would be worse than no number.
type Plan struct {
	// Ordered is how many kinds of damage the named layers describe.
	Ordered int
	// Implemented is how many of those this build can actually carry out.
	Implemented int
	// Writes are the files the run would leave behind.
	Writes []string
	// Out is where it would leave them.
	Out string
}

// Plan says what the policy asks for and writes nothing.
func (o *Opened) Plan(settings Settings, described catalog.Catalog) Plan {
	coverage := mutators.Coverage{Described: described}
	ordered, implemented := 0, 0
	for _, layer := range settings.Layers {
		inLayer := len(described.InLayer(catalog.Layer(layer)))
		ordered += inLayer
		implemented += inLayer - coverage.Unimplemented(layer)
	}
	return Plan{
		Ordered: ordered, Implemented: implemented, Out: settings.Out,
		Writes: append([]string{OutputName(o.path)}, damage.Files(settings.Manifest)...),
	}
}

// Result is what a finished run left behind.
type Result struct {
	// Written are the files, in the order a person is told about them.
	Written []string
	// Out is the directory they are in.
	Out string
	// Counts is what the run did, as numbers.
	Counts damage.Counts
}

// Corrupt runs the engine over the file and puts everything on the disk, or
// leaves the directory exactly as it found it.
//
// The halt is handed in rather than made here, because what can stop a run is a
// property of the surface: a terminal is interrupted by a signal and a window by
// somebody pressing a button.
func (o *Opened) Corrupt(settings Settings, described catalog.Catalog,
	stop corrupt.Halt, at time.Time) (Result, error) {
	files := store.New(settings.Out, settings.Overwrite)
	committed := false
	defer func() {
		if !committed {
			files.Discard()
		}
	}()

	run, written, err := o.start(settings, described, files)
	if err != nil {
		return Result{}, err
	}
	if err := corrupt.Run(o.file, run.Writer(), stop); err != nil {
		return Result{}, err
	}
	if err := run.Finish(o.envelope(settings, run, at)); err != nil {
		return Result{}, err
	}
	if err := files.Commit(); err != nil {
		return Result{}, err
	}
	committed = true
	return Result{
		Written: append([]string{written}, damage.Files(settings.Manifest)...),
		Out:     settings.Out, Counts: run.Counts(),
	}, nil
}

// start assembles the writer, the engine and the manifest, and says what the
// damaged file is called.
func (o *Opened) start(settings Settings, described catalog.Catalog,
	files *store.Set) (*damage.Run, string, error) {
	written := OutputName(o.path)
	writer, err := format.NewPlainCSV(fileMaker{files}, written, settings.Delimiter)
	if err != nil {
		return nil, "", err
	}
	run, err := damage.Start(damage.Ask{
		Files: files, Next: writer, Described: described, Schema: o.shape,
		Policy: settings.Policy(), Seed: settings.Seed, Can: o.canDo(),
		Schemes: o.from, Form: settings.Manifest, WidestRows: o.file.Table().Rows,
	})
	if err != nil {
		return nil, "", err
	}
	return run, written, nil
}

// canDo gathers what this run is able to do when nothing declared anything.
//
// Four of the five suppliers answer as they always do. The one that changes is
// the schema, which here is what was worked out by looking rather than what
// somebody wrote down, and it therefore brings much less: no maximum length, no
// required column, no key and no relation. That is the point rather than a
// shortfall, and every entry it leaves inapplicable says so in the manifest.
func (o *Opened) canDo() capability.Set {
	names := format.CapabilitiesOf(format.NameCSV)
	names = append(names, recipe.CapabilitiesOf(o.shape)...)
	names = append(names, values.CapabilitiesOfView(o.shape, o.from)...)
	names = append(names, capability.UnicodeNormalization)
	return capability.NewSet(names...)
}

// envelope is what the manifest carries about the run as a whole, including the
// evidence for everything this mode had to guess.
func (o *Opened) envelope(settings Settings, run *damage.Run, at time.Time) manifest.Run {
	return manifest.Run{
		GeneratedAt:    at.UTC().Format(time.RFC3339),
		Seed:           settings.Seed,
		Locale:         settings.Locale,
		CatalogVersion: version.Catalog,
		Source:         o.path,
		Inference:      o.inference(),
		RowsTotal:      o.file.Table().Rows,
		RowsMutated:    run.RowsMutated(),
	}
}

// inference turns what was worked out into what the manifest carries.
func (o *Opened) inference() *manifest.Inference {
	record := &manifest.Inference{Method: InferenceMethod, NotInferred: infer.NotInferred()}
	for _, finding := range o.shape.Findings() {
		record.Columns = append(record.Columns, manifest.InferredColumn{
			Field: finding.Field, Type: finding.Type,
			Scheme: finding.Scheme, Basis: finding.Basis,
		})
	}
	for _, relation := range o.shape.Relations() {
		record.Relations = append(record.Relations, manifest.InferredRelation{
			Field: relation.Field, With: relation.With,
			Kind: string(relation.Kind), Basis: relation.Basis,
		})
	}
	return record
}

// fileMaker gives the format writer somewhere to put its bytes, through the
// same set of files everything else in the run goes through.
type fileMaker struct{ set *store.Set }

func (m fileMaker) Create(name string) (format.WriteCloser, error) {
	return m.set.Create(name)
}
