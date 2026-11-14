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

## Then

[Getting started](/docs/getting-started/) takes you from your first value to a damaged dataset and the answer for every damaged value.
