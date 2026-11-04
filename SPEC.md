# Spezifikation: Fahrrad-Verleih auf Verleih-Framework (Aufgabe 1 c + d)

Grundlage für die Implementierung (Spec-Driven Development).
Ausgangspunkt ist der Ski-Verleih aus Teil a) und b); dessen Anforderungen **A-1 … A-19** und
**B-1 … B-8** gelten hier unverändert, nur mit „Artikel“ statt „Ski“.
Neu sind die Anforderungen **C-x** (Fahrradverleih) und **D-x** (Framework / Plugin-Architektur).

## 1. Ziel

Statt Ski werden Fahrräder verliehen (c). Dafür wird der Verleih so umgebaut, dass er nichts mehr
über die verliehenen Dinge weiß: Was verliehen wird, kommt als **Plugin** dazu (d).

```
            RentalItem  (Interface im Framework)
                 ▲
        ┌────────┴─────────┐
     Fahrrad              Ski          …   (Plugins)
```

## 2. Teil c) – Fahrradverleih

- **C-1** Kategorien: City, Trekking, Mountainbike, E-Bike, Rennrad, Kinder – jeweils mit eigenem Stunden- und Tagespreis.
- **C-2** Felder eines Fahrrads: Inventarnummer (Pflicht, eindeutig), Marke (Pflicht), Modell,
  Rahmengröße (XS–XL, Pflicht), Laufradgröße (16″–29″, Pflicht), Gänge (1–30), Notiz.
- **C-3** E-Bikes brauchen zusätzlich die Akkukapazität (200–1000 Wh, Pflicht). Bei anderen Kategorien wird das Feld nicht angezeigt und nicht gespeichert.
- **C-4** Kinderräder haben nur 16″, 20″ oder 24″.
- **C-5** In Listen wird ein Fahrrad kurz beschrieben, z. B. „Rahmen M · 28″ · 10 Gänge · 625 Wh“.
- **C-6** Oberfläche heißt „Fahrrad-Verleih“ und spricht von „Fahrrädern“ (Dashboard, Menü, Dialoge).
- **C-7** Demo-Daten: 20 Fahrräder, 10 Personen, 3 laufende Ausleihen, 2 Reservierungen.

## 3. Teil d) – Verleih-Framework mit Plugin-Architektur

- **D-1** Das Framework (`internal/verleih`, `internal/store`, `internal/handler`, `web/`) enthält **keinen**
  fahrrad- oder ski-spezifischen Code.
- **D-2** Jeder Artikel implementiert das Interface `RentalItem`
  (`Basis()`, `Bezeichnung()`, `Kurzinfo()`, `Merkmale()`).
- **D-3** Eine Artikelart ist ein `Plugin`: Name, Symbol, Kategorien, **Feldbeschreibungen**,
  Standardpreise und eine Fabrikmethode `Neu(...)`, die den Artikel erzeugt.
- **D-4** Das Framework prüft Pflichtfelder, Zahlenbereiche und Auswahlwerte **generisch** anhand der
  Feldbeschreibungen. Das Plugin prüft nur zusätzliche eigene Regeln (z. B. C-4).
- **D-5** Felder können auf Kategorien beschränkt sein (z. B. Akku nur bei E-Bike, C-3).
- **D-6** Das Frontend baut Formulare, Filter und Texte aus `/api/meta` (Plugin-Beschreibung), ohne
  die Artikelart zu kennen.
- **D-7** Plugins werden in einer **Registry** registriert; doppelte Arten werden abgelehnt.
- **D-8** Eine neue Artikelart braucht nur ein neues Paket unter `internal/plugins/` und eine Zeile in
  `main.go`. Nachweis: das Ski-Plugin, aktivierbar mit `PLUGINS=fahrrad,ski`.
- **D-9** Bei mehreren Plugins zeigt die Oberfläche einen Filter nach Art und neutrale Texte („Artikel“).
- **D-10** Preise sind pro (Art, Kategorie) hinterlegt.
- **D-11** Optional kann ein Plugin Demo-Daten liefern (Interface `MitBeispielen`).

## 4. Akzeptanzkriterien (als Tests umgesetzt)

| ID | Kriterium | Test |
|----|-----------|------|
| D-4 | fehlendes Pflichtfeld / ungültige Auswahl / Zahl außerhalb → 400 | `verleih/plugin_test.go` |
| C-3, D-5 | E-Bike ohne Akku wird abgelehnt, City verwirft Akku-Feld | `verleih/plugin_test.go` |
| C-4 | Kinderrad mit 28″ wird abgelehnt | `verleih/plugin_test.go` |
| D-7 | Plugin doppelt registrieren → Fehler | `verleih/plugin_test.go` |
| D-8 | Ski-Plugin funktioniert ohne Framework-Änderung | `verleih/plugin_test.go`, `store/store_test.go` |
| A-9 | 50 parallele Buchungen → genau 1 erfolgreich | `store/store_test.go` |
| B-5, B-6 | Rückgabe rechnet echte Dauer mit Preis zum Buchungszeitpunkt | `store/store_test.go` |
| D-6 | `/api/meta` liefert Plugin mit Feldern | `handler/handler_test.go` |
| A-6 | Upload: falscher Typ / > 5 MB abgelehnt | `handler/handler_test.go` |
