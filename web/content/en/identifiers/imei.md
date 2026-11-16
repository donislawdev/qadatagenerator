---
title: IMEI
seoTitle: "IMEI check digit explained, and IMEIs for testing"
description: "How the 15th digit of an IMEI is calculated with the Luhn formula, and how to generate valid IMEIs from the block for test equipment, and broken ones."
lead: "An IMEI identifies a mobile phone or a modem. It has 15 digits, and the last is a Luhn check digit."
sources:
  - "ETSI TS 123 003 version 19.7.0, the published text of 3GPP TS 23.003: clause 6.2.1 and annex B."
  - "GSMA TS.06, IMEI Allocation and Approval Process, version 28.1: table 1 and clause 9.1."
---

## What an IMEI is

Eight digits of Type Allocation Code, which names the model, six digits of serial number and one check digit. The GSM Association gives out the type allocation codes.

## The check digit

Start with the digit just left of the check digit and double every second digit going left. Add up the digits of the doubled numbers and the digits that were not doubled. Subtract the total from the next number ending in 0. The difference is the check digit, and a total that already ends in 0 gives 0.

The standard works one through, `26053179311383`. The doubled digits give 12, 10, 2, 18, 2, 6 and 6, everything adds up to 53, and 60 - 53 is 7, so the IMEI is `260531793113837`.

## Valid IMEIs for your tests

{{< run "qadg quick imei --count 5 --seed example" >}}

Every IMEI starts with 00, the block the GSM Association gives to test equipment, which may not be sold. So no phone on sale carries one. Codes inside the block are still given to manufacturers, so a value can match a test device somewhere.

## What your application should do with a broken one

Refuse an IMEI whose check digit does not match. A changed digit always shows. Of all the swaps of two neighbouring digits, only 0 next to 9 leaves the check digit right, and the program does not count that swap as damage.
