---
translatedFrom: 5ed15c9c80ffb6f4cfb949ee66e4764459123ee0
title: REGON
seoTitle: "Generator REGON do testów i cyfra kontrolna REGON"
description: "Jak liczy się cyfra kontrolna 9-cyfrowego numeru REGON, dlaczego jego cyfry nic nie znaczą i jak wygenerować poprawne i zepsute numery REGON do testów."
lead: "REGON to polski numer statystyczny firmy albo instytucji. Program robi postać dziewięciocyfrową, której ostatnia cyfra to cyfra kontrolna."
sources:
  - "Rozporządzenie Rady Ministrów z 30 listopada 2015, Dz.U. 2015 poz. 2009, § 12."
---

## Czym jest REGON

Dziewięć cyfr dla podmiotu: osiem cyfr liczby porządkowej i cyfra kontrolna. Jednostka lokalna podmiotu dostaje czternaście cyfr, z których dziewięć pierwszych to numer podmiotu. Rozporządzenie, które ustala numer, mówi, że jego cyfry nie mogą mieć charakteru znaczącego, więc nie kodują ani województwa, ani rodzaju działalności. Numer wykreślonego podmiotu nie trafia już do nikogo innego.

Program robi tylko postać dziewięciocyfrową. Reguły cyfry kontrolnej dla czternastu cyfr nie dało się sprawdzić na prawdziwych numerach, więc program jej nie ma, zamiast ją zgadywać.

## Cyfra kontrolna

Pomnóż pierwsze osiem cyfr przez 8, 9, 2, 3, 4, 5, 6, 7, zsumuj iloczyny i podziel przez 11. Reszta to cyfra kontrolna, a resztę 10 zapisuje się jako 0.

Żaden z czterech dokumentów urzędowych przeczytanych przy pisaniu programu nie podaje tej reguły. Te wagi odtwarzają każdy ze 174 prawdziwych numerów wziętych z dwóch rejestrów publicznych, w tym 15 z resztą 10.

Ponieważ 0 i 10 zapisuje się tak samo, zmiana jednej cyfry przechodzi niezauważona mniej więcej raz na 55 prób.

## Poprawne numery REGON do testów

{{< run "qadg quick regon --count 5 --seed example" >}}

Nie ma puli numerów REGON wydzielonej do testów, więc wygenerowany numer może pokryć się z numerem prawdziwej firmy: trzymaj go w systemach testowych.

## Co aplikacja ma zrobić z zepsutym

Odrzucić REGON, którego cyfra kontrolna się nie zgadza. Gdy zamiana dwóch sąsiednich cyfr zostawia cyfrę kontrolną poprawną, program nie liczy jej jako uszkodzenia i mówi o tym w manifeście.
