---
title: People in Poland
seoTitle: "Polish test data: names and PESEL numbers in CSV"
description: "A recipe for a hundred people in Poland in CSV: Polish names, gender, birth date, city, and a PESEL that agrees with the birth date and the gender."
lead: "A hundred people with Polish names, each with a PESEL that carries their birth date and gender and has a correct check digit."
weight: 2
---

First and last names follow the gender, so a woman is Kowalska rather than Kowalski. The rule `pesel matches: [birth_date, gender]` makes every PESEL agree with the two columns beside it, which is what a Polish form checks. A correct PESEL can belong to a real person, so keep the file in test systems.

The city comes from a list of Polish towns. This version has no register that says which voivodeship and which postal code a town has, so the recipe leaves both out rather than pair a town with the wrong ones.
