# Changelog

Everything a user would notice is recorded here.

The format follows Keep a Changelog and the versions follow Semantic
Versioning. A change that breaks a public contract gets a BREAKING section
first, because this tool has an unusual number of them: mutator identifiers,
manifest fields, recipe keys and exit codes all live in somebody else's
repository once they are released.

## [Unreleased]

### Changed

- **Opening an application response no longer makes the Manifest browser
  taller.** How many records the response never answered now stands beside
  its file name, and when nothing disagreed the same place says so. What the
  comparison could not see is read in the card under the table, from a button
  in its heading that shows how many there are, the same way the envelope of
  the run is. The table has the room the notes above it used to take.

- **The window is drawn on a lighter graphite with a blue accent instead of a
  near black with violet.** The accent now also marks where things are, not
  only the one action a screen is for: the icons on buttons and in the column
  of screens, the headings over groups, the line under each section heading,
  the header of the manifest table, and the screen that is open, which keeps
  its colour when the pointer is on it. The red used for a rejection is a
  little lighter, so it stays readable on the new ground. The headings over
  the groups in the column of screens are now quieter than the names under
  them, as they were meant to be.

- **The button a screen is for now has the same margins as every other
  button**, so it is slightly wider, and when it cannot be pressed its label
  dims the way theirs does.

- **The Manifest browser has been rebuilt so that the whole table fits the
  window.** The table had six columns, and the two saying whether the damage
  happened and why it did not were off screen unless you scrolled sideways. It
  now has four: the damage, where it is, what was expected, and the outcome,
  which says what the application did with the row once you open a response.
  Everything else about an entry is read out in a card under the table, for the
  entry you choose with a click or the keyboard. The button in the heading of
  that card shows what the manifest says about the run in the same place, and a
  second press goes back to the entry, so opening it no longer makes the window
  taller. What a comparison could not check is in a section you can open over
  the table, and the command that runs the same comparison in a terminal runs
  along the foot.

- **The Recipe screen has been rebuilt with its button in sight.** The button
  that generates and the command line stood under the answer and fell below the
  fold of an ordinary window. A strip along the top now holds where a run
  writes, whether it may replace files already there, and the button. Under it
  the screen is two equal halves: your recipe on the left, and on the right one
  card saying what the recipe describes while you write it, or what the last
  run did once you press the button. The divider between the halves is gone,
  because the keyboard could not move it, and the recipe field shows where the
  keyboard is with the same ring as every other field.

- **The Take and corrupt screen has been rebuilt with its button in sight.** The
  button that damages the file, the state of a run and its result sat below the
  fold of a scrolling pane. The file, how it is read and the button now stand in
  a strip along the top, the settings in a column on the left, and on the right
  one card saying what a run would do, or what it did, over a panel saying what
  the file holds. The kinds of damage are check boxes, and the card says for
  each kind you ticked how much of it this build can carry out. Every setting
  can be changed before a file is chosen, and changing one after a run brings
  back the plan of the next run.

- **The About screen has been rebuilt so that what you come for is on the first
  screen.** The versions and the licenses stand side by side in one card, under
  them the two sentences about the data you generate, and then the lists of what
  went into the build, the short ones first. The two buttons that save a file
  are in the strip along the top, which does not scroll. The modules are one row
  each in columns you can scan, and their web address is no longer repeated on
  screen, because it is the name of the module with https:// in front of it. It
  is still in the report the button saves.

- **The About screen spells license the way the command line does.** The two
  buttons said licence, while the headings and `--license` said license.

- **A sign beside a sentence now stands level with its first line.** Beside a
  sentence long enough to wrap it stood level with the middle of it, on the
  result of a take and corrupt run, on the verdicts of the recipe screen and on
  the notes of a comparison in the manifest browser.

- **The button that does the main thing on a screen is now the accent colour**
  rather than grey, so you can find it without reading every control. On the
  Quick value screen that is Copy.

- **Every arrow on a control that opens a list is now drawn in the same family
  as the rest of the icons.** The choosers along the top of a screen and the
  panel of generators on the recipe screen used to draw the toolkit's arrow,
  which came from another set and read as a piece of a different program.

- **Controls in a row are now the same height.** A text field used to be four
  pixels shorter than the chooser beside it, which read as one of them being
  wrong rather than as a small difference.

