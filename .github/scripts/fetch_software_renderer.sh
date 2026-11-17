#!/usr/bin/env bash
# Put the software renderer beside the window binary on Windows: the two files
# of Mesa's llvmpipe the window loads when the graphics driver gives it no
# OpenGL, such as in a virtual machine without 3D acceleration, over a remote
# desktop, or on a server.
#
#   bash .github/scripts/fetch_software_renderer.sh <directory qadg-gui.exe is in>
#
# Run it from the root of the repository. What it leaves is <directory>/opengl
# with exactly the files .github/mesa-dist-win names and the notices in
# .github/opengl, and nothing else anywhere. The window asks for the two files
# by that path and by no other, see cmd/qadg-gui/software.go.
#
# THE ARCHIVE IS CHECKED BEFORE ANYTHING IS UNPACKED. A download that does not
# match its sum is an archive nobody read, and unpacking it first would put two
# files nobody read next to a program that loads them. After that each file is
# checked on its own, by sum and by size: the archive's sum already covers
# them, and these are the numbers the notices print, so a person can compare
# the folder they have with what was reviewed without downloading the archive.
#
# The notices go into the same folder because the archive carries no licence
# file, and the licences of what is compiled into the two files ask to travel
# with them.
#
# The version, the sums and the sizes come from .github/mesa-dist-win and
# nowhere else. A guard in internal/guard holds that file to
# .github/opengl/NOTICES.md and to the names the window asks for.
set -euo pipefail

die() {
  echo "fetch_software_renderer: $*" >&2
  exit 1
}

if [ "$#" -ne 1 ]; then
  echo "usage: fetch_software_renderer.sh <directory qadg-gui.exe is in>" >&2
  exit 2
fi
into="$1"
pin=".github/mesa-dist-win"
notices=".github/opengl"
project="pal1000/mesa-dist-win"

[ -f "$pin" ] || die "run this from the root of the repository, there is no $pin here"
[ -f "$notices/NOTICES.md" ] || die "run this from the root of the repository, there is no $notices/NOTICES.md here"
[ -d "$into" ] || die "$into is not a directory"

# One value for a key, or a refusal. A key given twice would let the line a
# reviewer read and the line this script used be different lines.
value_of() {
  local key="$1" found
  found="$(grep -c "^${key}=" "$pin" || true)"
  [ "$found" = "1" ] || die "$pin names ${key} ${found} time(s), and it is named once"
  grep "^${key}=" "$pin" | cut -d= -f2-
}
version="$(value_of version)"
archive="$(value_of archive)"
sha256="$(value_of sha256)"

# Each value ends up in a path or in an address, so its shape is asked before
# it is used.
printf '%s' "$version" | grep -qE '^[0-9]+(\.[0-9]+)+$' ||
  die "$pin gives the version as \"$version\", which is not a version"
printf '%s' "$archive" | grep -qE '^[A-Za-z0-9][A-Za-z0-9._-]*\.7z$' ||
  die "$pin gives the archive as \"$archive\", which is not the name of a 7z archive"
printf '%s' "$sha256" | grep -qE '^[0-9a-f]{64}$' ||
  die "$pin gives the archive's sum as \"$sha256\", which is not a SHA-256"

files=()
while IFS= read -r line; do
  files+=("$line")
done < <(grep '^file=' "$pin" | cut -d= -f2-)
[ "${#files[@]}" -gt 0 ] || die "$pin names no file to take out of the archive"
for line in "${files[@]}"; do
  printf '%s' "$line" | grep -qE '^[A-Za-z0-9][A-Za-z0-9._-]*(/[A-Za-z0-9][A-Za-z0-9._-]*)* [0-9a-f]{64} [0-9]+$' ||
    die "$pin has the line \"file=$line\", and a file is a path, a SHA-256 and a size"
done

command -v sha256sum >/dev/null 2>&1 || die "there is no sha256sum here, so nothing could be checked"
command -v curl >/dev/null 2>&1 || die "there is no curl here, so nothing could be downloaded"

