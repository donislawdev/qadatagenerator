#!/usr/bin/env python3
"""Sign a release on the two machines that hold its keys, then hand it back.

    python .github/scripts/sign_release.py v0.2.0 --macos-host user@mac --notary-profile NAME
    python .github/scripts/sign_release.py v0.2.0 --macos-host user@mac --notary-profile NAME --dry-run
    python .github/scripts/sign_release.py --rehearsal RUN --macos-host user@mac --notary-profile NAME

Run it from the root of a clone of this repository, on the Windows machine with
the card reader, after release.yml has finished for the tag. Start it from Git
Bash: its ssh is the one found first there, and the ssh.exe of Windows refuses a
private key whose file permissions it does not like. The Mac and the notarytool
profile can also come from QADG_MACOS_HOST and QADG_NOTARY_PROFILE. Neither has
a default, because both are facts about the maintainer's machines, and a public
repository should not carry them.

Why this is a step a person runs
--------------------------------
The Windows key is on a cryptographic card in a USB reader and cannot be
exported, which is the whole value of it, so no runner GitHub hosts can reach
it. A self hosted runner could, and on a public repository a self hosted runner
is a machine strangers can aim a pull request at. So the build happens where
builds belong, the signature happens where the card is, and this script is the
seam between them.

The Apple key is the same problem in another shape: Apple's notary service only
talks to credentials held in a keychain, and that keychain is on the
maintainer's Mac. So five of the eight archives are signed here, in two places,
the three for Windows at the card and the two for macOS on the Mac, the Windows
installer is built here from the signed window archive and signed at the card,
and the checksums are written once at the end over everything.

What it does, in order, and what it refuses
-------------------------------------------
 1. downloads the unsigned build release.yml produced for the tag.
 2. checks it before touching it: exactly the files a release is made of, and
    each one against the statement release.yml made about it, made by that
    workflow, for that tag, on a runner GitHub hosts. Signing something nobody
    checked is how a supply chain gets a signature on it.
 3. exports the tree of the tag with git archive and asks build_msi.py from it
    whether this machine can build the Windows installer (the template, the
    icon, WiX at the version it builds with), before anything is signed. A
    release candidate gets no installer.
 4. finds the pinned certificate among those this machine offers, before
    anything is signed anywhere.
 5. hands the two macOS archives to the Mac, which signs each program in the
    .app bundle inside each one and then the bundle, notarises it with Apple and
    staples the ticket on. The Mac goes before the card because Apple is the
    step that refuses for reasons of its own. In the project this script comes
    from, both refusals of the first rehearsal came from Apple, after the card
    had already signed.
 6. signs every program and library in the three Windows archives with the
    card, each with an RFC 3161 timestamp, because without one a signature dies
    with the certificate. Each distinct file is signed ONCE: the window's
    archive carries the very qadg.exe the command line's archive does. Each
    signed file is read back: the certificate in it has to hash to the pin, the
    signature has to carry a timestamp, and the file without its signature has
    to be the file that went in, byte for byte. The two files of the software
    renderer have to be the files their pin names before they are signed.
 7. builds the Windows installer from the window's SIGNED archive with the
    tagged tree's build_msi.py, signs it with the product's name as the
    description Windows shows when it asks an administrator, and reads the
    certificate back out of it. An installer made before the card signed the
    programs would carry unsigned copies of them.
 8. writes verify-SHA256SUMS.txt over everything it is about to publish, signed
    and unsigned alike.
 9. uploads the lot to the DRAFT release, refusing if it is no longer a draft,
    and asks attest-release.yml for the statement about the signed bytes.
10. waits for that statement and confirms the draft holds exactly the files of
    a release, because a draft missing one file looks almost exactly like a
    finished one.

Nothing here publishes. The release stays a draft until a person reads it and
presses the button.

The rehearsal
-------------
--rehearsal RUN takes the build of a run of release.yml on main, started by hand
or by a push that changed the release, instead of the build of a tag, and does
steps 1 to 8 on it for real: the Mac signs, Apple notarises, the card signs and
the installer is built from the tree of the commit that run built.
It uploads nothing, dispatches nothing and touches no draft. A run on main has
no provenance statement, because the job that makes one is fenced behind a tag,
so what stands in for step 2 is GitHub's own account of the run and of its
artefact: the workflow file, how it was started, that it succeeded, that main
holds its commit, and the digest GitHub keeps for the artefact against the
bytes downloaded. What it leaves is signed and is not a release.

Adapted on 2026-11-18 from the release script of Cert Doctor, a project of the
same owner under the same licence, which signed its releases 0.2.0 and 0.3.0
with it. Its mechanics and refusals were each paid for on a real release, and
the notes beside them say where.
"""
import argparse
import base64
import contextlib
import datetime
import hashlib
import io
import json
import os
import re
import shlex
import shutil
import struct
import subprocess
import sys
import tarfile
import tempfile
import time
import zipfile

REPO = "donislawdev/qadatagenerator"
RELEASE_WORKFLOW = ".github/workflows/release.yml"
ATTEST_WORKFLOW = "attest-release.yml"

# Certum's timestamp service. Every signature the card made for Cert Doctor's
# releases 0.2.0 and 0.3.0 carries a timestamp from it, read back from the
# signed files by certificate_of below.
TIMESTAMP_URL = "http://time.certum.pl/"

# Every archive of a release, by program and target: the five builds of the
# command line and the three of the window (DEC-324). internal/guard holds this
# table to the targets the release builds. archive_name says how each is
# named, the way build_release.sh names it, and SIGNED_ON where its system is
# signed. An archive whose system is not in SIGNED_ON is the finished article,
# so the statement the build made about it stays true, because its bytes do
# not move.
RELEASE = (
    ("qadg", "windows/amd64"),
    ("qadg", "windows/arm64"),
    ("qadg", "linux/amd64"),
    ("qadg", "linux/arm64"),
    ("qadg", "darwin/arm64"),
    ("qadg-gui", "windows/amd64"),
    ("qadg-gui", "linux/amd64"),
    ("qadg-gui", "darwin/arm64"),
)
SIGNED_ON = {"windows": "card", "darwin": "mac"}

# What the card signs inside a Windows archive of each program: every program
# and every library, and the archive may hold no other. A program this table
# does not name would go out unsigned beside signed ones, so it is refused
# rather than signed unread. The window's archive carries the command line, and
# the software renderer in opengl/, which the window loads when the graphics
# driver offers no OpenGL. RENDERER stands for the renderer's files, read from
# the pin they were fetched by rather than written here.
RENDERER = "<the files .github/mesa-dist-win pins>"
SIGNED_INSIDE = {
    "qadg": ("qadg.exe",),
    "qadg-gui": ("qadg-gui.exe", "qadg.exe", RENDERER),
}
RENDERER_PIN = os.path.join(".github", "mesa-dist-win")

