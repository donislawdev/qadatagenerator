---
title: Faker and QA Data Generator
linkTitle: Faker
seoTitle: "Faker vs QA Data Generator for test data"
description: "Faker and QA Data Generator compared: values from Python code or datasets from a recipe, seeds, Polish identifiers, and values made to test validation."
lead: "Faker gives your code values. QA Data Generator writes whole datasets, breaks them on purpose and says what your application should do with each broken value. This page says where each one fits."
sources:
  - "Faker documentation for version 40.54.0, faker.readthedocs.io: the first page (seeding, unique values, the command line, Factory Boy, pytest) and the list of localized providers, read on 16 November 2026."
  - "Faker documentation for version 40.54.0, the providers of the locales pl_PL and en_US: pesel, nip, regon, identity_card_number, iban, ssn, invalid_ssn and credit_card_number, read on 16 November 2026."
  - "Faker on PyPI: version 40.54.0 of 10 November 2026, MIT licence, read on 16 November 2026."
  - "Faker for JavaScript, fakerjs.dev, Usage, the section Reproducible results, read on 16 November 2026."
  - "Measured on 16 November 2026: Faker 40.54.0 on Python 3.13, locale en_US, seed 4321, five rows of first_name and last_name, then five rows of first_name, email and last_name."
---

## What is Faker?

A Python package that generates fake data, published under the MIT licence. You create a generator in your code and call it once for every value: a name, an address, a PESEL. Its list of locales has 110 entries, Polish among them. It has a pytest plugin, works with Factory Boy to fill models, and its `faker` command prints values in a terminal.

The facts on this page are from version 40.54.0, published on 10 November 2026, and from its documentation, read on 16 November 2026. Faker for JavaScript, `@faker-js/faker`, is a separate project. This page is about Faker for Python.

## What does QA Data Generator do differently?

- **It writes datasets from a recipe, not from code.** A YAML file names the tables, the columns, how many rows and the keys between tables, and `qadg generate` writes them as CSV, JSON or SQL. The desktop window reads the same recipe.
- **It breaks the data on purpose and says what should happen.** It knows {{< count "damage" >}}, from a check digit off by one to a name written in another Unicode form. A manifest beside the data names every damaged value and the expected result, and `qadg verify` compares what your application did with it.
- **A new column leaves the others as they were.** Each value is drawn from its own place: the table, the row, the column and the seed. In Faker one seed starts one sequence of random numbers, and every call takes the next one. A five-row table drawn with Faker 40.54.0 and the seed 4321 changed in every last name, and in every first name after the first row, when an email column was put between the two.
- **SSNs that belong to nobody, and test card numbers.** Every SSN starts with 666, an area that is never issued, and card numbers are only the test numbers payment providers publish. Faker's `ssn()` gives numbers in areas that are issued, such as 042-34-8377 from its documentation, and its documentation says `credit_card_number()` generates "a valid credit card number". [SSN](/identifiers/ssn/) says which areas are issued.
- **It damages a file you already have.** `qadg corrupt` reads a CSV, works out what each column holds and writes a damaged copy with its manifest.

## When is Faker enough?

When your test builds its data in code, one object at a time, and needs only valid values. Faker lives inside the test, knows many more locales and kinds of data, and has more Polish identifiers than QA Data Generator: besides PESEL, NIP, REGON and IBAN, also the number of an identity card. QA Data Generator is a program, not a library, so your test reads the files it writes.

## How do they compare?

| | Faker | QA Data Generator |
|---|---|---|
| What it is | a Python package your code calls | a program: the `qadg` command line and a desktop window |
| How you describe the data | Python code, one call for each value | a YAML recipe with tables, rows and keys |
| Locales | 110 entries in its list | {{< count "locales" >}} |
| Polish identifiers | PESEL, NIP, REGON, identity card number, IBAN | PESEL, NIP, REGON, IBAN |
| SSN | in areas that are issued | always in area 666, never issued |
| Card numbers | "a valid credit card number" | only published test numbers |
| Values made to test validation | for example `invalid_ssn()` | {{< count "damage" >}}, and the expected result for each damaged value |
| A column added in the middle | changes the values drawn after it | leaves the other columns as they were |
| The same seed in a newer version | "not guaranteed to be consistent across patch versions" | can draw some values differently |
| What you get | values in your code, and `faker` prints them | CSV, JSON or SQL files, and a manifest |
| Licence | MIT | GPL-3.0, and the data you generate is yours |

## Can I use both?

Yes, and QA Data Generator already does: its lists of first and last names, and the Polish cities and voivodeships, come from Faker 40.36.0 under the MIT licence, and `qadg --license` names them. Use Faker where a test builds one object in code, and QA Data Generator where a test needs a whole dataset, damaged values and the right answer for each.
