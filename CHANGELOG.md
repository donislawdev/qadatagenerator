# Changelog

Everything a user would notice is recorded here.

The format follows Keep a Changelog and the versions follow Semantic
Versioning. A change that breaks a public contract gets a BREAKING section
first, because this tool has an unusual number of them: mutator identifiers,
manifest fields, recipe keys and exit codes all live in somebody else's
repository once they are released.

## [Unreleased]

### Added

- **`--manifest` writes the manifest in a form you can use directly.** Both
  `qadg generate` and `qadg corrupt` take it, and both still write the
  canonical JSON beside it. Three forms: `gherkin` gives an `Examples` table
  you paste into a scenario outline, `csv` opens in a spreadsheet, and
  `markdown` is a table for a bug report.
- **The three carry different things, because different people open them.**
  The Gherkin table has one row per defect that was applied, since a row there
  tells a test what to send and there is nothing to send for damage that never
  happened. The CSV has a row for **every** defect that was ordered, with
  columns saying whether it happened and why not, because it is a review of the
  whole run. The Markdown table shows what happened and counts the rest in a
  sentence under it.
- Whatever a form leaves out, it **counts** in the file itself, so a missing
  entry is a statement rather than a gap. The JSON beside it always carries
  every entry with its reason.
- **The CSV will not run in your spreadsheet.** A column name from the file you
  damaged can start with `=`, `+`, `-` or `@`, which a spreadsheet treats as a
  formula rather than as text, so those are written to be shown instead. This
  protects the first pass: spreadsheets commonly drop the guard when they save
  the file and it is opened again, so read the JSON when the exact bytes matter.
  The values a mutator produced are not in the CSV at all, only in the JSON.
- The flag has **no default**. Without it you get the JSON manifest and nothing
  else. A form this build does not write, such as the parameterised cases for
  pytest, JUnit and NUnit, is refused before your file is read rather than
  after.

- **`qadg corrupt` damages a file this tool did not make.** Point it at a comma
  separated file, name the layers to attack, and it writes the same records with
  damage in them beside a manifest saying what a system reading them should do
  about each one. No recipe is needed and none is written.
- **The manifest says what the tool worked out about your columns, and what it
  worked it out from.** A column is only called a national identifier or a
  routing number when every value in it is accepted by that scheme's own check
  digit. Anything less certain is left as text and the manifest says why.
- **The manifest also lists what this mode never reads out of a file.** Maximum
  length, required columns, uniqueness and relations between columns are things
  a file does not carry, so damage needing any of them is recorded as not
  applied with the reason rather than quietly skipped.
- Flags for this verb: `--layers`, `--rows`, `--out`, `--seed`, `--delimiter`,
  `--per-row`, `--exclude-fields`, `--locale`, `--overwrite` and `--dry-run`.
  `--layers` is required, because damaging the bytes of a file and damaging a
  value in it are different requests and there is no safe default.

### Fixed

- **`--manifest` was missing from the help of both verbs that accept it.** It
  was accepted, named in refusals and used in examples, and `qadg help corrupt`
  never mentioned it, so the only way to find the flag was to already know it.

- **A run that wrote the Examples table did not say it had written it.** Asking
  for the form wrote a third file, and every sentence naming what the run
  produced listed two, so a mistyped output path looked exactly like a form that
  was never written. The summary, the line naming what was written and the
  `--dry-run` announcement now all name it, and none of them names it when you
  did not ask for it.

- **The README said the second pillar was a specification rather than a feature.**
  It has been one since the verb above landed, so the status paragraph now names
  all four verbs and says plainly that this build implements a small part of the
  corruption catalog.
- **The README promised manifest exports in the present tense.** JSON is written,
  and CSV, Markdown and the Gherkin table are specified rather than built. The
  paragraph now says which is which, because a README describing the plan in the
  present tense is where a project starts overstating itself.
- **The manifest specification said no tool emitted it yet.** One does, since
  2026-08-11. What has not changed is the part that matters, which is that one
  implementation is not two.
