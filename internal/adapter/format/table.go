package format

import (
	"strconv"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// The tables the SQL form creates when a recipe asks for them with
// output.create_tables (DEC-341).
//
// # What a table holds
//
// Every row the recipe calls correct, which is the measure each rule here was
// chosen by. The types come from the recipe: a length from max_length, a
// number as wide as the widest the field draws, a date, a UUID. The keys are
// the primary key, every unique field and the column naming the parent, a
// required field is NOT NULL, and a range or a list of values is a CHECK. A
// damaged row is taken or refused by the database according to those, which is
// what a test of broken data wants to see.
//
// Two exceptions are the databases' own and the recipes page names them: SQL
// Server counts a character outside the basic plane, such as an emoji, as two
// in nvarchar(n), and SQLite keeps a decimal as a floating point number.
//
// # What a file never does
//
// It never drops a table and never writes IF NOT EXISTS. Dropping would delete
// somebody's table on the day the file is loaded into the wrong database, and
// IF NOT EXISTS would hide a table of another shape behind a file that seems to
// have worked. A table that is already there makes CREATE TABLE fail, and the
// rows go into it as they would without the setting.

// sqlColumn is one column of a table the SQL form creates.
type sqlColumn struct {
	name  string
	shape generate.Shape
	// length is the max_length the recipe gives, and limited says it gives
	// one. The unit does not change the type: a value of n bytes has at most n
	// characters and at most n UTF-16 units, so it fits a column of n.
	length  int
	limited bool
	notNull bool
	// key says the column is in the primary key, unique, or names the parent.
	// A database indexes it, so it needs a type it can index, and it compares
	// its text byte for byte, because the generator keeps unique values apart
	// byte for byte and a collation ignoring case would take two of them for
	// one (measured, DEC-341).
	key bool
	// unique says the recipe asks for unique values in this column alone.
	unique bool
	// values, smallest and largest are what a CHECK holds the column to.
	values            []string
	smallest, largest string
}

// sqlTableShape is a table the SQL form creates, in the order its statement
// names it.
type sqlTableShape struct {
	entity  string
	columns []sqlColumn
	primary []string
	// parent is the column naming the parent row, and parentEntity and
	// parentKey are what it points at. All three are empty for a table of no
	// parent.
	parent, parentEntity, parentKey string
}

// shapeOfTable works out the table one entity becomes, its columns in the
// order given and the column naming the parent after them, the order the
// statements of rows use.
func shapeOfTable(rec recipe.Recipe, entity recipe.Entity, columns []string,
	values generate.Values) sqlTableShape {
	inKey := map[string]bool{}
	for _, name := range entity.PrimaryKey {
		inKey[name] = true
	}
	onlyKey := ""
	if len(entity.PrimaryKey) == 1 {
		onlyKey = entity.PrimaryKey[0]
	}
	table := sqlTableShape{entity: entity.Name, primary: entity.PrimaryKey}
	for _, name := range columns {
		field, known := entity.Field(name)
		if !known {
			continue
		}
		column := columnOf(field, values)
		column.notNull = field.Required || inKey[name]
		column.key = inKey[name] || field.Unique
		// A unique field that is the whole key is unique already.
		column.unique = field.Unique && name != onlyKey
		table.columns = append(table.columns, column)
	}
	if len(entity.Parents) == 0 {
		return table
	}
	table.parentEntity = entity.Parents[0].Entity
	parent, known := rec.Entity(table.parentEntity)
	if !known || len(parent.PrimaryKey) != 1 {
		return table
	}
	table.parentKey = parent.PrimaryKey[0]
	table.parent = table.parentEntity
	field, _ := parent.Field(table.parentKey)
	column := columnOf(field, values)
	// The column holds the key of a parent, so it has the type of that key
	// and none of its checks, and every child has a parent.
	column.name, column.notNull, column.key = table.parent, true, true
	column.values, column.smallest, column.largest = nil, "", ""
	table.columns = append(table.columns, column)
	return table
}

// columnOf is the column one field becomes, before the table says whether it
// is in a key.
func columnOf(field recipe.Field, values generate.Values) sqlColumn {
	column := sqlColumn{name: field.Name, shape: values.ShapeOf(generate.FieldRequest(field)),
		length: field.MaxLength, limited: field.HasMaxLength, values: field.Values}
	if column.shape.Kind == generate.ShapeInteger || column.shape.Kind == generate.ShapeDecimal {
		// A bound is written into the statement as the number it is, so only
		// a bound written in plain digits gets there. The plan refuses any
		// other before a run, so the only effect of the test is that nothing
		// else could reach the file.
		if plainNumber(field.MinValue) {
			column.smallest = field.MinValue
		}
		if plainNumber(field.MaxValue) {
			column.largest = field.MaxValue
		}
	}
	return column
}

// plainNumber reports whether a text is a number in digits, with a minus sign
// in front and a point allowed.
func plainNumber(text string) bool {
	whole, fraction, pointed := strings.Cut(strings.TrimPrefix(text, "-"), ".")
	return digitsOnly(whole) && (!pointed || digitsOnly(fraction))
}

func digitsOnly(text string) bool {
	if text == "" {
		return false
	}
	for _, digit := range text {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}

// checked reports whether the table holds the column to a range or a list.
func (c sqlColumn) checked() bool {
	return c.smallest != "" || c.largest != "" || len(c.values) > 0
}

// checks are the conditions a table holds its columns to, one for each column
// with a range or a list.
func (t sqlTableShape) checks(written dialect) []string {
	var found []string
	for _, column := range t.columns {
		if !column.checked() {
			continue
		}
		name := written.quote(column.name)
		var parts []string
		if column.smallest != "" {
			parts = append(parts, name+" >= "+column.smallest)
		}
		if column.largest != "" {
			parts = append(parts, name+" <= "+column.largest)
		}
		if len(column.values) > 0 {
			listed := make([]string, 0, len(column.values))
			for _, value := range column.values {
				listed = append(listed, written.literal(value))
			}
			parts = append(parts, name+" IN ("+strings.Join(listed, ", ")+")")
		}
		found = append(found, "CHECK ("+strings.Join(parts, " AND ")+")")
	}
	return found
}

// create writes the statements that create one table, each followed by what
// follows every statement of the dialect.
func (t sqlTableShape) create(written dialect) string {
	quote := written.quote
	lines := make([]string, 0, len(t.columns)+4)
	for _, column := range t.columns {
		line := quote(column.name) + " " + written.table.typeOf(column)
		if column.key && written.table.keyCollation != "" && column.textual(written) {
			line += " COLLATE " + written.table.keyCollation
		}
		if column.notNull {
			line += " NOT NULL"
		}
		lines = append(lines, line)
	}
	if len(t.primary) > 0 {
		lines = append(lines, "PRIMARY KEY ("+quoteAll(quote, t.primary)+")")
	}
	var filtered []string
	for position, column := range t.columns {
		if !column.unique {
			continue
		}
		if !column.notNull && written.table.filteredUnique {
			// SQL Server takes NULL for a value in a unique constraint and
			// lets one row have it (measured), and a filtered index is how it
			// keeps unique the values that are there.
			filtered = append(filtered, "CREATE UNIQUE INDEX "+quote("unique_"+strconv.Itoa(position+1))+
				" ON "+quote(t.entity)+" ("+quote(column.name)+") WHERE "+quote(column.name)+
				" IS NOT NULL"+statementEnd+"\n"+written.after)
			continue
		}
		lines = append(lines, "UNIQUE ("+quote(column.name)+")")
	}
	if t.parent != "" {
		lines = append(lines, "FOREIGN KEY ("+quote(t.parent)+") REFERENCES "+quote(t.parentEntity)+
			" ("+quote(t.parentKey)+")")
	}
	lines = append(lines, t.checks(written)...)
	return "CREATE TABLE " + quote(t.entity) + " (\n  " + strings.Join(lines, ",\n  ") + "\n)" +
		written.table.options + statementEnd + "\n" + written.after + strings.Join(filtered, "")
}

// textual reports whether the database keeps the column as text, which is
// what a collation applies to.
func (c sqlColumn) textual(written dialect) bool {
	switch c.shape.Kind {
	case generate.ShapeText:
		return true
	case generate.ShapeUUID:
		return !written.table.nativeUUID
	}
	return false
}

func quoteAll(quote func(string) string, names []string) string {
	quoted := make([]string, 0, len(names))
	for _, name := range names {
		quoted = append(quoted, quote(name))
	}
	return strings.Join(quoted, ", ")
}

// tableRules is what one database needs from a table the SQL form creates.
type tableRules struct {
	// typeOf writes the type of a column.
	typeOf func(sqlColumn) string
	// nativeUUID says the database has a type of its own for a UUID.
	nativeUUID bool
	// keyCollation compares the text of a column in a key byte for byte, and
	// is empty where the default does.
	keyCollation string
	// options follow the closing bracket of CREATE TABLE.
	options string
	// filteredUnique says a unique column that takes NULL needs a filtered
	// index rather than a constraint.
	filteredUnique bool
	// refuses says what the database would refuse in a table and what to do
	// instead, or nothing.
	refuses func(sqlTableShape) (string, string)
}

// wholeType writes the type of a whole number of so many digits: the 32 bit
// type up to nine digits, the 64 bit one up to eighteen, and a decimal of no
// places beyond, which only a list of values can ask for.
func wholeType(digits int, small, big, exact string) string {
	switch {
	case digits <= 9:
		return small
	case digits <= 18:
		return big
	}
	return exact + "(" + strconv.Itoa(digits) + ",0)"
}

// decimalType writes the type of a decimal with the places it carries.
//
// Without the places MySQL and SQL Server make a decimal of none, and a value
// with cents would be rounded to a whole number.
func decimalType(name string, shape generate.Shape) string {
	return name + "(" + strconv.Itoa(max(shape.Digits+shape.Places, 1)) + "," +
		strconv.Itoa(shape.Places) + ")"
}

// digitsOf is how many digits the type of a numeric column holds, nought for a
// column of another kind.
func digitsOf(c sqlColumn) int {
	switch c.shape.Kind {
	case generate.ShapeInteger:
		return c.shape.Digits
	case generate.ShapeDecimal:
		return c.shape.Digits + c.shape.Places
	}
	return 0
}

// textLength is the length a text column is declared with, at least one,
// since no database takes a column of none.
func textLength(c sqlColumn) string { return strconv.Itoa(max(c.length, 1)) }

// keyWidth is the length of a text column in a key whose field gives none. A
// key cannot be long text in MySQL or SQL Server, and 255 is the usual width:
// it fits one key of either and three in one key of MySQL (measured).
const keyWidth = 255

// uuidWidth is how many characters a UUID is written in.
const uuidWidth = 36

func postgresType(c sqlColumn) string {
	switch c.shape.Kind {
	case generate.ShapeInteger:
		return wholeType(c.shape.Digits, "integer", "bigint", "numeric")
	case generate.ShapeDecimal:
		return decimalType("numeric", c.shape)
	case generate.ShapeDate:
		return "date"
	case generate.ShapeUUID:
		return "uuid"
	}
	// The longest varchar PostgreSQL declares.
	if c.limited && c.length <= 10485760 {
		return "varchar(" + textLength(c) + ")"
	}
	return "text"
}

func postgresRefuses(t sqlTableShape) (string, string) {
	return numbersFit(t, 1000, "PostgreSQL")
}

func mysqlType(c sqlColumn) string {
	switch c.shape.Kind {
	case generate.ShapeInteger:
		return wholeType(c.shape.Digits, "int", "bigint", "decimal")
	case generate.ShapeDecimal:
		return decimalType("decimal", c.shape)
	case generate.ShapeDate:
		return "date"
	}
	if length, inline := mysqlLength(c); inline {
		return "varchar(" + strconv.Itoa(length) + ")"
	}
	if c.limited && c.length <= mysqlMediumText {
		return "mediumtext"
	}
	// A field without max_length has no limit, and longtext is the one type
	// of MySQL that holds any value a recipe can give.
	return "longtext"
}

// The limits of MySQL a table is held to, measured in MySQL 26.7 and MariaDB
// 12.3 (DEC-341).
const (
	// mysqlVarchar is the longest varchar of four byte characters.
	mysqlVarchar = 16383
	// mysqlMediumText is how many four byte characters a mediumtext holds.
	mysqlMediumText = 4194303
	// mysqlKey is how many bytes one index holds.
	mysqlKey = 3072
	// mysqlRow is how many bytes a row holds, long text counted by the few
	// bytes that point at it.
	mysqlRow = 65535
	// mysqlName is the longest name, which a constraint the database names
	// itself may not pass either.
	mysqlName = 64
)

// mysqlLength is the length of a varchar column in characters, and whether
// the column is one. A column of long text is not.
func mysqlLength(c sqlColumn) (int, bool) {
	switch {
	case c.shape.Kind == generate.ShapeUUID && c.limited:
		return max(c.length, 1), true
	case c.shape.Kind == generate.ShapeUUID:
		return uuidWidth, true
	case c.limited && c.length <= mysqlVarchar:
		return max(c.length, 1), true
	case !c.limited && c.key:
		return keyWidth, true
	}
	return 0, false
}

// mysqlBytes is what a column takes of a row of MySQL, the way the database
// counts it: four bytes for each character of a varchar and one or two for its
// length, a few for long text, which lives outside the row.
func mysqlBytes(c sqlColumn) int {
	switch c.shape.Kind {
	case generate.ShapeInteger, generate.ShapeDecimal, generate.ShapeDate:
		return mysqlFixedBytes(c)
	}
	length, inline := mysqlLength(c)
	if !inline {
		return 12
	}
	if 4*length > 255 {
		return 4*length + 2
	}
	return 4*length + 1
}

// mysqlKeyBytes is what a column takes of an index of MySQL.
func mysqlKeyBytes(c sqlColumn) int {
	switch c.shape.Kind {
	case generate.ShapeInteger, generate.ShapeDecimal, generate.ShapeDate:
		return mysqlFixedBytes(c)
	}
	if length, inline := mysqlLength(c); inline {
		return 4 * length
	}
	return 4 * max(c.length, 1)
}

// mysqlFixedBytes is what a number or a date takes: four bytes for each nine
// digits of a decimal and fewer for the rest, on each side of the point.
func mysqlFixedBytes(c sqlColumn) int {
	switch {
	case c.shape.Kind == generate.ShapeDate:
		return 3
	case c.shape.Kind == generate.ShapeInteger && c.shape.Digits <= 9:
		return 4
	case c.shape.Kind == generate.ShapeInteger && c.shape.Digits <= 18:
		return 8
	}
	leftover := []int{0, 1, 1, 2, 2, 3, 3, 4, 4}
	part := func(digits int) int { return digits/9*4 + leftover[digits%9] }
	return part(c.shape.Digits) + part(c.shape.Places)
}

func mysqlRefuses(t sqlTableShape) (string, string) {
	if what, instead := numbersFit(t, 65, "MySQL"); what != "" {
		return what, instead
	}
	shorter := "lower max_length of its text fields, or leave create_tables out and create the table yourself"
	row, nullable := 0, 0
	var primary int
	inPrimary := map[string]bool{}
	for _, name := range t.primary {
		inPrimary[name] = true
	}
	for _, column := range t.columns {
		row += mysqlBytes(column)
		if !column.notNull {
			nullable++
		}
		if inPrimary[column.name] {
			primary += mysqlKeyBytes(column)
		}
		if (column.unique || column.name == t.parent && t.parent != "") && mysqlKeyBytes(column) > mysqlKey {
			return "the column " + column.name + " may take " + strconv.Itoa(mysqlKeyBytes(column)) +
				" bytes of an index in MySQL, which holds at most " + strconv.Itoa(mysqlKey) +
				", 4 bytes for each character", shorter
		}
	}
	if primary > mysqlKey {
		return "the primary key may take " + strconv.Itoa(primary) + " bytes in MySQL, which indexes at most " +
			strconv.Itoa(mysqlKey) + ", 4 bytes for each character of a text column", shorter
	}
	// One byte for each eight columns that take NULL.
	if row += (nullable + 7) / 8; row > mysqlRow {
		return "a row may take " + strconv.Itoa(row) + " bytes in MySQL, which keeps at most " +
			strconv.Itoa(mysqlRow) + " in a row, 4 bytes for each character of a text column up to " +
			strconv.Itoa(mysqlVarchar) + " characters long", shorter
	}
	if longest := mysqlGeneratedName(t); len(longest) > mysqlName {
		return "MySQL names a constraint of this table " + longest + ", which is longer than " +
				strconv.Itoa(mysqlName) + " bytes, and refuses the table",
			"shorten the name of the entity, or leave create_tables out and create the table yourself"
	}
	return "", ""
}

// mysqlGeneratedName is the longest name MySQL gives a constraint of the table
// itself: the table and _ibfk_1 for the link to the parent, the table and
// _chk_ with a number for each check (measured, MariaDB names them otherwise).
func mysqlGeneratedName(t sqlTableShape) string {
	longest := ""
	if t.parent != "" {
		longest = t.entity + "_ibfk_1"
	}
	checks := 0
	for _, column := range t.columns {
		if column.checked() {
			checks++
		}
	}
	if checks > 0 {
		if named := t.entity + "_chk_" + strconv.Itoa(checks); len(named) > len(longest) {
			longest = named
		}
	}
	return longest
}

func sqlServerType(c sqlColumn) string {
	switch c.shape.Kind {
	case generate.ShapeInteger:
		return wholeType(c.shape.Digits, "int", "bigint", "decimal")
	case generate.ShapeDecimal:
		return decimalType("decimal", c.shape)
	case generate.ShapeDate:
		return "date"
	case generate.ShapeUUID:
		return "uniqueidentifier"
	}
	switch {
	case c.limited && c.length <= sqlServerNvarchar:
		return "nvarchar(" + textLength(c) + ")"
	case !c.limited && c.key:
		return "nvarchar(" + strconv.Itoa(keyWidth) + ")"
	}
	return "nvarchar(max)"
}

// sqlServerNvarchar is the longest nvarchar SQL Server declares, beyond which
// it is nvarchar(max), which no index takes (measured).
const sqlServerNvarchar = 4000

func sqlServerRefuses(t sqlTableShape) (string, string) {
	if what, instead := numbersFit(t, 38, "SQL Server"); what != "" {
		return what, instead
	}
	for _, column := range t.columns {
		if column.key && column.shape.Kind == generate.ShapeText && column.limited &&
			column.length > sqlServerNvarchar {
			return "the column " + column.name + " is in a key and may be " + strconv.Itoa(column.length) +
					" characters long, and SQL Server indexes no text longer than " +
					strconv.Itoa(sqlServerNvarchar), "lower its max_length, or leave create_tables out and " +
					"create the table yourself"
		}
	}
	return "", ""
}

func sqliteType(c sqlColumn) string {
	switch c.shape.Kind {
	case generate.ShapeInteger:
		return "integer"
	case generate.ShapeDecimal:
		return decimalType("decimal", c.shape)
	case generate.ShapeDate:
		return "date"
	case generate.ShapeUUID:
		if c.limited {
			return "varchar(" + textLength(c) + ")"
		}
		return "varchar(" + strconv.Itoa(uuidWidth) + ")"
	}
	if c.limited {
		return "varchar(" + textLength(c) + ")"
	}
	return "text"
}

// numbersFit says which column holds numbers longer than the database's
// numeric type, a whole number from a list of values or a decimal of many
// places.
func numbersFit(t sqlTableShape, widest int, database string) (string, string) {
	for _, column := range t.columns {
		if digits := digitsOf(column); digits > widest {
			return "the column " + column.name + " holds numbers of " + strconv.Itoa(digits) +
					" digits, and " + database + " holds at most " + strconv.Itoa(widest),
				"shorten the numbers, or leave create_tables out and create the table yourself"
		}
	}
	return "", ""
}

// tablesFit refuses a recipe whose tables the database it names would not
// create, and says nothing when the recipe does not ask for tables.
//
// The measure is the database's own: what it refuses to create is refused here,
// and what it creates with a warning is not, so SQL Server's warning about a key
// longer than 900 bytes is left to SQL Server, which refuses only a row whose
// key is that long (measured).
func tablesFit(rec recipe.Recipe, values generate.Values, written dialect) error {
	if !rec.Output.CreateTables || values == nil {
		return nil
	}
	for _, entity := range rec.Entities {
		table := shapeOfTable(rec, entity, entity.FieldNames(), values)
		if what, instead := written.table.refuses(table); what != "" {
			return fault.New(fault.Config, "entities."+entity.Name+": "+what).DoInstead(instead)
		}
	}
	return nil
}
