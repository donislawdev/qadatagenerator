---
title: Test a CSV import
seoTitle: "How to test a CSV import with broken rows"
description: "Damage a copy of a real CSV file on purpose, import it, and compare what your application did with the expected result for every damaged value."
lead: "Take a CSV file your import already accepts, damage a copy of it on purpose, import the copy, and let `qadg verify` tell you which rows your application got wrong."
weight: 2
notYet: [L0, L1]
sources:
  - "Keeping leading zeros and large numbers, Microsoft Support, read on 16 November 2026."
---

## Start from a file your import accepts

Use a file of the kind your users send, one your import takes without complaint today. Every change in the damaged copy is then one that `qadg corrupt` made, and the manifest knows the right answer for it. A file that already has problems mixes your faults with the ones you ordered.

`qadg corrupt` reads the file and works out what each column holds from its values and its header, such as card numbers or surnames. `--dry-run` prints what it worked out and what it would damage, and writes nothing.

## Damage a copy

{{< readme marker="run" nth="4" kind="run" output="output" outputNth="2" >}}

Your file stays as it was. The damaged copy is `customers.corrupted.csv`, and `expectation-manifest.json` beside it has an entry for every value that was changed.

`--layers` says what to attack and is required. For an import, start with Field format and Value meaning (`L2,L3`), values that should be refused, and Valid but unusual values (`REV`), names that should get in. Add Unicode characters (`UNI`) to see what your import does with emoji and invisible characters. `--rows` sets the share of rows to damage, 5% unless you say otherwise.

This version changes values inside the rows. It does not yet break the file itself: a row with a column missing, a stray quote or a file cut off halfway. [What it breaks](/docs/damage/) lists every kind it carries out.

## Import it the way your users do

Send the damaged copy through the same path a real file takes: the upload form, the API or the batch job. Do not open it in a spreadsheet and save it on the way. Excel drops the leading zeros of a value such as `02116` and keeps only 15 significant digits of a longer number, so the last digit of a 16-digit card number becomes a zero, and the file no longer matches the manifest.

## Write down what happened to every row

For each row, write down whether your application accepted or rejected it, in a small JSON file. Rows count from 1, the first record after the header.

{{< readme marker="response response.json customers.corrupted.csv" caption="response.json" >}}

- Write an entry for every row, the clean ones too. A row with no entry is reported as one your application never reacted to, which is how a dropped row or an import that stopped halfway shows up.
- Add `code` with your application's own error code, and `verify` repeats it in its report.
- When you can read back what was stored, add it as `stored`. An application can take a row and keep something else, such as a name cut to the width of its column, and only `stored` lets `verify` tell that from a value that got in as it was sent.

## Compare

{{< readme marker="run exit=4" kind="run" output="output" outputNth="3" exit="4" >}}

Each warning is a row your application treated differently from what the manifest says: a value that should have been refused and got in, a valid one that was turned away, or a row nobody answered for. It ends with exit code 4, so a pipeline can fail on it. Answers marked `~` depend on your own rules, and a disagreement on one is reported without failing.

`verify` answers for the values that were damaged. A clean row your application refused is not in the manifest, so check those in your own report.

[Using it in CI](/docs/ci/) puts these steps in a pipeline.
