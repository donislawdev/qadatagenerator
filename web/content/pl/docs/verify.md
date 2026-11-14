---
translatedFrom: 4bdeca496405dac000aa29ff21e75d3f6d73cb93
title: Sprawdź, co zrobiła aplikacja
seoTitle: "Porównaj to, co zrobiła aplikacja, z manifestem"
description: "Zapisz, co twoja aplikacja zrobiła z każdym wierszem, a qadg verify wypisze każdy wiersz niezgodny z manifestem, z kodem wyjścia, na którym build może paść."
lead: "Manifest mówi, co twoja aplikacja powinna zrobić z każdą popsutą wartością. `qadg verify` porównuje to z tym, co naprawdę zrobiła."
---

## Zapisz, co zrobiła twoja aplikacja

Zaimportuj popsuty plik do aplikacji i zapisz, co zrobiła z każdym wierszem, jeden wpis na wiersz. Wystarczy kilka linii na końcu testu. Trzy wpisy takiego zapisu:

{{< readme marker="response response.json customers.corrupted.csv" caption="response.json" >}}

`verdict` to `accepted` albo `rejected`. `code` jest opcjonalny: to własny kod aplikacji mówiący, co się stało, który `verify` powtarza w raporcie. Gdy możesz odczytać to, co aplikacja zapisała, dodaj `stored`, a `verify` odróżni wartość, która przetrwała, od takiej, którą aplikacja po cichu zmieniła. Każde dodane pole poszerza to, co porównanie może powiedzieć, a raport mówi, na które pytania nie umiał odpowiedzieć.

## Porównaj

{{< readme marker="run exit=4" kind="run" output="output" outputNth="3" exit="4" >}}

Kończy się kodem wyjścia 4. Wiersz 135 to numer karty z dwiema zamienionymi cyframi, który przeszedł, a wiersz 432 to O'Brien, którego odrzucono. Pipeline może paść na tym kodzie. [Użycie w CI](/docs/ci/) opisuje każdy kod wyjścia.

Odpowiedź oznaczona w manifeście `~` zależy od reguł twojej aplikacji, więc `verify` zgłasza przy niej rozbieżność, ale nie kończy się z jej powodu błędem.

## W oknie

Ekran Check results otwiera manifest i zapis tego, co zrobiła twoja aplikacja, przechodzi przez każde zamówione uszkodzenie i pokazuje, gdzie się nie zgadzają.

## Format

Zapis tego, co zrobiła twoja aplikacja, jest opisany w otwartej specyfikacji, po angielsku: [application response record](https://github.com/donislawdev/qadatagenerator/blob/main/spec/application-response.md).
