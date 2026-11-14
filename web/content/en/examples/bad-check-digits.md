---
title: Numbers that should be refused
seoTitle: "Invalid check digits: test numbers to reject"
description: "A recipe that mixes broken card, IBAN, ISBN, EAN, IMEI, VIN and routing numbers in with valid ones, and a manifest that names every broken value."
lead: "The table of [test numbers](/examples/test-numbers/), with a broken value ordered in about a fifth of the rows. Your application should reject each one, and the manifest says which they are."
weight: 6
---

The damage is the kind a person makes typing a number, or a system makes copying one: a check digit moved by one, two digits swapped, a routing number from a block that is never issued. Each keeps the length and the characters of a valid number, so only a real check catches it.

Import the file, then give [`qadg verify`](/docs/verify/) the manifest and what your application did with each row. It lists every row where your application did something other than what the manifest says.
