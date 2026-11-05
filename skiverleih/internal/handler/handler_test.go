package handler

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"skiverleih/internal/store"
)

func testServer() *httptest.Server {
	s := store.Neu()
	s.Beispieldaten()
	mux := http.NewServeMux()
	(&API{Store: s}).Registrieren(mux)
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
