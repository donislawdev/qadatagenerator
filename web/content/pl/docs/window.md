---
translatedFrom: ed7efb02f9a2a1833d0101b6a3833a9e62f2bc7d
title: Okno
seoTitle: "Okno programu: cztery ekrany"
description: "Quick value, Recipe, Damage a file i Check results: co robi każdy ekran okna i które polecenie robi to samo."
lead: "`qadg-gui` zamyka ten sam silnik w oknie. Każdy z czterech ekranów robi to samo co jedno z poleceń i pokazuje to polecenie gotowe do skopiowania. Okno mówi po angielsku, więc nazwy ekranów i zrzuty są angielskie."
---

## Quick value

Wybierz, co wygenerować, i dostań dziesięć wartości do skopiowania. Ekran robi to samo co `qadg quick`.

{{< window name="quick-website" alt="Ekran Quick value z dziesięcioma numerami VIN wylosowanymi z seeda readme" >}}
Dziesięć numerów VIN z seeda `readme`. Pierwsze trzy to te, które wypisuje przykład z [Pierwszych kroków](/docs/getting-started/), bo okno i `qadg quick` losują te same wartości z tego samego seeda.
{{< /window >}}

## Recipe

Napisz albo wklej przepis. Okno sprawdza go w trakcie pisania, pokazuje, co opisuje, i uruchamia go, kiedy zechcesz. Ekran robi to samo co `qadg validate` i `qadg generate`.

Każda linia przepisu ma numer, liczony tak samo, jak liczy linie `qadg validate` w swoich komunikatach. Linia z problemem ma czerwony znak przy numerze, a kliknięcie problemu po prawej przenosi do tej linii. Enter zaczyna następną linię z takim samym wcięciem jak linia wyżej, a po linii zakończonej dwukropkiem o jeden poziom głębiej.

W trakcie pisania lista przy kursorze pokazuje, co można w tym miejscu wpisać: klucze pasujące do tego miejsca przepisu, a po kluczu takim jak `generator` albo `type` wartości, które przyjmuje, każdą z opisem. Pisanie zawęża listę. Strzałki wybierają, Enter albo Tab wpisuje wybraną pozycję, a Esc zamyka listę. Ctrl+Spacja otwiera ją w dowolnej chwili.

{{< window name="recipe-website" alt="Ekran Recipe sprawdzający przepis shop.yaml, z plikami, które zapisałby przebieg, i pierwszymi wierszami klientów" >}}
Przepis z [Pierwszych kroków](/docs/getting-started/), wklejony do okna. Okno sprawdza go w trakcie pisania, wypisuje pliki, które zapisałby przebieg, mówi, jaką część zamówionych uszkodzeń ta wersja wykonuje, i pokazuje pierwsze wiersze, zanim cokolwiek zostanie zapisane.
{{< /window >}}

## Damage a file

Otwórz plik CSV, zobacz, co zawiera każda kolumna i dlaczego, wybierz, co popsuć, zobacz, co by się stało, a potem uruchom. Ekran robi to samo co `qadg corrupt`.

{{< window name="take-website" alt="Ekran Damage a file z plikiem customers.csv odczytanym kolumna po kolumnie i czterema wybranymi rodzajami uszkodzeń" >}}
`customers.csv` z przykładu na stronie [Popsuj plik, który już masz](/docs/damage-a-file/), odczytany kolumna po kolumnie, z czterema rodzajami uszkodzeń, które zamawia tamto polecenie. Panel po prawej mówi, co zrobiłby przebieg, zanim cokolwiek zostanie zapisane.
{{< /window >}}

## Check results

Otwórz manifest, przejdź przez każde zamówione uszkodzenie i porównaj je z tym, co zrobiła twoja aplikacja. Ekran robi to samo co `qadg verify`.

{{< window name="browser-website" alt="Ekran Check results porównujący manifest z tym, co zrobiła aplikacja, z rozpisanym wierszem 135" >}}
Manifest tamtego przebiegu obok tego, co aplikacja zrobiła z każdym wierszem. Wiersz 135 to numer karty z dwiema zamienionymi cyframi, który aplikacja przyjęła, choć powinna go odrzucić. Wpisy oznaczone `not built yet` to rodzaje uszkodzeń, które ta wersja opisuje, ale jeszcze ich nie wykonuje.
{{< /window >}}

## About

Wersja programu i jego licencja, tak jak wypisują je `qadg --version` i `qadg --license`. Przycisk Save the third-party licenses zapisuje do pliku licencje cudzego kodu, fontów i ikon, które są w oknie, czyli tę część, którą `qadg --license` wypisuje dla wiersza poleceń.
