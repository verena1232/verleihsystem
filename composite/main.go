package main

import (
	"fmt"
	"log"
)

func must(w *Window, err error) *Window {
	if err != nil {
		log.Fatal(err)
	}
	return w
}

func main() {
	// Unterfenster unten rechts: eigener Rahmen + 4 kleine Scheiben
	sub := must(NewWindow("unten rechts",
		Frame{"Innenrahmen", 40},
		Glass{"klein 1", 15},
		Glass{"klein 2", 15},
		Glass{"klein 3", 15},
		Glass{"klein 4", 15},
	))

	// Gesamtfenster
	window := must(NewWindow("Fenster",
		Frame{"Außenrahmen", 120},
		Glass{"oben links", 50},
		Glass{"oben rechts", 60},
		Glass{"unten links", 50},
		sub,
	))

	window.Display(0)
	fmt.Printf("\nGesamtpreis: %.2f €\n", window.Price())
}
