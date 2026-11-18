#!/usr/bin/env bash
# Build the window, qadg-gui, on the system it is for.
#
#   bash .github/scripts/build_window.sh <goos>/<goarch> <empty directory outside the checkout>
#
# Run from the root of a clean checkout, on a machine of the target it is asked
# for. The release workflow runs it on three runners, one for each target the
# window is released for (DEC-324 point 1), and hands what it leaves to
# build_release.sh, which packs it with the command line.
#
# What it leaves in the directory it is given:
#
#   qadg-gui         Linux and macOS
#   qadg-gui.exe     Windows
#   opengl/          Windows only: the software renderer and its notices, put
#                    there and checked by fetch_software_renderer.sh
#
# WHY ON THE MACHINE IT IS FOR, AND NOT CROSS COMPILED LIKE THE COMMAND LINE.
# The window reaches OpenGL and the windowing system through C, the GLFW
# library the toolkit compiles in, so it is built with cgo, and cgo builds for
# the system the C compiler is for. The command line has no C in it and is
# built for all five of its targets on one runner.
#
# Every build is read back before it is handed on: what the binary records of
# how it was built, and, from the linker's map, what the C compiler linked into
# it of its own, which window_toolchain.sh holds to .github/window-toolchain.
# That is the build that ships, so the reader runs on it and not only on the
# test build of ci.yml.
#
# NOTHING IS WRITTEN INSIDE THE CHECKOUT, for the reason build_release.sh gives:
# the binary records whether the tree had changes, an untracked file counts,
# and the bill of materials refuses a build of no commit.
#
# Adapted on 2026-11-18 from the same script in Cert Doctor, a project of the
# same owner under the same licence.
set -euo pipefail

die() {
  echo "build_window: $*" >&2
  exit 1
}

if [ "$#" -ne 2 ]; then
  echo "usage: build_window.sh <goos>/<goarch> <empty directory outside the checkout>" >&2
  exit 2
fi
target="$1"
out="$2"
goos="${target%/*}"
goarch="${target#*/}"

if [ ! -f go.mod ] || [ ! -d cmd/qadg-gui ]; then
  die "run this from the root of the repository"
fi

# Asked of the toolchain on this machine, because that is what the binary will
# be built for. A runner that is not the one the workflow names would build a
# window for another system under this one's name.
here="$(go env GOOS)/$(go env GOARCH)"
[ "$here" = "$target" ] || die "this machine builds for $here, and it was asked for $target"

if [ -n "$(git status --porcelain)" ]; then
  git status --short >&2
  die "the checkout has changes that are not committed, and a build from it
records a commit that does not describe it. The bill of materials refuses such a
build. Commit them, or build from a clean clone."
fi

mkdir -p "$out"
root="$(pwd -P)"
out="$(cd "$out" && pwd -P)"
case "$out/" in
  "$root/"*) die "$out is inside the checkout, and every file written there makes the
build a build of no commit. Give a directory outside it." ;;
esac
if [ -n "$(ls -A "$out")" ]; then
  die "$out is not empty. What is in it would be packed with this release."
fi

binary="qadg-gui"
if [ "$goos" = "windows" ]; then
  binary="qadg-gui.exe"
fi

# The linker's map of the build, which window_toolchain.sh reads below for
# what the C compiler linked in of its own. In a directory of its own outside
# the checkout and outside the output, so nothing of it is packed, and removed
# when this ends. The go command splits -ldflags at spaces, so a path with one
# would reach the linker in pieces.
maps="$(mktemp -d)"
trap 'rm -rf "$maps"' EXIT
case "$maps/" in "$root/"*) die "the temporary directory $maps is inside the checkout" ;; esac
map="$maps/qadg-gui.map"
if [ "$goos" = "windows" ]; then
  # The linker is a Windows program and reads a Windows path.
  map="$(cygpath -m "$map")"
fi
case "$map" in *" "*) die "the map would be written at $map, and the go command splits a path with a space" ;; esac
case "$goos" in
  darwin) ldflags="-extldflags=-Wl,-map,$map" ;;
  *) ldflags="-extldflags=-Wl,-Map=$map" ;;
esac

# CGO_ENABLED=1 named rather than left to the default, because the default is
# 0 wherever no C compiler is found, and the window does not compile without
# cgo. -trimpath keeps the build machine's directory names out of a file that
# is published, and leaves -ldflags, with the map's path in it, out of what the
# binary records.
CGO_ENABLED=1 go build -trimpath "-ldflags=$ldflags" -o "$out/$binary" ./cmd/qadg-gui

# What the C compiler linked in of its own, held to the list the window's
# notices were written for (DEC-327).
bash .github/scripts/window_toolchain.sh "$target" "$map" "$(go env CC)"

# What the toolchain was told and what it recorded, read back out of the
# binary rather than trusted from the line above.
info="$(go version -m "$out/$binary")"
printf '%s\n' "$info" | grep -qE '^[[:space:]]+path[[:space:]]+github\.com/donislawdev/qadatagenerator/cmd/qadg-gui$' ||
  die "$binary is not a build of cmd/qadg-gui"
for want in "CGO_ENABLED=1" "vcs.modified=false" "GOOS=$goos" "GOARCH=$goarch"; do
  printf '%s\n' "$info" | grep -qE -- "[[:space:]]${want}$" || die "$binary does not record $want"
done

if [ "$goos" = "windows" ]; then
  # A program for the Windows subsystem rather than the console one, which
  # cmd/qadg-gui/console_windows.go asks the linker for. Without it Windows
  # opens a console window behind the window and keeps it for as long as the
  # window is open. Read where it acts: the Subsystem field of the PE header,
  # 2 for the Windows subsystem and 3 for the console one. The PE header starts
  # at the offset stored at 0x3C, and Subsystem lies 92 bytes into it: past the
  # 4 byte signature, the 20 byte file header and 68 bytes of the optional
  # header.
  pe="$(od -An -tu4 --endian=little -j60 -N4 "$out/$binary" | tr -d ' ')"
  [ "$(od -An -tx1 -j"$pe" -N4 "$out/$binary" | tr -d ' ')" = "50450000" ] ||
    die "$binary has no PE header where its DOS header points"
  subsystem="$(od -An -tu2 --endian=little -j"$((pe + 92))" -N2 "$out/$binary" | tr -d ' ')"
  [ "$subsystem" = "2" ] ||
    die "$binary is for subsystem $subsystem, not 2: Windows would open a console window behind it"
fi
echo "built $binary for $target"

# The software renderer beside it, on Windows alone: Linux has Mesa in the
# system, and macOS has never offered the toolkit less OpenGL than it needs.
# Here rather than in the step that packs the archives, because 7z, which reads
# the archive Mesa is published in, is on the Windows runner (DEC-323).
if [ "$goos" = "windows" ]; then
  bash .github/scripts/fetch_software_renderer.sh "$out"
fi

ls -l "$out"
