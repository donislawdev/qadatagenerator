---
translatedFrom: fd9421d0b58ecff97d3bbbc3618ec3cf67650343
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
- **Miejsca.** Miasto, jego region i kod pocztowy pasują do siebie. W `en-US` region to stan, a kod pocztowy to kod ZIP.
- **PESEL.** Numer PESEL niesie tę samą datę urodzenia i płeć co kolumny obok niego.

## Cyfry kontrolne i numery testowe

Każdy identyfikator ma poprawną cyfrę kontrolną. Numery kart to wyłącznie opublikowane numery testowe, nigdy numer prawdziwej karty. Numery SSN zaczynają się od 666, czyli od obszaru, którego nigdy się nie przydziela, więc wygenerowany SSN nie może należeć do nikogo. Aplikacja, która sprawdza obszar, odrzuci je, także w poprawnych wierszach.

## Listy słów

Imiona, nazwiska, miasta i regiony pochodzą z list słów dla `en-US` i `pl-PL`. Wybiera je `--locale` albo `locale:` w przepisie.

## Jedna wartość bez przepisu

{{< readme marker="run" kind="run" output="output" >}}

`--count` ustala, ile ich ma być, a `--seed` sprawia, że wartości da się powtórzyć. Bez seeda każdy przebieg losuje nowy i podaje go w kanale logu, więc wartość, która ci się spodobała, da się dostać jeszcze raz.
