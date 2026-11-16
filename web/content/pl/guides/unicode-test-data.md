---
translatedFrom: 5148fcffbbe6b7c1566fe1c873ae5b8c023c7d3e
title: Dane testowe z Unicode
seoTitle: "Dane testowe Unicode: emoji, niewidoczne spacje, tureckie ı"
description: "Wartości, które wyglądają tak samo, a nie są: emoji z kilku znaków, spacja zerowej szerokości, tureckie ı bez kropki i ten sam tekst w innej postaci Unicode."
lead: "Tekst, który na ekranie wygląda dobrze, może być innym tekstem. Te wartości łapią kod, który liczy, porównuje albo zmienia wielkość liter bez myślenia o Unicode."
weight: 6
sources:
  - "Unicode Technical Standard #51, Unicode Emoji, wersja 18.0, czytane 16 listopada 2026."
  - "Unicode Standard Annex #15, Unicode Normalization Forms, rewizja 58, czytane 16 listopada 2026."
  - "SpecialCasing.txt z Unicode Character Database, wersja 18.0.0, czytane 16 listopada 2026."
  - "The utf8mb3 Character Set, MySQL 8.4 Reference Manual, czytane 16 listopada 2026."
  - "Mikael Goldmann, Creative usernames and Spotify account hijacking, Spotify Engineering, 18 czerwca 2013, czytane 16 listopada 2026."
---

## Sześć niewinnie wyglądających wartości

**Emoji, jak w `Kłodzko 🙂`.** To jeden ze znaków, które w UTF-8 zajmują cztery bajty. Zestaw znaków `utf8` w MySQL, dziś nazywany `utf8mb3`, mieści najwyżej trzy bajty na znak i takiego znaku nie zapisze. `utf8mb4` zapisze.

**Emoji rodziny, jak w `Malbork 👩‍👩‍👧‍👦`.** Jeden obrazek na ekranie, złożony z siedmiu punktów kodowych: czterech osób połączonych trzema łącznikami zerowej szerokości. W UTF-8 to 25 bajtów, w UTF-16 11 jednostek. Tam, gdzie system nie ma obrazka rodziny, pokazuje cztery osoby obok siebie. Limit 40 liczony w jednej z tych jednostek i limit 40 liczony w tym, co widzi czytelnik, to różne limity, dlatego `max_length` w przepisie przyjmuje `counted_in: bytes` albo `graphemes` obok znaków.

**Spacja zerowej szerokości, jak w `Kętrzyn` ze spacją po `Kęt`.** Nie zajmuje miejsca na ekranie, a mimo to wartość różni się od tej bez niej. W loginie albo adresie e-mail tworzy dwa konta, które wyglądają tak samo. Lista niżej wypisuje ją jako kod, żeby było widać, gdzie stoi.

**Tureckie ı bez kropki, jak w `Wıeluń`.** Turecki ma i z kropką oraz ı bez kropki, każde z własną wielką literą: i oraz İ, ı oraz I. Zamień `Wıeluń` na wielkie litery i z powrotem, a dostaniesz `wieluń`, bo wielką literą ı jest zwykłe I. W drugą stronę, poza językiem tureckim wielkie İ staje się w małych literach dwoma punktami kodowymi: i oraz łączącą kropką. Kod, który zmienia wielkość liter, żeby porównać tekst, może więc dopasować wartości, których nie powinien, albo ominąć te, które powinien.

**Ten sam tekst w innej postaci, jak w `Giżycko`.** Litera ż może być jednym punktem kodowym albo literą z, po której stoi łącząca kropka. Obie wyglądają tak samo i Unicode uznaje je za ten sam tekst, ale porównanie bajtów mówi, że się różnią. Przed porównaniem sprowadź obie strony do tej samej postaci, zwykle NFC, jak opisuje Unicode Standard Annex #15. Lista niżej wypisuje łączącą kropkę jako kod.

**Wartość, która zmienia się przy drugiej normalizacji, jak `ᴬᴸᴱᴷsᴬᴺᴰᴿzᴬᴷ`.** Funkcja, która zamienia ten tekst na małe litery, a potem normalizuje go do NFKC, daje za pierwszym razem `ALEKsANDRzAK`, a za drugim `aleksandrzak`. Jeśli aplikacja sprowadza nazwy użytkowników do jednej postaci, żeby wykryć duplikaty, taka funkcja może wysłać reset hasła na cudze konto. Tak było w Spotify w 2013 roku, a naprawa polegała na odrzucaniu nazw, które zmieniają się przy drugim przejściu.

## Dane testowe ze wszystkimi sześcioma

Przepis niżej korzysta z polskich imion, nazwisk i miejscowości, bo zapis tego samego tekstu w innej postaci wymaga liter ze znakami diakrytycznymi, a polski ma ich dużo. Prosi o Unicode characters (`UNI`) w mniej więcej trzech wierszach na dziesięć.

{{< example "recipe" >}}

{{< example "downloads" >}}

{{< example "run" >}}

{{< example "damage" >}}

Większość odpowiedzi ma `~`, bo zależy od twoich reguł: emoji w nazwie miasta może być w porządku w jednym systemie i błędem w innym. `qadg verify` zgłasza niezgodność przy nich i nie kończy się na niej błędem. Ten sam tekst w innej postaci jest oznaczony do przyjęcia bez `~`, bo to ten sam tekst.

## Co sprawdzić w kodzie

- Licz długość tak, jak rozumie ją limit: w bajtach przy zapisie, w grafemach przy tym, co widzi człowiek.
- Sprowadzaj tekst do jednej postaci normalizacji, zwykle NFC, zanim go porównasz albo sprawdzisz jego unikalność.
- Zapisuj tekst w kodowaniu, które mieści cztery bajty na znak, w MySQL to `utf8mb4`.
- Zdecyduj, gdzie wolno stać niewidocznym znakom, i odrzucaj je albo usuwaj w loginach i adresach e-mail.
- Testuj zmianę wielkości liter z ustawieniami języka, z którymi działa aplikacja, bo od nich zależy wynik.
