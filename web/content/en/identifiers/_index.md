---
title: Identifiers
seoTitle: "Valid identifiers for testing, check digits explained"
description: "Card numbers, ISBN, EAN, IMEI, VIN, SSN, routing numbers and Polish national numbers: how each check digit works, and how to get valid and broken ones."
lead: "Eleven kinds of number a form or an import has to check. Each page says how the number is built, how its check digit works, and how to get valid and broken ones for your tests."
---

Every number the program makes has a correct check digit, and some kinds are made in a block set aside so that no real number can match. The table says which.

| Number | Check digit | Can a generated one match a real one? |
|---|---|---|
| [Test card numbers](/identifiers/test-card-numbers/) | Luhn | They are test numbers payment providers publish |
| [VIN](/identifiers/vin/) | ninth character, remainder after dividing by 11 | Yes |
| [SSN](/identifiers/ssn/) | none | No, area 666 is never issued |
| [PESEL](/identifiers/pesel/) | last digit, weights 1, 3, 7, 9 | Yes |
| [ISBN](/identifiers/isbn/) | last digit, weights 1 and 3 | Not today, no registration group holds the block |
| [EAN-13](/identifiers/ean/) | last digit, weights 3 and 1 | No, prefix 952 is set aside for examples |
| [IMEI](/identifiers/imei/) | Luhn | Only a test device, never a phone on sale |
| [Routing number](/identifiers/routing-number/) | weights 3, 7, 1 | Yes |
| [NIP](/identifiers/nip/) | last digit, remainder after dividing by 11 | Yes |
| [REGON](/identifiers/regon/) | last digit, remainder after dividing by 11 | Yes |
| [Polish IBAN](/identifiers/polish-iban/) | two digits, remainder after dividing by 97 | Yes |

Where a generated number can match a real one, keep the data in test systems. [Every value is made up](/security/) says more.

Every page shows numbers this version of the program printed, and a recipe that breaks some of them, with what your application should do with each broken one. All of that comes from running the program, so a page cannot show what the program no longer does.
