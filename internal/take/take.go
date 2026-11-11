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
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/donislawdev/qadatagenerator/internal/adapter/format"
	"github.com/donislawdev/qadatagenerator/internal/adapter/infer"
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
	"github.com/donislawdev/qadatagenerator/internal/fault"
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

// ColumnsListed is how many worked out columns get an account of their own on a
// surface before it starts counting instead.
//
// # Why there is a number here at all
//
// There was not one, and the reason written down was that a column nothing was
// recognised in is exactly the one somebody needs to know about. That reason is
// true of a file with twenty columns and false of a file with five thousand,
// where nobody reads any of it. Measured 2026-09-02 with tools/kanal_i_pamiec.py:
// a file of 5000 columns and 86 670 bytes puts 804 476 bytes on the log channel
// in 5005 lines, nine times the file being described. At 500 columns it is
// 80 475 bytes and at 50 it is 8524, which is a screen and a bit. So the cut is
// between a list somebody reads and a list nobody does, and fifty is the top of
// the first. Recorded as DEC-200.
//
// # Why one number for both surfaces rather than one each
//
// The log and the panel show the same conclusions to the same person. Two
// numbers would mean the surfaces disagreed about how much of a file they
// account for, which is the rule about parity broken over an implementation
// detail. The cost per column does differ between them, measured on 2026-09-03
// as about 170 bytes of log against 1088 bytes of widgets, and that is a real
// difference which does not bear on how much a person reads. The cost is what
// each surface lives with and the count is shared.
//
// 🔴 The sentence that used to close this reasoning said the manifest carries
// every column either way. That stopped being true on 2026-09-03: the manifest
// stops at manifest.ColumnsDescribed and records the width of the file beside
// it. Nothing is dropped in silence anywhere, and each of the three says how
// much of the file it covered.
const ColumnsListed = 50

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

// Beside is the settings with the directory to write to counted from the
// folder of the source: that folder where no directory was named, and a
// directory named without its root taken inside it.
//
// # Why the window asks this and the command line does not
//
// 🔴 A run counts a directory without a root from where the person typing
// stands, and DefaultOut is that place itself. A window has no such place:
// started from a file manager its directory is the one the program lives in,
// and the damaged copy landed there while the window promised it beside the
// file (B-302, B-207). The folder of the file is the one place a person at the
// window knows, so the window counts from it, and the command line keeps what
// "." has always meant in a terminal.
//
// A path rooted without being whole, which on Windows is one starting at a
// separator or naming a drive and nothing after it, is left as it was typed.
// Taken inside the folder it would become a path nobody wrote, and as typed it
// still names the drive somebody meant.
func (s Settings) Beside(source string) Settings {
	folder := filepath.Dir(source)
	switch {
	case s.Out == "":
		s.Out = folder
	case !rooted(s.Out):
		s.Out = filepath.Join(folder, s.Out)
	}
	return s
}

