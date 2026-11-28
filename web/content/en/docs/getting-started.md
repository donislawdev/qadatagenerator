---
title: Getting started
description: "Get QA Data Generator, print your first valid value, then generate a whole dataset from a recipe and break it on purpose, in a few minutes."
lead: "From nothing to a dataset, a damaged copy and the answer for every damaged value. Every command and every output on this page comes from a real run."
---

## Get it

[Get it](/download/) has the archive for each system, the installer for Windows, and how to build it from source. The examples below type `qadg`, so put the folder you unpacked it to on your `PATH`, or install it on Windows.

## One value

{{< readme marker="run" kind="run" output="output" >}}

Without `--seed` every run draws new values. With it, the same values come back every time. `qadg help generators` lists everything this build can make, and [Generators](/docs/generators/) has the same list.

## A whole dataset

Save this as `shop.yaml`:

{{< readme marker="file shop.yaml" caption="shop.yaml" >}}

Check it, then run it:

{{< readme marker="run" nth="2" kind="run" output="output 1" >}}

`data/customer.csv` starts like this:

{{< readme marker="head data/customer.csv" caption="data/customer.csv" >}}

and `data/payment.csv` like this, each payment carrying the key of its customer:

{{< readme marker="head data/payment.csv" caption="data/payment.csv" >}}

- `belongs_to: { customer: 0-5 }` gives each customer zero to five payments.
- `matches` keeps first names in line with the gender column. City, state and ZIP code agree without a rule, and `validate` says so.
- `required` and `max_length` describe your columns. They also tell the damage below what to break.
- `seed` makes the run repeatable. Change it for different data of the same shape.

[Recipes](/docs/recipes/) lists every key.

## Break it on purpose

Add three lines to the recipe:

{{< readme marker="append shop.yaml" caption="added to shop.yaml" >}}

and run it again, asking for the manifest as a Markdown table too:

{{< readme marker="run" nth="3" kind="run" >}}

The data comes out as before, with damage in 5% of the rows, and beside it is `expectation-manifest.json`. Each entry says where the damaged value is, what it was before and after, and what your application should do with it. [The manifest](/docs/manifest/) shows a few entries and explains each field.

## Next

- [Damage a file you already have](/docs/damage-a-file/), when you have a CSV file to test with.
- [Check what your app did](/docs/verify/), to compare your application's answers with the manifest.
- [Using it in CI](/docs/ci/), to fail a build on a wrong answer.