- **A source file changed while it was being read produced a manifest that
  disagreed with the data beside it.** This mode reads a file twice, and nothing
  compared the second reading with the first, so a file replaced in between gave
  an output whose rows did not match its own header and a manifest promising a
  number of rows that were not there, all of it ending with a success code. The
  run is now refused and says what differed. A file whose columns and record
  count are unchanged but whose values are not still gets through, and the
  refusal says so rather than claiming more.
- **`--rows NaN%` was accepted and quietly damaged nothing.** Every comparison
  against a value that is not a number is false, so it passed the check for a
  share between nought and a hundred and then matched no row, and the run
  reported success with an empty manifest. It is refused now, as it always was
  in a recipe.
- **A refusal naming a column of your file grew as large as the name.** A column
  named with sixty four kilobytes produced a refusal of about the same size, on
  its way into whatever collects your build log. Names are trimmed now, as they
  already were when they came from a recipe.
- **The same was true of a value a numbering scheme refused.** Thirteen messages
  across four schemes wrote the value straight into the sentence, so a long
  value produced a long message. They are trimmed now, and nothing stops a
  fourteenth scheme from doing it again by hand, which is why the repair is a
  check over the schemes rather than thirteen edits.
- **A file called `CON.corrupted.csv` was accepted on Windows.** A name reserved
  for a device is reserved by the part before the first dot, and this build cut
  at the last one, so it refused `CON.csv` and let the two dotted form through.
  The verb that damages a file you already have always produces two dots.

### Changed

- A run that could not apply damage for want of something in the data now says
  where the answer is rather than pointing at a recipe, which the mode above
  does not have.
- **Damage that overruns a declared maximum length no longer slows down as the
  limit grows.** The cost used to rise with the square of the limit, so a field
  declaring a very large maximum could hold a run for hours with nothing written
  and nothing said. It rises in step with the limit now: a million characters
  takes about four milliseconds.
- **BREAKING for recipes, before any release: `max_length` may not exceed
  1000000.** A limit is not only described, it is built, because damage that
  goes past it puts that many characters into one cell of every row it lands on.
  The largest limit this project has ever needed is sixty four. A recipe above
  the ceiling is refused by `qadg validate`, so it fails before the run rather
  than during it.


Nothing has been released yet. The binary reports 0.1.0, which is the version
being built rather than one anybody can download. Releasing is a separate act
and has not happened.

### Added

- **The manifest now says why a defect did not happen in a form a script can
  read.** Every entry with `applied: false` carries a `failure_kind` beside the
  sentence: one of `undescribed`, `unimplemented`, `run_lacks_capability`,
  `field_type_not_covered`, `scheme_not_covered` or `declined_by_mutator`. The
  sentence is written for a person and may be reworded between versions. The
  kind is a closed vocabulary, so a pipeline never has to match on prose to tell
  "this build cannot do it yet" from "your recipe does not carry what it needs".
- **A run that could not apply everything says why, in one line per reason.**
  A bare count of defects that did not happen reads like a shortfall, and most
  of it usually is not one. `qadg generate` now breaks that count down, so a
  handful of entries you can act on is not buried under fifty that nobody can.
- **A run with defects that did not happen still exits 0.** That is deliberate
  and now settled: damage nobody implemented yet, and damage a recipe does not
  carry what for, are the ordinary state of this tool rather than a failure. A
  non-zero code there would fire on nearly every run and teach everybody to
  ignore it. If you want a run to fail on that, the manifest carries what you
  need to decide it yourself.
- **A text value can now be pushed one character past the limit its column
  declares.** Layer L2 builds a value of exactly the declared maximum plus the
  overshoot, which is the boundary a length check written with an off by one
  error gets wrong. Byte limits are overrun in bytes and character limits in
  characters, because the two differ for accented and non Latin text, and a
  value that cannot land exactly on the mark is refused with the reason rather
  than left somewhere past it.
- **A text value can now be emptied.** Layer L2 replaces it with an empty
  string, which is the difference between empty and absent that schemas, forms
  and databases confuse most often. The expectation is soft, because whether an
  empty string is allowed where a value is expected is your decision.
- **A birth date can now be moved without breaking the number that carries it.**
  Layer L4 rebuilds a personal number around a birth date five years from the
  one in the column beside it. The number stays correct and the day, the month
  and the gender stay where they were, so a report has one difference to explain
  rather than several. A date with nowhere to land, such as the twenty ninth of
  February, is refused with the day named instead of being moved somewhere else.
