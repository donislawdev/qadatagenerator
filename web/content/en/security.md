---
title: Every value is made up
seoTitle: "Every value is made up: privacy and safety"
description: "Every value QA Data Generator makes is made up. No real people, no network connection, no telemetry. What that means for SSNs, card numbers and your own files."
lead: "Test data has to look real to prove anything, and it must never be anybody's. This page says how the program keeps both, and what it promises about your files."
---

## Every value is made up

Names, addresses, identifiers, card and account numbers: none of them describe a real person, company or institution, and no register of real people was used to build them. Where a value looks real, that is arithmetic, because a test that only ever sees obviously fake data proves nothing about the day real data arrives. If you recognise one, it is a coincidence.

- **Social security numbers start with 666.** That area is never issued, so a generated SSN cannot be anybody's. An application that checks the area will refuse them, valid rows included.
- **Card numbers are only the test numbers payment providers publish.** The program never makes up a number that merely passes the check digit. None of the providers promises that no bank has issued the same number to a real card, so keep them in test systems like the rest.
- **Some numbering schemes have no block set aside for testing.** A value of such a scheme may match one that was really issued, and the window says so beside it. [SECURITY.md](https://github.com/donislawdev/qadatagenerator/blob/main/SECURITY.md) lists which schemes have such a block and which have none.

A generated dataset is safe in a test environment and is not safe to present as a record of anybody. Mark it as synthetic wherever it travels. `qadg --license` prints the same notice, because a web page does not travel with a file of data.

## No network connection

The program opens no network connection: no account, no telemetry, no update check. It works the same on a machine with no network at all.

## Your files stay as they were

- A run writes nothing until it has succeeded, so a run that fails or is stopped leaves the output directory as it found it.
- An existing file stops a run unless you add `--overwrite`.
- The file you give `qadg corrupt` is never written over. The damaged copy is written beside it under another name.

## The data is yours

The code is GPL-3.0. Data, files and manifests you generate are yours: they are not covered by the licence, and you can use them in closed source products and commercial test suites.

## Reporting a security problem

[SECURITY.md](https://github.com/donislawdev/qadatagenerator/blob/main/SECURITY.md) says how to report a vulnerability privately.
