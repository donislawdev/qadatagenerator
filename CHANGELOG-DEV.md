# Developer changelog

The technical record. What changed, and above all why, so that the reason
survives without reading the whole history. The user facing counterpart is in
CHANGELOG.md.

The format follows Keep a Changelog and the versions follow Semantic
Versioning.

## [Unreleased]

### Added

- **A guard that closes concurrency until somebody opens it deliberately.**
  The architecture already said the output is assembled by row number rather
  than by whichever goroutine finished first, and that nothing mutable is
  shared. Nothing enforced either sentence. There is no concurrency in the
  product today, so the list of files allowed to have it is empty, which makes
  this the cheapest moment it will ever be: everything the guard will ever
  report is somebody adding the first goroutine.
  It reads the syntax tree rather than the text, so the word chan in a comment
  and the phrase go func in a message are not findings. They both appear in the
  comments explaining the rule, and a guard that fires on prose is switched off
  within a week.
  The race detector and this answer different halves of one question. The
  detector catches a race a test actually triggers. This catches the code that
  could have one before any test has to be lucky.
- **A ceiling on the length of a file, which may only ever be lowered.** A long
  file of short well named functions is not spaghetti and is ordinary in Go, so
  the signal is weaker here than for a function and the threshold is set with
  room rather than at what exists today.
- **Every date the Polish identifier can express is walked, both genders
  included, rather than sampled.** Five hand picked boundaries cover the ones
  somebody thought of. Three hundred and sixty five thousand combinations in
  under a second cover the rest, and each is checked against the independent
  validator on the way through.
  This is also the answer to whether a property based testing library is worth
  taking. Where a domain is countable, exhaustive beats random: a generator
  would have drawn a few thousand of these and called it evidence. The
  condition for revisiting it is written down rather than left to taste, and it
  is the first round trip over a structured value, which arrives with the
  canonical recipe form or the corruption engine.
- **A vulnerability scanner in CI.** It reports only what our code can actually
  reach rather than every advisory naming a module in go.mod, which is what
  keeps it quiet enough to stay switched on. The tool is aimed at places that
  ask what is inside a binary before running it, and the licence flag answers
  only half of that.
- **Fuzzing that actually fuzzes, on a schedule rather than on a push.** An
  ordinary run walks the seeds and stops, which is right for a commit. Finding
  something new takes time rather than triggers, and a thirty second run on
  every push would have been theatre: sixteen point nine million executions
  found nothing. A finding now leaves the runner as an artifact, because an
  input that took ten minutes to find must not die with the machine.
- **The race detector, fuzzing, a linter and a hard limit on the shape of a
  function.** There is no second author and the owner does not read the code,
  so the compiler, the tests and the scripts are the only readers there are. A
  written rule with nothing enforcing it would have been a sentence that
  outlives the decade and changes no line.
  The first run earned its keep: the linter found a dead constant and five
  unchecked writes, the shape limit found three functions past the threshold,
  and the probe written for the new guard broke the guard itself on its first
  run. All four were fixed rather than excused, and the threshold stayed where
  it was, because a threshold bent to fit the code stops being one.
- **The race detector runs with cgo enabled while everything else runs without
  it.** That looks like a contradiction and is not. The promise that the
  command line binary builds without a C toolchain is about the product, and
  the matrix and the cross build keep it. The detector is a test tool that does
  not work without cgo, so refusing it would give up the only mechanical guard
  against a data race to protect a promise it does not touch.
- **Concurrency rules written before the first goroutine exists.** Adding
  parallelism to code that assumes one thread is a rewrite rather than an
  optimisation, so the rules and the detector are in place while there is still
  nothing to catch.
- **Differential fuzzing between the two identifier validators.** They were
  written by one hand from one description, so the question worth asking is not
  whether they agree on the values we produced but whether anything at all
  makes them disagree. Sixteen point nine million executions found nothing, and
  the recipe reader took three and a half million without a panic or a silent
  refusal.
- **`internal/adapter/provider/pl`: the first country provider, and the
  pattern every later one follows.** The rules of the number were read from the
  government description before any code was written, because this project has
  already shipped a catalog entry telling it to break a check digit on an
  identifier that turns out not to have one. An algorithm arrives with a source
  or it does not arrive.
- **`internal/oracle`: validators written from the published description
  instead of from our generator.** A generator and its test are usually written
  by one hand in one hour, and when that hand is wrong both are wrong in the
  same direction and agree with each other. The validator here is shaped around
  the wording of the source rather than around the arithmetic that suited the
  provider: it keeps the weights as the quoted string and takes them apart, and
  it stores the month ranges the description lists rather than the offset the
  provider adds. It is barred from the product by a test, because two
  validators in one binary invites calling whichever is nearer.
- **A third layer outside the repository.** A script recomputes the same
  answers in another language from the same quoted description. Two
  implementations in one language share the idioms of that language, and the
  golden file would otherwise be checked only against the code that wrote it.
- 🔴 **The weights rest on prose and nothing else.** The source gives the
  algorithm in words and **carries no worked example**, so there is no external
  number to check them against, and three agreeing implementations all descend
  from one reading. What stands in its place is a test asking whether the table
  has the property such a table must have, that every single digit error
  changes the result. That test also measured the limit of the same table:
  neighbouring weights differ by even numbers, so some transpositions of
  adjacent digits leave the total unchanged. The catalog already carried a
  transposition mutator, and it now rests on arithmetic rather than on belief.
