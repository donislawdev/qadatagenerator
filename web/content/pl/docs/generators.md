---
translatedFrom: c8cd42b6b2cdada7d50d057baa78cc5d996a3a5e
title: Generatory
seoTitle: "Generatory: PESEL, NIP, REGON, IBAN, VIN, ISBN i inne"
description: "Czym QA Data Generator wypełnia kolumny: PESEL, NIP, REGON, IBAN, imiona, miejsca, e-maile, hasła, UUID, testowe numery kart, ISBN, VIN i SSN."
lead: "Generator wypełnia kolumnę. Przepis wskazuje go kluczem `generator:`, a `qadg quick` wypisuje jego wartości bez przepisu."
---

## Wszystkie generatory

{{< generators >}}

`qadg help generators` wypisuje tę samą tabelę prosto z programu, po angielsku.

## Wartości, które zgadzają się ze sobą

- **Ludzie.** Imiona i nazwiska mogą iść za kolumną płci dzięki `first_name matches: [gender]`.
- **Miejsca.** W `en-US` miasto, jego stan i kod ZIP pasują do siebie. W `pl-PL` miejscowość i województwo są losowane osobno, a kodów pocztowych jeszcze nie ma.
- **PESEL.** Numer PESEL niesie tę samą datę urodzenia i płeć co kolumny obok niego.
- **Adresy e-mail.** Adres powstaje z imienia i nazwiska z tego samego wiersza, bez dodatkowej reguły: wiersz z Adamem Małyszkiem dostaje adres w rodzaju `amalyszek@example.org`. Wiersz bez tych kolumn dostaje imię i nazwisko wylosowane dla samego adresu.

## Cyfry kontrolne i numery testowe

Każdy identyfikator, który ma cyfrę kontrolną, ma ją poprawną. Numery kart to wyłącznie numery testowe publikowane przez operatorów płatności, nigdy numer wymyślony tak, żeby przeszedł cyfrę kontrolną. Numery SSN zaczynają się od 666, czyli od obszaru, którego nigdy się nie przydziela, więc wygenerowany SSN nie może należeć do nikogo. Aplikacja, która sprawdza obszar, odrzuci je, także w poprawnych wierszach. [Identyfikatory](/identifiers/) opisują każdą cyfrę kontrolną i to, które numery mogą pokryć się z prawdziwymi.

## Adresy e-mail, hasła i UUID

Adresy e-mail kończą się na `example.com`, `example.net` albo `example.org`. Te domeny są wydzielone do dokumentacji i nie przyjmują poczty, więc wiadomość wysłana na taki adres przez twój test nie dotrze do nikogo. Litery tracą znaki diakrytyczne, więc polskie imię daje adres bez polskich liter. `max_length` skraca adres, ale nie poniżej 13 znaków, a krótszy limit jest odrzucany, zanim cokolwiek zostanie zapisane.

Hasła są zapisane jawnie, mają od 12 do 16 znaków i każde zawiera małą literę, wielką literę, cyfrę i jeden ze znaków `! # $ % & * + - = ? @ ^ _`, czyli to, czego wymagają typowe reguły haseł. Żadne nie zaczyna się od `=`, `+`, `-` ani `@`, bo arkusz kalkulacyjny wziąłby je za formułę. W Bashu, zsh albo PowerShellu ujmij hasło w pojedyncze cudzysłowy. `max_length` skraca hasła, ale nie poniżej 4 znaków. Hasła pochodzą z seeda, więc każdy, kto ma twój przepis i seed, dostanie te same: używaj ich do kont testowych, nigdy do prawdziwego.

UUID to wersja 4 z RFC 9562, losowa i zapisana małymi literami, a `type: uuid` daje te same wartości co `generator: uuid`. UUID-ów nikt nie wydaje, więc nie istnieje wydany UUID, z którym wygenerowany mógłby się pokryć. UUID nie ma cyfry kontrolnej, więc uszkodzenia wymierzone w cyfry kontrolne go nie ruszają. `max_length` poniżej 36 jest odrzucany, zanim cokolwiek zostanie zapisane.

## Listy słów

Imiona, nazwiska, miasta i regiony pochodzą z list słów dla `en-US` i `pl-PL`. Wybiera je `--locale` albo `locale:` w przepisie.

## Jedna wartość bez przepisu

{{< readme marker="run" kind="run" output="output" >}}

`--count` ustala, ile ich ma być, a `--seed` sprawia, że wartości da się powtórzyć. Bez seeda każdy przebieg losuje nowy i podaje go w kanale logu, więc wartość, która ci się spodobała, da się dostać jeszcze raz.
