---
title: PESEL
seoTitle: "PESEL for testing, matching birth date and gender"
description: "Generate PESEL numbers whose birth date and gender agree with the columns beside them, with a correct check digit, and broken ones your app should refuse."
lead: "The Polish national identification number carries a person's birth date and gender. A PESEL the program makes always agrees with both."
sources:
  - "gov.pl, Czym jest numer PESEL, read on 2 August 2026."
  - "Ministry of Finance of Poland, POLAND - Information on Tax Identification Numbers, on the European Commission's TIN portal, which works the check digit through."
---

## What a PESEL is

Eleven digits: the birth date as year, month and day, four serial digits, and a check digit. The century is hidden in the month: 1 to 12 for the 1900s, 21 to 32 for the 2000s, 41 to 52 for the 2100s, 61 to 72 for the 2200s and 81 to 92 for the 1800s. The last serial digit gives the gender: even for a woman, odd for a man.

## The check digit

Multiply the first ten digits by 1, 3, 7, 9, 1, 3, 7, 9, 1, 3 and add up the products. Take the last digit of the total and subtract it from 10. The result is the check digit, and when the total ends in 0 the check digit is 0.

## Valid PESEL numbers for your tests

{{< run "qadg quick pesel --count 5 --gender female --birth-date 1990-04-12 --seed example" >}}

`--gender` and `--birth-date` fix what the number carries, and without them each number gets its own. In a recipe, the rule `pesel matches: [birth_date, gender]` makes every PESEL agree with the two columns beside it. The scheme covers the years 1800 to 2299 only. No block of PESEL numbers is set aside for testing, so a generated one can belong to a real person: keep it in test systems.

## What your application should do with a broken one

Refuse a PESEL whose check digit does not match. A few swaps of two neighbouring digits leave the check digit right, so the program does not count such a swap as damage and says so in the manifest.

Then the harder case: a PESEL that is correct on its own and carries a different birth date or gender than the form beside it. Each value passes every check on its own and only the pair is wrong, so only a form that compares the PESEL with the birth date and the gender will catch it. The manifest marks this answer as depending on your application, because not every form asks for a birth date and a gender.
