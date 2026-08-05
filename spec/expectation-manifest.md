# Expectation Manifest Specification

**Version 0.1, draft.**

An expectation manifest accompanies a deliberately corrupted dataset. For every defect it records where it is, what produced it, and **what the application under test is expected to do about it**.

Without that last part corrupted data is useless in an automated test, because the test has no way to know what to expect. This is why lists of hostile strings never made it into anybody's pipeline. They supply the input and leave out the answer.

---

## Status

**Draft. No tool emits this yet.** It is a documented, versioned format open to implementation. It is not a standard. A format that one tool emits is that tool's format until somebody else adopts it, and one implementation is not two.

### This format is still changing, and here is when it stops

**Everything in it is mutable until the first public release of the reference implementation.** From that release the rules under Versioning apply and every change to an existing field is a breaking one.

That sentence is here because the moment a format freezes is a decision nobody makes on purpose. What happens instead is that one day somebody already depends on it, and from that day every change breaks something without anybody having announced it. The transition is only ever visible in hindsight, so the trigger is named in advance rather than recognised afterwards.

Until then, do not treat silence as stability. If you implement this now, pin the version you implemented and expect to move.

---

## Structure

A manifest has three top level members: `spec_version`, an **envelope** under `run`, and a list of **defects**.

```
{ "spec_version": ..., "run": { ... }, "defects": [ ... ] }
```

### Envelope, under `run`

| Field | Required | Meaning |
|---|---|---|
| `generated_at` | yes | timestamp of the run |
| `seed` | yes | the seed that produced this run |
| `locale` | yes | data locale, for example `pl-PL` |
| `catalog_version` | yes | version of the corruption catalog used |
| `recipe` | no | reference to the recipe, with its version |
| `policy` | no | reference to the corruption policy |
| `tag` | no | run tag, used later to clean up exactly what this run inserted |
| `rows_total` | yes | rows in the dataset |
| `rows_mutated` | yes | rows carrying at least one defect |
| `defects_requested` | yes | how many defects the policy asked for |
| `defects_applied` | yes | how many were actually applied |

**`defects_requested` and `defects_applied` are always both present.** A dataset where twenty seven of forty requested defects were applied looks exactly like a complete one, and it will be trusted as a complete one.

**`locale` is required even though it may look redundant.** Without it nobody can reconstruct why certain mutators are absent, because availability at layers `L3` and `L4` depends on the provider. See the [corruption catalog](corruption-catalog.md).

### Defect entry

| Field | Required | Meaning |
|---|---|---|
| `id` | yes | stable within the manifest, so a test can refer to one case |
| `location` | yes | where the defect is, see below |
| `mutator` | yes | identifier from the corruption catalog |
| `layer` | yes | copied from the catalog entry, so the manifest is readable on its own |
| `applied` | yes | whether the mutator actually ran |
| `value_before` | no | present when the mutator is reversible and `applied` is true |
| `value_after` | no | present when `applied` is true |
| `expectation` | yes | `rejection`, `acceptance` or `warning` |
| `confidence` | yes | `hard` or `soft` |
| `rationale` | yes | one sentence in plain language, written to be pasted into a test report |
| `failure_reason` | no | required when `applied` is false |
| `profile` | no | expected reaction of one specific application, see below |

---

## Location

Location differs by layer, because the layers damage different things.

| Member | Type | Used at |
|---|---|---|
| `entity` | string | `L2` to `L5`, when the dataset has named entities |
| `row` | integer, 1 based | `L1` to `L5` |
| `field` | string | `L2` to `L5` |
| `path` | string | nested structures, when `field` is not enough |
| `byte_offset` | integer | `L0` |
| `related` | array of locations | `L5` |

Requirements per layer:

- **`L0`** carries `byte_offset`, or nothing at all when the whole file is affected.
- **`L1`** carries `row` or `path`.
- **`L2` to `L4`** carry `entity`, `row` and `field`, or `path`.
- **`L5`** carries a primary location plus `related`, because set integrity defects span more than one record. A duplicate primary key involves two rows and naming only one of them is useless to whoever has to look at it.

---

## A defect that could not be applied is still an entry

**Silence is forbidden.** If the policy asked for a mutator that cannot be used in a given place, the manifest says so. It does not simply omit it.

This happens more often than it sounds. The field has no check digit to break. The format does not know that layer. No row matched the condition. **No mutator exists for that layer in the active locale.**

Such an entry carries `applied: false`, the intended location, and a `failure_reason`. The envelope counters make the total visible at a glance.

---

## Confidence, and why it is mandatory

- **`hard`** means a specification, an algorithm or a type is violated. The application must react. No reaction is a defect in the application.
- **`soft`** means the correct reaction depends on the business rules of the system under test. No reaction is not necessarily a defect.

Almost all of `L0` to `L2` is hard. Most of `L3` is hard, except the parts that depend on reference data rather than an algorithm, such as whether a postal code exists at all. Most of `L4` is soft, because business rules belong to the application. `REV` sits on both sides: a plus addressed email is hard because it conforms to the addressing specification, while a very long surname is soft because the upper limit is a product decision.

In practice a team runs hard expectations first and enables soft ones once it has filled in an application profile.

---

## Application profile

The manifest core describes two levels: the fact of the defect, and the expected class of reaction. A third level is optional and **does not come from the tool**.

