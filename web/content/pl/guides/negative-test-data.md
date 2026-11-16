---
translatedFrom: 8f18b9cfcb52d9fdf7fab1b36fabc80aab842010
title: Dane negatywne
seoTitle: "Negatywne dane testowe z oczekiwanym wynikiem dla każdej"
description: "Złe dane to połowa testu. Wygeneruj wartości, które aplikacja powinna odrzucić, poprawne, które powinna przyjąć, i oczekiwany wynik każdej, w jednym przebiegu."
lead: "Test negatywny potrzebuje dwóch rzeczy: wartości, którą aplikacja powinna odrzucić, i sposobu, żeby sprawdzić, że to zrobiła. Jeden przebieg qadg daje obie, a do tego poprawne wartości, które niestaranna walidacja odrzuca."
weight: 1
---

## Odpowiedź jest tak samo ważna jak wartość

Plik złych wartości łatwo zrobić. Trudniejszy jest test, który czyta wynik: przy każdym wierszu musi wiedzieć, czy aplikacja powinna go odrzucić. Lista odpowiedzi pisana ręcznie rozjeżdża się z danymi przy ich pierwszej zmianie.

QA Data Generator zapisuje odpowiedzi sam. Każda popsuta wartość dostaje wpis w `expectation-manifest.json`: gdzie jest, wartość przed i po oraz co aplikacja powinna z nią zrobić. Twój test bierze odpowiedź z manifestu, zamiast trzymać własną.

## Trzy rodzaje odpowiedzi

- **Odrzuć.** Wartość łamie regułę, której powinna pilnować każda aplikacja tego rodzaju: nazwisko dłuższe niż kolumna, numer karty, którego cyfra kontrolna już się nie zgadza.
- **Przyjmij.** Wartość jest poprawna, a niestaranna walidacja ją odrzuca: O'Brien, Łukasz, 佐藤, nazwisko Null. Test negatywny, który ich nie wysyła, nie widzi ludzi, których aplikacja odprawia. Więcej w przewodniku [Nazwiska, które formularz powinien przyjąć](/guides/names-your-form-should-accept/).
- **`~`, twoja decyzja.** Właściwa odpowiedź zależy od reguł twojej aplikacji. Puste imię albo SSN z obszaru, którego nikt nie przydziela, zwykle powinny zostać odrzucone, ale tylko ty wiesz, czy twoja aplikacja to sprawdza. `qadg verify` zgłasza niezgodność przy takiej odpowiedzi i nie kończy się na niej błędem.

## Przepis, który prosi o wszystkie trzy

Przepis niżej opisuje tabelę klientów: imię i nazwisko wymagane i z najdłuższą długością, SSN i numer karty. Jego `policy` prosi o uszkodzenia z trzech grup, Field format (`L2`), Value meaning (`L3`) i Valid but unusual values (`REV`), w mniej więcej dwóch wierszach na pięć, po jednej wartości w każdym, więc każdy odrzucony wiersz ma jedną przyczynę.

{{< example "recipe" >}}

{{< example "downloads" >}}

## Co zmienił

{{< example "run" >}}

{{< example "damage" >}}

Części zamówionych uszkodzeń ta wersja jeszcze nie wykonuje. Manifest wymienia każde z nich z powodem, więc przebieg, który zmienił mniej, niż prosi przepis, mówi o tym.

## Włącz to do testu

1. Wygeneruj dane z przepisu.
2. Wyślij każdy wiersz do aplikacji tak, jak zrobiłby to użytkownik: przez formularz, API albo import.
3. Zapisz w małym pliku JSON, co aplikacja zrobiła z każdym wierszem: przyjęła czy odrzuciła.
4. Uruchom `qadg verify` na manifeście i tym pliku. Wypisze każdy wiersz, z którym aplikacja zrobiła coś innego, i zakończy się kodem 4, na którym pipeline może paść.

[Sprawdź, co zrobiła aplikacja](/docs/verify/) pokazuje zapis i raport. Zmień `layers`, żeby wybrać, co atakować, `rows`, żeby wybrać, ile, a `seed`, żeby dostać inne wartości tych samych rodzajów. [Co psuje](/docs/damage/) wymienia każdy rodzaj uszkodzenia, który wykonuje ta wersja.
