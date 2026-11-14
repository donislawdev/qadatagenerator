---
translatedFrom: 90c9af03e65ddfeb27e167a389f9d070c9c6aec5
title: Sklep w trzech tabelach
seoTitle: "Powiązane dane testowe: klienci i zamówienia w JSON"
description: "Przepis na sklep w trzech powiązanych tabelach, klienci, zamówienia i pozycje zamówień, w jednym pliku JSON, z kluczami, listami wartości i pustymi polami."
lead: "Pięćdziesięciu klientów, ich zamówienia i pozycje każdego zamówienia. O liczbie wierszy decydują relacje, a każde dziecko niesie klucz rodzica."
weight: 5
---

`belongs_to: { customer: 0-5 }` daje każdemu klientowi do pięciu zamówień, niektórym żadnego, a `belongs_to: { order: 1-4 }` daje każdemu zamówieniu od jednej do czterech pozycji. W JSON-ie każde zamówienie stoi wewnątrz swojego klienta, a każda pozycja wewnątrz swojego zamówienia, i mimo to każde nazywa rodzica w osobnym polu.

Statusy, plany i ilości pochodzą z list w przepisie, więc to wartości, które twoja aplikacja zna. Sumy i ceny są liczbami dziesiętnymi, a mniej więcej siedem kuponów na dziesięć jest pustych, żeby pokazać, jak twój kod traktuje brak wartości. Zmień `format: json` na `csv`, a dostaniesz osobny plik na każdą tabelę.
