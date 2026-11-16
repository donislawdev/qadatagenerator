---
translatedFrom: 417efb6a45bb698c766db38abcd7d92aeab13d91
title: SSN
seoTitle: "Numery SSN do testów, które nie należą do nikogo"
description: "Numery SSN do danych testowych: dlaczego numery od 666 nigdy nie są przydzielane, jakie zasady ma SSN i jak wygenerować tysiące naraz."
lead: "Każdy SSN z programu zaczyna się od 666, czyli od obszaru, którego Social Security Administration nigdy nie przydziela, więc żaden z nich nie jest niczyim numerem."
sources:
  - "Social Security Administration, Social Security Number Randomization i jej FAQ, ssa.gov/employer, czytane 2 sierpnia 2026."
  - "Social Security Administration, Program Operations Manual System, RM 10201.020."
---

## Czym jest SSN

Dziewięć cyfr w zapisie 123-45-6789: trzycyfrowy obszar, dwucyfrowa grupa i czterocyfrowy numer seryjny. Nie ma cyfry kontrolnej, więc literówki nie wyłapie żaden rachunek. Da się sprawdzić tylko, czy numer leży w części zakresu, której nigdy się nie przydziela.

## Numery, których nigdy się nie przydziela

- obszar 000, obszar 666 i obszary od 900 do 999,
- grupa 00,
- numer seryjny 0000.

Obszary od 900 do 999 nie są wolne. Mają kształt numerów ITIN (Individual Taxpayer Identification Number), które należą do prawdziwych osób. Zostaje więc 666 jako jedyny obszar, który wygląda zwyczajnie i nie należy do nikogo.

Do pojedynczego numeru w reklamie albo na pokazie Social Security Administration sama radzi użyć 000-00-0000.

## Poprawne numery SSN do testów

{{< run "qadg quick ssn --count 5 --seed example" >}}

`--count 1000` daje tysiąc. Aplikacja, która sprawdza, czy obszar bywa przydzielany, odrzuci każdy z nich, także w poprawnych wierszach. To jest cena numerów, które nie mogą do nikogo należeć.

## Co aplikacja ma zrobić z zepsutym

Uszkodzenie przenosi numer do obszaru 000, grupy 00 albo numeru seryjnego 0000, a takie numery odrzuci każdy walidator, który zna zasady SSN. Manifest oznacza tę odpowiedź jako zależną od aplikacji, bo takie sprawdzenie wymaga znajomości zasad, a nie rachunku, i wiele systemów jej nie ma.
