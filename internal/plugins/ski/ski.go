// Package ski zeigt, dass das Framework erweiterbar ist (Aufgabe 1 d):
// ein zweites Plugin, ohne eine Zeile im Framework zu ändern.
// Aktivieren mit:  PLUGINS=fahrrad,ski go run .
package ski

import (
	"fmt"
	"strings"

	"fahrradverleih/internal/verleih"
)

// Ski ist ein RentalItem.
type Ski struct {
	basis    verleih.Basis
	Marke    string
	Modell   string
	LaengeCm int
}

func (s *Ski) Basis() verleih.Basis { return s.basis }
func (s *Ski) Bezeichnung() string  { return strings.TrimSpace(s.Marke + " " + s.Modell) }
func (s *Ski) Kurzinfo() string     { return fmt.Sprintf("%d cm", s.LaengeCm) }
func (s *Ski) Merkmale() map[string]string {
	return map[string]string{"marke": s.Marke, "modell": s.Modell, "laenge": fmt.Sprint(s.LaengeCm)}
}

// Plugin für Ski.
type Plugin struct{}

func (Plugin) Art() string      { return "ski" }
func (Plugin) Name() string     { return "Ski" }
func (Plugin) Mehrzahl() string { return "Ski" }
func (Plugin) Symbol() string   { return "⛷" }
func (Plugin) Praefix() string  { return "S-" }
func (Plugin) Kategorien() []string {
	return []string{"Kinder", "Allround", "Race", "Freeride", "Touren"}
}

func (Plugin) Felder() []verleih.Feld {
	return []verleih.Feld{
		{Name: "marke", Label: "Marke", Typ: verleih.FeldText, Pflicht: true},
		{Name: "modell", Label: "Modell", Typ: verleih.FeldText},
		{Name: "laenge", Label: "Länge", Typ: verleih.FeldZahl, Einheit: "cm", Pflicht: true, Min: 70, Max: 220},
	}
}

func (Plugin) Standardpreise() []verleih.Preisregel {
	r := func(k string, h, t int) verleih.Preisregel {
		return verleih.Preisregel{Art: "ski", Kategorie: k, StundenpreisCent: h, TagespreisCent: t}
	}
	return []verleih.Preisregel{r("Kinder", 250, 1500), r("Allround", 400, 2500), r("Race", 600, 3500),
		r("Freeride", 600, 3800), r("Touren", 700, 4200)}
}

func (Plugin) Neu(b verleih.Basis, m verleih.Merkmale) (verleih.RentalItem, error) {
	return &Ski{basis: b, Marke: m.Text("marke"), Modell: m.Text("modell"), LaengeCm: m.Zahl("laenge")}, nil
}

func (Plugin) Beispiele() []verleih.Beispiel {
	b := func(kat, marke, modell, laenge string) verleih.Beispiel {
		return verleih.Beispiel{Kategorie: kat, Merkmale: verleih.Merkmale{"marke": marke, "modell": modell, "laenge": laenge}}
	}
	return []verleih.Beispiel{
		b("Allround", "Head", "Supershape", "170"), b("Race", "Atomic", "Redster G9", "175"),
		b("Freeride", "Salomon", "QST 98", "181"), b("Kinder", "Head", "Monster Kid", "107"),
	}
}
