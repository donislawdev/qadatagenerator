#!/usr/bin/env python3
"""Reports the pages of the website whose translation is behind the English.

Every page in a language other than English carries, in its front matter, the
git blob id of the English file it was translated from:

    translatedFrom: 791211fd04eb96f56fe37c80a2da470a717b7ddb

and a language's file of interface words carries the same on a comment line,
naming the English file of words:

    # translatedFrom: de7ca11343f740c443bbce46032cf6dcc7361720

When an English file changes, its blob id changes. This names every
translation made from an older one, with the git command that shows what
changed in English since then. Translate that change, then write the id this
prints as the new translatedFrom.

It reports and never fails, because a translation a little behind the English
is still a page in the reader's language, and holding back every language
over one late page would cost readers more than it saves. Inside GitHub
Actions each finding is also a warning on the run, so it shows without opening
the log.

The blob id is git's own: the SHA-1 of "blob <size>", a zero byte and the
file's bytes. Content is checked out with LF on every platform
(.gitattributes), so the id computed here from the working tree is the one git
stores, and the old id in a stamp is one git can show.

What it cannot see: whether a translation says what the English says. It only
knows which English it was made from.

Usage:
    python .github/scripts/translations.py --config web/hugo.toml
"""
from __future__ import annotations

import argparse
import hashlib
import os
import re
import sys
import tomllib
from pathlib import Path

#: The stamp in the front matter of a page.
PAGE_STAMP = re.compile(r"^translatedFrom:\s*([0-9a-f]{40})\s*$", re.MULTILINE)
#: The stamp on a comment line of a language's file of interface words.
WORDS_STAMP = re.compile(r"^#\s*translatedFrom:\s*([0-9a-f]{40})\s*$", re.MULTILINE)


def blob_id(data: bytes) -> str:
    """The id git gives a file holding these bytes."""
    return hashlib.sha1(b"blob %d\0" % len(data) + data).hexdigest()


def front_matter(text: str) -> str:
    """The YAML between the first two lines of three dashes, or nothing."""
    text = text.replace("\r\n", "\n")
    if not text.startswith("---\n"):
        return ""
    end = text.find("\n---", 3)
    return text[4:end] if end != -1 else ""


def check(translation: Path, english: Path, stamp: re.Pattern[str], text: str, line: str) -> list[tuple[Path, str]]:
    """What is wrong with one translation, each with the file it is about. Line is how a stamp is written in it."""
    if not english.is_file():
        return [(translation, f"{translation.as_posix()} has no English file at {english.as_posix()} to be a translation of")]
    now = blob_id(english.read_bytes())
    found = stamp.search(text)
    if not found:
        return [(translation, f"{translation.as_posix()} does not say which English it was translated from. "
                              f"Once it says what {english.as_posix()} says, add {line}{now}")]
    if found.group(1) != now:
        return [(translation, f"{translation.as_posix()} was translated from an older {english.as_posix()}. "
                              f"See what changed with git diff {found.group(1)} {now}, "
                              f"translate that, then write {line}{now}")]
    return []


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--config", required=True, type=Path, help="the site's hugo.toml")
    args = parser.parse_args(argv)

    try:
        with args.config.open("rb") as config:
            settings = tomllib.load(config)
    except (OSError, tomllib.TOMLDecodeError) as problem:
        print(f"cannot read {args.config}: {problem}", file=sys.stderr)
        return 2
    site = args.config.parent
    default = settings.get("defaultContentLanguage", "en")
    languages = settings.get("languages", {})
    if default not in languages:
        print(f"{args.config} has no [languages.{default}]", file=sys.stderr)
        return 2
    english = site / languages[default]["contentDir"]
    words = site / "i18n"

    findings: list[tuple[Path, str]] = []
    checked = 0
    for language in sorted(languages):
        if language == default:
            continue
        content = site / languages[language]["contentDir"]
        for page in sorted(content.rglob("*.md")):
            checked += 1
            text = front_matter(page.read_text(encoding="utf-8"))
            findings += check(page, english / page.relative_to(content), PAGE_STAMP, text, "translatedFrom: ")
        own_words = words / f"{language}.toml"
        if own_words.is_file():
            checked += 1
            text = own_words.read_text(encoding="utf-8")
            findings += check(own_words, words / f"{default}.toml", WORDS_STAMP, text, "# translatedFrom: ")

    in_actions = os.environ.get("GITHUB_ACTIONS") == "true"
    for path, finding in findings:
        print(finding)
        if in_actions:
            print(f"::warning file={path.as_posix()},title=Translation behind the English::{finding}")
    print(f"translations: {checked} checked, {len(findings)} to look at")
    return 0


if __name__ == "__main__":
    sys.exit(main())
