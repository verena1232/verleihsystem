// Verleih-Frontend – generisch für alle Plugins. Spricht nur mit der Go-API unter /api.
// Was ein Artikel für Felder hat, kommt aus /api/meta (Plugin-Beschreibung).
"use strict";

const $ = (sel, el = document) => el.querySelector(sel);
const $$ = (sel, el = document) => [...el.querySelectorAll(sel)];

let meta = { plugins: [], ausweisTypen: [] };
let artikelCache = [], personenCache = [], ausleihenCache = [];
const plugin = art => meta.plugins.find(p => p.art === art) || meta.plugins[0];
const symbol = art => plugin(art)?.symbol || "";

// ---------- Hilfsfunktionen ----------

async function api(methode, url, body) {
  const opt = { method: methode, headers: {} };
  if (body instanceof FormData) opt.body = body;
  else if (body !== undefined) {
    opt.headers["Content-Type"] = "application/json";
    opt.body = JSON.stringify(body);
  }
  const res = await fetch(url, opt);
  if (res.status === 204) return null;
  const daten = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(daten.fehler || `Fehler ${res.status}`);
  return daten;
}

function esc(s) {
  return String(s ?? "").replace(/[&<>"']/g, c => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
}

const euroFmt = new Intl.NumberFormat("de-AT", { style: "currency", currency: "EUR" });
const euro = cent => euroFmt.format((cent || 0) / 100);

const zeitFmt = new Intl.DateTimeFormat("de-AT", { weekday: "short", day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" });
const zeit = iso => (iso ? zeitFmt.format(new Date(iso)) : "–");

// Date → Wert für <input type="datetime-local">
function alsInput(d) {
  const p = n => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`;
}
// Standard-Beginn: jetzt, auf die Minute (B-1)
function jetztMinute() {
  const d = new Date();
  d.setSeconds(0, 0);
  return d;
}

function dauerText(stunden) {
  const t = Math.floor(stunden / 24), h = stunden % 24;
  const teile = [];
  if (t) teile.push(`${t} ${t === 1 ? "Tag" : "Tage"}`);
  if (h) teile.push(`${h} Std.`);
  return teile.join(" ") || "0 Std.";
}

function toast(text, art = "ok") {
  const el = document.createElement("div");
  el.className = `toast ${art === "fehler" ? "fehler-t" : "ok"}`;
  el.textContent = text;
  $("#toasts").append(el);
  setTimeout(() => el.remove(), 3500);
}

function frage(text) {
  const dlg = $("#dlg-frage");
  $("#frage-text").textContent = text;
  dlg.showModal();
  return new Promise(ok => dlg.addEventListener("close", () => ok(dlg.returnValue === "ja"), { once: true }));
}

function zeigeFehler(dlg, err) {
  const p = $(".fehler", dlg);
  p.textContent = err ? err.message : "";
  p.hidden = !err;
}

function badge(text, klasse = text) {
  return `<span class="badge b-${esc(klasse)}">${esc(text)}</span>`;
}

function optionen(werte, gewaehlt) {
  return werte.map(w => `<option ${w === gewaehlt ? "selected" : ""}>${esc(w)}</option>`).join("");
}

// ---------- Navigation ----------

const seiten = {
  dashboard: ladeDashboard,
  artikel: ladeArtikel,
  personen: ladePersonen,
  ausleihen: ladeAusleihen,
  preise: ladePreise,
};

function navigiere() {
  const name = location.hash.slice(1) || "dashboard";
  const seite = seiten[name] ? name : "dashboard";
  $$("section[data-page]").forEach(s => (s.hidden = s.dataset.page !== seite));
  $$("[data-nav]").forEach(a => a.classList.toggle("aktiv", a.dataset.nav === seite));
  document.body.classList.remove("menu-offen");
  seiten[seite]().catch(e => toast(e.message, "fehler"));
}

function aktuelleSeiteNeuLaden() {
  const name = location.hash.slice(1) || "dashboard";
  (seiten[name] || ladeDashboard)().catch(e => toast(e.message, "fehler"));
}

// ---------- Dashboard ----------

async function ladeDashboard() {
  const ende = new Date(); ende.setHours(23, 59, 59, 999);
  const d = await api("GET", `/api/dashboard?tagesende=${encodeURIComponent(ende.toISOString())}`);
  const k = d.kennzahlen;
  $("#k-verfuegbar").textContent = k.verfuegbar;
  $("#k-ausgeliehen").textContent = k.ausgeliehen;
  $("#k-gesamt").textContent = k.gesamt;
  $("#k-personen").textContent = k.personen;
  $("#k-ueberfaellig").textContent = k.ueberfaellig;
  $("#k-ueberfaellig").parentElement.classList.toggle("aktiv", k.ueberfaellig > 0);
  $("#k-heute").textContent = k.heuteFaellig;
  $("#k-reserv").textContent = k.reservierungen;
  $("#k-umsatz").textContent = euro(k.umsatzCent);

  const reihenfolge = { "überfällig": 0, aktiv: 1, reserviert: 2 };
  const offen = d.offen.sort((a, b) => reihenfolge[a.status] - reihenfolge[b.status] || new Date(a.ende) - new Date(b.ende));
  $("#dash-offen").innerHTML = offen.length
    ? offen.map(ausleiheZeile).join("")
    : `<tr><td colspan="7" class="leer">Keine offenen Ausleihen – alles im Lager.</td></tr>`;
}

function ausleiheZeile(a) {
  let aktion = "";
  if (a.status === "aktiv" || a.status === "überfällig")
    aktion = `<button class="knopf klein gruen" onclick="zurueckgeben(${a.id})">Rückgabe</button>`;
  else if (a.status === "reserviert")
    aktion = `<button class="link rot" onclick="stornieren(${a.id})">Stornieren</button>`;
  const endeSpalte = a.rueckgabe
    ? `${zeit(a.rueckgabe)} <small class="leise">(geplant ${zeit(a.ende)})</small>`
    : zeit(a.ende);
  return `<tr>
    <td>${badge(a.status)}</td>
    <td>${symbol(a.art)} ${esc(a.artikelText)} <small class="leise">${esc(a.kategorie)}</small></td>
    <td>${esc(a.personName)}</td>
    <td>${zeit(a.beginn)}</td>
    <td>${endeSpalte}</td>
    <td class="rechts">${euro(a.preisCent)}</td>
    <td class="aktionen">${aktion}</td>
  </tr>`;
}

async function zurueckgeben(id) {
  try {
    const a = await api("POST", `/api/ausleihen/${id}/rueckgabe`);
    toast(`Zurückgenommen – zu zahlen: ${euro(a.preisCent)}`);
    aktuelleSeiteNeuLaden();
  } catch (e) { toast(e.message, "fehler"); }
}

async function stornieren(id) {
  if (!(await frage("Reservierung wirklich stornieren?"))) return;
  try {
    await api("POST", `/api/ausleihen/${id}/storno`);
    toast("Reservierung storniert");
    aktuelleSeiteNeuLaden();
  } catch (e) { toast(e.message, "fehler"); }
}

// ---------- Artikel (generisch) ----------

async function ladeArtikel() {
  artikelCache = await api("GET", "/api/artikel");
  zeigeArtikel();
}

function zeigeArtikel() {
  const q = $("#art-suche").value.trim().toLowerCase();
  const art = $("#art-art").value, kat = $("#art-kat").value, st = $("#art-status").value;
  const liste = artikelCache.filter(a =>
    (!art || a.art === art) && (!kat || a.kategorie === kat) && (!st || a.status === st) &&
    (!q || `${a.nummer} ${a.bezeichnung} ${a.kurzinfo} ${a.notiz}`.toLowerCase().includes(q)));
  $("#art-liste").innerHTML = liste.length ? liste.map(a => `<tr>
      <td><b>${esc(a.nummer)}</b></td>
      <td>${symbol(a.art)} ${esc(a.bezeichnung)}${a.notiz ? `<br><small class="leise">${esc(a.notiz)}</small>` : ""}</td>
      <td>${esc(a.kurzinfo)}</td>
      <td>${badge(a.kategorie, "kat")}</td>
      <td>${badge(a.status)}</td>
      <td class="aktionen">
        <button class="link" onclick="artikelDialog(${a.id})">Bearbeiten</button>
        <button class="link rot" onclick="artikelLoeschen(${a.id})">Löschen</button>
      </td></tr>`).join("")
    : `<tr><td colspan="6" class="leer">Nichts gefunden.</td></tr>`;
}

// Kategorie-Filter passend zur gewählten Art befüllen
function kategorieFilter(sel, art) {
  const kats = art ? plugin(art).kategorien : [...new Set(meta.plugins.flatMap(p => p.kategorien))];
  const alt = sel.value;
  sel.innerHTML = `<option value="">Alle Kategorien</option>` + kats.map(k => `<option>${esc(k)}</option>`).join("");
  if (kats.includes(alt)) sel.value = alt;
}

// Formularfelder aus der Plugin-Beschreibung erzeugen
function baueFelder(p, kategorie, werte) {
  $("#plugin-felder").innerHTML = p.felder
    .filter(f => !f.nurKategorie || f.nurKategorie.includes(kategorie))
    .map(f => {
      const label = `${esc(f.label)}${f.einheit ? ` (${esc(f.einheit)})` : ""}${f.pflicht ? " *" : ""}`;
      const v = werte[f.name] ?? "";
      const req = f.pflicht ? "required" : "";
      let input;
      if (f.typ === "auswahl")
        input = `<select name="m-${f.name}" ${req}><option value="">– wählen –</option>${f.optionen.map(o =>
          `<option ${o === v ? "selected" : ""}>${esc(o)}</option>`).join("")}</select>`;
      else if (f.typ === "zahl")
        input = `<input name="m-${f.name}" type="number" ${f.min ? `min="${f.min}"` : ""} ${f.max ? `max="${f.max}"` : ""} value="${esc(v)}" ${req}>`;
      else
        input = `<input name="m-${f.name}" value="${esc(v)}" ${req}>`;
      return `<label>${label}${input}</label>`;
    }).join("");
}

function sammleMerkmale() {
  const m = {};
  $$("#plugin-felder [name^=m-]").forEach(el => (m[el.name.slice(2)] = el.value));
  return m;
}

function artikelDialog(id) {
  const dlg = $("#dlg-artikel"), f = $("form", dlg);
  const vorhanden = artikelCache.find(x => x.id === id);
  const art = vorhanden ? vorhanden.art : ($("#art-art").value || meta.plugins[0].art);
  const p = plugin(art);
  const a = vorhanden || { art, nummer: naechsteNummer(p), kategorie: p.kategorien[0], notiz: "", merkmale: {} };

  $("#art-titel").textContent = id ? `${a.nummer} bearbeiten` : `${p.name} anlegen`;
  f.art.innerHTML = meta.plugins.map(x => `<option value="${x.art}" ${x.art === art ? "selected" : ""}>${x.symbol} ${esc(x.name)}</option>`).join("");
  f.art.disabled = !!id;
  f.kategorie.innerHTML = optionen(p.kategorien, a.kategorie);
  f.nummer.value = a.nummer;
  f.notiz.value = a.notiz;
  f.dataset.id = id || "";
  baueFelder(p, a.kategorie, a.merkmale);
  zeigeFehler(dlg);
  dlg.showModal();
}

function naechsteNummer(p) {
  const nr = artikelCache.filter(a => a.art === p.art).map(a => parseInt(a.nummer.replace(/\D/g, ""), 10) || 0);
  return `${p.praefix}${String(Math.max(0, ...nr) + 1).padStart(3, "0")}`;
}

async function artikelSpeichern() {
  const dlg = $("#dlg-artikel"), f = $("form", dlg);
  if (!f.reportValidity()) return;
  const daten = {
    art: f.art.value, nummer: f.nummer.value, kategorie: f.kategorie.value, notiz: f.notiz.value,
    merkmale: sammleMerkmale(),
  };
  try {
    const id = f.dataset.id;
    await api(id ? "PUT" : "POST", id ? `/api/artikel/${id}` : "/api/artikel", daten);
    dlg.close();
    toast(id ? "Gespeichert" : `${plugin(daten.art).name} angelegt`);
    ladeArtikel();
  } catch (e) { zeigeFehler(dlg, e); }
}

async function artikelLoeschen(id) {
  const a = artikelCache.find(x => x.id === id);
  if (!(await frage(`${a.nummer} ${a.bezeichnung} löschen?`))) return;
  try {
    await api("DELETE", `/api/artikel/${id}`);
    toast("Gelöscht");
    ladeArtikel();
  } catch (e) { toast(e.message, "fehler"); }
}

// ---------- Personen ----------

async function ladePersonen() {
  personenCache = await api("GET", "/api/personen");
  zeigePersonen();
}

function zeigePersonen() {
  const q = $("#person-suche").value.trim().toLowerCase();
  const liste = personenCache.filter(p =>
    !q || `${p.vorname} ${p.nachname} ${p.email} ${p.telefon} ${p.ausweis.nummer}`.toLowerCase().includes(q));
  $("#person-liste").innerHTML = liste.length ? liste.map(p => `<tr>
      <td><b>${esc(p.vorname)} ${esc(p.nachname)}</b></td>
      <td>${esc(p.email) || ""}${p.email && p.telefon ? "<br>" : ""}${esc(p.telefon)}</td>
      <td>${esc(p.ausweis.typ)}<br><small class="leise">${esc(p.ausweis.nummer)}</small>
        ${p.ausweisDatei ? `<br><a href="/api/personen/${p.id}/ausweis" target="_blank" rel="noopener">📎 Datei ansehen</a>` : ""}</td>
      <td>${p.offeneAusleihen || "–"}</td>
      <td class="aktionen">
        <button class="link" onclick="ausleiheDialog(${p.id})">Ausleihe</button>
        <button class="link" onclick="personDialog(${p.id})">Bearbeiten</button>
        <button class="link rot" onclick="personLoeschen(${p.id})">Löschen</button>
      </td></tr>`).join("")
    : `<tr><td colspan="5" class="leer">Keine Personen gefunden.</td></tr>`;
}

let personAusAusleihe = false;
let gewaehlteDatei = null;

function personDialog(id, ausAusleihe = false) {
  personAusAusleihe = ausAusleihe;
  const dlg = $("#dlg-person"), f = $("form", dlg);
  const p = personenCache.find(x => x.id === id) ||
    { vorname: "", nachname: "", email: "", telefon: "", ausweis: { typ: meta.ausweisTypen[0], nummer: "" } };
  $("#person-titel").textContent = id ? `${p.vorname} ${p.nachname} bearbeiten` : "Neue Person anlegen";
  for (const k of ["vorname", "nachname", "email", "telefon"]) f[k].value = p[k];
  f.ausweisTyp.innerHTML = optionen(meta.ausweisTypen, p.ausweis.typ);
  f.ausweisNummer.value = p.ausweis.nummer;
  f.dataset.id = id || "";
  f.datei.value = "";
  gewaehlteDatei = null;
  zeigeDateiVorschau(null);
  zeigeVorhandeneDatei(p);
  zeigeFehler(dlg);
  dlg.showModal();
  f.vorname.focus();
}

function zeigeVorhandeneDatei(p) {
  const info = $("#dlg-person .datei-info");
  info.hidden = !p.ausweisDatei;
  if (p.ausweisDatei) {
    $("span", info).textContent = `📎 ${p.ausweisDatei}`;
    $("a", info).href = `/api/personen/${p.id}/ausweis`;
  }
}

function zeigeDateiVorschau(datei) {
  const img = $("#dlg-person .vorschau"), txt = $("#dlg-person .ablage-text");
  if (!datei) {
    img.hidden = true; img.removeAttribute("src");
    txt.innerHTML = `📎 Ausweis-Foto oder PDF hierher ziehen oder <u>auswählen</u><br><small>JPG, PNG, WebP oder PDF · max. 5 MB · optional</small>`;
    return;
  }
  txt.innerHTML = `✅ <b>${esc(datei.name)}</b> (${(datei.size / 1024).toFixed(0)} KB) – wird beim Speichern hochgeladen<br><small>klicken zum Ändern</small>`;
  if (datei.type.startsWith("image/")) { img.src = URL.createObjectURL(datei); img.hidden = false; }
  else img.hidden = true;
}

function dateiWaehlen(datei) {
  const dlg = $("#dlg-person");
  if (!datei) return;
  const erlaubt = ["image/jpeg", "image/png", "image/webp", "application/pdf"];
  if (!erlaubt.includes(datei.type)) return zeigeFehler(dlg, new Error("Nur JPG, PNG, WebP oder PDF erlaubt."));
  if (datei.size > meta.maxUpload) return zeigeFehler(dlg, new Error("Die Datei ist größer als 5 MB."));
  zeigeFehler(dlg);
  gewaehlteDatei = datei;
  zeigeDateiVorschau(datei);
}

async function personSpeichern() {
  const dlg = $("#dlg-person"), f = $("form", dlg);
  if (!f.reportValidity()) return;
  const daten = {
    vorname: f.vorname.value, nachname: f.nachname.value, email: f.email.value, telefon: f.telefon.value,
    ausweis: { typ: f.ausweisTyp.value, nummer: f.ausweisNummer.value },
  };
  try {
    const id = f.dataset.id;
    let p = await api(id ? "PUT" : "POST", id ? `/api/personen/${id}` : "/api/personen", daten);
    if (gewaehlteDatei) {
      const fd = new FormData();
      fd.append("datei", gewaehlteDatei);
      try {
        p = await api("POST", `/api/personen/${p.id}/ausweis`, fd);
      } catch (e) {
        f.dataset.id = p.id; // Person ist gespeichert, nur Upload fehlgeschlagen
        throw new Error(`Person gespeichert, aber Upload fehlgeschlagen: ${e.message}`);
      }
    }
    dlg.close();
    toast(id ? "Person gespeichert" : `${p.vorname} ${p.nachname} angelegt`);
    if (personAusAusleihe) await ausleihePersonenLaden(p.id);
    else ladePersonen();
  } catch (e) { zeigeFehler(dlg, e); }
}

async function ausweisEntfernen() {
  const f = $("#dlg-person form"), id = f.dataset.id;
  if (!id) return;
  try {
    await api("DELETE", `/api/personen/${id}/ausweis`);
    $("#dlg-person .datei-info").hidden = true;
    const p = personenCache.find(x => x.id === +id);
    if (p) p.ausweisDatei = "";
    toast("Ausweisdatei entfernt");
    if (!personAusAusleihe) ladePersonen();
  } catch (e) { zeigeFehler($("#dlg-person"), e); }
}

async function personLoeschen(id) {
  const p = personenCache.find(x => x.id === id);
  if (!(await frage(`${p.vorname} ${p.nachname} löschen? Der Ausweis wird mitgelöscht.`))) return;
  try {
    await api("DELETE", `/api/personen/${id}`);
    toast("Person gelöscht");
    ladePersonen();
  } catch (e) { toast(e.message, "fehler"); }
}

// ---------- Ausleihen-Liste ----------

async function ladeAusleihen() {
  ausleihenCache = await api("GET", "/api/ausleihen");
  zeigeAusleihen();
}

function zeigeAusleihen() {
  const q = $("#aus-suche").value.trim().toLowerCase();
  const st = $("#aus-status").value;
  const liste = ausleihenCache.filter(a =>
    (st === "" || (st === "offen" ? !a.rueckgabe && !a.storniert : a.status === st)) &&
    (!q || `${a.artikelText} ${a.personName}`.toLowerCase().includes(q)));
  $("#aus-liste").innerHTML = liste.length ? liste.map(ausleiheZeile).join("")
    : `<tr><td colspan="7" class="leer">Keine Ausleihen gefunden.</td></tr>`;
}

// ---------- Neue Ausleihe (A-8, A-17, B-1, B-2) ----------

let gewaehlterArtikel = null, angebote = [], ladeZaehler = 0;

async function ausleihePersonenLaden(gewaehlt) {
  personenCache = await api("GET", "/api/personen");
  const sel = $("#dlg-ausleihe [name=personId]");
  sel.innerHTML = `<option value="">– Person wählen –</option>` + personenCache.map(p =>
    `<option value="${p.id}" ${p.id === gewaehlt ? "selected" : ""}>${esc(p.nachname)} ${esc(p.vorname)} · ${esc(p.ausweis.typ)} ${esc(p.ausweis.nummer)}</option>`).join("");
  pruefeBuchbar();
}

async function ausleiheDialog(personId) {
  const dlg = $("#dlg-ausleihe"), f = $("form", dlg);
  zeigeFehler(dlg);
  gewaehlterArtikel = null;
  await ausleihePersonenLaden(personId);
  const start = jetztMinute();
  f.beginn.value = alsInput(start);
  f.beginn.min = alsInput(new Date(Date.now() - 5 * 60e3));
  setzeDauer(4);
  dlg.showModal();
}

function setzeDauer(stunden) {
  const f = $("#dlg-ausleihe form");
  const b = new Date(f.beginn.value);
  if (isNaN(b)) return;
  f.ende.value = alsInput(new Date(b.getTime() + stunden * 3600e3));
  zeitraumGeaendert();
}

async function zeitraumGeaendert() {
  const dlg = $("#dlg-ausleihe"), f = $("form", dlg);
  const von = new Date(f.beginn.value), bis = new Date(f.ende.value);
  const stunden = Math.ceil((bis - von) / 3600e3);
  $$("#dauer-chips button").forEach(b => b.classList.toggle("aktiv", +b.dataset.h === stunden));
  f.ende.min = f.beginn.value;
  if (isNaN(von) || isNaN(bis)) return;

  const nr = ++ladeZaehler;
  try {
    const d = await api("GET", `/api/verfuegbar?von=${encodeURIComponent(von.toISOString())}&bis=${encodeURIComponent(bis.toISOString())}`);
    if (nr !== ladeZaehler) return; // veraltete Antwort
    zeigeFehler(dlg);
    angebote = d.artikel;
    const inZukunft = von > Date.now();
    $("#dauer-text").innerHTML = `Dauer: <b>${dauerText(d.stunden)}</b> (${d.stunden} angefangene Stunden)` +
      (inZukunft ? ` · wird als <b>Reservierung</b> gespeichert` : ` · Ausgabe sofort`);
    if (gewaehlterArtikel && !angebote.some(s => s.id === gewaehlterArtikel)) gewaehlterArtikel = null;
  } catch (e) {
    if (nr !== ladeZaehler) return;
    angebote = []; gewaehlterArtikel = null;
    $("#dauer-text").textContent = "";
    zeigeFehler(dlg, e);
  }
  zeigeAngebote();
}

function zeigeAngebote() {
  const art = $("#aw-art").value, kat = $("#aw-kat").value;
  const q = $("#aw-suche").value.trim().toLowerCase();
  const liste = angebote.filter(a => (!art || a.art === art) && (!kat || a.kategorie === kat) &&
    (!q || `${a.nummer} ${a.bezeichnung} ${a.kurzinfo}`.toLowerCase().includes(q)));
  $("#artikel-auswahl").innerHTML = liste.length ? liste.map(a => `
    <button type="button" class="artikel-option ${a.id === gewaehlterArtikel ? "gewaehlt" : ""}" data-id="${a.id}">
      <b>${symbol(a.art)} ${esc(a.nummer)} · ${esc(a.bezeichnung)}</b>
      <span class="leise">${esc(a.kategorie)} · ${esc(a.kurzinfo)}</span>
      <span class="preis">${euro(a.preisCent)}</span>
    </button>`).join("")
    : `<p class="leise">Für diesen Zeitraum ist nichts Passendes frei.</p>`;
  pruefeBuchbar();
}

function pruefeBuchbar() {
  const f = $("#dlg-ausleihe form");
  const s = angebote.find(x => x.id === gewaehlterArtikel);
  $("#summe").innerHTML = s ? `Gesamt: <b>${euro(s.preisCent)}</b>` : "";
  $("#buchen").disabled = !(s && f.personId.value);
}

async function ausleiheSpeichern() {
  const dlg = $("#dlg-ausleihe"), f = $("form", dlg);
  const daten = {
    artikelId: gewaehlterArtikel,
    personId: parseInt(f.personId.value, 10),
    beginn: new Date(f.beginn.value).toISOString(),
    ende: new Date(f.ende.value).toISOString(),
  };
  try {
    const a = await api("POST", "/api/ausleihen", daten);
    dlg.close();
    const reserviert = new Date(a.beginn) > new Date();
    toast(`${reserviert ? "Reservierung" : "Ausleihe"} gespeichert – ${euro(a.preisCent)}`);
    aktuelleSeiteNeuLaden();
  } catch (e) {
    zeigeFehler(dlg, e);
    zeitraumGeaendert(); // Liste aktualisieren, falls der Artikel inzwischen weg ist
  }
}

// ---------- Preise (B-5) ----------

async function ladePreise() {
  const preise = await api("GET", "/api/preise");
  $("#preis-liste").innerHTML = preise.map(p => `<tr data-art="${esc(p.art)}" data-kat="${esc(p.kategorie)}">
      <td class="nur-mehrere">${symbol(p.art)} ${esc(plugin(p.art).name)}</td>
      <td><b>${esc(p.kategorie)}</b></td>
      <td><input class="preis-input" type="number" step="0.10" min="0.10" name="stunde" value="${(p.stundenpreisCent / 100).toFixed(2)}"></td>
      <td><input class="preis-input" type="number" step="0.50" min="0" name="tag" value="${(p.tagespreisCent / 100).toFixed(2)}"></td>
      <td class="bsp3"></td><td class="bsp48"></td>
      <td class="aktionen"><button class="knopf klein" onclick="preisSpeichern(this)">Speichern</button></td>
    </tr>`).join("");
  $$("#preis-liste tr").forEach(beispielPreise);
}

// gleiche Formel wie domain.Preisregel.Berechne – nur zur Vorschau
function berechne(stundeCent, tagCent, stunden) {
  if (!tagCent) return stunden * stundeCent;
  return Math.floor(stunden / 24) * tagCent + Math.min((stunden % 24) * stundeCent, tagCent);
}

function beispielPreise(tr) {
  const h = Math.round(parseFloat($("[name=stunde]", tr).value) * 100) || 0;
  const t = Math.round(parseFloat($("[name=tag]", tr).value) * 100) || 0;
  $(".bsp3", tr).textContent = euro(berechne(h, t, 3));
  $(".bsp48", tr).textContent = euro(berechne(h, t, 48));
}

async function preisSpeichern(knopf) {
  const tr = knopf.closest("tr");
  const daten = {
    stundenpreisCent: Math.round(parseFloat($("[name=stunde]", tr).value) * 100),
    tagespreisCent: Math.round(parseFloat($("[name=tag]", tr).value || "0") * 100),
  };
  try {
    await api("PUT", `/api/preise/${encodeURIComponent(tr.dataset.art)}/${encodeURIComponent(tr.dataset.kat)}`, daten);
    toast(`Preis für ${tr.dataset.kat} gespeichert`);
  } catch (e) { toast(e.message, "fehler"); }
}

// ---------- Start ----------

function verbinde() {
  $("#art-suche").addEventListener("input", zeigeArtikel);
  $("#art-art").addEventListener("change", () => { kategorieFilter($("#art-kat"), $("#art-art").value); zeigeArtikel(); });
  $("#art-kat").addEventListener("change", zeigeArtikel);
  $("#art-status").addEventListener("change", zeigeArtikel);

  // Artikel-Dialog: Art/Kategorie ändern → passende Felder neu bauen
  const fd = $("#dlg-artikel form");
  fd.art.addEventListener("change", () => {
    const p = plugin(fd.art.value);
    fd.kategorie.innerHTML = optionen(p.kategorien, p.kategorien[0]);
    fd.nummer.value = naechsteNummer(p);
    baueFelder(p, fd.kategorie.value, {});
  });
  fd.kategorie.addEventListener("change", () => baueFelder(plugin(fd.art.value), fd.kategorie.value, sammleMerkmale()));
  $("#person-suche").addEventListener("input", zeigePersonen);
  $("#aus-suche").addEventListener("input", zeigeAusleihen);
  $("#aus-status").addEventListener("change", zeigeAusleihen);
  $("#preis-liste").addEventListener("input", e => beispielPreise(e.target.closest("tr")));

  // Ausleihe-Dialog
  const fa = $("#dlg-ausleihe form");
  let vorherBeginn = null;
  fa.beginn.addEventListener("focus", () => (vorherBeginn = new Date(fa.beginn.value)));
  fa.beginn.addEventListener("change", () => {
    // Dauer beibehalten, wenn der Beginn verschoben wird
    const alteDauer = new Date(fa.ende.value) - (vorherBeginn || new Date(fa.beginn.value));
    const b = new Date(fa.beginn.value);
    if (!isNaN(b) && alteDauer > 0) fa.ende.value = alsInput(new Date(b.getTime() + alteDauer));
    vorherBeginn = b;
    zeitraumGeaendert();
  });
  fa.ende.addEventListener("change", zeitraumGeaendert);
  fa.personId.addEventListener("change", pruefeBuchbar);
  $("#dauer-chips").addEventListener("click", e => { if (e.target.dataset.h) setzeDauer(+e.target.dataset.h); });
  $("#aw-art").addEventListener("change", () => { kategorieFilter($("#aw-kat"), $("#aw-art").value); zeigeAngebote(); });
  $("#aw-kat").addEventListener("change", zeigeAngebote);
  $("#aw-suche").addEventListener("input", zeigeAngebote);
  $("#artikel-auswahl").addEventListener("click", e => {
    const b = e.target.closest(".artikel-option");
    if (!b) return;
    gewaehlterArtikel = +b.dataset.id;
    zeigeAngebote();
  });

  // Ausweis-Upload: Klick oder Drag & Drop (A-6)
  const ablage = $("#ablage"), input = $("#dlg-person [name=datei]");
  input.addEventListener("change", () => dateiWaehlen(input.files[0]));
  ablage.addEventListener("dragover", e => { e.preventDefault(); ablage.classList.add("ueber"); });
  ablage.addEventListener("dragleave", () => ablage.classList.remove("ueber"));
  ablage.addEventListener("drop", e => {
    e.preventDefault(); ablage.classList.remove("ueber");
    dateiWaehlen(e.dataTransfer.files[0]);
  });

  window.addEventListener("hashchange", navigiere);
}

(async function start() {
  verbinde();
  meta = await api("GET", "/api/meta");
  // Texte an die Plugins anpassen: ein Plugin → "Fahrräder", mehrere → "Artikel"
  const ein = meta.plugins.length === 1, p0 = meta.plugins[0];
  document.body.classList.toggle("ein-plugin", ein);
  document.title = meta.titel;
  $("#logo").textContent = `${ein ? p0.symbol : "📦"} ${meta.titel}`;
  $$(".t-mehrzahl").forEach(el => (el.textContent = ein ? p0.mehrzahl : "Artikel"));
  $$(".t-name").forEach(el => (el.textContent = ein ? p0.name : "Artikel"));
  const artOpts = meta.plugins.map(p => `<option value="${p.art}">${p.symbol} ${esc(p.mehrzahl)}</option>`).join("");
  $("#art-art").insertAdjacentHTML("beforeend", artOpts);
  $("#aw-art").insertAdjacentHTML("beforeend", artOpts);
  kategorieFilter($("#art-kat"), "");
  kategorieFilter($("#aw-kat"), "");
  navigiere();
})();
