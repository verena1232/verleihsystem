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

	"skiverleih/internal/domain"
	"skiverleih/internal/store"
)

// API verbindet HTTP mit dem Store.
type API struct {
	Store *store.Memory
}

// Registrieren hängt alle Routen an den Mux.
func (api *API) Registrieren(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/meta", api.meta)
	mux.HandleFunc("GET /api/dashboard", api.dashboard)

	mux.HandleFunc("GET /api/ski", api.skiListe)
	mux.HandleFunc("POST /api/ski", api.skiSpeichern)
	mux.HandleFunc("PUT /api/ski/{id}", api.skiSpeichern)
	mux.HandleFunc("DELETE /api/ski/{id}", api.skiLoeschen)

	mux.HandleFunc("GET /api/personen", api.personen)
	mux.HandleFunc("POST /api/personen", api.personSpeichern)
	mux.HandleFunc("PUT /api/personen/{id}", api.personSpeichern)
	mux.HandleFunc("DELETE /api/personen/{id}", api.personLoeschen)
	mux.HandleFunc("GET /api/personen/{id}/ausweis", api.ausweisAnzeigen)
	mux.HandleFunc("POST /api/personen/{id}/ausweis", api.ausweisHochladen)
	mux.HandleFunc("DELETE /api/personen/{id}/ausweis", api.ausweisEntfernen)

	mux.HandleFunc("GET /api/preise", api.preise)
	mux.HandleFunc("PUT /api/preise/{kategorie}", api.preisSetzen)

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
	case errors.Is(err, domain.ErrValidierung):
		status = http.StatusBadRequest
	case errors.Is(err, domain.ErrNichtGefunden):
		status = http.StatusNotFound
	case errors.Is(err, domain.ErrKonflikt):
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
		return fmt.Errorf("%w: Anfrage konnte nicht gelesen werden (%v)", domain.ErrValidierung, err)
	}
	return nil
}

func pfadID(r *http.Request) (int, error) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("%w: ungültige ID", domain.ErrValidierung)
	}
	return id, nil
}

func parseZeit(s, name string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %s ist kein gültiger Zeitpunkt", domain.ErrValidierung, name)
	}
	return t, nil
}

// ---------- Meta & Dashboard ----------

func (api *API) meta(w http.ResponseWriter, _ *http.Request) {
	schreibeJSON(w, http.StatusOK, map[string]any{
		"kategorien":   domain.Kategorien,
		"ausweisTypen": domain.AusweisTypen,
		"maxUpload":    domain.MaxAusweisBytes,
	})
}

// AusleiheDTO ergänzt eine Ausleihe um Status und lesbare Namen.
type AusleiheDTO struct {
	domain.Ausleihe
	Status     domain.Status `json:"status"`
	SkiText    string        `json:"skiText"`
	Kategorie  string        `json:"kategorie"`
	PersonName string        `json:"personName"`
}

