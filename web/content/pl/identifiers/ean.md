---
translatedFrom: 6bce775a053114d94484e9de37b3a9350d132d3c
title: EAN-13
seoTitle: "Kody EAN-13 do testów, które nie należą do żadnego produktu"
description: "Jak działa cyfra kontrolna numeru EAN-13 i jak wygenerować poprawne numery z prefiksem 952, który GS1 wydziela na przykłady, oraz zepsute do testów."
lead: "EAN-13 to 13-cyfrowy numer pod większością kodów kreskowych w sklepach. Każdy z programu zaczyna się od 952, prefiksu, który GS1 trzyma na przykłady."
sources:
  - "GS1 General Specifications, Release 26.0, styczeń 2026: tabele 1-4 i 1-5 (prefiks 952), tabela 7-8 (cyfra kontrolna)."
  - "International ISBN Agency, ISBN Users' Manual, dodatek A1.1."
---

## Czym jest EAN-13

Trzynaście cyfr, z których ostatnia to cyfra kontrolna. GS1 nazywa go GTIN-13. Pierwsze cyfry to prefiks GS1 organizacji, która wydała numer, a ISBN to ten sam rodzaj numeru z prefiksem 978 albo 979.

## Cyfra kontrolna

Licząc od prawej, pomnóż cyfrę obok cyfry kontrolnej przez 3, następną przez 1 i dalej na przemian. Zsumuj iloczyny i odejmij sumę od najbliższej wielokrotności 10, która jest jej równa albo większa. Różnica to cyfra kontrolna.

Przy 13 cyfrach to to samo co wagi 1 i 3 od lewej, tak jak zapisuje to podręcznik ISBN. GS1 liczy od prawej, bo jego dłuższe numery wyrównuje się do prawej.

## Poprawne numery EAN-13 do testów

{{< run "qadg quick ean --count 5 --seed example" >}}

GS1 trzyma prefiks 952 na pokazy i przykłady systemu GS1 i sam umieszcza tam swoje przykłady, więc żaden produkt nie dostaje numeru z tym prefiksem. GS1 wydzielił go w wydaniu ze stycznia 2023, a jego specyfikacja nie mówi, czy któraś firma dostała tam numer wcześniej.

## Co aplikacja ma zrobić z zepsutym

Odrzucić numer, którego cyfra kontrolna się nie zgadza. Zmiana jednej cyfry zawsze wychodzi na jaw. Zamiana miejscami dwóch sąsiednich cyfr różniących się o 5 już nie, więc program nie liczy takiej zamiany jako uszkodzenia i mówi o tym w manifeście.