- **An identifier can now be moved into a block its issuer never gives out.**
  Layer L3 rewrites a social security or routing number into a range any
  competent validator refuses. This is a different range from the one the
  correct data already sits in, which is what makes it a corruption rather than
  a no-op, and a value already sitting there is refused with the reason.
- **A row can now carry a correct identifier that contradicts the row itself.**
  Ask for layer L4 and the run rebuilds a personal number so it encodes the
  other gender than the gender column beside it. Nothing is malformed: the
  number passes its own validator and the column holds a legal value, and only
  the pair is wrong. This is the defect applications almost never catch, and the
  manifest marks the expectation as soft because whether to reject such a row is
  the application's own decision.
- **A national identifier can now be broken at its check digit.** Ask for layer
  L3 and the run changes the last digit of a personal number, a tax number, a
  statistical number, a routing number or an international account number, so
  the value keeps its shape and stops adding up. The manifest carries the value
  before and after, and says a correct application should reject it.
- **Damage meant for one numbering scheme is no longer aimed at another.** A
  dataset holding both a personal number and a social security number has two
  columns of the same type, and only one of them carries a check digit. Damage
  aimed at a check digit now lands on the column that has one, and where no such
  column exists the manifest says which scheme the damage was for and which the
  column follows.
- **A recipe can now order damage, and the run writes down what it did.** Give a
  recipe a `policy` section and `qadg generate` produces the dataset, breaks it
  where the policy says, and leaves an `expectation-manifest.json` beside the
  data. Every ordered defect gets exactly one entry: applied, with the value
  before and after, or not applied, with the reason.
- **Nothing ordered can go missing.** A defect this build has no implementation
  for still gets its entry, saying so. That is deliberate: a dataset where most
  of the ordered damage silently never happened looks exactly like a complete
  one, and would be trusted as one.
- **The run says what it did to the data.** How many defects were ordered, how
  many were applied, and where the manifest is. `--dry-run` says what the policy
  will draw on and how much of it this build implements, without writing
  anything.
- **A field described only by a type now gets a value.** Write
  `quantity: { type: integer }` or `amount: { type: decimal }` and the column
  arrives filled instead of empty. A number never overruns the width its field
  declares, and a decimal carries two places on every row.
- **A column this build cannot fill says so before it is written.** A field
  declared as a type this build has no values for, such as `money`, still comes
  out empty, but the run names it and says why. It used to arrive blank with
  nothing anywhere mentioning it.

### Changed

- **Damage aimed at a kind of field now reaches fields described by a
  generator.** The corruption catalog aims at field types and a recipe usually
  declares generators, so the type of a field is worked out from its generator:
  a column drawing `pesel` is a column of identifiers, one drawing `birth_date`
  is a column of dates. A type written in the recipe is used as written.

- **`--license` now prints a machine readable document too.** Run
  `qadg --license --json` and the licenses, the third party modules and the
  reference data come out as JSON, each entry carrying the day its terms were
  read. It is meant for a compliance scanner: without it, a scanner has to parse
  text written for a person, and text written for a person gets reworded.
  Asking for two reports at once with `--json` is refused, because two documents
  printed one after another are not a document.
- **`qadg generate` turns a recipe into files.** It reads a recipe, produces the
  records it describes and writes them as CSV or as JSON. This is the first
  pillar of the tool working on its own: correct data, useful without anything
  being broken on purpose.
- **Related entities come out related.** A recipe saying orders belong to
  customers produces orders that name the customer they belong to, and the tool
  keeps the number of rows the recipe asked for while respecting how many
  children one parent may have. A count the relation cannot reach is refused
  with the arithmetic shown rather than quietly trimmed.
- **A note travels beside the data.** Every run writes a file recording the four
  versions, the seed, the locale, how many rows each table holds, what the run
  did not do, and which settings came from the command line rather than from the
  recipe. It also states that every value is synthetic, because a README does not
  travel with a folder of records.
- **A file that already exists stops the run.** The tool will not overwrite a
  dataset somebody has on disk without being told to. `--overwrite` allows it.
