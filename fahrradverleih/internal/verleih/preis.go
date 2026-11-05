package verleih

import "time"

// Preisregel legt fest, was eine Kategorie einer Artikelart kostet.
type Preisregel struct {
	Art              string `json:"art"`
	Kategorie        string `json:"kategorie"`
	StundenpreisCent int    `json:"stundenpreisCent"`
	TagespreisCent   int    `json:"tagespreisCent"` // Höchstpreis je 24 h, 0 = kein Deckel
}

// Schluessel identifiziert eine Regel eindeutig.
func (r Preisregel) Schluessel() string { return PreisSchluessel(r.Art, r.Kategorie) }

// PreisSchluessel baut den Schlüssel "art/kategorie".
func PreisSchluessel(art, kategorie string) string { return art + "/" + kategorie }

// Validieren prüft, ob die Regel sinnvoll ist.
func (r Preisregel) Validieren() error {
	switch {
	case r.StundenpreisCent <= 0:
		return validierung("Stundenpreis muss größer als 0 sein")
	case r.TagespreisCent < 0:
		return validierung("Tagespreis darf nicht negativ sein")
	case r.TagespreisCent > 0 && r.TagespreisCent < r.StundenpreisCent:
		return validierung("Tagespreis muss mindestens so hoch sein wie der Stundenpreis")
	}
	return nil
}

// AngefangeneStunden rundet eine Dauer auf volle Stunden auf.
func AngefangeneStunden(d time.Duration) int {
	if d <= 0 {
		return 0
	}
	return int((d + time.Hour - 1) / time.Hour)
}

// Berechne liefert den Preis in Cent für den Zeitraum [von, bis).
//
//	Preis = volleTage × Tagespreis + min(restStunden × Stundenpreis, Tagespreis)
func (r Preisregel) Berechne(von, bis time.Time) int {
	stunden := AngefangeneStunden(bis.Sub(von))
	if r.TagespreisCent == 0 {
		return stunden * r.StundenpreisCent
	}
	tage, rest := stunden/24, stunden%24
	return tage*r.TagespreisCent + min(rest*r.StundenpreisCent, r.TagespreisCent)
}
