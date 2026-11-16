---
translatedFrom: 31771ced3e47f9770a3914c9570ed5dbc5c9f588
title: NIP
seoTitle: "Generator NIP do testów i jak liczy się cyfra kontrolna"
description: "Jak liczy się ostatnia cyfra numeru NIP i jak wygenerować poprawne numery NIP oraz zepsute, które aplikacja ma odrzucić."
lead: "NIP to polski numer identyfikacji podatkowej. Ma dziesięć cyfr, a ostatnia to cyfra kontrolna."
sources:
  - "Ministerstwo Finansów, POLAND - Information on Tax Identification Numbers, w portalu TIN Komisji Europejskiej, czytane 2 sierpnia 2026."
---

## Czym jest NIP

Dziesięć cyfr. Trzy pierwsze to kod urzędu skarbowego, który nadał numer. Poza tym numer nic nie mówi o właścicielu.

## Cyfra kontrolna

Pomnóż pierwsze dziewięć cyfr przez 6, 5, 7, 2, 3, 4, 5, 6, 7, zsumuj iloczyny i podziel przez 11. Reszta to cyfra kontrolna. Reszta 10 nie mieści się w jednej cyfrze, więc takie dziewięć cyfr nie zaczyna żadnego poprawnego numeru NIP.

## Poprawne numery NIP do testów

{{< run "qadg quick nip --count 5 --seed example" >}}

Program nigdy nie wybiera dziewięciu cyfr, które dałyby resztę 10. Nie ma puli numerów NIP wydzielonej do testów, więc wygenerowany numer może pokryć się z prawdziwym: trzymaj go w systemach testowych.

## Co aplikacja ma zrobić z zepsutym

Odrzucić NIP, którego cyfra kontrolna się nie zgadza, i taki, którego pierwsze dziewięć cyfr daje resztę 10. Zmiana jednej cyfry zawsze wychodzi na jaw. Część zamian dwóch sąsiednich cyfr już nie, więc program nie liczy takiej zamiany jako uszkodzenia i mówi o tym w manifeście.
