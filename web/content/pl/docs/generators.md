---
translatedFrom: 7805952065891787f765292e19658e723acc0355
title: Generatory
seoTitle: "Generatory: PESEL, NIP, REGON, IBAN, VIN, ISBN i inne"
description: "Czym QA Data Generator wypełnia kolumny: PESEL, NIP, REGON, IBAN, imiona i miejsca, testowe numery kart, ISBN, EAN, IMEI, VIN, SSN i numery ABA."
lead: "Generator wypełnia kolumnę. Przepis wskazuje go kluczem `generator:`, a `qadg quick` wypisuje jego wartości bez przepisu."
---

## Wszystkie generatory

{{< generators >}}

`qadg help generators` wypisuje tę samą tabelę prosto z programu, po angielsku.

## Wartości, które zgadzają się ze sobą

- **Ludzie.** Imiona i nazwiska mogą iść za kolumną płci dzięki `first_name matches: [gender]`.
- **Miejsca.** W `en-US` miasto, jego stan i kod ZIP pasują do siebie. W `pl-PL` miejscowość i województwo są losowane osobno, a kodów pocztowych jeszcze nie ma.
- **PESEL.** Numer PESEL niesie tę samą datę urodzenia i płeć co kolumny obok niego.

## Cyfry kontrolne i numery testowe

Każdy identyfikator, który ma cyfrę kontrolną, ma ją poprawną. Numery kart to wyłącznie numery testowe publikowane przez operatorów płatności, nigdy numer wymyślony tak, żeby przeszedł cyfrę kontrolną. Numery SSN zaczynają się od 666, czyli od obszaru, którego nigdy się nie przydziela, więc wygenerowany SSN nie może należeć do nikogo. Aplikacja, która sprawdza obszar, odrzuci je, także w poprawnych wierszach. [Identyfikatory](/identifiers/) opisują każdą cyfrę kontrolną i to, które numery mogą pokryć się z prawdziwymi.

## Listy słów

Imiona, nazwiska, miasta i regiony pochodzą z list słów dla `en-US` i `pl-PL`. Wybiera je `--locale` albo `locale:` w przepisie.

## Jedna wartość bez przepisu

{{< readme marker="run" kind="run" output="output" >}}

`--count` ustala, ile ich ma być, a `--seed` sprawia, że wartości da się powtórzyć. Bez seeda każdy przebieg losuje nowy i podaje go w kanale logu, więc wartość, która ci się spodobała, da się dostać jeszcze raz.