- **`qadg help`, and help for a single command.** `qadg help generate` and
  `qadg generate --help` print the same thing: what the command does, its most
  important limit, a line to paste, the flags of that command and the flags of
  the tool that work beside it. Until now `qadg help` was an unknown command,
  and the five flags of `generate` were documented nowhere a user could reach.
- **Stopping a run leaves nothing behind.** Pressing Ctrl+C ends the run with
  exit code 130 and removes the files it had started, so the output directory is
  left as it was found. Nothing ever appears under its real name before the whole
  run has succeeded, so a stopped run cannot leave half a dataset looking like a
  whole one. Pressing the key a second time ends the process immediately, which
  matters when a run is stuck writing to a full disk.
- **A run the system stops is caught too.** `docker stop`, `kill` and a cancelled
  CI job end the run with exit code 143 and the same clean directory. Those are
  the runs nobody is watching, which is exactly where a leftover file would sit
  unnoticed. The two codes stay apart because they mean different things: 130
  says a person stopped it, 143 says nothing was there to press anything.
- **`--dry-run` says what would be produced and touches nothing.** It is the same
  arithmetic the real run does rather than a separate estimate.
- **Identifiers agree with the columns they were bound to.** A recipe saying the
  national number matches the date of birth and the gender produces numbers that
  actually encode them, so the record is coherent rather than merely well formed.
- **International bank account numbers, as `iban`.** It produces a correct
  Polish account number and checks the check digits of an account number from
  any country.
- **What it checks depends on the country, and it says which.** The check digit
  arithmetic is the same everywhere, so a value whose check digits disagree is
  reported as wrong whatever country it names. The length and the national
  structure differ by country, and only the Polish ones were read at a source
  this project can use. So an account number from anywhere else with agreeing
  check digits is answered as **not checked** rather than approved, and the
  answer names the country. Approving it would mean claiming something nobody
  verified.
- **What that costs, measured rather than estimated.** For a country whose
  length is not checked, a value with one digit removed still passes in 3,391%
  of cases and one with a digit added in 1,690%. Both were measured over 89
  example account numbers from 89 countries. Neither is silent: the answer
  states what was not checked.
- **A Polish settlement number is checked against its own rule.** The eight
  digits naming the bank carry a check digit of their own, so a value the
  central bank could not have assigned is reported as that rather than as an
  arithmetic mismatch. The two are different defects with different fixes.
- **Any spacing is read and none is invented.** An account number written in
  groups of four, in groups of any size, or with no spaces at all is the same
  number. A value handed back keeps the spacing it arrived with, so a corruption
  changes one thing rather than two.
- **The tool now states, from the binary, that every value it produces is
  synthetic.** `--license` prints it beside the licence report. Names,
  addresses, identifiers and account numbers describe nobody, and where one
  resembles a real person, company or institution that is a coincidence of
  arithmetic rather than a record of anything. This is printed rather than only
  written in the README, because a README does not travel with a file of data,
  and because several of these numbering schemes have no block that is never
  issued: a value with a correct check digit cannot be shown not to be
  somebody's, which is exactly why it has to be labelled.
- **United States postal codes paired with their places, as `US.postal`.** A
  registry of 28 658 codes built from two files of the United States census.
  It answers which place a code belongs to and which codes a place holds, and
  it can produce a real code belonging to a different place, which is what the
  corruption that breaks a postal and city pair needs.
- **The registry is an approximation of the postal scheme and the tool says
  so.** The census publishes tabulation areas rather than postal codes, and
  states that not every valid code is represented by one. So a code the
  registry does not carry is not reported as wrong: it may be a real code the
  census does not cover, and faulting it would report a defect in data that has
  none. The check answers about the shape of a code and nothing else.
- **Which place a code belongs to is computed, not quoted.** The census lists
  every place a code overlaps, sometimes dozens, and ranks none of them. The
  registry keeps the one sharing the largest land area. Codes falling outside
  every place are not in the registry at all.
- **United States routing numbers, as `US.aba`.** Nine digits with a check
  digit, readable and writable both bare and in the four four one grouping that
  federal regulations print them in. It brings the check digit capability and
  the blocks a corruption can move a value into.
