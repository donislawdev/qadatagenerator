---
title: Documentation
seoTitle: "QA Data Generator documentation - recipes, damage, CLI"
description: "How QA Data Generator works: recipes, generators, the damage it does, the manifest, checking what your application did, the window and every command."
lead: "Everything the command line and the window do, and how to read what they tell you."
cards:
  - /docs/getting-started
  - /docs/recipes
  - /docs/generators
  - /docs/damage
  - /docs/manifest
  - /docs/damage-a-file
  - /docs/verify
  - /docs/window
  - /docs/ci
  - /docs/cli
  - /docs/exit-codes
---

QA Data Generator makes test data in two kinds. **Good data** passes validation: whole datasets from a recipe, where names, places and identifiers agree with each other and every check digit is correct. **Bad data** is the same data broken on purpose, beside a manifest that says what your application should do with every value that was changed.

It comes as two programs over one engine. `qadg` is the command line, for a terminal and a pipeline. `qadg-gui` is a window with the same engine behind four screens.

If you have never run it, start with [Getting started](/docs/getting-started/). If you already have a CSV file to test with, go to [Damage a file you already have](/docs/damage-a-file/).
