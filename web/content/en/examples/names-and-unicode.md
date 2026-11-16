---
title: Names and Unicode
seoTitle: "A recipe for names and Unicode that careless code breaks"
description: "A recipe that puts emoji, zero-width spaces, a Turkish dotless i, CJK names and names like O'Brien or Null into a table, with the right answer for each."
lead: "Real names a strict check refuses, and Unicode that careless code mangles, in about a fifth of the rows, each with what your application should do."
weight: 7
---

The values marked accept are names real people have: O'Brien, Łukasz, a name in Chinese characters, even the surname Null. An application that refuses them turns people away. The Unicode marked with ~ depends on your own rules: an emoji in a city name may be fine in one system and a fault in another, and a zero-width space is invisible on the screen and still makes two values differ. The manifest says which is which, and you decide the ones marked ~.

Run it again with another `seed` for other names and other damage.