# The Windows installer, one for a release and none for a release candidate
# (DEC-331). Built after the card signed the programs in it, because an
# installer made before would carry unsigned copies, by build_msi.py taken from
# the tree of the tag, so it is built from what was tagged. Signed with the
# product's name as its description, which is the name Windows shows when it
# asks an administrator to let the installer run. Without it the prompt showed
# the digits of a temporary copy's name (measured in another Go project of the
# same owner on 2026-09-29, as Cert Doctor's release script records).
INSTALLER = "qadg-setup_%s_windows_amd64.msi"
BUILD_MSI = os.path.join(".github", "scripts", "build_msi.py")

# The files a person uses to CHECK a download rather than the download itself.
# GitHub lists the files of a release by name and nothing else, so the prefix is
# what puts these at the end of the list.
AUX_PREFIX = "verify-"
SUMS = AUX_PREFIX + "SHA256SUMS.txt"

# What release.yml hands over besides the archives and the bill of materials:
# the list its statement covers, and the statement.
BUILD_SUMS = "build.sha256"
BUILD_STATEMENT = "build.provenance.sigstore.json"

# The one written copy of each certificate digest. Read here rather than
# copied, because two written copies of one digest is the drift a pin exists to
# catch.
PIN_FILE = os.path.join(".github", "signing-certificates.txt")
WINDOWS_PIN = "windows-code-signing"
APPLE_PIN = "apple-developer-id"

# The OID, not the display name. Measured in the maintainer's other project on
# 2026-08-28: filtering certificates by the name "Code Signing" finds NOTHING on
# a Polish Windows, which calls it "Podpisywanie kodu", and the script then
# reported a missing card while the card was plugged in and working.
CODE_SIGNING_OID = "1.3.6.1.5.5.7.3.3"

# The tag goes into a directory name on the Mac and into a command line there,
# so it is held to the shape of a version before anything uses it, the shape
# release.yml holds the version in the code to. The version a rehearsal reads
# out of the name of the bill of materials is held to the same shape.
TAG_SHAPE = re.compile(r"^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$")

# The Mac and the profile reach ssh and a shell on the Mac, so each is held to
# a shape with nothing a shell or ssh would read as more than a name. A host
# starting with a dash would be an option to ssh.
HOST_SHAPE = re.compile(r"^(?:[A-Za-z0-9][A-Za-z0-9._-]*@)?[A-Za-z0-9][A-Za-z0-9._-]*$")
PROFILE_SHAPE = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]*$")


def is_candidate(version):
    """A release candidate: a hyphen in the version, the rule release.yml marks
    a pre-release by and build_msi.py refuses an installer by, because Windows
    Installer reads only the three numbers of a version, and a candidate's
    installer would take the release's own version."""
    return "-" in version


def run(argv, **kw):
    """Run something and let it print. A failure raises."""
    print("    $ %s" % " ".join(str(a) for a in argv))
    return subprocess.run(argv, check=True, **kw)


def powershell(script, env=None):
    """Run a PowerShell 7 script and return what it printed.

    The script goes as -EncodedCommand, so no quote and no line in it is read
    by anything on the way to PowerShell. Values the script needs go in env,
    never in its text: a file name with a quote in it would end a string and the
    rest would run as PowerShell.
    """
    encoded = base64.b64encode(script.encode("utf-16-le")).decode("ascii")
    out = subprocess.run(
        ["pwsh", "-NoProfile", "-NonInteractive", "-EncodedCommand", encoded],
        capture_output=True, text=True,
        env=None if env is None else {**os.environ, **env})
    if out.returncode != 0:
        raise SystemExit("sign_release: pwsh failed:\n%s" % out.stderr.strip())
    # PowerShell errors are NON TERMINATING by default, so a script can print a
    # page of complaints and still exit zero. Reading stderr only on a failure
    # threw away the one sentence that said what went wrong in the project this
    # comes from, and it cost an hour blaming a card that was in the reader.
    if out.stderr.strip():
        print("    pwsh also said:")
        for line in out.stderr.strip().splitlines():
            print("      %s" % line)
    return out.stdout


def pinned(name):
    """The digest a certificate is pinned to, read from the pin file."""
    with open(PIN_FILE, encoding="utf-8") as handle:
        for line in handle:
            fields = line.split()
            if len(fields) == 3 and fields[0] == name:
                if not re.fullmatch(r"[0-9a-f]{64}", fields[1]):
                    raise SystemExit("sign_release: %s in %s is not a sha256" % (name, PIN_FILE))
                return fields[1]
    raise SystemExit("sign_release: no %s line in %s" % (name, PIN_FILE))


def renderer_files():
    """The software renderer's files as the window's archive holds them, each
    with the sha256 and the size its pin gives.

    Read from the file= lines of .github/mesa-dist-win, each a path in the
    renderer's own archive, a sum and a size, and placed in opengl/, where
    build_release.sh packs them.
    """
    found = {}
    with open(RENDERER_PIN, encoding="utf-8") as handle:
        for line in handle:
            if not line.startswith("file="):
                continue
            fields = line[len("file="):].split()
            if len(fields) != 3 or not re.fullmatch(r"[0-9a-f]{64}", fields[1]) or not fields[2].isdigit():
                raise SystemExit("sign_release: %s has a file= line that is not a path, a sha256 and "
                                 "a size: %s" % (RENDERER_PIN, line.strip()))
            found["opengl/" + fields[0].rsplit("/", 1)[-1]] = (fields[1], int(fields[2]))
    if not found:
        raise SystemExit("sign_release: %s pins no file of the software renderer" % RENDERER_PIN)
    return found


def signed_inside(program):
    """The programs and libraries the card signs in a Windows archive of program."""
    files = []
    for name in SIGNED_INSIDE[program]:
        files.extend(renderer_files() if name == RENDERER else [name])
    return sorted(files)


def archive_name(program, target, version):
    """The file name build_release.sh gives the archive of program for target."""
    goos, goarch = target.split("/")
    system = "macos" if goos == "darwin" else goos
    end = ".zip" if goos == "windows" else ".tar.gz"
    return "%s_%s_%s_%s%s" % (program, version, system, goarch, end)


def archive_names(version):
    """Every archive of the release by its file name, with its program, its
    target and where it is signed, or None."""
    names = {}
    for program, target in RELEASE:
        names[archive_name(program, target, version)] = (program, target, SIGNED_ON.get(target.split("/")[0]))
    return names


def bill_of_materials(version):
    return "%sqadg_%s.spdx.json" % (AUX_PREFIX, version)


