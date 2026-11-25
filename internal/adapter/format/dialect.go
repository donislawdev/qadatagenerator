package format

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
)

// dialect is what the SQL form needs to know about one database: how a file
// starts, how a name and a value are written, and what follows a statement.
//
// # Where every rule here comes from
//
// Each one was measured on 2026-11-25 against the database itself, in
// throwaway containers: PostgreSQL 18.6, MySQL 26.7.0, MariaDB 12.3.2, SQL
// Server 2025 (17.0.4065.4) through sqlcmd 18.6 and SQLite 3.53.4. The numbers
// and the probes are in DEC-341. A rule that disagrees with a newer release of
// one of them is a rule to measure again, not to argue about.
type dialect struct {
	name string
	// opening are the statements at the top of every file, so that the bytes
	// mean the same whatever the session they are loaded into was set to.
	opening []string
	// quote writes a name of a table or a column.
	quote func(name string) string
	// literal writes a value that is present, the empty text included.
	literal func(value string) string
	// after is what follows every statement, on lines of its own.
	after string
	// nameFits says what is wrong with a name in this database, or nothing.
	nameFits func(name string) string
	// clipped is how many bytes of a name the database keeps, or zero when it
	// keeps every one. Two names equal in their first clipped bytes are one
	// name to it.
	clipped int
	// table is what a table the file creates needs in this database.
	table tableRules
}

// statementEnd ends a statement in every one of these databases. It is the one
// place the character stands in this package, because it is a byte of the
// generated file rather than a mark of punctuation in a sentence.
const statementEnd = ";" // punctuation: test data

// dialectOf returns the dialect of a database by the name a recipe gives it.
func dialectOf(name string) (dialect, bool) {
	switch name {
	case dataset.DialectPostgreSQL:
		return dialect{
			name: name,
			// A client on Windows talks in the code page of its console unless
			// told otherwise, and the bytes of this file are UTF-8.
			opening:  []string{"SET client_encoding = 'UTF8'" + statementEnd},
			quote:    quotedWith(`"`, `"`),
			literal:  postgresLiteral,
			nameFits: func(string) string { return "" },
			// The database cuts a longer name to its first 63 bytes with a
			// notice, the same way in the CREATE TABLE a person wrote, so a
			// long name works and two that share their first 63 bytes collide.
			clipped: 63,
			table:   tableRules{typeOf: postgresType, nativeUUID: true, refuses: postgresRefuses},
		}, true
	case dataset.DialectMySQL:
		return dialect{
			name: name,
			opening: []string{
				"SET NAMES utf8mb4" + statementEnd,
				// Every value is written with backslash escapes, which mean
				// what they say only while this mode is off. The rest of the
				// session's mode is the person's own, strictness included,
				// and stays as it is.
				"SET SESSION sql_mode = REPLACE(@@SESSION.sql_mode, 'NO_BACKSLASH_ESCAPES', '')" + statementEnd,
			},
			quote:    quotedWith("`", "`"),
			literal:  mysqlLiteral,
			nameFits: mysqlNameFits,
			table: tableRules{typeOf: mysqlType, keyCollation: "utf8mb4_bin",
				// A database of another character set made the table of that
				// set, and it refused Polish letters (measured).
				options: " DEFAULT CHARSET=utf8mb4", refuses: mysqlRefuses},
		}, true
	case dataset.DialectSQLServer:
		return dialect{
			name: name,
			opening: []string{
				// The count of rows after every statement says nothing a person
				// loading thousands of rows needs.
				"SET NOCOUNT ON" + statementEnd,
				// sqlcmd starts with this off, and with it off every INSERT into
				// a table with a filtered index failed (measured, error 1934).
				// Values here are N'...' and names [...], so turning it on
				// changes the meaning of nothing this file says.
				"SET QUOTED_IDENTIFIER ON" + statementEnd,
			},
			quote:   quotedWith("[", "]"),
			literal: sqlServerLiteral,
			// A value the server cannot convert ends the batch it is in, so
			// every statement is its own batch and a refused row stops only
			// itself.
			after:    "GO\n",
			nameFits: sqlServerNameFits,
			table: tableRules{typeOf: sqlServerType, nativeUUID: true,
				keyCollation: "Latin1_General_100_BIN2", filteredUnique: true, refuses: sqlServerRefuses},
		}, true
	case dataset.DialectSQLite:
		return dialect{
			name:     name,
			quote:    quotedWith(`"`, `"`),
			literal:  sqliteLiteral,
			nameFits: func(string) string { return "" },
			table: tableRules{typeOf: sqliteType,
				refuses: func(sqlTableShape) (string, string) { return "", "" }},
		}, true
	}
	return dialect{}, false
}

