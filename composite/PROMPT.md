# Prompt (Claude Code)

Implementiere in Go (nur Standardbibliothek, mit go.mod) das Composite Pattern aus folgendem UML:

- Interface `Item` mit `Price() float64` und `Display(indent int)`.
- Blätter: `Glass` und `Frame` (Name + Preis).
- Composite: `Window` enthält 1..* `Item`s (also auch weitere Windows). `Price()` = Summe der Kinder, `Display()` gibt den Baum eingerückt aus. `NewWindow` erzwingt mindestens ein Item.
- `main()`: baue ein Fenster nach: Rahmen; oben 2 Scheiben; unten links 1 Scheibe; unten rechts ein Unterfenster mit eigenem Rahmen und 4 kleinen Scheiben. Gib Baum und Gesamtpreis aus.
- Ein Unit-Test für `Price()`. Idiomatisch, kurz kommentiert.
