package corrupted_test

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/corrupted"
	"github.com/donislawdev/qadatagenerator/internal/app/corrupt"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// The three entries of the fixture, chosen so that one run produces every
// answer a mutator can give and one answer no mutator gives at all.
const (
	damageApplied       catalog.ID = "L2.type.string_in_numeric"
	damageDeclined      catalog.ID = "L2.string.empty"
	damageUnimplemented catalog.ID = "L2.number.comma_decimal_separator"
)

// stated is damage that behaves exactly as a test says.
type stated struct {
	id      catalog.ID
	outcome mutator.Outcome
}

func (s stated) ID() catalog.ID                        { return s.id }
func (s stated) Damage(mutator.Target) mutator.Outcome { return s.outcome }

type registry map[catalog.ID]mutator.Mutator

func (r registry) Lookup(id catalog.ID) (mutator.Mutator, bool) {
	found, ok := r[id]
	return found, ok
}

// noSchemes is a build carrying no numbering scheme, which is what this fixture
// needs and what the mode damaging somebody else's file starts from.
type noSchemes struct{}

func (noSchemes) Of(string) (provider.Provider, bool) { return nil, false }

// schema is a view of one table of one numeric column, which is all the engine
// is asked about here.
type schema struct{}

func (schema) EntityNames() []string { return []string{"invoice"} }
func (schema) FieldsOf(string) []recipe.FieldInfo {
	return []recipe.FieldInfo{{Name: "amount", Type: "integer"}}
}
func (schema) ParentsOf(string) []recipe.ParentInfo { return nil }
func (schema) RulesOf(string) []recipe.Rule         { return nil }
func (schema) PrimaryKeyOf(string) []string         { return nil }

// recorder is a sink that keeps every entry it was handed and writes its own
// name into a log shared with its peers, so that the order of the sinks is
// observable and not only the fact that both were fed.
type recorder struct {
	name    string
	entries []manifest.Defect
	log     *[]string
}

func (r *recorder) Add(defect manifest.Defect) error {
	r.entries = append(r.entries, defect)
	*r.log = append(*r.log, r.name)
	return nil
}

// silent is the writer the records reach once the damage is done to them.
type silent struct{ rows int }

func (s *silent) Begin(dataset.Table) error  { return nil }
func (s *silent) Write(dataset.Record) error { s.rows++; return nil }
func (s *silent) End(dataset.Table) error    { return nil }
func (s *silent) Finish() error              { return nil }

func entry(id catalog.ID, summary string) catalog.Entry {
	return catalog.Entry{
		ID:      id,
		Layer:   catalog.FieldShape,
		Summary: summary,
		AppliesTo: catalog.AppliesTo{
			FieldTypes: []string{"integer"},
			Formats:    []string{catalog.Any},
			Providers:  []string{catalog.Any},
		},
		Expectation: catalog.Rejection,
		Confidence:  catalog.Hard,
		Reversible:  true,
		Requires:    []capability.Name{capability.SchemaFieldType},
		Example:     catalog.Example{Before: "1250", After: "n/a"},
	}
}

// damaged drives one run of four records through the writer and hands back what
// counted them and what the entries were written to.
//
// It goes through the public door rather than reaching into the counters,
// because what both surfaces read is what a finished run says about itself and
// not what the fields held part way through.
func damaged(t *testing.T) (*corrupted.Writer, *recorder) {
	t.Helper()
	log := []string{}
	written := &recorder{name: "first", log: &log}
	writer, _, table := writerInto(t, corrupted.Tee{written}, 4)
	for row := 0; row < table.Rows; row++ {
		record := dataset.Record{Values: []string{"1250"}, Parent: dataset.NoParent}
		if err := writer.Write(record); err != nil {
			t.Fatalf("row %d was refused: %v", row, err)
		}
	}
	if err := writer.End(table); err != nil {
		t.Fatalf("the run refused to end: %v", err)
	}
	return writer, written
}

