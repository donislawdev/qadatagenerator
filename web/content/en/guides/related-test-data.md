---
title: Related tables
seoTitle: "Test data for related tables, with keys that match"
description: "Generate customers and their orders in one run: every child row points at a parent that exists, and each parent gets the number of children you ask for."
lead: "Customers, their orders and the lines of each order, from one recipe and one run. Every order carries the key of a customer that exists, and every line the key of an order."
weight: 4
notYet: [L5]
---

## Why tables filled one by one fall short

Fill three tables separately and the keys do not meet: an order names a customer that is not there, and the import or the foreign key refuses it before your test even starts. A recipe describes the tables and the relations between them together, and one run fills them so the keys match.

## Describe the relations in the recipe

{{< example "recipe" >}}

{{< example "downloads" >}}

- `primary_key` names the column that identifies a row. It is what a child row carries.
- `belongs_to: { customer: 0-3 }` gives each customer zero to three orders, so some have none, and `belongs_to: { order: 1-4 }` gives each order one to four lines.
- A child table needs no `count`. The relation decides how many rows it has, and another seed spreads them differently.
- Each child gets a column named after its parent, holding the parent's key: `customer` in the orders, `order` in the lines.

## What it writes

{{< example "run" >}}

{{< example "writes" >}}

Each value in the `customer` column of the orders is a `customer_id` of a customer, and each value in the `order` column of the lines is an `order_id` of an order.

## A count that fits the relation

A child table can name a `count` of its own. The relation then decides only how its rows are spread over the parents, and a count the relation cannot reach is refused before anything is written. The run ends with exit code 5, and the message has the arithmetic: how many rows the parent has, the range, and what the count has to be between.

## In JSON

With `format: json` every order stands inside its customer and every line inside its order, in one file, and each child still names its parent in a field of its own. [A shop in three tables](/examples/orders/) is a similar shop in JSON.

## What this version does not do yet

Every key in the data matches. This version does not yet damage the relations: a child that points at a parent that is not there, or a key that repeats. [What it breaks](/docs/damage/) lists every kind it carries out.
