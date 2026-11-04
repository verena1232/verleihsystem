package domain

import (
	"testing"
	"time"
)

func TestBerechne(t *testing.T) {
	allround := Preisregel{Kategorie: KatAllround, StundenpreisCent: 400, TagespreisCent: 2500}
	ohneDeckel := Preisregel{Kategorie: KatAllround, StundenpreisCent: 400}
	von := time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC)

	faelle := []struct {
		name  string
		regel Preisregel
		dauer time.Duration
		cent  int
	}{
		{"genau 1 h", allround, time.Hour, 400},
		{"1 h 1 min = 2 angefangene Stunden (B-3)", allround, time.Hour + time.Minute, 800},
		{"4 h", allround, 4 * time.Hour, 1600},
		{"7 h gedeckelt auf Tagespreis (B-4)", allround, 7 * time.Hour, 2500},
		{"24 h = 1 Tag", allround, 24 * time.Hour, 2500},
		{"26 h = 1 Tag + 2 h (B-4)", allround, 26 * time.Hour, 3300},
		{"3 Tage", allround, 72 * time.Hour, 7500},
		{"ohne Tagesdeckel 26 h", ohneDeckel, 26 * time.Hour, 26 * 400},
		{"0 Dauer", allround, 0, 0},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if got := f.regel.Berechne(von, von.Add(f.dauer)); got != f.cent {
				t.Errorf("Berechne = %d, erwartet %d", got, f.cent)
			}
		})
	}
}

func TestPreisregelValidieren(t *testing.T) {
	if err := (Preisregel{Kategorie: KatRace, StundenpreisCent: 500, TagespreisCent: 300}).Validieren(); err == nil {
		t.Error("Tagespreis < Stundenpreis sollte ungültig sein")
	}
	if err := (Preisregel{Kategorie: "Snowboard", StundenpreisCent: 500}).Validieren(); err == nil {
		t.Error("unbekannte Kategorie sollte ungültig sein")
	}
}

func TestNeueAusleiheZeitraum(t *testing.T) {
	jetzt := time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC)
	r := Preisregel{Kategorie: KatAllround, StundenpreisCent: 400, TagespreisCent: 2500}

	if _, err := NeueAusleihe(1, 1, jetzt, jetzt.Add(30*time.Minute), jetzt, r); err == nil {
		t.Error("unter 1 h sollte abgelehnt werden (B-2)")
	}
	if _, err := NeueAusleihe(1, 1, jetzt.Add(-time.Hour), jetzt.Add(time.Hour), jetzt, r); err == nil {
		t.Error("Beginn in der Vergangenheit sollte abgelehnt werden (B-1)")
	}
	if _, err := NeueAusleihe(1, 1, jetzt, jetzt.Add(31*24*time.Hour), jetzt, r); err == nil {
		t.Error("über 30 Tage sollte abgelehnt werden (B-2)")
	}
	a, err := NeueAusleihe(1, 1, jetzt.Add(48*time.Hour), jetzt.Add(50*time.Hour), jetzt, r)
	if err != nil {
		t.Fatal(err)
	}
	if a.Status(jetzt) != StatusReserviert {
		t.Errorf("Beginn in Zukunft → reserviert, war %s (B-8)", a.Status(jetzt))
	}
	if a.Status(jetzt.Add(49*time.Hour)) != StatusAktiv {
		t.Error("während Zeitraum → aktiv")
	}
	if a.Status(jetzt.Add(51*time.Hour)) != StatusUeberfaellig {
		t.Error("nach Ende → überfällig")
	}
}

func TestPersonValidieren(t *testing.T) {
	p := Person{Vorname: "Anna", Nachname: "Huber", Ausweis: Ausweis{Typ: Reisepass, Nummer: " p123456 "}}
	if err := p.Validieren(); err != nil {
		t.Fatal(err)
	}
	if p.Ausweis.Nummer != "P123456" {
		t.Errorf("Ausweisnummer sollte bereinigt werden, war %q", p.Ausweis.Nummer)
	}
	p.Ausweis.Typ = ""
	if err := p.Validieren(); err == nil {
		t.Error("ohne Ausweistyp sollte ungültig sein (A-5)")
	}
}
