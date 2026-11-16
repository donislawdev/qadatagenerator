---
translatedFrom: 5bc3b5b6cbd7c6e665aaa241f093e0cefa114ce4
title: Każda wartość jest zmyślona
seoTitle: "Każda wartość jest zmyślona: prywatność i bezpieczeństwo"
description: "Każda wartość z QA Data Generator jest zmyślona. Żadnych prawdziwych osób, połączeń z siecią ani telemetrii. Co to znaczy dla SSN, numerów kart i twoich plików."
lead: "Dane testowe muszą wyglądać prawdziwie, żeby czegokolwiek dowieść, i nigdy nie mogą należeć do nikogo. Ta strona mówi, jak program pilnuje obu tych rzeczy i co obiecuje w sprawie twoich plików."
---

## Każda wartość jest zmyślona

Imiona, nazwiska, adresy, identyfikatory, numery kart i rachunków: żadne z nich nie opisuje prawdziwej osoby, firmy ani instytucji, a do ich zbudowania nie użyto żadnego rejestru prawdziwych osób. Gdy wartość wygląda prawdziwie, to zasługa arytmetyki, bo test, który widzi tylko wyraźnie fałszywe dane, nie mówi nic o dniu, w którym przyjdą prawdziwe. Jeśli rozpoznajesz którąś z wartości, to przypadek.

- **Numery SSN zaczynają się od 666.** Tego obszaru nigdy się nie przydziela, więc wygenerowany SSN nie może należeć do nikogo. Aplikacja, która sprawdza obszar, odrzuci je, także w poprawnych wierszach.
- **Numery kart to wyłącznie numery testowe publikowane przez operatorów płatności.** Program nigdy nie wymyśla numeru, który tylko przechodzi cyfrę kontrolną. Żaden z operatorów nie obiecuje, że żaden bank nie wydał tego samego numeru prawdziwej karcie, więc trzymaj je w systemach testowych jak resztę.
- **Niektóre systemy numeracji nie mają puli wydzielonej do testów.** Wartość z takiego systemu może pokryć się z numerem, który naprawdę wydano, i okno mówi to przy niej. [SECURITY.md](https://github.com/donislawdev/qadatagenerator/blob/main/SECURITY.md) wymienia, które systemy mają taką pulę, a które nie.

Wygenerowany zestaw danych jest bezpieczny w środowisku testowym i nie wolno go przedstawiać jako danych kogokolwiek. Oznaczaj go jako syntetyczny wszędzie, dokąd trafi. `qadg --license` wypisuje to samo zastrzeżenie, bo strona internetowa nie podróżuje razem z plikiem danych.

## Żadnych połączeń z siecią

Program nie otwiera żadnego połączenia sieciowego: bez konta, bez telemetrii, bez sprawdzania aktualizacji. Działa tak samo na maszynie, która w ogóle nie ma sieci.

## Twoje pliki zostają nietknięte

- Przebieg nie zapisuje niczego, dopóki się nie powiedzie, więc przebieg, który się nie uda albo zostanie przerwany, zostawia katalog wyjściowy taki, jaki zastał.
- Istniejący plik zatrzymuje przebieg, chyba że dodasz `--overwrite`.
- Pliku podanego `qadg corrupt` program nigdy nie nadpisuje. Popsuta kopia zapisuje się obok niego pod inną nazwą.

## Dane są twoje

Kod jest na licencji GPL-3.0. Dane, pliki i manifesty, które wygenerujesz, należą do ciebie: licencja ich nie obejmuje i możesz ich używać w produktach o zamkniętym kodzie i w komercyjnych zestawach testów.

## Zgłaszanie problemu z bezpieczeństwem

[SECURITY.md](https://github.com/donislawdev/qadatagenerator/blob/main/SECURITY.md) mówi, jak zgłosić podatność bez ujawniania jej publicznie.
