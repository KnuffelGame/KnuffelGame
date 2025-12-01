# Healthcheck Library

Übersicht und Zweck
- Leichtgewichtige Bibliothek, die einen Liveness-Endpoint für Services bereitstellt.
- Registriert ohne weitere Konfiguration den Pfad `GET /healthcheck` auf einem chi.Router.
- Antwortet deterministisch und ohne externe Abhängigkeiten; ideal für Container-/Load-Balancer-Healthchecks.

Exportoberfläche und API-Referenz
- Öffentliche Funktionen:
  - [func Mount(r chi.Router)](backend/libs/healthcheck/healthcheck.go:11) – registriert `GET /healthcheck` auf dem übergebenen Router.
  - [func Handler() http.Handler](backend/libs/healthcheck/healthcheck.go:21) – liefert einen eigenständigen Handler für den Health-Endpoint.
- Bereitgestellter Pfad:
  - `GET /healthcheck`
- Antwort:
  - Status: `200 OK`
  - Header: `Content-Type: application/json`
  - Body: `{"status":"ok"}`
- Middleware-Implikationen:
  - Die Bibliothek bringt keine eigene Middleware mit. Wenn Services eine Logging-Middleware wie [logger.ChiMiddleware](backend/libs/logger/middleware.go:1) nutzen, wird der Health-Endpoint automatisch mitgeloggt.

Unterstützte Laufzeitumgebungen
- Go-Version gemäß [go.mod](backend/libs/healthcheck/go.mod): `go 1.25.3`
- Ziel: Nutzung innerhalb der Monorepo-Services mit Go 1.25.x (oder konsistenter Toolchain), wie in den Service-Modulen konfiguriert.

Abhängigkeiten
- Framework: `github.com/go-chi/chi/v5 v5.2.3` (siehe [go.sum](backend/libs/healthcheck/go.sum))
- Weitere externe Abhängigkeiten: keine

Konfiguration
- Keine Umgebungsvariablen erforderlich.
- Keine konfigurierbaren Parameter; der Pfad `/healthcheck` ist fest (Hardcoded).
- Keine speziellen Header notwendig; die Bibliothek setzt `Content-Type: application/json`.

Verwendung
- Mount auf einem chi.Router:
  ```go
  import (
      "github.com/go-chi/chi/v5"
      "github.com/KnuffelGame/KnuffelGame/backend/libs/healthcheck"
  )

  r := chi.NewRouter()
  healthcheck.Mount(r)
  ```
- Standalone-Handler:
  ```go
  import "github.com/KnuffelGame/KnuffelGame/backend/libs/healthcheck"

  h := healthcheck.Handler()
  // z. B. http.Handle("/healthcheck", h)
  ```
- cURL-Beispiel:
  ```bash
  curl -s http://localhost:PORT/healthcheck
  # {"status":"ok"}
  ```
- Integration in Services (Referenzen):
  - AuthService: [internal/router.go](backend/services/AuthService/internal/router.go:21)
  - LobbyService: [internal/router.go](backend/services/LobbyService/internal/router.go:23)
  - SSEService: [internal/router.go](backend/services/SSEService/internal/router.go:22)

Observability & Logging
- Wenn eine Logging-Middleware aktiv ist (z. B. [logger.ChiMiddleware](backend/libs/logger/middleware.go:1)), wird auch der Health-Endpunkt mitgeloggt (Felder wie method, path, status, duration_ms).
- Keine zusätzlichen Header oder Tracing-Konfigurationen notwendig.

Performance-Hinweise
- Zeitkomplexität: O(1).
- Keine Datenbank- oder Netzwerkzugriffe.
- Antwortgröße minimal; hervorragend geeignet als Liveness-Probe in Container-Orchestrierungen.

Versionierungskompatibilität
- Interne Bibliothek ohne externes Versioning; Nutzung via `replace`-Direktiven in Service-Modulen (siehe [backend/libs/README.md](backend/libs/README.md)).
- Kompatibel mit der im Monorepo verwendeten Go-Version laut [go.mod](backend/libs/healthcheck/go.mod).

Bekannte Stolpersteine & Troubleshooting
- Reverse-Proxy-Caching vermeiden:
  - Health-Endpunkte sollten niemals gecacht werden. Stelle sicher, dass Proxy-/Gateway-Konfigurationen Caching für `/healthcheck` deaktivieren (z. B. `Cache-Control: no-store` serverseitig oder Proxy-Regel).
- Load-Balancer-Intervall:
  - Wähle vernünftige Prüfintervalle (z. B. 5–10 Sekunden), um unnötige Last zu vermeiden.
- Pfad-Fixierung:
  - Der Pfad ist fest `/healthcheck`. Falls ein anderer Pfad benötigt wird, muss Service-seitig geroutet oder ein Wrapper-Handler verwendet werden.

Bekannte Abweichungen
- Vorherige Dokumentation (z. B. [backend/libs/README.md](backend/libs/README.md)) und ältere README-Versionen behaupteten eine Plain-Text-Antwort `1` mit `Content-Type: text/plain`. Tatsächliche Implementierung liefert JSON `{"status":"ok"}` mit `Content-Type: application/json` (siehe [func Mount(r chi.Router)](backend/libs/healthcheck/healthcheck.go:11), [func Handler() http.Handler](backend/libs/healthcheck/healthcheck.go:21)).
  - Remediation:
    - Dokumente, die noch `1`/`text/plain` nennen, auf `{"status":"ok"}` und `application/json` aktualisieren.
    - Falls erforderlich, zentralen Abschnitt „Library Reference“ in [backend/libs/README.md](backend/libs/README.md) auf die JSON-Antwort korrigieren.
- Readiness-Checks:
  - Die Bibliothek stellt ausschließlich einen Liveness-Endpoint bereit. Frühere Hinweise zu separaten Readiness-Probes sind als Infrastruktur-Entscheidung zu verstehen.
  - Vorschlag (nicht implementiert): Optionalen Readiness-Endpoint oder konfigurierbaren Pfad als Erweiterung entwerfen, falls künftig benötigt.

Lizenz/Metadaten
- Keine Lizenzinformationen in [go.mod](backend/libs/healthcheck/go.mod) hinterlegt; zusätzliche Lizenzangaben entfallen hier.