# 7z where it is installed, as on the Windows runner, and otherwise bsdtar.
# Windows 10 and later carry bsdtar as System32\tar.exe, which Git Bash hides
# behind a GNU tar of its own, so that one is asked for by its path. GNU tar
# does not read this format, so a tar is taken only when it says it is bsdtar.
unpack=""
kind=""
for tool in 7z 7zz bsdtar tar "${SYSTEMROOT:+$SYSTEMROOT/System32/tar.exe}"; do
  [ -n "$tool" ] || continue
  command -v "$tool" >/dev/null 2>&1 || continue
  case "$tool" in
    7z | 7zz)
      unpack="$tool"
      kind="7z"
      ;;
    *)
      if "$tool" --version 2>/dev/null | grep -q '^bsdtar'; then
        unpack="$tool"
        kind="bsdtar"
      fi
      ;;
  esac
  [ -z "$unpack" ] || break
done
[ -n "$unpack" ] || die "neither 7z nor bsdtar is installed, and the archive is a 7z"

dest="$into/opengl"
if [ -e "$dest" ]; then
  die "$dest is already there. What is in it was not put there by this run, so it was not checked by it."
fi

# Beside the target rather than in the system's temporary directory, so the
# download and the files it becomes are on one disk and under one owner.
fetched="$(mktemp -d "$into/.software-renderer.XXXXXX")"
trap 'rm -rf "$fetched"' EXIT

url="https://github.com/${project}/releases/download/${version}/${archive}"
echo "fetching ${url}"
curl --silent --show-error --fail --location --retry 3 --output "$fetched/$archive" "$url" ||
  die "could not download $url"
(cd "$fetched" && echo "${sha256}  ${archive}" | sha256sum -c - >/dev/null) ||
  die "$archive does not have the SHA-256 $pin names, so it was not unpacked"
echo "${archive}: the sum is the pinned one"
echo "unpacking with ${unpack}"

# Unpacked and checked in the same hidden directory, and moved into place only
# when every file has passed, so a refusal half way leaves no opengl folder that
# looks like a finished one.
stage="$fetched/opengl"
mkdir "$fetched/out" "$stage"
for line in "${files[@]}"; do
  read -r inside sum size <<<"$line"
  name="$(basename "$inside")"
  case "$kind" in
    bsdtar)
      "$unpack" -xf "$fetched/$archive" -C "$fetched/out" "$inside" ||
        die "$archive does not unpack $inside"
      [ -f "$fetched/out/$inside" ] || die "$archive holds no $inside"
      mv "$fetched/out/$inside" "$stage/$name"
      ;;
    *)
      "$unpack" e -bso0 -bsp0 -o"$stage" "$fetched/$archive" "$inside" ||
        die "$archive does not unpack $inside"
      ;;
  esac
  [ -f "$stage/$name" ] || die "$archive holds no $inside"
  (cd "$stage" && echo "${sum}  ${name}" | sha256sum -c - >/dev/null) ||
    die "$inside does not have the SHA-256 $pin names"
  got="$(wc -c <"$stage/$name" | tr -d '[:space:]')"
  [ "$got" = "$size" ] || die "$inside is $got bytes, and $pin names $size"
  echo "opengl/${name}: ${size} bytes, the sum is the pinned one"
done

# The notices go in beside the two files. None of them may take the name of a
# file of the renderer, or it would replace a file that was checked.
travelling=0
for notice in "$notices"/*; do
  name="$(basename "$notice")"
  [ ! -e "$stage/$name" ] || die "$notices/$name has the name of a file of the renderer"
  cp "$notice" "$stage/$name"
  travelling=$((travelling + 1))
done

# As many files as the pin and the notices name and no more, or the folder is
# not what the notices describe.
count="$(find "$stage" -type f | wc -l | tr -d '[:space:]')"
[ "$count" = "$((${#files[@]} + travelling))" ] ||
  die "the folder holds $count file(s), and $pin with $notices names $((${#files[@]} + travelling))"
mv "$stage" "$dest"
echo "the software renderer is in $dest"
