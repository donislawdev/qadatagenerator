---
translatedFrom: 07d223b59753796062a5a6c93126cb943f9e2a67
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
      - from: "`type: integer` or `type: decimal` | field | a random number: a whole number from 0 to 999999, or one with two decimal places from 0.00 to 999.99"
        to: ["`type: integer` albo `type: decimal`", "pole", "losowa liczba: całkowita od 0 do 999999 albo z dwoma miejscami po przecinku od 0.00 do 999.99"]
      - from: "`min_value` and `max_value` | field | the smallest and the largest number a `type: integer` or `type: decimal` field draws, both included, such as `min_value: 1` and `max_value: 10`. Either can be left out, and either can be negative"
        to: ["`min_value` i `max_value`", "pole", "najmniejsza i największa liczba, jaką losuje pole `type: integer` albo `type: decimal`, obie włącznie, na przykład `min_value: 1` i `max_value: 10`. Każdą z nich można pominąć i każda może być ujemna"]
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
      - from: "`output` | top | which files a run writes and where, with the keys below"
        to: ["`output`", "główny poziom", "jakie pliki zapisuje przebieg i gdzie, kluczami poniżej"]
      - from: "`format` | output | `csv`, the default, or `json`"
        to: ["`format`", "output", "`csv`, domyślnie, albo `json`"]
      - from: "`dir` | output | the folder the files go to, relative to where you run `qadg`. `--out` takes its place, and the Recipe screen writes to the folder chosen at its top"
        to: ["`dir`", "output", "folder, do którego trafiają pliki, względem miejsca, w którym uruchamiasz `qadg`. `--out` go zastępuje, a ekran Recipe zapisuje do folderu wybranego na jego górze"]
      - from: "`delimiter` | output | for `csv`, the one character between values: a comma when left out, or any other in quotes, such as a semicolon or a tab written `\"\\t\"`. A double quote and a line break are refused"
        to: ["`delimiter`", "output", "dla `csv` jeden znak między wartościami: przecinek, gdy klucza nie ma, albo dowolny inny w cudzysłowie, na przykład średnik albo tabulator zapisany jako `\"\\t\"`. Cudzysłów i koniec linii są odrzucane"]
      - from: "`file_per_entity` | output | for `json` only: `true` writes a file per table, such as `customer.json`, and `false`, the default, writes one `dataset.json` with each row's children inside it. `csv` always writes a file per table"
        to: ["`file_per_entity`", "output", "tylko dla `json`: `true` zapisuje plik na tabelę, na przykład `customer.json`, a `false`, domyślnie, zapisuje jeden `dataset.json` z wierszami podrzędnymi wewnątrz swoich rodziców. `csv` zawsze zapisuje plik na tabelę"]
      - from: "`encoding` | output | `utf-8`, the only encoding written today"
        to: ["`encoding`", "output", "`utf-8`, jedyne kodowanie, w jakim dziś zapisujemy"]
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

## Liczby

`type: integer` wypełnia kolumnę losowymi liczbami całkowitymi od 0 do 999999, a `type: decimal` losowymi liczbami z dwoma miejscami po przecinku od 0.00 do 999.99. `min_value` i `max_value` ustawiają najmniejszą i największą liczbę, obie włącznie:

```yaml
fields:
  quantity: { type: integer, min_value: 1, max_value: 10 }
  price:    { type: decimal, min_value: 0.5, max_value: 99.99 }
  change:   { type: integer, min_value: -50, max_value: 50 }
```

Gdy pominiesz jedną z nich, ten koniec zostaje tam, gdzie był, więc samo `max_value: 10` daje liczby od 0 do 10. Granica liczby dziesiętnej ma najwyżej dwa miejsca po przecinku, a granica liczby całkowitej żadnego. Gdy obok stoi `max_length`, oba końce muszą się w nim zmieścić. `validate` mówi, która granica jest zła i dlaczego.

## Gdzie trafiają pliki

`output` mówi, jakie pliki zapisuje przebieg i gdzie. Każdy klucz można pominąć:

```yaml
output:
  format: csv
  dir: ./data
  delimiter: "\t"
```

- `format` to `csv`, domyślnie, albo `json`.
- `dir` to folder, do którego trafiają pliki, względem miejsca, w którym uruchamiasz `qadg`, a gdy go nie ma, sam ten folder. `qadg generate --out` go zastępuje, a ekran Recipe zapisuje do folderu wybranego na jego górze.
- `delimiter` to znak między wartościami w pliku `csv`, przecinek, gdy klucza nie ma. Każdy inny pojedynczy znak działa w cudzysłowie, na przykład średnik dla arkusza, który go używa, albo `"\t"` dla tabulatora. Cudzysłów i koniec linii są odrzucane, bo już coś w pliku znaczą. `json` nie ma separatora i `validate` ostrzega, gdy go podasz.
- `file_per_entity` ma znaczenie tylko dla `json`. `true` zapisuje plik na tabelę, na przykład `customer.json` i `payment.json`, każdy jako listę wierszy, w której wiersz podrzędny wskazuje rodzica w polu. `false`, domyślnie, zapisuje jeden `dataset.json` z wierszami podrzędnymi wewnątrz swoich rodziców. `csv` zawsze zapisuje plik na tabelę, więc `validate` ostrzega, że `false` nic tam nie zmienia.
- `encoding` może być tylko `utf-8`, i tak samo zapisuje przebieg bez tego klucza.

## Powiązane tabele

`belongs_to` wskazuje tabelę nadrzędną i to, ile wierszy dostaje każdy jej wiersz. `{ customer: 0-5 }` daje każdemu klientowi od zera do pięciu płatności, a każda płatność niesie klucz swojego klienta w kolumnie nazwanej od tabeli nadrzędnej. `primary_key` wskazuje kolumnę albo kolumny, które identyfikują wiersz, i to ich wartość niesie wiersz podrzędny.

## Reguły między kolumnami

`consistency` przechowuje reguły takie jak `first_name matches: [gender]`, która dopasowuje imiona do kolumny płci. Miasto, jego region i kod pocztowy zgadzają się bez reguły, a `validate` o tym mówi.

## Psucie z przepisu

`policy` mówi, co popsuć, gdy przepis jest uruchamiany: jakie rodzaje uszkodzeń w `layers`, jaki udział wierszy w `rows`, ile wartości w jednym wierszu w `per_row` i których kolumn nie ruszać w `exclude_fields`. [Co psuje](/docs/damage/) wymienia każdy rodzaj.
