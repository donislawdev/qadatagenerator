---
translatedFrom: b82ccc4f70f0ae5ef37ca3e0b1d16ae3eed774ac
title: VIN
seoTitle: "Generator VIN do testów i jak liczy się cyfra kontrolna"
description: "Jak liczy się dziewiąty znak numeru VIN, dlaczego nie ma w nim liter I, O i Q, i jak wygenerować tysiące poprawnych numerów VIN oraz zepsute do testów."
lead: "Numer identyfikacyjny pojazdu ma 17 znaków, a dziewiąty to cyfra kontrolna liczona z pozostałych szesnastu."
sources:
  - "49 CFR 565, Vehicle Identification Number Requirements, eCFR w stanie na 3 września 2026: § 565.13 (znaki), § 565.15 (cyfra kontrolna i rok modelowy), tabele od III do VII."
---

## Czym jest VIN

Cyfry i wielkie litery bez I, O i Q, żeby nic nie dało się przeczytać jako 1 albo 0. Dziewiąty znak to cyfra kontrolna, a dziesiąty to rok modelowy. Zasady z tej strony to zasady amerykańskie z przepisu federalnego 49 CFR 565, który przelicza też jeden przykład krok po kroku.

## Cyfra kontrolna

1. Każdemu znakowi nadaj wartość. Cyfra jest warta tyle, ile wynosi. Litera jest warta tyle, ile podaje przepis: A i J to 1, B, K i S to 2, i tak dalej aż do 9.
2. Pomnóż każdą wartość przez wagę jej miejsca: 8, 7, 6, 5, 4, 3, 2, 10, 0, 9, 8, 7, 6, 5, 4, 3, 2. Dziewiąte miejsce ma wagę 0, więc cyfra kontrolna nie liczy się do własnej sumy.
3. Zsumuj iloczyny i podziel przez 11. Reszta to cyfra kontrolna, a resztę 10 zapisuje się jako X.

Przykład z samego przepisu to `1G4AH59H45G118341`. Iloczyny sumują się do 411, a 411 dzielone przez 11 daje resztę 4, czyli dziewiąty znak.

## Poprawne numery VIN do testów

{{< run "qadg quick vin --count 5 --seed example" >}}

Każdy VIN kończy się pięcioma cyframi, jak musi się kończyć VIN samochodu osobowego, a jego dziesiąty znak to kod roku modelowego z przepisu. `--count 1000` daje tysiąc. Wygenerowany VIN może pokryć się z numerem prawdziwego pojazdu, więc trzymaj go w systemach testowych.

## Co aplikacja ma zrobić z zepsutym

Odrzucić VIN, którego dziewiąty znak się nie zgadza. Najpierw dwie rzeczy. Pojazd zbudowany poza zasięgiem przepisów amerykańskich może mieć VIN bez cyfry kontrolnej, a program, czytając VIN, stosuje regułę amerykańską, więc uzna taki numer za zepsuty. Poza tym zmiana litery przechodzi niezauważona, gdy nowa litera ma tę samą wartość, na przykład J zamiast A, więc cyfra kontrolna nie wyłapie każdej literówki.
