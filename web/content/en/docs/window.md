---
title: The window
seoTitle: "The window: four screens"
description: "Quick value, Recipe, Damage a file and Check results: what each screen of the window does, and which command does the same."
lead: "`qadg-gui` puts the same engine in a window. Each of its four screens does what a command does, and shows that command ready to copy."
---

## Quick value

Pick what to generate and get ten values to copy. It does what `qadg quick` does.

{{< window name="quick-website" alt="The Quick value screen with ten vehicle identification numbers drawn from the seed readme" >}}
Ten VINs from the seed `readme`. The first three are the ones the example in [Getting started](/docs/getting-started/) prints, because the window and `qadg quick` draw the same values from the same seed.
{{< /window >}}

## Recipe

Write or paste a recipe. It is checked as you type, shows what it describes, and runs when you ask. It does what `qadg validate` and `qadg generate` do.

Every line of the recipe is numbered, the same way `qadg validate` counts lines in its messages. A line with a problem has a red mark beside its number, and a click on the problem on the right takes you to that line. Enter starts the next line indented like the one above, one step deeper after a line ending in a colon.

{{< window name="recipe-website" alt="The Recipe screen checking the shop.yaml recipe, with the files a run would write and the first rows of customers" >}}
The recipe from [Getting started](/docs/getting-started/), pasted in. The window checks it as you type, lists the files a run would write, says how much of the ordered damage this build carries out, and shows the first rows before anything is written.
{{< /window >}}

## Damage a file

Open a CSV file, see what each column holds and why, pick what to damage, see what would happen, then run it. It does what `qadg corrupt` does.

{{< window name="take-website" alt="The Damage a file screen with customers.csv read column by column and four kinds of damage chosen" >}}
`customers.csv`, the file [Damage a file](/docs/damage-a-file/) runs `qadg corrupt` on, read column by column, with the four kinds of damage that command orders. The panel on the right says what a run would do before anything is written.
{{< /window >}}

## Check results

Open a manifest, go through every ordered defect, and compare it with what your application did. It does what `qadg verify` does.

{{< window name="browser-website" alt="The Check results screen comparing a manifest with what an application did, with row 135 read out in full" >}}
The manifest of that run beside what an application did with every row. Row 135 is a card number with two digits swapped that the application took, though it should have been rejected. Entries marked not built yet are kinds of damage this build describes and does not carry out yet.
{{< /window >}}

## About

The version of the program and its licence, as `qadg --version` and `qadg --license` print them. Save the third-party licenses writes to a file the licences of the code, fonts and icons by other people inside the window, the part `qadg --license` prints for the command line.