- **Buttons are drawn by this program rather than by the toolkit underneath it**,
  so they look the way the design says instead of the way the library ships. The
  visible differences: an icon on a button no longer touches its label, a button
  that cannot be pressed loses its outline rather than only going pale, and the
  mark showing where the keyboard is no longer floods the whole button.

- **Every screen in the sidebar now has an icon beside its name**, so you can
  find the one you want without reading the whole column.

- **The screen you are on is marked by a bar at the edge of its name and a
  shaded background**, instead of by an arrow. The arrow occupied the same
  place as the icons, so it could not stay once every screen had one.

- **The Quick value screen has been rebuilt.** The values are a numbered list you
  can read down rather than one block of text, every other row is shaded so your
  eye does not slip between them, and the list grows with the window. The
  settings sit in one strip along the top with the seed taking the room left
  over, and the command that would do the same thing in a terminal runs along
  the foot.

- **The screen now tells you what kind of value you are looking at.** Beside the
  values it says what the kind produces, which numbering scheme is behind it,
  whether a wrong character would be caught by the number itself, and whether
  the values come from a block that is never issued to anybody. Where a scheme
  has no such block, it says so plainly, because a value from one of those may
  match a number that was really issued.

### Added

- **`qadg-gui -catalogue` opens a catalogue of controls instead of the
  window.** It shows every control the screens are built from, in every state
  it can be seen in: at rest, under the pointer, pressed, with the keyboard on
  it and out of reach, with very long text and values wider than a field. It
  is meant for reviewing how the window looks and does no work. Any other flag
  is refused with exit code 2 and a message on the error channel, and no
  window opens, so a misspelt `-catalog` does not quietly open the ordinary
  window. `qadg-gui -h` lists the flags.

- **The About screen lists the reference data built into the program**: where
  the names, places and other lists come from, under which terms, the day those
  terms were read, and which lists came from each source. `--license` has
  printed this all along and the window never showed it.

- **`--license` now names the typeface and the icon set built into the window.**
  The report listed the modules this tool links and the reference data it
  carries, and said nothing about the font and the icons, which travel in the
  window binary and are kept on the condition that their licence travels with
  them. The About screen shows the same two entries, and a button there writes
  their full licence texts to a file you can keep.

### Fixed

- **The dialog you choose a file in, and the menu a text field opens under the
  right mouse button, now speak the language of the window.** They used to
  take the language of the operating system, so on a desktop set to another
  language they spoke that language inside an English window. The names of
  the places at the side of the file dialog, such as Home and Documents, stay
  in English whatever the desktop.

- **The card under the manifest table reads out the entry you chose.** It
  followed the pointer, so moving the mouse across the table changed what it
  said while the highlighted row stayed where you had clicked.

- **Opening a response in the Manifest browser keeps the entry you had
  chosen**, and your place in the table, instead of clearing both.

- **A comparison missing one record says "1 was never reacted to"** rather than
  "1 were", on the command line and in the window. The notes of a comparison in
  the window begin with a capital letter, and the reason an entry was not
  applied is joined to its kind by "because" rather than after a full stop.

- **`qadg verify` begins the detail of a finding with a capital letter.** A
  finding read "accepted although it should have been rejected. the value was
  taken".

- **The licence texts of the font and the icons are now inside the program.**
  They were in the source tree, which covers anybody who builds this tool, and
  they were missing from the built program, which is what most people receive.

### Fixed

- **Every heading of the manifest table is now readable.** The column saying how
  certain an expectation is had nine times the room its two words need, and the
  last column, the one carrying the explanation, was cut through the middle of
  its own heading. The table is narrower and all six headings fit.

- **The Quick value screen now tells you why it turned a request down.** Asking
  for a value it could not produce showed the line it shows before you have
  chosen anything, so a date of birth written the wrong way round answered with
  a request to choose a kind you had already chosen. It now says a date of birth
  is written as YYYY-MM-DD, and a kind this build carries no data for in the
  chosen locale says that instead.

- **Every list you pick from now shows what is picked.** The chooser controls on
  the Quick value and Take and corrupt screens drew as empty boxes: the kind of
  value, the locale, the gender and the settings all had their text missing,
  with a stray mark in the corner where it should have been. They report what
  you chose and act on it correctly again.

