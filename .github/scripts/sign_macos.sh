#!/usr/bin/env bash
# Sign, notarise and staple the macOS archives of a release. Runs ON THE MAC,
# copied there and started by .github/scripts/sign_release.py.
#
#   bash sign_macos.sh <directory> <certificate-sha256> <notarytool-profile>
#
# The Developer ID key lives in a keychain on the maintainer's Mac, and Apple's
# notary service only talks to credentials stored there. The Windows card and
# this key are on two different machines, so a release is signed in two places
# and the checksums are written once, at the end, over everything.
#
# Every macOS archive in <directory> is signed. The command line's holds
# qadg.app with the command line alone. The window's holds qadg-gui.app with
# the window and, beside it in Contents/MacOS, the command line, which travels
# with the window for the person who downloaded it (make_app_bundle.sh). A
# program inside a bundle that is not the bundle's own executable is code of its
# own, so it is signed first and the bundle after it, the inside out order
# Apple's codesign documentation gives.
#
# What it refuses, and why. Each refusal was measured in the maintainer's other
# Go project on 2026-08-28, on real hardware, and several are refusals because
# the failure is SILENT otherwise:
#
#  * a locked keychain. codesign then exits 1 with errSecInternalComponent and
#    leaves the file UNSIGNED. An ssh session is a security session of its own,
#    so unlocking the keychain in the Mac's own Terminal does NOT carry over.
#  * a missing notarytool profile, told apart from a locked keychain, which
#    reports the same error, so the message points at the right thing.
#  * a certificate that is not the pinned one, on the bundle or on any program
#    in it. A second Developer ID on the same machine signs just as willingly
#    and the release looks the same.
#  * a bundle whose signature does not verify strictly, every program in it
#    included, before anything is sent to Apple.
#  * notarisation that came back anything but Accepted.
#  * a staple that did not take, or a bundle Gatekeeper still refuses. Without
#    this last one the script would hand back an archive that fails on the
#    first machine that downloads it.
#
# Adapted on 2026-11-18 from the same script in Cert Doctor, a project of the
# same owner under the same licence, which signed its releases 0.2.0 and 0.3.0
# with it.
set -uo pipefail

if [ "$#" -ne 3 ]; then
  echo "usage: sign_macos.sh <directory> <certificate-sha256> <profile>" >&2
  exit 2
fi

DIR="$1"
PIN="$(echo "$2" | tr '[:lower:]' '[:upper:]')"
PROFILE="$3"

say() { echo "  $*"; }
die() {
  echo "sign_macos: $*" >&2
  exit 1
}

# Asked for on the terminal rather than taken as an argument or from the
# environment, because a password in argv is visible in ps to every process on
# the machine. It asks every time and does NOT claim to know whether it needed
# to: over ssh, show-keychain-info answers "User interaction is not allowed"
# even when the keychain is unlocked, so a test of it would report a locked
# keychain that is not locked. Unlocking an unlocked keychain changes nothing,
# and the proof that the key is reachable is codesign itself, further down.
unlock_keychain() {
  echo
  echo "  An ssh session is a security session of its own and cannot see whether"
  echo "  the keychain is unlocked, so it asks. If it is already unlocked, press"
  echo "  Enter. Otherwise this is your Mac password."
  echo
  if ! security unlock-keychain "$HOME/Library/Keychains/login.keychain-db"; then
    die "the keychain would not unlock, so nothing has been signed.
Run this by hand on the Mac and try again:
  security unlock-keychain ~/Library/Keychains/login.keychain-db"
  fi
  say "keychain reachable"
}

check_profile() {
  local out rc
  out="$(xcrun notarytool history --keychain-profile "$PROFILE" 2>&1)"
  rc=$?
  if [ $rc -eq 0 ]; then
    say "notarytool profile '$PROFILE' answers"
    return 0
  fi
  case "$out" in
    *keychainLocked*)
      die "the keychain locked itself again between two steps. Nothing has been signed."
      ;;
    *"No Keychain password item"*)
      die "there is no notarytool profile called '$PROFILE' on this Mac.
Create it once, in the Mac's own Terminal rather than over ssh:

  xcrun notarytool store-credentials \"$PROFILE\" --apple-id <your-apple-id> --team-id <your-team-id>

It asks for an APP-SPECIFIC password from appleid.apple.com, not the password
of the Apple account itself. A profile made for another project works too:
pass its name to sign_release.py with --notary-profile."
      ;;
    *"agreement"*)
      die "Apple refuses notarisation until an agreement on the developer account is