def find_signtool():
    """The newest x64 signtool.exe from the Windows SDK, or a clear refusal."""
    kits = r"C:\Program Files (x86)\Windows Kits\10\bin"
    if os.path.isdir(kits):
        for version in sorted(os.listdir(kits), reverse=True):
            candidate = os.path.join(kits, version, "x64", "signtool.exe")
            if os.path.isfile(candidate):
                return candidate
    raise SystemExit(
        "sign_release: no signtool.exe under %s. Install the Windows SDK, where the "
        "Signing Tools component is enough." % kits)


def expiry_notice(not_after, now):
    """What to say about a certificate that will not last forever."""
    if not not_after:
        return ["  the certificate does not say when it expires"]
    try:
        when = datetime.datetime.fromisoformat(str(not_after).replace("Z", "+00:00"))
    except ValueError:
        return ["  cannot read the expiry date: %s" % not_after]
    if when.tzinfo:
        when = when.replace(tzinfo=None)
    days = (when - now).days
    if days < 0:
        raise SystemExit(
            "sign_release: the certificate expired on %s. Nothing can be signed with it.\n"
            "A renewal is a DIFFERENT certificate, so its line in %s has to move with it."
            % (when.date(), PIN_FILE))
    if days < 90:
        return ["  %d days left on this certificate." % days,
                "     A renewal is a DIFFERENT certificate, so its line in %s goes with it." % PIN_FILE]
    return ["  %d days left on this certificate" % days]


# The certificates of the Windows store that may sign code, each with the
# sha256 of its DER bytes. The store is opened through .NET rather than through
# the Cert: drive, which belongs to a PowerShell module: in the project this
# comes from, a shell started from another shell came up without the module,
# answered "Cannot find drive", exited zero, and the script reported a missing
# card while the card was in the reader. X509Store is in the runtime.
CERTIFICATES_SCRIPT = """
$out = @()
foreach ($where in 'CurrentUser', 'LocalMachine') {
  $store = New-Object System.Security.Cryptography.X509Certificates.X509Store('My', $where)
  try { $store.Open('ReadOnly') } catch { continue }
  foreach ($c in $store.Certificates) {
    $eku = @()
    foreach ($x in $c.Extensions) {
      if ($x -is [System.Security.Cryptography.X509Certificates.X509EnhancedKeyUsageExtension]) {
        foreach ($u in $x.EnhancedKeyUsages) { $eku += $u.Value }
      }
    }
    if ($eku -notcontains $env:QADG_CODE_SIGNING_OID) { continue }
    $h = [System.Security.Cryptography.SHA256]::Create().ComputeHash($c.RawData)
    $out += [pscustomobject]@{
      sha256 = (($h | ForEach-Object { $_.ToString('x2') }) -join '')
      thumb = $c.Thumbprint
      subject = $c.Subject
      notAfter = $c.NotAfter.ToString('s')
    }
  }
  $store.Close()
}
ConvertTo-Json -InputObject @($out) -Compress
"""


def signing_thumbprint(pin):
    """The SHA-1 thumbprint of the certificate whose DER bytes hash to the pin.

    signtool selects by SHA-1 because that is the only selector it takes, and
    the repository pins SHA-256 because that is the digest worth pinning.
    Resolving one to the other here means the two never drift apart.
    """
    entries = json.loads(powershell(CERTIFICATES_SCRIPT, env={"QADG_CODE_SIGNING_OID": CODE_SIGNING_OID})
                         .strip() or "[]")
    if isinstance(entries, dict):
        entries = [entries]
    for entry in entries:
        if entry.get("sha256") == pin:
            # The common name only. The rest of the subject carries a town and
            # a region, and a terminal gets pasted into chats.
            print("  certificate: %s" % entry.get("subject", "").split(",")[0])
            for line in expiry_notice(entry.get("notAfter"), datetime.datetime.now()):
                print(line)
            return entry["thumb"]
    raise SystemExit(
        "sign_release: the pinned certificate (%s...) is not in the Windows store.\n"
        "Plug in the card reader and check the card's software can see it. If the "
        "certificate was renewed, its line in %s has to move with it." % (pin[:16], PIN_FILE))


# The certificate that actually signed a file, read back out of it. A signature
# that is not Valid, or that carries no timestamp, is refused here.
SIGNER_SCRIPT = """
$s = Get-AuthenticodeSignature -LiteralPath $env:QADG_SIGNED_FILE
if ($s.Status -ne 'Valid') {
  Write-Error ('signature status: ' + $s.Status)
  exit 1
}
if (-not $s.TimeStamperCertificate) {
  Write-Error 'the signature carries no timestamp'
  exit 1
}
$h = [System.Security.Cryptography.SHA256]::Create().ComputeHash($s.SignerCertificate.RawData)
(($h | ForEach-Object { $_.ToString('x2') }) -join '')
"""


def certificate_of(path):
    """The sha256 of the certificate that signed a file. The path reaches
    PowerShell as an environment variable and never as part of the script."""
    return powershell(SIGNER_SCRIPT, env={"QADG_SIGNED_FILE": os.path.abspath(path)}).strip()


def pe_layout(data, name):
    """Where a PE file keeps its checksum and its certificate table entry.

    The checksum is the 4 bytes at offset 64 of the optional header. The
    certificate table is the fifth data directory, whose place depends on
    whether the optional header is PE32 or PE32+ (Microsoft's PE format
    specification, "Optional Header Data Directories").
    """
    if len(data) < 0x40 or data[:2] != b"MZ":
        raise SystemExit("sign_release: %s is not a Windows program or library" % name)
    pe = struct.unpack_from("<I", data, 0x3C)[0]
    optional = pe + 4 + 20
    if optional + 2 > len(data) or data[pe:pe + 4] != b"PE\0\0":
        raise SystemExit("sign_release: %s has no PE header where its DOS header says" % name)
    magic = struct.unpack_from("<H", data, optional)[0]
    if magic == 0x10B:
        directories = optional + 96
    elif magic == 0x20B:
        directories = optional + 112
    else:
        raise SystemExit("sign_release: %s has an optional header of kind %#x" % (name, magic))
    if directories + 5 * 8 > len(data) or struct.unpack_from("<I", data, directories - 4)[0] < 5:
        raise SystemExit("sign_release: %s has no certificate table entry" % name)
    return optional + 64, directories + 4 * 8


