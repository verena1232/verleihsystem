package domain

import "strings"

// Kategorie eines Ski. Die Kategorie bestimmt die Preisregel.
type Kategorie string

const (
	KatKinder   Kategorie = "Kinder"
	KatAllround Kategorie = "Allround"
	KatRace     Kategorie = "Race"
	KatFreeride Kategorie = "Freeride"
	KatTouren   Kategorie = "Touren"
)

// Kategorien in Anzeigereihenfolge.
var Kategorien = []Kategorie{KatKinder, KatAllround, KatRace, KatFreeride, KatTouren}

// Gueltig prüft, ob die Kategorie bekannt ist.
func (k Kategorie) Gueltig() bool {
	for _, x := range Kategorien {
		if x == k {
			return true
		}
	}
	return false
}

// Ski ist ein verleihbares Paar Ski (A-1).
type Ski struct {
	ID        int       `json:"id"`
	Nummer    string    `json:"nummer"`
	Marke     string    `json:"marke"`
	Modell    string    `json:"modell"`
	LaengeCm  int       `json:"laengeCm"`
	Kategorie Kategorie `json:"kategorie"`
	Notiz     string    `json:"notiz"`
}

// Bereinigen entfernt Leerzeichen an den Rändern.
func (s *Ski) Bereinigen() {
	s.Nummer = strings.TrimSpace(s.Nummer)
	s.Marke = strings.TrimSpace(s.Marke)
	s.Modell = strings.TrimSpace(s.Modell)
	s.Notiz = strings.TrimSpace(s.Notiz)
}

// Validieren prüft die Pflichtfelder (A-1).
func (s *Ski) Validieren() error {
	s.Bereinigen()
	switch {
	case s.Nummer == "":
		return validierung("Inventarnummer fehlt")
	case s.Marke == "":
		return validierung("Marke fehlt")
	case s.LaengeCm < 70 || s.LaengeCm > 220:
		return validierung("Länge muss zwischen 70 und 220 cm liegen")
	case !s.Kategorie.Gueltig():
		return validierung("unbekannte Kategorie %q", s.Kategorie)
	}
	return nil
}

// Bezeichnung für Anzeigen, z. B. "S-007 Atomic Redster".
func (s Ski) Bezeichnung() string {
	b := s.Nummer + " " + s.Marke
	if s.Modell != "" {
		b += " " + s.Modell
	}
	return b
}
