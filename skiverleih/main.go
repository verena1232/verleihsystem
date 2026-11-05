// Ski-Verleih – Prototyp (Aufgabe 1 a + b).
// Start: go run .   → http://localhost:8080
package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"

	"skiverleih/internal/handler"
	"skiverleih/internal/store"
)

//go:embed web
var webDateien embed.FS

func main() {
	s := store.Neu()
	if os.Getenv("LEER") == "" {
		s.Beispieldaten() // LEER=1 go run .  → ohne Demo-Daten starten
	}

	mux := http.NewServeMux()
	api := &handler.API{Store: s}
	api.Registrieren(mux)

	web, err := fs.Sub(webDateien, "web")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("GET /", http.FileServerFS(web))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Ski-Verleih läuft auf http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
