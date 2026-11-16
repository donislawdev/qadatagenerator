---
title: REGON
seoTitle: "Polish REGON numbers for testing, check digit explained"
description: "How the check digit of a 9-digit Polish REGON is calculated, why its digits carry no meaning, and how to generate valid and broken REGON numbers for tests."
lead: "REGON is the Polish statistical number of a business or an institution. The program makes the nine-digit form, whose last digit is a check digit."
sources:
  - "Regulation of the Council of Ministers of 30 November 2015, Dz.U. 2015 poz. 2009, § 12."
---

## What a REGON is

Nine digits for a business: an eight-digit serial number and a check digit. A local unit of a business gets fourteen digits, the first nine of which are the business's own. The regulation that sets the number says its digits must carry no meaning about the business, so they encode neither a region nor a kind of business. The number of a business that is struck off is never given to another.

The program makes only the nine-digit form. The check digit rule for fourteen digits could not be checked against real numbers, so the program leaves it out rather than guess it.

## The check digit

Multiply the first eight digits by 8, 9, 2, 3, 4, 5, 6, 7, add up the products and divide by 11. The remainder is the check digit, and a remainder of 10 is written 0.

None of the four official documents read when the program was written publishes this rule. These weights reproduce every one of 174 real numbers taken from two public registers, 15 of them with a remainder of 10.

Because 0 and 10 are written the same way, a changed digit goes unnoticed about once in 55 tries.

## Valid REGON numbers for your tests

{{< run "qadg quick regon --count 5 --seed example" >}}

No block of REGON numbers is set aside for testing, so a generated one can match a real business's: keep it in test systems.

## What your application should do with a broken one

Refuse a REGON whose check digit does not match. When a swap of two neighbouring digits leaves the check digit right, the program does not count it as damage and says so in the manifest.
