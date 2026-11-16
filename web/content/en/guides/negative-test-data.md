---
title: Negative test data
seoTitle: "Negative test data, with the expected result for each"
description: "Invalid test data is half a test. Generate values your app should reject, valid ones it should accept, and the expected result for each, in one run."
lead: "A negative test needs two things: a value your application should refuse, and a way to check that it did. One run of qadg gives you both, and valid values that careless validation refuses as well."
weight: 1
---

## The answer matters as much as the value

A file of bad values is easy to make. The hard part is the test that reads the result: for every row it has to know whether your application should have refused it. Written by hand, that list of answers falls behind the data the first time the data changes.

QA Data Generator writes the answers itself. Every value it damages gets an entry in `expectation-manifest.json`: where it is, the value before and after, and what your application should do with it. Your test reads the answer from the manifest instead of keeping its own.

## Three kinds of answer

- **Reject.** The value breaks a rule every application of its kind should keep: a name longer than its column, a card number whose check digit no longer matches.
- **Accept.** The value is valid, and careless validation refuses it: O'Brien, Łukasz, 佐藤, the surname Null. A negative test that never sends these misses the people your application turns away. [Names your form should accept](/guides/names-your-form-should-accept/) has more.
- **`~`, your call.** The right answer depends on your own rules. An empty name or an SSN from an area that is never issued should usually be refused, but only you know whether your application checks for it. `qadg verify` reports a disagreement on one of these and does not fail on it.

## A recipe that asks for all three

The recipe below describes a table of customers: names that are required and have a longest length, an SSN and a card number. Its `policy` asks for damage from three groups, Field format (`L2`), Value meaning (`L3`) and Valid but unusual values (`REV`), in about two rows in five, one value in each, so every refused row has a single cause.

{{< example "recipe" >}}

{{< example "downloads" >}}

## What it changed

{{< example "run" >}}

{{< example "damage" >}}

Some of the damage it ordered is not carried out by this version yet. The manifest lists each one with the reason, so a run that changed less than the recipe asks for says so.

## Put it in your test

1. Generate the data from the recipe.
2. Send each row to your application the way a user would: through the form, the API or the import.
3. Write down in a small JSON file what your application did with each row, accepted or rejected.
4. Run `qadg verify` on the manifest and that file. It lists every row your application treated differently, and ends with exit code 4, so a pipeline can fail on it.

[Check what your app did](/docs/verify/) shows the record and the report. Change `layers` to choose what to attack, `rows` for how much, and `seed` for other values of the same kinds. [What it breaks](/docs/damage/) lists every kind of damage this version carries out.
