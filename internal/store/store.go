// Package store hält alle Daten thread-safe im Speicher.
package store

import (
	"slices"
	"strings"
	"sync"
	"time"

	"skiverleih/internal/domain"
)

// Memory ist der In-Memory-Store. Alle Methoden sind thread-safe.
// Nach außen werden nur Kopien herausgegeben.
type Memory struct {
	mu        sync.RWMutex
	naechste  int
	ski       map[int]domain.Ski
	personen  map[int]domain.Person
	ausleihen map[int]domain.Ausleihe
	preise    map[domain.Kategorie]domain.Preisregel

	// Jetzt liefert die aktuelle Zeit; in Tests austauschbar.
	Jetzt func() time.Time
}

// Neu erzeugt einen leeren Store mit Standardpreisen.
func Neu() *Memory {
	m := &Memory{
		ski:       map[int]domain.Ski{},
		personen:  map[int]domain.Person{},
		ausleihen: map[int]domain.Ausleihe{},
		preise:    map[domain.Kategorie]domain.Preisregel{},
		Jetzt:     time.Now,
	}
	for _, r := range Standardpreise() {
		m.preise[r.Kategorie] = r
	}
	return m
}

// Standardpreise je Kategorie in Cent.
func Standardpreise() []domain.Preisregel {
	return []domain.Preisregel{
		{Kategorie: domain.KatKinder, StundenpreisCent: 250, TagespreisCent: 1500},
		{Kategorie: domain.KatAllround, StundenpreisCent: 400, TagespreisCent: 2500},
		{Kategorie: domain.KatRace, StundenpreisCent: 600, TagespreisCent: 3500},
		{Kategorie: domain.KatFreeride, StundenpreisCent: 600, TagespreisCent: 3800},
		{Kategorie: domain.KatTouren, StundenpreisCent: 700, TagespreisCent: 4200},
	}
}

func (m *Memory) neueID() int {
	m.naechste++
	return m.naechste
}

func (m *Memory) jetzt() time.Time { return m.Jetzt().UTC() }

// ---------- Ski ----------

// SkiListe liefert alle Ski sortiert nach Inventarnummer.
func (m *Memory) SkiListe() []domain.Ski {
	m.mu.RLock()
	defer m.mu.RUnlock()
	liste := make([]domain.Ski, 0, len(m.ski))
	for _, s := range m.ski {
		liste = append(liste, s)
	}
	slices.SortFunc(liste, func(a, b domain.Ski) int { return strings.Compare(a.Nummer, b.Nummer) })
	return liste
}

// Ski liefert einen Ski.
func (m *Memory) Ski(id int) (domain.Ski, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.ski[id]
	if !ok {
		return domain.Ski{}, domain.NichtGefunden("Ski %d", id)
	}
	return s, nil
}

