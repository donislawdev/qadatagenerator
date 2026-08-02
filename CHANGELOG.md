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

- **Polish tax numbers can now be generated and checked.** A generated number
  is ten digits with a correct check digit, and the checker accepts the three
  ways Poland writes one: bare, and either of the two hyphenated forms. A value
  handed in with hyphens comes back with them.
  Two limits are worth knowing before you rely on this. The scheme sets aside
  no block of numbers for documentation or testing, so a generated tax number
  could be some real company's, exactly as with the personal number. Mark
  datasets carrying either as synthetic. And because the scheme divides by
  eleven, some ten digit values can never be tax numbers whatever check digit
  is written on the end. Asking for such a value to be made consistent gets a
  refusal that says so, rather than a number quietly changed somewhere else.
- **A recipe that nests square brackets more than sixty four deep is refused
  before it is read, with the configuration exit code and a message naming the
  limit.** The size limit alone did not catch this. Memory grows with the
  square of that nesting, so a sixteen kilobyte recipe, well inside the size
  limit, cost four hundred megabytes, and one at the limit cost tens of
  gigabytes. A recipe arrives through a pull request, so that was a build
  machine killed by the operating system rather than a build that failed and
  said why. Nothing written by hand nests anywhere near this deep. The
  format's own deepest use is a list of field names, which is one level.
- Ordinary indented nesting and flow mappings are unaffected and have no
  limit, because the same measurement shows both are cheap.

### Fixed

- **A file holding two recipes is refused instead of being read halfway.** A
  file declaring two entities, separated by the usual `---`, reported one
  entity, no problems and success. The rest was dropped without a word.
  Layouts that are still one recipe keep working: a leading separator, a
  leading comment before one, a trailing separator and a trailing comment.
- **A recipe saved by Notepad is read.** Notepad writes a byte order mark by
  default, and the mark used to become part of the first key, so the recipe was
  refused for not declaring a schema version. The hint told you to add a line
  that was already the first line of your file, and the message naming the real
  cause was unreadable because the mark does not render. Only a leading mark is
  removed. Further into a file it is ordinary text that a value keeps.

- **A recipe can no longer paint its own text into this tool's messages.** A
  key named with an escape sequence put those bytes straight onto the log
  channel, where a terminal acts on them: the sequence cleared the line and
  recoloured the text, so a refusal could be made to read on screen as an
  approval. Neither `--no-color` nor `NO_COLOR` helped, because both govern the
  colour this tool adds and neither has anything to do with what a file
  carries. Control characters on the log channel are now replaced by a visible
  marker, so a name that had one still looks like a name that had one.
- **The data channel is deliberately not cleaned, and there is a test saying
  so.** Corruptions of the lowest layer exist to inject control characters, so
  cleaning what the tool produces would break what it is for. The machine
  readable report was already safe, since JSON carries the escape rather than
  the byte.

- **An oversized recipe is no longer read into memory in order to be refused
  for being too large.** Reading stops one byte past the limit. Measured
  before: a 250 MB file cost 258 MB of memory on its way to being told it was
  over a 256 kB limit. Measured after: 3 to 7 MB. The refusal no longer quotes
  the file's size, since knowing it would mean holding the file.
- **An error message no longer grows with the recipe that caused it.** A
  recipe carrying one long value produced a message the same size as the
  value, which went into the build log and, with `--json`, onto standard
  output. Every part of a reported problem is now bounded, and the cut lands
  on a character boundary rather than a byte, so a report stays valid UTF-8 for
  every alphabet this tool generates.

- The second country provider, the United States Social Security number. It
  produces numbers, says whether a number is well formed and why not, and says
  whether a number is one the Social Security Administration would never have
  issued.
- Every number it produces sits in area 666, which the Administration has
  never assigned and has excluded from assignment, so no generated number can
  be a real person's. That is a property of the provider rather than a flag to
  remember. **It has a price worth knowing before you meet it as a bug: an
  application that validates assignment ranges will reject this tool's correct
  data as well as its corrupted data.**
- The rules were read at the Administration rather than recalled, and they
  draw a line this tool now follows. Group 00 and serial 0000 are called
  invalid outright. The area numbers are only excluded from assignment, which
  is a weaker statement, and a number in area 666 therefore looks entirely
  ordinary to any format check. That gap is what makes generated data safe and
  still plausible.
- **A Social Security number has no check digit, and encodes no date and no
  gender.** Nothing in it verifies anything else in it, so a wrong digit
  produces another well formed number. The corruptions that break a check
  digit or move an encoded date do not apply to this identifier and say so
  rather than appearing to work.
- Numbers are written 123-45-6789. A number written as nine bare digits is
  read as the same number, because it is, and a corruption hands back the way
  of writing it was given rather than reformatting on the way through.
- A recipe larger than 256 kB is refused before it is read, with the
  configuration exit code and a message naming the limit. Reading a
  deliberately nested document costs time growing with the square of its size,
  and a recipe arriving through a pull request could otherwise hang a build for
  minutes. The documented example is about one and a half kilobytes, so the
  limit leaves room for one a hundred and seventy times larger.
- A SECURITY.md saying how to report a vulnerability privately, which versions
  are supported, what to expect from a project maintained by one person, and
  what is deliberately not protected.
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
