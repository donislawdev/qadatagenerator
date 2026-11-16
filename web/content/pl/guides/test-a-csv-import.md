---
translatedFrom: 07e563c968465894fc9aac9f529499fda2465a45
title: Test importu CSV
seoTitle: "Jak przetestować import CSV z popsutymi wierszami"
description: "Popsuj celowo kopię prawdziwego pliku CSV, zaimportuj ją i porównaj to, co zrobiła aplikacja, z oczekiwanym wynikiem dla każdej popsutej wartości."
lead: "Weź plik CSV, który twój import już przyjmuje, celowo popsuj jego kopię, zaimportuj ją i pozwól `qadg verify` wskazać wiersze, z którymi aplikacja zrobiła coś źle."
weight: 2
notYet: [L0, L1]
sources:
  - "Keeping leading zeros and large numbers, Microsoft Support, czytane 16 listopada 2026."
---

## Zacznij od pliku, który import przyjmuje

Użyj pliku takiego, jakie przysyłają twoi użytkownicy, który import dziś przyjmuje bez zastrzeżeń. Wtedy każda zmiana w popsutej kopii pochodzi od `qadg corrupt`, a manifest zna dla niej właściwą odpowiedź. Plik, który już ma problemy, miesza twoje błędy z zamówionymi.

`qadg corrupt` czyta plik i z wartości oraz nagłówka ustala, co jest w każdej kolumnie, na przykład numery kart albo nazwiska. `--dry-run` wypisuje, co ustalił i co by popsuł, i niczego nie zapisuje.

## Popsuj kopię

{{< readme marker="run" nth="4" kind="run" output="output" outputNth="2" >}}

Twój plik zostaje bez zmian. Popsuta kopia to `customers.corrupted.csv`, a `expectation-manifest.json` obok ma wpis dla każdej zmienionej wartości.

`--layers` mówi, co atakować, i jest wymagane. Przy imporcie zacznij od Field format i Value meaning (`L2,L3`), czyli wartości do odrzucenia, oraz Valid but unusual values (`REV`), czyli nazwisk, które powinny przejść. Dodaj Unicode characters (`UNI`), żeby zobaczyć, co import robi z emoji i niewidocznymi znakami. `--rows` ustala udział psutych wierszy, domyślnie 5%.

Ta wersja zmienia wartości w wierszach. Nie psuje jeszcze samego pliku: wiersza z brakującą kolumną, zabłąkanego cudzysłowu ani pliku uciętego w połowie. [Co psuje](/docs/damage/) wymienia każdy rodzaj, który wykonuje.

## Zaimportuj tak, jak robią to użytkownicy

Wyślij popsutą kopię tą samą drogą co prawdziwy plik: przez formularz wgrywania, API albo zadanie wsadowe. Nie otwieraj jej po drodze w arkuszu i nie zapisuj. Excel usuwa zera wiodące z wartości takiej jak `02116` i zachowuje tylko 15 cyfr znaczących dłuższej liczby, więc ostatnia cyfra 16-cyfrowego numeru karty staje się zerem, a plik przestaje zgadzać się z manifestem.

## Zapisz, co stało się z każdym wierszem

Dla każdego wiersza zapisz w małym pliku JSON, czy aplikacja go przyjęła, czy odrzuciła. Wiersze liczy się od 1, od pierwszego rekordu po nagłówku.

{{< readme marker="response response.json customers.corrupted.csv" caption="response.json" >}}

- Zapisz wpis dla każdego wiersza, także czystego. Wiersz bez wpisu jest zgłaszany jako taki, na który aplikacja nigdy nie zareagowała, i tak właśnie widać zgubiony wiersz albo import przerwany w połowie.
- Dodaj `code` z własnym kodem błędu aplikacji, a `verify` powtórzy go w raporcie.
- Gdy możesz odczytać to, co zapisano, dodaj to jako `stored`. Aplikacja może przyjąć wiersz i zachować coś innego, na przykład nazwisko ucięte do szerokości kolumny, a tylko `stored` pozwala `verify` odróżnić to od wartości, która weszła taka, jaką ją wysłano.

## Porównaj

{{< readme marker="run exit=4" kind="run" output="output" outputNth="3" exit="4" >}}

Każde ostrzeżenie to wiersz, z którym aplikacja zrobiła coś innego, niż mówi manifest: wartość, która powinna zostać odrzucona, a weszła, poprawna wartość, którą odprawiono, albo wiersz, za który nikt nie odpowiedział. Przebieg kończy się kodem 4, na którym pipeline może paść. Odpowiedzi oznaczone `~` zależą od twoich reguł, a niezgodność przy nich jest zgłaszana bez błędu.

`verify` odpowiada za wartości, które zostały popsute. Czystego wiersza, który aplikacja odrzuciła, nie ma w manifeście, więc takie wiersze sprawdź we własnym raporcie.

[Użycie w CI](/docs/ci/) układa te kroki w pipeline.
