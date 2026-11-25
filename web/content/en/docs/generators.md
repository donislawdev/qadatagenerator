---
title: Generators
seoTitle: "Generators: names, places, cards, ISBN, VIN, SSN"
description: "Everything QA Data Generator can fill a column with: names, places, emails, passwords, UUIDs, test card numbers, ISBN, VIN, SSN and Polish national numbers."
lead: "A generator fills a column. A recipe names one with `generator:`, and `qadg quick` prints values of one without a recipe."
---

## Every generator

{{< generators >}}

`qadg help generators` prints the same table from the program itself.

## Values that agree with each other

- **People.** First and last names can follow the gender column with `first_name matches: [gender]`.
- **Places.** In `en-US` a city, its state and its ZIP code belong together. In `pl-PL` a town and a voivodeship are drawn apart, and there are no postal codes yet.
- **Polish national ID numbers.** A PESEL carries the same birth date and gender as the columns beside it.
- **Email addresses.** An address is made of the first and last name of its row, with no rule needed: a row for Adam Małyszek gets an address such as `amalyszek@example.org`. A row without those columns gets a name drawn for the address.

## Check digits and test numbers

Every identifier that has a check digit comes with a correct one. Card numbers are only the test numbers payment providers publish, never a number made up to pass the check digit. Social security numbers start with 666, an area that is never issued, so a generated SSN cannot be anybody's. An application that checks the area will refuse them, valid rows included. [Identifiers](/identifiers/) explains every check digit, and which numbers can match real ones.

## Email addresses, passwords and UUIDs

Email addresses end in `example.com`, `example.net` or `example.org`. These domains are set aside for documentation and take no mail, so a message your test sends to one reaches nobody. Accents are taken off the letters, so a Polish name gives a plain address. `max_length` shortens an address to no fewer than 13 characters, and a shorter limit is refused before anything is written.

Passwords are written in plain text, 12 to 16 characters long, each with a lowercase letter, an uppercase letter, a digit and one of `! # $ % & * + - = ? @ ^ _`, which is what common password rules ask for. None begins with `=`, `+`, `-` or `@`, which a spreadsheet would read as a formula. In Bash, zsh or PowerShell, put a password in single quotes. `max_length` shortens them to no fewer than 4 characters. Passwords come from the seed, so anybody with your recipe and seed gets the same ones: use them for test accounts, never for a real one.

UUIDs are version 4 of RFC 9562, random and in lowercase, and `type: uuid` gives the same values as `generator: uuid`. Nobody issues UUIDs, so there is no issued one a generated value could match. A UUID has no check digit, so damage aimed at check digits leaves it alone. A `max_length` below 36 is refused before anything is written.

## Word lists

Names, cities and regions come from word lists for `en-US` and `pl-PL`. `--locale` or `locale:` in a recipe picks one.

## One value without a recipe

{{< readme marker="run" kind="run" output="output" >}}

`--count` sets how many, and `--seed` makes the values repeatable. Without a seed every run draws a new one and names it on the log channel, so a value you liked can be had again.
