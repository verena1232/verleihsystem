// Package handler stellt die JSON-REST-API unter /api bereit.
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"fahrradverleih/internal/store"
	"fahrradverleih/internal/verleih"
)

// API verbindet HTTP mit dem Store. Sie kennt keine konkreten Artikel.
type API struct {
	Store *store.Memory
	Titel string // z. B. "Fahrrad-Verleih"
}

// Registrieren hängt alle Routen an den Mux.
func (api *API) Registrieren(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/meta", api.meta)
	mux.HandleFunc("GET /api/dashboard", api.dashboard)

	mux.HandleFunc("GET /api/artikel", api.artikelListe)
	mux.HandleFunc("POST /api/artikel", api.artikelSpeichern)
	mux.HandleFunc("PUT /api/artikel/{id}", api.artikelSpeichern)
	mux.HandleFunc("DELETE /api/artikel/{id}", api.artikelLoeschen)

	mux.HandleFunc("GET /api/personen", api.personen)
	mux.HandleFunc("POST /api/personen", api.personSpeichern)
	mux.HandleFunc("PUT /api/personen/{id}", api.personSpeichern)
	mux.HandleFunc("DELETE /api/personen/{id}", api.personLoeschen)
	mux.HandleFunc("GET /api/personen/{id}/ausweis", api.ausweisAnzeigen)
	mux.HandleFunc("POST /api/personen/{id}/ausweis", api.ausweisHochladen)
	mux.HandleFunc("DELETE /api/personen/{id}/ausweis", api.ausweisEntfernen)

	mux.HandleFunc("GET /api/preise", api.preise)
	mux.HandleFunc("PUT /api/preise/{art}/{kategorie}", api.preisSetzen)

	mux.HandleFunc("GET /api/verfuegbar", api.verfuegbar)
	mux.HandleFunc("GET /api/ausleihen", api.ausleihen)
	mux.HandleFunc("POST /api/ausleihen", api.ausleiheAnlegen)
	mux.HandleFunc("POST /api/ausleihen/{id}/rueckgabe", api.zurueckgeben)
	mux.HandleFunc("POST /api/ausleihen/{id}/storno", api.stornieren)
}

// ---------- Hilfsfunktionen ----------

func schreibeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("JSON schreiben: %v", err)
	}
}

// schreibeFehler bildet Domain-Fehler auf HTTP-Status ab.
func schreibeFehler(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, verleih.ErrValidierung):
		status = http.StatusBadRequest
	case errors.Is(err, verleih.ErrNichtGefunden):
		status = http.StatusNotFound
	case errors.Is(err, verleih.ErrKonflikt):
		status = http.StatusConflict
	default:
		log.Printf("interner Fehler: %v", err)
	}
	// Präfix wie "ungültige Eingabe: " für die Anzeige weglassen
	msg := err.Error()
	if i := strings.Index(msg, ": "); i >= 0 && status != http.StatusInternalServerError {
		msg = msg[i+2:]
	}
	schreibeJSON(w, status, map[string]string{"fehler": msg})
}

func leseJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("%w: Anfrage konnte nicht gelesen werden (%v)", verleih.ErrValidierung, err)
	}
	return nil
}

func pfadID(r *http.Request) (int, error) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("%w: ungültige ID", verleih.ErrValidierung)
	}
	return id, nil
}

func parseZeit(s, name string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %s ist kein gültiger Zeitpunkt", verleih.ErrValidierung, name)
	}
	return t, nil
}

// ---------- Meta & Dashboard ----------

// pluginDTO beschreibt ein Plugin für das Frontend.
type pluginDTO struct {
	Art        string         `json:"art"`
	Name       string         `json:"name"`
	Mehrzahl   string         `json:"mehrzahl"`
	Symbol     string         `json:"symbol"`
	Praefix    string         `json:"praefix"`
	Kategorien []string       `json:"kategorien"`
	Felder     []verleih.Feld `json:"felder"`
}

