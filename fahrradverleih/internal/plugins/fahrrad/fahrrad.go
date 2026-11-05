// Package fahrrad ist das Plugin für den Fahrradverleih (Aufgabe 1 c).
package fahrrad

import (
	"fmt"
	"strings"

	"fahrradverleih/internal/verleih"
)

// Kategorien
const (
	City         = "City"
	Trekking     = "Trekking"
	Mountainbike = "Mountainbike"
	EBike        = "E-Bike"
	Rennrad      = "Rennrad"
	Kinder       = "Kinder"
)

// Fahrrad ist ein RentalItem.
type Fahrrad struct {
	basis         verleih.Basis
	Marke         string
	Modell        string
	Rahmengroesse string
	Laufrad       string // Zoll
	Gaenge        int
	AkkuWh        int // nur E-Bike
}

// Basis implementiert verleih.RentalItem.
func (f *Fahrrad) Basis() verleih.Basis { return f.basis }

// Bezeichnung implementiert verleih.RentalItem.
func (f *Fahrrad) Bezeichnung() string { return strings.TrimSpace(f.Marke + " " + f.Modell) }

// Kurzinfo implementiert verleih.RentalItem, z. B. "M · 28″ · 8 Gänge · 500 Wh".
func (f *Fahrrad) Kurzinfo() string {
	teile := []string{"Rahmen " + f.Rahmengroesse, f.Laufrad + "″"}
	if f.Gaenge > 0 {
		teile = append(teile, fmt.Sprintf("%d Gänge", f.Gaenge))
	}
	if f.AkkuWh > 0 {
		teile = append(teile, fmt.Sprintf("%d Wh", f.AkkuWh))
	}
	return strings.Join(teile, " · ")
}

// Merkmale implementiert verleih.RentalItem.
func (f *Fahrrad) Merkmale() map[string]string {
	m := map[string]string{
		"marke": f.Marke, "modell": f.Modell, "rahmen": f.Rahmengroesse, "laufrad": f.Laufrad,
	}
	if f.Gaenge > 0 {
		m["gaenge"] = fmt.Sprint(f.Gaenge)
	}
	if f.AkkuWh > 0 {
		m["akku"] = fmt.Sprint(f.AkkuWh)
	}
	return m
}

// Plugin registriert Fahrräder beim Framework.
type Plugin struct{}

func (Plugin) Art() string      { return "fahrrad" }
func (Plugin) Name() string     { return "Fahrrad" }
func (Plugin) Mehrzahl() string { return "Fahrräder" }
func (Plugin) Symbol() string   { return "🚲" }
func (Plugin) Praefix() string  { return "F-" }

func (Plugin) Kategorien() []string {
	return []string{City, Trekking, Mountainbike, EBike, Rennrad, Kinder}
}

func (Plugin) Felder() []verleih.Feld {
	return []verleih.Feld{
		{Name: "marke", Label: "Marke", Typ: verleih.FeldText, Pflicht: true},
		{Name: "modell", Label: "Modell", Typ: verleih.FeldText},
		{Name: "rahmen", Label: "Rahmengröße", Typ: verleih.FeldAuswahl, Pflicht: true,
			Optionen: []string{"XS", "S", "M", "L", "XL"}},
		{Name: "laufrad", Label: "Laufradgröße", Typ: verleih.FeldAuswahl, Pflicht: true, Einheit: "Zoll",
			Optionen: []string{"16", "20", "24", "26", "27,5", "28", "29"}},
		{Name: "gaenge", Label: "Gänge", Typ: verleih.FeldZahl, Min: 1, Max: 30},
		{Name: "akku", Label: "Akku", Typ: verleih.FeldZahl, Einheit: "Wh", Min: 200, Max: 1000,
			Pflicht: true, NurKategorie: []string{EBike}},
	}
}

func (Plugin) Standardpreise() []verleih.Preisregel {
	r := func(k string, h, t int) verleih.Preisregel {
		return verleih.Preisregel{Art: "fahrrad", Kategorie: k, StundenpreisCent: h, TagespreisCent: t}
	}
	return []verleih.Preisregel{
		r(City, 300, 1500), r(Trekking, 400, 2000), r(Mountainbike, 600, 3000),
		r(EBike, 800, 4000), r(Rennrad, 700, 3500), r(Kinder, 200, 1000),
	}
}

// Neu baut ein Fahrrad. Pflichtfelder/Wertebereiche hat das Framework schon geprüft.
func (Plugin) Neu(b verleih.Basis, m verleih.Merkmale) (verleih.RentalItem, error) {
	f := &Fahrrad{
		basis:         b,
		Marke:         m.Text("marke"),
		Modell:        m.Text("modell"),
		Rahmengroesse: m.Text("rahmen"),
		Laufrad:       m.Text("laufrad"),
		Gaenge:        m.Zahl("gaenge"),
		AkkuWh:        m.Zahl("akku"),
	}
	// eigene Regel: Kinderräder nur bis 24 Zoll
	if b.Kategorie == Kinder && !(f.Laufrad == "16" || f.Laufrad == "20" || f.Laufrad == "24") {
		return nil, fmt.Errorf("%w: Kinderräder haben 16, 20 oder 24 Zoll", verleih.ErrValidierung)
	}
	return f, nil
}

// Beispiele liefert Demo-Fahrräder (optional, verleih.MitBeispielen).
func (Plugin) Beispiele() []verleih.Beispiel {
	b := func(kat, marke, modell, rahmen, laufrad, gaenge, akku string) verleih.Beispiel {
		return verleih.Beispiel{Kategorie: kat, Merkmale: verleih.Merkmale{
			"marke": marke, "modell": modell, "rahmen": rahmen, "laufrad": laufrad, "gaenge": gaenge, "akku": akku}}
	}
	return []verleih.Beispiel{
		b(City, "Gazelle", "Esprit", "M", "28", "3", ""),
		b(City, "Gazelle", "Esprit", "L", "28", "3", ""),
		b(City, "Puky", "Classic", "S", "26", "7", ""),
		b(Trekking, "KTM", "Life Joy", "M", "28", "24", ""),
		b(Trekking, "KTM", "Life Joy", "L", "28", "24", ""),
		b(Trekking, "Cube", "Touring", "XL", "28", "27", ""),
		b(Trekking, "Cube", "Touring", "S", "28", "27", ""),
		b(Mountainbike, "Scott", "Aspect 950", "M", "29", "12", ""),
		b(Mountainbike, "Scott", "Aspect 950", "L", "29", "12", ""),
		b(Mountainbike, "Canyon", "Spectral", "M", "29", "12", ""),
		b(EBike, "Haibike", "Trekking 6", "M", "28", "10", "625"),
		b(EBike, "Haibike", "Trekking 6", "L", "28", "10", "625"),
		b(EBike, "Riese & Müller", "Charger 4", "M", "27,5", "5", "750"),
		b(EBike, "Cube", "Reaction Hybrid", "L", "29", "12", "625"),
		b(Rennrad, "Specialized", "Allez", "M", "28", "16", ""),
		b(Rennrad, "Canyon", "Endurace 7", "L", "28", "22", ""),
		b(Kinder, "Woom", "4", "XS", "20", "8", ""),
		b(Kinder, "Woom", "5", "S", "24", "8", ""),
		b(Kinder, "Puky", "Cyke 16", "XS", "16", "1", ""),
		b(Kinder, "Woom", "5", "S", "24", "8", ""),
	}
}
