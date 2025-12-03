# APIGateway

API Gateway service for the Knuffel multiplayer Kniffel application. Acts as the single entry point for client requests, handling authentication via JWT cookies, proxying requests to LobbyService and GameService, and managing guest token creation for lobby operations.

## Übersicht und Zweck

- Single entry point for all client API requests
- JWT authentication via httpOnly cookies
- Automatic guest token creation for lobby creation and joining
- Request proxying to LobbyService (/lobbies/*) and GameService (/games/*)
- Header injection (X-User-ID, X-Username) for downstream services
- Healthcheck endpoint for service monitoring

## Architekturüberblick

- Router: Chi-Router mit Request-Logging ([internal/router/router.go](backend/services/APIGateway/internal/router/router.go))
- Middleware:
  - Request-Logging: ([backend/libs/logger/middleware.go](backend/libs/logger/middleware.go))
  - CORS: Permissive for development ([internal/router/router.go](backend/services/APIGateway/internal/router/router.go))
  - Authentication: JWT validation and guest token creation ([internal/middleware/auth.go](backend/services/APIGateway/internal/middleware/auth.go))
  - Header Injection: Adds X-User-ID and X-Username headers ([internal/middleware/auth.go](backend/services/APIGateway/internal/middleware/auth.go))
  - Healthcheck: ([backend/libs/healthcheck/healthcheck.go](backend/libs/healthcheck/healthcheck.go))
- Proxy Handler: Reverse proxy using httputil ([internal/handlers/proxy.go](backend/services/APIGateway/internal/handlers/proxy.go))
- Dependencies: AuthService (for token operations), LobbyService, GameService

## Endpunkte

Basis-URL: Service-Root (z. B. `http://api-gateway:8080`)

| Methode | Pfad                  | Beschreibung                          | Proxied To          |
|--------:|-----------------------|---------------------------------------|---------------------|
| GET     | /healthcheck          | Liveness-Check                        | -                  |
| POST    | /lobbies              | Create lobby                          | LobbyService       |
| POST    | /lobbies/join         | Join lobby                            | LobbyService       |
| GET     | /lobbies/{lobby_id}   | Get lobby details                     | LobbyService       |
| POST    | /lobbies/{lobby_id}/kick | Kick player                        | LobbyService       |
| GET     | /games/{game_id}      | Get game state                        | GameService        |
| POST    | /games/{game_id}/roll | Roll dice                             | GameService        |
| POST    | /games/{game_id}/toggle-dice | Toggle dice locks              | GameService        |
| POST    | /games/{game_id}/select-field | Select scorecard field          | GameService        |
| POST    | /games/{game_id}/end  | End game prematurely                  | GameService        |

### Authentication Flow

- For `/lobbies` and `/lobbies/join` (POST): If no JWT cookie present, creates guest token via AuthService and sets httpOnly cookie in response.
- For all other endpoints: Requires valid JWT cookie; validates via AuthService.
- Guest tokens are created automatically for lobby operations without prior authentication.

### GET /healthcheck

Response:
- 200 OK:
  ```json
  { "status": "ok" }
  ```

## Voraussetzungen und Installation

- Go-Version: 1.25.3 (siehe [go.mod](backend/services/APIGateway/go.mod))
- Optional: Docker (Multi-Stage Build, siehe [Dockerfile](backend/services/APIGateway/Dockerfile))
- Dependencies: AuthService, LobbyService, GameService running and accessible

Installation/Abhängigkeiten:
- Modulabhängigkeiten via `go mod tidy`/`go build` aufgelöst.

## Konfiguration

Umgebungsvariablen (siehe Beispieldatei [env.d/APIGateway.env.example](env.d/APIGateway.env.example)):

| Variable          | Typ     | Default | Erforderlich | Beschreibung                                      | Beispiel                    |
|-------------------|---------|---------|--------------|---------------------------------------------------|-----------------------------|
| PORT              | string  | 8080    | nein         | HTTP-Port                                         | `8080`                      |
| SERVICE_NAME      | string  | APIGateway | nein      | Service-Name in Logs                              | `APIGateway`                |
| LOG_LEVEL         | string  | info    | nein         | `debug`, `info`, `warn`, `error`                   | `info`                      |
| LOG_COLOR         | bool    | disabled| nein         | ANSI-Farb-Codierung für JSON-Logs                  | `true`                      |
| AUTH_SERVICE_URL  | string  | (leer)  | ja           | URL des AuthService                               | `http://auth-service:8081` |
| LOBBY_SERVICE_URL | string  | (leer)  | ja           | URL des LobbyService                              | `http://lobby-service:8083`|
| GAME_SERVICE_URL  | string  | (leer)  | ja           | URL des GameService                               | `http://game-service:8082` |
| COOKIE_DOMAIN     | string  | (leer)  | nein         | Domain für JWT-Cookie                             | `knuffelgame.example.com`  |
| COOKIE_SECURE     | bool    | false   | nein         | Secure-Flag für Cookie                            | `true`                      |
| COOKIE_SAME_SITE  | string  | Lax     | nein         | SameSite für Cookie (`Strict`, `Lax`, `None`)     | `Lax`                       |

Hinweise:
- Service-URLs müssen vollständige HTTP-URLs sein (inkl. Port).
- Cookie-Einstellungen für Produktion anpassen (Secure, Domain, SameSite).

Beispiel `.env`:
```
PORT=8080
SERVICE_NAME=APIGateway
LOG_LEVEL=info
LOG_COLOR=true
AUTH_SERVICE_URL=http://auth-service:8081
LOBBY_SERVICE_URL=http://lobby-service:8083
GAME_SERVICE_URL=http://game-service:8082
COOKIE_DOMAIN=localhost
COOKIE_SECURE=false
COOKIE_SAME_SITE=Lax
```

## Lokale Entwicklung und Start

Direkt mit Go:
```bash
go run ./cmd/APIGateway
```

Docker (Multi-Stage):
```bash
docker build -t apigateway ./backend/services/APIGateway
docker run --rm -p 8080:8080 --env-file ./env.d/APIGateway.env.example apigateway
```

Hinweis: Container definiert Healthcheck; prüfe abhängige Services.

## Migration/Seed

- Keine Datenbank, keine Migrationen/Seeds notwendig.

## Testen, Linting, Build und Release

Unit-Tests:
```bash
go test ./...
```

OpenAPI-Validierung (Spectral):
```bash
npx -y @stoplight/spectral-cli lint backend/services/APIGateway/openapi.yaml
```

Build:
```bash
go build -o apigateway ./cmd/APIGateway
```

Release:
- Üblicherweise per Container-Image basierend auf Dockerfile.

## Health- und Readiness-Checks

- Liveness: `GET /healthcheck` → 200 + `{"status":"ok"}`
- Keine separate Readiness-Probe; Infrastruktur kann `/healthcheck` nutzen.

## Observability & Logging

Logger-Middleware ([backend/libs/logger/middleware.go](backend/libs/logger/middleware.go)) loggt für jede Anfrage:
- http.method, http.path, http.status, http.duration_ms, http.request_id, http.remote_ip, http.user_agent

Weitere Details:
- `X-Request-ID` übernommen oder generiert
- Auth-Workflows loggen kontextuell (z. B. Token-Erzeugung/-Validierung)
- Debug-Logs bei Auth-Fehlern möglich (abhängig von `LOG_LEVEL`)

## Beispielaufrufe

cURL – Create Lobby (erstellt automatisch Guest-Token):
```bash
curl -s -X POST 'http://localhost:8080/lobbies' \
  -H 'Content-Type: application/json' \
  -d '{"username":"Alice"}'
```

cURL – Join Lobby (verwendet Cookie aus vorherigem Call):
```bash
TOKEN=$(curl -s -X POST 'http://localhost:8080/lobbies' -H 'Content-Type: application/json' -d '{"username":"Alice"}' -c cookies.txt | jq -r .token 2>/dev/null || echo "")
curl -s -X POST 'http://localhost:8080/lobbies/join' \
  -H 'Content-Type: application/json' \
  -b cookies.txt \
  -d '{"join_code":"ABC123"}'
```

cURL – Get Game State:
```bash
curl -s "http://localhost:8080/games/gam_xyz789" -b cookies.txt
```

Go (net/http) – Create Lobby:
```go
package main

import (
  "bytes"
  "encoding/json"
  "fmt"
  "net/http"
)

func main() {
  body := []byte(`{"username":"Alice"}`)
  resp, _ := http.Post("http://localhost:8080/lobbies", "application/json", bytes.NewReader(body))
  var res map[string]interface{}
  json.NewDecoder(resp.Body).Decode(&res)
  fmt.Println(res)
}
```

## Bekannte Stolpersteine & Troubleshooting

Häufige Fehlerbilder:
- 401 Unauthorized: Fehlendes oder ungültiges JWT-Cookie
- 500 Internal Server Error: AuthService/LobbyService/GameService nicht erreichbar
- Cookie nicht gesetzt: Prüfe COOKIE_DOMAIN und COOKIE_SECURE Einstellungen

Weitere Hinweise:
- CORS-Einstellungen für Produktion anpassen (AllowedOrigins)
- Proxy-Fehler: Prüfe Service-URLs und Netzwerk-Konnektivität
- Token-Ablauf: Guest-Tokens haben 24h Gültigkeit

## Bekannte Abweichungen

1) Cookie-Handling:
- Implementierung setzt Cookie nur bei erfolgreicher Lobby-Erstellung/Join; OpenAPI beschreibt dies nicht explizit.
- Remediation: OpenAPI um Cookie-Response-Details erweitern.

2) Auth-Workflow:
- Nur für Lobby-Erstellung/Join wird automatisch Guest-Token erstellt; andere Endpunkte erfordern vorhandenes Cookie.
- Remediation: Konsistent mit OpenAPI-Security-Schemes.

3) Header-Injection:
- X-User-ID und X-Username werden immer gesetzt, wenn Auth erfolgreich; Downstream-Services erwarten dies.
- Remediation: Dokumentation in abhängigen Services prüfen.

## Verweis auf OpenAPI

- Spezifikation: [backend/services/APIGateway/openapi.yaml](backend/services/APIGateway/openapi.yaml)
- Version: OpenAPI 3.0.3
- Validierung: Spectral (fehlerfrei)
  ```bash
  npx -y @stoplight/spectral-cli lint backend/services/APIGateway/openapi.yaml
  ```

## Konsistenz der Terminologie und Versionen

- README und OpenAPI sind auf OpenAPI 3.0.3 abgestimmt.
- Terminologie (JWT, Cookies, Proxying, Statuscodes) entspricht der Implementierung.