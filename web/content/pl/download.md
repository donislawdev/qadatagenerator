---
translatedFrom: 7514b03cc22f9a282af88c9dcc7a4b0072b9b17d
title: Instalacja QA Data Generator
linkTitle: Instalacja
description: "Pobierz QA Data Generator na Windows, macOS albo Linuksa, zainstaluj go na Windows albo zbuduj ze źródeł za pomocą Go."
lead: "Pobierz wiersz poleceń albo okno na swój system i rozpakuj je albo zainstaluj na Windows. Programy są podpisane na Windows i macOS."
readmeTables:
  - heading: Download and run
    rows:
      - from: "Your system | The command line, `qadg` | The window, `qadg-gui`, with `qadg` beside it"
        to: ["Twój system", "Wiersz poleceń, `qadg`", "Okno, `qadg-gui`, z `qadg` obok"]
      - from: "Windows on x64 | `qadg_<version>_windows_amd64.zip` | `qadg-gui_<version>_windows_amd64.zip`"
        to: ["Windows na x64", "`qadg_<version>_windows_amd64.zip`", "`qadg-gui_<version>_windows_amd64.zip`"]
      - from: "Windows on Arm | `qadg_<version>_windows_arm64.zip` | not built"
        to: ["Windows na Arm", "`qadg_<version>_windows_arm64.zip`", "brak"]
      - from: "Linux on x64 | `qadg_<version>_linux_amd64.tar.gz` | `qadg-gui_<version>_linux_amd64.tar.gz`"
        to: ["Linux na x64", "`qadg_<version>_linux_amd64.tar.gz`", "`qadg-gui_<version>_linux_amd64.tar.gz`"]
      - from: "Linux on Arm | `qadg_<version>_linux_arm64.tar.gz` | not built"
        to: ["Linux na Arm", "`qadg_<version>_linux_arm64.tar.gz`", "brak"]
      - from: "macOS on Apple silicon | `qadg_<version>_macos_arm64.tar.gz` | `qadg-gui_<version>_macos_arm64.tar.gz`"
        to: ["macOS na Apple silicon", "`qadg_<version>_macos_arm64.tar.gz`", "`qadg-gui_<version>_macos_arm64.tar.gz`"]
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

## Pobierz

Weź archiwum dla swojego systemu z [najnowszego wydania](https://github.com/donislawdev/qadatagenerator/releases/latest) i rozpakuj je. Programy działają z folderu, do którego je rozpakujesz, bez instalatora i bez uprawnień administratora.

{{< readme-table heading="Download and run" >}}

Archiwum okna zawiera też wiersz poleceń. Nie ma wersji dla Maków z procesorem Intela.

- **Na Windows** programy i instalator są podpisane certyfikatem do podpisywania kodu. Archiwum okna ma też folder `opengl`: renderer programowy, którym okno rysuje, gdy sterownik grafiki nie daje OpenGL, na przykład w maszynie wirtualnej albo przez pulpit zdalny. Trzymaj go obok `qadg-gui.exe`.
- **Na macOS** każdy program jest w pakiecie aplikacji, `qadg.app` albo `qadg-gui.app`, podpisanym identyfikatorem Apple Developer ID i poświadczonym przez Apple. Obok każdego pakietu jest dowiązanie do każdego programu w nim, więc nadal wpisujesz `./qadg`. Trzymaj pakiet i jego dowiązania razem: program skopiowany z pakietu osobno zostanie odrzucony, bo podpis obejmuje cały pakiet.
- **Na Linuksie** programy nie są podpisane. Wiersz poleceń to jeden plik, który nie potrzebuje niczego więcej. Okno potrzebuje sesji pulpitu z OpenGL.

## Instalacja na Windows

`qadg-setup_<version>_windows_amd64.msi` instaluje okno i wiersz poleceń w `C:\Program Files\QA Data Generator` dla wszystkich kont na komputerze. Dodaje okno do menu Start, a ten folder do systemowej zmiennej `PATH`, więc terminal otwarty później znajdzie `qadg`. Instalacja prosi o uprawnienia administratora, uruchamianie programów nie.

Nowsza wersja instaluje się na starszą i niczego nie zamyka. Jeśli `qadg` albo okno akurat działa, Windows podmienia pliki i usuwa starą kopię przy następnym uruchomieniu komputera. Starsza wersja nie zainstaluje się na nowszą. Żeby usunąć program, odinstaluj QA Data Generator w Ustawieniach, w sekcji Aplikacje.

## Sprawdź pobrane pliki

`verify-SHA256SUMS.txt` na stronie wydania podaje SHA-256 każdego archiwum i instalatora. Pobierz go obok pobranych plików, a potem:

{{< readme heading="Download and run" kind="run" >}}

Na Windows `Get-FileHash <plik>` w PowerShellu wypisuje SHA-256 do porównania z linią tego pliku. Notatki każdego wydania mówią też, jak sprawdzić plik poleceniem `gh attestation verify`, wobec poświadczeń, które GitHub podpisał dla tego wydania.

## Zbuduj ze źródeł

Potrzebujesz [Go](https://go.dev/dl/) w wersji {{< go-version >}} albo nowszej.

{{< readme heading="Build it from source" kind="run" >}}

Pierwsze `go build` tworzy `qadg`, czyli wiersz poleceń, i nie potrzebuje niczego poza Go. Drugie tworzy `qadg-gui`, czyli okno, do którego potrzebny jest jeszcze kompilator C i nagłówki OpenGL.

## Czego okno potrzebuje wcześniej

{{< readme-table heading="Build it from source" >}}

## Gdy okno nie może się otworzyć na Windows

Archiwum okna dla Windows i instalator mają to już w sobie. Na komputerze z Windows, który nie daje oknu OpenGL, na przykład w maszynie wirtualnej bez akceleracji 3D, w sesji pulpitu zdalnego albo na serwerze, okno może rysować procesorem zamiast karty graficznej. Okno zbudowane samodzielnie potrzebuje do tego Mesy, renderera programowego, w folderze `opengl` obok `qadg-gui.exe`. To polecenie, uruchomione w Git Bash z katalogu głównego repozytorium po zbudowaniu okna, kładzie ją tam:

{{< readme heading="When the window cannot open on Windows" kind="run" >}}

Pobiera około 70 MB, odrzuca pobrany plik, jeśli jego suma kontrolna nie jest przypiętą, i kładzie obok obu plików licencje tego, co z niego wziął. Gdy sterownik grafiki potem odmówi, okno otworzy się rysowane programowo, a `qadg-gui -software-gl` rysuje programowo na każdym komputerze.

## Dalej

[Pierwsze kroki](/docs/getting-started/) prowadzą od pierwszej wartości do popsutego zestawu danych i odpowiedzi dla każdej popsutej wartości.
