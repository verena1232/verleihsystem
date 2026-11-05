// Package store hält alle Daten thread-safe im Speicher.
// Er arbeitet nur mit verleih.RentalItem und kennt keine konkreten Artikel.
package store

import (
	"slices"
	"strings"
	"sync"
	"time"

	"fahrradverleih/internal/verleih"
)

// Memory ist der In-Memory-Store. Alle Methoden sind thread-safe.
type Memory struct {
	mu        sync.RWMutex
	naechste  int
	artikel   map[int]verleih.RentalItem // Artikel sind unveränderlich
	personen  map[int]verleih.Person
	ausleihen map[int]verleih.Ausleihe
	preise    map[string]verleih.Preisregel // Schlüssel "art/kategorie"

	Registry *verleih.Registry
	// Jetzt liefert die aktuelle Zeit; in Tests austauschbar.
	Jetzt func() time.Time
}

// Neu erzeugt einen leeren Store mit den Standardpreisen aller Plugins.
func Neu(reg *verleih.Registry) *Memory {
	m := &Memory{
		artikel:   map[int]verleih.RentalItem{},
		personen:  map[int]verleih.Person{},
		ausleihen: map[int]verleih.Ausleihe{},
		preise:    map[string]verleih.Preisregel{},
		Registry:  reg,
		Jetzt:     time.Now,
	}
	for _, p := range reg.Alle() {
		for _, r := range p.Standardpreise() {
			r.Art = p.Art()
			m.preise[r.Schluessel()] = r
		}
		for _, k := range p.Kategorien() { // fehlende Preise ergänzen
			if _, ok := m.preise[verleih.PreisSchluessel(p.Art(), k)]; !ok {
				m.preise[verleih.PreisSchluessel(p.Art(), k)] = verleih.Preisregel{Art: p.Art(), Kategorie: k, StundenpreisCent: 500}
			}
		}
	}
	return m
}

func (m *Memory) neueID() int {
	m.naechste++
	return m.naechste
}

func (m *Memory) jetzt() time.Time { return m.Jetzt().UTC() }

// ---------- Artikel ----------

// ArtikelListe liefert alle Artikel sortiert nach Inventarnummer.
func (m *Memory) ArtikelListe() []verleih.RentalItem {
	m.mu.RLock()
	defer m.mu.RUnlock()
	liste := make([]verleih.RentalItem, 0, len(m.artikel))
	for _, a := range m.artikel {
		liste = append(liste, a)
	}
	slices.SortFunc(liste, func(a, b verleih.RentalItem) int { return strings.Compare(a.Basis().Nummer, b.Basis().Nummer) })
	return liste
}

// Artikel liefert einen Artikel.
func (m *Memory) Artikel(id int) (verleih.RentalItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	a, ok := m.artikel[id]
	if !ok {
		return nil, verleih.NichtGefunden("Artikel %d", id)
	}
	return a, nil
}

// ArtikelSpeichern legt an (b.ID == 0) oder ersetzt einen Artikel.
// Das zuständige Plugin baut das Objekt.
func (m *Memory) ArtikelSpeichern(b verleih.Basis, merkmale verleih.Merkmale) (verleih.RentalItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if b.ID != 0 {
		alt, ok := m.artikel[b.ID]
		if !ok {
			return nil, verleih.NichtGefunden("Artikel %d", b.ID)
		}
		b.Art = alt.Basis().Art // Art ist nicht änderbar
	} else {
		b.ID = m.naechste + 1 // vorläufig, erst nach erfolgreicher Prüfung vergeben
	}
	item, err := m.Registry.Erzeugen(b, merkmale)
	if err != nil {
		return nil, err
	}
	nr := item.Basis().Nummer
	for _, x := range m.artikel {
		if x.Basis().ID != b.ID && strings.EqualFold(x.Basis().Nummer, nr) {
			return nil, verleih.Konflikt("Inventarnummer %s ist schon vergeben", nr)
		}
	}
	if item.Basis().ID > m.naechste {
		m.naechste = item.Basis().ID
	}
	m.artikel[item.Basis().ID] = item
	return item, nil
}

// ArtikelLoeschen entfernt einen Artikel ohne offene Ausleihe.
func (m *Memory) ArtikelLoeschen(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.artikel[id]; !ok {
		return verleih.NichtGefunden("Artikel %d", id)
	}
	for _, a := range m.ausleihen {
		if a.ArtikelID == id && a.Offen() {
			return verleih.Konflikt("Artikel hat noch eine offene Ausleihe oder Reservierung")
		}
	}
	delete(m.artikel, id)
	return nil
}

// ---------- Personen ----------