func (api *API) meta(w http.ResponseWriter, _ *http.Request) {
	plugins := []pluginDTO{}
	for _, p := range api.Store.Registry.Alle() {
		plugins = append(plugins, pluginDTO{p.Art(), p.Name(), p.Mehrzahl(), p.Symbol(), p.Praefix(), p.Kategorien(), p.Felder()})
	}
	schreibeJSON(w, http.StatusOK, map[string]any{
		"titel":        api.Titel,
		"plugins":      plugins,
		"ausweisTypen": verleih.AusweisTypen,
		"maxUpload":    verleih.MaxAusweisBytes,
	})
}

// ArtikelDTO ist die JSON-Form eines RentalItem.
type ArtikelDTO struct {
	verleih.Basis
	Bezeichnung string            `json:"bezeichnung"`
	Kurzinfo    string            `json:"kurzinfo"`
	Merkmale    map[string]string `json:"merkmale"`
	Status      string            `json:"status,omitempty"` // verfügbar | ausgeliehen | reserviert
	PreisCent   int               `json:"preisCent,omitempty"`
}

func artikelDTO(item verleih.RentalItem) ArtikelDTO {
	return ArtikelDTO{Basis: item.Basis(), Bezeichnung: item.Bezeichnung(), Kurzinfo: item.Kurzinfo(), Merkmale: item.Merkmale()}
}

// AusleiheDTO ergänzt eine Ausleihe um Status und lesbare Namen.
type AusleiheDTO struct {
	verleih.Ausleihe
	Status      verleih.Status `json:"status"`
	ArtikelText string         `json:"artikelText"`
	Art         string         `json:"art"`
	Kategorie   string         `json:"kategorie"`
	PersonName  string         `json:"personName"`
}

func (api *API) ausleiheDTOs() []AusleiheDTO {
	jetzt := api.Store.Jetzt()
	artikel := map[int]verleih.RentalItem{}
	for _, a := range api.Store.ArtikelListe() {
		artikel[a.Basis().ID] = a
	}
	pers := map[int]verleih.Person{}
	for _, p := range api.Store.Personen() {
		pers[p.ID] = p
	}
	liste := []AusleiheDTO{}
	for _, a := range api.Store.Ausleihen() {
		d := AusleiheDTO{Ausleihe: a, Status: a.Status(jetzt), ArtikelText: "(gelöscht)", PersonName: "(gelöscht)"}
		if item, ok := artikel[a.ArtikelID]; ok {
			b := item.Basis()
			d.ArtikelText = b.Nummer + " " + item.Bezeichnung() + " · " + item.Kurzinfo()
			d.Art, d.Kategorie = b.Art, b.Kategorie
		}
		if p, ok := pers[a.PersonID]; ok {
			d.PersonName = p.Name()
		}
		liste = append(liste, d)
	}
	return liste
}

func (api *API) dashboard(w http.ResponseWriter, r *http.Request) {
	// Tagesende aus Sicht des Browsers (optional ?tagesende=RFC3339)
	tagesende := api.Store.Jetzt().Truncate(24 * time.Hour).Add(24 * time.Hour)
	if s := r.URL.Query().Get("tagesende"); s != "" {
		if t, err := parseZeit(s, "tagesende"); err == nil {
			tagesende = t
		}
	}
	offen := []AusleiheDTO{}
	for _, a := range api.ausleiheDTOs() {
		if a.Offen() {
			offen = append(offen, a)
		}
	}
	schreibeJSON(w, http.StatusOK, map[string]any{
		"kennzahlen": api.Store.Dashboard(tagesende),
		"offen":      offen,
	})
}

// ---------- Artikel ----------

func (api *API) artikelListe(w http.ResponseWriter, _ *http.Request) {
	jetzt := api.Store.Jetzt()
	status := map[int]string{}
	for _, a := range api.Store.Ausleihen() {
		switch a.Status(jetzt) {
		case verleih.StatusAktiv, verleih.StatusUeberfaellig:
			status[a.ArtikelID] = "ausgeliehen"
		case verleih.StatusReserviert:
			if status[a.ArtikelID] == "" {
				status[a.ArtikelID] = "reserviert"
			}
		}
	}
	liste := []ArtikelDTO{}
	for _, item := range api.Store.ArtikelListe() {
		d := artikelDTO(item)
		d.Status = status[d.ID]
		if d.Status == "" {
			d.Status = "verfügbar"
		}
		liste = append(liste, d)
	}
	schreibeJSON(w, http.StatusOK, liste)
}

