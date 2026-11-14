---
translatedFrom: 31aed54946bccc1d2852f308323e6d4a5089c9f5
title: Popsuj plik, który już masz
seoTitle: "Popsuj plik CSV, który już masz"
description: "Uruchom qadg corrupt na pliku CSV. Program ustala, co zawiera każda kolumna, mówi dlaczego, psuje kopię i zapisuje, co aplikacja ma zrobić z każdą wartością."
lead: "Do testów na złych danych nie potrzebujesz przepisu. `qadg corrupt` bierze plik CSV, który już masz, psuje jego kopię i zapisuje obok niej manifest."
---

## Uruchom

{{< readme marker="run" nth="4" kind="run" output="output" outputNth="2" >}}

Twój plik zostaje taki, jaki był. Popsuta kopia to `customers.corrupted.csv`, zapisana obok manifestu.

## Co program ustala o twoich kolumnach

Nic w zwykłym pliku CSV nie mówi, co zawiera kolumna, więc `qadg corrupt` patrzy na wartości i na nagłówek. Kolumnę, której każda wartość przechodzi walidację numeru karty i nie przechodzi walidacji żadnego innego systemu z cyfrą kontrolną, uznaje za numery kart. Dla każdej kolumny program mówi, co ustalił i dlaczego, jak w liniach wyżej, i wypisuje to, czego nie umiał ustalić, zamiast zgadywać.

Uszkodzenie, które wymaga rodzaju kolumny, jakiego twój plik nie ma, trafia do manifestu jako niezastosowane, z powodem.

## Wybierz, co popsuć

`--layers` jest wymagane, więc to zawsze ty wybierasz, co atakować. [Co psuje](/docs/damage/) wymienia każdą grupę i każdy rodzaj uszkodzenia w niej. `--rows` ustala udział wierszy do popsucia, domyślnie 5%.

`--dry-run` pokazuje, co program ustalił i co by popsuł, i niczego nie zapisuje.

## Wszystkie flagi

{{< command-flags name="corrupt" >}}

## W oknie

Ekran Damage a file robi to samo: otwórz plik CSV, zobacz, co zawiera każda kolumna i dlaczego, zaznacz, co popsuć, zobacz, co by się stało, a potem uruchom. Pokazuje też polecenie, które robi to samo, gotowe do skopiowania.