- **`internal/domain/capability`: the capability registry, with a supplier
  beside every name.** The architecture said a provider declares the
  capabilities and that a mutator is available when the provider brings them.
  Counting them showed both halves were wrong. A provider supplies eight. The
  rest come from the format writer, the schema view, the corpus and the engine
  itself, and reading the field the old way would have left ten mutators of
  layer L0 and L1 permanently unapplied, since no provider brings
  `output.charset-control` or `format.csv`.
  The same mistake had already been caught twice, once for a mutator whose
  applicability condition could never hold and once for one with nothing to do
  in either supported locale. At the third occurrence the mechanism was fixed
  rather than the entry.
- **The supplier is data rather than a comment.** It sits in the published
  catalog beside the capability it describes, so it can be counted by a script
  instead of read and believed. A comment classifying twenty seven names is
  prose, and nothing watches prose.
- **`internal/domain/provider`: the contract, and a declared capability that
  has to mean something.** A provider names itself, says what it brings,
  produces a value and judges one. Four of the capabilities imply an interface,
  and declaring one without implementing it fails, as does implementing one
  without declaring it. A provider claiming a capability no provider can supply
  fails too, which is the registry defect caught from the other end.
  The contract is in the domain and the implementations belong under the
  adapters. The engine calls the interface, so the interface sits where the
  engine can see it, and reference data touches the world, so it does not.
- **Findings are returned as a list rather than the first failure.** A
  contributor writing a provider for their country sees every failing point at
  once, which is the rule the recipe validator already follows.
- **`internal/domain/recipe`: primary key, required and maximum length reach
  the schema view.** A mutator cannot break a constraint it cannot see, and
  four starter set entries were asking for constraints the model did not carry.
  The length unit is carried rather than assumed, which is the gap already
  named once for a mutator that needed a unit of measure and did not get one.
  `Required` is separate from `EmptyShare` on purpose. A share of zero says no
  row happens to be blank, while required says none may be, and collapsing them
  would make the two indistinguishable exactly when a mutator wants to blank a
  required field and find out whether anybody notices.
- **A second bridge between the two surfaces, in `tools/audyt.py`.** The
  capability names and their suppliers live in the published catalog and in the
  code, and the catalog is not compiled into the binary yet, so no test in the
  repository can see both. The check compares them in both directions including
  the supplier, and a probe proves it fails when a supplier is changed on
  either side. Matching names under different suppliers is the dangerous case,
  because everything compiles and every test passes.
- **`internal/adapter/provider/corpus`: reference word lists, as data rather
  than a dependency.** Names, surnames, cities and regions for Polish and
  American data were read out of an open source project for fake data and
  written down here under its licence. Nothing was added to `go.mod` and the
  binary links no external code, so an upstream release cannot move anybody's
  generated data.
  Reading somebody's list and keeping it takes a substantial part of their
  collection, whether it arrived through a generating function or through a
  file. The licence allows that on the condition that the notice travels with
  it, so it does: a header in every file, a registry beside the licences, and
  `--license` printing it from the binary, which is where the data actually
  ships. A gate fails the build in both directions, so a list without
  attribution and an attribution without a list are equally impossible.
  Identifiers and postal codes are deliberately not taken. An identifier has to
  be computed from the date and gender of its own row, and a number with a
  correct check digit may belong to a real person. Their postal codes are a
  digit mask unrelated to the city, which is this project's own
  `L4.coherence.postal_vs_city` mutator rather than reference data.
  The order of a list is part of the byte contract, because a coordinate maps
  to an index. Sorting is by code point, done once at write time, never at load
  and never by the collation of whichever machine runs the tool.
  *This entry was written on 2026-08-02, after the change itself. The commit
  carried the reasoning and this file did not, which is the gap that makes a
  developer changelog worth keeping in the first place.*
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

- **Reference corpora as data rather than as a dependency.** Nine lists for
  pl-PL and en-US, read out of a fake data project at a pinned tag by an
  internal tool, through the syntax tree rather than by installing or running
  anything. Nothing enters go.mod, and the order and contents belong here from
  the moment they were written down.
  Reading somebody's list and keeping it takes a substantial part of their
  collection whether it arrived through a function or through a file, so their
  notice travels with it: a header in every file, a registry beside the
  licences, and `--license` printing it from the binary, which is where the
  data actually ships. A gate fails the build in both directions, so a list
  without attribution and an attribution without a list are equally impossible.
  Identifiers and postcodes were deliberately left behind. An identifier has to
  be computed from the row's own date and gender, and one with a valid check
  digit may belong to a real person. A postcode in both libraries is a digit
  mask unrelated to the city, so the capability that needs a real gazetteer
  still needs one.
  Order is asserted rather than restored at load, because a coordinate becomes
  an index and reordering a list rewrites every dataset generated from it.
- **Continuous integration on three systems.** Build, vet, a formatting check
  and the tests on Windows, macOS and Linux, with the toolchain pinned through
  GOTOOLCHAIN, plus a job that cross builds the command line binary for three
  targets from one machine with CGO switched off.
  The reason is specific rather than tidy: a determinism test compares two runs
  of one machine and passes on each of them separately, so it cannot see a
  difference between systems. The golden values are what notices, and only if
  they run somewhere other than where they were written.
  What it cannot do is written in the file itself. The documents and the
  internal tooling stay out of the repository, so the document guards, the
  mutation runner and the oracle have nowhere to run there.
- **The dependency gate now covers indirect modules as well.** They end up in
  the binary just the same, and the licence binds whoever received the binary.
  Proved by adding a fake indirect requirement and watching the test go red.
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
- **A cross cutting flag only worked before the verb.** The flag package stops
  reading flags at the first argument that is not one, so `--json` after the
  command landed among the arguments and the run failed as a usage error. A
  flag that works on one side of the verb is worse than one that does not
  exist, because its refusal reads as the user's typo. Commands declare those
  flags again and the values are folded together.
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