- **This is the first identifier with no safe range, and the tool says so.**
  Every other scheme here has a block nothing is ever issued from, and correct
  data is generated inside it so that nothing produced could belong to a real
  subject. The routing number registrar marks several series reserved and adds
  that their use will be decided by its Board, which makes them stock rather
  than sanctuary. Correct values are therefore generated in the series issued to
  banks, and any one of them could belong to a real institution. A dataset
  carrying them has to be marked as synthetic, and the tool does not claim
  otherwise anywhere.

### Fixed

- **The catalog specification no longer says its examples are unverified.** That
  sentence was written when it was true and stopped being true the same day the
  examples became tests, and only the data file was corrected. It now says which
  examples are checked against a scheme and which are illustrative, which is the
  distinction an implementer needs.
- **The manifest specification says when it stops changing.** Everything in it is
  mutable until the first public release, and that trigger is now written down
  rather than left to be recognised afterwards. The moment a format freezes is
  otherwise only ever visible in hindsight, by which time somebody already
  depends on it.

- **The published examples now carry identifiers a validator would accept.**
  Three catalog entries showed a national identifier that no implementation of
  the scheme would take, and two of them contradicted their own note: one
  promised a recomputed check digit and did not have one, so the value failed
  on the checksum rather than on the thing the entry is about, and one showed
  two values encoding the same gender in an entry about gender disagreeing. The
  same wrong values appeared in the manifest specification. Every value is now
  computed from the arithmetic, and tests recompute them, so an example you
  implement against behaves the way its entry says it does.

- **`qadg validate` no longer approves a recipe `qadg generate` refuses.** A
  recipe naming a generator this build does not have passed validation with the
  words "can be used" and no warning at all, while generating from the same file
  failed. Validation now reports it as a problem, names the generator and lists
  the ones that exist, so both commands answer the same question the same way.
- **Every refusal says what to do next.** Failures coming from inside the tool
  reached you as a single line saying what went wrong. They now carry the same
  fourth part the command line refusals always had: a write failure points at
  the path and the permissions, an unusable recipe points at `qadg validate`, an
  impossible one points at what to change, and a stopped run says nothing was
  left behind.
- **A refusal after a command no longer offers flags that command cannot take.**
  Typing an unknown flag after `generate` or `validate` listed `--version` and
  `--license` among the accepted ones. Neither has ever worked there, so the
  message sent the reader looking for a mistake of their own. The list is now
  the list that command actually accepts, and the suggestion points at the help
  of that command.
- **The help now says that the tool generates data.** It stated that no command
  in this build produces any, which stopped being true the day `qadg generate`
  arrived, and it listed one of the two commands. A run with no arguments named
  one of them as well. Both now come from the list the tool itself reads.
- **A recipe could refuse a machine the memory it needed.** The reader limits
  how deeply a recipe may nest, and the limit counted only the shape written
  with square brackets. The same shape written with braces was not counted at
  all, so a file well inside the size limit could cost hundreds of megabytes
  and, at the size limit, enough to have the run killed by the operating
  system. A killed build says nothing about why it died. Both shapes now count
  towards the same depth.
- **A share written as NaN was accepted, and switched off a check while it was
  there.** Every comparison against NaN is false, so such a value passed the
  range test and then passed the rule that refuses a field which is both
  required and sometimes empty. A recipe carrying a contradiction reported that
  it could be used.
- **A recipe saved as UTF-16 now says so.** Notepad offers that encoding under
  the name Unicode. A recipe saved that way was refused for not being a mapping
  of keys, with advice to add a version line that was already the first line of
  the file. The refusal now names the encoding and points at the entry to pick
  instead.
- **A recipe saved in a single byte encoding is refused rather than quietly
  damaged.** On a Polish Windows the ANSI entry of a save dialog means
  Windows-1250, and older editors and spreadsheet exports still default to it.
  Such a file used to be accepted as usable while every accented name in it
  arrived broken, so an entity named zażółć would have become the heading of a
  generated column as za followed by four replacement characters. Nothing said
  so, because the report announced that the recipe could be used.
- **A number too large to hold is named rather than turned negative.** A count
  above the range this build can hold wrapped around and was reported as a
  count below zero, which described something nobody had written.
