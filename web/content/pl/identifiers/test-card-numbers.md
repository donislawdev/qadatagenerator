---
translatedFrom: 2e358bbab703954a15641255cf165885e36bf485
title: Numery kart testowych
seoTitle: "Numery kart testowych, które przechodzą walidację"
description: "Numery kart do testów formularza płatności: osiem numerów testowych publikowanych przez operatorów płatności, z poprawną cyfrą Luhna, i zepsute do odrzucenia."
lead: "Osiem numerów kart, które operatorzy płatności publikują do testów, każdy z poprawną cyfrą kontrolną Luhna, i zepsute, które formularz płatności ma odrzucić."
sources:
  - "Numery kart testowych w dokumentacji Stripe, Braintree, Adyen i Checkout.com, czytane 2 listopada 2026."
---

## Skąd są te numery

Kolumna `card` i `qadg quick card` dają zawsze jeden z ośmiu numerów: dwa Visa, jeden Mastercard, dwa American Express, jeden Discover, jeden JCB i jeden UnionPay. Każdy z nich co najmniej dwóch z czterech operatorów (Stripe, Braintree, Adyen, Checkout.com) podaje w dokumentacji jako numer testowy. Program nigdy nie wymyśla numeru, który tylko przechodzi cyfrę kontrolną, bo taki numer może być prawdziwą kartą.

Numer testowy działa w systemie testowym swojego operatora. Żaden z czterech nie mówi, że żaden bank nie wydał tego samego numeru prawdziwej karcie, więc traktuj je jak wartości testowe i nie wpuszczaj ich na produkcję.

## Cyfra kontrolna

Ostatnia cyfra to cyfra kontrolna Luhna. Zacznij od cyfry tuż na lewo od niej i podwajaj co drugą cyfrę, idąc w lewo. Gdy podwojona cyfra daje 10 albo więcej, dodaj do siebie jej dwie cyfry. Zsumuj wszystko razem z cyfrą kontrolną. Numer jest poprawny, gdy suma kończy się zerem.

Dla `4111111111111111` osiem podwojonych cyfr daje 8 i siedem dwójek, czyli 22. Pozostałe osiem cyfr to same jedynki, czyli 8, a 22 + 8 to 30.

## Poprawne numery kart do testów

{{< run "qadg quick card --count 5 --seed example" >}}

Za tym poleceniem stoi tylko osiem numerów, więc dłuższa lista je powtarza. `--count 1000` daje tysiąc wierszy wylosowanych z tych samych ośmiu.

## Co formularz ma zrobić z zepsutym

Odrzucić go, zanim trafi do operatora płatności. Na tych ośmiu numerach zmiana jednej cyfry zawsze psuje sumę Luhna, tak samo jak zamiana miejscami dwóch różnych sąsiednich cyfr, więc formularz może wyłapać oba błędy.
