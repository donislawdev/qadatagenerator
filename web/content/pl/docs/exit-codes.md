---
translatedFrom: 51bd184c57bb3f4051853d774d0f7c507d40e5aa
title: Kody wyjścia
seoTitle: "Kody wyjścia qadg"
description: "Co znaczy każdy kod wyjścia qadg, od 0 dla udanego przebiegu do 4, gdy aplikacja nie zrobiła tego, co mówi manifest, tak jak wypisuje je qadg --help."
lead: "Każde zakończenie ma własny kod, więc pipeline odróżni awarię narzędzia od błędu twojej aplikacji."
---

{{< exit-codes >}}

Znaczenia w tabeli są po angielsku, tak jak wypisuje je `qadg --help`. `130` i `143` znaczą, że przebieg został przerwany, przez ciebie albo przez system.

Tylko `4` dotyczy twojej aplikacji: dane powstały, a aplikacja nie zrobiła z nimi tego, co mówi manifest. Każdy inny kod poza `0` dotyczy samego przebiegu, a komunikat na standardowym wyjściu błędów mówi, co poszło nie tak i co z tym zrobić.
