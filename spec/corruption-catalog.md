# Corruption Catalog Specification

**Version 0.1, draft.**

This document defines what a corruption catalog is: how a mutator is identified, what metadata every entry carries, and which rules govern the catalog as it grows. It also lists the starter registry.

A mutator is a named, parameterised way of damaging data. It is not the damage itself. One application of a mutator to one place in one dataset is a **defect**, and defects are described by the [expectation manifest](expectation-manifest.md).

---

## Status

**Draft. No tool implements this yet.** Until the first public release, identifiers in this document can still change. After that release they are permanent, because they end up in manifests that live for years inside other people's test repositories.

This is a documented, versioned format open to implementation. It is not a standard. A format that one tool emits is that tool's format until somebody else adopts it.

**The examples in the entries are illustrative.** Their check digits, postal codes and identifier ranges have not been verified against an official source. They show the shape of the damage, not a correct value. An implementation produces them from the real algorithms and reference data.

---

## Identifier grammar

```
<layer>.<area>.<variant>
```

- `layer` is one of `L0` `L1` `L2` `L3` `L4` `L5` `SEC` `UNI` `REV`
- `area` and `variant` are lowercase ASCII letters, digits and underscores
- the full identifier is case sensitive and never contains spaces

Examples: `L3.checksum.off_by_one`, `L5.unique.case_collision`, `REV.email.plus_addressing`.

Write identifiers in full. A bare `off_by_one` means nothing.

### Naming: property, not country

Name a mutator after the **property** when the property is general, and after the **standard** when the mechanism belongs to one named standard.

- `L3.national_id.impossible_date` - general. Any national identifier that encodes a date can carry this defect.
- `L3.iban.wrong_length_for_country` - specific. The mechanism belongs to IBAN and nothing else.

The test is whether another country could reuse the identifier without stretching its meaning. A mutator named after one country forces a contributor from another to invent a second identifier for the same phenomenon, and manifests from the two stop being comparable.

---

## Layer model

The layer answers one question: **which component of the application under test is supposed to catch this defect.** It does not describe severity and it does not describe how far the value strayed.

| Layer | What is damaged | What should catch it |
|---|---|---|
| `L0` | bytes and encoding | the parser |
| `L1` | container structure | the deserializer |
| `L2` | field type and format | the schema validator |
| `L3` | domain semantics of one value | the domain validator |
| `L4` | business rules across a record | application logic |
| `L5` | integrity across a set of records | the database or data access layer |

The distinction between `L3` and `L4` is the one worth internalising. At `L3` a national identifier carries a broken check digit, which one field reveals on its own. At `L4` the identifier is perfectly valid but encodes a different year than the birth date field beside it, which takes two fields to see.

---

## Cross-cutting modules

Three groups cannot be assigned to a single layer, because they describe **which value to substitute** rather than **who should catch it**. The same string from `UNI` can break a parser at `L0`, violate a length limit at `L2` or cause a uniqueness collision at `L5`.

| Module | Contents | Default |
|---|---|---|
| `SEC` | security payloads for testing your own application | **never enabled by default** |
| `UNI` | Unicode and internationalisation |  |
| `REV` | reverse mutators |  |

**Reverse mutators are the inverse of the rest of the catalog.** They supply data that is entirely valid but tends to be rejected by overzealous validation. Their expected outcome is `acceptance`, not `rejection`. A false rejection costs more than a false acceptance, because it turns away a user who had every right to proceed.

**`UNI` is not a module for breaking things either.** Most of its entries also expect `acceptance`. Emoji, diacritics and NFC or NFD normalisation are values an application is supposed to take, and refusing them is the defect. Only a few, such as the zero width space, expect rejection.

Modules group entries by the **kind of value** they supply, not by the outcome expected of the application. Expectation is a property of the entry, so two entries in one module can expect opposite things.

---

## Catalog entry

Every entry carries all of the following. An entry missing any of them does not belong in the catalog.

