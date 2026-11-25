---
translatedFrom: a7cb576d5db9c07dd987db1690d926eedff3621a
title: QA Data Generator
seoTitle: "Generator danych testowych: dobre i złe dane z odpowiedzią"
description: "Realistyczne dane testowe z przepisu YAML, potem celowo popsute. Przy każdej popsutej wartości to, co ma z nią zrobić aplikacja. Za darmo, offline, CLI i GUI."
hero:
  eyebrow: Generator danych testowych dla QA
  title: Dobre dane. Złe dane.
  turn: I właściwa odpowiedź.
  lead: "**QA Data Generator** zamienia krótki przepis w YAML w całe zestawy danych, które przechodzą walidację, w CSV albo JSON. Potem celowo je psuje i zapisuje obok danych, co twoja aplikacja powinna zrobić z każdą popsutą wartością."
  facts:
    - Bezpłatny, z otwartym kodem, GPL-3.0
    - Offline, bez konta i bez telemetrii
    - Windows, macOS i Linux
  good: dobre dane
  should: twoja aplikacja powinna
  bad: złe i podchwytliwe dane
  file: expectation-manifest.json
  note: Z prawdziwych przebiegów qadg quick i qadg corrupt.
recipes:
  eyebrow: Przepisy
  title: Cały zestaw danych z jednego krótkiego pliku
  text: "Opisz swoje tabele, ile wierszy ma każda i jak są ze sobą powiązane. Klient może mieć od zera do pięciu płatności, każda płatność wskazuje swojego klienta, a jedno polecenie zapisuje je wszystkie."
  points:
    - "`belongs_to: { customer: 0-5 }` daje każdemu klientowi od zera do pięciu płatności, a każda z nich niesie klucz swojego klienta."
    - "`matches` dopasowuje imiona do kolumny płci. Miasto, stan i kod ZIP zgadzają się ze sobą bez dodatkowej reguły."
    - "`required` i `max_length` opisują twoje kolumny. Na ich podstawie psucie opisane niżej wybiera, co zepsuć."
    - "`seed` sprawia, że przebieg da się powtórzyć. Zmień go, a dostaniesz inne dane o tym samym kształcie."
  link:
    text: Wszystkie klucze przepisu
    page: /docs/recipes
damage:
  eyebrow: Psucie celowe
  title: Złe dane i odpowiedź, której potrzebuje twój test
  text: "Dopisz trzy linie do przepisu, a ten sam przebieg popsuje część wierszy. Obok danych zapisze `expectation-manifest.json` z wpisem dla każdej zmienionej wartości: gdzie jest, jaka była przed zmianą i po niej oraz co twoja aplikacja powinna z nią zrobić. Twój test czyta odpowiedź z tego pliku, więc nie trzeba jej pisać ręcznie."
  added: dopisane do shop.yaml
  note: "`~` oznacza odpowiedź zależną od reguł twojej aplikacji. `qadg verify` zgłasza rozbieżność przy takiej odpowiedzi, ale nie kończy się z jej powodu błędem."
  link:
    text: Jak działa manifest
    page: /docs/manifest
good:
  eyebrow: Dobre dane
  title: Wartości, które przechodzą walidację i zgadzają się ze sobą
  text: "Każdy identyfikator, który ma cyfrę kontrolną, ma ją poprawną, a numery kart to wyłącznie opublikowane numery testowe. Imiona, nazwiska, miasta i regiony pochodzą z list słów dla Stanów Zjednoczonych i Polski."
  cards:
    - title: Ludzie
      text: Imiona i nazwiska, które mogą iść za kolumną płci, oraz daty urodzenia.
      generators: [first_name, last_name, gender, birth_date]
    - title: Miejsca
      text: Miasto, jego stan i kod ZIP, które do siebie pasują.
      generators: [city, region, postal_code]
    - title: Konta
      text: Losowe UUID-y na identyfikatory użytkowników i rekordów oraz hasła w postaci jawnej, które przyjmują typowe reguły haseł.
      generators: [uuid, password]
    - title: Produkty i płatności
      text: Numery ISBN, kody kreskowe EAN, numery IMEI telefonów komórkowych, numery VIN pojazdów oraz numery kart wyłącznie z opublikowanych numerów testowych.
      generators: [isbn, ean, imei, vin, card]
    - title: Stany Zjednoczone
      text: Numery ubezpieczenia społecznego (SSN), które nie należą do nikogo, i numery rozliczeniowe banków (ABA).
      generators: [ssn, aba]
    - title: Polska
      text: PESEL zgodny z datą urodzenia i płcią, NIP, REGON i numery rachunków bankowych.
      generators: [pesel, nip, regon, iban]
  quick: jedna wartość w sekundę
  link:
    text: Wszystkie generatory
    page: /docs/generators
