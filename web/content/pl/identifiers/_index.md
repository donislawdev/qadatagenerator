---
translatedFrom: 376f76a2e21f762f4119abba6223ad7572332db0
title: Identyfikatory
seoTitle: "Poprawne identyfikatory do testów i ich cyfry kontrolne"
description: "Numery kart, ISBN, EAN, IMEI, VIN, SSN, ABA, PESEL, NIP, REGON i IBAN: jak działa cyfra kontrolna każdego z nich i skąd wziąć poprawne i zepsute do testów."
lead: "Jedenaście rodzajów numerów, które sprawdza formularz albo import. Każda strona mówi, jak numer jest zbudowany, jak działa jego cyfra kontrolna i skąd wziąć poprawne i zepsute do testów."
---

Każdy numer z programu ma poprawną cyfrę kontrolną, a część rodzajów powstaje w puli wydzielonej tak, żeby nie mógł pokryć się z prawdziwym. Tabela mówi, które.

| Numer | Cyfra kontrolna | Czy wygenerowany może pokryć się z prawdziwym? |
|---|---|---|
| [Numery kart testowych](/identifiers/test-card-numbers/) | Luhn | To numery testowe publikowane przez operatorów płatności |
| [VIN](/identifiers/vin/) | dziewiąty znak, reszta z dzielenia przez 11 | Tak |
| [SSN](/identifiers/ssn/) | brak | Nie, obszaru 666 nigdy się nie przydziela |
| [PESEL](/identifiers/pesel/) | ostatnia cyfra, wagi 1, 3, 7, 9 | Tak |
| [ISBN](/identifiers/isbn/) | ostatnia cyfra, wagi 1 i 3 | Dziś nie, ta pula nie ma grupy rejestracyjnej |
| [EAN-13](/identifiers/ean/) | ostatnia cyfra, wagi 3 i 1 | Nie, prefiks 952 jest wydzielony na przykłady |
| [IMEI](/identifiers/imei/) | Luhn | Tylko z urządzeniem testowym, nigdy z telefonem w sprzedaży |
| [Numer ABA routing](/identifiers/routing-number/) | wagi 3, 7, 1 | Tak |
| [NIP](/identifiers/nip/) | ostatnia cyfra, reszta z dzielenia przez 11 | Tak |
| [REGON](/identifiers/regon/) | ostatnia cyfra, reszta z dzielenia przez 11 | Tak |
| [IBAN (Polska)](/identifiers/polish-iban/) | dwie cyfry, reszta z dzielenia przez 97 | Tak |

Gdzie wygenerowany numer może pokryć się z prawdziwym, trzymaj dane w systemach testowych. Więcej mówi strona [Każda wartość jest zmyślona](/security/).

Każda strona pokazuje numery, które wypisała ta wersja programu, i przepis, który część z nich psuje, z tym, co aplikacja ma zrobić z każdym zepsutym. Wszystko to pochodzi z uruchomienia programu, więc strona nie pokaże czegoś, czego program już nie robi.
