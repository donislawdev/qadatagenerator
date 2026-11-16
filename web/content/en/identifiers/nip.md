---
title: NIP
seoTitle: "Polish NIP numbers for testing, check digit explained"
description: "How the last digit of a Polish tax identification number (NIP) is calculated, and how to generate valid NIP numbers and broken ones for your tests."
lead: "NIP is the Polish tax identification number. It has ten digits, and the last is a check digit."
sources:
  - "Ministry of Finance of Poland, POLAND - Information on Tax Identification Numbers, on the European Commission's TIN portal, read on 2 August 2026."
---

## What a NIP is

Ten digits. The first three are the code of the tax office that gave out the number. Beyond that, the number says nothing about its owner.

## The check digit

Multiply the first nine digits by 6, 5, 7, 2, 3, 4, 5, 6, 7, add up the products and divide by 11. The remainder is the check digit. A remainder of 10 does not fit in one digit, so those nine digits never start a valid NIP.

## Valid NIP numbers for your tests

{{< run "qadg quick nip --count 5 --seed example" >}}

The program never picks nine digits that would leave a remainder of 10. No block of NIP numbers is set aside for testing, so a generated one can match a real one: keep it in test systems.

## What your application should do with a broken one

Refuse a NIP whose check digit does not match, and one whose first nine digits leave a remainder of 10. A changed digit always shows. Some swaps of two neighbouring digits do not, so the program does not count such a swap as damage and says so in the manifest.
