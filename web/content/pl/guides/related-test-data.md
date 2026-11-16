---
translatedFrom: aec2933010d377b37cb4ad463f339d1344b9d2cc
title: Powiązane tabele
seoTitle: "Dane testowe dla powiązanych tabel, z pasującymi kluczami"
description: "Wygeneruj klientów i ich zamówienia w jednym przebiegu: każdy wiersz dziecka wskazuje istniejącego rodzica, a każdy rodzic ma tyle dzieci, ile zamówisz."
lead: "Klienci, ich zamówienia i pozycje każdego zamówienia, z jednego przepisu i jednego przebiegu. Każde zamówienie niesie klucz istniejącego klienta, a każda pozycja klucz zamówienia."
weight: 4
notYet: [L5]
---

## Dlaczego tabele wypełniane osobno nie wystarczą

Wypełnij trzy tabele osobno, a klucze się nie spotkają: zamówienie wskazuje klienta, którego nie ma, a import albo klucz obcy odrzuca je, zanim test się zacznie. Przepis opisuje tabele razem z relacjami między nimi, a jeden przebieg wypełnia je tak, że klucze się zgadzają.

## Opisz relacje w przepisie

{{< example "recipe" >}}

{{< example "downloads" >}}

- `primary_key` wskazuje kolumnę, która identyfikuje wiersz. To ją niesie wiersz dziecka.
- `belongs_to: { customer: 0-3 }` daje każdemu klientowi od zera do trzech zamówień, więc niektórzy nie mają żadnego, a `belongs_to: { order: 1-4 }` daje każdemu zamówieniu od jednej do czterech pozycji.
- Tabela dziecka nie potrzebuje `count`. Liczbę jej wierszy wyznacza relacja, a inny seed rozkłada je inaczej.
- Każde dziecko dostaje kolumnę nazwaną jak rodzic, z kluczem rodzica: `customer` w zamówieniach, `order` w pozycjach.

## Co zapisuje

{{< example "run" >}}

{{< example "writes" >}}

Każda wartość w kolumnie `customer` zamówień to `customer_id` jednego z klientów, a każda wartość w kolumnie `order` pozycji to `order_id` jednego z zamówień.

## Liczba, która pasuje do relacji

Tabela dziecka może podać własne `count`. Relacja decyduje wtedy tylko o tym, jak wiersze rozkładają się między rodziców, a liczba, której relacja nie może osiągnąć, jest odrzucana, zanim cokolwiek zostanie zapisane. Przebieg kończy się kodem 5, a komunikat podaje rachunek: ile wierszy ma rodzic, jaki jest zakres i między czym musi leżeć liczba.

## W JSON

Z `format: json` każde zamówienie stoi wewnątrz swojego klienta, a każda pozycja wewnątrz swojego zamówienia, w jednym pliku, i każde dziecko nadal wskazuje rodzica we własnym polu. [Sklep w trzech tabelach](/examples/orders/) to podobny sklep w JSON.

## Czego ta wersja jeszcze nie robi

Każdy klucz w danych się zgadza. Ta wersja nie psuje jeszcze relacji: dziecka, które wskazuje nieistniejącego rodzica, ani powtórzonego klucza. [Co psuje](/docs/damage/) wymienia każdy rodzaj, który wykonuje.
