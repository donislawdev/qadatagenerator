---
title: SSN
seoTitle: "SSNs for testing that belong to nobody"
description: "Social security numbers for test data: why numbers starting with 666 are never issued, which rules an SSN follows, and how to generate thousands at once."
lead: "Every SSN the program makes starts with 666, an area the Social Security Administration never issues, so none of them is anybody's number."
sources:
  - "Social Security Administration, Social Security Number Randomization and its FAQ, ssa.gov/employer, read on 2 August 2026."
  - "Social Security Administration, Program Operations Manual System, RM 10201.020."
---

## What an SSN is

Nine digits written 123-45-6789: a three-digit area, a two-digit group and a four-digit serial. There is no check digit, so no arithmetic can catch a typo. What can be checked is whether the number falls in a part of the range that is never issued.

## Numbers that are never issued

- area 000, area 666, and areas 900 to 999,
- group 00,
- serial 0000.

Areas 900 to 999 are not free. They have the shape of Individual Taxpayer Identification Numbers, which belong to real people. That leaves 666 as the one area that looks ordinary and belongs to nobody.

For a single number shown in an advertisement or a demonstration, the Social Security Administration's own advice is 000-00-0000.

## Valid SSNs for your tests

{{< run "qadg quick ssn --count 5 --seed example" >}}

`--count 1000` gives a thousand. An application that checks whether an area is ever issued will refuse every one of them, valid rows included. That is the price of numbers that cannot be anybody's.

## What your application should do with a broken one

The damage moves a number into area 000, group 00 or serial 0000, and any validator that knows the SSN rules refuses those. The manifest marks this answer as depending on your application, because the check needs knowledge of the rules rather than arithmetic, and many systems do not carry it.
