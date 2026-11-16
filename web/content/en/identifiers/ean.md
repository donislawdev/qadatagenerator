---
title: EAN-13
seoTitle: "EAN-13 barcodes for testing that belong to no product"
description: "How the check digit of an EAN-13 barcode number works, and how to generate valid ones under prefix 952, which GS1 sets aside for examples, and broken ones."
lead: "An EAN-13 is the 13-digit number under most shop barcodes. Every one the program makes starts with 952, a prefix GS1 keeps for examples."
sources:
  - "GS1 General Specifications, Release 26.0, January 2026: tables 1-4 and 1-5 for the prefix 952, table 7-8 for the check digit."
  - "International ISBN Agency, ISBN Users' Manual, appendix A1.1."
---

## What an EAN-13 is

Thirteen digits, the last of which is the check digit. GS1 calls it GTIN-13. The first digits are the GS1 prefix of the organisation that issued the number, and an ISBN is the same kind of number under the prefix 978 or 979.

## The check digit

Counting from the right, multiply the digit next to the check digit by 3, the one before it by 1, and so on in turn. Add up the products and subtract the total from the nearest multiple of 10 that is equal or higher. The difference is the check digit.

For 13 digits this is the same as weights 1 and 3 from the left, which is how the ISBN manual puts it. GS1 counts from the right because its longer numbers line up on the right.

## Valid EAN-13 numbers for your tests

{{< run "qadg quick ean --count 5 --seed example" >}}

GS1 keeps the prefix 952 for demonstrations and examples of the GS1 system and puts its own examples there, so no product is given a number under it. GS1 set the prefix aside in its January 2023 release, and its specification does not say whether any company was given a number there before.

## What your application should do with a broken one

Refuse a number whose check digit does not match. A changed digit always shows. Swapping two neighbouring digits that differ by 5 does not, so the program does not count such a swap as damage and says so in the manifest.
