# Changelog

Everything a user would notice is recorded here. The technical counterpart is
in CHANGELOG-DEV.md.

The format follows Keep a Changelog and the versions follow Semantic
Versioning. A change that breaks a public contract gets a BREAKING section
first, because this tool has an unusual number of them: mutator identifiers,
manifest fields, recipe keys and exit codes all live in somebody else's
repository once they are released.

## [Unreleased]

Nothing has been released yet. The binary reports 0.1.0, which is the version
being built rather than one anybody can download. Releasing is a separate act
and has not happened.

### Added

- The first country provider, the Polish national identification number. It
  produces numbers that agree with the date of birth and the gender in the
  same record, reads both back out of a number, and says whether a number is
  correct and why not.
- The rules behind it were read from the government description of the number
  rather than recalled, including the table that encodes the century in the
  month field. One limitation is worth knowing because it is a property of the
  scheme and not of this tool: **the check digit does not catch every swap of
  two neighbouring digits.** Some transpositions produce a different number
  that is still valid, which is exactly why the corruption catalog carries a
  transposition entry of its own.
- Numbers carrying a Polish identifier have no reserved range to be generated
  in. No block of numbers is set aside for documentation, so a dataset with
  these has to be treated as synthetic on the strength of where it came from
  rather than what it looks like.
- Recipes can declare three things they could not before: which fields form
  the primary key of an entity, that a field is required, and how long a text
  field is allowed to be. All three are optional, so recipes written before
  this still read.
- A maximum length is declared together with the unit it is counted in, one of
  `characters`, `bytes` or `graphemes`, defaulting to characters. The unit is
  the whole difference between a limit an application enforces and one it does
  not: a column of thirty characters holds thirty Polish names and ten emoji.
- `required: true` and `empty: 15%` on the same field are refused as the
  contradiction they are. A field cannot both always carry a value and be
  blank in some rows, and picking one silently would be the tool deciding what
  you meant. `required: true` with `empty: 0%` is accepted, because saying no
  row happens to be blank is not the same statement as saying none may be.
- The `qadg` command line, so far answering `--help`, `--version` and
  `--license` only. No command is implemented yet.
- `--version` reports four numbers at once: the tool, the corruption catalog,
  the manifest format and the recipe schema. They move independently, so a
  recipe that gives a different result later can be traced to whichever of the
  four changed. `--json` gives the same answer in machine readable form.
- `--license` names both licenses of the project, lists the third party
  components compiled into the binary and states that data generated with the
  tool belongs to whoever generated it.
- The table of exit codes, printed by `--help`. A pipeline can read it without
  leaving the terminal.
- Data goes to standard output and the log goes to standard error. A run that
  fails writes nothing to standard output, so a redirected file is either
  complete or empty and never half a dataset.
- Color and progress detail are off automatically when the output is not a
  terminal. `--no-color` and the NO_COLOR variable turn them off anyway, which
  matters on the systems that pretend to be a terminal.
- `--quiet` prints failures only. Warnings survive it, because a warning that
  a flag can hide is a warning that never happened.
- The full text of both licenses, as LICENSE for the code and LICENSE-spec for
  the specifications. The README links to both.
- Asking for several reports at once prints all of them. `--help --version`
  gives the help and the versions rather than quietly dropping one.
- Reference word lists for Polish and American data are built into the binary:
  first names by gender, surnames, cities and regions. They were read from an
  open source project for fake data under its licence, and `--license` names
  that project, its version and every list taken from it. Nothing external is
  linked into the tool, so the lists cannot change under you when somebody
  else publishes a release.
- `qadg validate --json` prints the same findings as one machine readable
  document, so a pipeline can decide for itself which of them block a build.
  The document is printed whether or not the recipe turned out to be usable,
  because withholding it exactly when it is needed would make the flag
  pointless. Cross cutting flags work on either side of the command, so
  `qadg validate --json r.yaml` and `qadg --json validate r.yaml` are the same
  invocation.
- `qadg validate <recipe>` reads a recipe and says whether it can be used. It
  reports every problem it found in one pass rather than one per run, and it
  lists separately what it could not check, so a clean result never means more
  than it says. A recipe that cannot be used ends with the configuration exit
  code, which is different from the tool failing.
- Recipes are read with the seed and the schema version as text. Writing them
  without quotes is refused with a hint, because YAML reads 010 as the number
  8 and 1.0 as 1, and a seed that changes on the way in reproduces nothing.
- The mechanism behind the promise that a seed reproduces a dataset exactly.
  A value depends on where it sits and on nothing else, so the order in which
  values are produced does not matter, adding a field leaves the other fields
  untouched, and the same seed gives the same bytes on every machine. The seed
  is text rather than a number, because a recipe is YAML and YAML reads 010 as
  the number 8.