| Field | Meaning |
|---|---|
| `id` | stable identifier, see grammar above |
| `layer` | one of the layers or cross-cutting modules |
| `summary` | one sentence in plain language |
| `applies_to` | field types, container formats and **providers** on which this mutator is meaningful |
| `parameters` | what can be tuned, with defaults |
| `expectation` | `rejection`, `acceptance` or `warning` |
| `confidence` | `hard` or `soft` |
| `reversible` | whether the original value can be reconstructed from the manifest |
| `requires` | what the valid data generator must be able to produce for this mutator to work |
| `example` | at least one before and after pair |

**The entries live in [corruption-catalog.yaml](corruption-catalog.yaml), which is the canonical form.** This document describes the shape and the rules.

It does repeat three fields per entry in the starter registry below, namely the identifier, the expectation and the confidence, because a reader needs to see the shape of the registry without opening a data file. **Those three can drift, so they are checked mechanically against the YAML rather than by eye.** Everything else about an entry exists in one place only.

### Optional fields

| Field | Meaning |
|---|---|
| `applicability_condition` | present when the mutator only makes sense under a stated condition. A policy requesting it outside that condition produces a manifest entry with `applied` set to false and the condition as the reason |
| `limitation` | a consequence the user has to be told about, including one that reaches beyond the corrupted rows |
| `example.note` | one paragraph on why the entry is worth running, or on what makes the default parameter the valuable variant |

### `applies_to`

Three dimensions, each a list: `field_types`, `formats` and `providers`. Two values are reserved. **`any`** means the mutator is meaningful for every value of that dimension. **`none`** means the dimension does not apply, which is the normal case for a mutator that damages the file rather than a field.

Provider identifiers use `<country>.<identifier>` for country bound providers, with the ISO 3166-1 alpha-2 code, and a bare name for international standards. So `PL.pesel` and `US.ssn`, but `iban`, `vin` and `card`.

**A provider belongs to a country, not to a language.** A social security number is a US identifier whatever language its holder speaks, so a locale prefix would force a second provider for the same number as soon as a second locale of one country appears. A data locale selects providers. It does not name them.

Provider identifiers become permanent at the first public release, on the same terms as mutator identifiers. Until then they can still change.

### `requires` and the capability list

`requires` names capabilities rather than describing them in place, and every name is defined once at the end of the catalog file. Read together, those definitions are the set of capabilities the valid data generator has to carry before the starter registry works end to end. A mutator that needs something the generator cannot produce is visible as a name with no definition behind it.

### Expectation

- `rejection` - the application should refuse this input
- `acceptance` - the application should accept it, and refusing is the defect
- `warning` - the application should accept it but flag it

**`acceptance` means the value survives, not merely that no error appeared.** An application that stores something other than what it was given has not accepted the value, it has accepted a different one. Silent type conversion is the common case, and it fails this expectation just as a rejection would. State this in the entry rationale whenever a mutator exists to catch mangling rather than refusal.

### Confidence

- `hard` - a specification, algorithm or type is violated. The application must react. No reaction is a defect in the application.
- `soft` - the correct reaction depends on the business rules of the system under test. No reaction is not necessarily a defect.

Without this distinction a manifest mixes "must react" with "probably should react", produces false alarms and loses trust within a week.

---

## Availability depends on the provider

The layer model is country neutral. **Individual mutators at `L3` and `L4` are not**, because identifiers differ in what they even have to break.

| Mutator | `pl-PL` | `en-US` | Reason |
|---|---|---|---|
| `L3.checksum.off_by_one` on the national person identifier | yes | **no** | the Polish PESEL carries a check digit, the US SSN does not |
| `L4.coherence.national_id_vs_birthdate` | yes | **no** | PESEL encodes a birth date, SSN encodes nothing |
| `L3.range.unassigned` on the national person identifier | **no** | yes | SSN has officially unassigned ranges, PESEL does not |

