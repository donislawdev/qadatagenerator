---
title: Mockaroo and QA Data Generator
linkTitle: Mockaroo
seoTitle: "Mockaroo vs QA Data Generator: an offline alternative"
description: "Mockaroo and QA Data Generator compared: a website or a program on your computer, types of data, formats, price, and broken data with expected results."
lead: "Mockaroo makes test data in a browser with no programming. QA Data Generator makes it on your computer, breaks it on purpose and says what your application should do with each broken value. This page says where each one fits."
sources:
  - "Mockaroo, the home page, mockaroo.com: up to 1,000 rows in CSV, JSON, SQL and Excel, no programming, mock APIs, saved schemas over its API, the Docker image, read on 16 November 2026."
  - "Mockaroo, Pricing, mockaroo.com/pricing: the Free, Silver, Gold and Enterprise plans, read on 16 November 2026."
  - "Mockaroo, API documentation, mockaroo.com/docs: percentBlank and the list of types, read on 16 November 2026."
  - "Mockaroo, the list of types in its schema editor: the descriptions of Naughty String and Blank, and the examples of SSN, read on 16 November 2026."
---

## What is Mockaroo?

A website for generating test data, at mockaroo.com. You list the fields, pick a type for each from more than 200, and download the rows as CSV, JSON, SQL or Excel, with no programming. It also mocks APIs, and a saved schema can be downloaded from a script through its API.

The free plan makes up to 1,000 rows per file and answers 200 API requests a day. Paid plans start at $60 a year. The Free, Silver and Gold plans are hosted on mockaroo.com, and the Enterprise plan runs as a Docker image in your own cloud or data centre. These facts are from its website, read on 16 November 2026.

## What does QA Data Generator do differently?

- **It runs on your computer.** The data is made where you run `qadg`, with no network connection and no account.
- **It breaks the data on purpose and says what should happen.** It knows {{< count "damage" >}}, from a check digit off by one to a name written in another Unicode form. A manifest beside the data names every damaged value and the expected result, and `qadg verify` compares what your application did with it. Mockaroo can leave a share of a field blank, and has a Naughty String type for "strings which have a high probability of causing issues".
- **Polish identifiers with check digits.** PESEL, NIP, REGON and a Polish IBAN. The list of types in Mockaroo's API documentation has no PESEL, NIP or REGON.
- **SSNs that belong to nobody.** Every SSN starts with 666, an area that is never issued. The examples in Mockaroo's list of types, such as 678-59-9455, are in areas that are issued. [SSN](/identifiers/ssn/) says which areas are issued.
- **The same seed gives the same data.** A recipe and a seed make the same files again on any machine, so you can keep the recipe instead of the data.
- **It damages a file you already have.** `qadg corrupt` reads a CSV, works out what each column holds and writes a damaged copy with its manifest.

## When is Mockaroo enough?

When you need valid data of many kinds quickly, without installing anything, or need it as SQL or Excel. Mockaroo has many more types of data than QA Data Generator, and it mocks APIs, which QA Data Generator does not do.

## How do they compare?

| | Mockaroo | QA Data Generator |
|---|---|---|
| What it is | a website, with an API | a program: the `qadg` command line and a desktop window |
| Where the data is made | on mockaroo.com, or in your own cloud with Enterprise | on your computer, with no network connection |
| Price | free up to 1,000 rows per file, paid plans from $60 a year | free, GPL-3.0 |
| Types of data | more than 200 | {{< count "generators" >}} |
| PESEL, NIP, REGON | not in the list of types in its API documentation | yes |
| Values made to test validation | a blank share for each field, a Naughty String type | {{< count "damage" >}}, and the expected result for each damaged value |
| Formats | CSV, JSON, SQL, Excel | CSV and JSON, and a manifest |
| Mock APIs | yes | no |

## Can I use both?

Yes. Make the valid data with Mockaroo, download it as CSV, and run it through `qadg corrupt`: it works out what each column holds, damages a copy and writes the manifest that says what your application should do with each damaged value. [Test a CSV import](/guides/test-a-csv-import/) shows the whole run.