// rooted reports whether a path says where it starts, whole or not.
func rooted(path string) bool {
	return path != "" &&
		(filepath.IsAbs(path) || filepath.VolumeName(path) != "" || os.IsPathSeparator(path[0]))
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
// # Every file the run writes, not only the damaged one
//
// 🔴 The suffix keeps the damaged file apart from its source by construction,
// and until 2026-10-26 that file was the only one asked about. The manifest
// and its export are written into the same directory under names fixed by this
// tool, so a source called expectation-manifest.csv, damaged with --overwrite
// into its own directory, was replaced by the table of the manifest and the run
// ended with success. Found by an outside review on 2026-10-22. Every name the
// run writes is now held against the source, through the same function that
// names them to the person afterwards.
//
// The check against the damaged file stays although the suffix makes it
// unreachable, because it guards something that cannot be taken back: the day
// somebody changes how the name is built, it is what stands between that change
// and a person losing the file they handed over.
func CheckOutput(source string, settings Settings) Refusal {
	for _, name := range append([]string{OutputName(source)}, damage.Files(settings.Manifest)...) {
		if sameFile(filepath.Join(settings.Out, name), source) {
			return Refusal{Kind: WouldOverwriteSource, Value: source}
		}
	}
	return Refusal{}
}

// sameFile reports whether two paths name one place.
//
// The text is compared first, which answers a directory and a name that meet
// before anything exists. When both exist, the file system is asked as well,
// which answers what text cannot: one name in two cases on a system that does
// not tell them apart, and a link. Two mounts of one volume are still not
// answered, and saying so is cheaper than a reader assuming they are.
func sameFile(written, source string) bool {
	left, err := filepath.Abs(written)
	if err != nil {
		return false
	}
	right, err := filepath.Abs(source)
	if err != nil {
		return false
	}
	if filepath.Clean(left) == filepath.Clean(right) {
		return true
	}
	there, err := os.Stat(left)
	if err != nil {
		return false
	}
	read, err := os.Stat(right)
	return err == nil && os.SameFile(there, read)
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
//
// stop is asked between the records of the reading, which walks the whole file,
// and then before every column the working out looks at, whose cost grows with
// the columns rather than the rows (B-344, B-356). A read stopped in either
// leaves nothing open, because the reading closes what it opened and the file
// is opened again only when its records are taken.
func Open(path string, settings Settings, stop corrupt.Halt) (*Opened, error) {
	// 🔴 The separator is held against the same rule Check holds it against,
	// and this is the second place rather than the first. Every caller is meant
	// to have been refused already, so this line should be unreachable.
	//
	// It exists because of what happens when it is not. Taking the first rune
	// of an empty setting is a panic, and a panic on the command line is an
	// ugly ending while a panic in a window is the program disappearing while
	// somebody was typing. A screen puts a separator under a person's cursor,
	// so the empty value between two keystrokes is ordinary rather than
	// exotic.
	separator, usable := dataset.UsableSeparator(settings.Delimiter)
	if !usable {
		return nil, fault.New(fault.Config,
			"the field separator has to be one character other than a quote or a line end, "+
				"and this run was given "+strconv.Quote(settings.Delimiter))
	}
	file, err := sourcecsv.Open(path, separator, stop)
	if err != nil {
		return nil, err
	}
	from := values.New(settings.Locale, settings.Seed)
	shape, err := infer.From(file.Table(), file.Samples(), from.Schemes(), stop)
	if err != nil {
		return nil, err
	}
	return &Opened{file: file, shape: shape, from: from, path: path}, nil
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

// NotInferred names what looking at values can never establish.
//
// It is answered here rather than left to each surface, because the list is a
// property of the way of looking rather than of who is asking. A surface that
// forgot to mention it would be promising damage this mode cannot do.
func (o *Opened) NotInferred() []string { return infer.NotInferred() }

// Plan is what a run would do, without doing any of it: how much of the damage
// ordered this build carries out, split by layer, and which files it would
// leave where.
//
// The split is a fact about the run rather than about a surface, so it is
// worked out once, by the same walk over the catalog the recipe mode uses, and
// both surfaces say it in their own words (DEC-245, DEC-252).
type Plan struct {
	damage.Planned
	// Writes are the files the run would leave behind.
	Writes []string
	// Out is where it would leave them.
	Out string
}

// Plan says what the policy asks for and writes nothing.
func (o *Opened) Plan(settings Settings, described catalog.Catalog) Plan {
	return Plan{
		Planned: damage.PlanOf(settings.Layers, described),
		Out:     settings.Out,
		Writes:  append([]string{OutputName(o.path)}, damage.Files(settings.Manifest)...),
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
	// Oversized are the files this run wrote that this build will not read
	// back, if any. It is a fact rather than a sentence, so that each surface
	// says it in its own words.
	Oversized []damage.Oversize
}

// Corrupt runs the engine over the file and puts everything on the disk, or
// leaves the directory exactly as it found it.
//
// The halt is handed in rather than made here, because what can stop a run is a
// property of the surface: a terminal is interrupted by a signal and a window by
// somebody pressing a button.
func (o *Opened) Corrupt(settings Settings, described catalog.Catalog,
	stop corrupt.Halt, at time.Time) (Result, error) {
	// 🔴 The walk belongs to this run, so this run ends it, however it ends.
	//
	// The reader goes over the file twice and the second pass is what a run
	// consumes. A run stopped between two records leaves that pass part way
	// through, and the NEXT run of the same opened file would then carry on
	// from where the last one stopped: a damaged file holding the records that
	// were left, beside a manifest counting the whole file. Nothing about that
	// result looks wrong.
	//
	// It became reachable the day a window existed. A command line run opens a
	// file, damages it once and ends. A screen keeps the file while somebody
	// reads what it holds, and stopping a run and pressing the button again is
	// the ordinary thing to do.
	defer func() { _ = o.file.Close() }()

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
		Written:   append([]string{written}, damage.Files(settings.Manifest)...),
		Out:       settings.Out,
		Counts:    run.Counts(),
		Oversized: damage.Oversized(settings.Out, settings.Manifest),
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
		// One file is one table, named the way every entry of this run names
		// it, so a response record about it may leave the entity out.
		Entities: []manifest.Entity{{Name: o.file.Table().Entity, Rows: o.file.Table().Rows}},
	}
}

// inference turns what was worked out into what the manifest carries.
//
// # Why the columns stop and the count does not
//
// An entry costs about two hundred bytes, so a file wide enough turns this one
// section into a manifest larger than this build reads back. The account stops
// at manifest.ColumnsDescribed and the width of the file is recorded whole, so
// that a reader of a truncated account can tell it from a complete one and work
// out exactly how much is missing. A count that stopped with the list would
// have made the two indistinguishable, which is the failure this guards.
func (o *Opened) inference() *manifest.Inference {
	findings := o.shape.Findings()
	record := &manifest.Inference{
		Method:       InferenceMethod,
		NotInferred:  infer.NotInferred(),
		ColumnsTotal: len(findings),
	}
	if len(findings) > manifest.ColumnsDescribed {
		findings = findings[:manifest.ColumnsDescribed]
	}
	for _, finding := range findings {
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
