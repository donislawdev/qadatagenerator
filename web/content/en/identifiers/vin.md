---
title: VIN
seoTitle: "VIN check digit explained, and valid VINs for testing"
description: "How the ninth character of a VIN is calculated, why I, O and Q never appear, and how to generate thousands of valid VINs, and broken ones, for your tests."
lead: "A vehicle identification number has 17 characters, and the ninth is a check digit worked out from the other sixteen."
sources:
  - "49 CFR 565, Vehicle Identification Number Requirements, eCFR as of 3 September 2026: § 565.13 for the characters, § 565.15 for the check digit and the model year, tables III to VII."
---

## What a VIN is

Digits and capital letters without I, O and Q, so that nothing reads like 1 or 0. The ninth character is the check digit and the tenth is the model year. The rules on this page are the US ones, from the federal regulation 49 CFR 565, which also works an example through.

## The check digit

1. Give every character a value. A digit is worth itself. A letter is worth the number the regulation gives it: A and J are 1, B, K and S are 2, and so on up to 9.
2. Multiply each value by the weight of its place: 8, 7, 6, 5, 4, 3, 2, 10, 0, 9, 8, 7, 6, 5, 4, 3, 2. The ninth place weighs 0, so the check digit does not count towards itself.
3. Add up the products and divide by 11. The remainder is the check digit, and a remainder of 10 is written X.

The regulation's own example is `1G4AH59H45G118341`. The products add up to 411, and 411 divided by 11 leaves 4, which is the ninth character.

## Valid VINs for your tests

{{< run "qadg quick vin --count 5 --seed example" >}}

Every VIN ends in five digits, as a passenger car's must, and its tenth character is a model year code from the regulation. `--count 1000` gives a thousand. A generated VIN can match a real vehicle, so keep it in test systems.

## What your application should do with a broken one

Refuse a VIN whose ninth character does not match. Two things to know first. A vehicle built outside the reach of the US rules may carry a VIN with no check digit, and when the program reads a VIN it applies the US rule, so it counts such a VIN as broken. And a changed letter goes unnoticed when the new one has the same value, such as J in place of A, so the check digit cannot catch every typing mistake.
