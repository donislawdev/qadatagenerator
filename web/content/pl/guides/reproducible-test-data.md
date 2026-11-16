---
translatedFrom: f1147ff81213a40b38ee2c4350d0a743f57cb321
title: Powtarzalne dane testowe
seoTitle: "Powtarzalne dane testowe: ten sam seed, te same bajty"
description: "Trzymaj w repozytorium mały przepis zamiast dużych plików z danymi. Ten sam przepis i seed dają te same wiersze i te same uszkodzenia na każdej maszynie."
lead: "Przepis i seed wystarczą, żeby odtworzyć te same dane, bajt w bajt, na Windows, macOS i Linuksie. Trzymaj w repozytorium przepis, a dane generuj tam, gdzie są potrzebne."
weight: 5
---

## Co zostaje takie samo

Ten sam przepis, uruchomiony tą samą wersją `qadg` z tym samym seedem, zapisuje te same pliki, bajt w bajt, na Windows, macOS i Linuksie. Dotyczy to też uszkodzeń: popsute są te same wiersze w ten sam sposób, a manifest mówi to samo. Dwa przebiegi różnią się tylko czasem zapisanym w manifeście.

Seed to dowolny tekst do 128 znaków. Wpisz go w przepisie jako `seed:` albo podaj `--seed` w wierszu poleceń, który ma pierwszeństwo przed przepisem. Przepis bez seeda jest czytany z seedem `0`, więc też się powtarza.

{{< run "qadg quick vin --count 3 --seed reproducible" >}}

To samo polecenie daje te trzy numery VIN, gdziekolwiek je uruchomisz. Inny seed daje trzy inne.

## Trzymaj w repozytorium przepis, nie dane

Plik z dziesięcioma tysiącami wierszy trudno przejrzeć i nikt go nie aktualizuje. Przepis ma dwadzieścia linii, mówi, czym są dane, i odtwarza je w pipeline. Gdy test padnie na jakiejś wartości, seed i numer wiersza z raportu prowadzą z powrotem do niej.

Każdy przebieg zapisuje obok danych `run-note.json`. Podaje wersję `qadg`, seed i locale przebiegu, czyli to, czego trzeba, żeby odtworzyć te same dane.

## Zmieniaj jedną rzecz naraz

Każda wartość jest losowana ze swojego miejsca: tabeli, wiersza, kolumny i seeda. Dlatego dodanie kolumny zostawia wartości pozostałych kolumn bez zmian, a diff nowych danych pokazuje nową kolumnę. Większe `count` dokłada wiersze na końcu i zostawia pierwsze, razem z ich uszkodzeniami, takie, jakie były.

## Co zmienia dane

- Inny seed.
- Inna wersja `qadg`. Nowa wersja może inaczej losować niektóre wartości, więc przypnij wersję, na której biegną testy, i zajrzyj do `run-note.json`, gdy zmieniły się dane, które miały zostać takie same.
- Zmiana w przepisie, która dotyka wartości: inne locale, inny generator, reguła między kolumnami, inna `policy`.

## Jedno polecenie, którego przebiegi się różnią

`qadg quick` bez `--seed` losuje nowy seed przy każdym uruchomieniu, bo kto prosi dwa razy o PESEL, zwykle chce dwóch różnych. Wylosowany seed podaje na kanale logu, więc wartość, która się spodobała, można dostać ponownie z `--seed`.

[Użycie w CI](/docs/ci/) ma kody wyjścia i kanały potrzebne w pipeline.
