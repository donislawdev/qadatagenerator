---
title: Names your form should accept
seoTitle: "Names your form should accept: O'Brien, José, Null"
description: "Real names that careless validation refuses: apostrophes, accents, names in other scripts and the surname Null. Put them in your test data on purpose."
lead: "Some real names fail checks that look reasonable. Put them in your test data on purpose, and the manifest says each one should be accepted."
weight: 3
sources:
  - "Personal names around the world, W3C Internationalization, read on 16 November 2026."
  - "Patrick McKenzie, Falsehoods Programmers Believe About Names, 2010, read on 16 November 2026."
  - "Chris Baraniuk, These unlucky people have names that break computers, BBC Future, 25 March 2016, read on 16 November 2026."
  - "SQL Injection Prevention Cheat Sheet, OWASP, read on 16 November 2026."
---

## Four kinds of name that get refused

**An apostrophe: `O'Brien`, `D'Arcy`.** A check that allows only letters refuses them. Worse, code that builds an SQL query by pasting the name into its text breaks on the apostrophe, because in SQL an apostrophe ends a quoted text. SQL injection uses the same mistake, and the fix is the same: send the name as a parameter of the query, never as part of its text.

**A letter with a mark: José, Łukasz, Wójcik, Dvořák.** A pattern such as `[A-Za-z]+` refuses them, and even names in English can have such letters. Keep them in a Unicode encoding all the way from the form to the database, as the W3C advises, or the marks can be lost on the way.

**Chinese, Japanese or Korean writing: 佐藤, 민준, 李明.** The ones in this data are two characters long, so a rule that a name has at least three refuses them, and so does any check limited to the Latin alphabet.

**The surname Null.** It is a real surname. Code that writes a missing value as the word null, or reads that word back as missing, cannot tell this person from an empty field. In 2016 the BBC reported that airline websites had told Jennifer Null, again and again, that she had left the surname field blank.

## Test data with these names

The recipe below makes a hundred sign-ups and puts one of these names in about a fifth of them. Every name comes from a short list of real ones, so the value in your report is a name somebody has, not a misspelt one.

{{< example "recipe" >}}

{{< example "downloads" >}}

{{< example "run" >}}

{{< example "damage" >}}

Every one is marked accept. A rejection of any of them is a person your application turned away, and `qadg verify` reports it and ends with exit code 4. A name longer than its column is never put in: the column says what it holds, and your application is right to refuse a name that does not fit.

## What your form should do

- Accept letters of any script, with apostrophes and marks. Check the length, not the alphabet.
- Do not ask for more than one character. Some people have a name of one letter.
- Do not require a family name. Many people have a given name only.
- Keep names in a Unicode encoding from the form to the database.
- Send names to the database as parameters of the query.
- Treat the word Null as a name like any other. Only an empty field is a missing one.

Patrick McKenzie lists forty such false assumptions in Falsehoods Programmers Believe About Names, and the W3C article Personal names around the world explains how names differ from one culture to another.
