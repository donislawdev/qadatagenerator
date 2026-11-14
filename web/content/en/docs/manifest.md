---
title: The manifest
seoTitle: "The manifest: an answer for every damaged value"
description: "The file written beside damaged data: where each defect is, the value before and after, what your app should do, and its Gherkin, CSV and Markdown forms."
lead: "Every run that damages data writes `expectation-manifest.json` beside it. Your test reads the right answer from that file instead of you writing it by hand."
---

## What an entry says

Every damaged value gets an entry: where it is, by table, row and column, which damage it was, the value before and after, what your application should do, and how sure that answer is. Damage that did not fit your data is listed too, with the reason, so a quiet run never looks like a clean one.

A few entries from the run in [Getting started](/docs/getting-started/):

{{< defects >}}

`~` marks an answer that depends on the rules of your own application. `qadg verify` reports a disagreement on one of those and does not fail on it.

## Other forms of the same entries

The canonical form is `expectation-manifest.json`. `--manifest gherkin`, `csv` or `markdown` writes the same entries in a second form beside it:

- **Gherkin.** An `Examples` table with one row per applied defect, for a scenario to run.
- **CSV.** Every ordered defect, with a column saying whether it was applied and why not.
- **Markdown.** The applied defects as a table, for a review or a pull request.

## Same seed, same manifest

The same recipe and seed give the same rows and the same damage, byte for byte, on Windows, macOS and Linux, including which rows were damaged and how. Two runs differ only in the time the manifest says it was written.

## The format

The manifest is written down as an open specification, so other tools can read and write it: [expectation manifest](https://github.com/donislawdev/qadatagenerator/blob/main/spec/expectation-manifest.md).
