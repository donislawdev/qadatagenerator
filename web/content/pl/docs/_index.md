---
translatedFrom: 6961e92b09f64b1d7be06098319109ed339a6ade
title: Dokumentacja
seoTitle: "Dokumentacja QA Data Generator: przepisy, uszkodzenia, CLI"
description: "Jak działa QA Data Generator: przepisy, generatory, uszkodzenia, manifest, sprawdzanie tego, co zrobiła aplikacja, okno i każde polecenie."
lead: "Wszystko, co robią wiersz poleceń i okno, i jak czytać to, co mówią."
cards:
  - /docs/getting-started
  - /docs/recipes
  - /docs/generators
  - /docs/damage
  - /docs/manifest
  - /docs/damage-a-file
  - /docs/verify
  - /docs/window
  - /docs/ci
  - /docs/cli
  - /docs/exit-codes
---

QA Data Generator tworzy dane testowe dwóch rodzajów. **Dobre dane** przechodzą walidację: to całe zestawy danych z przepisu, w których imiona, miejsca i identyfikatory zgadzają się ze sobą, a każda cyfra kontrolna jest poprawna. **Złe dane** to te same dane celowo popsute, obok manifestu, który mówi, co twoja aplikacja powinna zrobić z każdą zmienioną wartością.

To dwa programy na jednym silniku. `qadg` to wiersz poleceń, do terminala i do pipeline'u. `qadg-gui` to okno z tym samym silnikiem za czterema ekranami.

Program mówi po angielsku. Polecenia, to, co wypisują, opisy z pomocy programu i nazwy w oknie zostają więc na tych stronach po angielsku, tak jak zobaczysz je na ekranie.

Jeśli zaczynasz od zera, przejdź do [Pierwszych kroków](/docs/getting-started/). Jeśli masz już plik CSV do testów, przejdź do strony [Popsuj plik, który już masz](/docs/damage-a-file/).
