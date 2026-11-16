---
translatedFrom: ae0a946821121901c049581cc36e306fe95a628f
title: ISBN
seoTitle: "Generator ISBN do testów i cyfra kontrolna ISBN-13"
description: "Jak liczy się ostatnia cyfra ISBN-13, dlaczego program nie robi ISBN-10 i jak wygenerować poprawne numery ISBN, których nie nosi żadna książka, oraz zepsute."
lead: "ISBN ma 13 cyfr i zaczyna się od 978 albo 979. Ostatnia cyfra to cyfra kontrolna, liczona tak samo jak w kodzie kreskowym."
sources:
  - "International ISBN Agency, ISBN Users' Manual, wydanie siódme, 2017, dodatek A1.1."
  - "International ISBN Agency, plik zakresów numerów ISBN, czytany 7 września 2026."
  - "International ISMN Agency, dla numerów zaczynających się od 979-0."
---

## Czym jest ISBN

Od 1 stycznia 2007 każdy ISBN ma 13 cyfr. ISBN-13 jest zarazem numerem kodu kreskowego EAN-13 i dlatego zaczyna się od 978 albo 979. Numer zaczynający się od 979-0 nie jest w ogóle numerem ISBN, tylko ISMN, czyli numerem wydania nutowego, i program mówi to wprost, zamiast nazwać go zepsutym ISBN.

Program nie robi ISBN-10. Podręcznik ISBN nie podaje reguły jego cyfry kontrolnej, a od 2007 nie wydaje się już numerów dziesięciocyfrowych.

## Cyfra kontrolna

Pomnóż pierwsze dwanaście cyfr na przemian przez 1 i 3, zaczynając od 1 z lewej. Zsumuj iloczyny, weź resztę z dzielenia przez 10 i odejmij ją od 10. Wynik to cyfra kontrolna, a wynik 10 zapisuje się jako 0.

Podręcznik przelicza własny numer ISBN, `978-92-95055-12-4`. Iloczyny to 9, 21, 8, 27, 2, 27, 5, 0, 5, 15, 1 i 6, ich suma to 126, reszta to 6, a 10 - 6 to 4.

## Poprawne numery ISBN do testów

{{< run "qadg quick isbn --count 5 --seed example" >}}

Każdy ISBN zaczyna się od 978-67, 978-68 albo 978-69, czyli od puli, której agencja ISBN nie dała żadnej grupie rejestracyjnej, gdy sprawdzono to ostatnio, 7 września 2026. Dziś żadna książka nie nosi takiego numeru. Agencja może zacząć rozdawać tę pulę później, więc nie ma obietnicy, że zostanie pusta.

## Co aplikacja ma zrobić z zepsutym

Odrzucić ISBN, którego ostatnia cyfra się nie zgadza. Zmiana jednej cyfry zawsze wychodzi na jaw. Zamiana miejscami dwóch sąsiednich cyfr różniących się o 5, na przykład 2 i 7, nie zmienia cyfry kontrolnej, więc program nie liczy takiej zamiany jako uszkodzenia i mówi o tym w manifeście.
