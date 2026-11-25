package format_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/format"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// tablesRecipe is a customer and an order of hers, with a field of every kind
// a table types differently, asking the SQL form of one database to create the
// tables.
func tablesRecipe(dialect string) recipe.Recipe {
	customer := recipe.Entity{Name: "customer", Count: 1, HasCount: true, PrimaryKey: []string{"id"},
		Fields: []recipe.Field{
			{Name: "id", Type: values.TypeInteger, Unique: true},
			{Name: "name", Generator: values.FirstName, Required: true, MaxLength: 40, HasMaxLength: true},
			{Name: "code", Generator: values.SSN, Unique: true},
			{Name: "plan", Values: []string{"free", "O'Brien"}},
			{Name: "loyalty", Type: values.TypeInteger, MinValue: "-5", MaxValue: "100"},
			{Name: "total", Type: values.TypeDecimal},
			{Name: "token", Type: values.TypeUUID},
			{Name: "born", Generator: values.BirthDate},
		}}
	order := recipe.Entity{Name: "order", Count: 1, HasCount: true,
		Parents: []recipe.Parent{{Entity: "customer", Min: 1, Max: 1}},
		Fields:  []recipe.Field{{Name: "note", Generator: values.City}}}
	rec := sqlRecipe(dialect, false, customer, order)
	rec.Output.CreateTables = true
	return rec
}

// tablesOf hands the writer both tables of tablesRecipe without rows.
func tablesOf(rec recipe.Recipe) []table {
	customer := table{planned: dataset.Table{Entity: "customer",
		Columns: rec.Entities[0].FieldNames()}}
	order := table{planned: dataset.Table{Entity: "order", Columns: []string{"note"}, Parent: "customer"}}
	return []table{customer, order}
}

// A table is created from what the recipe says, in the words of its database,
// before its rows and after its parent (DEC-341).
//
// Each line is a rule measured on 2026-11-25: the places a decimal carries, a
// key compared byte for byte, a key without max_length at the width of 255 in
// the two databases that index no long text, a filtered index for a unique
// column that takes NULL in SQL Server, the character set of MySQL, a CHECK
// for a range and for a list written with the literals of the rows.
func TestATableIsCreatedFromWhatTheRecipeSays(t *testing.T) {
	want := map[string][]string{
		"postgresql": {"CREATE TABLE \"customer\" (\n" +
			"  \"id\" integer NOT NULL,\n  \"name\" varchar(40) NOT NULL,\n  \"code\" text,\n" +
			"  \"plan\" text,\n  \"loyalty\" integer,\n  \"total\" numeric(5,2),\n  \"token\" uuid,\n" +
			"  \"born\" date,\n  PRIMARY KEY (\"id\"),\n  UNIQUE (\"code\"),\n" +
			"  CHECK (\"plan\" IN ('free', 'O''Brien')),\n  CHECK (\"loyalty\" >= -5 AND \"loyalty\" <= 100)\n" +
			");\n", // punctuation: test data
			"CREATE TABLE \"order\" (\n  \"note\" text,\n  \"customer\" integer NOT NULL,\n" +
				"  FOREIGN KEY (\"customer\") REFERENCES \"customer\" (\"id\")\n);\n"}, // punctuation: test data
		"mysql": {"CREATE TABLE `customer` (\n" +
			"  `id` int NOT NULL,\n  `name` varchar(40) NOT NULL,\n  `code` varchar(255) COLLATE utf8mb4_bin,\n" +
			"  `plan` longtext,\n  `loyalty` int,\n  `total` decimal(5,2),\n  `token` varchar(36),\n" +
			"  `born` date,\n  PRIMARY KEY (`id`),\n  UNIQUE (`code`),\n" +
			"  CHECK (`plan` IN ('free', 'O''Brien')),\n  CHECK (`loyalty` >= -5 AND `loyalty` <= 100)\n" +
			") DEFAULT CHARSET=utf8mb4;\n", // punctuation: test data
			"CREATE TABLE `order` (\n  `note` longtext,\n  `customer` int NOT NULL,\n" +
				"  FOREIGN KEY (`customer`) REFERENCES `customer` (`id`)\n) DEFAULT CHARSET=utf8mb4;\n"}, // punctuation: test data
		"sqlserver": {"CREATE TABLE [customer] (\n" +
			"  [id] int NOT NULL,\n  [name] nvarchar(40) NOT NULL,\n" +
			"  [code] nvarchar(255) COLLATE Latin1_General_100_BIN2,\n" +
			"  [plan] nvarchar(max),\n  [loyalty] int,\n  [total] decimal(5,2),\n  [token] uniqueidentifier,\n" +
			"  [born] date,\n  PRIMARY KEY ([id]),\n" +
			"  CHECK ([plan] IN (N'free', N'O''Brien')),\n  CHECK ([loyalty] >= -5 AND [loyalty] <= 100)\n" +
			");\nGO\n" + // punctuation: test data
			"CREATE UNIQUE INDEX [unique_3] ON [customer] ([code]) WHERE [code] IS NOT NULL;\nGO\n", // punctuation: test data
			"CREATE TABLE [order] (\n  [note] nvarchar(max),\n  [customer] int NOT NULL,\n" +
				"  FOREIGN KEY ([customer]) REFERENCES [customer] ([id])\n);\nGO\n"}, // punctuation: test data
		"sqlite": {"CREATE TABLE \"customer\" (\n" +
			"  \"id\" integer NOT NULL,\n  \"name\" varchar(40) NOT NULL,\n  \"code\" text,\n" +
			"  \"plan\" text,\n  \"loyalty\" integer,\n  \"total\" decimal(5,2),\n  \"token\" varchar(36),\n" +
			"  \"born\" date,\n  PRIMARY KEY (\"id\"),\n  UNIQUE (\"code\"),\n" +
			"  CHECK (\"plan\" IN ('free', 'O''Brien')),\n  CHECK (\"loyalty\" >= -5 AND \"loyalty\" <= 100)\n" +
			");\n", // punctuation: test data
			"CREATE TABLE \"order\" (\n  \"note\" text,\n  \"customer\" integer NOT NULL,\n" +
				"  FOREIGN KEY (\"customer\") REFERENCES \"customer\" (\"id\")\n);\n"}, // punctuation: test data
	}
	for _, dialect := range dataset.Dialects() {
		t.Run(dialect, func(t *testing.T) {
			rec := tablesRecipe(dialect)
			written := writtenBy(t, rec, tablesOf(rec)...).written["dataset.sql"].String()
			customer := strings.Index(written, want[dialect][0])
			order := strings.Index(written, want[dialect][1])
			if customer < 0 || order < 0 {
				t.Fatalf("the tables are not created as\n%s%s\nin:\n%s", want[dialect][0], want[dialect][1], written)
			}
			if customer > order {
				t.Error("the child table is created before its parent")
			}
		})
	}
}

