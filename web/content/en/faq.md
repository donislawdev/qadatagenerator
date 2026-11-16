---
title: Questions
seoTitle: "QA Data Generator FAQ: test data, seeds, licence"
description: "Answers about QA Data Generator: what it does, whether it is free for work, whether it needs the internet, and why a run on another machine gives the same data."
lead: "The questions people ask first, answered in the first sentence."
layout: faq
faq:
  - question: What is QA Data Generator?
    answer: "A test data generator for QA engineers and developers, with a command line and a desktop window. It makes whole datasets from a short YAML recipe, breaks them on purpose, and writes a manifest saying what your application should do with every value it damaged."
  - question: How is it different from a fake data library?
    answer: "A fake data library gives you values. QA Data Generator also gives you values that should fail, such as a card number whose check digit is off by one, and valid values careless validation refuses, such as O'Brien, with the right answer for each in a file your test reads. [What it breaks](/docs/damage/) lists every kind of damage, and [QA Data Generator and other tools](/compare/) sets it beside Faker and Mockaroo."
  - question: Is it free? Can I use it at work?
    answer: "Yes. The code is GPL-3.0. Data, files and manifests you generate are yours: they are not covered by the licence, and you can use them in closed source products and commercial test suites."
  - question: Does it need the internet?
    answer: "No. It opens no network connection: no account, no telemetry, no update check."
  - question: Will I get the same data on another machine?
    answer: "Yes. The same recipe and seed give the same bytes on Windows, macOS and Linux, including which rows were damaged and how."
  - question: Can it damage a CSV file I already have?
    answer: "Yes. `qadg corrupt` works out what each column holds by looking at the values, damages a copy, and writes the manifest beside it. Your file stays as it was. [Damage a file you already have](/docs/damage-a-file/) shows a run."
  - question: Can I fail a build on a wrong answer?
    answer: "Yes. Write down what your application did with each row, and `qadg verify` compares it with the manifest. It ends with exit code 4 when they disagree. [Using it in CI](/docs/ci/) has every exit code."
  - question: Why do generated SSNs start with 666?
    answer: "That area is never issued, so a generated SSN cannot be anybody's. An application that checks the area will refuse them, valid rows included."
  - question: Are the card numbers real?
    answer: "They are the test numbers payment providers publish, not numbers the program makes up. None of the providers promises that no bank has issued the same number to a real card, so keep them in test systems."
  - question: Which output formats does it write?
    answer: "CSV and JSON for the data. The manifest is JSON, and can also be written as a Gherkin Examples table, as CSV or as Markdown."
  - question: Need test files rather than test records?
    answer: "[Testing Files Generator](https://github.com/donislawdev/TestingFilesGenerator) makes real files in dozens of formats at any exact size, with a manifest of its own."
---
