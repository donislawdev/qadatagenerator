---
title: Generators
seoTitle: "Generators: names, places, cards, ISBN, VIN, SSN"
description: "Everything QA Data Generator can fill a column with: people, places, card numbers, ISBN, EAN, IMEI, VIN, SSN, routing numbers and Polish national numbers."
lead: "A generator fills a column. A recipe names one with `generator:`, and `qadg quick` prints values of one without a recipe."
---

## Every generator

{{< generators >}}

`qadg help generators` prints the same table from the program itself.

## Values that agree with each other

- **People.** First and last names can follow the gender column with `first_name matches: [gender]`.
- **Places.** In `en-US` a city, its state and its ZIP code belong together. In `pl-PL` a town and a voivodeship are drawn apart, and there are no postal codes yet.
- **Polish national ID numbers.** A PESEL carries the same birth date and gender as the columns beside it.

## Check digits and test numbers

Every identifier that has a check digit comes with a correct one. Card numbers are only the test numbers payment providers publish, never a number made up to pass the check digit. Social security numbers start with 666, an area that is never issued, so a generated SSN cannot be anybody's. An application that checks the area will refuse them, valid rows included. [Identifiers](/identifiers/) explains every check digit, and which numbers can match real ones.

## Word lists

Names, cities and regions come from word lists for `en-US` and `pl-PL`. `--locale` or `locale:` in a recipe picks one.

## One value without a recipe

{{< readme marker="run" kind="run" output="output" >}}

`--count` sets how many, and `--seed` makes the values repeatable. Without a seed every run draws a new one and names it on the log channel, so a value you liked can be had again.
