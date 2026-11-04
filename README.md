# Ski-Verleih – Aufgabe 1 a) + b)

Prototyp eines Ski-Verleihs mit Dashboard, entwickelt nach **Spec-Driven Development**:
zuerst [`SPEC.md`](SPEC.md) (Anforderungen mit IDs), dann [`CLAUDE.md`](CLAUDE.md) (Regeln für den Code), dann Implementierung + Tests.

## Starten

Voraussetzung: Go ≥ 1.22 (keine weiteren Abhängigkeiten).

```bash
go run .
# → http://localhost:8080
```

| Variable | Wirkung |
|----------|---------|
| `PORT=9000` | anderer Port |
| `LEER=1`    | ohne Demo-Daten starten |

Tests: `go test ./...` (mit `-race` für den Nebenläufigkeitstest).

Beim Start werden Demo-Daten geladen: 20 Ski, 10 Personen, 3 laufende Ausleihen, 2 Reservierungen.
Alle Daten liegen nur im Speicher und sind nach einem Neustart weg.

## Funktionen

### a) Ski-Verleih mit Dashboard
- **Dashboard**: verfügbare / ausgeliehene / alle Ski, registrierte Personen, dazu überfällige Ausleihen,
  heute fällige Rückgaben, Reservierungen, Umsatz; offene Ausleihen mit Rückgabe-Knopf direkt dort.
- **Ski-Verwaltung**: anlegen, bearbeiten, löschen, suchen, nach Kategorie und Status filtern.
- **Personen**: mit **Ausweis** – Typ + Nummer eingeben (Pflicht) und optional **Foto/PDF hochladen**
  (Drag & Drop oder Auswahl, Vorschau, max. 5 MB, Typ wird am Inhalt geprüft).
- **Ausleihe in 3 Schritten**: Person → Zeitraum → freien Ski wählen. Es werden nur Ski angezeigt, die
  im Zeitraum frei sind, jeweils mit Preis. Neue Person direkt aus dem Dialog anlegbar.
- Benutzerfreundlich: Dialoge, verständliche Fehlermeldungen, Bestätigung vor dem Löschen, Toasts,
  funktioniert auch am Handy.

### b) Flexible Dauer, flexibler Beginn, Stundenpreise
- Beginn frei wählbar (Standard: jetzt). Liegt er in der Zukunft, ist es eine **Reservierung**, die storniert werden kann.
- Dauer frei (1 h bis 30 Tage), Schnellauswahl 2 h / 4 h / 8 h / 24 h / 3 Tage / 1 Woche.
- **Stundenpreise** pro Kategorie mit Tageshöchstpreis, im Menü **Preise** änderbar:

  ```
  Preis = volleTage × Tagespreis + min(restStunden × Stundenpreis, Tagespreis)
  ```
  Beispiel Allround (4 €/h, max. 25 €/Tag): 3 h = 12 €, 7 h = 25 €, 26 h = 25 € + 8 € = 33 €.
- Bei der Rückgabe wird mit der **tatsächlichen** Dauer gerechnet (zum Preis, der bei der Buchung galt).

## Architektur

```
main.go                 HTTP-Server, liefert web/ aus (embed)
internal/domain/        Ski, Person, Ausweis, Ausleihe, Preisregel + Fachlogik
internal/store/         thread-safer In-Memory-Store (RWMutex), Demo-Daten
internal/handler/       JSON-REST-API
web/                    index.html, style.css, app.js (ohne Framework)
```

Prüfen auf Überschneidung und Speichern einer Ausleihe passieren unter **einem** Lock,
deshalb kann ein Ski auch bei gleichzeitigen Anfragen nie doppelt vergeben werden (Test: 50 parallele Buchungen → genau 1 erfolgreich).

## REST-API

| Methode | Pfad | Zweck |
|---------|------|-------|
| GET | `/api/dashboard` | Kennzahlen + offene Ausleihen |
| GET/POST | `/api/ski` | Liste / anlegen |
| PUT/DELETE | `/api/ski/{id}` | ändern / löschen |
| GET/POST | `/api/personen` | Liste / anlegen |
| PUT/DELETE | `/api/personen/{id}` | ändern / löschen |
| GET/POST/DELETE | `/api/personen/{id}/ausweis` | Ausweisdatei ansehen / hochladen (multipart, Feld `datei`) / entfernen |
| GET | `/api/preise` | Preisregeln |
| PUT | `/api/preise/{kategorie}` | Preisregel ändern |
| GET | `/api/verfuegbar?von=…&bis=…` | freie Ski im Zeitraum inkl. Preis (RFC3339) |
| GET/POST | `/api/ausleihen` | Liste / neue Ausleihe |
| POST | `/api/ausleihen/{id}/rueckgabe` | Rückgabe |
| POST | `/api/ausleihen/{id}/storno` | Reservierung stornieren |

Fehler kommen als `{"fehler": "…"}` mit Status 400 (Eingabe), 404 (nicht gefunden) oder 409 (Konflikt).