accepted, and nothing has been signed. Accept it at developer.apple.com and run
this again. Apple said:
$out"
      ;;
  esac
  die "notarytool would not answer: $out"
}

# codesign selects a certificate by SHA-1, and SHA-1 is not worth pinning
# anything to, so the pin is a SHA-256 and the selector is derived from it here.
# That also settles the ambiguity of the name, because the same certificate is
# installed in both the login and the system keychain.
find_identity() {
  local sha1
  sha1="$(security find-certificate -a -c "Developer ID Application" -Z 2>/dev/null |
    awk -v pin="$PIN" '
      /^SHA-256 hash:/ { s256 = $3 }
      /^SHA-1 hash:/   { if (s256 == pin) { print $3
                                            exit } }')"
  [ -n "$sha1" ] ||
    die "no Developer ID certificate on this Mac hashes to the pin $PIN.
A renewal is a DIFFERENT certificate, and its line in .github/signing-certificates.txt
has to move with it."
  security find-identity -v -p codesigning 2>/dev/null | grep -qi "$sha1" ||
    die "the pinned certificate is installed but has no private key here, so it cannot sign"
  echo "$sha1"
}

# The certificate that ACTUALLY signed a bundle or a program, read back out of
# it. Nothing here changes directory: the path arrives relative, and in the
# other project a cd before this made it name nothing, so a correctly signed
# bundle was refused as signed by a different certificate. A refusal that is
# right about there being a problem and wrong about what it is sends somebody
# to look at the wrong machine.
certificate_of() {
  local code="$1" tmp
  tmp="$(mktemp -d)"
  codesign -d --extract-certificates="$tmp/cert" "$code" >/dev/null 2>&1
  if [ ! -f "$tmp/cert0" ]; then
    rm -rf "$tmp"
    echo "none"
    return
  fi
  shasum -a 256 "$tmp/cert0" | awk '{ print toupper($1) }'
  rm -rf "$tmp"
}

# What a bundle's Info.plist says under a key.
plist_value() {
  /usr/libexec/PlistBuddy -c "Print :$2" "$1/Contents/Info.plist" 2>/dev/null
}

# A 64-bit Mach-O starts with cf fa ed fe. codesign would sign anything put in
# Contents/MacOS, and a file that is not code would carry its signature in
# extended attributes, which an archive does not keep.
is_macho() {
  [ "$(od -An -tx1 -N4 "$1" | tr -d ' \n')" = "cffaedfe" ]
}

