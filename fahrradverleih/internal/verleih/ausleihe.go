package verleih

import "time"

// Grenzen für die Verleihdauer.
const (
	MinDauer = time.Hour
	MaxDauer = 30 * 24 * time.Hour
	Toleranz = 5 * time.Minute // so weit darf "Beginn" in der Vergangenheit liegen
)

// Status einer Ausleihe, ergibt sich aus der Zeit.
type Status string

const (
	StatusReserviert     Status = "reserviert"
	StatusAktiv          Status = "aktiv"
	StatusUeberfaellig   Status = "überfällig"
	StatusZurueckgegeben Status = "zurückgegeben"
	StatusStorniert      Status = "storniert"
)

// Ausleihe verbindet einen Artikel mit einer Person für einen Zeitraum.
type Ausleihe struct {
	ID        int        `json:"id"`
	ArtikelID int        `json:"artikelId"`
	PersonID  int        `json:"personId"`
	Beginn    time.Time  `json:"beginn"`
	Ende      time.Time  `json:"ende"` // geplantes Ende
	Rueckgabe *time.Time `json:"rueckgabe,omitempty"`
	Storniert bool       `json:"storniert"`
	Preis     Preisregel `json:"preisregel"` // bei Buchung gültige Regel
	PreisCent int        `json:"preisCent"`  // geplanter bzw. finaler Preis
	Erstellt  time.Time  `json:"erstellt"`
}

// NeueAusleihe prüft den Zeitraum und berechnet den Preis.
func NeueAusleihe(artikelID, personID int, beginn, ende, jetzt time.Time, regel Preisregel) (Ausleihe, error) {
	beginn, ende = beginn.UTC(), ende.UTC()
	switch {
	case beginn.IsZero() || ende.IsZero():
		return Ausleihe{}, validierung("Beginn und Ende angeben")
	case beginn.Before(jetzt.Add(-Toleranz)):
		return Ausleihe{}, validierung("Beginn liegt in der Vergangenheit")
	case ende.Sub(beginn) < MinDauer:
		return Ausleihe{}, validierung("Mindestdauer ist 1 Stunde")
	case ende.Sub(beginn) > MaxDauer:
		return Ausleihe{}, validierung("Höchstdauer ist 30 Tage")
	}
	return Ausleihe{
		ArtikelID: artikelID,
		PersonID:  personID,
		Beginn:    beginn,
		Ende:      ende,
		Preis:     regel,
		PreisCent: regel.Berechne(beginn, ende),
		Erstellt:  jetzt.UTC(),
	}, nil
}

// Status zum Zeitpunkt jetzt.
func (a Ausleihe) Status(jetzt time.Time) Status {
	switch {
	case a.Storniert:
		return StatusStorniert
	case a.Rueckgabe != nil:
		return StatusZurueckgegeben
	case jetzt.Before(a.Beginn):
		return StatusReserviert
	case jetzt.After(a.Ende):
		return StatusUeberfaellig
	default:
		return StatusAktiv
	}
}

// Offen ist eine Ausleihe, solange sie weder zurückgegeben noch storniert ist.
func (a Ausleihe) Offen() bool {
	return !a.Storniert && a.Rueckgabe == nil
}

// belegtBis: bis wann der Artikel durch diese Ausleihe blockiert ist.
// Ein überfälliger Artikel bleibt blockiert, bis er tatsächlich zurück ist.
func (a Ausleihe) belegtBis(jetzt time.Time) time.Time {
	if jetzt.After(a.Ende) {
		return time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return a.Ende
}

// Blockiert meldet, ob diese Ausleihe den Zeitraum [von, bis) überlappt.
func (a Ausleihe) Blockiert(von, bis, jetzt time.Time) bool {
	if a.Storniert || a.Rueckgabe != nil {
		return false
	}
	return von.Before(a.belegtBis(jetzt)) && a.Beginn.Before(bis)
}

// Zurueckgeben schließt die Ausleihe ab und rechnet mit der echten Dauer.
func (a *Ausleihe) Zurueckgeben(jetzt time.Time) error {
	switch a.Status(jetzt) {
	case StatusZurueckgegeben:
		return Konflikt("Artikel wurde bereits zurückgegeben")
	case StatusStorniert:
		return Konflikt("Ausleihe ist storniert")
	case StatusReserviert:
		return Konflikt("Reservierung hat noch nicht begonnen – bitte stornieren")
	}
	t := jetzt.UTC()
	a.Rueckgabe = &t
	a.PreisCent = a.Preis.Berechne(a.Beginn, t)
	return nil
}

// Stornieren ist nur vor Beginn möglich.
func (a *Ausleihe) Stornieren(jetzt time.Time) error {
	if a.Status(jetzt) != StatusReserviert {
		return Konflikt("nur Reservierungen, die noch nicht begonnen haben, können storniert werden")
	}
	a.Storniert = true
	a.PreisCent = 0
	return nil
}
