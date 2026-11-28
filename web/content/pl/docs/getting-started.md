---
translatedFrom: 38e6e35d315301aa15e45c84faa86260fe17073d
title: Pierwsze kroki
description: "Pobierz QA Data Generator, wypisz pierwszą poprawną wartość, potem wygeneruj cały zestaw danych z przepisu i celowo go popsuj, w kilka minut."
lead: "Od zera do zestawu danych, jego popsutej kopii i odpowiedzi dla każdej popsutej wartości. Każde polecenie i każdy wynik na tej stronie pochodzą z prawdziwego przebiegu."
---

## Pobierz go

[Instalacja](/download/) podaje archiwum dla każdego systemu, instalator dla Windows i sposób budowania ze źródeł. Przykłady niżej wpisują `qadg`, więc dodaj folder, do którego go rozpakujesz, do zmiennej `PATH` albo zainstaluj go na Windows.

## Jedna wartość

{{< readme marker="run" kind="run" output="output" >}}

Bez `--seed` każdy przebieg losuje nowe wartości. Z nim za każdym razem wracają te same. `qadg help generators` wypisuje wszystko, co ta wersja potrafi wygenerować, a [Generatory](/docs/generators/) mają tę samą listę.

## Cały zestaw danych

Zapisz to jako `shop.yaml`:

{{< readme marker="file shop.yaml" caption="shop.yaml" >}}

Sprawdź go, a potem uruchom:

{{< readme marker="run" nth="2" kind="run" output="output 1" >}}

`data/customer.csv` zaczyna się tak:

{{< readme marker="head data/customer.csv" caption="data/customer.csv" >}}

a `data/payment.csv` tak, każda płatność z kluczem swojego klienta:

{{< readme marker="head data/payment.csv" caption="data/payment.csv" >}}

- `belongs_to: { customer: 0-5 }` daje każdemu klientowi od zera do pięciu płatności.
- `matches` dopasowuje imiona do kolumny płci. Miasto, stan i kod ZIP zgadzają się bez reguły, a `validate` o tym mówi.
- `required` i `max_length` opisują twoje kolumny. Na ich podstawie psucie opisane niżej wybiera, co zepsuć.
- `seed` sprawia, że przebieg da się powtórzyć. Zmień go, a dostaniesz inne dane o tym samym kształcie.

[Przepisy](/docs/recipes/) wymieniają każdy klucz.

## Popsuj go celowo

Dopisz trzy linie do przepisu:

{{< readme marker="append shop.yaml" caption="dopisane do shop.yaml" >}}

i uruchom go jeszcze raz, prosząc też o manifest w postaci tabeli Markdown:

{{< readme marker="run" nth="3" kind="run" >}}

Dane wychodzą tak jak wcześniej, z uszkodzeniami w 5% wierszy, a obok nich leży `expectation-manifest.json`. Każdy wpis mówi, gdzie jest popsuta wartość, jaka była przed zmianą i po niej oraz co twoja aplikacja powinna z nią zrobić. [Manifest](/docs/manifest/) pokazuje kilka wpisów i objaśnia każde pole.

## Dalej

- [Popsuj plik, który już masz](/docs/damage-a-file/), gdy masz plik CSV do testów.
- [Sprawdź, co zrobiła aplikacja](/docs/verify/), żeby porównać odpowiedzi aplikacji z manifestem.
- [Użycie w CI](/docs/ci/), żeby build padał przy złej odpowiedzi.
