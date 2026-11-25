package format_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/donislawdev/qadatagenerator/internal/adapter/format"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// sqlRecipe is a recipe for the SQL form of one database, of the entities given.
func sqlRecipe(dialect string, perEntity bool, entities ...recipe.Entity) recipe.Recipe {
	return recipe.Recipe{
		Version: "0.1", Seed: "sql", Locale: "en-US",
		Output:   recipe.Output{Format: format.NameSQL, Dialect: dialect, FilePerEntity: perEntity},
		Entities: entities,
	}
}

// table is a planned table of one entity and the records handed to the writer.
type table struct {
	planned dataset.Table
	records []dataset.Record
}

// writtenBy hands tables to the writer of one database directly, so that a
// test can give it values no generator draws, and returns the files.
func writtenBy(t *testing.T, rec recipe.Recipe, tables ...table) *held {
	t.Helper()
	files := newHeld()
	writer, err := format.NewSQL(files, rec)
	if err != nil {
		t.Fatalf("the writer was refused: %v", err)
	}
	for _, one := range tables {
		if err := writer.Begin(one.planned); err != nil {
			t.Fatalf("table %s did not begin: %v", one.planned.Entity, err)
		}
		for _, record := range one.records {
			if err := writer.Write(record); err != nil {
				t.Fatalf("a record of %s was refused: %v", one.planned.Entity, err)
			}
		}
		if err := writer.End(one.planned); err != nil {
			t.Fatalf("table %s did not end: %v", one.planned.Entity, err)
		}
	}
	if err := writer.Finish(); err != nil {
		t.Fatalf("the writer did not finish: %v", err)
	}
	return files
}

// oneColumn is a table t of one column v holding the given values, one row each.
func oneColumn(values ...string) table {
	records := make([]dataset.Record, 0, len(values))
	for _, value := range values {
		records = append(records, dataset.Record{Values: []string{value}, Parent: dataset.NoParent})
	}
	return table{planned: dataset.Table{Entity: "t", Columns: []string{"v"}, Rows: len(values)}, records: records}
}

var entityT = recipe.Entity{Name: "t", Count: 1, HasCount: true, Fields: []recipe.Field{{Name: "v"}}}

// Every database is handed every value in the form it reads back byte for byte.
//
// The forms are the ones measured on 2026-11-25 (DEC-341): a backslash is an
// escape in MySQL and MariaDB only, PostgreSQL keeps a plain value plain and
// takes the escape form for the rest, SQL Server needs the N and has no escapes,
// and a line break, a zero byte and the substitute character never reach the
// file as they are.
func TestEachDatabaseIsHandedEveryValueInTheFormItReadsBack(t *testing.T) {
	replacement := string(utf8.RuneError)
	values := []string{"plain", "O'Brien", `a\b`, "a\nb", "\r", "x\x00y", "\x1a",
		"Zażółć 😀", "", "Null", "\xff", "a'\\\n", "$(SQLCMDSERVER)", "x$$(y\nz"}
	want := map[string][]string{
		"postgresql": {`'plain'`, `'O''Brien'`, `E'a\\b'`, `E'a\nb'`, `E'\r'`, `E'x\x00y'`, `E'\x1a'`,
			`'Zażółć 😀'`, `''`, `'Null'`, `'` + replacement + `'`, `E'a''\\\n'`, `'$(SQLCMDSERVER)'`,
			`E'x$$(y\nz'`},
		"mysql": {`'plain'`, `'O''Brien'`, `'a\\b'`, `'a\nb'`, `'\r'`, `'x\0y'`, `'\Z'`,
			`'Zażółć 😀'`, `''`, `'Null'`, `'` + replacement + `'`, `'a''\\\n'`, `'$(SQLCMDSERVER)'`,
			`'x$$(y\nz'`},
		"sqlserver": {`N'plain'`, `N'O''Brien'`, `N'a\b'`,
			`CAST(N'a' AS nvarchar(max)) + NCHAR(10) + N'b'`, `CAST(N'' AS nvarchar(max)) + NCHAR(13)`,
			`CAST(N'x' AS nvarchar(max)) + NCHAR(0) + N'y'`, `CAST(N'' AS nvarchar(max)) + NCHAR(26)`,
			`N'Zażółć 😀'`, `N''`, `N'Null'`, `N'` + replacement + `'`,
			`CAST(N'a''\' AS nvarchar(max)) + NCHAR(10)`,
			`CAST(N'' AS nvarchar(max)) + NCHAR(36) + N'(SQLCMDSERVER)'`,
			`CAST(N'x$' AS nvarchar(max)) + NCHAR(36) + N'(y' + NCHAR(10) + N'z'`},
		"sqlite": {`'plain'`, `'O''Brien'`, `'a\b'`, `'a' || char(10) || 'b'`, `'' || char(13)`,
			`'x' || char(0) || 'y'`, `'' || char(26)`, `'Zażółć 😀'`, `''`, `'Null'`,
			`'` + replacement + `'`, `'a''\' || char(10)`, `'$(SQLCMDSERVER)'`, `'x$$(y' || char(10) || 'z'`},
	}
	quoted := map[string]string{"postgresql": `"t" ("v")`, "mysql": "`t` (`v`)",
		"sqlserver": `[t] ([v])`, "sqlite": `"t" ("v")`}
	for _, dialect := range dataset.Dialects() {
		t.Run(dialect, func(t *testing.T) {
			written := writtenBy(t, sqlRecipe(dialect, false, entityT), oneColumn(values...)).
				written["dataset.sql"].String()
			for index, literal := range want[dialect] {
				statement := "INSERT INTO " + quoted[dialect] + " VALUES (" + literal + ");\n" // punctuation: test data
				if !strings.Contains(written, statement) {
					t.Errorf("%q is not written as %s in:\n%s", values[index], statement, written)
				}
			}
			for _, raw := range []string{"\x00", "\r", "\x1a"} {
				if strings.Contains(written, raw) {
					t.Errorf("the file carries the byte %q as it is", raw)
				}
			}
			// sqlcmd replaces a variable inside a quoted value as well, so a
			// file for SQL Server never holds the opening of one (DEC-341).
			if dialect == dataset.DialectSQLServer && strings.Contains(written, "$(") {
				t.Errorf("the file for SQL Server holds a variable for sqlcmd to replace:\n%s", written)
			}
			if lines := strings.Count(written, "\n"); lines != strings.Count(written, ";\n")+ // punctuation: test data
				strings.Count(written, "GO\n") {
				t.Errorf("a value broke a statement over two lines:\n%s", written)
			}
		})
	}
}

