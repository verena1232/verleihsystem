# Fahrrad-Verleih auf Verleih-Framework (Prototyp)

Nur Prototyp. Priorität: schnell verständlich, leicht änderbar.
Fachliche Anforderungen stehen in `SPEC.md` (A-/B-x aus dem Ski-Verleih, C-x Fahrrad, D-x Framework).
**Vor jeder Änderung zuerst SPEC.md anpassen, dann Code + Tests.**

## Stack
- Backend: Go (Structs + Methoden, Interfaces wo sie echte Erweiterungspunkte sind)
- DB: In-Memory (Maps + `sync.RWMutex`), kein ORM
- Frontend: reines HTML/CSS/JS in `web/`, per `embed` eingebaut
- Keine externen Dependencies (nur Go-Standardbibliothek)

## Commands
- Starten: `go run .` → http://localhost:8080
- Mit Ski-Plugin: `PLUGINS=fahrrad,ski go run .`
- Tests: `go test ./...` (gern mit `-race`)
- Format / Vet: `go fmt ./...`, `go vet ./...`

## Architektur
- `main.go` → registriert Plugins, startet Server
- `internal/verleih/` → **Framework-Kern**: `RentalItem`, `Plugin`, `Feld`, `Registry`, Person, Ausweis, Ausleihe, Preisregel
- `internal/store/` → thread-safer In-Memory-Store, arbeitet nur mit `RentalItem`
- `internal/handler/` → JSON-REST-API, kennt keine konkrete Artikelart
- `internal/plugins/fahrrad/` → Fahrrad-Plugin (Teil c)
- `internal/plugins/ski/` → Ski-Plugin (Nachweis der Erweiterbarkeit, Teil d)
- `web/` → generisches Frontend, baut Formulare aus `/api/meta`

## Regeln für das Framework
- In `verleih`, `store`, `handler`, `web` **nie** artikel-spezifischen Code schreiben (D-1).
  Braucht ein Plugin etwas Neues, das Framework allgemein erweitern (z. B. neuer `FeldTyp`).
- `RentalItem`-Objekte sind unveränderlich; Bearbeiten = neues Objekt über `Registry.Erzeugen`.
- Generische Feldprüfung im Framework, nur Sonderregeln im Plugin (`Plugin.Neu`).

## Neues Plugin anlegen
1. Paket `internal/plugins/<art>/` mit Struct, das `verleih.RentalItem` implementiert
2. `Plugin`-Struct mit `Art, Name, Mehrzahl, Symbol, Praefix, Kategorien, Felder, Standardpreise, Neu`
3. optional `Beispiele()` für Demo-Daten
4. Eintrag in `verfuegbarePlugins` in `main.go`
5. Test, der das Plugin über `Registry.Erzeugen` anlegt

## Konventionen
- Geld in Cent (`int`), Zeiten als `time.Time` / RFC3339
- Fehler mit `fmt.Errorf("...: %w", err)` wrappen; `ErrValidierung` → 400, `ErrNichtGefunden` → 404, `ErrKonflikt` → 409
- Prüfen + Einfügen einer Ausleihe unter einem Lock
- Meldungen an Benutzer:innen auf Deutsch
- Keine neuen Dependencies ohne Nachfrage, keine Secrets im Code
- Commit-Messages kurz und imperativ; bei Unklarheit fragen statt raten
