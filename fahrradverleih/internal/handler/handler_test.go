package handler

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"fahrradverleih/internal/plugins/fahrrad"
	"fahrradverleih/internal/store"
	"fahrradverleih/internal/verleih"
)

func testServer() *httptest.Server {
	reg := &verleih.Registry{}
	_ = reg.Registrieren(fahrrad.Plugin{})
	s := store.Neu(reg)
	s.Beispieldaten()
	mux := http.NewServeMux()
	(&API{Store: s, Titel: "Fahrrad-Verleih"}).Registrieren(mux)
	return httptest.NewServer(mux)
}

func upload(t *testing.T, url, name string, daten []byte) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("datei", name)
	_, _ = fw.Write(daten)
	_ = mw.Close()
	res, err := http.Post(url, mw.FormDataContentType(), &buf)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// minimales gültiges PNG (1×1 Pixel)
var pngBytes = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89\x00\x00\x00\rIDATx\x9cc\xf8\x0f\x00\x00\x01\x01\x00\x05\x18\xd8N\x00\x00\x00\x00IEND\xaeB`\x82")

func TestAusweisUpload(t *testing.T) {
	srv := testServer()
	defer srv.Close()
	url := srv.URL + "/api/personen/21/ausweis"

	if res := upload(t, url, "ausweis.png", pngBytes); res.StatusCode != http.StatusOK {
		t.Fatalf("PNG-Upload: Status %d", res.StatusCode)
	}
	res, _ := http.Get(url)
	if res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("Abruf: %d %s", res.StatusCode, res.Header.Get("Content-Type"))
	}
	// Textdatei mit .png-Endung → Typ wird am Inhalt erkannt und abgelehnt (A-6)
	if res := upload(t, url, "fake.png", []byte("hallo welt")); res.StatusCode != http.StatusBadRequest {
		t.Errorf("Textdatei sollte abgelehnt werden, Status %d", res.StatusCode)
	}
	// zu groß (A-6)
	gross := append(append([]byte{}, pngBytes...), make([]byte, 6<<20)...)
	if res := upload(t, url, "gross.png", gross); res.StatusCode != http.StatusBadRequest {
		t.Errorf("> 5 MB sollte abgelehnt werden, Status %d", res.StatusCode)
	}
}

func TestPersonOhneAusweisAbgelehnt(t *testing.T) {
	srv := testServer()
	defer srv.Close()
	res, _ := http.Post(srv.URL+"/api/personen", "application/json",
		strings.NewReader(`{"vorname":"Max","nachname":"Muster","ausweis":{"typ":"","nummer":""}}`))
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("Status %d, erwartet 400", res.StatusCode)
	}
	var body map[string]string
	_ = json.NewDecoder(res.Body).Decode(&body)
	if !strings.Contains(body["fehler"], "Ausweistyp") {
		t.Errorf("Fehlermeldung sollte Ausweistyp nennen: %q", body["fehler"])
	}
}

func TestDashboard(t *testing.T) {
	srv := testServer()
	defer srv.Close()
	res, _ := http.Get(srv.URL + "/api/dashboard")
	var d struct {
		Kennzahlen store.Kennzahlen `json:"kennzahlen"`
	}
	_ = json.NewDecoder(res.Body).Decode(&d)
	if d.Kennzahlen.Gesamt != 20 || d.Kennzahlen.Verfuegbar != 17 {
		t.Errorf("Dashboard: %+v", d.Kennzahlen)
	}
}

func TestArtikelAnlegenUeberAPI(t *testing.T) {
	srv := testServer()
	defer srv.Close()
	res, _ := http.Post(srv.URL+"/api/artikel", "application/json", strings.NewReader(
		`{"art":"fahrrad","nummer":"F-100","kategorie":"E-Bike","merkmale":{"marke":"Cube","rahmen":"L","laufrad":"29","akku":"750"}}`))
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("Status %d", res.StatusCode)
	}
	var a ArtikelDTO
	_ = json.NewDecoder(res.Body).Decode(&a)
	if a.Kurzinfo != "Rahmen L · 29″ · 750 Wh" || a.Merkmale["akku"] != "750" {
		t.Errorf("Antwort: %+v", a)
	}
}

func TestMetaLiefertPluginFelder(t *testing.T) {
	srv := testServer()
	defer srv.Close()
	res, _ := http.Get(srv.URL + "/api/meta")
	var m struct {
		Titel   string `json:"titel"`
		Plugins []struct {
			Art    string         `json:"art"`
			Felder []verleih.Feld `json:"felder"`
		} `json:"plugins"`
	}
	_ = json.NewDecoder(res.Body).Decode(&m)
	if m.Titel != "Fahrrad-Verleih" || len(m.Plugins) != 1 || len(m.Plugins[0].Felder) != 6 {
		t.Errorf("Meta: %+v", m)
	}
}
