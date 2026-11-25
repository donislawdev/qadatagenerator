# Security

## Reporting a vulnerability

Please report privately, through GitHub's private vulnerability reporting on
this repository: open the Security tab and choose to report a vulnerability.
That channel is visible only to the maintainer until a fix is published.

**Please do not open a public issue for a vulnerability.** A public issue tells
everybody at once, including before there is anything to update to.

## What is supported

Only the latest release. Nothing has been released yet, so at present that is
the main branch.

## What to expect

This project is maintained by one person. A first reply usually takes a few
days rather than a few hours, and saying so here is more useful than a promise
that would not be kept. You will be told whether the report is accepted, and
what the fix or the reason for declining is.

## What counts as a vulnerability here

The tool runs locally, writes files, and reads a recipe that describes what to
generate. It has no server, no accounts and makes no outbound connections, so
most of the usual categories do not apply. These do:

- Reading or writing outside the output directory the run was given.
- Executing anything named in a recipe, or interpreting part of one as code.
- Any outbound network connection. There should be none at all.
- A crash or a hang that a recipe can cause and a person cannot see coming.
- Generated identifiers that could belong to a real subject, **where the scheme
  has a block that is never issued and the tool generated outside it**. See the
  next section for which schemes have such a block, because not all of them do.

### Which numbering schemes have a safe block, and which have none

This decides whether a generated identifier resembling a real one is a defect or
a documented property, so it is stated here rather than left to be inferred.

A **safe block** is a range a registrar says nothing will ever be issued from.
Where one exists, correct values are generated inside it by default, and the tool
can say its own output could not belong to anybody.

| Scheme | Safe block | What that means |
|---|---|---|
| `US.ssn` | yes, the area the administration never assigns | a value outside it would be a defect worth reporting |
| `ean` | yes, the prefix 952, which GS1 sets aside for demonstrations and examples of its own system | a value outside it would be a defect worth reporting. What has not been established is whether a company prefix was issued there before GS1 carved it out, so a resemblance to a real article number is possible and is worth telling us about |
| `PL.pesel`, `PL.nip`, `PL.regon`, `US.aba`, `iban`, `vin`, `imei`, `isbn`, `card` | **none** | the registrar issues, or reserves the right to issue, across the whole space |
| `uuid` | **none, and none is needed** | nobody issues UUIDs. Every one is version 4, 122 random bits, so matching an existing UUID is as unlikely as for any other random UUID, and a UUID names no person |

`imei`, `isbn` and `card` are the three worth a sentence of their own, because
each narrows what it writes without claiming anything about it.

Every `card` is one of eight numbers that at least two of four payment
providers publish for testing, so this tool never makes up a number that
merely passes the check digit. That is not a safe block, because a test number
works in its provider's test system and none of the four says that no issuer
has given the same number to a real card.

Every `imei` is generated inside the block the GSM Association gives to test
equipment, which cannot be supplied to the market, so nothing this tool writes
can be a device somebody owns. That is deliberately not called a safe block,
because codes inside it are issued to named manufacturers, so a value there can
still resemble a real prototype.

Every `isbn` is generated inside the one window under prefix 978 that the
International ISBN Agency leaves without a registration group at all, so a value
there cannot be split into a group, a registrant and a publication and no book
carries it today. That is not a safe block either, because the agency allocates
into that space over time. Two registration groups did look like a safe block,
the ones its range file marks as reserved, and they were rejected: both had been
allocated to a country before being withdrawn, so numbers may have been issued
inside them.

For the schemes with none, a correct value **cannot be shown not to belong to
somebody**, and that is a property of the numbering scheme rather than a flaw in
this tool. Picking a range that merely looks free would be worse, not better:
one such range in another scheme turned out to be the space of a real taxpayer
identifier. The defence is that every value is labelled synthetic, which
`--license` states from the binary.

**So a resemblance in one of those schemes is not a vulnerability report.** A
value generated outside the safe block of a scheme that has one is.

### What is deliberately not protected

Listing this is part of the answer rather than an omission, and the reasoning
lives with the project's own threat model.

- **A recipe is trusted to the extent that whoever runs the tool trusts it.**
  It is read as data and never executed, and it is limited twice over: in size,
  so that a hostile one cannot hang a build, and in how deeply it may nest, so
  that one cannot exhaust the machine's memory instead. Both limits are set
  from measurements rather than from caution. Beyond them, a recipe with absurd
  but legal contents is the author's problem, in the same way a build script
  is.
- **The tool is not a security boundary between the person running it and the
  data they generate.** It generates synthetic data on request. It does not
  decide who may run it.
- **Generated data is not secret.** It is meant to be committed, shared and
  put into test systems.
