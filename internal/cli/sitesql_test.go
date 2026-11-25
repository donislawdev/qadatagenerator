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

// The sample of a SQL file on the Recipes page is what the SQL form writes for
// the recipe above, byte for byte, in both languages.
//
// The sample stands in a block the audit lets carry the semicolons of
// statements (DEC-341), so nothing else would notice it drifting from the
// writer: a change to the quoting or to the start of a file would leave the
// page showing a file this build no longer writes.
func TestTheRecipesPageShowsWhatTheSQLFormWrites(t *testing.T) {
	decoded := recipeyaml.Decode([]byte(sqlPageRecipe))
	if len(decoded.Problems) > 0 {
		t.Fatalf("the recipe of the sample is refused: %+v", decoded.Problems)
	}
	rec := decoded.Recipe
	source := values.New(rec.Locale, rec.Seed)
	plan, err := generate.Prepare(rec, source, generate.NeverStops{})
	if err != nil {
		t.Fatalf("the recipe of the sample cannot be planned: %v", err)
	}
	files := sqlFiles{}
	writer, err := format.NewSQL(files, rec)
	if err != nil {
		t.Fatalf("the writer refused the recipe of the sample: %v", err)
	}
	if err := generate.Run(plan, rec, source, writer, generate.NeverStops{}); err != nil {
		t.Fatalf("the run of the sample failed: %v", err)
	}
	written := files["dataset.sql"].String()

	for _, language := range []string{"en", "pl"} {
		path := filepath.Join("..", "..", "web", "content", language, "docs", "recipes.md")
		page, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("the Recipes page in %s cannot be read: %v", language, err)
		}
		_, after, opened := strings.Cut(string(page), "```sql\n")
		shown, _, closed := strings.Cut(after, "```")
		if !opened || !closed {
			t.Fatalf("the Recipes page in %s has no sample of a SQL file", language)
		}
		if shown != written {
			t.Errorf("the Recipes page in %s shows:\n%s\nand the SQL form writes:\n%s", language, shown, written)
		}
	}
}
