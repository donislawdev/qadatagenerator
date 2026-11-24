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

- `format` is `csv`, the default, or `json`.
- `dir` is the folder the files go to, relative to where you run `qadg`, and that folder itself when left out. `qadg generate --out` takes its place, and the Recipe screen writes to the folder chosen at its top.
- `delimiter` is the character between values in a `csv` file, a comma when left out. Any other single character works in quotes, such as a semicolon for a spreadsheet set to use one, or `"\t"` for a tab. A double quote and a line break are refused, because they already mean something in the file. `json` has no delimiter, and `validate` warns about one.
- `file_per_entity` matters to `json` only. `true` writes a file per table, such as `customer.json` and `payment.json`, each a list of rows with a child row naming its parent in a field. `false`, the default, writes one `dataset.json` with each row's children inside it. `csv` always writes a file per table, so `validate` warns that `false` changes nothing there.
- `encoding` can only be `utf-8`, which is also what a run writes without it.

## Related tables

`belongs_to` names the parent table and how many rows each parent gets. `{ customer: 0-5 }` gives every customer between zero and five payments, and each payment carries the key of its customer in a column named after the parent. `primary_key` names the column, or columns, that identify a row, and that is the value a child row carries.

## Rules between columns

`consistency` holds rules such as `first_name matches: [gender]`, which keeps first names in line with the gender column. A city, its region and its postal code agree without a rule, and `validate` says so.

## Damage from a recipe

`policy` says what to damage when the recipe runs: which kinds of damage in `layers`, the share of `rows`, how many values in one row with `per_row`, and columns to leave alone with `exclude_fields`. [What it breaks](/docs/damage/) lists every kind.
