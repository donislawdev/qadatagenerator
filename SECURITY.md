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
- Generated identifiers that could belong to a real person, where the scheme
  has a range set aside and the tool used a different one.

### What is deliberately not protected

Listing this is part of the answer rather than an omission, and the reasoning
lives with the project's own threat model.

- **A recipe is trusted to the extent that whoever runs the tool trusts it.**
  It is read as data and never executed, and it is size limited so that a
  hostile one cannot hang a build for minutes. Beyond that, a recipe with
  absurd but legal contents is the author's problem, in the same way a build
  script is.
- **The tool is not a security boundary between the person running it and the
  data they generate.** It generates synthetic data on request. It does not
  decide who may run it.
- **Generated data is not secret.** It is meant to be committed, shared and
  put into test systems.
