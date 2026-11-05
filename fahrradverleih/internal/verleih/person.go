package verleih

import (
	"net/mail"
	"strings"
	"time"
)

// AusweisTyp: welche Art von Ausweis hinterlegt ist.
type AusweisTyp string

const (
	Personalausweis AusweisTyp = "Personalausweis"
	Reisepass       AusweisTyp = "Reisepass"
	Fuehrerschein   AusweisTyp = "Führerschein"
)

// AusweisTypen in Anzeigereihenfolge.
var AusweisTypen = []AusweisTyp{Personalausweis, Reisepass, Fuehrerschein}

// Gueltig prüft, ob der Typ bekannt ist.
func (t AusweisTyp) Gueltig() bool {
	for _, x := range AusweisTypen {
		if x == t {
			return true
		}
	}
	return false
}

// MaxAusweisBytes ist die maximale Größe eines Ausweis-Uploads.
const MaxAusweisBytes = 5 << 20

// ErlaubteAusweisTypen sind die erlaubten MIME-Typen für Uploads.
var ErlaubteAusweisTypen = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/webp":      true,
	"application/pdf": true,
}

// AusweisDatei ist ein hochgeladenes Bild/PDF eines Ausweises.
type AusweisDatei struct {
	Name        string
	MimeType    string
	Daten       []byte
	Hochgeladen time.Time
}

// Ausweis einer Person.
type Ausweis struct {
	Typ    AusweisTyp    `json:"typ"`
	Nummer string        `json:"nummer"`
	Datei  *AusweisDatei `json:"-"`
}

// Person ist eine Kundin / ein Kunde.
type Person struct {
	ID       int     `json:"id"`
	Vorname  string  `json:"vorname"`
	Nachname string  `json:"nachname"`
	Email    string  `json:"email"`
	Telefon  string  `json:"telefon"`
	Ausweis  Ausweis `json:"ausweis"`
}

// Name liefert "Vorname Nachname".
func (p Person) Name() string {
	return strings.TrimSpace(p.Vorname + " " + p.Nachname)
}

// Validieren prüft Pflichtfelder und Formate.
func (p *Person) Validieren() error {
	p.Vorname = strings.TrimSpace(p.Vorname)
	p.Nachname = strings.TrimSpace(p.Nachname)
	p.Email = strings.TrimSpace(p.Email)
	p.Telefon = strings.TrimSpace(p.Telefon)
	p.Ausweis.Nummer = strings.ToUpper(strings.TrimSpace(p.Ausweis.Nummer))

	switch {
	case p.Vorname == "":
		return validierung("Vorname fehlt")
	case p.Nachname == "":
		return validierung("Nachname fehlt")
	case p.Email != "" && !gueltigeEmail(p.Email):
		return validierung("E-Mail-Adresse ist ungültig")
	case !p.Ausweis.Typ.Gueltig():
		return validierung("bitte Ausweistyp wählen")
	case len(p.Ausweis.Nummer) < 4:
		return validierung("Ausweisnummer fehlt oder ist zu kurz")
	}
	return nil
}

func gueltigeEmail(s string) bool {
	a, err := mail.ParseAddress(s)
	return err == nil && a.Address == s
}

// PruefeAusweisDatei kontrolliert Größe und Typ eines Uploads.
func PruefeAusweisDatei(mimeType string, groesse int) error {
	if groesse == 0 {
		return validierung("Datei ist leer")
	}
	if groesse > MaxAusweisBytes {
		return validierung("Datei ist größer als 5 MB")
	}
	if !ErlaubteAusweisTypen[mimeType] {
		return validierung("nur JPG, PNG, WebP oder PDF erlaubt")
	}
	return nil
}
