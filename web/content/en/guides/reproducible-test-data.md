---
title: Reproducible test data
seoTitle: "Reproducible test data: same seed, same bytes"
description: "Commit a small recipe instead of large fixture files. The same recipe and seed give the same rows and the same damage on every machine."
lead: "A recipe and a seed are enough to make the same data again, byte for byte, on Windows, macOS and Linux. Commit the recipe and generate the data where you need it."
weight: 5
---

## What stays the same

The same recipe, run by the same version of `qadg` with the same seed, writes the same files byte for byte on Windows, macOS and Linux. That holds for the damage too: the same rows are damaged in the same way, and the manifest says the same. Two runs differ only in the time the manifest says it was written.

The seed is any text up to 128 characters. Write it in the recipe as `seed:`, or give `--seed` on the command line, which wins over the recipe. A recipe without a seed is read with the seed `0`, so it repeats as well.

{{< run "qadg quick vin --count 3 --seed reproducible" >}}

The same command gives these three VINs wherever you run it. Another seed gives three others.

## Commit the recipe, not the data

A fixture file of ten thousand rows is hard to review, and nobody updates it. A recipe is twenty lines, says what the data is, and makes it again in the pipeline. When a test fails on a value, the seed and the row number in the report lead you back to it.

Every run writes `run-note.json` beside the data. It names the version of `qadg`, the seed and the locale the run used, which is what it takes to make the same data again.

## Change one thing at a time

Each value is drawn from its own place: the table, the row, the column and the seed. So adding a column leaves the values of the other columns as they were, and a diff of the new data shows the new column. A larger `count` adds rows at the end and leaves the first ones, with their damage, as they were.

## What makes the data different

- Another seed.
- Another version of `qadg`. A new version can draw some values differently, so pin the version your tests run with, and look at `run-note.json` when data you expected to stay has changed.
- A change to the recipe that touches the values: another locale, another generator, a rule between columns, another `policy`.

## The one command whose runs differ

`qadg quick` draws a new seed every time you run it without `--seed`, because asking twice for a PESEL usually means wanting two different ones. It names the seed it drew on the log channel, so a value you liked can be had again with `--seed`.

[Using it in CI](/docs/ci/) has the exit codes and channels a pipeline needs.
