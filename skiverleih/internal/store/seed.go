package store

import (
	"fmt"
	"time"

	"skiverleih/internal/domain"
)

// Beispieldaten füllt den Store mit Demo-Daten: 20 Ski, 10 Personen,
// laufende Ausleihen, Reservierungen und abgeschlossene Ausleihen.
func (m *Memory) Beispieldaten() {
	ski := []struct {
		marke, modell string
		laenge        int
		kat           domain.Kategorie
	}{
		{"Atomic", "Redster G9", 175, domain.KatRace},
		{"Atomic", "Redster G9", 182, domain.KatRace},
		{"Fischer", "RC4 WC", 165, domain.KatRace},
		{"Head", "Supershape e-Rally", 170, domain.KatAllround},
		{"Head", "Supershape e-Rally", 163, domain.KatAllround},
		{"Rossignol", "Experience 80", 168, domain.KatAllround},
		{"Rossignol", "Experience 80", 176, domain.KatAllround},
		{"Völkl", "Deacon 76", 158, domain.KatAllround},
		{"Völkl", "Deacon 76", 172, domain.KatAllround},
		{"Elan", "Wingman 82", 166, domain.KatAllround},
		{"Salomon", "QST 98", 181, domain.KatFreeride},
		{"Salomon", "QST 106", 188, domain.KatFreeride},
		{"Black Crows", "Atris", 184, domain.KatFreeride},
		{"Dynafit", "Blacklight 88", 170, domain.KatTouren},
		{"Dynafit", "Seven Summits", 177, domain.KatTouren},
		{"Ski Trab", "Maestro", 171, domain.KatTouren},
		{"Atomic", "Redster J4", 120, domain.KatKinder},
		{"Head", "Monster Kid", 107, domain.KatKinder},
		{"Rossignol", "Hero Jr", 130, domain.KatKinder},
		{"Fischer", "Ranger Jr", 140, domain.KatKinder},
	}
	for i, s := range ski {
		_, _ = m.SkiSpeichern(domain.Ski{
			Nummer: fmt.Sprintf("S-%03d", i+1), Marke: s.marke, Modell: s.modell,
			LaengeCm: s.laenge, Kategorie: s.kat,
		})
	}

	personen := []domain.Person{
		{Vorname: "Anna", Nachname: "Huber", Email: "anna.huber@example.at", Telefon: "+43 660 1234567", Ausweis: domain.Ausweis{Typ: domain.Personalausweis, Nummer: "AT1234567"}},
		{Vorname: "Lukas", Nachname: "Gruber", Email: "lukas.gruber@example.at", Telefon: "+43 664 2345678", Ausweis: domain.Ausweis{Typ: domain.Reisepass, Nummer: "P7654321"}},
		{Vorname: "Sophie", Nachname: "Wagner", Email: "sophie.w@example.de", Ausweis: domain.Ausweis{Typ: domain.Personalausweis, Nummer: "L01X00T47"}},
		{Vorname: "Jonas", Nachname: "Bauer", Telefon: "+43 650 9876543", Ausweis: domain.Ausweis{Typ: domain.Fuehrerschein, Nummer: "FS123456"}},
		{Vorname: "Lena", Nachname: "Pichler", Email: "lena.pichler@example.at", Ausweis: domain.Ausweis{Typ: domain.Personalausweis, Nummer: "AT9988776"}},
		{Vorname: "Maximilian", Nachname: "Steiner", Email: "max.steiner@example.at", Ausweis: domain.Ausweis{Typ: domain.Reisepass, Nummer: "U1234567"}},
		{Vorname: "Laura", Nachname: "Moser", Telefon: "+43 676 1112233", Ausweis: domain.Ausweis{Typ: domain.Personalausweis, Nummer: "AT5554443"}},
		{Vorname: "Felix", Nachname: "Hofer", Email: "felix.hofer@example.ch", Ausweis: domain.Ausweis{Typ: domain.Reisepass, Nummer: "X0000123"}},
		{Vorname: "Hannah", Nachname: "Leitner", Email: "hannah.l@example.at", Ausweis: domain.Ausweis{Typ: domain.Fuehrerschein, Nummer: "FS998877"}},
		{Vorname: "David", Nachname: "Berger", Telefon: "+43 699 4445566", Ausweis: domain.Ausweis{Typ: domain.Personalausweis, Nummer: "AT3332221"}},
	}
	for _, p := range personen {
		_, _ = m.PersonSpeichern(p)
	}

	// Ausleihen direkt einfügen, damit auch Vergangenes möglich ist.
	jetzt := m.jetzt().Truncate(15 * time.Minute)
	h := time.Hour
	demo := []struct {
		ski, person  int // Index ab 1
		beginn, ende time.Duration
		zurueck      *time.Duration
	}{
		// laufend (3 Ski ausgeliehen)
		{4, 21, -2 * h, 3 * h, nil},
		{11, 22, -26 * h, 22 * h, nil},
		{17, 23, -1 * h, 1 * h, nil},
		// Reservierungen
		{1, 24, 20 * h, 44 * h, nil},
		{14, 25, 48 * h, 120 * h, nil},
		// abgeschlossen
		{6, 26, -72 * h, -48 * h, dauer(-49 * h)},
		{12, 27, -30 * h, -26 * h, dauer(-25 * h)},
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, d := range demo {
		s := m.ski[d.ski]
		r := m.preise[s.Kategorie]
		a := domain.Ausleihe{
			ID: m.neueID(), SkiID: d.ski, PersonID: d.person,
			Beginn: jetzt.Add(d.beginn), Ende: jetzt.Add(d.ende),
			Preis: r, Erstellt: jetzt.Add(d.beginn - h),
		}
		a.PreisCent = r.Berechne(a.Beginn, a.Ende)
		if d.zurueck != nil {
			t := jetzt.Add(*d.zurueck)
			a.Rueckgabe = &t
			a.PreisCent = r.Berechne(a.Beginn, t)
		}
		m.ausleihen[a.ID] = a
	}
}

func dauer(d time.Duration) *time.Duration { return &d }