groups:
  eyebrow: Złe dane
  title: Pięć rodzajów uszkodzeń
  text: "Wybierz, które rodzaje zastosować, w oknie albo przez `--layers`. Nazwy grup są takie jak w oknie, po angielsku. Manifest zawiera odpowiedź dla każdej pojedynczej wartości, a plakietka pokazuje tę najczęstszą."
  cards:
    - code: L2
      text: Wydłuża wartość ponad to, na co pozwala kolumna, opróżnia wymagane pole, wstawia litery do liczby.
      answer: reject
    - code: L3
      text: Przesuwa cyfrę kontrolną o jeden, zamienia dwie cyfry miejscami, bierze numer z puli, której nigdy się nie przydziela.
      answer: reject
    - code: L4
      text: Sprawia, że PESEL nie zgadza się z datą urodzenia albo płcią zapisaną obok.
      answer: reject
    - code: UNI
      text: Dodaje emoji, niewidoczną spację zerowej szerokości, tureckie ı bez kropki albo zapisuje ten sam tekst w innej postaci Unicode.
      answer: "~ your rules"
    - code: REV
      text: "Wstawia prawdziwe imię albo nazwisko, które niestaranna walidacja odrzuca: O'Brien, José, 美咲, Null."
      answer: accept
  link:
    text: Wszystkie rodzaje uszkodzeń
    page: /docs/damage
yours:
  eyebrow: Twoje własne dane
  title: Popsuj plik, który już masz
  text:
    - "Uruchom `qadg corrupt` na pliku CSV. Program sam ustala, co zawiera każda kolumna, patrząc na wartości, mówi, dlaczego tak uznał, i wypisuje to, czego nie umiał ustalić, zamiast zgadywać."
    - "Twój plik zostaje taki, jaki był. Popsuta kopia i jej manifest zapisują się obok niego."
  links:
    - text: Popsuj plik, który już masz
      page: /docs/damage-a-file
pipeline:
  eyebrow: W pipelinie CI
  title: Sprawdź, co zrobiła twoja aplikacja
  text: "Zaimportuj popsuty plik, zapisz, co twoja aplikacja zrobiła z każdym wierszem, a `qadg verify` wypisze każdy wiersz, w którym aplikacja nie zgadza się z manifestem. Kończy się wtedy kodem wyjścia 4, więc pipeline może na nim paść."
  links:
    - text: Sprawdź, co zrobiła aplikacja
      page: /docs/verify
    - text: Użycie w CI
      page: /docs/ci
window:
  eyebrow: Okno na pulpicie
  title: Ten sam silnik w oknie
  text: "`qadg-gui` ma cztery ekrany do pracy i każdy robi to samo co jedno z poleceń. Okno mówi po angielsku, więc nazwy ekranów są tu angielskie."
  cards:
    - title: Quick value
      text: Wybierz, co wygenerować, i dostań dziesięć wartości do skopiowania.
      command: "`qadg quick`"
      shot: quick-website
      alt: Ekran Quick value z dziesięcioma numerami VIN wylosowanymi z seeda readme
    - title: Recipe
      text: Napisz albo wklej przepis. Okno sprawdza go w trakcie pisania, pokazuje, co opisuje, i uruchamia go, kiedy zechcesz.
      command: "`qadg validate` i `qadg generate`"
      shot: recipe-website
      alt: Ekran Recipe sprawdzający przepis shop.yaml, z plikami, które zapisałby przebieg, i pierwszymi wierszami klientów
    - title: Damage a file
      text: Otwórz plik CSV, zobacz, co zawiera każda kolumna i dlaczego, wybierz, co popsuć, zobacz, co by się stało, a potem uruchom.
      command: "`qadg corrupt`"
      shot: take-website
      alt: Ekran Damage a file z plikiem customers.csv odczytanym kolumna po kolumnie i czterema wybranymi rodzajami uszkodzeń
    - title: Check results
      text: Otwórz manifest, przejdź przez każde zamówione uszkodzenie i porównaj je z tym, co zrobiła twoja aplikacja.
      command: "`qadg verify`"
      shot: browser-website
      alt: Ekran Check results porównujący manifest z tym, co zrobiła aplikacja, z rozpisanym wierszem 135
  link:
    text: Okno
    page: /docs/window
promises:
  eyebrow: Ostrożny z założenia
  title: Bezpieczny obok prawdziwych danych
  items:
    - title: Offline
      text: Żadnego połączenia z siecią, konta, telemetrii ani sprawdzania aktualizacji.
    - title: Ten sam seed, te same dane
      text: Ten sam przepis i ten sam seed dają te same wiersze i te same uszkodzenia, bajt w bajt, na Windows, macOS i Linuksie.
    - title: Twoje pliki zostają nietknięte
      text: "Przebieg nie zapisuje niczego, dopóki się nie powiedzie. Istniejący plik go zatrzymuje, chyba że dodasz `--overwrite`, a pliku podanego `qadg corrupt` program nigdy nie nadpisuje."
    - title: Każda wartość jest zmyślona
      text: Nie użyto żadnego rejestru prawdziwych osób. Numery SSN zaczynają się od 666, czyli od obszaru, którego nigdy się nie przydziela.
    - title: Dane są twoje
      text: Kod jest na licencji GPL-3.0. Licencja nie obejmuje danych, plików ani manifestów, które wygenerujesz, więc możesz ich używać w komercyjnych zestawach testów.
    - title: Otwarte formaty
      text: Katalog uszkodzeń, manifest i zapis tego, co zrobiła twoja aplikacja, są opisane w otwartych specyfikacjach, więc inne narzędzia mogą je czytać i zapisywać.
  link:
    text: Każda wartość jest zmyślona
    page: /security
cta:
  title: Pierwszy zestaw danych w dwie minuty
  text: Bezpłatny i z otwartym kodem. Zbuduj go, napisz przepis i zobacz, co twoja aplikacja robi ze złymi danymi.
---