// A table comes before its rows, a table that follows another starts after an
// empty line, and a recipe that does not ask for tables gets none.
func TestATableComesBeforeItsRowsAndOnlyWhenAskedFor(t *testing.T) {
	entity := recipe.Entity{Name: "t", Count: 1, HasCount: true, Fields: []recipe.Field{{Name: "v"}}}
	rec := sqlRecipe("sqlite", false, entity, recipe.Entity{Name: "u", Count: 1, HasCount: true,
		Fields: []recipe.Field{{Name: "v"}}})
	other := oneColumn()
	other.planned.Entity = "u"
	plain := writtenBy(t, rec, oneColumn("a"), other).written["dataset.sql"].String()
	if strings.Contains(plain, "CREATE") {
		t.Errorf("a recipe that asks for no tables got some:\n%s", plain)
	}
	rec.Output.CreateTables = true
	got := writtenBy(t, rec, oneColumn("a"), other).written["dataset.sql"].String()
	want := "CREATE TABLE \"t\" (\n  \"v\" text\n);\n" + // punctuation: test data
		"INSERT INTO \"t\" (\"v\") VALUES ('a');\n\n" + // punctuation: test data
		"CREATE TABLE \"u\" (\n  \"v\" text\n);\n" // punctuation: test data
	if got != want {
		t.Errorf("the file is\n%s\nwant\n%s", got, want)
	}
}