sign_archive() {
  local archive="$1" identity="$2"
  local name work app main bundle_id program base actual line
  local nested=()
  name="$(basename "$archive")"
  work="${archive}.unpacked"
  rm -rf "$work"
  mkdir -p "$work"
  tar -xzf "$archive" -C "$work" || die "$name did not unpack"

  local apps=()
  while IFS= read -r line; do
    apps+=("$line")
  done < <(find "$work" -mindepth 1 -maxdepth 1 -name '*.app' -type d | sort)
  [ "${#apps[@]}" -eq 1 ] ||
    die "$name holds ${#apps[@]} .app bundles, and an archive holds one. The release build
makes it, because a bare binary cannot carry a notarisation ticket at all."
  app="${apps[0]}"

  main="$(plist_value "$app" CFBundleExecutable)"
  if [ -z "$main" ] || [ ! -f "$app/Contents/MacOS/$main" ]; then
    die "$(basename "$app") in $name names no executable it holds"
  fi
  bundle_id="$(plist_value "$app" CFBundleIdentifier)"
  [[ "$bundle_id" =~ ^[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)+$ ]] ||
    die "$(basename "$app") in $name has the bundle identifier '$bundle_id'"

  # Every program in the bundle has a link of its name beside it, so a person
  # still types ./qadg. A copy out there is refused by Gatekeeper, and ln makes
  # a copy silently where links do not work.
  for program in "$app"/Contents/MacOS/*; do
    [ -f "$program" ] || die "$(basename "$app") holds $(basename "$program") in Contents/MacOS, and it is not a file"
    base="$(basename "$program")"
    is_macho "$program" || die "$base in $(basename "$app") is not a 64-bit Mach-O program"
    [ -L "$work/$base" ] || die "$name does not carry a link called $base beside the bundle"
    if [ "$base" != "$main" ]; then
      nested+=("$program")
    fi
  done

  say "$name"
  # --options runtime is the hardened runtime, which notarisation requires.
  # --timestamp is what keeps the signature alive past the certificate.
  # --force because the Go linker signs every darwin/arm64 binary ad hoc.
  #
  # A program of its own keeps the identifier of the bundle it has of its own:
  # the bundle's identifier with the last part replaced by the program's name.
  # qadg inside qadg-gui.app (com.donislawdev.qadg-gui) is com.donislawdev.qadg,
  # the identifier of qadg.app, because it is that very program. Derived from
  # the bundle rather than written here, so the identifiers have one written
  # copy, in build_release.sh.
  for program in "${nested[@]+"${nested[@]}"}"; do
    base="$(basename "$program")"
    say "  signing $base inside $(basename "$app") as ${bundle_id%.*}.$base"
    codesign --force --options runtime --timestamp --identifier "${bundle_id%.*}.$base" \
      --sign "$identity" "$program" >/dev/null 2>&1 ||
      die "codesign failed on $base in $(basename "$app") in $name.
The usual cause is a locked keychain, which fails as errSecInternalComponent and
leaves the file unsigned. Unlock it on the Mac and run this again:
  security unlock-keychain ~/Library/Keychains/login.keychain-db"
  done

  say "  signing $(basename "$app")"
  codesign --force --options runtime --timestamp --sign "$identity" "$app" >/dev/null 2>&1 ||
    die "codesign failed on $(basename "$app") in $name.
The usual cause is a locked keychain, which fails as errSecInternalComponent and
leaves the file unsigned. Unlock it on the Mac and run this again:
  security unlock-keychain ~/Library/Keychains/login.keychain-db"

  # The bundle and every program in it, read back. --strict --deep is what
  # Apple's notary asks of a submission, asked here first, where the answer
  # comes in a second rather than after an upload.
  codesign --verify --strict --deep "$app" >/dev/null 2>&1 ||
    die "$(basename "$app") in $name does not verify after signing:
$(codesign --verify --strict --deep --verbose=2 "$app" 2>&1)"
  for code in "$app" "${nested[@]+"${nested[@]}"}"; do
    actual="$(certificate_of "$code")"
    [ "$actual" = "$PIN" ] ||
      die "$(basename "$code") in $name was signed by a DIFFERENT certificate
  expected $PIN
  got      $actual
Nothing has been handed back."
  done

  say "  notarising, this takes a few minutes"
  # Apple takes a zip, a pkg or a dmg, and not the tar.gz that is published, so
  # the zip exists only to carry the bundle there. What gets stapled and handed
  # back is the bundle itself. No cd here either, for the reason above.
  local zip="${work}.notarise.zip"
  ditto -c -k --keepParent "$app" "$zip" ||
    die "could not zip $(basename "$app") for notarisation"

  local out
  out="$(xcrun notarytool submit "$zip" --keychain-profile "$PROFILE" --wait 2>&1)"
  echo "$out" | grep -q "status: Accepted" ||
    die "notarisation did not come back Accepted for $name:
$out"
  rm -f "$zip"

  say "  stapling"
  xcrun stapler staple "$app" >/dev/null 2>&1 ||
    die "the ticket would not staple onto $(basename "$app") in $name"

  # The two questions worth asking after the fact, because a staple that did not
  # take and a bundle Gatekeeper refuses both look like success up to here.
  xcrun stapler validate "$app" >/dev/null 2>&1 ||
    die "$(basename "$app") does not validate against its own ticket"
  spctl -a -t exec "$app" >/dev/null 2>&1 ||
    die "Gatekeeper still refuses $(basename "$app") after signing and stapling"

  say "  repacking"
  rm -f "$archive"
  # -C and . so the archive keeps the shape it had, the links included.
  tar -czf "$archive" -C "$work" . || die "could not repack $name"
  rm -rf "$work"
  say "  done, signed, notarised and stapled"
}

main() {
  [ -d "$DIR" ] || die "$DIR is not a directory"

  # Counted before the keychain is asked anything. sign_release.py sends the
  # archives its table names, and a directory holding none has nothing to sign.
  local archives=() line
  while IFS= read -r line; do
    archives+=("$line")
  done < <(find "$DIR" -mindepth 1 -maxdepth 1 -name '*_macos_*.tar.gz' | sort)
  [ "${#archives[@]}" -gt 0 ] || die "$DIR holds no macOS archive"

  echo "[1/4] the keychain"
  unlock_keychain

  echo "[2/4] the credentials"
  check_profile

  echo "[3/4] the certificate"
  local identity
  identity="$(find_identity)" || exit 1
  say "signing with the pinned certificate"

  echo "[4/4] the archives: ${#archives[@]}"
  local archive
  for archive in "${archives[@]}"; do
    sign_archive "$archive" "$identity"
  done

  echo
  echo "Every macOS archive is signed, notarised and stapled."
}

main "$@"
