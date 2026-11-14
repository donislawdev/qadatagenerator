---
title: What it breaks
seoTitle: "What it breaks, and what your app should do"
description: "Every kind of damage this version does to a value, grouped as the window shows it, with what your application should do about each one."
lead: "Damage comes in groups. The window shows the group's name, and `--layers` on the command line or `layers:` in a recipe takes its code."
tocFacts: damage
---

Each kind of damage below is one this version carries out. The example is the one the catalog keeps beside it and holds to what the program really does. The badge beside each one is what an application should do with the damaged value, and the manifest writes the same answer for every value it changed.

`~` marks an answer that depends on the rules of your own application. Whether an emoji belongs in a city name is your call. `qadg verify` reports a disagreement on one of those and does not fail on it.

{{< damage >}}

## Choosing what to damage

On the command line, `qadg corrupt --layers L2,L3` takes the codes, and `--rows` sets the share of rows to damage, 5% unless you say otherwise. In a recipe, `policy:` takes the same codes in `layers:`. In the window, tick the groups by name.

Damage that does not fit your data is listed in the manifest too, with the reason, so a quiet run never looks like a clean one.
