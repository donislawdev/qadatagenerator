---
title: Routing number
seoTitle: "ABA routing numbers for testing, check digit explained"
description: "How the ninth digit of an ABA routing number is checked with the weights 3, 7 and 1, what the first two digits mean, and how to generate valid and broken ones."
lead: "An ABA routing number names a US bank on a cheque or a transfer. It has nine digits, and the ninth is a check digit."
sources:
  - "ABA Routing Number Policy & Procedures, version 9/20, from LexisNexis Risk Solutions as the official ABA registrar, section IV."
  - "12 CFR 229 (Regulation CC), appendix A."
---

## What a routing number is

Four digits of Federal Reserve routing symbol, four digits of institution identifier, and a check digit. The first two digits say what kind of number it is: 01 to 12 for banks in the twelve Federal Reserve districts, 21 to 32 for savings institutions, 61 to 72 for electronic transactions and 80 for traveller's cheques. Most of the other series are reserved for future use.

## The check digit

Multiply the nine digits by 3, 7, 1, 3, 7, 1, 3, 7, 1 and add up the products. The number is correct when the total is a multiple of 10.

The registrar's policy describes the parts of the number and leaves the arithmetic to a paid publication. Every one of the 67 routing numbers of the Federal Reserve Banks and the Federal Home Loan Banks printed in Regulation CC passes this check.

## Valid routing numbers for your tests

{{< run "qadg quick aba --count 5 --seed example" >}}

Every number starts with 01 to 12, a series of banks, because no series is kept free for testing: the registrar says its Board will decide how the series reserved for future use are used. A generated routing number can match a real bank's, so keep it in test systems.

## What your application should do with a broken one

Refuse a number whose check digit does not match. The damage can also move a number into a series reserved for future use, with a correct check digit. No bank holds such a number today, so an application that checks routing numbers against the Federal Reserve's directory should refuse it. The manifest marks this answer as depending on your application, because many applications do not hold the directory.