Two consequences follow.

1. **`applies_to` must name providers**, not only field types. Without it a contribution from another country has nothing to attach to.
2. **A policy requesting a layer with no applicable mutator for the active locale produces a manifest entry stating that**, never a silent omission. A dataset where twenty seven of forty requested defects were applied looks exactly like a complete one.

---

## Rules

1. **Identifiers are permanent.** Retiring a mutator means marking it deprecated. Never delete it and never move it to another layer.
2. **A mutator without a declared expectation does not exist.** Damage whose expected outcome is unknown is useless in an automated test, and that is the exact problem this whole format solves.
3. **Every entry carries at least one before and after example.**
4. **An entry with `soft` confidence explains what the reaction depends on.**
5. **Security payloads live in `SEC` only** and are never part of a default selection.
6. **An entry that needs a capability from the valid data generator declares it in `requires`.** This keeps visible what has to exist before the mutator can work at all.

---

## Starter registry

The full catalog is larger. This is the set an implementation should support first, chosen so that every layer is represented by something that finds real defects.

### `L0` bytes and encoding

| `id` | Expectation | Confidence |
|---|---|---|
| `L0.encoding.wrong_charset` | rejection | hard |
| `L0.eol.mixed` | rejection | soft |
| `L0.file.truncated` | rejection | hard |
| `L0.byte.nul_injected` | rejection | hard |
| `L0.bom.unexpected` | rejection | soft |

`L0.file.truncated` can cut in the middle of a multi byte character. That variant breaks parsers which assume one byte is one character.

### `L1` container structure

| `id` | Expectation | Confidence |
|---|---|---|
| `L1.csv.ragged_row` | rejection | hard |
| `L1.csv.unescaped_quote` | rejection | hard |
| `L1.csv.wrong_delimiter` | rejection | hard |
| `L1.json.unclosed_brace` | rejection | hard |
| `L1.json.duplicate_key` | warning | soft |

`L1.json.duplicate_key` is a warning rather than a rejection because parsers disagree on whether the first or the last value wins, and both behaviours exist in the wild.

### `L2` field type and format

| `id` | Expectation | Confidence |
|---|---|---|
| `L2.string.exceeds_maxlength` | rejection | hard |
| `L2.string.empty` | rejection | soft |
| `L2.type.string_in_numeric` | rejection | hard |
| `L2.enum.outside_set` | rejection | hard |
| `L2.date.impossible` | rejection | hard |
| `L2.date.wrong_format` | rejection | hard |
| `L2.number.comma_decimal_separator` | rejection | soft |
| `L2.coercion.text_to_date` | acceptance | soft |
| `L2.coercion.text_to_number` | acceptance | soft |

`L2.string.exceeds_maxlength` overshoots by exactly one character by default. That is the classic boundary and the most common place where a length check is written with an off by one error.

### `L3` domain semantics

| `id` | Expectation | Confidence |
|---|---|---|
| `L3.checksum.off_by_one` | rejection | hard |
| `L3.checksum.transposed_digits` | rejection | hard |
| `L3.national_id.impossible_date` | rejection | hard |
| `L3.date.future_birthdate` | rejection | soft |
| `L3.currency.wrong_scale` | rejection | hard |
| `L3.range.unassigned` | rejection | soft |
| `L3.sentinel.placeholder_value` | warning | soft |

`L3.sentinel.placeholder_value` is the only entry in the whole starter registry whose expectation is `warning`. A placeholder value is present, well formed and empty of meaning, so neither accepting it as real nor refusing it outright is the answer the entry looks for. The application is expected to notice.

`L3.range.unassigned` is the only entry here that applies to the US national person identifier and not to the Polish one, so without it a US dataset gets no `L3` defect on that field at all. It carries an `applicability_condition`, because generated valid data already sits inside reserved ranges wherever they exist.