- **A gender written the short way now reaches every value that reads one.**
  A recipe binding a gender as f, k or the digits an export gives was understood
  by a PESEL and ignored by the generator of first names, so one row could carry
  a woman's identifier beside a name drawn without regard to it, under a rule
  that said they had to agree, and nothing said so. Every value that reads a
  gender now reads the same twelve spellings. Data produced from recipes binding
  female or male is unchanged.

### Added

- **Each screen now shows which button it is for.** Every button looked the
  same, so nothing said which one you came for. Damage this file, Copy on the
  Quick value screen, and Open manifest are now filled. Stop, Again, Choose a
  file and the buttons that copy a command stay as they were.

- **The window opens showing values.** The Quick value screen, which is the one
  the window opens on, used to start as an empty form with nothing chosen. It
  now starts on a kind and shows ten values, the line that reproduces them in a
  terminal, and the seed they came from. The kind it starts on draws from a
  range reserved for testing, so nothing shown before you have asked for
  anything can belong to a real person.

- **The manifest browser says what it will show you before you open anything.**
  It used to open on one sentence and an empty screen. It now lists the fields
  every manifest carries about the run that wrote it, and the six columns each
  ordered damage arrives in, with a line saying what each one holds. The button
  for an application response also says why it cannot be pressed until a
  manifest is open.

- **Ask for more than one value at a time.** `qadg quick pesel --count 10`
  writes ten values, one to a line and nothing else, so a line of them can be
  piped or pasted straight into a form. Without the flag it writes one, exactly
  as before. The values are the ones a recipe of that many rows would produce
  from the same seed, in the same order, so asking for more never changes the
  ones already there. They are drawn rather than dealt, so a kind of value with
  a short list of data behind it may repeat.
  The Quick value screen opens showing ten, with a list to change how many.

- **The Take and corrupt screen now shows what it does before you choose a
  file.** The settings a run would use are on screen from the start, greyed
  out until a file is chosen, and each half of the screen says what will
  appear in it. It used to open as one button and two sentences with an
  otherwise empty window.

- **Damage that swaps two adjacent digits, and says when the swap was invisible.**
  Transposing a pair of neighbouring digits is the mistake people make most
  often when copying a number, and it is now one of the kinds of damage this
  tool can do to an identifier. Not every scheme can see one. An article number
  and a book number miss exactly the pairs differing by five, and a mobile
  equipment identity misses nought against nine. Rather than promise a rejection
  that will not happen, the run swaps the pair, asks the scheme whether the
  value is still correct, and where it is, puts the value back and records the
  damage in the manifest as not applied, with a sentence naming the two
  positions and saying the scheme still accepts the result. The pair is drawn
  from the seed, so a run repeats exactly and two seeds differ.

- **Four more kinds of identifier: `vin`, `ean`, `imei` and `isbn`.** A vehicle
  identification number of seventeen characters, a thirteen digit article
  number, a fifteen digit mobile equipment identity, and a thirteen digit book
  number. Each one carries a correct check digit, each is checked against the
  published rules rather than against the way this tool happens to build them,
  and each can be damaged by the corruption catalog like the identifiers that
  were already here.

  `isbn` produces the thirteen digit form only. The ten character form that
  books carried until the end of 2006 is not generated, because the
  International ISBN Agency does not publish its calculation in either edition
  of its manual and states that an ISBN now always consists of thirteen digits.

  Two of these narrow what they write without promising anything about it.
  `imei` writes only inside the block reserved for test equipment, and `isbn`
  only inside the one window the agency currently leaves without a registration
  group, so a generated value is not one a real device or a real book carries
  today. Neither is described as a safe range, because the bodies that issue
  those numbers can issue inside those spaces. `SECURITY.md` says which schemes
  have a genuinely safe range and which do not.

  **Two of them keep a promise the others cannot.** An article number is
  generated only inside the prefix GS1 sets aside for demonstrations of its own
  system, so it cannot belong to a real article. An equipment identity is
  generated only inside the block reserved for test equipment, which cannot be
  supplied to the market, so it cannot be a device somebody owns - though unlike
  the article number that block does have codes issued inside it, so this is a
  narrower promise and SECURITY.md says which schemes make which. A vehicle
  number has no such block at all and its values sit in the space that is
  actually issued.

  **Checking is deliberately wider than generating for all three.** A correct
  value from outside those blocks is somebody's real one, and reporting it as
  wrong would be a defect in your data that is not there. One case is worth
  naming: a sixteen digit equipment identity carries a software version number
  and has no check digit at all, so it is reported as recognised and not checked
  rather than as valid or wrong, and no check digit damage is aimed at it.

