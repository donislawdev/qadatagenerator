# Application response record

Version 0.2, draft. Licensed CC BY 4.0, like the rest of `spec/`.

This is the one format in this project that the tool only ever **reads**. The expectation manifest says what should happen to each record of a dataset. This says what actually happened, and `qadg verify` compares the two.

You produce this file. Nothing here requires a particular framework, database or transport: it is a plain record of what your system did with the rows it was given.

---

## Why a file, and why the tool does not fetch it

The tool makes no outgoing connections at all, so it cannot ask your application anything. A mode that called an endpoint would have to be switched on deliberately and is not part of this format.

That has a consequence worth stating plainly: **something has to write this file, and that something is yours.** A wrapper around your import, a few lines after your test run, a query against the table afterwards. The format is small on purpose so that writing it is not a project.

## The shape

```json
{
  "spec_version": "0.2",
  "dataset": { "rows_total": 60 },
  "rows": [
    { "row": 1, "verdict": "rejected", "code": "ERR_CHECKSUM",
      "message": "national identifier failed its check digit" },
    { "row": 2, "verdict": "accepted",
      "stored": { "pesel": "02070803628", "amount": "0" },
      "warnings": ["amount was not a number and was stored as zero"] },
    { "row": 3, "verdict": "accepted" }
  ]
}
```

| Field | Required | Meaning |
|---|---|---|
| `spec_version` | yes | the version of this format the file is written to |
| `dataset` | no | what the producer believes it sent, for a cross check |
| `dataset.rows_total` | no | how many rows were sent |
| `rows` | yes | one entry per row the application reacted to, and an empty list when it reacted to none |
| `rows[].entity` | no, from 0.2 | the entity the row belongs to, named as the manifest names it. May be left out when the dataset has one entity |
| `rows[].row` | yes | the row number within its entity, counting the first record after the header as one |
| `rows[].verdict` | yes | `accepted` or `rejected` |
| `rows[].code` | no | your application's own code for what happened |
| `rows[].message` | no | your application's own words |
| `rows[].stored` | no | what the application actually holds now, by column |
| `rows[].warnings` | no | anything the application said without refusing the row |

## One entry per row, not per defect

A row can carry more than one defect. Your application does not react to a defect, it reacts to a **record**, and asking which of three defects caused a rejection is a question nobody can answer, including the application. So the unit here is the row, and the tool maps rows onto manifest entries itself using the row number every entry carries.

## A dataset of several entities

A dataset made from a recipe can hold several tables, and each of them has a row 1. A row says which one it belongs to through `entity`, written exactly as the manifest lists it under `run.entities`.

```json
{
  "spec_version": "0.2",
  "rows": [
    { "entity": "customer", "row": 1, "verdict": "rejected", "code": "ERR_CHECKSUM" },
    { "entity": "order", "row": 1, "verdict": "accepted" }
  ]
}
```

- **One entity:** `entity` may be left out, and a row that leaves it out is a row of that entity. A row naming it is the same row, so naming it once and leaving it out once for the same number answers that row twice.
- **Several entities:** every row names its entity. A row that does not is refused, because comparing it would mean guessing which table it is in.
- **An entity the manifest does not list** is refused, and so is a row numbered past the end of its entity. Past the end is usually every row shifted by one because the header was counted as a record, and a comparison of shifted rows is a verdict about the wrong records.

A record refused for any of these is refused whole, with every reason at once, and nothing is compared.

## Absence means the application never reacted

A row of the manifest with no entry in `rows` is reported as **the dataset arrived incomplete**: the row was neither accepted nor rejected, it disappeared on the way. A receiver with a hard row limit that silently drops everything past it produces exactly this, and no per-row check finds it.

What is missing is counted for each entity on its own, from the rows of that entity the record answers for. A row of one entity never makes up for a row of another.

This is why omitting rows you did see is a misreport rather than a shortcut. If your application reacted to a row, say so.

An empty `rows` says the application reacted to none of the rows it was sent, so every row of the dataset is reported as never reacted to. For a dataset of no rows it is the only true answer, and it agrees with the manifest. A record that leaves `rows` out, or writes it as `null`, is refused. It does not say what happened, and reading it as every row going missing would blame the application for a producer that wrote nothing.

## 🔴 `verdict: accepted` is not the same as the value surviving

An application that stores something other than what it was sent did not accept the value, it accepted a **different** value. This is measured rather than imagined: a widely deployed database configuration takes a row, turns `not a number` into `0`, cuts an overlong string down to the column width, and reports no error at all. Warnings only.

That is why `stored` exists. When it is present the tool can tell a value that survived from one that was quietly changed, which is a defect class that no status code reveals.

**`stored` is optional, and its absence is not free.** Without it a run cannot tell those two apart, so the report says which entries it could not settle and why. A tool that stayed silent about that would be reporting a clean result it has no grounds for.

The same applies to `warnings`. A manifest entry expecting the application to accept a value **and mark it** cannot be settled by a file that never says whether anything was marked.

## What the tool reports

Four kinds of mismatch, kept apart because they mean different things.

- **Accepted although it should have been rejected.** A hole in validation.
- **Rejected although it should have been accepted.** Overzealous validation blocking legitimate users.
- **Expected a mark, got silence.** The application took the value and said nothing, where the manifest expected it to complain.
- **The dataset arrived incomplete.** Rows the application never reacted to.

Mismatches on entries whose expectation is **soft** are reported and do not, on their own, make the run fail. A soft expectation means the correct reaction depends on your application's own rules, so a failing exit status on one would be this tool asserting something it was never entitled to assert.

## Producing the file

The shortest honest version is one entry per row with a verdict, which every import loop already knows. Add `stored` when you can read back what landed, and `warnings` when your application has them. Each field you add widens what the comparison can tell you, and the report always says which questions it could not answer with what it was given.

## Versioning and compatibility

`spec_version` is required, and a reader is expected to act on what it says rather than only on whether it is there.

- The format evolves by **adding** fields. The meaning of an existing field never changes.
- **A reader ignores fields it does not know in a record written to a newer minor version**, so that a record from a newer producer stays readable.
- **A reader may refuse a field it does not know in a record written to a version it speaks, or to an earlier one.** There every field has a name the reader knows, so an unknown one is a misspelling, and skipping it would mean comparing against evidence the producer believes they supplied. The reference implementation refuses it.
- **An unknown MINOR version is read, and the reader says it did.** Whatever the newer version added is invisible to an older reader, and a comparison that did not mention this would be reporting on evidence it only partly understands.
- **Version 0.2 added `rows[].entity`.** A record written to 0.1 names no entity, which a dataset of one entity allows, so it is read by the same rules.
- 🔴 **An unknown MAJOR version is refused.** A major version is the one where the meaning of an existing field changes. Reading this file is how a tool decides whether an application behaved correctly, so a verdict drawn from a record the reader cannot claim to understand is worse than no verdict.
- A reader that refuses a version says which one it speaks, because otherwise the producer cannot act on the refusal.
