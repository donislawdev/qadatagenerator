---
title: ISBN
seoTitle: "ISBN-13 check digit, and valid ISBNs for testing"
description: "How the last digit of an ISBN-13 is calculated, why the program makes no ISBN-10, and how to generate valid ISBNs no book carries, and broken ones."
lead: "An ISBN has 13 digits and starts with 978 or 979. Its last digit is a check digit, worked out the same way as a barcode's."
sources:
  - "International ISBN Agency, ISBN Users' Manual, seventh edition, 2017, appendix A1.1."
  - "International ISBN Agency, ISBN ranges file, read on 7 September 2026."
  - "International ISMN Agency, for numbers starting with 979-0."
---

## What an ISBN is

Since 1 January 2007 every ISBN has 13 digits. An ISBN-13 is also an EAN-13 barcode number, which is why it starts with 978 or 979. A number that starts with 979-0 is not an ISBN at all but an ISMN, a number for printed music, and the program says so rather than call it a broken ISBN.

The program makes no ISBN-10. The ISBN Users' Manual does not give its check digit rule, and no ten-digit ISBN has been issued since 2007.

## The check digit

Multiply the first twelve digits by 1 and 3 in turn, starting with 1 on the left. Add up the products, take the remainder after dividing by 10, and subtract it from 10. The result is the check digit, and a result of 10 is written 0.

The manual works its own ISBN through, `978-92-95055-12-4`. The products are 9, 21, 8, 27, 2, 27, 5, 0, 5, 15, 1 and 6, they add up to 126, the remainder is 6, and 10 - 6 is 4.

## Valid ISBNs for your tests

{{< run "qadg quick isbn --count 5 --seed example" >}}

Every ISBN starts with 978-67, 978-68 or 978-69, a block the ISBN agency had given to no registration group when it was last checked, on 7 September 2026. No book carries such a number today. The agency can start giving the block out later, so it is not promised to stay empty.

## What your application should do with a broken one

Refuse an ISBN whose last digit does not match. A changed digit always shows. Swapping two neighbouring digits that differ by 5, such as 2 and 7, does not change the check digit, so the program does not count such a swap as damage and says so in the manifest.