// writerInto builds the writer of the fixture over the given documents and
// begins a table of so many rows, every one of them ordered damage.
func writerInto(t *testing.T, sinks corrupted.Entries, rows int) (*corrupted.Writer, *silent, dataset.Table) {
	t.Helper()
	described, problems := catalog.New("0.1", "draft", []catalog.Entry{
		entry(damageApplied, "Puts text where the schema declares a number."),
		entry(damageDeclined, "Empties a field the schema requires."),
		entry(damageUnimplemented, "Writes a decimal with a comma for a point."),
	})
	if len(problems) > 0 {
		t.Fatalf("the fixture catalog was refused: %v", problems)
	}

	implementations := registry{
		damageApplied:  stated{id: damageApplied, outcome: mutator.Done("not a number")},
		damageDeclined: stated{id: damageDeclined, outcome: mutator.Skipped("the value was already empty")},
	}
	policy := recipe.Policy{
		Layers: []string{string(catalog.FieldShape)}, Rows: "100%", PerRow: "any",
	}
	selector, err := corrupt.NewSelector(policy, described, schema{}, seed.New("a fixed seed"))
	if err != nil {
		t.Fatalf("the fixture policy was refused: %v", err)
	}
	engine := corrupt.New(described, implementations, noSchemes{},
		capability.NewSet(capability.SchemaFieldType), seed.New("corrupt test"))

	next := &silent{}
	writer := corrupted.New(next, engine, &selector, sinks, schema{})
	table := dataset.Table{Entity: "invoice", Columns: []string{"amount"}, Rows: rows}
	if err := writer.Begin(table); err != nil {
		t.Fatalf("the run refused to begin: %v", err)
	}
	return writer, next, table
}

// refusing is a document that cannot take any entry.
type refusing struct{ handed int }

var errNoRoom = errors.New("the document has no room for the entry")

func (r *refusing) Add(manifest.Defect) error {
	r.handed++
	return errNoRoom
}

// 🔴 An entry no document can record ends the run before its record goes on.
//
// Until 2026-10-28 the refusal was held by the document and said only when it
// closed, so a run went on to produce every remaining record of a dataset that
// the failure at its close then threw away (found by an outside review on
// 2026-10-22). A record passed on after its entry was refused is a record whose
// damage nothing describes.
func TestAnEntryNoDocumentCanRecordEndsTheRunBeforeItsRecordGoesOn(t *testing.T) {
	refused := &refusing{}
	writer, next, _ := writerInto(t, corrupted.Tee{refused}, 4)
	err := writer.Write(dataset.Record{Values: []string{"1250"}, Parent: dataset.NoParent})
	if !errors.Is(err, errNoRoom) {
		t.Fatalf("the first record was written with %v although its entry was refused", err)
	}
	if refused.handed != 1 {
		t.Errorf("the document was handed %d entries, want the one it refused", refused.handed)
	}
	if next.rows != 0 {
		t.Errorf("%d records reached the data after their entry was refused", next.rows)
	}
}

// The documents after the one that refused an entry are not handed it, because
// the run ends there and none of them will be kept.
func TestTheFirstSinkToRefuseAnEntryIsTheLastOneHandedIt(t *testing.T) {
	log := []string{}
	first := &recorder{name: "first", log: &log}
	refused := &refusing{}
	third := &recorder{name: "third", log: &log}

	err := corrupted.Tee{first, refused, third}.Add(manifest.Defect{ID: "d1"})
	if !errors.Is(err, errNoRoom) {
		t.Fatalf("three documents, one refusing, took the entry with %v", err)
	}
	if len(first.entries) != 1 || refused.handed != 1 {
		t.Errorf("the documents before the refusal were handed %d and %d entries, want 1 and 1",
			len(first.entries), refused.handed)
	}
	if len(third.entries) != 0 {
		t.Error("a document after the one that refused was still handed the entry")
	}
}