def signature_only(before, after, name):
    """Refuse a signed file that differs from what went in by more than its
    signature.

    Authenticode signs a PE file in three places and no other: it appends the
    signature at the end, after up to 7 bytes of zeros that align it, points
    the certificate table entry at it, and rewrites the checksum. Measured on
    2026-11-18 on the two libraries of the software renderer, as the same card
    signed them for Cert Doctor's releases 0.2.0 and 0.3.0: with those three
    put back, each is the file its pin names, to the byte.
    """
    checksum_at, table_at = pe_layout(before, name)
    if pe_layout(after, name) != (checksum_at, table_at):
        raise SystemExit("sign_release: signing moved the headers of %s. Nothing has been uploaded." % name)
    if struct.unpack_from("<II", before, table_at) != (0, 0):
        raise SystemExit("sign_release: %s was already signed when it reached the card, and a "
                         "release build is not. Nothing has been uploaded." % name)
    where, size = struct.unpack_from("<II", after, table_at)
    if size == 0 or where < len(before) or where - len(before) >= 8 or where + size != len(after):
        raise SystemExit("sign_release: the signature of %s is not where signing puts one: %d bytes at "
                         "%d in a file of %d that held %d. Nothing has been uploaded."
                         % (name, size, where, len(after), len(before)))
    if after[len(before):where].strip(b"\0"):
        raise SystemExit("sign_release: signing put something other than zeros before the "
                         "signature of %s. Nothing has been uploaded." % name)
    restored = bytearray(after[:len(before)])
    restored[checksum_at:checksum_at + 4] = before[checksum_at:checksum_at + 4]
    restored[table_at:table_at + 8] = b"\0" * 8
    if restored != before:
        first = next(i for i, (a, b) in enumerate(zip(restored, before)) if a != b)
        raise SystemExit("sign_release: signing changed %s beyond its signature, first at byte %d. "
                         "Nothing has been uploaded." % (name, first))


def sha256_of(path):
    digest = hashlib.sha256()
    with open(path, "rb") as handle:
        for block in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def read_bytes(path):
    with open(path, "rb") as handle:
        return handle.read()


def gh_json(*args):
    out = subprocess.run(["gh", *args], capture_output=True, text=True)
    if out.returncode != 0:
        raise SystemExit("sign_release: gh %s failed:\n%s" % (args[0], out.stderr.strip()))
    return json.loads(out.stdout or "null")


def files_in(directory):
    """The files directly in a directory, sorted. Never a directory."""
    return sorted(n for n in os.listdir(directory)
                  if os.path.isfile(os.path.join(directory, n)))


def files_under(work):
    """Every file under work, as a path relative to it with forward slashes.

    Recursive on purpose. In the project this comes from, an archive was put
    back together from a listing of the top level, which names a directory and
    none of its contents, so a subdirectory came out of the signing EMPTY, with
    no error and a valid archive. The window's archive carries opengl/.
    """
    found = []
    for base, _dirs, names in os.walk(work):
        for name in names:
            found.append(os.path.relpath(os.path.join(base, name), work).replace(os.sep, "/"))
    return sorted(found)


def fresh(directory):
    """An empty directory at this path, whatever was there before."""
    if os.path.isdir(directory):
        shutil.rmtree(directory)
    os.makedirs(directory)


# --------------------------------------------------------------------------- #
# the steps
# --------------------------------------------------------------------------- #

def reach_the_mac(host):
    """Ask the Mac for nothing at all, before anything is signed.

    BatchMode, so a key that is refused fails here in seconds rather than
    asking for a password nobody expects.
    """
    print("  ssh is %s" % (shutil.which("ssh") or "not on the PATH"))
    out = subprocess.run(["ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=15", host, "true"],
                         capture_output=True, text=True)
    if out.returncode != 0:
        raise SystemExit(
            "sign_release: %s does not answer over ssh, so the macOS archives cannot be "
            "signed and nothing has been.\n%s\n"
            "On Windows, run this from Git Bash: the ssh.exe of Windows refuses a private "
            "key whose file permissions it does not like. A Mac with its screen locked "
            "can stop answering ssh as well." % (host, out.stderr.strip()))
    print("  %s answers" % host)


def fetch_build(tag, into):
    """The build release.yml made for this tag.

    A missing artefact is the ordinary case rather than a crash: the tag has not
    been pushed, the build is still running, or it failed.
    """
    fresh(into)
    try:
        run(["gh", "run", "download", "--repo", REPO, "--name", "unsigned-build-%s" % tag, "--dir", into])
    except subprocess.CalledProcessError:
        raise SystemExit(
            "sign_release: there is no build called unsigned-build-%s.\n"
            "Either the tag has not been pushed, or release.yml has not finished, "
            "or it failed. Look at the run before signing anything." % tag)
    names = files_in(into)
    print("  %d file(s): %s" % (len(names), ", ".join(names)))


def rehearsal_run(run_id):
    """GitHub's account of a run, refused unless it is one a rehearsal may sign.

    A run of release.yml on main, started by hand or by a push, finished and
    green, on a commit main holds. A run of another branch is code nobody
    merged, and a run that failed is a build the release workflow itself
    refused. A signature on either would be the certificate vouching for
    something nobody did. Asked of the API by the run's number in this
    repository, so the answer cannot come from a fork.
    """
    meta = gh_json("api", "repos/%s/actions/runs/%d" % (REPO, run_id))
    said = {
        "the workflow": (meta.get("path"), (RELEASE_WORKFLOW,)),
        "how it was started": (meta.get("event"), ("workflow_dispatch", "push")),
        "its state": (meta.get("status"), ("completed",)),
        "how it ended": (meta.get("conclusion"), ("success",)),
        "its branch": (meta.get("head_branch"), ("main",)),
        "its repository": ((meta.get("head_repository") or {}).get("full_name"), (REPO,)),
    }
    wrong = ["%s is %r, not %s" % (what, got, " or ".join(repr(w) for w in want))
             for what, (got, want) in said.items() if got not in want]
    if wrong:
        raise SystemExit("sign_release: run %d is not a build a rehearsal signs:\n  %s\n"
                         "A rehearsal takes the build of release.yml on main, after it went green."
                         % (run_id, "\n  ".join(wrong)))
    sha = meta.get("head_sha") or ""
    if not re.fullmatch(r"[0-9a-f]{40}", sha):
        raise SystemExit("sign_release: run %d names no commit (%r)" % (run_id, sha))
    # compare/<sha>...main is "ahead" when main holds the commit and has moved
    # on since, "identical" when it stands on it. Anything else is a commit main
    # does not hold, whatever branch name the run carries.
    status = gh_json("api", "repos/%s/compare/%s...main" % (REPO, sha), "--jq", "{status: .status}")
    if (status or {}).get("status") not in ("ahead", "identical"):
        raise SystemExit("sign_release: run %d built %s, and main does not hold that commit (%s)"
                         % (run_id, sha, (status or {}).get("status")))
    print("  run %d: %s, a %s on main at %s, %s" % (
        run_id, RELEASE_WORKFLOW, meta.get("event"), sha[:12], meta.get("conclusion")))
    return sha


