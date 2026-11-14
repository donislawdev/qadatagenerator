---
translatedFrom: 4b0cead580162c68819556d106dd3bf866eb1841
title: Użycie w CI
seoTitle: "Dane testowe w CI: seed, kody wyjścia, JSON"
description: "Uruchom QA Data Generator w pipelinie: ten sam seed daje te same bajty, dane i log idą osobnymi kanałami, a każde zakończenie ma własny kod wyjścia."
lead: "Wszystko, czego potrzebuje pipeline, jest w wierszu poleceń: powtarzalne dane, wynik czytelny dla skryptu i kod wyjścia dla każdego zakończenia."
---

## Commituj przepis, nie dane

Ten sam przepis i ten sam seed dają te same wiersze i te same uszkodzenia, bajt w bajt, na Windows, macOS i Linuksie. Zamiast dużych plików z danymi trzymaj obok testów mały przepis i generuj dane w pipelinie.

`qadg quick` to jedyne polecenie, którego przebiegi się różnią. Bez `--seed` za każdym razem losuje nowy seed i podaje go w kanale logu.

## Kanały

Dane idą na standardowe wyjście, a log na standardowe wyjście błędów. Przebieg, który się nie uda, nie zapisuje niczego na standardowe wyjście, a `--json` daje raport czytelny dla maszyny dla `validate`, `generate`, `--version` i `--license`.

## Kody wyjścia

{{< exit-codes >}}

`qadg --help` wypisuje tę samą tabelę, po angielsku.

## Krok, który pada przy złej odpowiedzi

1. Uruchom `qadg generate` na przepisie z `policy` albo `qadg corrupt` na własnym pliku.
2. W teście zaimportuj popsute dane do aplikacji i zapisz, co zrobiła z każdym wierszem.
3. Uruchom `qadg verify` na manifeście i tym zapisie. Kod wyjścia 4 kończy krok porażką.

[Sprawdź, co zrobiła aplikacja](/docs/verify/) pokazuje taki zapis i to, co wypisuje `verify`.
