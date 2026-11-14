---
title: A sign-up form
seoTitle: "Form validation test data: PESEL and required fields"
description: "A recipe for Polish sign-ups where some rows break a rule a form should enforce: a name too long, or a PESEL that disagrees with the birth date or gender."
lead: "A hundred sign-ups for a Polish form, with one thing the form should catch ordered in about a fifth of them."
weight: 8
---

The recipe says what the form expects: names are required and have a longest length, and the PESEL agrees with the birth date and the gender. The damage breaks those rules one at a time, so every refused row has a single cause. The manifest says each damaged value should be rejected, and the clean rows should go through.
