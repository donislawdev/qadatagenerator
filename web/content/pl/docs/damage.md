---
translatedFrom: 791211fd04eb96f56fe37c80a2da470a717b7ddb
title: Co psuje
seoTitle: "Co psuje i co powinna zrobić twoja aplikacja"
description: "Każdy rodzaj uszkodzenia, który ta wersja robi z wartością, pogrupowany tak jak w oknie, z tym, co twoja aplikacja powinna z nim zrobić."
lead: "Uszkodzenia są pogrupowane. Okno pokazuje nazwę grupy, a `--layers` w wierszu poleceń i `layers:` w przepisie przyjmują jej kod."
tocFacts: damage
---

Każdy rodzaj uszkodzenia niżej to taki, który ta wersja wykonuje. Przykład to ten, który katalog trzyma obok niego i porównuje z tym, co program naprawdę robi. Plakietka przy każdym z nich mówi, co aplikacja powinna zrobić z popsutą wartością, a manifest zapisuje tę samą odpowiedź przy każdej zmienionej wartości. Nazwy grup i opisy są po angielsku, tymi samymi słowami, którymi mówi o nich program.

`~` oznacza odpowiedź zależną od reguł twojej aplikacji. Czy emoji pasuje do nazwy miasta, decydujesz ty. `qadg verify` zgłasza rozbieżność przy takiej odpowiedzi, ale nie kończy się z jej powodu błędem.

{{< damage >}}

## Wybór, co popsuć

W wierszu poleceń `qadg corrupt --layers L2,L3` przyjmuje kody, a `--rows` ustala udział wierszy do popsucia, domyślnie 5%. W przepisie `policy:` przyjmuje te same kody w `layers:`. W oknie zaznaczasz grupy po nazwie.

Uszkodzenie, które nie pasuje do twoich danych, też trafia do manifestu, z powodem, więc cichy przebieg nigdy nie wygląda jak czysty.