def fetch_rehearsal(run_id, into):
    """The build of a run on main, checked against the digest GitHub keeps for
    it, its version, and the commit it built, which the installer is built from.

    The artefact is downloaded as the zip GitHub stored and its sha256 compared
    with the digest the API gives for it before anything is unpacked. In the
    project this comes from the two were measured equal on 2026-11-11. The zip
    goes beside the folder rather than into it, because everything in the
    folder gets a checksum, and it is gone once unpacked.
    """
    sha = rehearsal_run(run_id)
    found = gh_json("api", "repos/%s/actions/runs/%d/artifacts" % (REPO, run_id))
    builds = [a for a in (found or {}).get("artifacts", []) if a.get("name") == "build"]
    if len(builds) != 1 or builds[0].get("expired"):
        raise SystemExit("sign_release: run %d keeps no build to sign, it %s" % (
            run_id, "expired" if builds and builds[0].get("expired") else "has no artefact called build"))
    artefact = builds[0]
    want = artefact.get("digest") or ""
    if not re.fullmatch(r"sha256:[0-9a-f]{64}", want):
        raise SystemExit("sign_release: GitHub gives no sha256 for the build of run %d (%r), so there "
                         "is nothing to check the download against" % (run_id, want))

    fresh(into)
    stored = into.rstrip("/\\") + ".artefact.zip"
    print("    $ gh api repos/%s/actions/artifacts/%d/zip > %s" % (REPO, artefact["id"], stored))
    with open(stored, "wb") as handle:
        got = subprocess.run(["gh", "api", "repos/%s/actions/artifacts/%d/zip" % (REPO, artefact["id"])],
                             stdout=handle, stderr=subprocess.PIPE)
    if got.returncode != 0:
        raise SystemExit("sign_release: the build of run %d did not download:\n%s"
                         % (run_id, got.stderr.decode(errors="replace").strip()))
    actual = "sha256:" + sha256_of(stored)
    if actual != want:
        raise SystemExit("sign_release: the build of run %d downloaded as %s, and GitHub keeps %s for it"
                         % (run_id, actual, want))
    print("  the download is the artefact GitHub keeps: %s" % want)

    # The artefact is the build's dist/ as it was: files and no folders. A name
    # with a slash in it is not something the build makes.
    with zipfile.ZipFile(stored) as archive:
        names = archive.namelist()
        odd = [n for n in names if "/" in n or "\\" in n or n in ("", ".", "..")]
        if odd:
            raise SystemExit("sign_release: the build of run %d holds what the release build never "
                             "makes: %s" % (run_id, ", ".join(odd)))
        for name in names:
            with archive.open(name) as source, open(os.path.join(into, name), "wb") as target:
                shutil.copyfileobj(source, target)
    os.remove(stored)

    sboms = [n for n in names if re.fullmatch(r"%sqadg_.+\.spdx\.json" % AUX_PREFIX, n)]
    if len(sboms) != 1:
        raise SystemExit("sign_release: the build of run %d carries %d bill(s) of materials, and a "
                         "release one" % (run_id, len(sboms)))
    version = sboms[0][len(AUX_PREFIX + "qadg_"):-len(".spdx.json")]
    if not TAG_SHAPE.match("v" + version):
        raise SystemExit("sign_release: the build of run %d names version %r" % (run_id, version))
    print("  %d file(s) of version %s: %s" % (len(names), version, ", ".join(sorted(names))))
    return version, sha


def check_build(directory, version, tagged):
    """Refuse a build that is not exactly the files of a release.

    Everything in the folder is published in the end, so a file the build does
    not make is refused here rather than uploaded beside the release.
    """
    expected = set(archive_names(version)) | {bill_of_materials(version)}
    if tagged:
        expected |= {BUILD_SUMS, BUILD_STATEMENT}
    found = set(files_in(directory))
    missing, extra = sorted(expected - found), sorted(found - expected)
    if missing or extra:
        raise SystemExit("sign_release: the build is not the files of release %s%s%s"
                         % (version,
                            "\n  missing: " + ", ".join(missing) if missing else "",
                            "\n  not part of a release: " + ", ".join(extra) if extra else ""))
    print("  the %d archives of %s and its bill of materials, and nothing else"
          % (len(archive_names(version)), version))


def verify_before_touching(directory, tag):
    """Refuse to sign a build whose own statement does not check out.

    The step that makes the rest worth anything: the whole attack on a person
    with a key is to hand them something that is not what they think it is.
    The statement has to be one release.yml made, for this tag, on a runner
    GitHub hosts. A statement any other workflow of this repository made would
    pass a check of the repository alone. Measured on 2026-11-18 on Cert
    Doctor's release 0.3.0: these three flags accept its statement, and refuse
    it for another tag and for another workflow.
    """
    version = tag[1:]
    check_build(directory, version, tagged=True)
    bundle = os.path.join(directory, BUILD_STATEMENT)
    subjects = sorted(set(archive_names(version)) | {bill_of_materials(version)})
    for name in subjects:
        run(["gh", "attestation", "verify", os.path.join(directory, name), "-R", REPO,
             "--bundle", bundle,
             "--signer-workflow", "%s/%s" % (REPO, RELEASE_WORKFLOW),
             "--source-ref", "refs/tags/%s" % tag,
             "--deny-self-hosted-runners"])
    print("  %d file(s) verified against the build's own statement" % len(subjects))


@contextlib.contextmanager
def tagged_tree(ref):
    """The tree of ref, the tag or the commit a rehearsal's run built, exported
    with git archive into a temporary folder and removed afterwards.

    The installer is built from what was tagged, whatever this checkout stands
    on: the template, the names and the script that fills them all come from
    ref. filter="data" refuses a link or a name that leads out of the folder,
    which an archive of this tree never holds.
    """
    where = tempfile.mkdtemp(prefix="qadg-tree-")
    try:
        out = subprocess.run(["git", "archive", "--format=tar", ref], capture_output=True)
        if out.returncode != 0:
            raise SystemExit(
                "sign_release: git archive %s failed, so there is no tree to build the installer "
                "from. Fetch it first (git fetch --tags origin) and run this again.\n%s"
                % (ref, out.stderr.decode(errors="replace").strip()))
        with tarfile.open(fileobj=io.BytesIO(out.stdout)) as archive:
            archive.extractall(where, filter="data")
        print("  the tree of %s, exported for the installer" % ref)
        yield where
    finally:
        shutil.rmtree(where, ignore_errors=True)