// quotedWith writes a name between two marks, doubling the closing mark inside
// it, which is how every one of these databases lets a name hold any character
// it allows at all. A reserved word such as order needs it, and so does a name
// with a space.
func quotedWith(open, closing string) func(string) string {
	return func(name string) string {
		return open + strings.ReplaceAll(name, closing, closing+closing) + closing
	}
}

// Characters a value never carries into a file as they are.
//
// A line break would let a value change when somebody's tools turn the line
// endings of the file into their own, a zero byte made psql lose the
// statements after it (measured), and the substitute character ends a text
// file for some readers on Windows. Each dialect writes them its own way.
const escapedCharacters = "\x00\n\r\x1a"

// escapeAt is where the first character to escape stands in a value, or -1.
func escapeAt(value string) int { return strings.IndexAny(value, escapedCharacters) }

// sqlcmdVariable opens a variable of sqlcmd, which the tool replaces wherever it
// stands, inside a quoted value as well. A value of $(SQLCMDSERVER) loaded with
// the command the recipes page gives went into the table as the name of the
// server (measured, DEC-341), and any variable of the environment would have
// gone in the same way.
const sqlcmdVariable = "$("

// sqlServerEscapeAt is escapeAt for SQL Server, where the dollar sign of a
// variable is escaped as well, so that the file never holds one for sqlcmd to
// replace.
func sqlServerEscapeAt(value string) int {
	at := escapeAt(value)
	if variable := strings.Index(value, sqlcmdVariable); variable >= 0 && (at < 0 || variable < at) {
		return variable
	}
	return at
}

// valid replaces each byte that is not UTF-8 with the replacement character,
// the rule the object form follows as well. Nothing a run produces today has
// such a byte, because damage of that kind needs a writer that offers bytes.
func valid(value string) string {
	if utf8.ValidString(value) {
		return value
	}
	return strings.ToValidUTF8(value, string(utf8.RuneError))
}