func (api *API) ausleiheDTOs() []AusleiheDTO {
	jetzt := api.Store.Jetzt()
	ski := map[int]domain.Ski{}
	for _, s := range api.Store.SkiListe() {
		ski[s.ID] = s
	}
	pers := map[int]domain.Person{}
	for _, p := range api.Store.Personen() {
		pers[p.ID] = p
	}
	liste := []AusleiheDTO{}
	for _, a := range api.Store.Ausleihen() {
		d := AusleiheDTO{Ausleihe: a, Status: a.Status(jetzt), SkiText: "(gelöscht)", PersonName: "(gelöscht)"}
		if s, ok := ski[a.SkiID]; ok {
			d.SkiText = s.Bezeichnung() + fmt.Sprintf(" · %d cm", s.LaengeCm)
			d.Kategorie = string(s.Kategorie)
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

// ---------- Ski ----------

// SkiDTO ergänzt einen Ski um seinen aktuellen Status (A-3).
type SkiDTO struct {
	domain.Ski
	Status string `json:"status"` // verfügbar | ausgeliehen | reserviert
}

func (api *API) skiListe(w http.ResponseWriter, _ *http.Request) {
	jetzt := api.Store.Jetzt()
	status := map[int]string{}
	for _, a := range api.Store.Ausleihen() {
		switch a.Status(jetzt) {
		case domain.StatusAktiv, domain.StatusUeberfaellig:
			status[a.SkiID] = "ausgeliehen"
		case domain.StatusReserviert:
			if status[a.SkiID] == "" {
				status[a.SkiID] = "reserviert"
			}
		}
	}
	liste := []SkiDTO{}
	for _, s := range api.Store.SkiListe() {
		st := status[s.ID]
		if st == "" {
			st = "verfügbar"
		}
		liste = append(liste, SkiDTO{Ski: s, Status: st})
	}
	schreibeJSON(w, http.StatusOK, liste)
}

func (api *API) skiSpeichern(w http.ResponseWriter, r *http.Request) {
	var s domain.Ski
	if err := leseJSON(r, &s); err != nil {
		schreibeFehler(w, err)
		return
	}
	s.ID = 0
	status := http.StatusCreated
	if r.Method == http.MethodPut {
		id, err := pfadID(r)
		if err != nil {
			schreibeFehler(w, err)
			return
		}
		s.ID, status = id, http.StatusOK
	}
	gespeichert, err := api.Store.SkiSpeichern(s)
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	schreibeJSON(w, status, gespeichert)
}

func (api *API) skiLoeschen(w http.ResponseWriter, r *http.Request) {
	id, err := pfadID(r)
	if err == nil {
		err = api.Store.SkiLoeschen(id)
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
	domain.Person
	AusweisDatei  string `json:"ausweisDatei,omitempty"`
	AusweisMime   string `json:"ausweisMime,omitempty"`
	OffeneAusleih int    `json:"offeneAusleihen"`
}

func personDTO(p domain.Person, offen int) PersonDTO {
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
	var p domain.Person
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

// ausweisHochladen nimmt ein multipart-Formular mit Feld "datei" an (A-6).
func (api *API) ausweisHochladen(w http.ResponseWriter, r *http.Request) {
	id, err := pfadID(r)
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, domain.MaxAusweisBytes+64<<10)
	datei, kopf, err := r.FormFile("datei")
	if err != nil {
		schreibeFehler(w, fmt.Errorf("%w: keine Datei oder Datei größer als 5 MB", domain.ErrValidierung))
		return
	}
	defer datei.Close()
	daten, err := io.ReadAll(io.LimitReader(datei, domain.MaxAusweisBytes+1))
	if err != nil {
		schreibeFehler(w, fmt.Errorf("Datei lesen: %w", err))
		return
	}
	// Typ aus dem Inhalt bestimmen, nicht aus dem Dateinamen
	mime := http.DetectContentType(daten)
	if i := strings.Index(mime, ";"); i >= 0 {
		mime = mime[:i]
	}
	err = api.Store.AusweisDateiSetzen(id, &domain.AusweisDatei{Name: kopf.Filename, MimeType: mime, Daten: daten})
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
		schreibeFehler(w, domain.NichtGefunden("kein Ausweis hochgeladen"))
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
	var p domain.Preisregel
	if err := leseJSON(r, &p); err != nil {
		schreibeFehler(w, err)
		return
	}
	p.Kategorie = domain.Kategorie(r.PathValue("kategorie"))
	if err := api.Store.PreisSetzen(p); err != nil {
		schreibeFehler(w, err)
		return
	}
	schreibeJSON(w, http.StatusOK, p)
}

// ---------- Ausleihen ----------

// verfuegbar: GET /api/verfuegbar?von=...&bis=... → freie Ski mit Preis (A-8, A-17).
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
	if _, err := domain.NeueAusleihe(0, 0, von, bis, api.Store.Jetzt(), domain.Preisregel{}); err != nil {
		schreibeFehler(w, err)
		return
	}
	type angebot struct {
		domain.Ski
		PreisCent int `json:"preisCent"`
	}
	liste := []angebot{}
	for _, s := range api.Store.Verfuegbar(von, bis) {
		preis, err := api.Store.Angebot(s.ID, von, bis)
		if err != nil {
			continue
		}
		liste = append(liste, angebot{Ski: s, PreisCent: preis})
	}
	schreibeJSON(w, http.StatusOK, map[string]any{
		"stunden": domain.AngefangeneStunden(bis.Sub(von)),
		"ski":     liste,
	})
}

func (api *API) ausleihen(w http.ResponseWriter, _ *http.Request) {
	schreibeJSON(w, http.StatusOK, api.ausleiheDTOs())
}

func (api *API) ausleiheAnlegen(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SkiID    int       `json:"skiId"`
		PersonID int       `json:"personId"`
		Beginn   time.Time `json:"beginn"`
		Ende     time.Time `json:"ende"`
	}
	if err := leseJSON(r, &req); err != nil {
		schreibeFehler(w, err)
		return
	}
	a, err := api.Store.AusleiheAnlegen(req.SkiID, req.PersonID, req.Beginn, req.Ende)
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

func (api *API) aendern(w http.ResponseWriter, r *http.Request, f func(int) (domain.Ausleihe, error)) {
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
