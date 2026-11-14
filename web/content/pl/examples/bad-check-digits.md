---
translatedFrom: cc14f9a5e14c6503ae237d59b4f68bd5076a8297
title: Numery do odrzucenia
seoTitle: "Błędna cyfra kontrolna: numery testowe do odrzucenia"
description: "Przepis, który miesza zepsute numery kart, IBAN, ISBN, EAN, IMEI, VIN i ABA z poprawnymi, oraz manifest, który wskazuje każdą zepsutą wartość."
lead: "Tabela [numerów testowych](/examples/test-numbers/) z zamówionym zepsuciem w mniej więcej co piątym wierszu. Twoja aplikacja ma odrzucić każdą zepsutą wartość, a manifest mówi, które to są."
weight: 6
---

Uszkodzenia są takie, jakie robi człowiek przy przepisywaniu numeru albo system przy jego kopiowaniu: cyfra kontrolna przesunięta o jeden, zamienione dwie cyfry, numer ABA z puli, której nikt nie wydaje. Każde zachowuje długość i znaki poprawnego numeru, więc wyłapie je tylko prawdziwe sprawdzenie.

Zaimportuj plik, a potem daj poleceniu [`qadg verify`](/docs/verify/) manifest i zapis tego, co twoja aplikacja zrobiła z każdym wierszem. Wypisze każdy wiersz, w którym aplikacja zrobiła coś innego, niż mówi manifest.
