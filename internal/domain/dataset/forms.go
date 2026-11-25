package dataset

// The forms a run can be written in, as a recipe names them under
// output.format.
//
// # Why the names live here rather than beside the writers
//
// The reader of recipes accepts them and the writers implement them, and the
// two are adapters that do not import each other. The list used to stand in
// both, and a copy is a copy that parts company the first time a form is
// added, which adding the SQL form would have been. Both read this one.
const (
	FormCSV  = "csv"
	FormJSON = "json"
	FormSQL  = "sql"
)

// Forms are the names a recipe may give output.format, in the order a person is
// shown them.
func Forms() []string { return []string{FormCSV, FormJSON, FormSQL} }

// The databases the SQL form writes for, as a recipe names them under
// output.dialect (DEC-341).
//
// # Why a recipe has to name one
//
// No file of statements means the same in all of them. MySQL and MariaDB read a
// backslash in a quoted value as an escape and the other three do not, so a
// value written for one of them changes in the other, and measured on
// 2026-11-25 a value written the way PostgreSQL reads it dropped a table in
// MySQL. A recipe that names no database is refused rather than given a guess.
// MySQL covers MariaDB, which reads the same file the same way (same day).
const (
	DialectPostgreSQL = "postgresql"
	DialectMySQL      = "mysql"
	DialectSQLServer  = "sqlserver"
	DialectSQLite     = "sqlite"
)

// Dialects are the names a recipe may give output.dialect, in the order a
// person is shown them.
func Dialects() []string {
	return []string{DialectPostgreSQL, DialectMySQL, DialectSQLServer, DialectSQLite}
}

// UsableDialect reports whether the SQL form writes for a database of this name.
func UsableDialect(name string) bool {
	for _, known := range Dialects() {
		if name == known {
			return true
		}
	}
	return false
}
