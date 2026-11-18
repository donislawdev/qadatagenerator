#!/usr/bin/env bash
# Wrap a macOS binary in a .app bundle, because that is the only shape that can
# carry a notarisation ticket.
#
#   bash .github/scripts/make_app_bundle.sh <workdir> <binary> <bundle-id> <version> <name> [<another binary>...]
#
# The binary the bundle opens is <binary>, and <name> is what macOS calls the
# program on screen, in the menu bar for one. The command line's bundle is qadg,
# the command. The window's is QA Data Gen, the product's name cut to the 15
# characters Apple allows it, by the owner's choice of 2026-11-18, while its
# file and the link beside it stay qadg-gui, a name with no space to type. Each
# binary named after <name> goes into the bundle beside <binary>, in
# Contents/MacOS, so the ticket of the bundle covers it as well: the window's
# archive carries the command line too (DEC-324), and a binary outside a
# bundle cannot carry a ticket of its own. Every binary gets a link beside the
# bundle, so a person types ./qadg-gui or ./qadg.
#
# Measured in Testing Files Generator, a project of the same owner, on
# 2026-08-28, on a real Developer ID certificate and two real notarisations, as
# its copy of this script records:
#
#   * a bare Mach-O binary cannot be stapled at all. stapler exits 66, and
#     Gatekeeper then refuses the binary (spctl exits 3) even when it is signed
#     AND notarised.
#   * a .dmg staples and does not help: the binary INSIDE a stapled dmg is still
#     refused, because the ticket describes the container.
#   * a .app staples, is accepted, and the ticket survives being packed into
#     tar.gz and unpacked again.
#
# Without a ticket in the file, macOS asks Apple over the network whether the
# program is notarised, and a person whose network gets in the way is refused.
#
# The binary must NOT be lifted out of the bundle afterwards. A copy of it on
# its own is refused, because the signature covers the bundle, not the file.
#
# Adapted on 2026-11-18 from the same script in Cert Doctor, a project of the
# same owner under the same licence.
set -euo pipefail

if [ "$#" -lt 5 ]; then
  echo "usage: make_app_bundle.sh <workdir> <binary> <bundle-id> <version> <name> [<another binary>...]" >&2
  exit 2
fi

work="$1"
binary="$2"
bundle_id="$3"
version="$4"
shown="$5"
shift 5
inside=("$binary" "$@")

# CFBundleName: "This name can contain up to 15 characters." Apple's
# Information Property List reference, read 2026-11-18. It is written into the
# property list below as it is, so it is held to letters, digits, spaces, dots
# and dashes: nothing XML would read as markup.
if ! [[ "$shown" =~ ^[A-Za-z0-9][A-Za-z0-9\ .-]{0,14}$ ]]; then
  echo "make_app_bundle: the name '${shown}' is not 1 to 15 letters, digits, spaces, dots or dashes" >&2
  exit 2
fi
if ! [[ "$bundle_id" =~ ^[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)+$ ]]; then
  echo "make_app_bundle: '${bundle_id}' is not a bundle identifier" >&2
  exit 2
fi
if ! [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
  echo "make_app_bundle: '${version}' is not a version" >&2
  exit 2
fi

for name in "${inside[@]}"; do
  if [ ! -f "${work}/${name}" ]; then
    echo "make_app_bundle: ${work}/${name} is not there, nothing to wrap" >&2
    exit 1
  fi
done

# The icon, refused rather than skipped when it is missing. A bundle without
# one shows a blank sheet of paper in the Finder and in the Dock. The file is in
# the repository rather than made here, because this runs where Apple's icon
# tools are not: the bundles are put together on Linux. The test beside it in
# internal/gui/appicon draws it from the same drawing as the icon on Windows.
here="$(cd "$(dirname "$0")" && pwd)"
icon="${here}/../../internal/gui/appicon/lentils.icns"
if [ ! -f "${icon}" ]; then
  echo "make_app_bundle: no icon at ${icon}, so the bundle would show a blank page" >&2
  exit 1
fi

app="${work}/${binary}.app"
mkdir -p "${app}/Contents/MacOS" "${app}/Contents/Resources"
for name in "${inside[@]}"; do
  mv "${work}/${name}" "${app}/Contents/MacOS/${name}"
  chmod 755 "${app}/Contents/MacOS/${name}"
done
cp "${icon}" "${app}/Contents/Resources/icon.icns"
chmod 644 "${app}/Contents/Resources/icon.icns"

# LSMinimumSystemVersion is 13.0 because that is the oldest macOS the Go
# toolchain this is built with supports: "For Go 1.27 and later: macOS 13
# Ventura or newer", https://go.dev/wiki/MinimumRequirements, read 2026-11-18.
# It moves when go.mod moves past a release that raises it.
#
# NSHighResolutionCapable says the program draws for a Retina screen, and the
# window does. The command line draws nothing, and the key costs it nothing.
#
# NSHumanReadableCopyright is the LegalCopyright of the Windows resources in
# cmd/qadg-gui/qadg-gui.rc, word for word, so the author reads the same on both
# systems. A guard in internal/guard holds the two equal.
cat > "${app}/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key>
	<string>${binary}</string>
	<key>CFBundleIconFile</key>
	<string>icon</string>
	<key>CFBundleIdentifier</key>
	<string>${bundle_id}</string>
	<key>CFBundleName</key>
	<string>${shown}</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleShortVersionString</key>
	<string>${version}</string>
	<key>CFBundleVersion</key>
	<string>${version}</string>
	<key>LSMinimumSystemVersion</key>
	<string>13.0</string>
	<key>NSHighResolutionCapable</key>
	<true/>
	<key>NSHumanReadableCopyright</key>
	<string>Copyright (C) 2026 DonislawDev. Licensed under GPL-3.0-only.</string>
</dict>
</plist>
PLIST
chmod 644 "${app}/Contents/Info.plist"

# Relative, so it still points at the binary after the archive is unpacked
# anywhere. On a filesystem that cannot make links, ln quietly makes a COPY
# instead, and a copy is worse than nothing here: Gatekeeper refuses a binary
# outside its bundle, so the archive would carry something that looks like the
# program and will not run.
for name in "${inside[@]}"; do
  ln -s "${binary}.app/Contents/MacOS/${name}" "${work}/${name}"
  if [ ! -L "${work}/${name}" ]; then
    echo "make_app_bundle: ${name} beside the bundle is a copy, not a link." >&2
    echo "A copy of the binary outside the bundle is refused by Gatekeeper." >&2
    exit 1
  fi
done

echo "wrapped ${inside[*]} in ${binary}.app (${bundle_id}, ${version})"
