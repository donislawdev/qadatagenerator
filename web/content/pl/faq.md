---
translatedFrom: 5d3b932d36c6d47acf3ad37b3c3257fafbff1cde
title: Pytania
seoTitle: "QA Data Generator FAQ: dane testowe, seed, licencja"
description: "Odpowiedzi o QA Data Generator: co robi, czy można go używać w pracy za darmo, czy potrzebuje internetu i dlaczego na innej maszynie daje te same dane."
lead: "Pytania, które ludzie zadają najpierw, z odpowiedzią w pierwszym zdaniu."
layout: faq
faq:
  - question: Czym jest QA Data Generator?
    answer: "Generatorem danych testowych dla testerów i programistów, z wierszem poleceń i oknem na pulpicie. Tworzy całe zestawy danych z krótkiego przepisu w YAML, celowo je psuje i zapisuje manifest, który mówi, co twoja aplikacja powinna zrobić z każdą popsutą wartością."
  - question: Czym różni się od biblioteki do losowych danych testowych?
    answer: "Taka biblioteka daje ci wartości. QA Data Generator daje też wartości, które powinny zostać odrzucone, takie jak numer karty z cyfrą kontrolną przesuniętą o jeden, oraz poprawne wartości, które niestaranna walidacja odrzuca, takie jak O'Brien, a do każdej właściwą odpowiedź w pliku, który czyta twój test. [Co psuje](/docs/damage/) wymienia każdy rodzaj uszkodzenia."
  - question: Czy jest bezpłatny? Czy mogę go używać w pracy?
    answer: "Tak. Kod jest na licencji GPL-3.0. Dane, pliki i manifesty, które wygenerujesz, należą do ciebie: licencja ich nie obejmuje i możesz ich używać w produktach o zamkniętym kodzie i w komercyjnych zestawach testów."
  - question: Czy potrzebuje internetu?
    answer: "Nie. Nie otwiera żadnego połączenia sieciowego: bez konta, bez telemetrii, bez sprawdzania aktualizacji."
  - question: Czy na innej maszynie dostanę te same dane?
    answer: "Tak. Ten sam przepis i ten sam seed dają te same bajty na Windows, macOS i Linuksie, łącznie z tym, które wiersze zostały popsute i jak."
  - question: Czy może popsuć plik CSV, który już mam?
    answer: "Tak. `qadg corrupt` ustala, co zawiera każda kolumna, patrząc na wartości, psuje kopię i zapisuje obok niej manifest. Twój plik zostaje taki, jaki był. [Popsuj plik, który już masz](/docs/damage-a-file/) pokazuje przebieg."
  - question: Czy build może paść, gdy aplikacja odpowie źle?
    answer: "Tak. Zapisz, co twoja aplikacja zrobiła z każdym wierszem, a `qadg verify` porówna to z manifestem. Gdy się nie zgadzają, kończy się kodem wyjścia 4. [Użycie w CI](/docs/ci/) opisuje każdy kod wyjścia."
  - question: Dlaczego wygenerowane numery SSN zaczynają się od 666?
    answer: "Tego obszaru nigdy się nie przydziela, więc wygenerowany SSN nie może należeć do nikogo. Aplikacja, która sprawdza obszar, odrzuci je, także w poprawnych wierszach."
  - question: Czy numery kart są prawdziwe?
    answer: "Nie. Numery kart to wyłącznie opublikowane numery testowe, które przechodzą walidację i nie należą do żadnej prawdziwej karty."
  - question: W jakich formatach zapisuje wyniki?
    answer: "Dane w CSV i JSON. Manifest w JSON, a do tego może go zapisać jako tabelę Examples w Gherkinie, jako CSV albo jako Markdown."
  - question: Potrzebujesz plików testowych, a nie rekordów?
    answer: "[Testing Files Generator](https://github.com/donislawdev/TestingFilesGenerator) tworzy prawdziwe pliki w dziesiątkach formatów, o dowolnym dokładnym rozmiarze, z własnym manifestem."
---
