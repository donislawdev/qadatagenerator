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

## Related tables

`belongs_to` names the parent table and how many rows each parent gets. `{ customer: 0-5 }` gives every customer between zero and five payments, and each payment carries the key of its customer in a column named after the parent. `primary_key` names the column, or columns, that identify a row, and that is the value a child row carries.

## Rules between columns

`consistency` holds rules such as `first_name matches: [gender]`, which keeps first names in line with the gender column. A city, its region and its postal code agree without a rule, and `validate` says so.

## Damage from a recipe

`policy` says what to damage when the recipe runs: which kinds of damage in `layers`, the share of `rows`, how many values in one row with `per_row`, and columns to leave alone with `exclude_fields`. [What it breaks](/docs/damage/) lists every kind.
