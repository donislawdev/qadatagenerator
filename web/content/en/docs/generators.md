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
- **Places.** A city, its region and its postal code belong together. In `en-US` a region is a state and a postal code is a ZIP code.
- **Polish national ID numbers.** A PESEL carries the same birth date and gender as the columns beside it.

## Check digits and test numbers

Every identifier comes with a correct check digit. Card numbers are only the published test numbers, never a number of a real card. Social security numbers start with 666, an area that is never issued, so a generated SSN cannot be anybody's. An application that checks the area will refuse them, valid rows included.

## Word lists

Names, cities and regions come from word lists for `en-US` and `pl-PL`. `--locale` or `locale:` in a recipe picks one.

## One value without a recipe

{{< readme marker="run" kind="run" output="output" >}}

`--count` sets how many, and `--seed` makes the values repeatable. Without a seed every run draws a new one and names it on the log channel, so a value you liked can be had again.
