package main

import "testing"

func TestPrice(t *testing.T) {
	sub, _ := NewWindow("sub", Frame{"f", 10}, Glass{"g", 5})
	w, _ := NewWindow("w", Frame{"F", 100}, Glass{"G", 20}, sub)
	if got := w.Price(); got != 135 {
		t.Errorf("Price() = %v, want 135", got)
	}
}

func TestNewWindowEmpty(t *testing.T) {
	if _, err := NewWindow("leer"); err == nil {
		t.Error("leeres Window sollte Fehler liefern")
	}
}