An application profile is a file owned by the team. It maps mutators to what one specific system is expected to do: an error code, an HTTP status, which field the message should point at. When a profile is supplied, entries carry a `profile` object with those values.

The reason for the split is that layers `L0` to `L3` are objective while `L4` is not. A discount over one hundred percent is a validation error in one application and a legitimate correction in another. Keeping that knowledge in a team owned file leaves the core universal and puts the specifics where they belong.

A side effect worth having: **the profile becomes a readable record of how the system reacts to bad data**, which is documentation nobody usually writes.

---

## Example

The identifiers below are synthetic and describe nobody. Several of them
satisfy the arithmetic of their scheme, because an entry about a pair being
wrong has to show two values that are each right on their own, and a national
identifier has no block that is never issued, so a value that satisfies the
arithmetic cannot be shown not to belong to somebody. Where one resembles a
real person that is a coincidence of arithmetic rather than a record of
anything. The values are computed rather than invented, and a test recomputes
them, so an implementation reading this example gets a pair that behaves the
way the entry says it does.

```json
{
  "spec_version": "0.1",
  "run": {
    "generated_at": "2026-08-01T10:14:22Z",
    "seed": 20260801,
    "locale": "pl-PL",
    "catalog_version": "0.1",
    "recipe": "shop-basic@1",
    "tag": "run-1042",
    "rows_total": 13,
    "rows_mutated": 5,
    "defects_requested": 6,
    "defects_applied": 5
  },
  "defects": [
    {
      "id": "d1",
      "location": { "entity": "customer", "row": 1, "field": "national_id" },
      "mutator": "L3.checksum.off_by_one",
      "layer": "L3",
      "applied": true,
      "value_before": "85011412345",
      "value_after": "85011412346",
      "expectation": "rejection",
      "confidence": "hard",
      "rationale": "The format is still valid and the check digit no longer matches.",
      "profile": { "code": "ERR_NATIONAL_ID_INVALID", "http_status": 422, "field": "national_id" }
    },
    {
      "id": "d3",
      "location": { "entity": "customer", "row": 3, "field": "national_id" },
      "mutator": "L4.coherence.national_id_vs_birthdate",
      "layer": "L4",
      "applied": true,
      "value_before": "08280298771",
      "value_after": "85011467895",
      "expectation": "rejection",
      "confidence": "soft",
      "rationale": "The identifier is valid on its own but encodes 1985 while the birth date says 2008."
    },
    {
      "id": "d4",
      "location": { "entity": "customer", "row": 4, "field": "email" },
      "mutator": "REV.email.plus_addressing",
      "layer": "REV",
      "applied": true,
      "value_before": "maria.w@example.test",
      "value_after": "maria+shop@example.test",
      "expectation": "acceptance",
      "confidence": "hard",
      "rationale": "The address conforms to the addressing specification. Rejecting it blocks a legitimate user."
    },
    {
      "id": "d6",
      "location": { "entity": "customer", "row": 2, "field": "national_id" },
      "mutator": "L3.range.unassigned",
      "layer": "L3",
      "applied": false,
      "expectation": "rejection",
      "confidence": "hard",
      "rationale": "Not applied.",
      "failure_reason": "No unassigned range is defined for this identifier in locale pl-PL."
    }
  ]
}
```

Entry `d6` is the whole point of the previous section. It did not happen, and the manifest says so instead of leaving a hole.

---

## Exports

The canonical form is JSON. Other forms are derived from it and carry no information the JSON does not.

| Form | Purpose |
|---|---|
| JSON | canonical, for machines |
| CSV | for a spreadsheet and for review by a manual tester |
| Markdown | a readable table for a bug report |
| Gherkin `Examples` | pasted straight into an existing scenario |
| parameterised cases | for pytest, JUnit, NUnit |

### Gherkin mapping

| Column | Source |
|---|---|
| `row` | `location.row` |
| `field` | `location.field` |
| `mutator` | `mutator` |
| `expectation` | `expectation` |

Entries with `applied: false` are excluded from this export. There is nothing in the dataset for a test to send.

The Gherkin and parameterised forms are what decide whether this format is useful at all. They close the gap between having corrupted data and still having to write the assertions by hand.

---

## Verification

A tool reading a manifest compares each entry against what the application actually did. Three kinds of mismatch must be reported separately, because they mean different things.

- **Accepted although it should have been rejected.** A hole in validation.
- **Rejected although it should have been accepted.** Overzealous validation blocking legitimate users. This is what the `REV` module exists to find, and almost nobody tests for it.
- **The dataset arrived incomplete.** The number of rows the application took does not match `rows_total`. The row was neither accepted nor rejected, it disappeared on the way.

The first two assume the application reacted to the row somehow. The third is the case where it did not react at all, because it never saw the row. A receiver with a hard row limit that silently drops everything past it produces exactly this, and no per-row check finds it.

All three are failures. Which one it was belongs in the report rather than in the exit status, because continuous integration needs one bit and a human needs the detail.

---

## Versioning and compatibility

- The format evolves by **adding** fields. The meaning of an existing field never changes.
- **Mutator identifiers are permanent.** Manifests live for years inside other people's test repositories.
- **Consumers must ignore unknown fields** rather than failing on them, so that a manifest from a newer producer stays readable.
- A consumer that meets an unknown mutator identifier reports it as unknown instead of guessing.
- A tool must be able to read an older manifest and say what is missing from it.
