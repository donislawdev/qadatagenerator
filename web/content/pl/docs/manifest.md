---
translatedFrom: 9ab6d6101259bff35f1ad8d5d80ea18b8f9a70d5
title: Manifest
seoTitle: "Manifest: odpowiedź dla każdej popsutej wartości"
description: "Plik zapisywany obok popsutych danych: gdzie jest każde uszkodzenie, wartość przed i po, co ma zrobić aplikacja, oraz jego postacie Gherkin, CSV i Markdown."
lead: "Każdy przebieg, który psuje dane, zapisuje obok nich `expectation-manifest.json`. Twój test czyta z tego pliku właściwą odpowiedź, więc nie trzeba jej pisać ręcznie."
---

## Co mówi wpis

Każda popsuta wartość dostaje wpis: gdzie jest, według tabeli, wiersza i kolumny, jakie to było uszkodzenie, wartość przed i po, co twoja aplikacja powinna zrobić i jak pewna jest ta odpowiedź. Uszkodzenie, które nie pasowało do twoich danych, też jest na liście, z powodem, więc cichy przebieg nigdy nie wygląda jak czysty.

Kilka wpisów z przebiegu z [Pierwszych kroków](/docs/getting-started/):

{{< defects >}}

`~` oznacza odpowiedź zależną od reguł twojej aplikacji. `qadg verify` zgłasza rozbieżność przy takiej odpowiedzi, ale nie kończy się z jej powodu błędem.

## Inne postacie tych samych wpisów

Postacią kanoniczną jest `expectation-manifest.json`. `--manifest gherkin`, `csv` albo `markdown` zapisuje obok niego te same wpisy w drugiej postaci:

- **Gherkin.** Tabela `Examples` z jednym wierszem na każde zastosowane uszkodzenie, do uruchomienia w scenariuszu.
- **CSV.** Każde zamówione uszkodzenie, z kolumną, która mówi, czy zostało zastosowane, a jeśli nie, to dlaczego.
- **Markdown.** Zastosowane uszkodzenia jako tabela, do przeglądu albo do pull requesta.

## Ten sam seed, ten sam manifest

Ten sam przepis i ten sam seed dają te same wiersze i te same uszkodzenia, bajt w bajt, na Windows, macOS i Linuksie, łącznie z tym, które wiersze zostały popsute i jak. Dwa przebiegi różnią się tylko czasem zapisu podanym w manifeście.

## Format

Manifest jest opisany w otwartej specyfikacji, po angielsku, więc inne narzędzia mogą go czytać i zapisywać: [expectation manifest](https://github.com/donislawdev/qadatagenerator/blob/main/spec/expectation-manifest.md).
