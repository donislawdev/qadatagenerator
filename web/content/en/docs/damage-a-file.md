---
title: Damage a file you already have
seoTitle: "Damage a CSV file you already have"
description: "Point qadg corrupt at a CSV file. It works out what each column holds, says why, damages a copy and writes what your application should do with each value."
lead: "You do not need a recipe to test with bad data. `qadg corrupt` takes a CSV file you already have, damages a copy of it and writes the manifest beside it."
---

## Run it

{{< readme marker="run" nth="4" kind="run" output="output" outputNth="2" >}}

Your file stays as it was. The damaged copy is `customers.corrupted.csv`, written beside the manifest.

## What it works out about your columns

Nothing in a plain CSV file says what a column holds, so `qadg corrupt` looks at the values, and at the header. A column whose every value passes the card validator, and no other scheme with a check digit, is taken for card numbers. For every column it says what it decided and why, as in the lines above, and it lists what it could not work out rather than guessing.

Damage that needs a kind of column your file does not have is listed in the manifest as not applied, with the reason.

## Choose what to damage

`--layers` is required, so you always choose what to attack. [What it breaks](/docs/damage/) lists every group and every kind of damage in it. `--rows` sets the share of rows to damage, 5% unless you say otherwise.

`--dry-run` shows what it worked out and what it would damage, and writes nothing.

## Every flag

{{< command-flags name="corrupt" >}}

## In the window

The Damage a file screen does the same: open a CSV file, see what each column holds and why, tick what to damage, see what would happen, then run it. It shows the command that does the same, ready to copy.
