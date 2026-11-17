---
title: Get QA Data Generator
linkTitle: Get it
description: "Build QA Data Generator from source on Windows, macOS or Linux: one command for the command line, one more for the window, and what each one needs."
lead: "QA Data Generator is built from its source with Go. One command builds the command line, one more builds the window."
---

## Build it

You need [Go](https://go.dev/dl/) {{< go-version >}} or newer.

{{< readme heading="Build it from source" kind="run" >}}

The first `go build` makes `qadg`, the command line, and needs nothing but Go. The second makes `qadg-gui`, the window, which also needs a C compiler and the OpenGL headers.

## What the window needs first

{{< readme-table heading="Build it from source" >}}

## When the window cannot open on Windows

On a Windows computer that gives the window no OpenGL, such as a virtual machine without 3D acceleration, a remote desktop session or a server, the window can draw with the processor instead. It needs Mesa, a software renderer, in a folder named `opengl` beside `qadg-gui.exe`. This command, run in Git Bash from the root of the repository after the window is built, puts it there:

{{< readme heading="When the window cannot open on Windows" kind="run" >}}

It downloads about 70 MB, refuses the download unless its checksum is the pinned one, and puts the licences of what it took beside the two files. When the graphics driver then refuses, the window opens drawn in software, and `qadg-gui -software-gl` draws in software on any computer.

## Then

[Getting started](/docs/getting-started/) takes you from your first value to a damaged dataset and the answer for every damaged value.
