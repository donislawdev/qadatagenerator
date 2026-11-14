---
title: People in the US
seoTitle: "US test data: names, SSNs and ZIP codes in CSV"
description: "A recipe for a hundred US people in CSV: name, gender, birth date, a social security number that is never issued, and a city with its state and ZIP code."
lead: "A hundred people in one table, with places that belong together and social security numbers nobody can have."
weight: 1
---

First names follow the gender column. The state and the ZIP code follow the city and come from the same postal register, so a city, its state and its ZIP code belong together. Every social security number is in area 666, which is never issued, so a test cannot touch a real person's number.

Change `count` for more rows and `seed` for other people. Remove a column you do not need, or add one from the [generators](/docs/generators/).
