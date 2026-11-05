package store

import (
	"errors"
	"sync"
	"testing"
	"time"

	"skiverleih/internal/domain"
)

var testJetzt = time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC)

func testStore(t *testing.T) (*Memory, *time.Time, domain.Ski, domain.Person) {
	t.Helper()
	uhr := testJetzt
	m := Neu()
	m.Jetzt = func() time.Time { return uhr }
	s, err := m.SkiSpeichern(domain.Ski{Nummer: "S-1", Marke: "Atomic", LaengeCm: 170, Kategorie: domain.KatAllround})
	if err != nil {
		t.Fatal(err)
	}
	p, err := m.PersonSpeichern(domain.Person{Vorname: "Anna", Nachname: "Huber",
		Ausweis: domain.Ausweis{Typ: domain.Personalausweis, Nummer: "AT123456"}})
	if err != nil {
		t.Fatal(err)
	}
	return m, &uhr, s, p
}

func TestUeberlappungAbgelehnt(t *testing.T) {
	m, _, s, p := testStore(t)
	h := time.Hour
	if _, err := m.AusleiheAnlegen(s.ID, p.ID, testJetzt, testJetzt.Add(4*h)); err != nil {
		t.Fatal(err)
	}
	_, err := m.AusleiheAnlegen(s.ID, p.ID, testJetzt.Add(3*h), testJetzt.Add(6*h))
	if !errors.Is(err, domain.ErrKonflikt) {
		t.Fatalf("Überlappung sollte Konflikt sein (A-9), war %v", err)
	}
	// direkt anschließend ist erlaubt ([Beginn, Ende) halboffen)
	if _, err := m.AusleiheAnlegen(s.ID, p.ID, testJetzt.Add(4*h), testJetzt.Add(6*h)); err != nil {
		t.Fatalf("Anschluss-Buchung sollte gehen: %v", err)
	}
}

func TestParalleleBuchungenNurEineErfolgreich(t *testing.T) {
	m, _, s, p := testStore(t)
	var wg sync.WaitGroup
	var mu sync.Mutex
	erfolgreich := 0
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := m.AusleiheAnlegen(s.ID, p.ID, testJetzt, testJetzt.Add(2*time.Hour)); err == nil {
				mu.Lock()
				erfolgreich++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if erfolgreich != 1 {
		t.Fatalf("genau 1 Buchung erwartet (A-9), waren %d", erfolgreich)
	}
}

func TestSkiMitOffenerAusleiheNichtLoeschbar(t *testing.T) {
	m, uhr, s, p := testStore(t)
	a, _ := m.AusleiheAnlegen(s.ID, p.ID, testJetzt, testJetzt.Add(2*time.Hour))
	if err := m.SkiLoeschen(s.ID); !errors.Is(err, domain.ErrKonflikt) {
		t.Fatalf("Löschen sollte Konflikt sein (A-2), war %v", err)
	}
	if err := m.PersonLoeschen(p.ID); !errors.Is(err, domain.ErrKonflikt) {
		t.Fatalf("Person löschen sollte Konflikt sein (A-7), war %v", err)
	}
	*uhr = uhr.Add(time.Hour)
	if _, err := m.Zurueckgeben(a.ID); err != nil {
		t.Fatal(err)
	}
	if err := m.SkiLoeschen(s.ID); err != nil {
		t.Fatalf("nach Rückgabe sollte Löschen gehen: %v", err)
	}
}

func TestRueckgabeRechnetEchteDauer(t *testing.T) {
	m, uhr, s, p := testStore(t)
	a, _ := m.AusleiheAnlegen(s.ID, p.ID, testJetzt, testJetzt.Add(5*time.Hour))
	if a.PreisCent != 2000 {
		t.Fatalf("geplant 5 h × 4 € = 20 €, war %d", a.PreisCent)
	}
	// Preisänderung danach darf die Buchung nicht beeinflussen (B-5)
	_ = m.PreisSetzen(domain.Preisregel{Kategorie: domain.KatAllround, StundenpreisCent: 1000, TagespreisCent: 5000})

	*uhr = uhr.Add(2*time.Hour + 10*time.Minute) // früher zurück
	a, err := m.Zurueckgeben(a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if a.PreisCent != 1200 {
		t.Errorf("3 angefangene Stunden × 4 € = 12 € erwartet (B-6), war %d", a.PreisCent)
	}
	if _, err := m.Zurueckgeben(a.ID); !errors.Is(err, domain.ErrKonflikt) {
		t.Error("zweite Rückgabe sollte Konflikt sein")
	}
}

func TestReservierungStornieren(t *testing.T) {
	m, uhr, s, p := testStore(t)
	a, _ := m.AusleiheAnlegen(s.ID, p.ID, testJetzt.Add(24*time.Hour), testJetzt.Add(26*time.Hour))
	if _, err := m.Zurueckgeben(a.ID); !errors.Is(err, domain.ErrKonflikt) {
		t.Error("Reservierung zurückgeben sollte nicht gehen")
	}
	if _, err := m.Stornieren(a.ID); err != nil {
		t.Fatal(err)
	}
	// nach Storno ist der Ski wieder frei
	if len(m.Verfuegbar(testJetzt.Add(24*time.Hour), testJetzt.Add(26*time.Hour))) != 1 {
		t.Error("Ski sollte nach Storno verfügbar sein (B-7)")
	}
	_ = uhr
}

func TestUeberfaelligBlockiertSki(t *testing.T) {
	m, uhr, s, p := testStore(t)
	_, _ = m.AusleiheAnlegen(s.ID, p.ID, testJetzt, testJetzt.Add(2*time.Hour))
	*uhr = uhr.Add(5 * time.Hour) // 3 h überfällig
	if n := len(m.Verfuegbar(*uhr, uhr.Add(time.Hour))); n != 0 {
		t.Errorf("überfälliger Ski darf nicht verfügbar sein, %d verfügbar", n)
	}
	k := m.Dashboard(uhr.Add(12 * time.Hour))
	if k.Ueberfaellig != 1 || k.Ausgeliehen != 1 || k.Verfuegbar != 0 {
		t.Errorf("Dashboard falsch: %+v", k)
	}
}

func TestInventarnummerEindeutig(t *testing.T) {
	m, _, _, _ := testStore(t)
	_, err := m.SkiSpeichern(domain.Ski{Nummer: "s-1", Marke: "Head", LaengeCm: 160, Kategorie: domain.KatRace})
	if !errors.Is(err, domain.ErrKonflikt) {
		t.Errorf("doppelte Inventarnummer sollte Konflikt sein, war %v", err)
	}
}

func TestBeispieldaten(t *testing.T) {
	m := Neu()
	m.Beispieldaten()
	k := m.Dashboard(time.Now().Add(24 * time.Hour))
	if k.Gesamt != 20 || k.Personen != 10 || k.Ausgeliehen != 3 || k.Reservierungen != 2 {
		t.Errorf("Beispieldaten unerwartet: %+v", k)
	}
}