def build_msi(tree, *args):
    """build_msi.py from the tagged tree, with these arguments, and what it
    printed. Its refusals are sentences that name the input, so a failure is
    said in its words and nothing goes on."""
    script = os.path.join(tree, BUILD_MSI)
    if not os.path.isfile(script):
        raise SystemExit("sign_release: the tree this release is built from has no %s, so it is "
                         "from before the installer existed, and it cannot have one. Nothing has "
                         "been signed or uploaded." % BUILD_MSI.replace(os.sep, "/"))
    print("    $ build_msi.py %s" % " ".join(args))
    out = subprocess.run([sys.executable, script, *args], capture_output=True, text=True)
    for line in out.stdout.strip().splitlines():
        print("      %s" % line)
    if out.returncode != 0:
        # build_msi.py refuses on stderr in one sentence. It ends this message
        # rather than standing above WiX's output, where nobody reads it.
        raise SystemExit("sign_release: the installer cannot be built.\n%s\nNothing has been uploaded."
                         % (out.stderr.strip() or "build_msi.py exited %d" % out.returncode))
    for line in out.stderr.strip().splitlines():
        print("      %s" % line)
    return out.stdout


def check_installer(tree, version):
    """Before anything is signed: whether this machine can build the installer
    of version from the tagged tree, its template, its icon, WiX."""
    if is_candidate(version):
        print("  %s is a release candidate, which gets no installer" % version)
        return
    build_msi(tree, "--tag", "v" + version, "--check")


def build_installer(tree, directory, version, thumbprint, pin, signtool, dry_run):
    """The installer, built from the signed window archive, signed with the
    product's name, and the certificate read back out of it."""
    if is_candidate(version):
        print("  %s is a release candidate, which gets no installer" % version)
        return
    tag = "v" + version
    name = build_msi(tree, "--tag", tag, "--product-name").strip()
    if not name:
        raise SystemExit("sign_release: build_msi.py --product-name said nothing, and the "
                         "installer is signed with that name. Nothing has been uploaded.")
    build_msi(tree, "--tag", tag, "--archives", directory, "--out-dir", directory)
    path = os.path.join(directory, INSTALLER % version)
    if not os.path.isfile(path):
        raise SystemExit("sign_release: build_msi.py wrote no %s. Nothing has been uploaded."
                         % os.path.basename(path))
    command = [signtool, "sign", "/sha1", thumbprint, "/fd", "sha256",
               "/tr", TIMESTAMP_URL, "/td", "sha256", "/d", name, "/v", path]
    if dry_run:
        print("    DRY RUN, would run: %s" % " ".join(command))
        print("  %s built from an unsigned archive, and not signed" % os.path.basename(path))
        return
    run(command)
    run([signtool, "verify", "/pa", "/v", path])
    actual = certificate_of(path)
    if actual != pin:
        raise SystemExit(
            "sign_release: %s was signed by a DIFFERENT certificate\n"
            "  expected %s\n  got      %s\nNothing has been uploaded."
            % (os.path.basename(path), pin, actual))
    print("  %s built, and signed as %s" % (os.path.basename(path), name))


def unpack_windows(directory, version):
    """Each Windows archive unpacked beside itself, refused unless it holds
    exactly the programs and libraries its program ships, and the renderer's
    files are the ones their pin names."""
    renderer = renderer_files()
    unpacked = {}
    for name, (program, target, where) in sorted(archive_names(version).items()):
        if where != "card":
            continue
        path = os.path.join(directory, name)
        work = path + ".unpacked"
        fresh(work)
        with zipfile.ZipFile(path) as archive:
            archive.extractall(work)
        inside = files_under(work)
        signable = sorted(n for n in inside if n.lower().endswith((".exe", ".dll")))
        wanted = signed_inside(program)
        if signable != wanted:
            raise SystemExit("sign_release: %s holds the programs and libraries\n  %s\nand a Windows "
                             "archive of %s holds\n  %s\nNothing has been signed."
                             % (name, ", ".join(signable) or "none", program, ", ".join(wanted)))
        for rel in signable:
            if rel in renderer:
                digest, size = renderer[rel]
                file = os.path.join(work, rel)
                if os.path.getsize(file) != size or sha256_of(file) != digest:
                    raise SystemExit("sign_release: %s in %s is not the file %s pins. "
                                     "Nothing has been signed." % (rel, name, RENDERER_PIN))
        unpacked[name] = (work, inside, signable, target)
    return unpacked


def distinct_files(unpacked):
    """The files to sign, each distinct one once: digest to its places.

    The window's archive carries the very qadg.exe the command line's archive
    does, because build_release.sh packs one build into both, and signing it
    twice would ask the card twice and publish two qadg.exe for one release
    that differ in their timestamps. So a file is signed once and its signed
    bytes copied to every other place it is. The same name in two archives for
    one target has to be the same file, or the release packed two builds of one
    program. Two targets of one system are two builds, so the key is the
    target and not the system.
    """
    by_digest, by_place = {}, {}
    for name, (work, _inside, signable, target) in sorted(unpacked.items()):
        for rel in signable:
            digest = sha256_of(os.path.join(work, rel))
            by_digest.setdefault(digest, []).append((name, rel))
            by_place.setdefault((target, rel), set()).add(digest)
    split = ["%s in the archives for %s" % (rel, target) for (target, rel), digests
             in sorted(by_place.items()) if len(digests) > 1]
    if split:
        raise SystemExit("sign_release: one name, two files. %s differ between the archives that "
                         "carry them, and a release packs one build of each program.\n"
                         "Nothing has been signed." % ", ".join(split))
    return by_digest


def sign_windows(directory, version, thumbprint, pin, signtool, dry_run):
    """Sign every program and library of the Windows archives, each distinct
    file once, read each one back, and put each archive back together whole."""
    unpacked = unpack_windows(directory, version)
    places = distinct_files(unpacked)
    for _digest, where in sorted(places.items(), key=lambda item: item[1][0]):
        first = os.path.join(unpacked[where[0][0]][0], where[0][1])
        command = [signtool, "sign", "/sha1", thumbprint, "/fd", "sha256",
                   "/tr", TIMESTAMP_URL, "/td", "sha256", "/v", first]
        if dry_run:
            print("    DRY RUN, would run: %s" % " ".join(command))
        else:
            before = read_bytes(first)
            run(command)
            run([signtool, "verify", "/pa", "/v", first])
            actual = certificate_of(first)
            if actual != pin:
                raise SystemExit(
                    "sign_release: %s in %s was signed by a DIFFERENT certificate\n"
                    "  expected %s\n  got      %s\nNothing has been uploaded."
                    % (where[0][1], where[0][0], pin, actual))
            signature_only(before, read_bytes(first), where[0][1])
        signed = sha256_of(first)
        for name, rel in where[1:]:
            copy = os.path.join(unpacked[name][0], rel)
            shutil.copyfile(first, copy)
            if sha256_of(copy) != signed:
                raise SystemExit("sign_release: %s in %s is not the signed %s after copying it"
                                 % (rel, name, where[0][1]))
        print("    %s: %s%s" % (where[0][1], "would be signed" if dry_run else "signed, nothing else changed",
                                 "".join(", and copied into %s" % name for name, _rel in where[1:])))

    for name, (work, inside, _signable, _target) in sorted(unpacked.items()):
        path = os.path.join(directory, name)
        if dry_run:
            shutil.rmtree(work)
            continue
        os.remove(path)
        with zipfile.ZipFile(path, "w", zipfile.ZIP_DEFLATED) as archive:
            for rel in files_under(work):
                archive.write(os.path.join(work, rel), rel)
        # What went in is what comes out. A repack that lost a file would be the
        # quietest defect this script could have, so it is counted, not trusted.
        with zipfile.ZipFile(path) as archive:
            repacked = sorted(n for n in archive.namelist() if not n.endswith("/"))
        if repacked != inside:
            raise SystemExit("sign_release: %s was repacked with %d file(s) and held %d\n"
                             "Nothing has been uploaded." % (name, len(repacked), len(inside)))
        shutil.rmtree(work)
        print("    %s: repacked, %d file(s)" % (name, len(repacked)))
    print("  %d distinct file(s) %s across %d archive(s)"
          % (len(places), "would be signed" if dry_run else "signed", len(unpacked)))