// A recipe whose tables the database would not create is refused before
// anything is written, by validate as by the writer, and only when it asks for
// tables. The limits are the ones measured on 2026-11-25 (DEC-341): an index
// of 3072 bytes, a row of 65 535 and names MySQL gives its own constraints in
// MySQL, text over 4000 characters in a key of SQL Server, and the digits of
// each database's numbers.
func TestATableTheDatabaseWouldNotCreateIsRefused(t *testing.T) {
	text := func(name string, length int) recipe.Field {
		return recipe.Field{Name: name, Generator: values.City, MaxLength: length, HasMaxLength: true}
	}
	keyed := func(name string, key []string, fields ...recipe.Field) recipe.Entity {
		return recipe.Entity{Name: name, Count: 1, HasCount: true, PrimaryKey: key, Fields: fields}
	}
	parent := keyed("p", []string{"id"}, recipe.Field{Name: "id", Type: values.TypeInteger})
	childOf := func(name string) recipe.Entity {
		return recipe.Entity{Name: name, Count: 1, HasCount: true,
			Parents: []recipe.Parent{{Entity: "p", Min: 1, Max: 1}}, Fields: []recipe.Field{text("v", 5)}}
	}
	bounded := func(name string) recipe.Entity {
		return keyed(name, nil, recipe.Field{Name: "n", Type: values.TypeInteger, MinValue: "0"})
	}
	digits := func(count int) recipe.Entity {
		return keyed("t", nil, recipe.Field{Name: "n", Type: values.TypeInteger,
			Values: []string{strings.Repeat("9", count)}})
	}
	unique := text("u", 769)
	unique.Unique = true
	for _, one := range []struct {
		name, dialect string
		entities      []recipe.Entity
		said          string
	}{
		{"a key of 3200 bytes in MySQL", "mysql", []recipe.Entity{keyed("t", []string{"k"}, text("k", 800))},
			"primary key may take 3200 bytes"},
		{"a unique column of 3076 bytes in MySQL", "mysql", []recipe.Entity{keyed("t", nil, unique)},
			"may take 3076 bytes of an index"},
		{"a row of three long columns in MySQL", "mysql", []recipe.Entity{keyed("t", nil,
			text("a", 6000), text("b", 6000), text("c", 6000))}, "a row may take"},
		{"a row one byte too long in MySQL", "mysql", []recipe.Entity{keyed("t", nil,
			text("a", 16383), recipe.Field{Name: "i", Type: values.TypeInteger})}, "a row may take 65539"},
		{"a link MySQL would name in 65 bytes", "mysql", []recipe.Entity{parent, childOf(strings.Repeat("c", 58))},
			"_ibfk_1"},
		{"a check MySQL would name in 65 bytes", "mysql", []recipe.Entity{bounded(strings.Repeat("c", 59))},
			"_chk_1"},
		{"a number of 66 digits in MySQL", "mysql", []recipe.Entity{digits(66)}, "at most 65"},
		{"text of 4001 characters in a key of SQL Server", "sqlserver",
			[]recipe.Entity{keyed("t", []string{"k"}, text("k", 4001))}, "no text longer than 4000"},
		{"a number of 39 digits in SQL Server", "sqlserver", []recipe.Entity{digits(39)}, "at most 38"},
		{"a number of 1001 digits in PostgreSQL", "postgresql", []recipe.Entity{digits(1001)}, "at most 1000"},
	} {
		t.Run(one.name, func(t *testing.T) {
			rec := sqlRecipe(one.dialect, false, one.entities...)
			source := values.New(rec.Locale, rec.Seed)
			if err := format.Fits(rec, source); err != nil {
				t.Fatalf("the recipe is refused without asking for tables: %v", err)
			}
			rec.Output.CreateTables = true
			err := format.Fits(rec, source)
			if err == nil || fault.KindOf(err) != fault.Config || !strings.Contains(err.Error(), one.said) {
				t.Fatalf("Fits answered %v, want a refusal saying %q", err, one.said)
			}
			if _, built := format.NewSQL(newHeld(), rec, source); built == nil {
				t.Error("validate refuses the recipe and the writer takes it")
			}
		})
	}

	// The other side of each limit, measured to fit.
	for _, one := range []struct {
		name, dialect string
		entities      []recipe.Entity
	}{
		{"a key of 3072 bytes in MySQL", "mysql", []recipe.Entity{keyed("t", []string{"k"}, text("k", 768))}},
		{"a key of three columns without a length in MySQL", "mysql", []recipe.Entity{keyed("t",
			[]string{"a", "b", "c"}, recipe.Field{Name: "a", Generator: values.City},
			recipe.Field{Name: "b", Generator: values.City}, recipe.Field{Name: "c", Generator: values.City})}},
		{"two columns of 8191 characters in MySQL", "mysql", []recipe.Entity{keyed("t", nil,
			text("a", 8191), text("b", 8191))}},
		{"a column of 16382 characters and a number in MySQL", "mysql", []recipe.Entity{keyed("t", nil,
			text("a", 16382), recipe.Field{Name: "i", Type: values.TypeInteger})}},
		{"a link MySQL names in 64 bytes", "mysql", []recipe.Entity{parent, childOf(strings.Repeat("c", 57))}},
		{"a check MySQL names in 64 bytes", "mysql", []recipe.Entity{bounded(strings.Repeat("c", 58))}},
		{"a column of 20000 characters outside a key in MySQL", "mysql", []recipe.Entity{keyed("t", nil,
			text("a", 20000))}},
		{"text of 4000 characters in a key of SQL Server", "sqlserver",
			[]recipe.Entity{keyed("t", []string{"k"}, text("k", 4000))}},
		{"a number of 38 digits in SQL Server", "sqlserver", []recipe.Entity{digits(38)}},
		{"long names and numbers in SQLite", "sqlite", []recipe.Entity{digits(2000),
			bounded(strings.Repeat("c", 200))}},
	} {
		rec := sqlRecipe(one.dialect, false, one.entities...)
		rec.Output.CreateTables = true
		if err := format.Fits(rec, values.New(rec.Locale, rec.Seed)); err != nil {
			t.Errorf("%s: refused a table the database creates: %v", one.name, err)
		}
	}
}
