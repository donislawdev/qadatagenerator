---
title: Polish IBAN
seoTitle: "Polish IBANs for testing that pass the mod 97 check"
description: "How the two check digits of a Polish IBAN are calculated, what the 26 digits after PL hold, and how to generate valid account numbers and broken ones."
lead: "A Polish IBAN is PL followed by 26 digits. Without PL, the same 26 digits are the account number written on a domestic transfer in Poland, the NRB."
sources:
  - "Order No. 7/2017 of the President of Narodowy Bank Polski: § 13 and § 14 for the shape of the number, annex 2 for the sort code's check digit."
  - "Order No. 5/2002 of the President of Narodowy Bank Polski, annexes 1 and 2, for the check digits. The order has since been repealed."
---

## What a Polish IBAN is

28 characters: PL, two check digits, an eight-digit bank sort code, and a sixteen-digit account number. The last digit of the sort code is a check digit of its own.

## The check digits

1. Move PL and the two check digits to the end.
2. Replace P with 25 and L with 21.
3. Read everything as one long number and divide it by 97. The IBAN is correct when the remainder is 1.

To work out the check digits, do the same with 00 in their place and subtract the remainder from 98.

For the sort code's own check digit, multiply its first seven digits by 3, 9, 7, 1, 3, 9, 7, add up the products, and subtract the last digit of the total from 10. A result of 10 is written 0.

## Valid Polish IBANs for your tests

{{< run "qadg quick iban --count 5 --seed example" >}}

The program makes Polish IBANs only. Given an IBAN from another country, it checks the check digits and, when they are right, says it cannot judge the rest, because it carries no list of the lengths other countries use. The sort code is drawn rather than taken from a list of banks, yet it can still match a real bank's, and the account number can match a real account: keep generated numbers in test systems.

## What your application should do with a broken one

Refuse an IBAN whose check digits do not match. The damage changes one digit or swaps two neighbouring ones. When a swap leaves the check digits right, the program does not count it as damage and says so in the manifest.
