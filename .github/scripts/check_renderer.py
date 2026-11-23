#!/usr/bin/env python3
"""Check that the software renderer in a window archive is the one its pin names.

    python .github/scripts/check_renderer.py <qadg-gui_VERSION_windows_amd64.zip> .github/mesa-dist-win

The two libraries of the software renderer travel in the opengl folder of the
window's archive for Windows, signed by the maintainer's certificate. A
signature says who signed a file and nothing about what the file is, and a
library beside a signed program is the file a tamperer would swap, because the
card signs whatever it is handed. So this takes the signature back off each
library and compares what is left with the sum its pin gives, which is the sum
of the file as the renderer's own release ships it.

Authenticode changes a PE file in three places and no other: it appends the
signature after up to 7 bytes of zeros that align it, points the certificate
table entry at it and rewrites the checksum (Microsoft's PE format
specification, "The Attribute Certificate Table"). The pin gives the size of
each file before it was signed, so the padding needs no guessing, and the
checksum is computed again over the restored file. Measured on 2026-11-18 on
the two libraries of Mesa 26.2.5 as the same card signed them for Cert Doctor's
releases 0.2.0 and 0.3.0: restored this way, each is the file its pin names, to
the byte.

A library without a signature is compared as it is, and said to be unsigned.
Whether it ought to be signed is asked by the signature check of the release,
not here, so an unsigned build can be checked with this as well.

sign_release.py makes the same comparison while it signs, holding the bytes
from before the card signed them. This holds only the published file and the
pin, so it restores rather than compares, and it is a script of its own.

Exit status 0 when every pinned file is the file its pin names, 1 otherwise,
with every problem found printed rather than only the first.
"""
import array
import hashlib
import re
import struct
import sys
import zipfile

# Where build_release.sh packs the renderer's files in the window's archive.
FOLDER = "opengl/"


def pinned_files(pin_path):
    """The renderer's files as the archive holds them, each with the sha256 and
    the size in bytes its pin gives, read from the file= lines of the pin."""
    found = []
    with open(pin_path, encoding="utf-8") as handle:
        for line in handle:
            if not line.startswith("file="):
                continue
            fields = line[len("file="):].split()
            if len(fields) != 3 or not re.fullmatch(r"[0-9a-f]{64}", fields[1]) or not fields[2].isdigit():
                raise SystemExit("check_renderer: %s has a file= line that is not a path, a sha256 and "
                                 "a size: %s" % (pin_path, line.strip()))
            found.append((FOLDER + fields[0].rsplit("/", 1)[-1], fields[1], int(fields[2])))
    if not found:
        raise SystemExit("check_renderer: %s pins no file of the software renderer" % pin_path)
    return found


def layout(data):
    """Where a PE file keeps its checksum and its certificate table entry.

    The checksum is the 4 bytes at offset 64 of the optional header. The
    certificate table is the fifth data directory, whose place depends on
    whether the optional header is PE32 or PE32+.
    """
    if len(data) < 0x40 or data[:2] != b"MZ":
        raise ValueError("is not a Windows library")
    pe = struct.unpack_from("<I", data, 0x3C)[0]
    optional = pe + 4 + 20
    if optional + 2 > len(data) or data[pe:pe + 4] != b"PE\0\0":
        raise ValueError("has no PE header where its DOS header says")
    magic = struct.unpack_from("<H", data, optional)[0]
    if magic == 0x10B:
        directories = optional + 96
    elif magic == 0x20B:
        directories = optional + 112
    else:
        raise ValueError("has an optional header of kind %#x" % magic)
    if directories + 5 * 8 > len(data) or struct.unpack_from("<I", data, directories - 4)[0] < 5:
        raise ValueError("has no certificate table entry")
    return optional + 64, directories + 4 * 8


def checksum(data, checksum_at):
    """The PE checksum of data: its 16-bit words added with the carry folded
    back in, the checksum field counted as zero, plus the length."""
    body = bytearray(data)
    body[checksum_at:checksum_at + 4] = b"\0\0\0\0"
    if len(body) % 2:
        body.append(0)
    words = array.array("H", bytes(body))
    if sys.byteorder != "little":
        words.byteswap()
    total = sum(words)
    while total >> 16:
        total = (total & 0xFFFF) + (total >> 16)
    return (total + len(data)) & 0xFFFFFFFF


def restore(data, size):
    """The file as it was before it was signed, given the size it had then, or
    None when it carries no signature. Raises ValueError when the signature is
    not where signing puts one, because then more than a signature was added."""
    checksum_at, table_at = layout(data)
    where, length = struct.unpack_from("<II", data, table_at)
    if (where, length) == (0, 0):
        return None
    if where < size or where - size >= 8 or where + length != len(data):
        raise ValueError("carries a signature that is not where signing puts one: %d bytes at %d in "
                         "a file of %d, which held %d before it was signed"
                         % (length, where, len(data), size))
    if data[size:where].strip(b"\0"):
        raise ValueError("carries something other than zeros between the file and its signature")
    restored = bytearray(data[:size])
    restored[table_at:table_at + 8] = b"\0" * 8
    struct.pack_into("<I", restored, checksum_at, checksum(restored, checksum_at))
    return bytes(restored)


def check(archive_path, pin_path):
    """Every problem with the renderer's files in the archive, as sentences."""
    problems = []
    with zipfile.ZipFile(archive_path) as archive:
        held = set(archive.namelist())
        for name, want, size in pinned_files(pin_path):
            if name not in held:
                problems.append("%s holds no %s, which %s pins" % (archive_path, name, pin_path))
                continue
            data = archive.read(name)
            try:
                restored = restore(data, size)
            except ValueError as err:
                problems.append("%s %s" % (name, err))
                continue
            if restored is None:
                state, compared = "unsigned", data
            else:
                state, compared = "signed", restored
            got = hashlib.sha256(compared).hexdigest()
            if got != want:
                problems.append("%s (%s) is not the file its pin names: without a signature it is %d "
                                "bytes with sha256 %s, and the pin says %d bytes with sha256 %s"
                                % (name, state, len(compared), got, size, want))
                continue
            print("%s: %s, and without a signature it is the file its pin names, %s" % (name, state, want))
    return problems


def main(argv):
    if len(argv) != 3:
        raise SystemExit(__doc__)
    try:
        problems = check(argv[1], argv[2])
    except (OSError, zipfile.BadZipFile) as err:
        raise SystemExit("check_renderer: cannot read %s: %s" % (argv[1], err))
    for problem in problems:
        print("check_renderer: " + problem)
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