def macos_archives(directory, version):
    """The archives the Mac signs, by the table."""
    names = sorted(n for n, (_p, _target, where) in archive_names(version).items() if where == "mac")
    for name in names:
        if not os.path.isfile(os.path.join(directory, name)):
            raise SystemExit("sign_release: %s is not in the build" % name)
    return names


def sign_macos(label, directory, version, host, profile, dry_run):
    """Hand the macOS archives to the Mac, and take back signed ones.

    -t because the script over there asks for the keychain password on the
    terminal. A password given that way is not in argv, so it is not in ps,
    and it is nowhere in this repository.
    """
    names = macos_archives(directory, version)
    if dry_run:
        print("  DRY RUN, would sign %s on %s" % (", ".join(names), host))
        return
    before = {name: sha256_of(os.path.join(directory, name)) for name in names}
    remote = "qadg-signing-%s" % label
    run(["ssh", host, "rm -rf %s && mkdir -p %s/dist" % (shlex.quote(remote), shlex.quote(remote))])
    run(["scp", os.path.join(".github", "scripts", "sign_macos.sh"), "%s:%s/" % (host, remote)])
    for name in names:
        run(["scp", os.path.join(directory, name), "%s:%s/dist/" % (host, remote)])
    run(["ssh", "-t", host, "bash %s %s %s %s" % (
        shlex.quote(remote + "/sign_macos.sh"), shlex.quote(remote + "/dist"),
        pinned(APPLE_PIN), shlex.quote(profile))])
    for name in names:
        run(["scp", "%s:%s/dist/%s" % (host, remote, name), directory])
    run(["ssh", host, "rm -rf %s" % shlex.quote(remote)])
    # An archive handed back with the bytes it went with was not signed, and the
    # script over there says so itself. This is the same question asked of what
    # came back rather than of what was said.
    unchanged = [name for name in names if sha256_of(os.path.join(directory, name)) == before[name]]
    if unchanged:
        raise SystemExit("sign_release: %s came back from the Mac unchanged, so it was not signed"
                         % ", ".join(unchanged))
    print("  %s signed, notarised and stapled" % ", ".join(names))


def name_for_publication(directory, version):
    """Give the build's own files the names a person will see, or drop them.

    build.sha256 was the list of subjects of the statement the runner made. It
    describes archives that no longer exist in that form, so publishing it
    beside the real checksums would put two lists on one page, one of them wrong
    about five files. The statement stays, under a name a person can use: it
    still describes the Linux archives and the bill of materials byte for byte.
    """
    os.remove(os.path.join(directory, BUILD_SUMS))
    os.rename(os.path.join(directory, BUILD_STATEMENT),
              os.path.join(directory, "%sqadg_%s.provenance.sigstore.json" % (AUX_PREFIX, version)))


def write_checksums(directory):
    """The checksum file over what is about to be published, and its own digest."""
    published = files_in(directory)
    sums = os.path.join(directory, SUMS)
    with open(sums, "w", encoding="utf-8", newline="\n") as handle:
        for name in published:
            handle.write("%s  %s\n" % (sha256_of(os.path.join(directory, name)), name))
    print("  %d file(s) listed in %s" % (len(published), SUMS))
    return sha256_of(sums)


def draft_assets(tag):
    """The names of the files on the release of tag, refused unless it is
    still a draft. Nothing here publishes, so a release that is no longer a
    draft was published by somebody or something else, and nothing goes onto it
    from here."""
    release = gh_json("release", "view", tag, "--repo", REPO, "--json", "assets,isDraft")
    if not (release or {}).get("isDraft"):
        raise SystemExit("sign_release: %s is not a draft. Nothing here publishes, so somebody or "
                         "something else did, and nothing has been uploaded to it." % tag)
    return sorted(asset["name"] for asset in release.get("assets", []))


def upload_to_draft(tag, directory, dry_run):
    """Everything a person downloads, onto the draft."""
    files = [os.path.join(directory, n) for n in files_in(directory)]
    if dry_run:
        print("  DRY RUN, would upload %d file(s)" % len(files))
        return
    draft_assets(tag)
    run(["gh", "release", "upload", tag, "--repo", REPO, "--clobber", *files])


def ask_for_the_statement(tag, digest, dry_run):
    """Hand the signed bytes back to a workflow, which can sign a statement."""
    if dry_run:
        print("  DRY RUN, would dispatch %s for %s" % (ATTEST_WORKFLOW, digest[:16]))
        return
    run(["gh", "workflow", "run", ATTEST_WORKFLOW, "--repo", REPO,
         "-f", "tag=%s" % tag, "-f", "digest=%s" % digest])


def release_files(version):
    """Every file a finished draft of version holds, and no other. The
    installer is one of them for a release and never for a candidate, the rule
    build_msi.py refuses a candidate's installer by."""
    files = set(archive_names(version)) | {
        SUMS,
        bill_of_materials(version),
        "%sqadg_%s.provenance.sigstore.json" % (AUX_PREFIX, version),
        "%sqadg_%s.sbom.sigstore.json" % (AUX_PREFIX, version),
    }
    if not is_candidate(version):
        installer = INSTALLER % version
        files.add(installer)
    return sorted(files)


