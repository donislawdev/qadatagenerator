---
translatedFrom: 810a3249f79b3166809d24ab67cbd46a3cba5120
title: Przepisy
seoTitle: "Przepisy danych testowych w YAML: każdy klucz"
description: "Opisz tabele, liczbę wierszy, relacje i reguły kolumn w jednym pliku YAML. Każdy klucz przepisu, co robi i jak validate zgłasza problem."
lead: "Przepis to plik YAML, który trzymasz obok testów. Mówi, jakie tabele utworzyć, ile wierszy ma każda, jak są powiązane i czym wypełnić każdą kolumnę."
readmeTables:
  - heading: Recipes
    rows:
      - from: "Key | Where | What it does"
        to: ["Klucz", "Gdzie", "Co robi"]
      - from: "`seed` | top | makes the run repeatable"
        to: ["`seed`", "główny poziom", "sprawia, że przebieg da się powtórzyć"]
      - from: "`locale` | top | which word lists to draw names and places from"
        to: ["`locale`", "główny poziom", "z których list słów brać imiona i miejsca"]
      - from: "`count` | entity | how many rows"
        to: ["`count`", "encja", "ile wierszy"]
      - from: "`belongs_to` | entity | the parent table and how many rows each parent gets, such as `{ customer: 0-5 }`"
        to: ["`belongs_to`", "encja", "tabela nadrzędna i to, ile wierszy dostaje każdy jej wiersz, na przykład `{ customer: 0-5 }`"]
      - from: "`primary_key` | entity | the column, or columns, that identify a row"
        to: ["`primary_key`", "encja", "kolumna albo kolumny, które identyfikują wiersz"]
      - from: "`consistency` | entity | rules such as `first_name matches: [gender]`"
        to: ["`consistency`", "encja", "reguły takie jak `first_name matches: [gender]`"]
      - from: "`generator` | field | what fills the column"
        to: ["`generator`", "pole", "czym wypełnić kolumnę"]
      - from: "`values` | field | a list to draw from, such as `[free, pro, team]`"
        to: ["`values`", "pole", "lista, z której losować, na przykład `[free, pro, team]`"]
      - from: "`type: integer` or `type: decimal` | field | a number"
        to: ["`type: integer` albo `type: decimal`", "pole", "liczba"]
      - from: "`unique` | field | no value repeats"
        to: ["`unique`", "pole", "żadna wartość się nie powtarza"]
      - from: "`required` | field | the column is never empty"
        to: ["`required`", "pole", "kolumna nigdy nie jest pusta"]
      - from: "`max_length` | field | the longest value allowed, counted in characters unless `counted_in` says `bytes` or `graphemes`"
        to: ["`max_length`", "pole", "najdłuższa dozwolona wartość, liczona w znakach, chyba że `counted_in` mówi `bytes` albo `graphemes`"]
      - from: "`empty` | field | the share of rows left empty, such as `15%`"
        to: ["`empty`", "pole", "udział wierszy zostawionych pustych, na przykład `15%`"]
      - from: "`policy` | top | what to damage: `layers`, the share of `rows`, `per_row` and `exclude_fields`"
        to: ["`policy`", "główny poziom", "co popsuć: `layers`, udział wierszy `rows`, `per_row` i `exclude_fields`"]
      - from: "`output` | top | `format` (`csv` or `json`), `dir`, `file_per_entity` and the CSV `delimiter`"
        to: ["`output`", "główny poziom", "`format` (`csv` albo `json`), `dir`, `file_per_entity` i separator CSV `delimiter`"]
---

## Przepis od początku do końca

{{< readme marker="file shop.yaml" caption="shop.yaml" >}}

Ten przepis tworzy tysiąc klientów i od zera do pięciu płatności dla każdego. Klienci są w `en-US`, więc region to stan, a kod pocztowy to kod ZIP. `seed` sprawia, że każdy przebieg daje te same wiersze.

## Wszystkie klucze

{{< readme-table heading="Recipes" >}}

## Sprawdź przepis przed uruchomieniem

`qadg validate` czyta przepis i zgłasza wszystkie problemy naraz, zamiast zatrzymać się na pierwszym. Przepis, który przejdzie, mówi, co zapisałby przebieg:

{{< readme marker="run" nth="2" kind="run" output="output 1" >}}

Ekran Recipe w oknie sprawdza przepis tak samo, w trakcie pisania.

## Powiązane tabele

`belongs_to` wskazuje tabelę nadrzędną i to, ile wierszy dostaje każdy jej wiersz. `{ customer: 0-5 }` daje każdemu klientowi od zera do pięciu płatności, a każda płatność niesie klucz swojego klienta w kolumnie nazwanej od tabeli nadrzędnej. `primary_key` wskazuje kolumnę albo kolumny, które identyfikują wiersz, i to ich wartość niesie wiersz podrzędny.

## Reguły między kolumnami

`consistency` przechowuje reguły takie jak `first_name matches: [gender]`, która dopasowuje imiona do kolumny płci. Miasto, jego region i kod pocztowy zgadzają się bez reguły, a `validate` o tym mówi.

## Psucie z przepisu

`policy` mówi, co popsuć, gdy przepis jest uruchamiany: jakie rodzaje uszkodzeń w `layers`, jaki udział wierszy w `rows`, ile wartości w jednym wierszu w `per_row` i których kolumn nie ruszać w `exclude_fields`. [Co psuje](/docs/damage/) wymienia każdy rodzaj.
