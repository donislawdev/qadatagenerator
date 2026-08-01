# Developer changelog

The technical record. What changed, and above all why, so that the reason
survives without reading the whole history. The user facing counterpart is in
CHANGELOG.md.

The format follows Keep a Changelog and the versions follow Semantic
Versioning.

## [Unreleased]

### Added

- **Module `github.com/donislawdev/qadatagenerator`, Go pinned to 1.26.5.**
  The module path has to equal the repository path, so it reaches every import
  line. The Go version is pinned rather than treated as a minimum, because
  byte for byte determinism makes the toolchain part of what has to be
  reproducible. A redundant toolchain line is dropped by the go command, so CI
  has to pin through GOTOOLCHAIN instead.
- **`internal/fault`: the failure kind travels with the error.** The exit code
  is a property of the failure, taken from where it happens. A switch over
  failures at the top of the program hands every new failure mode the default
  code 1 and breaks the promise that every ending has a code from the table,
  without anything visible going wrong.
  The zero value is `Unknown` on purpose, so a kind that was never set cannot
  pass for a real one. A failure that reaches the surface without a kind is
  reported as a defect of ours rather than given a number.
- **`internal/cli`: one table for the exit codes.** The help text prints it
  and the mapping from failure kind reads it. Nothing holds a second copy,
  because two copies are how a table and the behaviour beside it part company.
- **Channel split behind one type.** Every write goes through it. Calls to
  fmt.Fprintf spread through a program break the split at the first moment of
  inattention and nothing notices, because the output still looks right on a
  terminal.
- **A failed write to the data channel ends the run as an input output
  failure.** Output that went nowhere is not success, and a closed pipe is the
  ordinary way that happens.
- **`internal/version`: the four version numbers, with the specifications as
  their source.** Two of them are copies of numbers that live in `spec/`, and a
  copy with nothing watching it drifts, so a test reads both files and compares.
  The recipe schema has no specification file yet, so a test fails the day one
  appears rather than letting the constant quietly go stale.
- **`internal/license`: the registry behind `--license`.** A test compares it
  with the direct requirements in go.mod, so a dependency that nobody described
  fails the build instead of shipping unlisted. The list is empty today and
  that is a fact rather than a gap.
- **`internal/guard`: the architecture checks, imported by tests only.** They
  cover the ban on anything that can reach outside the machine or start another
  program, the dependency rule between the layers, and the one edge that would
  stop the command line binary from cross compiling, which is `cmd/qadg`
  reaching the window package through any number of steps.
  Each check is proved twice: against a graph built to break it, and against
  the real tree. A check that has only ever passed says nothing about either.

- **One list of flags.** The help text renders it and the failure that names
  what would be accepted reads the same list. Two lists answer the same
  question differently the first time somebody adds a flag and updates one.
- **LICENSE and LICENSE-spec, verbatim from the issuers.** Checked for size,
  line endings and characters outside ASCII before being committed to. The
  punctuation guard excludes them by a rule that names the exception, rather
  than by the accident of those files having no extension.

- **The recipe, in three layers.** `internal/domain/recipe` holds the model and
  the schema view, `internal/adapter/recipeyaml` is the only place that knows a
  recipe is YAML, and `internal/app/validate` is the use case behind the first
  command. The use case takes the decoder as a port, so it never learns where
  its input came from.
- **Field order is the order the user wrote.** The decoding keeps the mapping
  order rather than taking the convenient unordered form, because column order
  in the output follows it and a pull request diff has to stay small.
- **Everything wrong is reported in one pass.** Handing problems out one at a
  time turns a five minute fix into five rounds.
- **What could not be checked is reported too.** Generator settings cannot be
  verified before the generators exist, so they are carried and listed rather
  than accepted in silence. That also makes a misspelled generic key visible,
  which would otherwise land among the settings and turn itself off.
- **`github.com/goccy/go-yaml` v1.19.2, MIT.** The sieve was run rather than
  quoted: the license was read in the module and the module graph was checked
  for anything transitive, of which there is none.
- **`internal/domain/seed`: randomness derived from the address of a value.**
  SHA-256 over a length framed message built from the seed, the entity, the
  row, the field, the purpose and a round counter. The lengths are what make
  the message unambiguous: plain concatenation gives the same bytes for entity
  "ab" with field "c" and entity "a" with field "bc", and a collision that rare
  reads as coincidence rather than as a defect.
  No version of anything enters the message, so an upgrade moves nobody's
  bytes. Byte stability is therefore a promise held within a major version.
  `IntN` draws again instead of folding the remainder, because folding makes
  the low results more likely whenever the bound does not divide evenly. The
  lean never shows up as a failure, only as data tilted one way.
- **Three guards rather than one, and the reason is worth keeping.** A
  determinism test compares two runs of the same build, so it only ever finds
  randomness that crept in and stays green when the derivation changes
  completely. The golden values in testdata compare today with the past, and a
  locality test asks whether drawing other places moves this one. Until there
  is a build matrix, the golden file is also the whole of the promise that the
  derivation does not depend on the machine.
- **An oracle that does not trust the package.** One test builds the message by
  hand from the documented layout, and an internal script recomputes every
  golden value in another language on another SHA-256 implementation. Thirteen
  of thirteen agree. Without it the golden file would defend its own mistake.
- **The line ending protection now reaches every testdata directory.** Measured
  before changing it: `testdata/x` was protected and
  `internal/domain/seed/testdata/x` was not, because a pattern containing a
  slash is anchored where the attributes file lives. Go puts fixtures next to
  the package that owns them, so most of them would have been unprotected.

### Fixed

- **`--no-color` was ignored whenever flag parsing itself failed**, which is
  exactly when a failure has to be printed. A failed parse leaves every flag at
  its default, so the answer now comes from the raw arguments. Found by a test
  rather than by reading the code.
- **A crash left through Go's own exit code, which is 2, the number that means
  a wrong invocation here.** A pipeline would have read a defect of ours as a
  mistake of its own. Recovered at the top of the run and reported as a tool
  failure instead.
- **`--json` with more than one report would have put JSON and prose on the
  same channel.** It is now accepted only when `--version` is the only report
  asked for.
- **Asking for two reports printed one of them.** Every report that was asked
  for is printed now, in a fixed order.
- **A failure with an empty message reached the log as a prefix with nothing
  after it.** It names its own kind instead.
- **The rule that the command line binary never reaches the window package
  passed by looking for a package that could have been renamed.** The scan is
  now asserted to contain it, because a rule about something absent checks
  nothing and says nothing.
- **A rule that could not be read shifted the positions of the ones after it.**
  The place in a message was counted from the model, and a rule left out of the
  model moved every later position by one, sending somebody to the wrong line.
  The places are now kept beside the rules as they are read.
- **A cardinality that ran backwards hid a second problem on the same line.**
  The relation was dropped before anything asked whether the parent entity
  exists, so fixing the range revealed a new error instead of finishing.
- **The layer rule let a use case import an adapter.** It forbade the surfaces
  and not the adapters, which is two edges out of three, and the missing one is
  the easiest to cross by accident because the adapter is right there.
- **`StdoutIsTerminal` was computed and never read.** Data carries no escape
  sequences at all, so the field had no consumer. Removed rather than kept for
  a use that may never come.

### Notes

- No signal handler yet. The codes 130 and 143 are in the table and a shell
  reports them on its own for a process that does not catch the signal. A
  handler earns its place with the first command that writes a file, because
  then there is a temporary file to remove.
- Test coverage is not gated yet. When it is, `-coverpkg` has to be set at the
  same time, because the guards live in their own package and a gate that
  measures the wrong set of packages reads zero and looks like it works.
