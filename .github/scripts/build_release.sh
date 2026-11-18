#!/usr/bin/env bash
# Build every release target of the command line from this checkout, take the
# window's builds the three runners made, write one bill of materials over all
# eight binaries, and pack each with the files that travel with it.
#
#   bash .github/scripts/build_release.sh <version> <directory outside the checkout> <window builds>
#
# <window builds> is the directory the release workflow downloads the window
# job's builds into: one directory for each target, qadg-gui_<os>_<arch>, each
# holding what .github/scripts/build_window.sh left on that runner. The window
# is built with cgo on the system it is for (that script says why) and packed
# here, so the command line in the window's archive is the very file the
# command line's archive carries, and every archive is checked in one place.
#
# Without it this makes the five archives of the command line and nothing
# else, and says that this is not a release: every release carries the window
# as well (DEC-324). The release workflow always gives it, and a guard in
# internal/guard holds the workflow to that.
#
# Run from the root of a clean checkout. The release workflow runs exactly this,
# and so can anybody with Go, zip and a Unix shell, which is the reason it is a
# script rather than lines in a workflow: a build that only a tag can start is
# a build nobody tries before the tag.
#
# What it leaves in the directory it is given:
#
#   dist/qadg_<version>_<os>_<arch>.zip            the command line, Windows
#   dist/qadg_<version>_<os>_<arch>.tar.gz         the command line, Linux and macOS
#   dist/qadg-gui_<version>_<os>_<arch>.zip        the window, Windows
#   dist/qadg-gui_<version>_<os>_<arch>.tar.gz     the window, Linux and macOS
#   dist/verify-qadg_<version>.spdx.json           the bill of materials of all of them
#   work/                                          the binaries and what was packed
#
# The names of the archives are a public contract from the first release
# (DEC-324 point 2). The bill of materials is named verify- rather than qadg-
# because GitHub lists the files of a release by name and by nothing else, and
# the prefix puts the files a person checks a download with together, after the
# archives they check.
#
# NOTHING IS WRITTEN INSIDE THE CHECKOUT, and that is a rule rather than tidiness.
# Go records in every binary whether the tree it was built from had changes that
# are not committed, and cmd/sbom refuses a build that says it had. An untracked
# file counts. So an archive written into the checkout after the first build
# would make the second one a build of no commit, and the bill of materials
# would refuse the release.
#
# Adapted on 2026-11-18 from the same script in Cert Doctor, a project of the
# same owner under the same licence.
set -euo pipefail

die() {
  echo "build_release: $*" >&2
  exit 1
}

if [ "$#" -lt 2 ] || [ "$#" -gt 3 ]; then
  echo "usage: build_release.sh <version> <directory outside the checkout> <window builds>" >&2
  exit 2
fi
version="$1"
out="$2"
windows="${3:-}"

if [ ! -f go.mod ] || [ ! -d cmd/qadg ]; then
  die "run this from the root of the repository"
fi
printf '%s' "$version" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$' ||
  die "\"$version\" is not a version, and it goes into the name of every archive"

# The five builds DEC-324 point 1 ships the command line for, and the one list
# of them in this repository that a build reads. A guard in internal/guard
# holds it to the list the notices are checked against.
#
# No darwin/amd64: Macs with Intel processors are not supported, so nothing is
# built for them and nothing pretends to be.
targets=(
  windows/amd64
  windows/arm64
  linux/amd64
  linux/arm64
  darwin/arm64
)

# The three builds DEC-324 point 1 ships the window for, held by the same guard
# to the list the window's notices are checked against and to the matrix of
# the window job in the release workflow. Each is also a target of the command
# line, whose build for it goes into the window's archive.
window_targets=(
  windows/amd64
  linux/amd64
  darwin/arm64
)

if [ -z "$windows" ]; then
  echo "build_release: no window builds were given, so this makes the five archives of the command line" >&2
  echo "and nothing else. That is not a release: every release carries the window as well." >&2
fi

# Asked before anything is built, because the bill of materials asks it after,
# and a refusal at the end of a build reads like a fault in the build.
if [ -n "$(git status --porcelain)" ]; then
  git status --short >&2
  die "the checkout has changes that are not committed, and a build from it
records a commit that does not describe it. The bill of materials refuses such a
build. Commit them, or build from a clean clone."
fi
command -v zip >/dev/null 2>&1 || die "there is no zip here, and the archives for Windows are zip files"
command -v unzip >/dev/null 2>&1 || die "there is no unzip here, and what went into a zip file is asked of the file"
command -v sha256sum >/dev/null 2>&1 || die "there is no sha256sum here, so the software renderer could not be checked"