- **One value, now: `qadg quick pesel`.** A sixth command that produces a single
  value of the kind you name and writes nothing anywhere. No recipe, no output
  directory, no manifest. It is the only command whose runs differ from each
  other: without a seed it draws a new one every time, so asking twice gives two
  values, and naming a seed makes it repeatable again. The seed it drew is
  printed on the log channel, so the value can always be asked for a second time.

  `--gender` and `--birth-date` bind the value to what it has to agree with, and
  either one is refused for a value that takes no account of it rather than
  being accepted and ignored. Which values take account of which depends on the
  locale: a Polish surname reads a gender and an English one does not.

- **The window has a Quick value screen**, which is the same thing on the other
  surface. Choosing what kind of value you want produces it, with no button to
  press, and Copy puts it in the clipboard and says that it did. The fields for
  a gender and a date of birth appear only beside a value that takes account of
  them.

- **Every control you can use has a visible edge.** Buttons and choosers were
  drawn without one, so on this dark theme they read as words rather than as
  things to press. A disabled control still has none, which is now the clearest
  way to tell that it cannot be used.

- **The window writes datasets.** The recipe editor has a bar along the bottom:
  a directory to write to with a chooser beside it, a tick box for replacing
  files already there, and a button. Above the button it says which files a run
  would write, so you know before you press it. While a run goes on the settings
  are out of reach and the button becomes a way to stop it, and closing the
  window stops the run first, because nothing appears under its real name until
  the whole run has succeeded.

  The directory starts empty and the button waits for one. A window has no
  working directory anybody can see, so this one refuses to guess where your
  files should go.

  What the button writes is what the rows above it showed: the preview and the
  run are the same plan, the same engine and the same writer, and the only
  difference is that one of them reaches a disk.

- **The recipe editor shows the rows your recipe would produce.** Under the
  verdict there is now a table for each entity, carrying the first ten rows of
  it exactly as a run would write them, columns included: a child table shows
  the column naming the parent it belongs to, which comes from the file rather
  than from the recipe. A table says how much of itself you are looking at, so
  ten of twenty five rows reads as ten of twenty five. Nothing is written to
  disk to produce it.

  Two answers that had nowhere to appear before now do. A recipe that reads
  correctly and still cannot be run says so instead of being approved, and a
  recipe that asks for no rows says that in a sentence rather than drawing an
  empty table. Under the rows, beside the list of what was not checked, is a
  second list: what the run will not do, such as a column this build has no
  values for and would leave empty.

- **A recipe editor in the window, and the window opens on it.** Write or paste a
  recipe and see what is wrong with it as you type: every problem at the place
  and the line it belongs to, the values that would have been accepted, and what
  to do instead. A recipe with nothing wrong says so and says how much of it was
  read, so an approval is never bare. Underneath either answer is the list of
  what this build did not check. Nothing is written to disk: this screen reads
  what you type and judges it, and the step that produces files comes next.
  `qadg validate` is now reachable from the window as well as from a terminal,
  and both give the same answer because both ask the same question.
- **The Tab key leaves the recipe editor instead of being typed into it.** A
  multi line field normally swallows Tab, which leaves anybody working from the
  keyboard unable to get out of it. Tab now moves on to the next control. A tab
  character cannot be used to indent YAML anyway, so nothing that could have
  been written is lost, and a tab inside a quoted value is still written as `\t`.

- **A problem in a recipe says which line it is on.** `qadg validate` used to name
  the place a problem belongs to, such as `entities.customer.count`, and leave
  you to find it. It now says the line as well, on the report you read and in
  the one `--json` produces, including the problems that are about what this
  build can do rather than about the shape of the file, such as a generator it
  has no name for. A problem about the file as a whole, or about a key that was
  never written, carries no line rather than a misleading one.
- **The window stays answerable while it reads a file.** Choosing a large file,
  dropping one onto the window, changing the field separator or opening a
  manifest no longer freezes the window for as long as the reading takes. The
  screen says what it is reading and gives everything back when it is done, and
  the button that damages a file is unavailable while the description of that
  file is being replaced.
