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

When the program changes, the two facts tests fail until the facts are written again:

```
go test ./internal/cli -run TestTheSiteFactsAreWhatTheProgramSays -update
go test ./internal/gui/damageparts -run TestTheWindowFactsAreWhatTheWindowSays -update
```

Then read the diff of `data/facts` before committing it.

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

English is served at the root and every other language under its own prefix. A language goes live
only when every page is translated: an English page under another language's address is a duplicate
to a search engine.

A language lives in two places, `content/<language>/` with the same file names as `content/en/`
(that is how Hugo pairs a page with its translations), and `i18n/<language>.toml`, the words of the
site's interface. It takes an entry under `[languages]` in `hugo.toml`, with `disableKinds` naming
`404` as well as the site's own three kinds, because GitHub Pages serves one 404 page, at the root.

What the program prints stays English on every page, because that is what the program prints.

## Publishing

`.github/workflows/site.yml` builds and checks the site on every pull request and publishes nothing
there. It publishes only once the repository variable `PUBLISH_SITE` is `true`, and then after CI
has passed on `main`, for the commit CI passed on. Starting the workflow by hand from `main`
publishes `main` again. GitHub Pages serves the site from the address in its settings, and
`static/CNAME` holds the same address.
