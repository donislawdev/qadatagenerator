---
translatedFrom: 4b68646c87475e88344833491f91e458c3e6f6f5
title: Nazwiska, które formularz powinien przyjąć
seoTitle: "Nazwiska, które formularz musi przyjąć: O'Brien, José, Null"
description: "Prawdziwe imiona i nazwiska, które odrzuca niestaranna walidacja: apostrofy, znaki diakrytyczne, inne pisma i nazwisko Null. Wstaw je celowo do testów."
lead: "Niektóre prawdziwe imiona i nazwiska nie przechodzą sprawdzeń, które wyglądają rozsądnie. Wstaw je celowo do danych testowych, a manifest powie przy każdym, że powinno zostać przyjęte."
weight: 3
sources:
  - "Personal names around the world, W3C Internationalization, czytane 16 listopada 2026."
  - "Patrick McKenzie, Falsehoods Programmers Believe About Names, 2010, czytane 16 listopada 2026."
  - "Chris Baraniuk, These unlucky people have names that break computers, BBC Future, 25 marca 2016, czytane 16 listopada 2026."
  - "SQL Injection Prevention Cheat Sheet, OWASP, czytane 16 listopada 2026."
---

## Cztery rodzaje nazwisk, które są odrzucane

**Apostrof: `O'Brien`, `D'Arcy`.** Sprawdzenie, które dopuszcza same litery, je odrzuca. Gorzej, kod, który składa zapytanie SQL, wklejając nazwisko w jego tekst, wywraca się na apostrofie, bo w SQL apostrof kończy tekst w cudzysłowie. Z tego samego błędu korzysta SQL injection i lekarstwo jest to samo: przekazuj nazwisko jako parametr zapytania, nigdy jako część jego tekstu.

**Litera ze znakiem diakrytycznym: José, Łukasz, Wójcik, Dvořák.** Wzorzec taki jak `[A-Za-z]+` je odrzuca, a takie litery mają nawet angielskie imiona i nazwiska. Trzymaj je w kodowaniu Unicode na całej drodze, od formularza po bazę danych, jak radzi W3C, bo inaczej znaki mogą zginąć po drodze.

**Pismo chińskie, japońskie albo koreańskie: 佐藤, 민준, 李明.** Te w danych mają po dwa znaki, więc reguła, że nazwisko ma co najmniej trzy, je odrzuca, tak samo jak każde sprawdzenie ograniczone do alfabetu łacińskiego.

**Nazwisko Null.** To prawdziwe nazwisko. Kod, który zapisuje brak wartości jako słowo null albo czyta to słowo jako brak, nie odróżni tej osoby od pustego pola. W 2016 roku BBC opisało, jak strony linii lotniczych raz po raz odpowiadały Jennifer Null, że zostawiła puste pole nazwiska.

## Dane testowe z takimi nazwiskami

Przepis niżej tworzy sto rejestracji i w mniej więcej co piątej wstawia jedno z takich imion albo nazwisk. Każde pochodzi z krótkiej listy prawdziwych, więc w raporcie stoi imię, które ktoś nosi, a nie literówka.

{{< example "recipe" >}}

{{< example "downloads" >}}

{{< example "run" >}}

{{< example "damage" >}}

Każde jest oznaczone jako do przyjęcia. Odrzucenie któregokolwiek to osoba, którą aplikacja odprawiła, a `qadg verify` zgłasza je i kończy się kodem 4. Nazwisko dłuższe niż kolumna nigdy nie jest wstawiane: kolumna mówi, co mieści, a aplikacja słusznie odrzuca nazwisko, które się w niej nie mieści.

## Co powinien robić formularz

- Przyjmuj litery każdego pisma, z apostrofami i znakami diakrytycznymi. Sprawdzaj długość, nie alfabet.
- Nie wymagaj więcej niż jednego znaku. Zdarzają się imiona z jednej litery.
- Nie wymagaj nazwiska. Wiele osób ma tylko imię.
- Trzymaj imiona i nazwiska w kodowaniu Unicode, od formularza po bazę danych.
- Przekazuj je do bazy jako parametry zapytania.
- Traktuj słowo Null jak każde inne nazwisko. Brak wartości to tylko puste pole.

Patrick McKenzie wypisał czterdzieści takich fałszywych założeń w tekście Falsehoods Programmers Believe About Names, a artykuł W3C Personal names around the world opisuje, jak imiona i nazwiska różnią się między kulturami.
