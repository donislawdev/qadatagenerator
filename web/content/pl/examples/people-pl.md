---
translatedFrom: 1586dec27a0c196612a0fa1cb401fc85a922310b
title: Osoby z Polski
seoTitle: "PESEL do testów: sto osób z Polski w CSV"
description: "Przepis na sto osób z Polski w CSV: imię, nazwisko, płeć, data urodzenia, miasto i PESEL zgodny z datą urodzenia i płcią, z poprawną cyfrą kontrolną."
lead: "Sto osób z polskimi imionami i nazwiskami, każda z numerem PESEL, który niesie jej datę urodzenia i płeć i ma poprawną cyfrę kontrolną."
weight: 2
---

Imiona i nazwiska idą za płcią, więc kobieta nazywa się Kowalska, a nie Kowalski. Reguła `pesel matches: [birth_date, gender]` każe każdemu numerowi PESEL zgadzać się z dwiema kolumnami obok, a to właśnie sprawdza polski formularz. Poprawny PESEL może należeć do prawdziwej osoby, więc trzymaj ten plik w systemach testowych.

Miasto pochodzi z listy polskich miejscowości. Ta wersja nie ma rejestru, który mówi, w jakim województwie i pod jakim kodem pocztowym leży miejscowość, więc przepis pomija oba, zamiast łączyć miasto z niewłaściwymi.
