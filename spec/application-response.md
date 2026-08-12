# Application response record

Version 0.1, draft. Licensed CC BY 4.0, like the rest of `spec/`.

This is the one format in this project that the tool only ever **reads**. The expectation manifest says what should happen to each record of a dataset. This says what actually happened, and `qadg verify` compares the two.

You produce this file. Nothing here requires a particular framework, database or transport: it is a plain record of what your system did with the rows it was given.

---

## Why a file, and why the tool does not fetch it

The tool makes no outgoing connections at all, so it cannot ask your application anything. A mode that called an endpoint would have to be switched on deliberately and is not part of this format.

That has a consequence worth stating plainly: **something has to write this file, and that something is yours.** A wrapper around your import, a few lines after your test run, a query against the table afterwards. The format is small on purpose so that writing it is not a project.

## The shape

```json
{
  "spec_version": "0.1",
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
| `rows` | yes | one entry per row the application reacted to |
| `rows[].row` | yes | the row number, counting the first record after the header as one |
| `rows[].verdict` | yes | `accepted` or `rejected` |
| `rows[].code` | no | your application's own code for what happened |
| `rows[].message` | no | your application's own words |
| `rows[].stored` | no | what the application actually holds now, by column |
| `rows[].warnings` | no | anything the application said without refusing the row |

## One entry per row, not per defect

A row can carry more than one defect. Your application does not react to a defect, it reacts to a **record**, and asking which of three defects caused a rejection is a question nobody can answer, including the application. So the unit here is the row, and the tool maps rows onto manifest entries itself using the row number every entry carries.

## Absence means the application never reacted

A row of the manifest with no entry in `rows` is reported as **the dataset arrived incomplete**: the row was neither accepted nor rejected, it disappeared on the way. A receiver with a hard row limit that silently drops everything past it produces exactly this, and no per-row check finds it.

This is why omitting rows you did see is a misreport rather than a shortcut. If your application reacted to a row, say so.

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