// postgresLiteral writes a value as PostgreSQL reads it.
//
// A value without a backslash or one of the escaped characters is written
// plain, which reads the same whether standard_conforming_strings is on or
// off. Any other goes in the escape form, which reads the same in both modes
// as well, while a plain one with a backslash loses it when the setting is off
// (measured). A zero byte is written as an escape the database then refuses,
// because no text column can hold one, and only that row is lost.
func postgresLiteral(value string) string {
	value = valid(value)
	if !strings.ContainsAny(value, `\`+escapedCharacters) {
		return "'" + strings.ReplaceAll(value, "'", "''") + "'"
	}
	var built strings.Builder
	built.WriteString("E'")
	for index := 0; index < len(value); index++ {
		switch symbol := value[index]; symbol {
		case '\'':
			built.WriteString("''")
		case '\\':
			built.WriteString(`\\`)
		case '\n':
			built.WriteString(`\n`)
		case '\r':
			built.WriteString(`\r`)
		case 0x00:
			built.WriteString(`\x00`)
		case 0x1a:
			built.WriteString(`\x1a`)
		default:
			built.WriteByte(symbol)
		}
	}
	built.WriteByte('\'')
	return built.String()
}

// mysqlLiteral writes a value the way the C client library of MySQL escapes
// one, which gave every byte back in MySQL and MariaDB alike (measured).
//
// The quote is doubled rather than escaped, so that a server that has the
// backslash mode switched on anyway gets a value with a doubled backslash and
// never a statement that ends early: a value written with only the quote
// doubled dropped a table there (DEC-341).
func mysqlLiteral(value string) string {
	value = valid(value)
	var built strings.Builder
	built.Grow(len(value) + 2)
	built.WriteByte('\'')
	for index := 0; index < len(value); index++ {
		switch symbol := value[index]; symbol {
		case '\'':
			built.WriteString("''")
		case '\\':
			built.WriteString(`\\`)
		case '\n':
			built.WriteString(`\n`)
		case '\r':
			built.WriteString(`\r`)
		case 0x00:
			built.WriteString(`\0`)
		case 0x1a:
			built.WriteString(`\Z`)
		default:
			built.WriteByte(symbol)
		}
	}
	built.WriteByte('\'')
	return built.String()
}

// sqlServerLiteral writes a value as a Unicode literal.
//
// Without the N the server reads the text in the code page of the database, and
// a Polish name lost its letters (measured). It has no escapes at all, so an
// escaped character is joined on as NCHAR. The first part is made long before
// anything is joined to it, because two literals joined together are cut to
// 4000 characters without a word (measured).
func sqlServerLiteral(value string) string {
	value = valid(value)
	quote := func(part string) string { return "N'" + strings.ReplaceAll(part, "'", "''") + "'" }
	at := sqlServerEscapeAt(value)
	if at < 0 {
		return quote(value)
	}
	return joined(value, at, "CAST("+quote(value[:at])+" AS nvarchar(max))", quote,
		func(symbol byte) string { return "NCHAR(" + strconv.Itoa(int(symbol)) + ")" }, " + ",
		sqlServerEscapeAt)
}

// sqliteLiteral writes a value as SQLite reads it. Its literals have no escapes
// either, so an escaped character is joined on with char.
func sqliteLiteral(value string) string {
	value = valid(value)
	quote := func(part string) string { return "'" + strings.ReplaceAll(part, "'", "''") + "'" }
	at := escapeAt(value)
	if at < 0 {
		return quote(value)
	}
	return joined(value, at, quote(value[:at]), quote,
		func(symbol byte) string { return "char(" + strconv.Itoa(int(symbol)) + ")" }, " || ", escapeAt)
}

// joined writes a value as the given start, which holds its text up to the
// first character to escape at, then each such character as a call and each
// later part as a literal, joined by the operator of the dialect. next finds
// the character to escape in what is left.
func joined(value string, at int, start string, quote func(string) string, call func(byte) string,
	operator string, next func(string) int) string {
	var built strings.Builder
	built.WriteString(start)
	rest := value[at:]
	for rest != "" {
		built.WriteString(operator + call(rest[0]))
		rest = rest[1:]
		plain := next(rest)
		if plain < 0 {
			plain = len(rest)
		}
		if plain > 0 {
			built.WriteString(operator + quote(rest[:plain]))
		}
		rest = rest[plain:]
	}
	return built.String()
}

// mysqlNameFits says what MySQL or MariaDB refuses in a name.
//
// MySQL 26.7 refused a name over 64 bytes and MariaDB 12.3 one over 64
// characters, so the bytes are the limit, the stricter of the two. Both
// refused a name ending in a space, and MariaDB one with a character outside
// the basic plane, an emoji. A table name has a further limit from the name
// of its file on the disk, which this build does not work out (a table of 32
// Polish letters failed in MySQL where a column of them did not).
func mysqlNameFits(name string) string {
	switch {
	case len(name) > 64:
		return "MySQL takes names of up to 64 bytes and this one has " + strconv.Itoa(len(name))
	case strings.HasSuffix(name, " "):
		return "MySQL and MariaDB refuse a name that ends in a space"
	case beyondBasicPlane(name):
		return "MariaDB refuses a name with a character such as an emoji, which takes four bytes"
	}
	return ""
}

// sqlServerNameFits says what SQL Server refuses in a name: more than 128
// characters, counted the way the server counts them, where an emoji is two.
// It also refuses the opening of a variable of sqlcmd, which the tool replaces
// inside a name as it does inside a value, and a name has no way to escape it.
func sqlServerNameFits(name string) string {
	if strings.Contains(name, sqlcmdVariable) {
		return "sqlcmd reads " + sqlcmdVariable + " as the start of a variable of its own and would " +
			"change the name before SQL Server saw it"
	}
	units := 0
	for _, symbol := range name {
		units++
		if symbol > 0xFFFF {
			units++
		}
	}
	if units > 128 {
		return "SQL Server takes names of up to 128 characters and this one has " + strconv.Itoa(units)
	}
	return ""
}

func beyondBasicPlane(name string) bool {
	for _, symbol := range name {
		if symbol > 0xFFFF {
			return true
		}
	}
	return false
}

// clip returns the first so many bytes of a name, cut at the start of a
// character, which is where PostgreSQL cuts one.
func clip(name string, bytes int) string {
	if bytes <= 0 || len(name) <= bytes {
		return name
	}
	cut := bytes
	for cut > 0 && !utf8.RuneStart(name[cut]) {
		cut--
	}
	return name[:cut]
}