// A cell given no value is NULL, and a value that is the empty text is the
// empty text.
//
// The two are one in a comma separated file and two in a database, where a
// column may refuse the first and take the second (DEC-341).
func TestACellWithoutAValueIsNullAndTheEmptyTextIsNot(t *testing.T) {
	both := table{
		planned: dataset.Table{Entity: "t", Columns: []string{"a", "b"}, Rows: 1},
		records: []dataset.Record{{Values: []string{"", ""}, Absent: []bool{true, false}, Parent: dataset.NoParent}},
	}
	entity := recipe.Entity{Name: "t", Count: 1, HasCount: true, Fields: []recipe.Field{{Name: "a"}, {Name: "b"}}}
	for dialect, want := range map[string]string{
		"postgresql": "VALUES (NULL, '');", "mysql": "VALUES (NULL, '');", // punctuation: test data
		"sqlserver": "VALUES (NULL, N'');", "sqlite": "VALUES (NULL, '');", // punctuation: test data
	} {
		written := writtenBy(t, sqlRecipe(dialect, false, entity), both).written["dataset.sql"].String()
		if !strings.Contains(written, want) {
			t.Errorf("%s: an absent cell and an empty one are not written as %q:\n%s", dialect, want, written)
		}
	}
}

// Every file starts with what its database needs, a table that follows another
// starts after an empty line, and on SQL Server every statement is its own
// batch.
func TestEveryFileStartsWithWhatItsDatabaseNeeds(t *testing.T) {
	second := recipe.Entity{Name: "u", Count: 1, HasCount: true, Fields: []recipe.Field{{Name: "v"}}}
	other := oneColumn("x")
	other.planned.Entity = "u"
	for dialect, want := range map[string]string{
		"postgresql": "SET client_encoding = 'UTF8';\n" + // punctuation: test data
			"INSERT INTO \"t\" (\"v\") VALUES ('a');\n\n" + // punctuation: test data
			"INSERT INTO \"u\" (\"v\") VALUES ('x');\n", // punctuation: test data
		"mysql": "SET NAMES utf8mb4;\n" + // punctuation: test data
			"SET SESSION sql_mode = REPLACE(@@SESSION.sql_mode, 'NO_BACKSLASH_ESCAPES', '');\n" + // punctuation: test data
			"INSERT INTO `t` (`v`) VALUES ('a');\n\n" + // punctuation: test data
			"INSERT INTO `u` (`v`) VALUES ('x');\n", // punctuation: test data
		"sqlserver": "SET NOCOUNT ON;\nGO\nSET QUOTED_IDENTIFIER ON;\nGO\n" + // punctuation: test data
			"INSERT INTO [t] ([v]) VALUES (N'a');\nGO\n\n" + // punctuation: test data
			"INSERT INTO [u] ([v]) VALUES (N'x');\nGO\n", // punctuation: test data
		"sqlite": "INSERT INTO \"t\" (\"v\") VALUES ('a');\n\n" + // punctuation: test data
			"INSERT INTO \"u\" (\"v\") VALUES ('x');\n", // punctuation: test data
	} {
		written := writtenBy(t, sqlRecipe(dialect, false, entityT, second), oneColumn("a"), other)
		if got := written.written["dataset.sql"].String(); got != want {
			t.Errorf("%s wrote:\n%s\nwant:\n%s", dialect, got, want)
		}
	}

	perTable := writtenBy(t, sqlRecipe("sqlserver", true, entityT, second), oneColumn("a"), other)
	for _, name := range []string{"t.sql", "u.sql"} {
		file, made := perTable.written[name]
		if !made {
			t.Fatalf("a file per table did not give %s, only %v", name, names(perTable))
		}
		if !strings.HasPrefix(file.String(), "SET NOCOUNT ON;\nGO\nSET QUOTED_IDENTIFIER ON;\nGO\nINSERT INTO ") { // punctuation: test data
			t.Errorf("%s does not start with what SQL Server needs:\n%s", name, file)
		}
	}
	if _, made := perTable.written["dataset.sql"]; made {
		t.Error("a file per table also wrote the one file")
	}
}

