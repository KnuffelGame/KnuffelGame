# Auth-Bibliothek (Header-basierte Auth)

Übersicht und Zweck
-------------------
Diese Bibliothek stellt eine schlanke Middleware bereit, die zwei vom Gateway gesetzte HTTP-Header ausliest, validiert und daraus einen typisierten Benutzerkontext erstellt. Standard-Header sind `X-User-ID` und `X-Username`. Der Benutzer wird als `User` im Request-Context gespeichert und steht nachgelagerten Handlern und Guards zur Verfügung. Siehe [auth.User()](backend/libs/auth/auth.go:21).

Exportoberfläche und API-Referenz
---------------------------------
- Typen
  - [auth.User()](backend/libs/auth/auth.go:21)
    - Felder: `ID uuid.UUID`, `Username string`
- Funktionen
  - [auth.FromContext()](backend/libs/auth/auth.go:27)
    - Liefert den zuvor injizierten `User` aus `context.Context`. Rückgabe `(User, bool)`; `bool=false`, wenn kein Benutzer im Kontext vorhanden ist.
  - [auth.NewAuthMiddleware()](backend/libs/auth/auth.go:35)
    - Konstruktor für eine Chi-kompatible Middleware. Parameter: `userHeader`, `usernameHeader` (beide `string`). Liest die angegebenen Headernamen, validiert sie und injiziert `User` in den Kontext. Signatur: `func(userHeader, usernameHeader string) func(http.Handler) http.Handler`.
- Variablen
  - [auth.AuthMiddleware()](backend/libs/auth/auth.go:63)
    - Vorkonfigurierte Middleware mit den Standard-Headern.
- Konstanten
  - [auth.DefaultHeaderUserID()](backend/libs/auth/auth.go:14) = `"X-User-ID"`
  - [auth.DefaultHeaderUsername()](backend/libs/auth/auth.go:15) = `"X-Username"`

Unterstützte Laufzeitumgebungen
-------------------------------
- Go-Version: aus [go.mod](backend/libs/auth/go.mod) abgeleitet – `go 1.25.3`.
- Chi-Router wird indirekt unterstützt (über Services), ist jedoch kein harte Abhängigkeit der Bibliothek.

Abhängigkeiten
--------------
- Interne Bibliotheken:
  - [backend/libs/httpx](backend/libs/httpx/httpx.go) – standardisierte JSON- und Fehlerantworten (z. B. [httpx.WriteBadRequest()](backend/libs/httpx/httpx.go:42))
  - [backend/libs/logger](backend/libs/logger/middleware.go) – Request-Logging (z. B. [logger.ChiMiddleware()](backend/libs/logger/middleware.go:15))
- Extern:
  - `github.com/google/uuid` – UUID-Parsing
- Siehe [go.mod](backend/libs/auth/go.mod) für genaue Versionen und `replace`-Direktiven im Monorepo-Kontext.

Konfiguration und erwartete Header
----------------------------------
| Headername  | Typ                         | Pflicht | Beispiel                                    | Hinweise                                                                                   |
|-------------|------------------------------|---------|---------------------------------------------|--------------------------------------------------------------------------------------------|
| X-User-ID   | UUID (RFC 4122, i. d. R. v4) | Ja      | `550e8400-e29b-41d4-a716-446655440000`      | Validierung via `uuid.Parse()`. Kein strikter v4-Check; es wird allgemeines UUID-Format geprüft. |
| X-Username  | string                       | Ja      | `Alice`                                     | Keine Längen-/Zeichenvalidierung durch die Bibliothek; Services können eigene Regeln erzwingen.   |

Verhalten und Fehlerbehandlung
------------------------------
- Fehlende Header (`X-User-ID` oder `X-Username`): 400 Bad Request mit JSON-Fehlerhülle (Code `bad_request`), geschrieben via [httpx.WriteBadRequest()](backend/libs/httpx/httpx.go:42).
- Ungültige UUID in `X-User-ID`: 400 Bad Request mit Detailhinweis, geschrieben via [httpx.WriteBadRequest()](backend/libs/httpx/httpx.go:42).
- Logging: Warnungen unter der Logger-Gruppe `"middleware"` mit Attribut `action="auth"`, erzeugt über das Logger-Paket.
- Autorisierung findet NICHT in der Middleware statt; dafür sind service-spezifische Guards zuständig (z. B. DB-/Rollenprüfungen).

Verwendung (chi.Router)
-----------------------
Middleware einbinden:
```go
r := chi.NewRouter()
// Auth für /lobbies aktivieren
r.Route("/lobbies", func(r chi.Router) {
    r.Use(auth.AuthMiddleware) // injiziert auth.User in den Kontext
    r.Post("/", handlers.CreateLobbyHandler(...))
    r.With(handlers.RequireLobbyMember(db)).Get("/{lobby_id}", handlers.GetLobbyHandler(db))
})
```
Referenz: [auth.AuthMiddleware()](backend/libs/auth/auth.go:63), [handlers.RequireLobbyMember()](backend/services/LobbyService/internal/handlers/authorizers.go:16)