- **A consistency rule that means nothing is refused.** A rule with no fields
  on the right, and a rule naming its own field, both used to be accepted in
  silence. Neither can be satisfied when data is generated or broken when it is
  corrupted, so both would have surfaced much later as a corruption that was
  never applied.
- **A flag given a value is no longer told it does not exist.** Writing
  `--no-color=nonsense` answered that this build does not define that flag,
  while the line below listed it among those accepted. The message now says the
  flag is a switch and takes no value.
- **A warning keeps its meaning under `--quiet`.** The warning about
  declarations this build could not check printed its headline on a channel
  `--quiet` keeps and its entries on one it silences, leaving a warning that
  something had been skipped and nothing saying what. The headline now carries
  the count and says how to see the detail.
- **A directory given as a recipe is refused in plain words.** The message used
  to come from the operating system, which said "Incorrect function." on
  Windows and something else elsewhere, and neither said what to do next.
- **A repeated `--no-color` is read the way the parser reads it.** The last
  spelling wins in both, where the colour decision used to take the first.

### Changed

- **A bound gender accepts far more spellings.** The column saying whether a
  person is a woman or a man comes from your recipe or from your own file, and
  there is no telling whether it says female, kobieta, f, k or the digits a
  database exports. All of those are read now, in either language, in any
  letter case. The codes 1 and 2 of ISO 5218 are read as well, because that is
  what a column exported from a database usually holds. The codes 0 and 9 of
  the same standard are refused on purpose: they mean not known and not
  applicable, and a number that encodes a sex cannot be made to agree with
  either. A value none of this recognises is now an error naming what would
  have been accepted, where it used to be a silently drawn gender that
  contradicted the rest of the row.
- **A validator can now say it did not check a value.** A verdict used to have
  two answers, correct and faulty, and some schemes have a third case: a form
  the build recognises and does not check, such as the fourteen digit
  statistical number. Reporting that as faulty means reporting a defect in a
  value that has none, which matters most when the column came from somebody
  else's file.
- **The capability describing reserved ranges is now two capabilities.**
  `identifier.reserved-range` used to mean three things at once: that a scheme
  has a block nothing is ever issued from, that values can be generated inside
  it, and that a value can be moved into an unassigned block. Some numbering
  schemes have the last of those and not the first, because their registrar
  names spare blocks it may still issue from. Writing them as one capability
  forced such a scheme to claim a safe range it does not have. The write side
  is now `identifier.unassigned-target`, and `L3.range.unassigned` requires
  that one, which is what it always needed.
- **A mistyped command now tells you which one you meant.** `qadg vlaidate`
  answers that `validate` is one small edit away and gives you the line to run.
  A word that is nothing like a command still gets the plain refusal and the
  list of what exists, because a suggestion offered for anything vaguely
  similar would be misleading rather than helpful.
- **The help starts with two examples you can paste**, before the usage pattern
  and the flag list. The pattern is still there for people who know the tool.

### Added

- **A new reverse mutator, `REV.identifier.reserved_range`.** It supplies a
  correct identifier taken from the block this tool generates inside on
  purpose, and expects the application to accept it. This is the one entry
  where a rule of this tool meets a rule of a real application head on:
  generating inside a never assigned block is what keeps a generated number
  from resembling somebody's real one, and an application that checks
  assignment ranges will refuse such a number anyway. The entry makes that
  collision testable, so a team finds out which side its application is on
  rather than reading a red test as a defect. Confidence is soft, because both
  answers are reasonable depending on what the application knows.
- **Polish statistical numbers (REGON) can now be generated and checked.** A
  generated number is nine digits with a correct check digit, written as bare
  digits with no separators.
  Three limits are worth knowing before you rely on this. The fourteen digit
  number of a local unit is refused, and the refusal says that this build does
  not check that form rather than calling your value wrong. The scheme sets
  aside no block of numbers for testing, so a generated one could be some real
  organisation's, exactly as with the personal and tax numbers, and datasets
  carrying any of them should be marked synthetic. And this check digit is
  weaker than the tax number's: it folds two cases onto the same digit, so
  about 1.8 percent of single digit errors pass it unnoticed. If you are
  corrupting a REGON on purpose and expecting an application to reject it, that
  is the share of corruptions no check digit test can catch.
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
