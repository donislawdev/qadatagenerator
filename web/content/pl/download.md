---
translatedFrom: fa20d989b6e3db41795fec27e5c37634428bb42e
title: Instalacja QA Data Generator
linkTitle: Instalacja
description: "Zbuduj QA Data Generator ze źródeł na Windows, macOS albo Linuksie: jedno polecenie dla wiersza poleceń, drugie dla okna, i czego każde potrzebuje."
lead: "QA Data Generator buduje się ze źródeł za pomocą Go. Jedno polecenie buduje wiersz poleceń, drugie okno."
readmeTables:
  - heading: Build it from source
    rows:
      - from: "System | What to install first"
        to: ["System", "Co zainstalować wcześniej"]
      - from: "Windows | a MinGW-w64 `gcc` on your `PATH`"
        to: ["Windows", "`gcc` z MinGW-w64 w zmiennej `PATH`"]
      - from: "macOS | the Xcode command line tools, `xcode-select --install`"
        to: ["macOS", "narzędzia wiersza poleceń Xcode, `xcode-select --install`"]
      - from: "Debian or Ubuntu | `sudo apt-get install gcc libgl1-mesa-dev xorg-dev libwayland-dev libxkbcommon-dev`"
        to: ["Debian albo Ubuntu", "`sudo apt-get install gcc libgl1-mesa-dev xorg-dev libwayland-dev libxkbcommon-dev`"]
---

## Zbuduj go

Potrzebujesz [Go](https://go.dev/dl/) w wersji {{< go-version >}} albo nowszej.

{{< readme heading="Build it from source" kind="run" >}}

Pierwsze `go build` tworzy `qadg`, czyli wiersz poleceń, i nie potrzebuje niczego poza Go. Drugie tworzy `qadg-gui`, czyli okno, do którego potrzebny jest jeszcze kompilator C i nagłówki OpenGL.

## Czego okno potrzebuje wcześniej

{{< readme-table heading="Build it from source" >}}

## Dalej

[Pierwsze kroki](/docs/getting-started/) prowadzą od pierwszej wartości do popsutego zestawu danych i odpowiedzi dla każdej popsutej wartości.
