---
translatedFrom: 546b018c1819b5a71dd3a7f694eba19e7dd456ed
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

## Gdy okno nie może się otworzyć na Windows

Na komputerze z Windows, który nie daje oknu OpenGL, na przykład w maszynie wirtualnej bez akceleracji 3D, w sesji pulpitu zdalnego albo na serwerze, okno może rysować procesorem zamiast karty graficznej. Potrzebuje do tego Mesy, renderera programowego, w folderze `opengl` obok `qadg-gui.exe`. To polecenie, uruchomione w Git Bash z katalogu głównego repozytorium po zbudowaniu okna, kładzie ją tam:

{{< readme heading="When the window cannot open on Windows" kind="run" >}}

Pobiera około 70 MB, odrzuca pobrany plik, jeśli jego suma kontrolna nie jest przypiętą, i kładzie obok obu plików licencje tego, co z niego wziął. Gdy sterownik grafiki potem odmówi, okno otworzy się rysowane programowo, a `qadg-gui -software-gl` rysuje programowo na każdym komputerze.

## Dalej

[Pierwsze kroki](/docs/getting-started/) prowadzą od pierwszej wartości do popsutego zestawu danych i odpowiedzi dla każdej popsutej wartości.
