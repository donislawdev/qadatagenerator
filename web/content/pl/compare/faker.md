---
translatedFrom: 6a8be219bd49a4f460e7efa95386ba530e4226a3
title: Faker i QA Data Generator
linkTitle: Faker
seoTitle: "Faker czy QA Data Generator: dane testowe"
description: "Faker i QA Data Generator porównane: wartości z kodu w Pythonie czy zestawy danych z przepisu, seed, polskie identyfikatory i wartości do testu walidacji."
lead: "Faker daje wartości twojemu kodowi. QA Data Generator zapisuje całe zestawy danych, celowo je psuje i mówi, co twoja aplikacja powinna zrobić z każdą popsutą wartością. Ta strona mówi, gdzie pasuje każde z nich."
sources:
  - "Dokumentacja Fakera dla wersji 40.54.0, faker.readthedocs.io: pierwsza strona (seed, wartości unikalne, wiersz poleceń, Factory Boy, pytest) i lista locale, odczyt 16 listopada 2026."
  - "Dokumentacja Fakera dla wersji 40.54.0, providery locale pl_PL i en_US: pesel, nip, regon, identity_card_number, iban, ssn, invalid_ssn i credit_card_number, odczyt 16 listopada 2026."
  - "Faker w PyPI: wersja 40.54.0 z 10 listopada 2026, licencja MIT, odczyt 16 listopada 2026."
  - "Faker dla JavaScriptu, fakerjs.dev, Usage, sekcja Reproducible results, odczyt 16 listopada 2026."
  - "Pomiar z 16 listopada 2026: Faker 40.54.0 na Pythonie 3.13, locale en_US, seed 4321, pięć wierszy z first_name i last_name, potem pięć wierszy z first_name, email i last_name."
---

## Czym jest Faker?

Pakietem Pythona, który generuje fałszywe dane, na licencji MIT. W kodzie tworzysz generator i wołasz go raz dla każdej wartości: imienia, adresu, numeru PESEL. Jego lista locale ma 110 pozycji, w tym polskie. Ma wtyczkę do pytest, współpracuje z Factory Boy przy wypełnianiu modeli, a jego polecenie `faker` wypisuje wartości w terminalu.

Fakty na tej stronie pochodzą z wersji 40.54.0, wydanej 10 listopada 2026, i z jej dokumentacji czytanej 16 listopada 2026. Faker dla JavaScriptu, `@faker-js/faker`, to osobny projekt. Ta strona mówi o Fakerze dla Pythona.

## Co QA Data Generator robi inaczej?

- **Zapisuje zestawy danych z przepisu, a nie z kodu.** Plik YAML wymienia tabele, kolumny, liczbę wierszy i klucze między tabelami, a `qadg generate` zapisuje je jako CSV albo JSON. Okno na pulpicie czyta ten sam przepis.
- **Celowo psuje dane i mówi, co powinno się stać.** Zna {{< count "damage" >}}, od cyfry kontrolnej przesuniętej o jeden po nazwisko zapisane w innej postaci Unicode. Manifest obok danych wymienia każdą popsutą wartość i oczekiwany wynik, a `qadg verify` porównuje z nim to, co zrobiła twoja aplikacja.
- **Nowa kolumna zostawia pozostałe bez zmian.** Każda wartość jest losowana ze swojego miejsca: tabeli, wiersza, kolumny i seeda. W Fakerze jeden seed uruchamia jeden ciąg liczb losowych, a każde wywołanie bierze z niego następną. Tabela pięciu wierszy z Fakera 40.54.0 i seedem 4321 zmieniła wszystkie nazwiska i wszystkie imiona od drugiego wiersza, gdy między imię a nazwisko weszła kolumna z adresem e-mail.
- **SSN, które nie należą do nikogo, i testowe numery kart.** Każdy SSN zaczyna się od 666, obszaru, którego nigdy się nie przydziela, a numery kart to wyłącznie numery testowe publikowane przez operatorów płatności. `ssn()` Fakera daje numery z obszarów, które są przydzielane, jak 042-34-8377 z jego dokumentacji, a według dokumentacji `credit_card_number()` generuje poprawny numer karty kredytowej. [SSN](/identifiers/ssn/) mówi, które obszary są przydzielane.
- **Psuje plik, który już masz.** `qadg corrupt` czyta CSV, ustala, co zawiera każda kolumna, i zapisuje popsutą kopię z manifestem.

## Kiedy Faker wystarczy?

Gdy test buduje dane w kodzie, obiekt po obiekcie, i potrzebuje tylko poprawnych wartości. Faker żyje w teście, zna dużo więcej locale i rodzajów danych i ma więcej polskich identyfikatorów niż QA Data Generator: poza PESEL, NIP, REGON i IBAN także numer dowodu osobistego. QA Data Generator jest programem, a nie biblioteką, więc twój test czyta pliki, które on zapisuje.

## Jak wypadają obok siebie?

| | Faker | QA Data Generator |
|---|---|---|
| Czym jest | pakietem Pythona, który woła twój kod | programem: wiersz poleceń `qadg` i okno na pulpicie |
| Jak opisujesz dane | kodem w Pythonie, jedno wywołanie na wartość | przepisem YAML z tabelami, wierszami i kluczami |
| Locale | 110 pozycji na liście | {{< count "locales" >}} |
| Polskie identyfikatory | PESEL, NIP, REGON, numer dowodu osobistego, IBAN | PESEL, NIP, REGON, IBAN |
| SSN | z obszarów, które są przydzielane | zawsze z obszaru 666, nigdy nieprzydzielanego |
| Numery kart | według dokumentacji poprawne numery kart | tylko opublikowane numery testowe |
| Wartości do testu walidacji | na przykład `invalid_ssn()` | {{< count "damage" >}} i oczekiwany wynik dla każdej popsutej wartości |
| Kolumna dodana w środku | zmienia wartości losowane po niej | zostawia pozostałe kolumny bez zmian |
| Ten sam seed w nowszej wersji | wyniki nie są gwarantowane nawet między wersjami poprawkowymi | może wylosować część wartości inaczej |
| Co dostajesz | wartości w swoim kodzie, a `faker` je wypisuje | pliki CSV albo JSON i manifest |
| Licencja | MIT | GPL-3.0, a wygenerowane dane należą do ciebie |

## Czy mogę używać obu?

Tak, i QA Data Generator już to robi: jego listy imion i nazwisk oraz polskich miast i województw pochodzą z Fakera 40.36.0 na licencji MIT, a `qadg --license` je wymienia. Używaj Fakera tam, gdzie test buduje jeden obiekt w kodzie, a QA Data Generator tam, gdzie test potrzebuje całego zestawu danych, popsutych wartości i właściwej odpowiedzi dla każdej z nich.
