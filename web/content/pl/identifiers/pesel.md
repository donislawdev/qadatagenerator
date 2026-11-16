---
translatedFrom: e305672e198a6e826413a3fd9321da171bbd4eb7
title: PESEL
seoTitle: "Generator PESEL do testów: zgodna data urodzenia i płeć"
description: "Generator numerów PESEL zgodnych z datą urodzenia i płcią w kolumnach obok, z poprawną cyfrą kontrolną, i zepsutych, które aplikacja ma odrzucić."
lead: "Numer PESEL niesie datę urodzenia i płeć. PESEL z programu zawsze zgadza się z jednym i drugim."
sources:
  - "gov.pl, Czym jest numer PESEL, czytane 2 sierpnia 2026."
  - "Ministerstwo Finansów, POLAND - Information on Tax Identification Numbers, w portalu TIN Komisji Europejskiej, z przeliczonym przykładem."
---

## Czym jest PESEL

Jedenaście cyfr: data urodzenia jako rok, miesiąc i dzień, cztery cyfry liczby porządkowej i cyfra kontrolna. Stulecie jest ukryte w miesiącu: od 1 do 12 dla lat 1900, od 21 do 32 dla lat 2000, od 41 do 52 dla lat 2100, od 61 do 72 dla lat 2200 i od 81 do 92 dla lat 1800. Ostatnia cyfra liczby porządkowej mówi o płci: parzysta to kobieta, nieparzysta to mężczyzna.

## Cyfra kontrolna

Pomnóż pierwsze dziesięć cyfr przez 1, 3, 7, 9, 1, 3, 7, 9, 1, 3 i zsumuj iloczyny. Weź ostatnią cyfrę sumy i odejmij ją od 10. Wynik to cyfra kontrolna, a gdy suma kończy się zerem, cyfra kontrolna to 0.

## Poprawne numery PESEL do testów

{{< run "qadg quick pesel --count 5 --gender female --birth-date 1990-04-12 --seed example" >}}

`--gender` i `--birth-date` ustalają, co numer niesie, a bez nich każdy numer dostaje własne. W przepisie reguła `pesel matches: [birth_date, gender]` każe każdemu numerowi PESEL zgadzać się z dwiema kolumnami obok. System obejmuje tylko lata od 1800 do 2299. Nie ma puli numerów PESEL wydzielonej do testów, więc wygenerowany numer może należeć do prawdziwej osoby: trzymaj go w systemach testowych.

## Co aplikacja ma zrobić z zepsutym

Odrzucić PESEL, którego cyfra kontrolna się nie zgadza. Część zamian dwóch sąsiednich cyfr zostawia cyfrę kontrolną poprawną, więc program nie liczy takiej zamiany jako uszkodzenia i mówi o tym w manifeście.

Potem trudniejszy przypadek: PESEL poprawny sam w sobie, ale z inną datą urodzenia albo płcią niż w formularzu obok. Każda z wartości przechodzi każde sprawdzenie z osobna, a zła jest tylko para, więc wyłapie ją tylko formularz, który porównuje PESEL z datą urodzenia i płcią. Manifest oznacza tę odpowiedź jako zależną od aplikacji, bo nie każdy formularz pyta o datę urodzenia i płeć.