// The counts are kept here rather than gathered from the manifest afterwards,
// because the entries are on a disk by then. That makes them a second telling
// of the same run, and a second telling can disagree with the first without
// anything failing, which is what this asks about.
func TestWhatTheRunCountedAgreesWithWhatItWroteToTheManifest(t *testing.T) {
	writer, written := damaged(t)

	requested, applied := 0, 0
	rows := map[int]bool{}
	for _, defect := range written.entries {
		requested++
		if defect.Applied {
			applied++
			rows[defect.Location.Row] = true
		}
	}

	if applied == 0 || applied == requested {
		t.Fatalf("the fixture ordered %d defects and applied %d of them, so this test "+
			"compared only one of the two paths", requested, applied)
	}
	if writer.Requested() != requested {
		t.Errorf("the run says it ordered %d defects and wrote %d entries",
			writer.Requested(), requested)
	}
	if writer.Applied() != applied {
		t.Errorf("the run says %d defects reached the data and %d entries say they did",
			writer.Applied(), applied)
	}
	if writer.RowsMutated() != len(rows) {
		t.Errorf("the run says %d records carry damage and the entries name %d",
			writer.RowsMutated(), len(rows))
	}
}

// A count of what was applied is true and leaves a false impression of the
// scope, so the shortfall is broken down. These lines reach a person through
// both surfaces and nothing else in this repository names them.
func TestTheShortfallIsBrokenDownByKindAndAccountsForEveryOrderThatFailed(t *testing.T) {
	writer, _ := damaged(t)

	lines := writer.ByKind()
	if len(lines) == 0 {
		t.Fatal("a run with refused orders broke the shortfall down into nothing")
	}

	counted := 0
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[1] != "because" {
			t.Errorf("a line of the breakdown does not say how many and why: %q", line)
			continue
		}
		many, err := strconv.Atoi(fields[0])
		if err != nil {
			t.Errorf("a line of the breakdown does not start with a count: %q", line)
			continue
		}
		counted += many
		if strings.TrimSpace(strings.Join(fields[2:], " ")) == "" {
			t.Errorf("a line of the breakdown counts %d and says nothing: %q", many, line)
		}
	}

	if missing := writer.Requested() - writer.Applied(); counted != missing {
		t.Errorf("%d orders did not reach the data and the breakdown accounts for %d",
			missing, counted)
	}
}

// Two documents describing one run have to be comparable line by line, and the
// kinds are gathered from a map. Asking three times is what makes an order that
// depends on the draw fail rather than pass four times in five.
func TestTheBreakdownReadsTheSameWayEveryTimeItIsAsked(t *testing.T) {
	writer, _ := damaged(t)

	first := strings.Join(writer.ByKind(), "\n")
	for again := 0; again < 3; again++ {
		if now := strings.Join(writer.ByKind(), "\n"); now != first {
			t.Fatalf("the breakdown came back in another order:\n%s\n\nrather than:\n%s",
				now, first)
		}
	}
}

// The order is kept rather than left to a map, because two documents describing
// one run have to be comparable line by line. Nothing else in this repository
// names this, and a sink dropped from the middle would leave one document short
// without any run failing.
func TestEverySinkIsHandedEveryEntryInTheOrderTheSinksWereGiven(t *testing.T) {
	log := []string{}
	first := &recorder{name: "first", log: &log}
	second := &recorder{name: "second", log: &log}
	third := &recorder{name: "third", log: &log}

	sinks := corrupted.Tee{first, second, third}
	sinks.Add(manifest.Defect{ID: "d1"})
	sinks.Add(manifest.Defect{ID: "d2"})

	want := "first second third first second third"
	if now := strings.Join(log, " "); now != want {
		t.Errorf("the entries reached the sinks as %q rather than %q", now, want)
	}
	for _, sink := range []*recorder{first, second, third} {
		if len(sink.entries) != 2 {
			t.Errorf("the sink %q was handed %d of the two entries", sink.name, len(sink.entries))
			continue
		}
		if sink.entries[0].ID != "d1" || sink.entries[1].ID != "d2" {
			t.Errorf("the sink %q was handed %q and %q",
				sink.name, sink.entries[0].ID, sink.entries[1].ID)
		}
	}
}
