---
title: Test card numbers
seoTitle: "Test card numbers that pass validation"
description: "Card numbers for testing a payment form: eight test numbers that payment providers publish, each passing the Luhn check, and broken ones to refuse."
lead: "Eight card numbers that payment providers publish for testing, each with a correct Luhn check digit, and broken ones your payment form should refuse."
sources:
  - "Test card numbers in the documentation of Stripe, Braintree, Adyen and Checkout.com, read on 2 November 2026."
---

## Where the numbers come from

A `card` column and `qadg quick card` only ever give one of eight numbers: two Visa, one Mastercard, two American Express, one Discover, one JCB and one UnionPay. At least two of Stripe, Braintree, Adyen and Checkout.com publish each of them as a test number. The program never makes up a number that merely passes the check digit, because such a number can be a real card.

A test number works in its provider's test system. None of the four providers says that no bank has issued the same number to a real card, so treat these as test values and keep them out of production.

## The check digit

The last digit is a Luhn check digit. Start at the digit just left of it and double every second digit going left. Where a doubled digit comes to 10 or more, add its two digits together. Add up everything, the check digit included. The number is correct when the total ends in 0.

For `4111111111111111`, the eight doubled digits give 8 and seven 2s, which is 22. The other eight digits are all 1, which is 8, and 22 + 8 is 30.

## Valid card numbers for your tests

{{< run "qadg quick card --count 5 --seed example" >}}

With only eight numbers behind it, a longer list repeats them. `--count 1000` gives a thousand lines drawn from the same eight.

## What your form should do with a broken one

Refuse it before it reaches the payment provider. On these eight numbers a changed digit always fails the Luhn check, and so does swapping two different digits side by side, so your form can catch both.