// SkiSpeichern legt an (ID == 0) oder aktualisiert (A-1).
func (m *Memory) SkiSpeichern(s domain.Ski) (domain.Ski, error) {
	if err := s.Validieren(); err != nil {
		return domain.Ski{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if s.ID != 0 {
		if _, ok := m.ski[s.ID]; !ok {
			return domain.Ski{}, domain.NichtGefunden("Ski %d", s.ID)
		}
	}
	for _, x := range m.ski {
		if x.ID != s.ID && strings.EqualFold(x.Nummer, s.Nummer) {
			return domain.Ski{}, domain.Konflikt("Inventarnummer %s ist schon vergeben", s.Nummer)
		}
	}
	if s.ID == 0 {
		s.ID = m.neueID()
	}
	m.ski[s.ID] = s
	return s, nil
}

// SkiLoeschen entfernt einen Ski ohne offene Ausleihe (A-2).
func (m *Memory) SkiLoeschen(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.ski[id]; !ok {
		return domain.NichtGefunden("Ski %d", id)
	}
	for _, a := range m.ausleihen {
		if a.SkiID == id && a.Offen() {
			return domain.Konflikt("Ski hat noch eine offene Ausleihe oder Reservierung")
		}
	}
	delete(m.ski, id)
	return nil
}

// ---------- Personen ----------

// Personen liefert alle Personen sortiert nach Nachname.
func (m *Memory) Personen() []domain.Person {
	m.mu.RLock()
	defer m.mu.RUnlock()
	liste := make([]domain.Person, 0, len(m.personen))
	for _, p := range m.personen {
		liste = append(liste, p)
	}
	slices.SortFunc(liste, func(a, b domain.Person) int {
		return strings.Compare(a.Nachname+a.Vorname, b.Nachname+b.Vorname)
	})
	return liste
}

// Person liefert eine Person.
func (m *Memory) Person(id int) (domain.Person, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.personen[id]
	if !ok {
		return domain.Person{}, domain.NichtGefunden("Person %d", id)
	}
	return p, nil
}

// PersonSpeichern legt an (ID == 0) oder aktualisiert (A-4, A-5).
// Eine bereits hochgeladene Ausweisdatei bleibt beim Aktualisieren erhalten.
func (m *Memory) PersonSpeichern(p domain.Person) (domain.Person, error) {
	if err := p.Validieren(); err != nil {
		return domain.Person{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if p.ID == 0 {
		p.ID = m.neueID()
		p.Ausweis.Datei = nil
	} else {
		alt, ok := m.personen[p.ID]
		if !ok {
			return domain.Person{}, domain.NichtGefunden("Person %d", p.ID)
		}
		p.Ausweis.Datei = alt.Ausweis.Datei
	}
	m.personen[p.ID] = p
	return p, nil
}

// PersonLoeschen entfernt eine Person ohne offene Ausleihe (A-7).
func (m *Memory) PersonLoeschen(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.personen[id]; !ok {
		return domain.NichtGefunden("Person %d", id)
	}
	for _, a := range m.ausleihen {
		if a.PersonID == id && a.Offen() {
			return domain.Konflikt("Person hat noch eine offene Ausleihe oder Reservierung")
		}
	}
	delete(m.personen, id)
	return nil
}

// AusweisDateiSetzen speichert (oder entfernt bei nil) die Ausweisdatei (A-6).
func (m *Memory) AusweisDateiSetzen(personID int, d *domain.AusweisDatei) error {
	if d != nil {
		if err := domain.PruefeAusweisDatei(d.MimeType, len(d.Daten)); err != nil {
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
		return domain.NichtGefunden("Person %d", personID)
	}
	p.Ausweis.Datei = d
	m.personen[personID] = p
	return nil
}

// ---------- Preise ----------

// Preise liefert alle Preisregeln in Kategorie-Reihenfolge.
func (m *Memory) Preise() []domain.Preisregel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	liste := make([]domain.Preisregel, 0, len(m.preise))
	for _, k := range domain.Kategorien {
		liste = append(liste, m.preise[k])
	}
	return liste
}

// Preisregel liefert die Regel einer Kategorie.
func (m *Memory) Preisregel(k domain.Kategorie) (domain.Preisregel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.preise[k]
	if !ok {
		return domain.Preisregel{}, domain.NichtGefunden("Preisregel %s", k)
	}
	return r, nil
}

// PreisSetzen ändert eine Preisregel (B-5).
func (m *Memory) PreisSetzen(r domain.Preisregel) error {
	if err := r.Validieren(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.preise[r.Kategorie] = r
	return nil
}

// ---------- Ausleihen ----------

// Ausleihen liefert alle Ausleihen, neueste Beginnzeit zuerst.
func (m *Memory) Ausleihen() []domain.Ausleihe {
	m.mu.RLock()
	defer m.mu.RUnlock()
	liste := make([]domain.Ausleihe, 0, len(m.ausleihen))
	for _, a := range m.ausleihen {
		liste = append(liste, a)
	}
	slices.SortFunc(liste, func(a, b domain.Ausleihe) int { return b.Beginn.Compare(a.Beginn) })
	return liste
}

// skiFrei prüft (unter Lock!), ob ein Ski im Zeitraum frei ist (A-9).
func (m *Memory) skiFrei(skiID int, von, bis, jetzt time.Time) bool {
	for _, a := range m.ausleihen {
		if a.SkiID == skiID && a.Blockiert(von, bis, jetzt) {
			return false
		}
	}
	return true
}

// Verfuegbar liefert alle Ski, die im Zeitraum [von, bis) frei sind (A-8).
func (m *Memory) Verfuegbar(von, bis time.Time) []domain.Ski {
	jetzt := m.jetzt()
	frei := []domain.Ski{}
	for _, s := range m.SkiListe() {
		m.mu.RLock()
		ok := m.skiFrei(s.ID, von, bis, jetzt)
		m.mu.RUnlock()
		if ok {
			frei = append(frei, s)
		}
	}
	return frei
}

// Angebot berechnet den Preis für einen Ski im Zeitraum, ohne zu buchen (A-17).
func (m *Memory) Angebot(skiID int, von, bis time.Time) (int, error) {
	s, err := m.Ski(skiID)
	if err != nil {
		return 0, err
	}
	r, err := m.Preisregel(s.Kategorie)
	if err != nil {
		return 0, err
	}
	a, err := domain.NeueAusleihe(skiID, 0, von, bis, m.jetzt(), r)
	if err != nil {
		return 0, err
	}
	return a.PreisCent, nil
}

// AusleiheAnlegen bucht einen Ski. Prüfen und Einfügen passieren unter
// demselben Lock, damit nie doppelt gebucht wird (A-8, A-9).
func (m *Memory) AusleiheAnlegen(skiID, personID int, von, bis time.Time) (domain.Ausleihe, error) {
	jetzt := m.jetzt()
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.ski[skiID]
	if !ok {
		return domain.Ausleihe{}, domain.NichtGefunden("Ski %d", skiID)
	}
	if _, ok := m.personen[personID]; !ok {
		return domain.Ausleihe{}, domain.NichtGefunden("Person %d", personID)
	}
	a, err := domain.NeueAusleihe(skiID, personID, von, bis, jetzt, m.preise[s.Kategorie])
	if err != nil {
		return domain.Ausleihe{}, err
	}
	if !m.skiFrei(skiID, a.Beginn, a.Ende, jetzt) {
		return domain.Ausleihe{}, domain.Konflikt("Ski %s ist in diesem Zeitraum schon vergeben", s.Nummer)
	}
	a.ID = m.neueID()
	m.ausleihen[a.ID] = a
	return a, nil
}

// aendern wendet f unter Lock auf eine Ausleihe an.
func (m *Memory) aendern(id int, f func(*domain.Ausleihe, time.Time) error) (domain.Ausleihe, error) {
	jetzt := m.jetzt()
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.ausleihen[id]
	if !ok {
		return domain.Ausleihe{}, domain.NichtGefunden("Ausleihe %d", id)
	}
	if err := f(&a, jetzt); err != nil {
		return domain.Ausleihe{}, err
	}
	m.ausleihen[id] = a
	return a, nil
}

// Zurueckgeben schließt eine Ausleihe ab (A-10, B-6).
func (m *Memory) Zurueckgeben(id int) (domain.Ausleihe, error) {
	return m.aendern(id, (*domain.Ausleihe).Zurueckgeben)
}

// Stornieren storniert eine Reservierung (B-7).
func (m *Memory) Stornieren(id int) (domain.Ausleihe, error) {
	return m.aendern(id, (*domain.Ausleihe).Stornieren)
}

// ---------- Dashboard ----------

// Kennzahlen für das Dashboard (A-11, A-12).
type Kennzahlen struct {
	Verfuegbar      int `json:"verfuegbar"`
	Ausgeliehen     int `json:"ausgeliehen"`
	Gesamt          int `json:"gesamt"`
	Personen        int `json:"personen"`
	Ueberfaellig    int `json:"ueberfaellig"`
	HeuteFaellig    int `json:"heuteFaellig"`
	Reservierungen  int `json:"reservierungen"`
	UmsatzCent      int `json:"umsatzCent"`
	OffenerWertCent int `json:"offenerWertCent"`
}

// Dashboard berechnet die Kennzahlen zum aktuellen Zeitpunkt.
func (m *Memory) Dashboard(tagesende time.Time) Kennzahlen {
	jetzt := m.jetzt()
	m.mu.RLock()
	defer m.mu.RUnlock()

	k := Kennzahlen{Gesamt: len(m.ski), Personen: len(m.personen)}
	unterwegs := map[int]bool{}
	for _, a := range m.ausleihen {
		switch a.Status(jetzt) {
		case domain.StatusAktiv:
			unterwegs[a.SkiID] = true
			k.OffenerWertCent += a.PreisCent
			if !a.Ende.After(tagesende) {
				k.HeuteFaellig++
			}
		case domain.StatusUeberfaellig:
			unterwegs[a.SkiID] = true
			k.Ueberfaellig++
			k.OffenerWertCent += a.PreisCent
		case domain.StatusReserviert:
			k.Reservierungen++
		case domain.StatusZurueckgegeben:
			k.UmsatzCent += a.PreisCent
		}
	}
	k.Ausgeliehen = len(unterwegs)
	k.Verfuegbar = k.Gesamt - k.Ausgeliehen
	return k
}