- **Both panels of the take and corrupt screen scroll from the keyboard.** Down,
  Up, Page Down, Page Up, Home and End drive whichever panel holds the keyboard,
  and a frame says which one that is. Everything on that screen can now be
  reached without a mouse, including while a run is going on, so you can read
  what the file holds while it is being damaged.
- **A manifest browser in the window.** Open the manifest a run left beside its
  dataset and see what that run promised: what was damaged, where, what your
  application should do about it, and how much each of those expectations is
  worth. Damage that was ordered and could not be applied is on the screen too,
  in its own colour, together with the reason. A run where some of the ordered
  damage did not happen looks nothing like a complete one.
- **The same screen shows what your application did.** Open a response record
  beside the manifest and each judged row carries the verdict, with
  disagreements named. What the comparison could not settle, and what it could
  not see at all given what the record carried, are stated under the table
  rather than left out.
- **A screen that damages a file you already have.** Choose a file this tool did
  not make, or drop it onto the window, and the screen says what it holds: how
  many records and columns, what each column was taken to carry, and what that
  conclusion rests on. Nothing declared any of it, so you are the only person
  who can say a conclusion about your own file is wrong, and you can only say it
  if you can see it. What looking at values can never establish is listed too.
- **The same screen says what a run would do before it does anything.** The
  kinds of damage, the share of records, the columns to leave alone, the field
  separator, the seed and the rest are all there, and the answer beside them
  changes as you change them. A setting the run would refuse takes the place of
  that answer with the reason, what would be accepted instead, and what to do.
  Nothing is written until you press the button, so this is what the command
  line prints under its dry run flag without a switch to turn on.
- **The button, and the way to stop what it started.** A run in progress greys
  out every setting and offers the way to end it. Nothing appears under its own
  name until a whole run has succeeded, so a run you stop leaves nothing behind,
  and closing the window during a run stops the run first rather than leaving
  files nobody will clean up. When it finishes, the screen says which files it
  wrote and where, how many defects were ordered and how many reached the data.
  If some could not be applied, it says how many and where the reason for each
  one is.
- **The window now holds more than one screen.** Screens are listed down the
  left and the one being read is marked. Today there are three: the take and
  corrupt screen, the manifest browser and the About screen.
- **A file chooser.** The window can open a manifest, a response record and a
  file to damage from anywhere on your disk. Nothing is read until you choose
  it, and nothing is written until you ask for it.

### Changed

- **A defect in the tool no longer blames a recipe you may not have.** When
  something goes wrong inside `qadg` itself, the advice said the fault was in the
  tool rather than in the recipe. `qadg corrupt` and `qadg verify` are run
  against files rather than recipes, so that sentence sent people looking for a
  document they never wrote. It now says the fault is in the tool rather than in
  anything you gave it.

- **Twenty refusals were telling you two different things to do.** A refusal
  ends with a `try:` line saying what to do about it. Where the message above it
  already carried advice, you got both, and the second was picked from the kind
  of failure rather than from what actually went wrong: one told you to split the
  run and then to change the flags describing the file, and another pointed at a
  flag, which the window does not have. Each of them now says one thing, and
  none of them names a control, so the same advice makes sense whichever surface
  you are on.

- **Take and corrupt stays quick on a wide file.** The panel saying what your
  file holds described every column and built six things on screen for each one,
  which at five thousand columns was 5.4 MB of interface before anything was
  drawn. It now accounts for the same fifty columns the log does and says how
  many it did not describe, so it costs about 55 kB however wide your file is. A
  file of ordinary width looks exactly as it did, with every column described.

- **`corrupt` no longer writes a manifest it cannot read back, and the manifest
  now says how wide your file was.** The part describing a file nobody declared
  wrote an entry per column, so a wide enough file produced a manifest `qadg
  verify` refused, even when the run had damaged nothing. Measured: 260 000
  columns produced 50.2 MB and was refused. The account now stops at 10 000
  columns and the section settles at about 1.9 MB however wide the file is. A
  file of ordinary width is unchanged.

  `schema_inference` carries a new required member, `columns_total`, holding how
  many columns your file has. When it is larger than the number of entries, the
  entries cover the first columns of the file and nothing is claimed about the
  rest. It is always written, so a complete account and a bounded one are never
  confused for each other.

  This narrows what `schema_inference.columns` promised. Nothing has been
  released yet, so no manifest in anybody's repository changes meaning, and the
  format is still a draft you should pin if you implement it.

