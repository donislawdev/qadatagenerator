package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/format"
	"github.com/donislawdev/qadatagenerator/internal/adapter/recipeyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
)

// sqlPageRecipe is the recipe behind the sample of a SQL file the Recipes page
// shows, in English and in Polish.
const sqlPageRecipe = `version: "0.1"
seed: "sql-page"
locale: en-US
entities:
  customer:
    count: 2
    primary_key: [customer_id]
    fields:
      customer_id: { type: integer, unique: true }
      last_name:   { generator: last_name }
      coupon:      { values: [SPRING10, WELCOME], empty: 50% }
  order:
    belongs_to: { customer: 1-1 }
    fields:
      total: { type: decimal }
output:
  format: sql
  dialect: postgresql
`

// sqlFiles holds what a writer writes, in memory.
type sqlFiles map[string]*strings.Builder

func (f sqlFiles) Create(name string) (format.WriteCloser, error) {
	built := &strings.Builder{}
	f[name] = built
	return closing{built}, nil
}

type closing struct{ *strings.Builder }

func (closing) Close() error { return nil }

// sqlPageFile is what the SQL form writes for the recipe of the sample, with
// the tables created or without them.
func sqlPageFile(t *testing.T, createTables bool) string {
	t.Helper()
	decoded := recipeyaml.Decode([]byte(sqlPageRecipe))
	if len(decoded.Problems) > 0 {
		t.Fatalf("the recipe of the sample is refused: %+v", decoded.Problems)
	}
	rec := decoded.Recipe
	rec.Output.CreateTables = createTables
	source := values.New(rec.Locale, rec.Seed)
	plan, err := generate.Prepare(rec, source, generate.NeverStops{})
	if err != nil {
		t.Fatalf("the recipe of the sample cannot be planned: %v", err)
	}
	files := sqlFiles{}
	writer, err := format.NewSQL(files, rec, source)
	if err != nil {
		t.Fatalf("the writer refused the recipe of the sample: %v", err)
	}
	if err := generate.Run(plan, rec, source, writer, generate.NeverStops{}); err != nil {
		t.Fatalf("the run of the sample failed: %v", err)
	}
	return files["dataset.sql"].String()
}

// The samples of a SQL file on the Recipes page are what the SQL form writes
// for the recipe above, byte for byte, in both languages: the file, then the
// tables it starts with when the recipe asks for them.
//
// The samples stand in blocks the audit lets carry the semicolons of
// statements (DEC-341), so nothing else would notice them drifting from the
// writer: a change to the quoting, to the start of a file or to the type of a
// column would leave the page showing a file this build no longer writes.
func TestTheRecipesPageShowsWhatTheSQLFormWrites(t *testing.T) {
	written := sqlPageFile(t, false)
	var created []string
	for statement := range strings.SplitAfterSeq(sqlPageFile(t, true), ");\n") { // punctuation: test data
		if at := strings.Index(statement, "CREATE TABLE "); at >= 0 {
			created = append(created, statement[at:])
		}
	}
	if len(created) != 2 {
		t.Fatalf("the sample creates %d tables, want 2", len(created))
	}
	want := []string{written, strings.Join(created, "\n")}

	for _, language := range []string{"en", "pl"} {
		path := filepath.Join("..", "..", "web", "content", language, "docs", "recipes.md")
		page, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("the Recipes page in %s cannot be read: %v", language, err)
		}
		rest := string(page)
		for index, sample := range want {
			_, after, opened := strings.Cut(rest, "```sql\n")
			shown, later, closed := strings.Cut(after, "```")
			if !opened || !closed {
				t.Fatalf("the Recipes page in %s has no sample %d of a SQL file", language, index+1)
			}
			if shown != sample {
				t.Errorf("sample %d of the Recipes page in %s shows:\n%s\nand the SQL form writes:\n%s",
					index+1, language, shown, sample)
			}
			rest = later
		}
	}
}
