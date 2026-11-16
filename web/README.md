# The project website

The source of `qadatagenerator.donislawdev.com`, built with [Hugo](https://gohugo.io/). It is the
product page and the documentation, in English, and in more languages as each one is finished.

## What it may say

Only what the program has. What the program can say about itself, the site reads from the program
or from files a test holds to it, and never writes by hand:

| What | From | Kept true by |
|---|---|---|
| every command with its flags, every exit code, every generator and where it can be filled, every kind of damage this build carries out with its example and its expected answer, and the version | `data/facts/cli.json` | `go test ./internal/cli -run TestTheSiteFactsAreWhatTheProgramSays` |
| the name the window gives each group of damage | `data/facts/window.json` | `go test ./internal/gui/damageparts -run TestTheWindowFactsAreWhatTheWindowSays` |
| the recipe, the commands and what they printed, the first rows of the files, the entries of the manifest, the record of what an application did, the table of recipe keys and the commands that build the program | `../README.md`, mounted, each block found by the marker or heading above it | `go test ./internal/cli -run TestTheReadme` runs every marked block |
| the six values of the first screen | `../.github/social-preview/says.txt`, mounted | `go test ./internal/gui/socialcard` |
| the oldest Go a build from source takes | `../go.mod`, mounted | the module itself |
| the pictures of the window, on the home page and on The window | `assets/window/`, drawn by the program from the data of the README, with the list of texts each one shows beside it | `go test ./internal/gui/screens` compares the texts everywhere and the pixels on windows/amd64 |
| the ready-made recipes, offered for download and shown on the pages of Examples, and the recipes of `../examples/identifiers/` and `../examples/guides/` on the page of each identifier and each guide | `../examples/`, mounted | `go test ./internal/cli -run TestTheSiteShowsWhatTheProgramDoes` runs each one |
| what a guide says this version does not yet damage | `notYet` in the guide's front matter, against `data/facts/cli.json` | the build stops on the day the program damages in a group a guide names there |
| how many kinds of damage, generators and locales a sentence names | `{{< count "damage" >}}`, `"generators"` or `"locales"`, counted in `data/facts/cli.json` | the facts test above, and a name the shortcode does not know stops the build |
| what each recipe printed, the first rows of the files it wrote, the tables with their rows, and its manifest | `data/runs.json`, `data/examples/`, `assets/example-runs/` | the same test, which writes them |
| any other command a page shows, with what it printed | `{{< run "qadg ..." >}}` in the page, `data/runs.json` | the same test finds every one in `content/` and runs it |

When the program changes, the facts and runs tests fail until their files are written again:

```
go test ./internal/cli -run TestTheSiteFactsAreWhatTheProgramSays -update
go test ./internal/gui/damageparts -run TestTheWindowFactsAreWhatTheWindowSays -update
go test ./internal/cli -run TestTheSiteShowsWhatTheProgramDoes -update
```

Then read the diff of `data/facts`, `data/runs.json`, `data/examples` and `assets/example-runs` before
committing it. A manifest is compared without the time of its run, so a run that changed nothing
else leaves it as it is.

### Examples and commands outside the README

A command a page shows is written once, as `{{< run "qadg quick pesel --count 3" >}}`, with no
quotes inside it. Run the test above with `-update` and the page shows the command and what the
program printed, the same in every language. A command `data/runs.json` does not have stops the build.

Each file in `../examples/` has a page of the same name in every language,
`content/<language>/examples/<name>.md`, and the build stops on a recipe with no page, a page with no
recipe, or a recipe the test has not run. The page holds only the sentences. The facts above it, the
downloads, the recipe, its run, the first rows of what it writes and each value it damaged come from
the recipe and from what the test saw it make, and the list of examples puts each one among the data
to take or the data with damage by whether its recipe has a `policy`.

A page of another part of the site can have its own recipe in a folder of `../examples/` named after
that part. Each page of Identifiers, `content/<language>/identifiers/<name>.md`, has
`../examples/identifiers/<name>.yaml`. The page holds what the number is, how its check digit works,
a `run` of `qadg quick` and what to do with a broken one. Below that the template shows the recipe,
its run and every value it damaged, and the `sources` of the page's front matter. The build stops on
a recipe with no page, a recipe that damaged nothing, a page with no sources, and a page the table on
the first page of Identifiers does not link to. A folder goes one level deep, and two recipes may not
share a file name, because a reader saves a recipe under its own name.

A guide, `content/<language>/guides/<name>.md`, may have `../examples/guides/<name>.yaml`, and puts a
part of it where its words want it with `{{< example "recipe" >}}`. The parts are `downloads`,
`recipe`, `run`, `writes` (the first rows of every file) and `damage` (every value it damaged), the
same pieces the pages of Examples show, and the guide writes the heading above each. A guide that says
this version does not yet damage something names the group in `notYet`, as `notYet: [L5]`, and the
build stops once the program does damage in it, so the sentence has to be rewritten. The build also
stops on a recipe with no guide, a guide the first page of Guides does not link to, a part it does
not know and the damage of a recipe that orders none. Sources in the front matter are listed at the end
of a guide, as on the page of an identifier and on any other page that names them.

The pages of Compared, `content/<language>/compare/<name>.md`, set the program beside another tool.
Every fact about that tool names its source and the day it was read, in the `sources` of the page,
and a page says nothing is missing from the other tool unless a list it read says so, naming the
list. Those facts go out of date with no change here, so read each page against its sources again
before a release. What the program has comes from the facts above, through `count` where a sentence
needs a number.

In the list of what a recipe damaged, a value written in another Unicode form shows each combining
mark as its escape, because drawn on its letter it looks like no change. Only that damage is spelt
out: in many scripts a mark is an ordinary part of a word. A mark `layouts/_partials/marks-spelled.html`
does not list stops the build.

A change to the window, or to what the program produces for the README, fails the test of the
pictures until they are drawn again, on Windows or Linux on amd64:

```
go test ./internal/gui/screens -update
```

That draws the stored trees of the window and the pictures together. Look at the pictures before
committing them. The site turns each into two WebP files at build time, at the size the window is
laid out at and at the size it was drawn at. Every language shows the same English pictures, and the
alt text and the caption of each are in the content of the page, so they are translated with it.

The home page lists the generators and the groups of damage on cards whose words are in its front
matter. A card naming a generator or a group the program does not have, and a generator or group no
card names, both stop the build.

## Building it

```
cd web
hugo --printI18nWarnings --panicOnWarning
python ../.github/scripts/site_check.py public --config hugo.toml
```

The version of Hugo is the one `.github/workflows/site.yml` pins with its digest. The build refuses,
rather than writing a page with a hole in it, when a page links to a path with no page, a language
file lacks a word, or the README no longer has a block where the site reads it. `site_check.py` then
reads every page it wrote: one `h1`, title and description lengths, canonical addresses, `hreflang`
that names back, the sitemap, every internal link and fragment, and nothing loaded from another host.

## Languages

English is served at the root and every other language under its own prefix, at the same paths:
`/pl/docs/recipes/` is the Polish page of `/docs/recipes/`. A language goes live only when every page
is translated: an English page under another language's address is a duplicate to a search engine.

A language lives in two places, and adding one changes no template:

- `content/<language>/`, with the same file names as `content/en/`, which is how Hugo pairs a page
  with its translations.
- `i18n/<language>.toml`, the words of the site's interface.

It takes an entry under `[languages]` in `hugo.toml`, with `disableKinds` naming `404` as well as
the site's own three kinds, because GitHub Pages serves one 404 page, at the root, and that page
links to the home page of every language. The header lists the languages under the name of the one
being read, so it stays one word however many there are.

What the program says stays English on every page, because the program speaks English: the commands
and what they print, the descriptions its help gives, the names of the window's screens and of its
groups of damage, and the pictures of the window. The two tables of the README that are prose rather
than output, the keys of a recipe and what to install before building the window, are translated in
the front matter of the page that shows them, under `readmeTables`, each row beside the English it
was made from. A row of the README with no translation stops the build.

### Keeping translations current

Each translated page names, as `translatedFrom` in its front matter, the git blob id of the English
file it was translated from, and each language's file of words names its English file the same way,
on a `# translatedFrom:` line. After a change to an English page, from `web/`:

```
python ../.github/scripts/translations.py --config hugo.toml
```

It names every translation made from an older English file, with the `git diff` that shows what
changed. Translate that, then write the id it prints. It reports and never stops the build, and the
site workflow shows each finding as a warning on its run.

## Publishing

`.github/workflows/site.yml` builds and checks the site on every pull request and publishes nothing
there. It publishes only once the repository variable `PUBLISH_SITE` is `true`, and then after CI
has passed on `main`, for the commit CI passed on. Starting the workflow by hand from `main`
publishes `main` again. GitHub Pages serves the site from the address in its settings, and
`static/CNAME` holds the same address.