- **Reading a manifest or a response record costs about half what it did, and
  turning away one that is too large costs almost nothing.** Both files are read
  in one piece and both are bounded at 48 MB, which has not changed. What
  changed is that the length is now asked for before the buffer is made, so a
  file just under the limit is held once instead of twice. Measured on a 38.6 MB
  manifest, five runs each: a peak of 88.8 MB before and 46.6 MB after. A file
  past the limit is refused on its length without being read at all, which took
  the refusal from 126.6 MB to 15.6 MB.

- **The window is built on Fyne 2.8.1.** A patch release of the toolkit the
  window uses, carrying its own performance work and fixes. What you see from
  here is the version `qadg --license` and the About screen report, and two
  components that left that list because this build no longer uses them.
  Nothing this program draws changed.

- **Generating from word lists is about thirty times faster.** Every name, city
  and region used to be drawn by reading and splitting the whole list again.
  Measured on 100000 rows of four columns with three word lists: a median of
  11.2 seconds before and 0.34 seconds after, over five interleaved pairs. The
  data is identical byte for byte, so the same seed still gives the same
  dataset.

### Fixed

- **A file too large to read back no longer sends you looking for a flag that
  does not exist.** `qadg verify` refused an oversized manifest and then advised
  changing the file or the flags describing it. The file is one a run wrote, and
  no flag decides how large a manifest comes out. It now says to compare against
  a manifest from a run over fewer columns. The refusal for an oversized
  response record carried two pieces of advice, the second contradicting the
  first, and now carries one.

- **The warning about a manifest too large to read back names both of the things
  that make one grow.** It told you to damage fewer rows. A manifest also
  describes every column of the file, whether anything in that column was
  damaged or not, so a wide file produces a large manifest with no entries in it
  at all and damaging fewer rows changes nothing. It now says to damage fewer
  records or work on a narrower file.

- **The window says it too.** Take and corrupt only warned about this on the
  command line, so damaging a wide file in the window gave you your files, counts
  that added up, and a refusal from the Manifest browser days later with nothing
  connecting the two. The run now reports which files it wrote that this build
  will not read back, and the screen says so directly under the files it wrote.

- **A recipe whose relation and one of its fields want the same column is now
  refused by every verb that reads it.** An entity that belongs to `customer`
  and also has a field called `customer` cannot be written, because the column
  naming the parent and the column holding the field are one column. Until now
  `qadg validate` called such a recipe usable, `qadg generate --dry-run` counted
  the rows it would write, and only the real run refused it, with a message
  telling you to run `qadg validate`. All three now refuse it, and `validate`
  names the field and the line it is on so you know which one to rename.

- **`qadg validate` no longer says `1 entities and 1 fields`.** A recipe with one
  entity and one field in it now reads `Understood 1 entity and 1 field.`, which
  is what the same sentence in the window already said. The two surfaces
  answered one question about the same recipe in two ways.

- **Closing the window from outside no longer leaves a run's working files
  behind.** Stopping the container, logging out or shutting the machine down
  during a run now ends that run first and gives the window up once it has, the
  way the close button already did and the way the command line already did.
  Until now those left temporary files and the marker naming what the run was
  about to write, and the next run refused because of a marker that no longer
  meant anything.

- **The nested JSON form put records of the third level under the wrong parent.**
  A recipe with three levels of relation, such as customers holding orders
  holding items, wrote the first parent's grandchildren again under the second
  and left the real ones out of the file. The count of records was right, the
  run ended successfully, and nothing said anything was wrong. If you have
  generated nested JSON from a recipe with three or more levels, regenerate it.
  Two levels were never affected, nor was the file per entity form.
- **A run could write two things under one name and report success.** If an
  entity was named after a file the run itself leaves beside your data, such as
  run-note with the JSON format, one silently replaced the other while the log
  reported both. Such a recipe is now refused before any work, and the message
  names the file and tells you to rename the entity. Without the overwrite flag
  the same recipe used to be refused with a message about another run writing
  the file, which was never true.
- **A failed run could leave half a dataset under its real names.** If a rename
  failed part way through, because a file was held open by another program or
  the disk was full, the files already renamed stayed behind without the record
  of the run or the manifest. The whole set is now taken back, so the directory
  is left as it was found. A machine losing power between two renames is still
  outside what this can promise.