mkdir -p "$out"
root="$(pwd -P)"
out="$(cd "$out" && pwd -P)"
case "$out/" in
  "$root/"*) die "$out is inside the checkout, and every file written there makes the
next build a build of no commit. Give a directory outside it." ;;
esac
if [ -n "$(ls -A "$out")" ]; then
  die "$out is not empty. What is in it would be packed or published with this release."
fi
dist="$out/dist"
work="$out/work"
mkdir -p "$dist" "$work/build" "$work/package"

# The software renderer's files, by the pin they were fetched by: path in the
# archive, sum and size. The window job checked them against it on the runner,
# and they are checked again here, after the trip through the workflow's
# artefact storage, because these are the bytes that are packed.
renderer=()
while IFS= read -r line; do
  renderer+=("$line")
done < <(grep '^file=' .github/mesa-dist-win | cut -d= -f2-)
[ "${#renderer[@]}" -gt 0 ] || die ".github/mesa-dist-win names no file of the software renderer"

# The notices that travel in the same folder, which fetch_software_renderer.sh
# copies from .github/opengl. The bill of materials reads the same folder with
# -renderer below, so the folder a person gets and the document that describes
# it are held to one copy of the notices here.
renderer_notices=".github/opengl"
travelling=()
for notice in "$renderer_notices"/*; do
  travelling+=("$(basename "$notice")")
done
[ "${#travelling[@]}" -gt 0 ] || die "$renderer_notices holds no notices of the software renderer"

# take_window copies one runner's build of the window into work/, and refuses
# it unless it is exactly what that runner was to leave: the binary built for
# that target with cgo from a commit, and on Windows the software renderer
# beside it, file for file as pinned, with its notices byte for byte as this
# commit has them. And nothing else, because what is in the directory is what
# would be packed.
take_window() {
  local os="$1" arch="$2" from to binary expected got info line inside sum size name
  from="$windows/qadg-gui_${os}_${arch}"
  to="$work/build/window_${os}_${arch}"
  binary="qadg-gui"
  if [ "$os" = "windows" ]; then
    binary="qadg-gui.exe"
  fi
  [ -f "$from/$binary" ] || die "$from has no $binary, so there is no window for ${os}/${arch}"

  expected="$binary"
  if [ "$os" = "windows" ]; then
    for line in "${renderer[@]}"; do
      read -r inside sum size <<<"$line"
      expected="$expected"$'\n'"opengl/$(basename "$inside")"
    done
    for name in "${travelling[@]}"; do
      expected="$expected"$'\n'"opengl/$name"
    done
  fi
  got="$(cd "$from" && find . -type f | sed 's#^\./##' | LC_ALL=C sort)"
  expected="$(printf '%s\n' "$expected" | LC_ALL=C sort)"
  [ "$got" = "$expected" ] || die "$from holds
$got
and the window for ${os}/${arch} is
$expected"

  mkdir -p "$to"
  cp -R "$from/." "$to/"
  # A workflow artefact keeps no file modes, so the binary arrives without the
  # bit that lets it run.
  chmod 755 "$to/$binary"

  info="$(go version -m "$to/$binary")"
  printf '%s\n' "$info" | grep -qE '^[[:space:]]+path[[:space:]]+github\.com/donislawdev/qadatagenerator/cmd/qadg-gui$' ||
    die "$from/$binary is not a build of cmd/qadg-gui"
  for want in "CGO_ENABLED=1" "vcs.modified=false" "GOOS=${os}" "GOARCH=${arch}"; do
    printf '%s\n' "$info" | grep -qE -- "[[:space:]]${want}$" || die "$from/$binary does not record $want"
  done

  if [ "$os" = "windows" ]; then
    for line in "${renderer[@]}"; do
      read -r inside sum size <<<"$line"
      name="$(basename "$inside")"
      (cd "$to/opengl" && echo "${sum}  ${name}" | sha256sum -c - >/dev/null) ||
        die "opengl/${name} from the window job is not the file .github/mesa-dist-win pins"
      [ "$(wc -c <"$to/opengl/$name" | tr -d '[:space:]')" = "$size" ] ||
        die "opengl/${name} from the window job is not $size bytes"
    done
    for name in "${travelling[@]}"; do
      cmp -s "$to/opengl/$name" "$renderer_notices/$name" ||
        die "opengl/${name} from the window job is not $renderer_notices/$name of this commit, and the
bill of materials describes the renderer from that folder"
    done
  fi
  echo "took the window for ${os}/${arch}"
}

window_binaries=()
if [ -n "$windows" ]; then
  [ -d "$windows" ] || die "$windows is not a directory of window builds"
  windows="$(cd "$windows" && pwd -P)"
  for target in "${window_targets[@]}"; do
    take_window "${target%/*}" "${target#*/}"
    binary="qadg-gui"
    if [ "${target%/*}" = "windows" ]; then
      binary="qadg-gui.exe"
    fi
    window_binaries+=("$work/build/window_${target%/*}_${target#*/}/$binary")
  done