def confirm_draft(tag, wait_seconds, dry_run):
    """Wait for the statement and check the draft holds exactly a release."""
    if dry_run:
        print("  DRY RUN, not waiting")
        return
    version = tag[1:]
    statement = "%sqadg_%s.sbom.sigstore.json" % (AUX_PREFIX, version)
    deadline = time.time() + wait_seconds
    while True:
        names = draft_assets(tag)
        if statement in names:
            break
        if time.time() > deadline:
            raise SystemExit(
                "sign_release: the statement about the signed bytes has not arrived after "
                "%d seconds.\nThe draft holds: %s\nLook at the run of %s before publishing "
                "anything." % (wait_seconds, ", ".join(names), ATTEST_WORKFLOW))
        print("    waiting for %s ..." % ATTEST_WORKFLOW)
        time.sleep(10)

    # Each asked for by its whole name. A suffix would be happy with a file the
    # prefix fell off, and the prefix is what keeps these at the end.
    expected = release_files(version)
    missing = [n for n in expected if n not in names]
    extra = [n for n in names if n not in expected]
    if missing or extra:
        raise SystemExit("sign_release: the draft is not a release.%s%s\nIt holds: %s"
                         % (" Missing: " + ", ".join(missing) + "." if missing else "",
                            " Not part of a release: " + ", ".join(extra) + "." if extra else "",
                            ", ".join(names)))
    print("  the draft holds the %d files of a release and is still a draft" % len(names))


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("tag", nargs="?", help="the release tag, for example v0.2.0")
    parser.add_argument("--rehearsal", type=int, metavar="RUN",
                        help="sign the build of this run of release.yml on main, upload nothing")
    parser.add_argument("--dry-run", action="store_true",
                        help="everything except signing, uploading and dispatching")
    parser.add_argument("--wait", type=int, default=300,
                        help="seconds to wait for the statement (default 300)")
    parser.add_argument("--macos-host", default=os.environ.get("QADG_MACOS_HOST"),
                        help="user@host of the Mac that signs the macOS archives, "
                             "or set QADG_MACOS_HOST")
    parser.add_argument("--notary-profile", default=os.environ.get("QADG_NOTARY_PROFILE"),
                        help="the notarytool keychain profile on the Mac, "
                             "or set QADG_NOTARY_PROFILE")
    args = parser.parse_args(argv)

    if not os.path.isfile(PIN_FILE):
        raise SystemExit("sign_release: run this from the root of the repository")
    rehearsal = args.rehearsal is not None
    if rehearsal == (args.tag is not None):
        raise SystemExit("sign_release: give a tag to sign a release, or --rehearsal RUN to "
                         "rehearse on a run of release.yml on main, one of the two")
    if rehearsal and args.rehearsal <= 0:
        raise SystemExit("sign_release: %d is not the number of a run" % args.rehearsal)
    if not rehearsal and not TAG_SHAPE.match(args.tag):
        raise SystemExit("sign_release: %r is not the tag of a release, which looks like v0.2.0"
                         % args.tag)
    if not args.macos_host:
        raise SystemExit(
            "sign_release: no Mac to sign the macOS archives on.\n"
            "Pass --macos-host user@host, or set QADG_MACOS_HOST.\n"
            "A bare macOS program cannot carry a notarisation ticket, so those\n"
            "archives are refused by Gatekeeper unless this step runs.")
    if not HOST_SHAPE.match(args.macos_host):
        raise SystemExit("sign_release: %r is not a host for ssh, which looks like user@host"
                         % args.macos_host)
    if not args.notary_profile:
        raise SystemExit(
            "sign_release: no notarytool profile.\n"
            "Pass --notary-profile NAME, or set QADG_NOTARY_PROFILE, with the name the\n"
            "profile was stored under on the Mac by xcrun notarytool store-credentials.")
    if not PROFILE_SHAPE.match(args.notary_profile):
        raise SystemExit("sign_release: %r is not the name of a notarytool profile, which is letters, "
                         "digits, dots, dashes and underscores" % args.notary_profile)
    windows_pin = pinned(WINDOWS_PIN)
    pinned(APPLE_PIN)
    renderer_files()
    label = "rehearsal-%d" % args.rehearsal if rehearsal else args.tag
    work = os.path.join("dist", "signing", label)

    print("\nbefore anything: can the Mac be reached")
    reach_the_mac(args.macos_host)

    steps = 8 if rehearsal else 10
    if rehearsal:
        print("\n[1/%d] fetching the build of run %d, a REHEARSAL, nothing will be uploaded"
              % (steps, args.rehearsal))
        version, ref = fetch_rehearsal(args.rehearsal, work)
        print("\n[2/%d] checking it before touching it" % steps)
        check_build(work, version, tagged=False)
    else:
        version = args.tag[1:]
        ref = "refs/tags/" + args.tag
        print("\n[1/%d] fetching the build for %s" % (steps, args.tag))
        fetch_build(args.tag, work)
        print("\n[2/%d] checking it before touching it" % steps)
        verify_before_touching(work, args.tag)

    print("\n[3/%d] whether the installer can be built, before anything is signed" % steps)
    with tagged_tree(ref) as tree:
        check_installer(tree, version)

        print("\n[4/%d] the card's certificate, before anything is signed anywhere" % steps)
        thumbprint = signing_thumbprint(windows_pin)
        signtool = find_signtool()
        print("  signtool: %s" % signtool)

        print("\n[5/%d] signing the macOS bundles on %s" % (steps, args.macos_host))
        sign_macos(label, work, version, args.macos_host, args.notary_profile, args.dry_run)

        print("\n[6/%d] signing the Windows programs and libraries with the card" % steps)
        sign_windows(work, version, thumbprint, windows_pin, signtool, args.dry_run)

        print("\n[7/%d] building and signing the Windows installer" % steps)
        build_installer(tree, work, version, thumbprint, windows_pin, signtool, args.dry_run)

    print("\n[8/%d] checksums over what will be published" % steps)
    if not rehearsal:
        name_for_publication(work, version)
    digest = write_checksums(work)
    print("  %s: %s" % (SUMS, digest))

    if rehearsal:
        if args.dry_run:
            print("\nDone. The rehearsal would have signed %s in %s, and signed nothing." % (version, work))
            print("Nothing was uploaded and no draft was touched.")
            return 0
        print("\nDone. The rehearsal signed %s in %s." % (version, work))
        print("Nothing was uploaded and no draft was touched. These files are signed and are")
        print("NOT a release: look at them, then delete the folder.")
        return 0

    print("\n[9/%d] uploading to the draft" % steps)
    upload_to_draft(args.tag, work, args.dry_run)
    ask_for_the_statement(args.tag, digest, args.dry_run)

    print("\n[10/%d] waiting for the statement and checking the draft" % steps)
    confirm_draft(args.tag, args.wait, args.dry_run)

    print("\nDone. %s is a complete DRAFT." % args.tag)
    print("Read it, then publish it from the Releases page.")
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except subprocess.CalledProcessError as failed:
        # The command that failed has printed its own reason just above, because
        # sign_macos.sh, signtool and gh each say why, and a Python traceback
        # under that sentence only pushes it off the screen.
        sys.exit("sign_release: %s stopped with exit code %d, and the lines above say why."
                 % (os.path.basename(str(failed.cmd[0])), failed.returncode))
