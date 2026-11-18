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
- `verify-qadg_<version>.spdx.json` lists every component of every program in
  this release and its licence, in SPDX 2.3, for a scanner to read.

**Checking where a file came from**

With the GitHub CLI, for any file of this release:

```
gh attestation verify <file> -R donislawdev/qadatagenerator
```

It says which repository, commit and workflow built the file, from a statement
GitHub signed when it was built.
