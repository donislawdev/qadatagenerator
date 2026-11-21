Every file of this release was built from the tagged commit by the release
workflow of this repository, on machines GitHub hosts. What changed is in the
[changelog](https://github.com/donislawdev/qadatagenerator/blob/main/CHANGELOG.md).

**Which file to download**

- `qadg_<version>_<system>_<arch>` holds the command line, `qadg`, for one
  system.
- `qadg-gui_<version>_<system>_<arch>` holds the window, `qadg-gui`, with the
  command line beside it. On Windows it also has an `opengl` folder: a software
  renderer the window uses when the graphics driver offers no OpenGL, such as
  in a virtual machine or over a remote desktop.
- `qadg-setup_<version>_windows_amd64.msi` installs the window and the command
  line on Windows for every account on the machine: the window in the Start
  menu, and `qadg` on the PATH, so a new terminal finds it. It asks for an
  administrator. Installing a newer version while `qadg` or the window is
  running does not close them. Windows replaces the files and removes the old
  copy at the next restart. A release candidate has no installer.
- The files whose names start with `verify-` are for checking a download, as
  described below.

**Signatures**

- On Windows, `qadg.exe`, `qadg-gui.exe`, the two files in the `opengl` folder
  and the installer are signed with a code signing certificate issued by
  Certum. Each
  signature carries a timestamp, so it stays valid after the certificate
  expires. Windows shows it in the file's Properties, on the Digital Signatures
  tab.
- On macOS each program is inside an application bundle, `qadg.app` or
  `qadg-gui.app`, signed with an Apple Developer ID and notarised by Apple. The
  notarisation ticket is inside the bundle, so macOS does not have to ask Apple
  over the network before it opens it. Beside each bundle is a link for each
  program in it, so you still type `./qadg`. Keep the bundle and its links
  together: a program copied out of the bundle on its own is refused, because
  the signature covers the bundle.
- On Linux the programs are not signed.

**Checking a download**

`verify-SHA256SUMS.txt` lists the SHA-256 of every file of this release except
itself and the statement made over it.

With the GitHub CLI, for any archive of this release or its installer:

```
gh attestation verify <file> -R donislawdev/qadatagenerator --predicate-type https://spdx.dev/Document/v2.3
```

It checks a statement GitHub signed after the release was signed, which ties
the file to `verify-qadg_<version>.spdx.json`. That file lists every component
of every program in this release with its version and licence, in SPDX 2.3, for
a scanner to read. The predicate type has to be given, because without it `gh`
looks for the other kind of statement, described next.

For a Linux archive or for `verify-qadg_<version>.spdx.json`:

```
gh attestation verify <file> -R donislawdev/qadatagenerator
```

It says which repository, commit and workflow built the file, from a statement
GitHub signed when it was built. It does not answer for a Windows or macOS
archive, because those were signed after the build, so their bytes are no
longer the ones the build made, nor for the installer, which is made from the
signed window archive. Their signatures answer for them instead. On macOS,
after unpacking:

```
spctl -a -vv qadg.app
xcrun stapler validate qadg.app
```

The same two commands answer for `qadg-gui.app`.

`verify-qadg_<version>.provenance.sigstore.json` and
`verify-qadg_<version>.sbom.sigstore.json` are the two statements themselves.
Given to either command with `--bundle <that file>`, they let `gh` read the
statement from the file instead of asking GitHub for it.
