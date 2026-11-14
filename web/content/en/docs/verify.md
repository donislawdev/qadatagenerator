---
title: Check what your app did
seoTitle: "Compare what your app did with the manifest"
description: "Record what your application did with each row, and qadg verify lists every row where it disagrees with the manifest, with an exit code to fail a build on."
lead: "The manifest says what your application should do with every damaged value. `qadg verify` compares that with what it really did."
---

## Write down what your application did

Import the damaged file into your application and write down what it did with each row, one entry per row. A few lines at the end of your test can write it. Three entries of such a record:

{{< readme marker="response response.json customers.corrupted.csv" caption="response.json" >}}

`verdict` is `accepted` or `rejected`. `code` is optional: your application's own code for what happened, which `verify` repeats in its report. When you can read back what was stored, add `stored`, and `verify` can tell a value that survived from one your application quietly changed. Every field you add widens what the comparison can tell you, and the report says which questions it could not answer.

## Compare

{{< readme marker="run exit=4" kind="run" output="output" outputNth="3" exit="4" >}}

It ends with exit code 4. Row 135 is a card number with two digits swapped that got in, and row 432 is O'Brien, who was turned away. Your pipeline can fail on that code. [Using it in CI](/docs/ci/) has every exit code.

An answer marked `~` in the manifest depends on the rules of your own application, so `verify` reports a disagreement on it and does not fail on it.

## In the window

The Check results screen opens a manifest and a record of what your application did, goes through every ordered defect, and shows where the two disagree.

## The format

The record of what your application did is written down as an open specification: [application response record](https://github.com/donislawdev/qadatagenerator/blob/main/spec/application-response.md).
