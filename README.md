# Verleihsystem – Aufgabe 1

Software Engineering, Aufgabe 1: Verleihsystem, entwickelt mit Spec-Driven Development (SDD).
Jedes Projekt hat eine `SPEC.md` (Anforderungen), eine `CLAUDE.md` (Regeln für den Code) und eine `README.md`.

| Teil | Ordner | Inhalt |
|------|--------|--------|
| a) + b) | [`skiverleih/`](skiverleih) | Ski-Verleih mit Dashboard, Ausweis-Upload/-Eingabe, flexible Dauer und flexibler Beginn, Stundenpreise |
| c) (+ d) | [`fahrradverleih/`](fahrradverleih) | Fahrradverleih auf einem Verleih-Framework mit Plugin-Architektur (`RentalItem` → `Fahrrad`, `Ski`) |

## Starten

Voraussetzung: Go ≥ 1.22, keine weiteren Abhängigkeiten.

```bash
cd skiverleih      && go run .   # → http://localhost:8080
cd fahrradverleih  && go run .   # → http://localhost:8080
PLUGINS=fahrrad,ski go run .     # Fahrräder + Ski gemeinsam
```

Tests: `go test ./...` im jeweiligen Ordner.
