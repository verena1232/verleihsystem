package main

import (
	"errors"
	"fmt"
	"strings"
)

// Item ist die gemeinsame Komponente des Composite Patterns.
type Item interface {
	Price() float64
	Display(indent int)
}

// Glass ist ein Blatt: eine einzelne Glasscheibe.
type Glass struct {
	Name string
	Cost float64
}

func (g Glass) Price() float64 { return g.Cost }

func (g Glass) Display(indent int) {
	fmt.Printf("%sGlass %q: %.2f €\n", strings.Repeat("  ", indent), g.Name, g.Cost)
}

// Frame ist ein Blatt: ein Rahmen.
type Frame struct {
	Name string
	Cost float64
}

func (f Frame) Price() float64 { return f.Cost }

func (f Frame) Display(indent int) {
	fmt.Printf("%sFrame %q: %.2f €\n", strings.Repeat("  ", indent), f.Name, f.Cost)
}

// Window ist das Composite: enthält 1..* Items (auch weitere Windows).
type Window struct {
	Name  string
	items []Item
}

// NewWindow erzwingt die Multiplizität 1..*.
func NewWindow(name string, items ...Item) (*Window, error) {
	if len(items) == 0 {
		return nil, errors.New("window braucht mindestens ein Item")
	}
	return &Window{Name: name, items: items}, nil
}

// Price summiert rekursiv die Preise aller Kinder.
func (w *Window) Price() float64 {
	sum := 0.0
	for _, it := range w.items {
		sum += it.Price()
	}
	return sum
}

// Display gibt das Fenster und rekursiv alle Kinder eingerückt aus.
func (w *Window) Display(indent int) {
	fmt.Printf("%sWindow %q: %.2f €\n", strings.Repeat("  ", indent), w.Name, w.Price())
	for _, it := range w.items {
		it.Display(indent + 1)
	}
}
