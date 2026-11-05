# Ski-Verleih (Prototyp)

Nur Prototyp. Priorität: schnell verständlich, leicht änderbar.
Fachliche Anforderungen stehen in `SPEC.md` (IDs A-x für Teil a, B-x für Teil b).
**Vor jeder Änderung zuerst SPEC.md anpassen, dann Code + Tests.**

## Stack
- Backend: Go (Structs + Methoden, Interfaces nur wo sinnvoll)
- DB: In-Memory (Maps + `sync.RWMutex`), kein ORM, keine echte DB
- Frontend: reines HTML/CSS/JS in `web/`, per `embed` in das Binary eingebaut
- Keine externen Dependencies (nur Go-Standardbibliothek)

## Commands
- Starten: `go run .` → http://localhost:8080 (Port über `PORT` änderbar)
- Tests: `go test ./...`
- Format: `go fmt ./...`
- Vet: `go vet ./...`

## Architektur
- `main.go` → Einstieg, HTTP-Server, statische Dateien
- `internal/domain/` → Ski, Person, Ausweis, Ausleihe, Preisregel + Fachlogik (Status, Preisberechnung, Überlappung)
- `internal/store/` → thread-safer In-Memory-Store, Beispieldaten (`seed.go`)
- `internal/handler/` → JSON-REST-API unter `/api/...`, ruft nur den Store auf
- `web/` → `index.html`, `style.css`, `app.js`

## Konventionen
- Geldbeträge immer als `int` in Cent (`...Cent`), nie float
- Zeiten als `time.Time`, JSON im RFC3339-Format; Frontend schickt `toISOString()`
- Fachregeln gehören in `domain` (Methoden an Structs), nicht in Handler
- Prüfen + Einfügen einer Ausleihe passiert unter einem Lock (A-9)
- Store gibt Kopien zurück, nie Pointer auf interne Daten
- Fehler mit `fmt.Errorf("...: %w", err)` wrappen; Handler mappt
  `ErrValidierung` → 400, `ErrNichtGefunden` → 404, `ErrKonflikt` → 409
- Fehlermeldungen an Benutzer:innen auf Deutsch
- Keine neuen Dependencies ohne Nachfrage, keine Secrets im Code

## Frontend
- Minimales JS, kein Framework, keine Build-Tools
- Alle Daten nur über die `/api`-Endpoints
- Neue Seite = neue `<section data-page="...">` + Eintrag in `seiten` in `app.js`

## Regeln
- Commit-Messages: kurz und imperativ
- Bei Unklarheit fragen statt raten
- Lieber einfach und klar als elegant und komplex