// A table without rows still leaves the one file behind, holding what every
// file starts with, so that the run never names a file it did not write.
func TestARunWithoutRowsStillWritesItsFile(t *testing.T) {
	written := writtenBy(t, sqlRecipe("postgresql", false, entityT), oneColumn())
	if got := written.written["dataset.sql"].String(); got != "SET client_encoding = 'UTF8';\n" { // punctuation: test data
		t.Errorf("a table of no rows wrote %q", got)
	}
}

// A child names its parent by the key of the parent row, and a row that
// belongs to none of the rows the parent holds names nothing.
func TestAChildNamesItsParentByKeyOrByNothing(t *testing.T) {
	rec := sample(t, false, 9)
	rec.Output = recipe.Output{Format: format.NameSQL, Dialect: "sqlite"}
	written := run(t, rec, asSQL).written["dataset.sql"].String()

	keys := map[string]bool{}
	children := 0
	for _, line := range strings.Split(written, "\n") {
		switch {
		case strings.HasPrefix(line, `INSERT INTO "customer"`):
			fields := strings.Split(strings.TrimSuffix(line[strings.Index(line, "VALUES (")+8:], ");"), ", ") // punctuation: test data
			keys[fields[2]] = true
		case strings.HasPrefix(line, `INSERT INTO "order"`):
			children++
			parent := line[strings.LastIndex(line, ", ")+2 : len(line)-2]
			if !keys[parent] {
				t.Errorf("an order names %s, which is no customer's key", parent)
			}
		}
	}
	if children != 9 || len(keys) != 4 {
		t.Errorf("the file holds %d orders of 9 and %d customer keys of 4", children, len(keys))
	}

	orphan := table{
		planned: dataset.Table{Entity: "order", Columns: []string{"placed_at"}, Rows: 1, Parent: "customer"},
		records: []dataset.Record{{Values: []string{"2026-01-01"}, Parent: dataset.NoParent}},
	}
	lone := writtenBy(t, sqlRecipe("postgresql", false, rec.Entities...), orphan).written["dataset.sql"].String()
	if !strings.Contains(lone, `("placed_at", "customer") VALUES ('2026-01-01', NULL);`) { // punctuation: test data
		t.Errorf("a row of no parent does not name NULL:\n%s", lone)
	}
}

func asSQL(files *held, rec recipe.Recipe) (generate.Writer, error) {
	return format.NewSQL(files, rec)
}

// A long value with a character to escape is not cut to 4000 on SQL Server,
// which cuts two literals joined together to that many (DEC-341).
func TestALongValueWithAnEscapeIsMadeLongBeforeAnythingIsJoined(t *testing.T) {
	long := strings.Repeat("a", 5000) + "\n" + "b"
	written := writtenBy(t, sqlRecipe("sqlserver", false, entityT), oneColumn(long)).written["dataset.sql"].String()
	if !strings.Contains(written, "VALUES (CAST(N'"+strings.Repeat("a", 5000)+"' AS nvarchar(max)) + NCHAR(10) + N'b');") { // punctuation: test data
		t.Errorf("the long value is not made long before it is joined:\n%.200s", written)
	}
}