fi

# Go calls it darwin and almost nobody else does. The name on a download is
# read by somebody deciding whether it is the file for their machine, so the
# label says macos (DEC-324 point 2). The compiler is still told darwin.
label_of() {
  case "$1" in
    darwin) echo "macos" ;;
    *) echo "$1" ;;
  esac
}

binaries=()
for target in "${targets[@]}"; do
  os="${target%/*}"
  arch="${target#*/}"
  binary="qadg"
  if [ "$os" = "windows" ]; then
    binary="qadg.exe"
  fi
  built="$work/build/${os}_${arch}/${binary}"

  # CGO_ENABLED=0 so the binary is static and runs in a container with no C
  # library at all, the promise the cross build in ci.yml keeps. -trimpath
  # keeps the build machine's directory names out of a file that is published.
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath -o "$built" ./cmd/qadg

  # What the toolchain was told, read back out of the binary rather than
  # trusted from the line above.
  info="$(go version -m "$built")"
  for want in "CGO_ENABLED=0" "vcs.modified=false" "GOOS=${os}" "GOARCH=${arch}"; do
    printf '%s\n' "$info" | grep -qE -- "[[:space:]]${want}$" || die "$target does not record $want"
  done
  binaries+=("$built")
  echo "built $target"
done

# The one build this machine can start, started. Everything above proves the
# binaries were made, and nothing above proves one of them runs.
host="$(go env GOHOSTOS)_$(go env GOHOSTARCH)"
started=""
for built in "${binaries[@]}"; do
  case "$built" in
    "$work/build/${host}/"*)
      "$built" --version >"$work/version.txt"
      said="$(head -n 1 "$work/version.txt")"
      [ "$said" = "qadg ${version}" ] ||
        die "the ${host} build says \"${said}\", and this release is ${version}"
      echo "the ${host} build starts and says ${said}"
      started="yes"
      ;;
  esac
done
[ -n "$started" ] || echo "build_release: this machine is ${host}, a target nothing is built for, so no build was started" >&2

# Written from every binary at once, so the document describes what was built
# rather than what the tree says would be. cmd/sbom refuses when a build links
# a module, a version or a toolchain the notices of its program do not name,
# or comes from another commit than the rest, and prints every disagreement.
# The software renderer is described from the folder its notices travel from,
# and cmd/sbom refuses it without the window for Windows, which it travels
# beside, so the flag goes with that build.
#
# Into work/ first and into dist/ only when it was written whole: a refusal
# would otherwise leave an empty document where the release's files are.
sbom=(go run ./cmd/sbom)
if [ -n "$windows" ]; then
  sbom+=(-renderer "$renderer_notices")
fi
go version -m "${binaries[@]}" "${window_binaries[@]}" | "${sbom[@]}" >"$work/bom.spdx.json"
mv "$work/bom.spdx.json" "$dist/verify-qadg_${version}.spdx.json"

# The licences and the notices travel with the binary because the licences in
# them ask for it, and the notices are also inside the binary, for the copy
# that leaves the archive. LICENSE-spec is there because the binary carries the
# corruption catalog, which is under CC BY 4.0. The readme is for the person
# who downloaded an archive and nothing else.
documents=(LICENSE LICENSE-spec THIRD-PARTY-NOTICES.md README.md)
window_documents=(LICENSE LICENSE-spec THIRD-PARTY-NOTICES.md THIRD-PARTY-NOTICES-GUI.md README.md)

# take copies files into a package and sets their mode, so an archive says
# the same of its files whichever machine packed it. A checkout seen through a
# Windows disk reads every file as executable (measured 2026-11-18 in a
# container), and a workflow artefact keeps no modes at all.
take() {
  local dir="$1" mode="$2" name
  shift 2
  for name in "$@"; do
    cp "$name" "$dir/"
    chmod "$mode" "$dir/$(basename "$name")"
  done
}