// Personen liefert alle Personen sortiert nach Nachname.
func (m *Memory) Personen() []verleih.Person {
	m.mu.RLock()
	defer m.mu.RUnlock()
	liste := make([]verleih.Person, 0, len(m.personen))
	for _, p := range m.personen {
		liste = append(liste, p)
	}
	slices.SortFunc(liste, func(a, b verleih.Person) int {
		return strings.Compare(a.Nachname+a.Vorname, b.Nachname+b.Vorname)
	})
	return liste
}

// Person liefert eine Person.
func (m *Memory) Person(id int) (verleih.Person, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.personen[id]
	if !ok {
		return verleih.Person{}, verleih.NichtGefunden("Person %d", id)
	}
	return p, nil
}

// PersonSpeichern legt an (ID == 0) oder aktualisiert.
// Eine bereits hochgeladene Ausweisdatei bleibt beim Aktualisieren erhalten.
func (m *Memory) PersonSpeichern(p verleih.Person) (verleih.Person, error) {
	if err := p.Validieren(); err != nil {
		return verleih.Person{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if p.ID == 0 {
		p.ID = m.neueID()
		p.Ausweis.Datei = nil
	} else {
		alt, ok := m.personen[p.ID]
		if !ok {
			return verleih.Person{}, verleih.NichtGefunden("Person %d", p.ID)
		}
		p.Ausweis.Datei = alt.Ausweis.Datei
	}
	m.personen[p.ID] = p
	return p, nil
}

// PersonLoeschen entfernt eine Person ohne offene Ausleihe.
func (m *Memory) PersonLoeschen(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.personen[id]; !ok {
		return verleih.NichtGefunden("Person %d", id)
	}
	for _, a := range m.ausleihen {
		if a.PersonID == id && a.Offen() {
			return verleih.Konflikt("Person hat noch eine offene Ausleihe oder Reservierung")
		}
	}
	delete(m.personen, id)
	return nil
}

// AusweisDateiSetzen speichert (oder entfernt bei nil) die Ausweisdatei.
func (m *Memory) AusweisDateiSetzen(personID int, d *verleih.AusweisDatei) error {
	if d != nil {
		if err := verleih.PruefeAusweisDatei(d.MimeType, len(d.Daten)); err != nil {
			return err
		}
		kopie := *d
		kopie.Daten = slices.Clone(d.Daten)
		kopie.Hochgeladen = m.jetzt()
		d = &kopie
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.personen[personID]
	if !ok {
		return verleih.NichtGefunden("Person %d", personID)
	}
	p.Ausweis.Datei = d
	m.personen[personID] = p
	return nil
}

// ---------- Preise ----------

// Preise liefert alle Preisregeln in Plugin- und Kategorie-Reihenfolge.
func (m *Memory) Preise() []verleih.Preisregel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	liste := []verleih.Preisregel{}
	for _, p := range m.Registry.Alle() {
		for _, k := range p.Kategorien() {
			liste = append(liste, m.preise[verleih.PreisSchluessel(p.Art(), k)])
		}
	}
	return liste
}

// PreisSetzen ändert eine Preisregel.
func (m *Memory) PreisSetzen(r verleih.Preisregel) error {
	if err := r.Validieren(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.preise[r.Schluessel()]; !ok {
		return verleih.NichtGefunden("Preisregel %s", r.Schluessel())
	}
	m.preise[r.Schluessel()] = r
	return nil
}

func (m *Memory) regelFuer(item verleih.RentalItem) verleih.Preisregel {
	b := item.Basis()
	return m.preise[verleih.PreisSchluessel(b.Art, b.Kategorie)]
}

// ---------- Ausleihen ----------

// Ausleihen liefert alle Ausleihen, neueste Beginnzeit zuerst.
func (m *Memory) Ausleihen() []verleih.Ausleihe {
	m.mu.RLock()
	defer m.mu.RUnlock()
	liste := make([]verleih.Ausleihe, 0, len(m.ausleihen))
	for _, a := range m.ausleihen {
		liste = append(liste, a)
	}
	slices.SortFunc(liste, func(a, b verleih.Ausleihe) int { return b.Beginn.Compare(a.Beginn) })
	return liste
}

// frei prüft (unter Lock!), ob ein Artikel im Zeitraum frei ist.
func (m *Memory) frei(id int, von, bis, jetzt time.Time) bool {
	for _, a := range m.ausleihen {
		if a.ArtikelID == id && a.Blockiert(von, bis, jetzt) {
			return false
		}
	}
	return true
}

// Angebot ist ein freier Artikel samt Preis für den angefragten Zeitraum.
type Angebot struct {
	Artikel   verleih.RentalItem
	PreisCent int
}

// Verfuegbar liefert alle Artikel, die im Zeitraum [von, bis) frei sind, mit Preis.
func (m *Memory) Verfuegbar(von, bis time.Time) []Angebot {
	jetzt := m.jetzt()
	m.mu.RLock()
	defer m.mu.RUnlock()
	liste := []Angebot{}
	for _, a := range m.artikel {
		if m.frei(a.Basis().ID, von, bis, jetzt) {
			liste = append(liste, Angebot{Artikel: a, PreisCent: m.regelFuer(a).Berechne(von, bis)})
		}
	}
	slices.SortFunc(liste, func(a, b Angebot) int { return strings.Compare(a.Artikel.Basis().Nummer, b.Artikel.Basis().Nummer) })
	return liste
}

// AusleiheAnlegen bucht einen Artikel. Prüfen und Einfügen passieren unter
// demselben Lock, damit nie doppelt gebucht wird.
func (m *Memory) AusleiheAnlegen(artikelID, personID int, von, bis time.Time) (verleih.Ausleihe, error) {
	jetzt := m.jetzt()
	m.mu.Lock()
	defer m.mu.Unlock()

	item, ok := m.artikel[artikelID]
	if !ok {
		return verleih.Ausleihe{}, verleih.NichtGefunden("Artikel %d", artikelID)
	}
	if _, ok := m.personen[personID]; !ok {
		return verleih.Ausleihe{}, verleih.NichtGefunden("Person %d", personID)
	}
	a, err := verleih.NeueAusleihe(artikelID, personID, von, bis, jetzt, m.regelFuer(item))
	if err != nil {
		return verleih.Ausleihe{}, err
	}
	if !m.frei(artikelID, a.Beginn, a.Ende, jetzt) {
		return verleih.Ausleihe{}, verleih.Konflikt("%s ist in diesem Zeitraum schon vergeben", item.Basis().Nummer)
	}
	a.ID = m.neueID()
	m.ausleihen[a.ID] = a
	return a, nil
}

func (m *Memory) aendern(id int, f func(*verleih.Ausleihe, time.Time) error) (verleih.Ausleihe, error) {
	jetzt := m.jetzt()
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.ausleihen[id]
	if !ok {
		return verleih.Ausleihe{}, verleih.NichtGefunden("Ausleihe %d", id)
	}
	if err := f(&a, jetzt); err != nil {
		return verleih.Ausleihe{}, err
	}
	m.ausleihen[id] = a
	return a, nil
}

// Zurueckgeben schließt eine Ausleihe ab.
func (m *Memory) Zurueckgeben(id int) (verleih.Ausleihe, error) {
	return m.aendern(id, (*verleih.Ausleihe).Zurueckgeben)
}

// Stornieren storniert eine Reservierung.
func (m *Memory) Stornieren(id int) (verleih.Ausleihe, error) {
	return m.aendern(id, (*verleih.Ausleihe).Stornieren)
}

// ---------- Dashboard ----------

// Kennzahlen für das Dashboard.
type Kennzahlen struct {
	Verfuegbar     int            `json:"verfuegbar"`
	Ausgeliehen    int            `json:"ausgeliehen"`
	Gesamt         int            `json:"gesamt"`
	Personen       int            `json:"personen"`
	Ueberfaellig   int            `json:"ueberfaellig"`
	HeuteFaellig   int            `json:"heuteFaellig"`
	Reservierungen int            `json:"reservierungen"`
	UmsatzCent     int            `json:"umsatzCent"`
	ProArt         map[string]int `json:"proArt"` // Anzahl Artikel je Plugin
}

// Dashboard berechnet die Kennzahlen zum aktuellen Zeitpunkt.
func (m *Memory) Dashboard(tagesende time.Time) Kennzahlen {
	jetzt := m.jetzt()
	m.mu.RLock()
	defer m.mu.RUnlock()

	k := Kennzahlen{Gesamt: len(m.artikel), Personen: len(m.personen), ProArt: map[string]int{}}
	for _, a := range m.artikel {
		k.ProArt[a.Basis().Art]++
	}
	unterwegs := map[int]bool{}
	for _, a := range m.ausleihen {
		switch a.Status(jetzt) {
		case verleih.StatusAktiv:
			unterwegs[a.ArtikelID] = true
			if !a.Ende.After(tagesende) {
				k.HeuteFaellig++
			}
		case verleih.StatusUeberfaellig:
			unterwegs[a.ArtikelID] = true
			k.Ueberfaellig++
		case verleih.StatusReserviert:
			k.Reservierungen++
		case verleih.StatusZurueckgegeben:
			k.UmsatzCent += a.PreisCent
		}
	}
	k.Ausgeliehen = len(unterwegs)
	k.Verfuegbar = k.Gesamt - k.Ausgeliehen
	return k
}
