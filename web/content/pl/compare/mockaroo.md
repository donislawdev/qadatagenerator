---
translatedFrom: 73e1670f1afb696a2d637c96bb684cdd8c35fbc6
title: Mockaroo i QA Data Generator
linkTitle: Mockaroo
seoTitle: "Mockaroo czy QA Data Generator: alternatywa offline"
description: "Mockaroo i QA Data Generator porównane: strona WWW czy program na twoim komputerze, rodzaje danych, formaty, cena i popsute dane z oczekiwanym wynikiem."
lead: "Mockaroo tworzy dane testowe w przeglądarce, bez programowania. QA Data Generator tworzy je na twoim komputerze, celowo je psuje i mówi, co twoja aplikacja powinna zrobić z każdą popsutą wartością. Ta strona mówi, gdzie pasuje każde z nich."
sources:
  - "Mockaroo, strona główna, mockaroo.com: do 1000 wierszy w CSV, JSON, SQL i Excelu, bez programowania, makiety API, zapisane schematy przez API, obraz Dockera, odczyt 16 listopada 2026."
  - "Mockaroo, Pricing, mockaroo.com/pricing: plany Free, Silver, Gold i Enterprise, odczyt 16 listopada 2026."
  - "Mockaroo, dokumentacja API, mockaroo.com/docs: percentBlank i lista typów, odczyt 16 listopada 2026."
  - "Mockaroo, lista typów w edytorze schematu: opisy Naughty String i Blank oraz przykłady SSN, odczyt 16 listopada 2026."
---

## Czym jest Mockaroo?

Stroną WWW do generowania danych testowych, pod adresem mockaroo.com. Wymieniasz pola, każdemu wybierasz typ spośród ponad 200 i pobierasz wiersze jako CSV, JSON, SQL albo Excel, bez programowania. Mockaroo tworzy też makiety API, a zapisany schemat można pobierać ze skryptu przez jego API.

Plan bezpłatny tworzy do 1000 wierszy w pliku i odpowiada na 200 zapytań API dziennie. Plany płatne zaczynają się od 60 dolarów rocznie. Plany Free, Silver i Gold działają na mockaroo.com, a plan Enterprise jako obraz Dockera w twojej własnej chmurze albo serwerowni. Te fakty pochodzą z jego strony, czytanej 16 listopada 2026.

## Co QA Data Generator robi inaczej?

- **Działa na twoim komputerze.** Dane powstają tam, gdzie uruchamiasz `qadg`, bez połączenia sieciowego i bez konta.
- **Celowo psuje dane i mówi, co powinno się stać.** Zna {{< count "damage" >}}, od cyfry kontrolnej przesuniętej o jeden po nazwisko zapisane w innej postaci Unicode. Manifest obok danych wymienia każdą popsutą wartość i oczekiwany wynik, a `qadg verify` porównuje z nim to, co zrobiła twoja aplikacja. Mockaroo potrafi zostawić pustą część wartości pola i ma typ Naughty String, czyli napisy, które z dużym prawdopodobieństwem sprawiają kłopoty.
- **Polskie identyfikatory z cyframi kontrolnymi.** PESEL, NIP, REGON i polski IBAN. Na liście typów w dokumentacji API Mockaroo nie ma PESEL, NIP ani REGON.
- **SSN, które nie należą do nikogo.** Każdy SSN zaczyna się od 666, obszaru, którego nigdy się nie przydziela. Przykłady na liście typów Mockaroo, jak 678-59-9455, są z obszarów, które są przydzielane. [SSN](/identifiers/ssn/) mówi, które obszary są przydzielane.
- **Ten sam seed daje te same dane.** Przepis i seed odtwarzają te same pliki na każdej maszynie, więc możesz trzymać przepis zamiast danych.
- **Psuje plik, który już masz.** `qadg corrupt` czyta CSV, ustala, co zawiera każda kolumna, i zapisuje popsutą kopię z manifestem.

## Kiedy Mockaroo wystarczy?

Gdy potrzebujesz szybko poprawnych danych wielu rodzajów, bez instalowania czegokolwiek, albo potrzebujesz ich w Excelu. Mockaroo ma dużo więcej typów danych niż QA Data Generator i tworzy makiety API, czego QA Data Generator nie robi.

## Jak wypadają obok siebie?

| | Mockaroo | QA Data Generator |
|---|---|---|
| Czym jest | stroną WWW z API | programem: wiersz poleceń `qadg` i okno na pulpicie |
| Gdzie powstają dane | na mockaroo.com albo w twojej chmurze z planem Enterprise | na twoim komputerze, bez połączenia sieciowego |
| Cena | bezpłatnie do 1000 wierszy w pliku, plany płatne od 60 dolarów rocznie | bezpłatnie, GPL-3.0 |
| Rodzaje danych | ponad 200 typów | {{< count "generators" >}} |
| PESEL, NIP, REGON | nie ma ich na liście typów w dokumentacji API | tak |
| Wartości do testu walidacji | pusta część wartości każdego pola, typ Naughty String | {{< count "damage" >}} i oczekiwany wynik dla każdej popsutej wartości |
| Formaty | CSV, JSON, SQL, Excel | CSV, JSON i SQL oraz manifest |
| Makiety API | tak | nie |

## Czy mogę używać obu?

Tak. Wygeneruj poprawne dane w Mockaroo, pobierz je jako CSV i przepuść przez `qadg corrupt`: ustali, co zawiera każda kolumna, popsuje kopię i zapisze manifest, który mówi, co twoja aplikacja powinna zrobić z każdą popsutą wartością. [Test importu CSV](/guides/test-a-csv-import/) pokazuje cały przebieg.
