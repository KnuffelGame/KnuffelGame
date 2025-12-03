# ToDos: Implementierung des API Gateways

Dieses Dokument beschreibt die geplanten Aufgaben und Teilschritte für die Implementierung des API Gateways gemäß den definierten Anforderungen und dem `CODING_STYLE.md`.

## Phase 1: Projekt-Setup & Abhängigkeiten

- [ ] **`go.mod` initialisieren:** Die `go.mod` Datei im Service-Verzeichnis `backend/services/APIGateway` anlegen und initialisieren.
- [ ] **Abhängigkeiten hinzufügen:** Die notwendigen Go-Module hinzufügen:
    - `github.com/go-chi/chi/v5` für das Routing.
    - `github.com/go-chi/cors` für CORS-Handling.
    - Lokale Bibliotheken (`logger`, `httpx`, `healthcheck`) über `replace` direktiven einbinden.
- [ ] **Verzeichnisstruktur anlegen:** Die grundlegende Ordnerstruktur gemäß `CODING_STYLE.md` erstellen:
    - `cmd/APIGateway/main.go`
    - `internal/router/router.go`
    - `internal/middleware/auth.go`
    - `internal/handlers/proxy.go`
    - `pkg/config/config.go`

## Phase 2: Konfiguration

- [ ] **Konfigurationsmodell implementieren:** In `pkg/config/config.go` eine `Config` struct und eine `Load()` Funktion erstellen, die folgende Umgebungsvariablen lädt:
    - `PORT`
    - `LOG_LEVEL`
    - `SERVICE_NAME`
    - `AUTH_SERVICE_URL`
    - `LOBBY_SERVICE_URL`
    - `GAME_SERVICE_URL`
    - `COOKIE_DOMAIN`
    - `COOKIE_SAMESITE`
    - `COOKIE_SECURE`
- [ ] **Beispiel-Konfiguration erstellen:** Eine `env.d/APIGateway.env.example` Datei mit allen oben genannten Variablen und sinnvollen Standardwerten für das lokale Development anlegen.

## Phase 3: Service Bootstrap (`main.go`)

- [ ] **`main` Funktion implementieren:** Den Service-Startpunkt in `cmd/APIGateway/main.go` nach dem im `CODING_STYLE.md` definierten Bootstrap-Pattern umsetzen (Config laden, Logger initialisieren, Router erstellen, HTTP-Server mit graceful shutdown starten).

## Phase 4: Routing & Reverse Proxy

- [ ] **Router implementieren:** In `internal/router/router.go` die Funktion `New()` implementieren, die den `chi.Router` konfiguriert.
- [ ] **Basis-Middleware einbinden:** Standard-Middleware für Logging (`logger.ChiMiddleware`), Request-ID und Health-Check (`/healthcheck`) einbinden.
- [ ] **Reverse-Proxy Handler erstellen:** In `internal/handlers/proxy.go` eine wiederverwendbare Handler-Funktion entwickeln, die eine Anfrage an eine Ziel-URL weiterleitet.
- [ ] **Routen definieren:** Die Weiterleitungsregeln im Router implementieren:
    - `/lobbies/{*}` -> `LOBBY_SERVICE_URL`
    - `/games/{*}` -> `GAME_SERVICE_URL`
    - Sicherstellen, dass `/internal/*` nicht öffentlich erreichbar ist.

## Phase 5: Authentifizierungs-Middleware

- [ ] **Zentrale Auth-Middleware erstellen:** In `internal/middleware/auth.go` die Haupt-Middleware-Funktion `Authentication()` implementieren, die die zwei unterschiedlichen Workflows handhabt.
- [ ] **Workflow 1: Token-Validierung (Standardfall):**
    - [ ] JWT aus dem `jwt`-Cookie extrahieren.
    - [ ] Eine Client-Funktion implementieren, die den `AuthService` via `POST /internal/validate` aufruft.
    - [ ] Bei ungültigem Token die Anfrage mit `401 Unauthorized` und einer Standard-Fehler-JSON abbrechen.
    - [ ] Bei Erfolg die `UserID` und den `Username` aus der Antwort extrahieren und in den Request-Kontext für den nächsten Middleware-Schritt legen.
- [ ] **Workflow 2: Token-Erstellung (Spezialfall `POST /lobbies`, `POST /lobbies/join`):**
    - [ ] Implementieren der Logik zur Erkennung dieser speziellen Endpunkte.
    - [ ] Request-Body sicher lesen, um den `username` zu extrahieren, ohne den Body für den Downstream-Service zu verbrauchen (z.B. durch `GetBody`).
    - [ ] Eine Client-Funktion implementieren, die den `AuthService` via `POST /internal/create` aufruft.
    - [ ] Die `UserID` und den `Username` aus der Antwort in den Request-Kontext legen.
    - [ ] Einen Response-Writer-Wrapper implementieren, der das `Set-Cookie` Header mit dem neuen Token, der konfigurierten Domain, SameSite etc. in die Antwort an den Client einfügt.
- [ ] **Header-Injection Middleware:** Eine separate, kleine Middleware erstellen, die nach der Auth-Middleware läuft. Sie liest `UserID` und `Username` aus dem Kontext und setzt die Header `X-User-ID` und `X-Username`.

## Phase 6: Docker & Orchestrierung

- [ ] **Dockerfile erstellen/anpassen:** Ein mehrstufiges `Dockerfile` für das API-Gateway erstellen, das den Konventionen aus `CODING_STYLE.md` folgt (non-root user, healthcheck).
- [ ] **`docker-compose.yaml` erweitern:** Den `APIGateway`-Service zur `docker-compose.yaml` hinzufügen, inklusive Port-Mapping (`8080:8080`), Abhängigkeiten (`depends_on`) und den notwendigen Umgebungsvariablen über `env_file`.

## Phase 7: Tests

- [ ] **Unit-Tests für Konfiguration:** Die `Load()` Funktion testen.
- [ ] **Integration-Tests für Middleware:** Umfassende Tests für die `Authentication`-Middleware schreiben. Dies erfordert das Mocking des `AuthService` mit `httptest`.
    - [ ] Testfall: Gültige Anfrage mit vorhandenem Cookie (Workflow 1).
    - [ ] Testfall: Anfrage mit fehlendem/ungültigem Cookie (Workflow 1, Fehlerfall).
    - [ ] Testfall: `POST /lobbies` Anfrage (Workflow 2), der prüft, ob der `Set-Cookie` Header korrekt in der Antwort gesetzt und die Anfrage mit den richtigen `X-User-*` Headern weitergeleitet wird.
