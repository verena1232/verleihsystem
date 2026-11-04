package verleih

import (
	"errors"
	"fmt"
)

// Fehlerarten, die der Handler auf HTTP-Statuscodes abbildet.
var (
	ErrValidierung   = errors.New("ungültige Eingabe")
	ErrNichtGefunden = errors.New("nicht gefunden")
	ErrKonflikt      = errors.New("Konflikt")
)

// validierung erzeugt einen Validierungsfehler mit lesbarer Meldung.
func validierung(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrValidierung, fmt.Sprintf(format, args...))
}

// Konflikt erzeugt einen Konfliktfehler mit lesbarer Meldung.
func Konflikt(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrKonflikt, fmt.Sprintf(format, args...))
}

// NichtGefunden erzeugt einen 404-Fehler mit lesbarer Meldung.
func NichtGefunden(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrNichtGefunden, fmt.Sprintf(format, args...))
}