func (api *API) artikelSpeichern(w http.ResponseWriter, r *http.Request) {
	var req struct {
		verleih.Basis
		Merkmale verleih.Merkmale `json:"merkmale"`
	}
	if err := leseJSON(r, &req); err != nil {
		schreibeFehler(w, err)
		return
	}
	req.ID = 0
	status := http.StatusCreated
	if r.Method == http.MethodPut {
		id, err := pfadID(r)
		if err != nil {
			schreibeFehler(w, err)
			return
		}
		req.ID, status = id, http.StatusOK
	}
	item, err := api.Store.ArtikelSpeichern(req.Basis, req.Merkmale)
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	schreibeJSON(w, status, artikelDTO(item))
}

func (api *API) artikelLoeschen(w http.ResponseWriter, r *http.Request) {
	id, err := pfadID(r)
	if err == nil {
		err = api.Store.ArtikelLoeschen(id)
	}
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- Personen ----------

// PersonDTO zeigt zusätzlich, ob eine Ausweisdatei vorhanden ist.
type PersonDTO struct {
	verleih.Person
	AusweisDatei  string `json:"ausweisDatei,omitempty"`
	AusweisMime   string `json:"ausweisMime,omitempty"`
	OffeneAusleih int    `json:"offeneAusleihen"`
}

func personDTO(p verleih.Person, offen int) PersonDTO {
	d := PersonDTO{Person: p, OffeneAusleih: offen}
	if p.Ausweis.Datei != nil {
		d.AusweisDatei = p.Ausweis.Datei.Name
		d.AusweisMime = p.Ausweis.Datei.MimeType
	}
	return d
}

func (api *API) personen(w http.ResponseWriter, _ *http.Request) {
	offen := map[int]int{}
	for _, a := range api.Store.Ausleihen() {
		if a.Offen() {
			offen[a.PersonID]++
		}
	}
	liste := []PersonDTO{}
	for _, p := range api.Store.Personen() {
		liste = append(liste, personDTO(p, offen[p.ID]))
	}
	schreibeJSON(w, http.StatusOK, liste)
}

func (api *API) personSpeichern(w http.ResponseWriter, r *http.Request) {
	var p verleih.Person
	if err := leseJSON(r, &p); err != nil {
		schreibeFehler(w, err)
		return
	}
	p.ID = 0
	status := http.StatusCreated
	if r.Method == http.MethodPut {
		id, err := pfadID(r)
		if err != nil {
			schreibeFehler(w, err)
			return
		}
		p.ID, status = id, http.StatusOK
	}
	gespeichert, err := api.Store.PersonSpeichern(p)
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	schreibeJSON(w, status, personDTO(gespeichert, 0))
}

func (api *API) personLoeschen(w http.ResponseWriter, r *http.Request) {
	id, err := pfadID(r)
	if err == nil {
		err = api.Store.PersonLoeschen(id)
	}
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ausweisHochladen nimmt ein multipart-Formular mit Feld "datei" an.
func (api *API) ausweisHochladen(w http.ResponseWriter, r *http.Request) {
	id, err := pfadID(r)
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, verleih.MaxAusweisBytes+64<<10)
	datei, kopf, err := r.FormFile("datei")
	if err != nil {
		schreibeFehler(w, fmt.Errorf("%w: keine Datei oder Datei größer als 5 MB", verleih.ErrValidierung))
		return
	}
	defer datei.Close()
	daten, err := io.ReadAll(io.LimitReader(datei, verleih.MaxAusweisBytes+1))
	if err != nil {
		schreibeFehler(w, fmt.Errorf("Datei lesen: %w", err))
		return
	}
	// Typ aus dem Inhalt bestimmen, nicht aus dem Dateinamen
	mime := http.DetectContentType(daten)
	if i := strings.Index(mime, ";"); i >= 0 {
		mime = mime[:i]
	}
	err = api.Store.AusweisDateiSetzen(id, &verleih.AusweisDatei{Name: kopf.Filename, MimeType: mime, Daten: daten})
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	p, _ := api.Store.Person(id)
	schreibeJSON(w, http.StatusOK, personDTO(p, 0))
}

func (api *API) ausweisAnzeigen(w http.ResponseWriter, r *http.Request) {
	id, err := pfadID(r)
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	p, err := api.Store.Person(id)
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	d := p.Ausweis.Datei
	if d == nil {
		schreibeFehler(w, verleih.NichtGefunden("kein Ausweis hochgeladen"))
		return
	}
	w.Header().Set("Content-Type", d.MimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", d.Name))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(d.Daten)
}

func (api *API) ausweisEntfernen(w http.ResponseWriter, r *http.Request) {
	id, err := pfadID(r)
	if err == nil {
		err = api.Store.AusweisDateiSetzen(id, nil)
	}
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- Preise ----------

func (api *API) preise(w http.ResponseWriter, _ *http.Request) {
	schreibeJSON(w, http.StatusOK, api.Store.Preise())
}

func (api *API) preisSetzen(w http.ResponseWriter, r *http.Request) {
	var p verleih.Preisregel
	if err := leseJSON(r, &p); err != nil {
		schreibeFehler(w, err)
		return
	}
	p.Art, p.Kategorie = r.PathValue("art"), r.PathValue("kategorie")
	if err := api.Store.PreisSetzen(p); err != nil {
		schreibeFehler(w, err)
		return
	}
	schreibeJSON(w, http.StatusOK, p)
}

// ---------- Ausleihen ----------

// verfuegbar: GET /api/verfuegbar?von=...&bis=... → freie Artikel mit Preis.
func (api *API) verfuegbar(w http.ResponseWriter, r *http.Request) {
	von, err := parseZeit(r.URL.Query().Get("von"), "Beginn")
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	bis, err := parseZeit(r.URL.Query().Get("bis"), "Ende")
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	// Zeitraum validieren (gleiche Regeln wie beim Buchen)
	if _, err := verleih.NeueAusleihe(0, 0, von, bis, api.Store.Jetzt(), verleih.Preisregel{}); err != nil {
		schreibeFehler(w, err)
		return
	}
	liste := []ArtikelDTO{}
	for _, a := range api.Store.Verfuegbar(von, bis) {
		d := artikelDTO(a.Artikel)
		d.PreisCent = a.PreisCent
		liste = append(liste, d)
	}
	schreibeJSON(w, http.StatusOK, map[string]any{
		"stunden": verleih.AngefangeneStunden(bis.Sub(von)),
		"artikel": liste,
	})
}

func (api *API) ausleihen(w http.ResponseWriter, _ *http.Request) {
	schreibeJSON(w, http.StatusOK, api.ausleiheDTOs())
}

func (api *API) ausleiheAnlegen(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ArtikelID int       `json:"artikelId"`
		PersonID  int       `json:"personId"`
		Beginn    time.Time `json:"beginn"`
		Ende      time.Time `json:"ende"`
	}
	if err := leseJSON(r, &req); err != nil {
		schreibeFehler(w, err)
		return
	}
	a, err := api.Store.AusleiheAnlegen(req.ArtikelID, req.PersonID, req.Beginn, req.Ende)
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	schreibeJSON(w, http.StatusCreated, a)
}

func (api *API) zurueckgeben(w http.ResponseWriter, r *http.Request) {
	api.aendern(w, r, api.Store.Zurueckgeben)
}

func (api *API) stornieren(w http.ResponseWriter, r *http.Request) {
	api.aendern(w, r, api.Store.Stornieren)
}

func (api *API) aendern(w http.ResponseWriter, r *http.Request, f func(int) (verleih.Ausleihe, error)) {
	id, err := pfadID(r)
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	a, err := f(id)
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	schreibeJSON(w, http.StatusOK, a)
}