# pack writes one archive of a directory and checks what went in, asked of the
# archive and not of the directory, because the archive is what a person gets.
pack() {
  local os="$1" base="$2" from="$3" listed name
  shift 3
  if [ "$os" = "windows" ]; then
    (cd "$from" && zip -q -r -X "$dist/${base}.zip" .)
    listed="$(unzip -Z1 "$dist/${base}.zip")"
  else
    tar -czf "$dist/${base}.tar.gz" -C "$from" .
    listed="$(tar -tzf "$dist/${base}.tar.gz" | sed 's#^\./##')"
  fi
  for name in "$@"; do
    printf '%s\n' "$listed" | grep -qxF -- "$name" || die "${base} does not carry ${name}"
  done
  echo "packed ${base}"
}

for target in "${targets[@]}"; do
  os="${target%/*}"
  arch="${target#*/}"
  base="qadg_${version}_$(label_of "$os")_${arch}"
  dir="$work/package/$base"
  mkdir -p "$dir"

  binary="qadg"
  if [ "$os" = "windows" ]; then
    binary="qadg.exe"
  fi
  take "$dir" 755 "$work/build/${os}_${arch}/${binary}"
  wanted=("${documents[@]}" "$binary")

  # macOS gets a bundle, and it is not decoration: a bare binary cannot carry a
  # notarisation ticket (make_app_bundle.sh says what was measured). A link
  # goes beside it, so a person still types ./qadg.
  if [ "$os" = "darwin" ]; then
    bash .github/scripts/make_app_bundle.sh "$dir" "$binary" "com.donislawdev.qadg" "$version" "$binary"
    wanted+=("${binary}.app/Contents/MacOS/${binary}" "${binary}.app/Contents/Info.plist"
      "${binary}.app/Contents/Resources/icon.icns")
  fi
  take "$dir" 644 "${documents[@]}"
  pack "$os" "$base" "$dir" "${wanted[@]}"
done

# The window's archives: the window, the command line of the same target, the
# software renderer on Windows, and the notices of both programs.
# THIRD-PARTY-NOTICES.md answers for the command line in it, and
# THIRD-PARTY-NOTICES-GUI.md for the window.
for target in "${window_targets[@]}"; do
  [ -n "$windows" ] || break
  os="${target%/*}"
  arch="${target#*/}"
  base="qadg-gui_${version}_$(label_of "$os")_${arch}"
  dir="$work/package/$base"
  mkdir -p "$dir"

  window="qadg-gui"
  binary="qadg"
  if [ "$os" = "windows" ]; then
    window="qadg-gui.exe"
    binary="qadg.exe"
  fi
  take "$dir" 755 "$work/build/window_${os}_${arch}/${window}"
  # The command line's own build for this target, the file its archive
  # carries, rather than a second build of the same thing.
  take "$dir" 755 "$work/build/${os}_${arch}/${binary}"
  wanted=("${window_documents[@]}" "$window" "$binary")
  if [ "$os" = "windows" ]; then
    mkdir "$dir/opengl"
    take "$dir/opengl" 644 "$work/build/window_${os}_${arch}/opengl/"*
    for line in "${renderer[@]}"; do
      read -r inside _ _ <<<"$line"
      wanted+=("opengl/$(basename "$inside")")
    done
    for name in "${travelling[@]}"; do
      wanted+=("opengl/$name")
    done
  fi

  # One bundle with both programs in it, under the window's identifier, which
  # is its own and not the toolkit's (DEC-325), and a link to each program
  # beside it.
  if [ "$os" = "darwin" ]; then
    bash .github/scripts/make_app_bundle.sh "$dir" "$window" "com.donislawdev.qadg-gui" "$version" "QA Data Gen" "$binary"
    wanted+=("${window}.app/Contents/MacOS/${window}" "${window}.app/Contents/MacOS/${binary}"
      "${window}.app/Contents/Info.plist" "${window}.app/Contents/Resources/icon.icns")
  fi
  take "$dir" 644 "${window_documents[@]}"
  pack "$os" "$base" "$dir" "${wanted[@]}"
done

ls -l "$dist"
if [ -z "$windows" ]; then
  echo "build_release: the five archives of the command line and no window, which is not a release." >&2
fi
