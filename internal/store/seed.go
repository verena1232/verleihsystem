package store

import (
	"fmt"
	"log"
	"time"

	"fahrradverleih/internal/verleih"
)

// Beispieldaten füllt den Store mit Demo-Daten: die Beispiele aller Plugins,
// 10 Personen, laufende, reservierte und abgeschlossene Ausleihen.
func (m *Memory) Beispieldaten() {
	var ids []int
	for _, p := range m.Registry.Alle() {
		bsp, ok := p.(verleih.MitBeispielen)
		if !ok {
			continue
		}
		for i, b := range bsp.Beispiele() {
			item, err := m.ArtikelSpeichern(verleih.Basis{
				Art: p.Art(), Nummer: fmt.Sprintf("%s%03d", p.Praefix(), i+1), Kategorie: b.Kategorie,
			}, b.Merkmale)
			if err != nil {
				log.Printf("Beispiel %s %d: %v", p.Art(), i+1, err)
				continue
			}
			ids = append(ids, item.Basis().ID)
		}
	}

	personen := []verleih.Person{
		{Vorname: "Anna", Nachname: "Huber", Email: "anna.huber@example.at", Telefon: "+43 660 1234567", Ausweis: verleih.Ausweis{Typ: verleih.Personalausweis, Nummer: "AT1234567"}},
		{Vorname: "Lukas", Nachname: "Gruber", Email: "lukas.gruber@example.at", Telefon: "+43 664 2345678", Ausweis: verleih.Ausweis{Typ: verleih.Reisepass, Nummer: "P7654321"}},
		{Vorname: "Sophie", Nachname: "Wagner", Email: "sophie.w@example.de", Ausweis: verleih.Ausweis{Typ: verleih.Personalausweis, Nummer: "L01X00T47"}},
		{Vorname: "Jonas", Nachname: "Bauer", Telefon: "+43 650 9876543", Ausweis: verleih.Ausweis{Typ: verleih.Fuehrerschein, Nummer: "FS123456"}},
		{Vorname: "Lena", Nachname: "Pichler", Email: "lena.pichler@example.at", Ausweis: verleih.Ausweis{Typ: verleih.Personalausweis, Nummer: "AT9988776"}},
		{Vorname: "Maximilian", Nachname: "Steiner", Email: "max.steiner@example.at", Ausweis: verleih.Ausweis{Typ: verleih.Reisepass, Nummer: "U1234567"}},
		{Vorname: "Laura", Nachname: "Moser", Telefon: "+43 676 1112233", Ausweis: verleih.Ausweis{Typ: verleih.Personalausweis, Nummer: "AT5554443"}},
		{Vorname: "Felix", Nachname: "Hofer", Email: "felix.hofer@example.ch", Ausweis: verleih.Ausweis{Typ: verleih.Reisepass, Nummer: "X0000123"}},
		{Vorname: "Hannah", Nachname: "Leitner", Email: "hannah.l@example.at", Ausweis: verleih.Ausweis{Typ: verleih.Fuehrerschein, Nummer: "FS998877"}},
		{Vorname: "David", Nachname: "Berger", Telefon: "+43 699 4445566", Ausweis: verleih.Ausweis{Typ: verleih.Personalausweis, Nummer: "AT3332221"}},
	}
	var pids []int
	for _, p := range personen {
		gp, _ := m.PersonSpeichern(p)
		pids = append(pids, gp.ID)
	}
	if len(ids) < 14 {
		return
	}

	jetzt := m.jetzt().Truncate(15 * time.Minute)
	h := time.Hour
	demo := []struct {
		artikel, person int // Index in ids / pids
		beginn, ende    time.Duration
		zurueck         time.Duration // 0 = offen
	}{
		{3, 0, -2 * h, 3 * h, 0},    // laufend
		{10, 1, -26 * h, 22 * h, 0}, // laufend, E-Bike
		{16, 2, -1 * h, 1 * h, 0},   // laufend, Kinderrad
		{0, 3, 20 * h, 44 * h, 0},   // Reservierung
		{13, 4, 48 * h, 120 * h, 0}, // Reservierung
		{5, 5, -72 * h, -48 * h, -49 * h},
		{8, 6, -30 * h, -26 * h, -25 * h},
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, d := range demo {
		item := m.artikel[ids[d.artikel%len(ids)]]
		r := m.regelFuer(item)
		a := verleih.Ausleihe{
			ID: m.neueID(), ArtikelID: item.Basis().ID, PersonID: pids[d.person],
			Beginn: jetzt.Add(d.beginn), Ende: jetzt.Add(d.ende), Preis: r, Erstellt: jetzt.Add(d.beginn - h),
		}
		a.PreisCent = r.Berechne(a.Beginn, a.Ende)
		if d.zurueck != 0 {
			t := jetzt.Add(d.zurueck)
			a.Rueckgabe = &t
			a.PreisCent = r.Berechne(a.Beginn, t)
		}
		m.ausleihen[a.ID] = a
	}
}
