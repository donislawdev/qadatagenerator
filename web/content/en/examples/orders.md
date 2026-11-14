---
title: A shop in three tables
seoTitle: "Related test data: customers, orders, lines in JSON"
description: "A recipe for a shop in three related tables, customers, orders and order lines, in one JSON file, with keys that join, lists of values and empty fields."
lead: "Fifty customers, their orders and the lines of each order. The relations decide how many rows there are, and every child carries the key of its parent."
weight: 5
---

`belongs_to: { customer: 0-5 }` gives every customer up to five orders, some none at all, and `belongs_to: { order: 1-4 }` gives every order one to four lines. In JSON every order stands inside its customer and every line inside its order, and each still names its parent in a field of its own.

Statuses, plans and quantities come from lists in the recipe, so they are values your application knows. Totals and prices are decimal numbers, and about seven coupons in ten are empty, to show how your code treats a missing value. Change `format: json` to `csv` for a file per table.