`L3.checksum.transposed_digits` swaps two adjacent digits, which is the most common human typing mistake. Some check digit algorithms do not detect it, and that is worth testing too.

### `L4` business rules

| `id` | Expectation | Confidence |
|---|---|---|
| `L4.coherence.national_id_vs_birthdate` | rejection | soft |
| `L4.coherence.national_id_vs_gender` | rejection | soft |
| `L4.coherence.postal_vs_city` | rejection | soft |
| `L4.order.dates_reversed` | rejection | soft |
| `L4.sum.invoice_total_mismatch` | rejection | soft |
| `L4.range.discount_over_100` | rejection | soft |
| `L4.coherence.declared_rule` | rejection | soft |

`L4.coherence.declared_rule` is the only entry in this layer that carries no domain knowledge of its own. Every other coherence entry names one pair of fields, which covers the pairs the catalog authors thought of. This one breaks whichever consistency rule the recipe declares, which covers the pairs the user thought of. Both are needed. The specific entries know things the generic one cannot, such as the fact that a postal code needs reference data to be checked at all.

Every entry here is `soft` on purpose. Business rules belong to the system under test, and an application that tolerates any of these may be doing so deliberately.

### `L5` set integrity

| `id` | Expectation | Confidence |
|---|---|---|
| `L5.fk.orphan` | rejection | hard |
| `L5.pk.duplicate` | rejection | hard |
| `L5.unique.case_collision` | rejection | soft |
| `L5.entity.near_duplicate` | rejection | soft |
| `L5.aggregate.mismatch` | rejection | soft |
| `L5.cardinality.violated` | rejection | soft |

`L5.aggregate.mismatch` and `L5.cardinality.violated` share a property that makes them worth the cost. Neither can be found by looking at a record. One needs the set added up, the other needs a record that is not there, and an application that validates row by row will pass both.

`L5.unique.case_collision` produces `Jan@example.test` next to `jan@example.test`. Whether that is a collision depends on the collation of the database, which is exactly why it is worth generating.

### `UNI` Unicode and internationalisation

| `id` | Expectation | Confidence |
|---|---|---|
| `UNI.emoji.basic` | acceptance | soft |
| `UNI.emoji.zwj_sequence` | acceptance | soft |
| `UNI.zero_width.space` | rejection | soft |
| `UNI.normalization.nfc_vs_nfd` | acceptance | hard |
| `UNI.normalization.non_idempotent` | rejection | soft |
| `UNI.case.turkish_dotless_i` | acceptance | hard |

`UNI.emoji.zwj_sequence` supplies a single visible character built from seven code points. It breaks length counters that measure the wrong unit.

### `REV` reverse mutators

| `id` | Expectation | Confidence |
|---|---|---|
| `REV.email.plus_addressing` | acceptance | hard |
| `REV.name.apostrophe` | acceptance | hard |
| `REV.name.diacritics` | acceptance | hard |
| `REV.name.null_surname` | acceptance | hard |
| `REV.name.cjk` | acceptance | hard |
| `REV.amount.negative_legal` | acceptance | soft |
| `REV.identifier.reserved_range` | acceptance | soft |

`REV.amount.negative_legal` is the first entry in this module that is not a piece of text about a person. Until it was added, `REV` covered email addresses, names and addresses and nothing else, while the documented cases of valid data being refused included a market price below zero and a national identifier that no longer carries a check digit. A module named for valid data that gets refused has no reason to stop at text.

`REV.name.null_surname` supplies the surname `Null`, which is a real surname and a well known breaker of systems that conflate it with an absent value.

---

## Versioning

The catalog carries a version. Manifests record which version produced them.

- **Adding** a mutator is a minor change.
- **Deprecating** one is a minor change. The identifier stays.
- **Changing the meaning of an existing identifier, its layer or its expectation is a breaking change** and requires a major version.

Consumers must ignore identifiers they do not know rather than failing on them. A manifest produced by a newer catalog stays readable.
