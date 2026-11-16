---
translatedFrom: e1843a416e62377f8e6c91ec7a8eb1eda841b090
title: Numer ABA routing
seoTitle: "Numery ABA routing do testów i ich cyfra kontrolna"
description: "Jak sprawdza się dziewiąta cyfra amerykańskiego numeru ABA routing wagami 3, 7 i 1, co znaczą dwie pierwsze cyfry i jak wygenerować poprawne i zepsute numery."
lead: "Numer ABA routing wskazuje amerykański bank na czeku albo w przelewie. Ma dziewięć cyfr, a dziewiąta to cyfra kontrolna."
sources:
  - "ABA Routing Number Policy & Procedures, wersja 9/20, od LexisNexis Risk Solutions jako oficjalnego rejestratora ABA, sekcja IV."
  - "12 CFR 229 (Regulation CC), załącznik A."
---

## Czym jest numer ABA routing

Cztery cyfry symbolu Rezerwy Federalnej (Federal Reserve routing symbol), cztery cyfry identyfikatora instytucji i cyfra kontrolna. Dwie pierwsze cyfry mówią, jaki to numer: od 01 do 12 to banki z dwunastu okręgów Rezerwy Federalnej, od 21 do 32 instytucje oszczędnościowe, od 61 do 72 transakcje elektroniczne, a 80 czeki podróżne. Większość pozostałych serii jest zarezerwowana na przyszłość.

## Cyfra kontrolna

Pomnóż dziewięć cyfr przez 3, 7, 1, 3, 7, 1, 3, 7, 1 i zsumuj iloczyny. Numer jest poprawny, gdy suma dzieli się przez 10.

Polityka rejestratora opisuje części numeru, a po rachunek odsyła do płatnej publikacji. Każdy z 67 numerów banków Rezerwy Federalnej i Federal Home Loan Banks wydrukowanych w Regulation CC przechodzi to sprawdzenie.

## Poprawne numery ABA routing do testów

{{< run "qadg quick aba --count 5 --seed example" >}}

Każdy numer zaczyna się od 01 do 12, czyli od serii banków, bo żadna seria nie jest wolna do testów: rejestrator mówi, że o użyciu serii zarezerwowanych na przyszłość zdecyduje jego zarząd. Wygenerowany numer może pokryć się z numerem prawdziwego banku, więc trzymaj go w systemach testowych.

## Co aplikacja ma zrobić z zepsutym

Odrzucić numer, którego cyfra kontrolna się nie zgadza. Uszkodzenie może też przenieść numer do serii zarezerwowanej na przyszłość, z poprawną cyfrą kontrolną. Dziś nie ma takiego numeru żaden bank, więc aplikacja, która sprawdza numery w spisie Rezerwy Federalnej, powinna go odrzucić. Manifest oznacza tę odpowiedź jako zależną od aplikacji, bo wiele aplikacji takiego spisu nie ma.
