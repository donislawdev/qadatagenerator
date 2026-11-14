---
title: Using it in CI
seoTitle: "Test data in CI: seeds, exit codes, JSON"
description: "Run QA Data Generator in a pipeline: the same seed gives the same bytes, data and log go to separate channels, and every ending has its own exit code."
lead: "Everything a pipeline needs is on the command line: repeatable data, output a script can read, and an exit code for every ending."
---

## Commit a recipe, not the data

The same recipe and seed give the same rows and the same damage, byte for byte, on Windows, macOS and Linux. Commit a small recipe next to your tests instead of large fixture files, and generate the data in the pipeline.

`qadg quick` is the only command whose runs differ. Without `--seed` it draws a new seed every time and names it on the log channel.

## Channels

Data goes to standard output and the log to standard error. A run that fails writes nothing to standard output, and `--json` gives a machine readable report for `validate`, `generate`, `--version` and `--license`.

## Exit codes

{{< exit-codes >}}

`qadg --help` prints the same table.

## A step that fails on a wrong answer

1. `qadg generate` the recipe with a `policy`, or `qadg corrupt` a file of your own.
2. Import the damaged data into your application in your test, and write down what it did with each row.
3. `qadg verify` the manifest against that record. Exit code 4 fails the step.

[Check what your app did](/docs/verify/) shows the record and what `verify` prints.
