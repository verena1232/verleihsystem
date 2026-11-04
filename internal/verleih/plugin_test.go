package verleih_test

import (
	"errors"
	"testing"
	"time"

	"fahrradverleih/internal/plugins/fahrrad"
	"fahrradverleih/internal/plugins/ski"
	"fahrradverleih/internal/verleih"
)

func registry(t *testing.T, plugins ...verleih.Plugin) *verleih.Registry {
	t.Helper()
	r := &verleih.Registry{}
	for _, p := range plugins {
		if err := r.Registrieren(p); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

func TestDoppeltesPluginAbgelehnt(t *testing.T) {
	r := registry(t, fahrrad.Plugin{})
	if err := r.Registrieren(fahrrad.Plugin{}); err == nil {
		t.Error("gleiches Plugin zweimal sollte Fehler sein")
	}
}

func TestFramworkPrueftFelderGenerisch(t *testing.T) {
	r := registry(t, fahrrad.Plugin{})
	basis := verleih.Basis{Art: "fahrrad", Nummer: "f-1", Kategorie: fahrrad.City}

	faelle := []struct {
		name string
		m    verleih.Merkmale
	}{
		{"Pflichtfeld Marke fehlt", verleih.Merkmale{"rahmen": "M", "laufrad": "28"}},
		{"Auswahl ungültig", verleih.Merkmale{"marke": "KTM", "rahmen": "XXL", "laufrad": "28"}},
		{"Zahl außerhalb", verleih.Merkmale{"marke": "KTM", "rahmen": "M", "laufrad": "28", "gaenge": "99"}},
		{"keine Zahl", verleih.Merkmale{"marke": "KTM", "rahmen": "M", "laufrad": "28", "gaenge": "viele"}},
	}
	for _, f := range faelle {
		if _, err := r.Erzeugen(basis, f.m); !errors.Is(err, verleih.ErrValidierung) {
			t.Errorf("%s: Validierungsfehler erwartet, war %v", f.name, err)
		}
	}

	item, err := r.Erzeugen(basis, verleih.Merkmale{"marke": "KTM", "rahmen": "M", "laufrad": "28", "gaenge": "8", "akku": "500"})
	if err != nil {
		t.Fatal(err)
	}
	if item.Basis().Nummer != "F-1" {
		t.Errorf("Nummer sollte normalisiert werden, war %q", item.Basis().Nummer)
	}
	if _, ok := item.Merkmale()["akku"]; ok {
		t.Error("Akku ist nur für E-Bikes und sollte bei City verworfen werden")
	}
}

func TestEBikeBrauchtAkku(t *testing.T) {
	r := registry(t, fahrrad.Plugin{})
	b := verleih.Basis{Art: "fahrrad", Nummer: "F-2", Kategorie: fahrrad.EBike}
	if _, err := r.Erzeugen(b, verleih.Merkmale{"marke": "Haibike", "rahmen": "M", "laufrad": "28"}); err == nil {
		t.Error("E-Bike ohne Akku sollte abgelehnt werden")
	}
	item, err := r.Erzeugen(b, verleih.Merkmale{"marke": "Haibike", "rahmen": "M", "laufrad": "28", "akku": "625"})
	if err != nil {
		t.Fatal(err)
	}
	if got := item.Kurzinfo(); got != "Rahmen M · 28″ · 625 Wh" {
		t.Errorf("Kurzinfo = %q", got)
	}
}

func TestPluginEigeneRegel(t *testing.T) {
	r := registry(t, fahrrad.Plugin{})
	b := verleih.Basis{Art: "fahrrad", Nummer: "F-3", Kategorie: fahrrad.Kinder}
	if _, err := r.Erzeugen(b, verleih.Merkmale{"marke": "Woom", "rahmen": "S", "laufrad": "28"}); !errors.Is(err, verleih.ErrValidierung) {
		t.Errorf("Kinderrad mit 28 Zoll sollte abgelehnt werden, war %v", err)
	}
}

func TestUnbekannteArtUndKategorie(t *testing.T) {
	r := registry(t, fahrrad.Plugin{})
	if _, err := r.Erzeugen(verleih.Basis{Art: "boot", Nummer: "B-1"}, nil); err == nil {
		t.Error("unbekannte Art sollte Fehler sein")
	}
	if _, err := r.Erzeugen(verleih.Basis{Art: "fahrrad", Nummer: "F-1", Kategorie: "Allround"}, nil); err == nil {
		t.Error("Ski-Kategorie bei Fahrrad sollte Fehler sein")
	}
}

func TestZweitesPluginOhneFrameworkAenderung(t *testing.T) {
	r := registry(t, fahrrad.Plugin{}, ski.Plugin{})
	item, err := r.Erzeugen(verleih.Basis{Art: "ski", Nummer: "S-1", Kategorie: "Race"},
		verleih.Merkmale{"marke": "Atomic", "laenge": "175"})
	if err != nil {
		t.Fatal(err)
	}
	if item.Kurzinfo() != "175 cm" || len(r.Alle()) != 2 {
		t.Errorf("Ski-Plugin funktioniert nicht: %q", item.Kurzinfo())
	}
}

func TestBerechne(t *testing.T) {
	r := verleih.Preisregel{StundenpreisCent: 300, TagespreisCent: 1500}
	von := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	for dauer, cent := range map[time.Duration]int{
		time.Hour:                  300,
		time.Hour + time.Minute:    600,
		6 * time.Hour:              1500, // gedeckelt
		26 * time.Hour:             2100, // 15 € + 2 × 3 €
		72 * time.Hour:             4500,
		26*time.Hour + time.Second: 2400,
	} {
		if got := r.Berechne(von, von.Add(dauer)); got != cent {
			t.Errorf("%v: %d, erwartet %d", dauer, got, cent)
		}
	}
}
