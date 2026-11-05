# Spezifikation: Ski-Verleih (Aufgabe 1 a + b)

Dieses Dokument ist die Grundlage für die Implementierung (Spec-Driven Development).
Jede Anforderung hat eine ID, auf die Code, Tests und README verweisen.

## 1. Ziel

Ein kleiner Ski-Verleih mit Weboberfläche. Mitarbeiter:innen verwalten Ski und Kund:innen,
legen Ausleihen/Reservierungen an und nehmen Ski zurück. Prototyp: Daten liegen nur im Speicher.

## 2. Begriffe

| Begriff     | Bedeutung |
|-------------|-----------|
| Ski         | Ein verleihbares Paar Ski mit eindeutiger Inventarnummer |
| Person      | Kund:in mit Kontaktdaten und Ausweis |
| Ausweis     | Ausweistyp + Ausweisnummer, optional ein hochgeladenes Bild/PDF |
| Ausleihe    | Zuordnung Ski ↔ Person für einen Zeitraum [Beginn, Ende) |
| Reservierung| Eine Ausleihe, deren Beginn in der Zukunft liegt |
| Preisregel  | Stundenpreis + Tageshöchstpreis pro Ski-Kategorie |

## 3. Anforderungen Teil a) – Ski-Verleih mit Dashboard

### Ski-Verwaltung
- **A-1** Ski anlegen, bearbeiten, löschen. Felder: Inventarnummer (Pflicht, eindeutig), Marke (Pflicht), Modell, Länge in cm (Pflicht, 70–220), Kategorie (Kinder, Allround, Race, Freeride, Touren), Notiz.
- **A-2** Ein Ski mit offener Ausleihe oder Reservierung kann nicht gelöscht werden.
- **A-3** Die Ski-Liste zeigt den aktuellen Status (verfügbar / ausgeliehen / reserviert) und ist durchsuchbar und nach Kategorie filterbar.

### Personen-Verwaltung
- **A-4** Person anlegen, bearbeiten, löschen. Felder: Vorname, Nachname (Pflicht), E-Mail, Telefon.
- **A-5** Ausweis **eingeben**: Ausweistyp (Personalausweis, Reisepass, Führerschein) und Ausweisnummer sind Pflichtfelder.
- **A-6** Ausweis **hochladen**: optional ein Bild (JPG/PNG/WebP) oder PDF, max. 5 MB. Die Datei kann angesehen, ersetzt und entfernt werden.
- **A-7** Eine Person mit offener Ausleihe kann nicht gelöscht werden.

### Ausleihe
- **A-8** Ausleihe anlegen: Person wählen, Zeitraum wählen, aus den **im Zeitraum verfügbaren** Ski einen auswählen.
- **A-9** Ein Ski kann sich nie in zwei überlappenden Ausleihen befinden (auch bei gleichzeitigen Anfragen; Store ist thread-safe).
- **A-10** Rückgabe eines ausgeliehenen Ski. Der Endpreis wird aus der tatsächlichen Dauer berechnet.

### Dashboard
- **A-11** Kennzahlen: verfügbare Ski, ausgeliehene Ski, Ski gesamt, registrierte Personen.
- **A-12** Zusätzlich: überfällige Ausleihen, heute fällige Rückgaben, bevorstehende Reservierungen, Umsatz (abgeschlossene Ausleihen).
- **A-13** Schnellzugriff: Ski verwalten, Personen verwalten, Neue Ausleihe.
- **A-14** Liste der aktuell laufenden Ausleihen mit Rückgabe-Button direkt am Dashboard.

### Benutzerfreundlichkeit
- **A-15** Alle Formulare in Dialogen, Pflichtfelder markiert, Fehlermeldungen in verständlichem Deutsch direkt im Dialog.
- **A-16** Erfolg/Fehler als kurze Meldung (Toast). Löschen nur nach Bestätigung.
- **A-17** Bei der Ausleihe wird der Preis **vor** dem Buchen angezeigt.
- **A-18** Neue Person kann direkt aus dem Ausleih-Dialog angelegt werden.
- **A-19** Oberfläche funktioniert auch auf schmalen Bildschirmen (Tablet/Handy).

## 4. Anforderungen Teil b) – Flexibilisierung

- **B-1** Verleih**beginn** frei wählbar (Datum + Uhrzeit), Standard: jetzt. Beginn in der Zukunft = Reservierung. Beginn darf höchstens 5 Minuten in der Vergangenheit liegen.
- **B-2** Verleih**dauer** frei wählbar, mindestens 1 Stunde, höchstens 30 Tage. Schnellauswahl: 2 h, 4 h, 1 Tag, 3 Tage, 1 Woche; oder Ende direkt eingeben.
- **B-3** Preise **stundenweise**: jede angefangene Stunde kostet den Stundenpreis der Kategorie.
- **B-4** Tageshöchstpreis: pro angefangenem 24-h-Block wird höchstens der Tagespreis verrechnet.
  `Preis = volleTage × Tagespreis + min(restStunden × Stundenpreis, Tagespreis)`
- **B-5** Preise sind im Bereich „Preise“ pro Kategorie änderbar. Bereits gebuchte Ausleihen behalten ihren vereinbarten Preis, solange sie pünktlich zurückgegeben werden.
- **B-6** Bei der Rückgabe wird mit der tatsächlichen Dauer neu gerechnet (früher zurück = günstiger, später = teurer), mit dem Preis, der bei der Buchung galt.
- **B-7** Reservierungen (Beginn in der Zukunft) können storniert werden.
- **B-8** Status einer Ausleihe ergibt sich aus der Zeit: reserviert → aktiv → überfällig → zurückgegeben (bzw. storniert).

## 5. Nicht-Ziele
- Keine Persistenz, kein Login, keine Zahlungsabwicklung.
- Zeitzone: Server rechnet in UTC, Browser zeigt Lokalzeit.

## 6. Akzeptanzkriterien (Auszug, als Tests umgesetzt)
| ID | Kriterium | Test |
|----|-----------|------|
| B-3 | 1 h 1 min Allround (4 €/h) kostet 8 € | `domain/preis_test.go` |
| B-4 | 26 h Allround (4 €/h, 25 €/Tag) kostet 25 € + 8 € = 33 € | `domain/preis_test.go` |
| A-9 | Zweite überlappende Ausleihe desselben Ski wird abgelehnt | `store/store_test.go` |
| A-9 | 50 parallele Buchungen desselben Ski → genau 1 erfolgreich | `store/store_test.go` |
| A-2 | Ski mit offener Ausleihe kann nicht gelöscht werden | `store/store_test.go` |
| B-6 | Frühe Rückgabe rechnet mit tatsächlicher Dauer | `store/store_test.go` |
| A-6 | Upload > 5 MB / falscher Typ wird abgelehnt | `handler/handler_test.go` |
