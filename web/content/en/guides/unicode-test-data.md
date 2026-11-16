---
title: Unicode test data
seoTitle: "Unicode test data: emoji, invisible spaces, dotless i"
description: "Values that look the same and are not: emoji built from several characters, a zero-width space, the Turkish dotless i, and one text in two Unicode forms."
lead: "Text that looks right on the screen can still be different text. These values catch code that counts, compares or changes the case of strings without Unicode in mind."
weight: 6
sources:
  - "Unicode Technical Standard #51, Unicode Emoji, version 18.0, read on 16 November 2026."
  - "Unicode Standard Annex #15, Unicode Normalization Forms, revision 58, read on 16 November 2026."
  - "SpecialCasing.txt of the Unicode Character Database, version 18.0.0, read on 16 November 2026."
  - "The utf8mb3 Character Set, MySQL 8.4 Reference Manual, read on 16 November 2026."
  - "Mikael Goldmann, Creative usernames and Spotify account hijacking, Spotify Engineering, 18 June 2013, read on 16 November 2026."
---

## Six values that look harmless

**An emoji, as in `Kłodzko 🙂`.** It is one of the characters that take four bytes in UTF-8. The `utf8` character set of MySQL, now called `utf8mb3`, holds at most three bytes a character and cannot store it. `utf8mb4` can.

**A family emoji, as in `Malbork 👩‍👩‍👧‍👦`.** One picture on the screen, made of seven code points: four people joined by three zero-width joiners. It is 25 bytes in UTF-8 and 11 units in UTF-16. Where a system has no picture for the family, it shows the four people side by side. A limit of 40 counted in one of these and a limit of 40 counted in what the reader sees are different limits, which is why `max_length` in a recipe takes `counted_in: bytes` or `graphemes` as well as characters.

**A zero-width space, as in `Kętrzyn` with one after `Kęt`.** It takes no room on the screen and still makes the value differ from the one without it. In a login or an email address it makes two accounts that look the same. The list below writes it out as a code, so you can see where it is.

**The Turkish dotless ı, as in `Wıeluń`.** Turkish has a dotted i and a dotless ı, each with a capital of its own: i and İ, ı and I. Turn `Wıeluń` into capitals and back and you get `wieluń`, because the capital of ı is the plain I. The other way round, outside Turkish the capital İ becomes two code points in small letters, an i and a combining dot. Code that changes the case of text to compare it can therefore match values it should not, or miss ones it should.

**The same text in another form, as in `Giżycko`.** The letter ż can be one code point, or a z followed by a combining dot above. Both look the same and Unicode treats them as the same text, but a comparison of bytes says they differ. Bring both sides to the same form, usually NFC, before you compare them, as Unicode Standard Annex #15 describes. The list below writes the combining dot out as a code.

**A value that changes when normalised twice, as in `ᴬᴸᴱᴷsᴬᴺᴰᴿzᴬᴷ`.** A function that puts this text into small letters and then normalises it to NFKC gives `ALEKsANDRzAK` the first time and `aleksandrzak` the second. If your application brings usernames to one form to find duplicates, such a function can send a password reset to somebody else's account. That happened at Spotify in 2013, and the fix was to refuse names that change on the second pass.

## Test data with all six

The recipe below uses Polish names and towns, because writing the same text in another form needs letters with marks, and Polish has plenty. It asks for Unicode characters (`UNI`) in about three rows in ten.

{{< example "recipe" >}}

{{< example "downloads" >}}

{{< example "run" >}}

{{< example "damage" >}}

Most answers are marked `~`, because they depend on your rules: an emoji in a city name may be fine in one system and a fault in another. `qadg verify` reports a disagreement on those and does not fail on it. The same text in another form is marked accept without `~`, because it is the same text.

## What to check in your code

- Count a length the way its limit means it: in bytes for storage, in graphemes for what a person sees.
- Bring text to one normalisation form, usually NFC, before you compare it or keep it unique.
- Store text in an encoding that holds four bytes a character, `utf8mb4` in MySQL.
- Decide where invisible characters are allowed, and refuse or remove them in logins and email addresses.
- Test changes of case with the language settings your application runs with, because the result depends on them.
