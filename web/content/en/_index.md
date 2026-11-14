---
title: QA Data Generator
seoTitle: "Test data generator: good data, bad data, expected results"
description: "Generate realistic test data from a YAML recipe, then break it on purpose. Each damaged value comes with what your app should do. Free, offline, CLI and GUI."
hero:
  eyebrow: Test data generator for QA
  title: Good data. Bad data.
  turn: And the right answer.
  lead: "**QA Data Generator** turns a short YAML recipe into whole datasets that pass validation, in CSV or JSON. Then it breaks them on purpose, and writes beside the data what your application should do with every value it damaged."
  facts:
    - Free and open source, GPL-3.0
    - Offline, no account, no telemetry
    - Windows, macOS and Linux
  good: good data
  should: your app should
  bad: bad and tricky data
  file: expectation-manifest.json
  note: From real runs of qadg quick and qadg corrupt.
recipes:
  eyebrow: Recipes
  title: A whole dataset from one short file
  text: "Describe your tables, how many rows each one has and how they relate. A customer can have zero to five payments, every payment points at its customer, and one command writes all of them."
  points:
    - "`belongs_to: { customer: 0-5 }` gives each customer zero to five payments, each carrying the key of its customer."
    - "`matches` keeps first names in line with the gender column. City, state and ZIP code agree without a rule."
    - "`required` and `max_length` describe your columns, and tell the damage below what to break."
    - "`seed` makes the run repeatable. Change it for different data of the same shape."
  link:
    text: Every key of a recipe
    page: /docs/recipes
damage:
  eyebrow: Damage on purpose
  title: Bad data, and the answer your test needs
  text: "Add three lines to the recipe and the same run damages a share of the rows. Beside the data it writes `expectation-manifest.json`, with an entry for every value it changed: where it is, the value before and after, and what your application should do with it. Your test reads the answer from that file instead of you writing it by hand."
  added: added to shop.yaml
  note: "`~` marks an answer that depends on the rules of your own application. `qadg verify` reports a disagreement on one of those and does not fail on it."
  link:
    text: How the manifest works
    page: /docs/manifest
good:
  eyebrow: Good data
  title: Values that pass validation, and agree with each other
  text: "Every identifier carries a correct check digit, and card numbers are only the published test numbers. Names, cities and regions come from word lists for the United States and Poland."
  cards:
    - title: People
      text: First and last names that can follow the gender column, and birth dates.
      generators: [first_name, last_name, gender, birth_date]
    - title: Places
      text: A city, its state and its ZIP code that belong together.
      generators: [city, region, postal_code]
    - title: Test card numbers
      text: Card numbers that pass validation, taken only from the published test numbers.
      generators: [card]
    - title: Books, barcodes, phones, vehicles
      text: Book numbers, product barcodes, mobile phone identities and vehicle identification numbers.
      generators: [isbn, ean, imei, vin]
    - title: United States
      text: Social security numbers that belong to nobody, and bank routing numbers.
      generators: [ssn, aba]
    - title: Poland
      text: National ID numbers that agree with the birth date and gender, tax and business numbers, and bank account numbers.
      generators: [pesel, nip, regon, iban]
  quick: one value in a second
  link:
    text: Every generator
    page: /docs/generators
groups:
  eyebrow: Bad data
  title: Five kinds of damage
  text: "Choose which kinds to apply, in the window or with `--layers`. The manifest holds the answer for every single value, and the badge shows the usual one."
  cards:
    - code: L2
      text: Makes a value longer than its column allows, empties a required field, puts letters in a number.
      answer: reject
    - code: L3
      text: Moves a check digit by one, swaps two digits, takes a number from a block that is never issued.
      answer: reject
    - code: L4
      text: Makes a Polish national ID number disagree with the birth date or gender beside it.
      answer: reject
    - code: UNI
      text: Adds an emoji, an invisible zero-width space, a Turkish dotless ı, or writes the same text in another Unicode form.
      answer: "~ your rules"
    - code: REV
      text: "Puts in a real name that careless validation refuses: O'Brien, José, 美咲, Null."
      answer: accept
  link:
    text: Every kind of damage
    page: /docs/damage
yours:
  eyebrow: Your own data
  title: Damage a file you already have
  text:
    - "Point `qadg corrupt` at a CSV file. It works out what each column holds by looking at the values, says why it decided so, and lists what it could not work out rather than guessing."
    - "Your file stays as it was. The damaged copy and its manifest are written beside it."
  links:
    - text: Damage a file you already have
      page: /docs/damage-a-file
pipeline:
  eyebrow: In your pipeline
  title: Check what your application did
  text: "Import the damaged file, write down what your application did with each row, and `qadg verify` lists every row where it disagrees with the manifest. It ends with exit code 4, so your pipeline can fail on it."
  links:
    - text: Check what your app did
      page: /docs/verify
    - text: Using it in CI
      page: /docs/ci
window:
  eyebrow: Desktop window
  title: The same engine, in a window
  text: "`qadg-gui` has four screens for the work, and each one does what a command does."
  cards:
    - title: Quick value
      text: Pick what to generate and get ten values to copy.
      command: "`qadg quick`"
    - title: Recipe
      text: Write or paste a recipe. It is checked as you type, shows what it describes, and runs when you ask.
      command: "`qadg validate` and `qadg generate`"
    - title: Damage a file
      text: Open a CSV file, see what each column holds and why, pick what to damage, see what would happen, then run it.
      command: "`qadg corrupt`"
    - title: Check results
      text: Open a manifest, go through every ordered defect, and compare it with what your application did.
      command: "`qadg verify`"
  link:
    text: The window
    page: /docs/window
promises:
  eyebrow: Careful by design
  title: Safe to run next to real data
  items:
    - title: Offline
      text: No network connection, no account, no telemetry and no update check.
    - title: Same seed, same data
      text: The same recipe and seed give the same rows and the same damage, byte for byte, on Windows, macOS and Linux.
    - title: Your files stay as they were
      text: "A run writes nothing until it has succeeded. An existing file stops it unless you add `--overwrite`, and the file you give `qadg corrupt` is never written over."
    - title: Every value is made up
      text: No register of real people was used. Social security numbers start with 666, an area that is never issued.
    - title: The data is yours
      text: The code is GPL-3.0. Data, files and manifests you generate are not covered by it, and you can use them in commercial test suites.
    - title: Open formats
      text: The damage catalog, the manifest and the record of what your application did are written down, for other tools to read and write.
  link:
    text: Every value is made up
    page: /security
cta:
  title: Your first dataset in two minutes
  text: Free and open source. Build it, write a recipe, and see what your application does with bad data.
---
