---
title: Get QA Data Generator
linkTitle: Get it
description: "Download QA Data Generator for Windows, macOS or Linux, install it on Windows, or build it from source with Go."
lead: "Download the command line or the window for your system and unpack it, or install it on Windows. The programs are signed on Windows and macOS."
---

## Download

Take the archive for your system from the [latest release](https://github.com/donislawdev/qadatagenerator/releases/latest) and unpack it. The programs run from the folder you unpack them to, with no installer and no administrator rights.

{{< readme-table heading="Download and run" >}}

The window's archive carries the command line as well. There is no build for Macs with an Intel processor.

- **On Windows** the programs and the installer are signed with a code signing certificate. The window's archive also has an `opengl` folder, a software renderer the window draws with when the graphics driver offers no OpenGL, such as in a virtual machine or over a remote desktop. Keep it beside `qadg-gui.exe`.
- **On macOS** each program is inside an application bundle, `qadg.app` or `qadg-gui.app`, signed with an Apple Developer ID and notarised by Apple. Beside each bundle is a link for each program in it, so you still type `./qadg`. Keep the bundle and its links together: a program copied out of the bundle on its own is refused, because the signature covers the bundle.
- **On Linux** the programs are not signed. The command line is one file that needs nothing else. The window needs a desktop session with OpenGL.

## Install it on Windows

`qadg-setup_<version>_windows_amd64.msi` installs the window and the command line in `C:\Program Files\QA Data Generator` for every account on the machine. It puts the window in the Start menu and that folder on the system `PATH`, so a terminal opened afterwards finds `qadg`. Installing asks for administrator rights, running the programs does not.

A newer version installs over an older one and closes nothing. If `qadg` or the window is running at the time, Windows replaces the files and removes the old copy at the next restart. An older version does not install over a newer one. To remove it, uninstall QA Data Generator in Settings, under Apps.

## Check what you downloaded

`verify-SHA256SUMS.txt` on the release page lists the SHA-256 of every archive and of the installer. Download it next to what you downloaded, then:

{{< readme heading="Download and run" kind="run" >}}

On Windows, `Get-FileHash <file>` in PowerShell prints the SHA-256 to compare with the line of that file. The notes of every release also say how to check a file with `gh attestation verify`, against statements GitHub signed about that release.

## Build it from source

You need [Go](https://go.dev/dl/) {{< go-version >}} or newer.

{{< readme heading="Build it from source" kind="run" >}}

The first `go build` makes `qadg`, the command line, and needs nothing but Go. The second makes `qadg-gui`, the window, which also needs a C compiler and the OpenGL headers.

## What the window needs first

{{< readme-table heading="Build it from source" >}}

## When the window cannot open on Windows

The window's archive for Windows and the installer carry this already. On a Windows computer that gives the window no OpenGL, such as a virtual machine without 3D acceleration, a remote desktop session or a server, the window can draw with the processor instead. A window you built yourself needs Mesa, a software renderer, in a folder named `opengl` beside `qadg-gui.exe`. This command, run in Git Bash from the root of the repository after the window is built, puts it there:

{{< readme heading="When the window cannot open on Windows" kind="run" >}}

It downloads about 70 MB, refuses the download unless its checksum is the pinned one, and puts the licences of what it took beside the two files. When the graphics driver then refuses, the window opens drawn in software, and `qadg-gui -software-gl` draws in software on any computer.

## Then

[Getting started](/docs/getting-started/) takes you from your first value to a damaged dataset and the answer for every damaged value.
