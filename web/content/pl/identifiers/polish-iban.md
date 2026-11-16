---
translatedFrom: d8b52d56753230be8f31551a9d179cc5bc4ae703
title: IBAN (Polska)
seoTitle: "Generator IBAN do testów: polski numer rachunku i NRB"
description: "Jak liczy się dwie cyfry kontrolne polskiego numeru IBAN, co zawiera 26 cyfr po PL i jak wygenerować poprawne numery rachunków oraz zepsute do testów."
lead: "Polski IBAN to PL i 26 cyfr. Te same 26 cyfr bez PL to numer rachunku z krajowego przelewu, czyli NRB."
sources:
  - "Zarządzenie nr 7/2017 Prezesa NBP: § 13 i § 14 (budowa numeru), załącznik nr 2 (cyfra kontrolna numeru rozliczeniowego)."
  - "Zarządzenie nr 5/2002 Prezesa NBP, załączniki nr 1 i 2 (cyfry kontrolne). Zarządzenie zostało już uchylone."
---

## Czym jest polski IBAN

28 znaków: PL, dwie cyfry kontrolne, ośmiocyfrowy numer rozliczeniowy banku i szesnastocyfrowy numer rachunku. Ostatnia cyfra numeru rozliczeniowego to jego własna cyfra kontrolna.

## Cyfry kontrolne

1. Przenieś PL i dwie cyfry kontrolne na koniec.
2. Zamień P na 25, a L na 21.
3. Przeczytaj całość jako jedną długą liczbę i podziel ją przez 97. IBAN jest poprawny, gdy reszta wynosi 1.

Żeby policzyć cyfry kontrolne, zrób to samo z 00 w ich miejscu i odejmij resztę od 98.

Własna cyfra kontrolna numeru rozliczeniowego: pomnóż jego pierwsze siedem cyfr przez 3, 9, 7, 1, 3, 9, 7, zsumuj iloczyny i odejmij ostatnią cyfrę sumy od 10. Wynik 10 zapisuje się jako 0.

## Poprawne polskie numery IBAN do testów

{{< run "qadg quick iban --count 5 --seed example" >}}

Program robi tylko polskie numery IBAN. Gdy dostanie IBAN z innego kraju, sprawdzi cyfry kontrolne i, jeśli się zgadzają, powie, że reszty nie umie ocenić, bo nie ma listy długości używanych w innych krajach. Numer rozliczeniowy jest losowany, a nie brany z listy banków, ale i tak może pokryć się z numerem prawdziwego banku, a numer rachunku z prawdziwym rachunkiem: trzymaj wygenerowane numery w systemach testowych.

## Co aplikacja ma zrobić z zepsutym

Odrzucić IBAN, którego cyfry kontrolne się nie zgadzają. Uszkodzenie zmienia jedną cyfrę albo zamienia miejscami dwie sąsiednie. Gdy zamiana zostawia cyfry kontrolne poprawne, program nie liczy jej jako uszkodzenia i mówi o tym w manifeście.