// The same seed gives the same bytes, in every database.
func TestTheSameSeedGivesTheSameStatements(t *testing.T) {
	for _, dialect := range dataset.Dialects() {
		rec := sample(t, false, 7)
		rec.Output = recipe.Output{Format: format.NameSQL, Dialect: dialect}
		first := run(t, rec, asSQL).written["dataset.sql"].String()
		second := run(t, rec, asSQL).written["dataset.sql"].String()
		if first != second || first == "" {
			t.Errorf("%s: two runs of one seed wrote different statements", dialect)
		}
	}
}

// A recipe the statements cannot express is refused, by the writer and by the
// question validate asks, with the place in the recipe, and a recipe they can
// express is not.
func TestARecipeTheStatementsCannotExpressIsRefused(t *testing.T) {
	field := func(name string) recipe.Field { return recipe.Field{Name: name, Type: "integer"} }
	parent := func(key ...string) recipe.Entity {
		return recipe.Entity{Name: "customer", Count: 2, HasCount: true, PrimaryKey: key,
			Fields: []recipe.Field{field("id"), field("country")}}
	}
	child := recipe.Entity{Name: "order", Count: 2, HasCount: true,
		Parents: []recipe.Parent{{Entity: "customer", Min: 1, Max: 1}}, Fields: []recipe.Field{field("total")}}
	named := func(name string) recipe.Entity {
		return recipe.Entity{Name: "t", Count: 1, HasCount: true, Fields: []recipe.Field{field(name)}}
	}
	long63 := strings.Repeat("a", 63)
	for _, one := range []struct {
		name, dialect string
		entities      []recipe.Entity
		said          string
	}{
		{"a parent without a key", "postgresql", []recipe.Entity{parent(), child}, "declares no primary key"},
		{"a parent with a key of two fields", "mysql", []recipe.Entity{parent("id", "country"), child}, "has 2 fields"},
		{"a name with a line break", "sqlite", []recipe.Entity{named("a\nb")}, "control character"},
		{"a name of 65 bytes in MySQL", "mysql", []recipe.Entity{named(strings.Repeat("a", 65))}, "64 bytes"},
		{"a name of 33 Polish letters in MySQL", "mysql", []recipe.Entity{named(strings.Repeat("ż", 33))}, "64 bytes"},
		{"a name ending in a space in MySQL", "mysql", []recipe.Entity{named("name ")}, "ends in a space"},
		{"a name with an emoji in MySQL", "mysql", []recipe.Entity{named("name😀")}, "emoji"},
		{"a name of 129 characters in SQL Server", "sqlserver", []recipe.Entity{named(strings.Repeat("a", 129))}, "128 characters"},
		{"a name of 65 emoji in SQL Server", "sqlserver", []recipe.Entity{named(strings.Repeat("😀", 65))}, "has 130"},
		{"a name sqlcmd would change", "sqlserver", []recipe.Entity{named("cost$(x)")}, "variable of its own"},
		{"two names one in PostgreSQL", "postgresql", []recipe.Entity{{Name: "t", Count: 1, HasCount: true,
			Fields: []recipe.Field{field(long63 + "x"), field(long63 + "y")}}}, "same 63 bytes"},
		{"a database this build does not know", "oracle", []recipe.Entity{named("v")}, "asks for \"oracle\""},
	} {
		t.Run(one.name, func(t *testing.T) {
			rec := sqlRecipe(one.dialect, false, one.entities...)
			err := format.Fits(rec)
			if err == nil || fault.KindOf(err) != fault.Config || !strings.Contains(err.Error(), one.said) {
				t.Fatalf("Fits answered %v, want a refusal saying %q", err, one.said)
			}
			if _, built := format.NewSQL(newHeld(), rec); built == nil {
				t.Error("validate refuses the recipe and the writer takes it")
			}
		})
	}

	fitting := []recipe.Entity{parent("id"), child, named(strings.Repeat("ż", 32)), {Name: "u", Count: 1,
		HasCount: true, Fields: []recipe.Field{field(long63 + "x"), field(strings.Repeat("😀", 64))}}}
	for _, dialect := range []string{"postgresql", "sqlserver", "sqlite"} {
		if err := format.Fits(sqlRecipe(dialect, false, fitting...)); err != nil {
			t.Errorf("%s refused a recipe it can express: %v", dialect, err)
		}
	}
	if err := format.Fits(sqlRecipe("mysql", false, parent("id"), child, named(strings.Repeat("ż", 32)))); err != nil {
		t.Errorf("mysql refused a recipe it can express: %v", err)
	}
}