Benutzerkontext in Handlern auslesen:
```go
u, ok := auth.FromContext(r.Context())
if !ok {
    // ggf. 401/400 zurückgeben (fehlende Auth) – abhängig vom Handler-/Service-Konzept
}
// Zugriff: u.ID, u.Username
```
Referenz: [auth.FromContext()](backend/libs/auth/auth.go:27)

Beispiel mit Service-Guards
---------------------------
- Mitgliedschaft prüfen: [handlers.RequireLobbyMember()](backend/services/LobbyService/internal/handlers/authorizers.go:16)
- Leader-Recht prüfen: [handlers.RequireLobbyLeader()](backend/services/LobbyService/internal/handlers/authorizers.go:82)

Diese Guards erwarten, dass der Benutzerkontext durch die Auth-Middleware gesetzt wurde und antworten auf Fehlersituationen mit [httpx.WriteUnauthorized()](backend/libs/httpx/httpx.go:47), [httpx.WriteBadRequest()](backend/libs/httpx/httpx.go:42), [httpx.WriteForbidden()](backend/libs/httpx/httpx.go:52) bzw. [httpx.WriteNotFound()](backend/libs/httpx/httpx.go:57).

Performance-Hinweise
--------------------
- Geringe Kosten: Header-Lookups und ein `uuid.Parse()` pro Request.
- Reihenfolge der Middleware:
  - Empfehlung: [logger.ChiMiddleware()](backend/libs/logger/middleware.go:15) möglichst außen (früh) einbinden, damit auch abgebrochene Requests (z. B. 400 aus Auth) geloggt werden.
  - Auth vor Authorizer-Guards, damit `auth.User` verfügbar ist.

Versionierung & Kompatibilität
------------------------------
- Interne libs verwenden im Monorepo `replace`-Direktiven (siehe [go.mod](backend/libs/auth/go.mod)). SemVer ist intern nicht streng erforderlich, externe Nutzung sollte Versionen pinnen.
- Kompatible Go-Version laut Modul: `1.25.3`. Services sollten diese oder neuere kompatible Minor-Version verwenden.

Bekannte Stolpersteine & Troubleshooting
----------------------------------------
- 400 Bad Request bei fehlenden/ungültigen Auth-Headern.
- Falsches UUID-Format (z. B. Tippfehler) führt zu 400.
- Proxy-/Gateway-Interaktion: Stellen Sie sicher, dass Upstream-Header (`X-User-ID`, `X-Username`) nicht überschrieben oder entfernt werden. Bei Reverse-Proxies ggf. Header-Weiterleitung explizit erlauben.

Bekannte Abweichungen
---------------------
- Striktes UUIDv4: Der Code prüft aktuell allgemeines UUID-Format via `uuid.Parse()` und erzwingt keine v4-Variante. Vorschlag: Optionalen Validierungsmodus ergänzen (z. B. `OptionStrictUUIDv4`), oder Services dokumentieren, dass Producer v4 liefern müssen.
- Username-Regeln: Diese Bibliothek erzwingt keine maximale Länge oder Zeichenmenge. Services (z. B. LobbyService) dokumentieren `max 20`. Vorschlag: Entweder Regeln in Services belassen (empfohlen) oder optionale Validierung in [auth.NewAuthMiddleware()](backend/libs/auth/auth.go:35) per Funktionsoptionen ermöglichen.
- Statuscode-Konsistenz: Die Auth-Middleware antwortet bei fehlenden Headers mit 400. Einige Guards verwenden 401, wenn der Benutzer im Kontext fehlt (z. B. initiale Prüfungen). Vorschlag: Dienst-spezifische Richtlinie definieren und ggf. Middleware-Option zur Auswahl von 400/401 anbieten; bis dahin Verhalten im Service-Handbuch klar dokumentieren.

Repository-Konventionen
-----------------------
- Stil und Terminologie synchron zu Services: „Benutzerkontext“, „Header-basierte Auth“, „Guards/Authorizers“.
- Keine OpenAPI in Libraries; Dokument bleibt fokussiert auf Exportoberfläche und Laufzeitverhalten.
- Monorepo-Setup: Services binden die Bibliothek via `require` und `replace`. Beispiel siehe [go.mod](backend/libs/auth/go.mod).

Praktische Beispiele (curl)
---------------------------
Create Lobby (Service-Beispiel, erfordert Auth-Header):
```bash
curl -sS -X POST http://localhost:8083/lobbies \
  -H "X-User-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -H "X-Username: Alice"
```

Join Lobby:
```bash
curl -sS -X POST http://localhost:8083/lobbies/join \
  -H "Content-Type: application/json" \
  -H "X-User-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -H "X-Username: Alice" \
  -d '{"join_code":"ABC123"}'
```

Tests
-----
Unit-Tests sind im Modul enthalten. Ausführen:
```bash
cd backend/libs/auth
go test ./...
```
