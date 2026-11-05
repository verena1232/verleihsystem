package store

import (
	"errors"
	"sync"
	"testing"
	"time"

	"fahrradverleih/internal/plugins/fahrrad"
	"fahrradverleih/internal/plugins/ski"
	"fahrradverleih/internal/verleih"
)

var testJetzt = time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)

func testStore(t *testing.T) (*Memory, *time.Time, verleih.RentalItem, verleih.Person) {
	t.Helper()
	reg := &verleih.Registry{}
	_ = reg.Registrieren(fahrrad.Plugin{})
	uhr := testJetzt
	m := Neu(reg)
	m.Jetzt = func() time.Time { return uhr }
	item, err := m.ArtikelSpeichern(verleih.Basis{Art: "fahrrad", Nummer: "F-1", Kategorie: fahrrad.City},
		verleih.Merkmale{"marke": "Gazelle", "rahmen": "M", "laufrad": "28"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := m.PersonSpeichern(verleih.Person{Vorname: "Anna", Nachname: "Huber",
		Ausweis: verleih.Ausweis{Typ: verleih.Personalausweis, Nummer: "AT123456"}})
	if err != nil {
		t.Fatal(err)
	}
	return m, &uhr, item, p
}

func TestUeberlappungAbgelehnt(t *testing.T) {
	m, _, item, p := testStore(t)
	id, h := item.Basis().ID, time.Hour
	if _, err := m.AusleiheAnlegen(id, p.ID, testJetzt, testJetzt.Add(4*h)); err != nil {
		t.Fatal(err)
	}
	if _, err := m.AusleiheAnlegen(id, p.ID, testJetzt.Add(3*h), testJetzt.Add(6*h)); !errors.Is(err, verleih.ErrKonflikt) {
		t.Fatalf("Überlappung sollte Konflikt sein, war %v", err)
	}
	if _, err := m.AusleiheAnlegen(id, p.ID, testJetzt.Add(4*h), testJetzt.Add(6*h)); err != nil {
		t.Fatalf("Anschluss-Buchung sollte gehen: %v", err)
	}
}

func TestParalleleBuchungenNurEineErfolgreich(t *testing.T) {
	m, _, item, p := testStore(t)
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := m.AusleiheAnlegen(item.Basis().ID, p.ID, testJetzt, testJetzt.Add(2*time.Hour)); err == nil {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if ok != 1 {
		t.Fatalf("genau 1 Buchung erwartet, waren %d", ok)
	}
}

func TestPreisProKategorieUndRueckgabe(t *testing.T) {
	m, uhr, item, p := testStore(t)
	a, _ := m.AusleiheAnlegen(item.Basis().ID, p.ID, testJetzt, testJetzt.Add(4*time.Hour))
	if a.PreisCent != 1200 {
		t.Fatalf("City 4 h × 3 € = 12 €, war %d", a.PreisCent)
	}
	_ = m.PreisSetzen(verleih.Preisregel{Art: "fahrrad", Kategorie: fahrrad.City, StundenpreisCent: 900})
	*uhr = uhr.Add(90 * time.Minute)
	a, err := m.Zurueckgeben(a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if a.PreisCent != 600 {
		t.Errorf("2 angefangene Stunden × 3 € (alter Preis) = 6 € erwartet, war %d", a.PreisCent)
	}
}

func TestBearbeitenErsetztArtikel(t *testing.T) {
	m, _, item, _ := testStore(t)
	b := item.Basis()
	b.Kategorie = fahrrad.EBike
	if _, err := m.ArtikelSpeichern(b, verleih.Merkmale{"marke": "Gazelle", "rahmen": "M", "laufrad": "28"}); err == nil {
		t.Fatal("Wechsel zu E-Bike ohne Akku sollte scheitern")
	}
	neu, err := m.ArtikelSpeichern(b, verleih.Merkmale{"marke": "Gazelle", "rahmen": "M", "laufrad": "28", "akku": "500"})
	if err != nil {
		t.Fatal(err)
	}
	if neu.Basis().ID != item.Basis().ID || len(m.ArtikelListe()) != 1 {
		t.Error("Bearbeiten darf keinen neuen Artikel anlegen")
	}
	if _, err := m.ArtikelSpeichern(verleih.Basis{Art: "fahrrad", Nummer: "f-1", Kategorie: fahrrad.City},
		verleih.Merkmale{"marke": "X", "rahmen": "M", "laufrad": "28"}); !errors.Is(err, verleih.ErrKonflikt) {
		t.Errorf("doppelte Nummer sollte Konflikt sein, war %v", err)
	}
}

func TestArtikelMitOffenerAusleiheNichtLoeschbar(t *testing.T) {
	m, _, item, p := testStore(t)
	_, _ = m.AusleiheAnlegen(item.Basis().ID, p.ID, testJetzt.Add(time.Hour), testJetzt.Add(3*time.Hour))
	if err := m.ArtikelLoeschen(item.Basis().ID); !errors.Is(err, verleih.ErrKonflikt) {
		t.Errorf("Löschen mit Reservierung sollte Konflikt sein, war %v", err)
	}
}

func TestBeispieldatenMitZweiPlugins(t *testing.T) {
	reg := &verleih.Registry{}
	_ = reg.Registrieren(fahrrad.Plugin{})
	_ = reg.Registrieren(ski.Plugin{})
	m := Neu(reg)
	m.Beispieldaten()
	k := m.Dashboard(time.Now().Add(24 * time.Hour))
	if k.ProArt["fahrrad"] != 20 || k.ProArt["ski"] != 4 || k.Personen != 10 || k.Ausgeliehen != 3 {
		t.Errorf("Beispieldaten unerwartet: %+v", k)
	}
	if len(m.Preise()) != 6+5 {
		t.Errorf("Preise aller Plugins erwartet, waren %d", len(m.Preise()))
	}
}