- **validate and dry runs no longer pass a recipe that cannot run.** A generator
  this build has, asked for in a locale it carries no word list for, used to
  pass both and fail on the first row. The same is now refused by all three
  verbs with the same sentence. An output format given with the flag rather than
  in the recipe is also checked before the run, so a dry run no longer reports
  how many rows it would write for a format nothing can write.
- **Excluding every field of an entity other than the first is refused before
  the run.** It used to generate everything and then fail at the end with a
  message telling you to report a defect in the tool, for something in your own
  recipe.
- **The record beside your data now says which settings came from the command
  line.** It recorded the value and not where it came from, so the file meant to
  explain why two runs of one recipe differ was silent about the commonest
  reason.
- **A field asked to be unique and also to be empty is refused.** It used to
  produce a column of empty values under a heading declaring every one of them
  different.
- **file_per_entity now says that it does nothing for CSV.** That form writes one
  file per table whatever the setting says. Only the JSON form reads it.
- **A manifest too large for this build to read back is now reported when it is
  written**, rather than when you later try to verify against it.
- **In the window, editing the separator no longer loses your open file.**
  Deleting the last character left the field briefly unusable, and the screen
  forgot which file was open and could not get it back.
- **Enter presses a button in the window, and not only the space bar.** A
  button nobody can press with the pointer, such as the one for a response
  record before a manifest is open, still ignores both.
- **Opening a response record with the manifest button now says so.** It used to
  list the fields a manifest envelope needs, one sentence each, which is true
  about the file and no help at all when you have opened the right file with the
  wrong button.
- **The window can now be used with a keyboard, and it shows you where you are.**
  Tab moves through the window in the order you read it, starting with the list
  of screens down the left. In the table of ordered damage, the arrow keys move
  from cell to cell and the cell you are on is drawn in bold and underlined, so
  it stays visible without taking away the colour that says what the entry means.
  The name of the screen you are on carries a mark of its own, so it is no longer
  confused with the name the keyboard happens to be on. One limit is worth
  knowing: the two panels of the take and corrupt screen still scroll with the
  wheel rather than with the keyboard.
- **The About screen can be read without a mouse.** Tab reaches the text itself
  and the arrows, Page Up, Page Down, Home and End move through it, with a page
  keeping the line you were on. Three quarters of that screen used to need a
  mouse to reach at all.
- **The take and corrupt screen can be used with a keyboard too.** Tab now
  reaches the button that chooses a file first, which is where it stands on the
  screen, then every setting of the run in the order they are listed, and the
  button that damages the file last. Each kind of damage is its own stop on the
  way and the space bar ticks it. When you start a run, the keyboard moves to
  the way to stop it, and when the run ends it moves back to the button that
  damages the file, so you never lose your place to a control that has left the
  screen. Starting a run with the mouse leaves the keyboard alone, and so does
  walking off to another screen while a run is going on.
- **A text field shows that the keyboard is in it.** Its outline turns pale blue
  and is drawn a little thicker, so it stands apart from the fields around it
  rather than shading from one grey into another. The same pale blue marks a
  tick box you have ticked.
- **A manifest this tool wrote could not always be read back by this tool.**
  Damage spanning several records, and damage located by a path inside a nested
  structure, lost part of their location when the manifest was read, so `verify`
  refused a manifest that `generate` had just written. A third case was quieter
  and worse: damage to the bytes of a file lost the offset saying where in the
  file it was, and nothing reported anything. If you have manifests produced by
  an earlier build, they were written correctly and can now be read.


### BREAKING

- **Exit code 3 now means any document you gave the run is not valid**, not
  only a recipe or a policy. A manifest or a response record that cannot be
  read used to end with code 5, which is the code for a recipe that is valid
  and cannot be satisfied. The three codes now line up: 6 means the bytes could
  not be read, 3 means the bytes are not a valid document, 5 means the document
  is valid and what it asks for cannot be done. A CI branching on 3 keeps
  working. A CI branching on 5 to catch an unreadable manifest has to move.
- **`verify` refuses a manifest or a response record written to a major format
  version this build does not speak.** It used to compare them and report on
  your application anyway, whatever the version said, including versions that
  are not numbers. A record from a newer MINOR version is still read, and the
  run now says so, because whatever that version added is invisible here.
