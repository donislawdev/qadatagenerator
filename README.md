# QA Data Generator

A **QA test data generator** with a CLI and a GUI. Cross-platform, fully offline, open source.

> A test data generator that understands **what the data means**, and can corrupt it on purpose while telling you exactly what it broke, at which layer, and how your application is supposed to respond.

**Status: early development.** The specifications below are drafts. The tool reads and checks recipes today with `qadg validate`, and it does not generate data yet.

---

## Two pillars

### Valid data

Records that are internally consistent and pass a real application's validation on the first try. The Polish national ID encodes the same date of birth and gender as the neighbouring fields. The postal code matches the city. Invoice line items add up to the declared total. Order dates run in a possible order.

Identifiers carry correct check digits for Poland and the United States, so the data survives contact with a domain validator instead of failing on the first screen.

### Invalid data

Controlled corruption across six layers, L0 to L5. The layer tells you **which component of your application is supposed to catch the defect**, so you are not choosing "break something at random" - you are choosing which line of defence you are testing.

| Layer | What gets broken | What should catch it |
|---|---|---|
| L0 | bytes and encoding | the parser |
| L1 | container structure | the deserializer |
| L2 | field type and format | the schema validator |
| L3 | domain semantics | the domain validator |
| L4 | business rules | application logic |
| L5 | set integrity | the database |

The difference between L3 and L4 is the one worth internalising. At L3 the national ID has a broken check digit, which one field reveals on its own. At L4 the ID is perfectly valid but encodes a different year than the date of birth field next to it, which takes two fields to see. Applications almost always catch the first and almost never the second.

That claim has an edge worth stating. The manifest says what your application should **do with a record**, which is accept it, reject it or warn about it. It does not say what value your application should **compute from it**. A defect where good data goes in and a wrong number comes out sits outside what the manifest can express today, so the L4 claim above is about validation and not about calculation.

### The expectation manifest

Every corrupted dataset ships with a second file. It records where each defect is, which mutator produced it, which layer it belongs to, and **what the application is expected to do about it**.

Without it, corrupted data is useless to an automated test, because the test has no way to know what to expect. This is why lists of hostile strings never made it into anyone's CI pipeline. They supply the input and leave out the answer.

The manifest exports as JSON, CSV, Markdown and as a Gherkin `Examples` table, so an existing scenario can consume it without hand-written assertions.

### Reverse mutators

The inverse of the rest of the catalog. They supply data that is **entirely valid** but tends to be rejected by overzealous validation: plus addressing in an email, an apostrophe in a surname, a single-character given name, the surname Null. Not only text about people. A negative amount is legal in plenty of domains, and a national identifier can stop carrying a check digit because the issuing authority decided so.

The expected outcome here is acceptance, not rejection. A false rejection costs more than a false acceptance, because it turns away a customer who had every right to sign up, and almost nobody tests for it.

**Each pillar is useful without the other.**

---

## Specifications

The `spec/` directory holds public specifications meant to be implemented by other tools as well.

| Specification | Status |
|---|---|
| [Corruption catalog](spec/corruption-catalog.md) | 0.1, draft |
| [Catalog entries](spec/corruption-catalog.yaml) | 0.1, draft, canonical form |
| [Expectation manifest](spec/expectation-manifest.md) | 0.1, draft |

The catalog is split on purpose. The Markdown describes the shape of an entry and the rules the catalog grows by. The YAML carries the entries themselves and is the form a tool reads.

Both are versioned and documented so that other tools can implement them. Neither is a standard - a format one tool emits is that tool's format until somebody else adopts it.

Identifiers become permanent at the first public release. Until then they can still change.

Product documentation is internal and is not part of this repository. What is public is exactly what other tools need to read and implement.

---

## Language

The CLI is **English only**, regardless of which country the data is generated for. The locale is a property of the **data**, not of the interface. `--locale pl-PL` decides what gets generated. Commands, flags, error messages and exit codes stay in English, because they end up in CI scripts and bug reports.

The GUI is translated. Adding a language means adding a translation file, not touching the code.

---

## Decisions

| Decision | Choice |
|---|---|
| MVP scope | Two equal pillars: valid data and invalid data |
| Differentiator | The invalid data engine with its expectation manifest |
| Reach | Global in the long run, Poland and the United States first |
| Surfaces | CLI and GUI at feature parity, one engine |
| Model | Fully open source, no backend |

---

## Principles

1. **CLI and GUI parity.** One engine, one recipe file, two surfaces.
2. **Recipe as code.** Configuration lives in your repository next to your tests, versioned and reviewed in pull requests.
3. **Determinism.** The same seed produces the same output, including which rows were corrupted and how.
4. **Everything local.** No server, no accounts, no telemetry. Data never leaves the machine.
5. **Value in five seconds.** Getting a single value must be faster than writing a recipe.
6. **Open formats.** The manifest and the recipe are public specifications that other tools can read.
7. **Synthetic data must look synthetic.** Reserved ranges wherever they exist, `.test` domains, official test card numbers only, SPECIMEN watermarks on generated documents.

### One limitation this creates, stated up front

Generating inside reserved ranges is what keeps a synthetic identifier from belonging to a real person, and it is not optional here. **The cost is that an application which validates identifier ranges will reject the generated valid data too, not only the deliberately corrupted rows.** This applies to every identifier that has a reserved range, the US social security number among them.

You will meet this as an unexpected rejection on data that should have passed. It is a consequence of the ethics rule rather than a defect, and the tool says so where it applies.

---

## License

**Two licenses, split by what the file is.**

| What | License |
|---|---|
| Code | GNU GPL v3.0, full text in [LICENSE](LICENSE) |
| `spec/`, including the catalog entries | CC BY 4.0, full text in [LICENSE-spec](LICENSE-spec) |

**The code is GPL-3.0 and that covers the code only.** Data, files and manifests you generate with this tool are **yours**. They are not derivative works, they are not covered by this license, and there is nothing to attribute. Use them in closed-source products, in commercial test suites, anywhere.

If your legal department needs one sentence: running a GPL tool over your own inputs does not place any obligation on the output.

**The specifications in `spec/` are CC BY 4.0**, the catalog entries in `spec/corruption-catalog.yaml` included. They exist to be implemented by other tools, so they carry a license that says so out loud. CC BY 4.0 also licenses the sui generis database right that a collection like the catalog attracts in the European Union, which matters because that right is separate from copyright and a copyright notice on its own does not reach it. Attribution is the only condition.

`--license` prints both licenses from the binary, along with every third-party component, its version, its license and where it came from. The GUI shows the same on its own screen.

---

## What this tool does not do

It does not produce binary artifacts as an end in themselves - images of a given size, deliberately damaged archives, files whose names break the filesystem. That is the scope of a separate tool, **Testing Files Generator**.

The boundary is sharp, and it does not run along the file. Both tools can write a file with a given number of records, so that tells you nothing. **There the artifact is the product. Here the product is a dataset together with a declared expectation of what should happen to it.**
