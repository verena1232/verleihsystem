# Fahrrad-Verleih – Aufgabe 1 c) (+ d)

Fahrradverleih statt Ski-Verleih, gebaut als **Verleih-Framework mit Plugin-Architektur**.
Entwickelt nach Spec-Driven Development: [`SPEC.md`](SPEC.md) → [`CLAUDE.md`](CLAUDE.md) → Code + Tests.

## Starten

Voraussetzung: Go ≥ 1.22, keine weiteren Abhängigkeiten.

```bash
go run .                       # Fahrrad-Verleih   → http://localhost:8080
PLUGINS=fahrrad,ski go run .   # Fahrräder + Ski im selben System
```

| Variable | Wirkung |
|----------|---------|
| `PLUGINS` | aktive Plugins, Standard `fahrrad` |
| `PORT`    | anderer Port, Standard `8080` |
| `LEER=1`  | ohne Demo-Daten starten |

Tests: `go test ./...`

## Was kann der Fahrrad-Verleih? (c)

Alles aus dem Ski-Verleih (Dashboard, Personen mit Ausweis-Upload, flexible Dauer und Beginn,
Stundenpreise mit Tagesdeckel, Reservierungen, Rückgabe mit Abrechnung der echten Dauer), aber für Fahrräder:

- Kategorien **City, Trekking, Mountainbike, E-Bike, Rennrad, Kinder**, jede mit eigenem Preis
- Felder: Marke, Modell, Rahmengröße, Laufradgröße, Gänge, bei **E-Bikes zusätzlich Akku (Wh)**
- Kinderräder nur mit 16″/20″/24″
- Demo-Daten: 20 Fahrräder, 10 Personen, laufende Ausleihen und Reservierungen

## Plugin-Architektur (d)

```
                 ┌──────────────── Framework ────────────────┐
  web/ (JS)  ──► │ handler  ──►  store  ──►  verleih         │
  /api/meta      │ (REST)       (Memory)     RentalItem,      │
                 │                           Plugin, Registry │
                 └──────────────────────────────▲────────────┘
                                                │ implementiert
                               ┌────────────────┴───────────┐
                        plugins/fahrrad                plugins/ski
```

Das Framework kennt nur das Interface:

```go
type RentalItem interface {
    Basis() Basis                // ID, Art, Nummer, Kategorie, Notiz
    Bezeichnung() string         // "KTM Life Joy"
    Kurzinfo() string            // "Rahmen M · 28″ · 24 Gänge"
    Merkmale() map[string]string // plugin-spezifische Werte
}

type Plugin interface {
    Art() string; Name() string; Mehrzahl() string; Symbol() string; Praefix() string
    Kategorien() []string
    Felder() []Feld              // beschreibt das Formular
    Standardpreise() []Preisregel
    Neu(b Basis, m Merkmale) (RentalItem, error)
}
```

- Ein Plugin **beschreibt seine Felder** (Text / Zahl / Auswahl, Pflicht, Min/Max, nur bei bestimmten Kategorien).
- Das Framework prüft diese Felder **generisch**; das Plugin prüft nur Sonderregeln.
- Das Frontend holt sich die Beschreibung über `/api/meta` und baut daraus Formulare, Filter und Texte.
  Darum steht im Frontend nirgends „Fahrrad“.

### Neue Artikelart hinzufügen

Beispiel Ski (liegt schon bei, `internal/plugins/ski/ski.go`, ca. 60 Zeilen):

1. Paket `internal/plugins/<art>/` mit einem Struct, das `RentalItem` implementiert, und einem `Plugin`
2. in `main.go` eintragen:
   ```go
   var verfuegbarePlugins = map[string]verleih.Plugin{
       "fahrrad": fahrrad.Plugin{},
       "ski":     ski.Plugin{},   // ← neue Zeile
   }
   ```
3. `PLUGINS=fahrrad,ski go run .`

An Framework, Store, API und Frontend ändert sich dabei **nichts**.

## Projektstruktur

```
main.go                       Plugins registrieren, Server starten
internal/verleih/             Framework-Kern (RentalItem, Plugin, Registry, Ausleihe, Preis, Person)
internal/store/               thread-safer In-Memory-Store + Demo-Daten
internal/handler/             REST-API
internal/plugins/fahrrad/     Fahrrad-Plugin (Teil c)
internal/plugins/ski/         Ski-Plugin (Nachweis Erweiterbarkeit)
web/                          generisches Frontend
```

## REST-API

Wie beim Ski-Verleih, aber mit `/api/artikel` statt `/api/ski`:

| Methode | Pfad | Zweck |
|---------|------|-------|
| GET | `/api/meta` | Titel, Plugins mit Kategorien und Feldern, Ausweistypen |
| GET/POST | `/api/artikel` | Liste / anlegen (`{art, nummer, kategorie, notiz, merkmale:{…}}`) |
| PUT/DELETE | `/api/artikel/{id}` | ändern / löschen |
| PUT | `/api/preise/{art}/{kategorie}` | Preisregel ändern |
| GET | `/api/verfuegbar?von=…&bis=…` | freie Artikel im Zeitraum inkl. Preis |
| … | `/api/personen`, `/api/ausleihen`, `/api/dashboard` | wie im Ski-Verleih |
