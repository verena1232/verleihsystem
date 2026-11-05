// Fahrrad-Verleih – Prototyp (Aufgabe 1 c), gebaut auf dem Verleih-Framework (d).
// Start: go run .   → http://localhost:8080
package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"fahrradverleih/internal/handler"
	"fahrradverleih/internal/plugins/fahrrad"
	"fahrradverleih/internal/plugins/ski"
	"fahrradverleih/internal/store"
	"fahrradverleih/internal/verleih"
)

//go:embed web
var webDateien embed.FS

// verfuegbarePlugins: neue Artikelart = eine Zeile hier + ein Paket unter internal/plugins.
var verfuegbarePlugins = map[string]verleih.Plugin{
	"fahrrad": fahrrad.Plugin{},
	"ski":     ski.Plugin{},
}

func main() {
	// Welche Plugins aktiv sind: PLUGINS=fahrrad,ski  (Standard: nur fahrrad)
	aktiv := os.Getenv("PLUGINS")
	if aktiv == "" {
		aktiv = "fahrrad"
	}
	reg := &verleih.Registry{}
	for _, art := range strings.Split(aktiv, ",") {
		p, ok := verfuegbarePlugins[strings.TrimSpace(art)]
		if !ok {
			log.Fatalf("unbekanntes Plugin %q", art)
		}
		if err := reg.Registrieren(p); err != nil {
			log.Fatal(err)
		}
	}

	s := store.Neu(reg)
	if os.Getenv("LEER") == "" {
		s.Beispieldaten() // LEER=1 go run .  → ohne Demo-Daten
	}

	titel := "Verleih"
	if alle := reg.Alle(); len(alle) == 1 {
		titel = alle[0].Name() + "-Verleih"
	}

	mux := http.NewServeMux()
	(&handler.API{Store: s, Titel: titel}).Registrieren(mux)
	web, err := fs.Sub(webDateien, "web")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("GET /", http.FileServerFS(web))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("%s läuft auf http://localhost:%s (Plugins: %s)", titel, port, aktiv)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
