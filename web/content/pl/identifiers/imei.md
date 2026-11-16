---
translatedFrom: e797bb4c5a85d421546e189cc84f02605b10c401
title: IMEI
seoTitle: "Generator IMEI do testów i jak działa algorytm Luhna"
description: "Jak liczy się 15. cyfra numeru IMEI algorytmem Luhna i jak wygenerować poprawne numery IMEI z puli dla sprzętu testowego oraz zepsute do testów."
lead: "IMEI identyfikuje telefon komórkowy albo modem. Ma 15 cyfr, a ostatnia to cyfra kontrolna Luhna."
sources:
  - "ETSI TS 123 003 wersja 19.7.0, opublikowany tekst 3GPP TS 23.003: punkt 6.2.1 i aneks B."
  - "GSMA TS.06, IMEI Allocation and Approval Process, wersja 28.1: tabela 1 i punkt 9.1."
---

## Czym jest IMEI

Osiem cyfr kodu TAC (Type Allocation Code), który wskazuje model, sześć cyfr numeru seryjnego i jedna cyfra kontrolna. Kody TAC rozdaje GSM Association.

## Cyfra kontrolna

Zacznij od cyfry tuż na lewo od cyfry kontrolnej i podwajaj co drugą cyfrę, idąc w lewo. Zsumuj cyfry podwojonych liczb i cyfry, których nie podwajano. Odejmij sumę od najbliższej większej liczby kończącej się zerem. Różnica to cyfra kontrolna, a suma, która już kończy się zerem, daje 0.

Norma przelicza jeden przykład, `26053179311383`. Podwojone cyfry dają 12, 10, 2, 18, 2, 6 i 6, całość sumuje się do 53, a 60 - 53 to 7, więc IMEI to `260531793113837`.

## Poprawne numery IMEI do testów

{{< run "qadg quick imei --count 5 --seed example" >}}

Każdy IMEI zaczyna się od 00, czyli od puli, którą GSM Association daje sprzętowi testowemu, a takiego sprzętu nie wolno sprzedawać. Żaden telefon w sprzedaży nie nosi więc takiego numeru. Kody w tej puli dostają jednak producenci, więc wartość może pokryć się z jakimś urządzeniem testowym.

## Co aplikacja ma zrobić z zepsutym

Odrzucić IMEI, którego cyfra kontrolna się nie zgadza. Zmiana jednej cyfry zawsze wychodzi na jaw. Spośród zamian dwóch sąsiednich cyfr tylko 0 obok 9 zostawia cyfrę kontrolną poprawną, a program nie liczy takiej zamiany jako uszkodzenia.
