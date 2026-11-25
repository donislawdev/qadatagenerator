---
title: Recipes
seoTitle: "Test data recipes in YAML, every key explained"
description: "Describe tables, row counts, relations and column rules in one YAML file. Every key a recipe takes, what it does, and how validate reports a problem."
lead: "A recipe is a YAML file you keep next to your tests. It says which tables to make, how many rows each one has, how they relate and what fills each column."
---

## A recipe from start to end

{{< readme marker="file shop.yaml" caption="shop.yaml" >}}

This one makes a thousand customers and zero to five payments for each. The customers are in `en-US`, so a region is a state and a postal code is a ZIP code. `seed` makes every run give the same rows.

## Every key

{{< readme-table heading="Recipes" >}}

## Check a recipe before you run it

`qadg validate` reads a recipe and reports every problem in it at once, rather than stopping at the first. A recipe that passes says what a run would write:

{{< readme marker="run" nth="2" kind="run" output="output 1" >}}

The Recipe screen of the window checks the recipe the same way as you type.

## Numbers

`type: integer` fills a column with random whole numbers from 0 to 999999, and `type: decimal` with random numbers with two decimal places from 0.00 to 999.99. `min_value` and `max_value` set the smallest and the largest number, and both are included:

```yaml
fields:
  quantity: { type: integer, min_value: 1, max_value: 10 }
  price:    { type: decimal, min_value: 0.5, max_value: 99.99 }
  change:   { type: integer, min_value: -50, max_value: 50 }
```

Leave one of the two out and that end stays where it was, so `max_value: 10` on its own gives numbers from 0 to 10. A bound of a decimal has at most two decimal places, and a bound of an integer has none. With `max_length` beside them, both ends have to fit in it. `validate` says which bound is wrong and why.

## Where the files go

`output` says which files a run writes and where. Every key in it can be left out:

```yaml
output:
  format: csv
  dir: ./data
  delimiter: "\t"
```

- `format` is `csv`, the default, `json`, or `sql`, which writes an INSERT statement for each row.
- `dialect` names the database a `sql` file is for: `postgresql`, `mysql`, `sqlserver` or `sqlite`. `mysql` also covers MariaDB. `sql` needs it, and `validate` says so when it is missing. `qadg generate --dialect` takes its place.
- `dir` is the folder the files go to, relative to where you run `qadg`, and that folder itself when left out. `qadg generate --out` takes its place, and the Recipe screen writes to the folder chosen at its top.
- `delimiter` is the character between values in a `csv` file, a comma when left out. Any other single character works in quotes, such as a semicolon for a spreadsheet set to use one, or `"\t"` for a tab. A double quote and a line break are refused, because they already mean something in the file. `json` and `sql` have no delimiter, and `validate` warns about one.
- `file_per_entity` matters to `json` and `sql`. For `json`, `true` writes a file per table, such as `customer.json` and `payment.json`, each a list of rows with a child row naming its parent in a field, and `false`, the default, writes one `dataset.json` with each row's children inside it. For `sql`, `true` writes `customer.sql` and `payment.sql`, and `false` writes one `dataset.sql` with the parent tables first. `csv` always writes a file per table, so `validate` warns that `false` changes nothing there.
- `encoding` can only be `utf-8`, which is also what a run writes without it.

## SQL files

`format: sql` writes every row as an INSERT statement for the database named under `dialect`:

```yaml
output:
  format: sql
  dialect: postgresql
```

```sql
SET client_encoding = 'UTF8';
INSERT INTO "customer" ("customer_id", "last_name", "coupon") VALUES ('746861', 'Michael', NULL);
INSERT INTO "customer" ("customer_id", "last_name", "coupon") VALUES ('516920', 'Lara', 'WELCOME');

INSERT INTO "order" ("total", "customer") VALUES ('230.73', '746861');
INSERT INTO "order" ("total", "customer") VALUES ('85.20', '516920');
```

The file creates no tables. Load it into a database whose tables already have these names and columns.

- Every value is written as text in quotes, and the database converts it to the type of its column, as it does when it imports a CSV file. A value the column cannot take, such as text in a number column, is refused by the database, which is often what a test of broken data wants to see.
- A cell left empty by `empty` is `NULL`. A value that is an empty text, which damage can write, stays `''`.
- A child row names its parent by the parent's primary key. With `sql`, every table that has child rows needs a `primary_key` of one field, and `validate` refuses the recipe otherwise, because a database has no row numbers for a child to point at.
- Names of tables and columns are always quoted, so `order` or `first name` work as they are. A name the database refuses, such as one longer than 64 bytes in MySQL or 128 characters in SQL Server, is refused before anything is written. MySQL also limits the name of a table by the name of its file on disk, which `validate` does not check.
- Each file starts with the settings its database needs to read it as written: the client encoding for PostgreSQL, and for MySQL `SET NAMES utf8mb4` and a line that keeps backslash escapes on for this session, with the rest of its `sql_mode` left as your server has it. A SQL Server file starts with `SET NOCOUNT ON` and `SET QUOTED_IDENTIFIER ON`, without which sqlcmd cannot insert a row into a table that has a filtered index, and every statement in it is followed by `GO`.

Each row is its own statement, so a row the database refuses stops only that row:

| Database | Load the file | When a row is refused |
|---|---|---|
| PostgreSQL | `psql -d testdb -f dataset.sql` | psql reports it and goes on |
| MySQL, MariaDB | `mysql --force testdb < dataset.sql` | without `--force`, the client stops at the first one |
| SQL Server | `sqlcmd -d testdb -f 65001 -i dataset.sql` | sqlcmd reports it and goes on |
| SQLite | `sqlite3 test.db < dataset.sql` | sqlite3 reports it and goes on |

`-f 65001` makes sqlcmd read the file as UTF-8. Without it, sqlcmd reads the file in the current code page, which on Windows is usually not UTF-8, so letters outside English can change.

A statement for each row loads slowly when the file is large: one to several milliseconds a row in our measurements. Inside one transaction the same file loads many times faster, but then one refused row means that nothing is loaded:

```bash
psql -d testdb --single-transaction -f dataset.sql
sqlite3 test.db "BEGIN" ".read dataset.sql" "COMMIT"
```

psql ends with exit code 0 even when it rolled the transaction back, so check its output for `ERROR`.

## Related tables

`belongs_to` names the parent table and how many rows each parent gets. `{ customer: 0-5 }` gives every customer between zero and five payments, and each payment carries the key of its customer in a column named after the parent. `primary_key` names the column, or columns, that identify a row, and that is the value a child row carries.

## Rules between columns

`consistency` holds rules such as `first_name matches: [gender]`, which keeps first names in line with the gender column. A city, its region and its postal code agree without a rule, and `validate` says so.

## Damage from a recipe

`policy` says what to damage when the recipe runs: which kinds of damage in `layers`, the share of `rows`, how many values in one row with `per_row`, and columns to leave alone with `exclude_fields`. [What it breaks](/docs/damage/) lists every kind.
