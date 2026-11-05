// Package verleih ist das Verleih-Framework (Aufgabe 1 d).
//
// Es kennt keine konkreten Artikel. Was verliehen wird (Fahrrad, Ski, …),
// kommt als Plugin dazu: ein Plugin beschreibt seine Felder und Kategorien
// und erzeugt passende RentalItem-Objekte.
//
//	      RentalItem (Interface)
//	           ▲
//	   ┌───────┴────────┐
//	Fahrrad            Ski        …
package verleih

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
)

// Basis enthält die Felder, die jeder Artikel hat.
type Basis struct {
	ID        int    `json:"id"`
	Art       string `json:"art"`    // Plugin-Schlüssel, z. B. "fahrrad"
	Nummer    string `json:"nummer"` // Inventarnummer, eindeutig
	Kategorie string `json:"kategorie"`
	Notiz     string `json:"notiz"`
}

// RentalItem ist alles, was verliehen werden kann.
// Implementierungen sollen unveränderlich sein: Der Store ersetzt einen
// Artikel beim Bearbeiten durch ein neues Objekt.
type RentalItem interface {
	Basis() Basis
	Bezeichnung() string         // z. B. "KTM Life Joy"
	Kurzinfo() string            // z. B. "M · 28″ · 8 Gänge"
	Merkmale() map[string]string // plugin-spezifische Felder, Schlüssel = Feld.Name
}

// FeldTyp steuert Formular und Prüfung.
type FeldTyp string

const (
	FeldText    FeldTyp = "text"
	FeldZahl    FeldTyp = "zahl"
	FeldAuswahl FeldTyp = "auswahl"
)

// Feld beschreibt ein plugin-spezifisches Eingabefeld.
// Das Frontend baut daraus das Formular, das Framework prüft damit die Eingabe.
type Feld struct {
	Name         string   `json:"name"`
	Label        string   `json:"label"`
	Typ          FeldTyp  `json:"typ"`
	Pflicht      bool     `json:"pflicht"`
	Optionen     []string `json:"optionen,omitempty"`
	Min          int      `json:"min,omitempty"`
	Max          int      `json:"max,omitempty"`
	Einheit      string   `json:"einheit,omitempty"`
	NurKategorie []string `json:"nurKategorie,omitempty"` // Feld nur bei diesen Kategorien
}

// giltFuer meldet, ob das Feld für eine Kategorie relevant ist.
func (f Feld) giltFuer(kategorie string) bool {
	return len(f.NurKategorie) == 0 || slices.Contains(f.NurKategorie, kategorie)
}

// Plugin beschreibt eine Artikelart.
type Plugin interface {
	Art() string      // Schlüssel, z. B. "fahrrad"
	Name() string     // "Fahrrad"
	Mehrzahl() string // "Fahrräder"
	Symbol() string   // "🚲"
	Praefix() string  // Vorschlag für Inventarnummern, z. B. "F-"
	Kategorien() []string
	Felder() []Feld
	Standardpreise() []Preisregel
	// Neu erzeugt einen Artikel. Basis und Felder sind bereits vom Framework geprüft;
	// das Plugin prüft nur noch eigene Regeln (z. B. "E-Bike braucht Akku").
	Neu(b Basis, m Merkmale) (RentalItem, error)
}

// Beispiel ist ein Demo-Artikel.
type Beispiel struct {
	Kategorie string
	Merkmale  Merkmale
}

// MitBeispielen ist optional: Plugins, die Demo-Daten liefern.
type MitBeispielen interface {
	Beispiele() []Beispiel
}

// ---------- Merkmale ----------

// Merkmale sind die rohen Eingaben eines Formulars.
type Merkmale map[string]string

// Text liefert ein Feld ohne Leerzeichen am Rand.
func (m Merkmale) Text(name string) string { return strings.TrimSpace(m[name]) }

// Zahl liefert ein Zahlenfeld, 0 wenn leer.
func (m Merkmale) Zahl(name string) int {
	n, _ := strconv.Atoi(m.Text(name))
	return n
}

// ---------- Registry ----------

// Registry hält alle registrierten Plugins in Registrierungsreihenfolge.
type Registry struct {
	mu      sync.RWMutex
	plugins []Plugin
}

// Registrieren fügt ein Plugin hinzu.
func (r *Registry) Registrieren(p Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p.Art() == "" || len(p.Kategorien()) == 0 {
		return fmt.Errorf("Plugin %T: Art und Kategorien sind Pflicht", p)
	}
	for _, x := range r.plugins {
		if x.Art() == p.Art() {
			return fmt.Errorf("Plugin %q ist schon registriert", p.Art())
		}
	}
	r.plugins = append(r.plugins, p)
	return nil
}

// Plugin sucht ein Plugin nach Art.
func (r *Registry) Plugin(art string) (Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.plugins {
		if p.Art() == art {
			return p, nil
		}
	}
	return nil, validierung("unbekannte Artikelart %q", art)
}

// Alle liefert alle Plugins.
func (r *Registry) Alle() []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return slices.Clone(r.plugins)
}

// Erzeugen prüft Basis + Felder generisch und lässt dann das Plugin den Artikel bauen.
func (r *Registry) Erzeugen(b Basis, m Merkmale) (RentalItem, error) {
	p, err := r.Plugin(b.Art)
	if err != nil {
		return nil, err
	}
	b.Nummer = strings.ToUpper(strings.TrimSpace(b.Nummer))
	b.Notiz = strings.TrimSpace(b.Notiz)
	switch {
	case b.Nummer == "":
		return nil, validierung("Inventarnummer fehlt")
	case !slices.Contains(p.Kategorien(), b.Kategorie):
		return nil, validierung("unbekannte Kategorie %q für %s", b.Kategorie, p.Name())
	}
	sauber := Merkmale{}
	for _, f := range p.Felder() {
		if !f.giltFuer(b.Kategorie) {
			continue // irrelevante Felder verwerfen
		}
		v := m.Text(f.Name)
		if err := f.pruefe(v); err != nil {
			return nil, err
		}
		sauber[f.Name] = v
	}
	return p.Neu(b, sauber)
}

// pruefe validiert einen Wert anhand der Feldbeschreibung.
func (f Feld) pruefe(v string) error {
	if v == "" {
		if f.Pflicht {
			return validierung("%s fehlt", f.Label)
		}
		return nil
	}
	switch f.Typ {
	case FeldZahl:
		n, err := strconv.Atoi(v)
		if err != nil {
			return validierung("%s muss eine ganze Zahl sein", f.Label)
		}
		if (f.Min != 0 || f.Max != 0) && (n < f.Min || n > f.Max) {
			return validierung("%s muss zwischen %d und %d liegen", f.Label, f.Min, f.Max)
		}
	case FeldAuswahl:
		if !slices.Contains(f.Optionen, v) {
			return validierung("ungültiger Wert für %s", f.Label)
		}
	}
	return nil
}
