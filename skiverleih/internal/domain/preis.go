package domain

import "time"

// Preisregel legt fest, was eine Kategorie kostet (B-3, B-4).
type Preisregel struct {
	Kategorie        Kategorie `json:"kategorie"`
	StundenpreisCent int       `json:"stundenpreisCent"`
	TagespreisCent   int       `json:"tagespreisCent"` // Höchstpreis je 24 h, 0 = kein Deckel
}

// Validieren prüft, ob die Regel sinnvoll ist.
func (r Preisregel) Validieren() error {
	switch {
	case !r.Kategorie.Gueltig():
		return validierung("unbekannte Kategorie %q", r.Kategorie)
	case r.StundenpreisCent <= 0:
		return validierung("Stundenpreis muss größer als 0 sein")
	case r.TagespreisCent < 0:
		return validierung("Tagespreis darf nicht negativ sein")
	case r.TagespreisCent > 0 && r.TagespreisCent < r.StundenpreisCent:
		return validierung("Tagespreis muss mindestens so hoch sein wie der Stundenpreis")
	}
	return nil
}

// AngefangeneStunden rundet eine Dauer auf volle Stunden auf (B-3).
func AngefangeneStunden(d time.Duration) int {
	if d <= 0 {
		return 0
	}
	return int((d + time.Hour - 1) / time.Hour)
}

// Berechne liefert den Preis in Cent für den Zeitraum [von, bis) (B-3, B-4).
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