- **A recipe whose tables ask for more rows in total than this build can count
  is refused.** Two tables that each fit could add up to a number that does
  not, and the run then reported a negative row count and succeeded. Only
  recipes no run could ever have finished are affected.

- **A number in a recipe must be written in plain decimal digits.** Writing
  `count: 010` used to produce eight rows, because YAML reads a leading nought
  as octal. It now stops the run and tells you that `010` is read as `8`.
  Leading noughts, `0x`, `0o`, `0b` and underscores between digits are all
  refused rather than guessed at. If you meant ten, write `10`.

### Security

- **This build now compiles with go1.26.6 and a newer `golang.org/x/net`.**
  Five vulnerabilities in the Go standard library and one in `x/net` were
  reachable from the window, through the toolkit it draws with. The command
  line tool is unaffected in what it does, and nothing about how you run it
  changes.

### Fixed

- **A trial run no longer says it wrote anything.** `generate --dry-run` ended
  with a line reading `2 tables, 7 rows written`, which sent you looking for
  files nobody made. It now says `7 rows would be written`. The same line also
  wrote `1 tables` for a single table, and now agrees with itself.
- **Two runs writing into the same directory no longer overwrite each other.**
  Starting a second run against a directory a first one is still writing now
  stops immediately and tells you so, instead of both finishing and one result
  quietly replacing the other. If a run is killed outright, the next one against
  that directory will name the leftover file to delete.

### Added

- **There is a window now, and it opens on the About screen.** `qadg-gui` shows
  the four versions this build carries, both licenses and what each one covers,
  every third party module with the day its license was read, and the notice
  that data you generate belongs to you. It answers the same questions as
  `--license` and `--version`, for people who would rather look than type.
- **The window is a separate program from the command line tool.** The command
  line binary is unchanged and still builds without a C compiler, so nothing
  about your pipeline changes. The window needs a C toolchain and OpenGL,
  because the toolkit it draws with does.
- **One dark theme, with every colour measured against the contrast standard
  rather than chosen by eye.** Colour is never the only thing carrying a
  meaning, which matters if you do not distinguish some of them.

- **`qadg verify` compares a manifest against what your application actually
  did.** Give it the manifest a run wrote and a record of how your system
  reacted, and it reports where the two disagree. It is the only command that
  fails because of your application rather than because of itself, and the only
  one that returns exit code 4.
- **Four kinds of disagreement, kept apart because they mean different things.**
  A value taken that should have been refused is a hole in validation. A value
  refused that should have been taken is validation blocking real users. A value
  taken in silence where a complaint was expected is the third. Records your
  system never reacted to at all are the fourth, and no check of one row against
  one verdict finds those.
- **A disagreement on a soft expectation is reported and does not fail the run.**
  Soft means the right reaction depends on your rules rather than on the data,
  so failing on one would be this tool asserting something it cannot know.
- **`stored` tells a value that survived from one quietly changed.** A system
  that takes a row, turns text into zero and cuts a long string to fit did not
  accept your value, it accepted a different one, and no status code says so.
  The field is optional, and when it is missing the report says which questions
  it could not answer rather than reporting a clean result.
- The format you produce is specified in `spec/application-response.md`. It is
  the one format this tool only ever reads.
- **Rows expecting a warning are no longer written to the Gherkin export.**
  What counts as a complaint is knowledge about your application, so a scenario
  had nothing to assert against such a row. They are counted in the file
  instead, and the JSON manifest carries them as it always did.

- **`qadg corrupt` now works out which of your columns go together, and damages
  that.** When a column of national identifiers agrees with a neighbouring
  column on the gender or the date it encodes, on every record, the run says so
  and the two defects that contradict a neighbouring column start working on a
  file with no recipe. Measured on a file of sixty records: eighteen defects
  applied where there had been none.
- **It does not need to know what your column is called or what its values
  mean.** A column standing one for one against the encoded gender is tied to
  the identifier whether it holds `M` and `F`, `K` and `M`, or two words in any
  other language. What is checked is the mapping, not the vocabulary.
- **The evidence is counted in how easily the agreement could have been a
  coincidence, not in how many records there are.** A column agreeing with a
  seven to one split of genders agrees by accident one time in four, so it is
  refused. The run prints the count behind every relation it states, and the
  manifest carries it.

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
  every verb this build carries and says plainly that this build implements a
  small part of the corruption catalog.
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
