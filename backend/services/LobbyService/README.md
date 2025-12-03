# LobbyService

Der LobbyService verwaltet Spiel-Lobbys: Erstellung, Beitritt, Abruf von Details sowie Spielerverwaltung.

Verweise:
- Router: [backend/services/LobbyService/internal/router.go](backend/services/LobbyService/internal/router.go)
- OpenAPI: [backend/services/LobbyService/openapi.yaml](backend/services/LobbyService/openapi.yaml)
- Haupteinstiegspunkt: [backend/services/LobbyService/cmd/LobbyService/main.go](backend/services/LobbyService/cmd/LobbyService/main.go)

## Architekturüberblick

- Router: Chi-Router mit Request-Logging via [logger.ChiMiddleware()](backend/libs/logger/middleware.go:19). Konstruktion des Routers über [router.New()](backend/services/LobbyService/internal/router.go:15).
- Healthcheck: Mount via [healthcheck.Mount()](backend/libs/healthcheck/healthcheck.go:11) auf GET /healthcheck (JSON {"status":"ok"}).
- AuthN: Header-basierte Identität über [auth.AuthMiddleware](backend/libs/auth/auth.go:63) (X-User-ID, X-Username) für /lobbies-Gruppe.
- Guards/Authorizers: [handlers.RequireLobbyMember()](backend/services/LobbyService/internal/handlers/authorizers.go:16) und [handlers.RequireLobbyLeader()](backend/services/LobbyService/internal/handlers/authorizers.go:82).
- Handler:
  - [handlers.CreateLobbyHandler()](backend/services/LobbyService/internal/handlers/create_lobby.go:23) – POST /lobbies
  - [handlers.JoinLobbyHandler()](backend/services/LobbyService/internal/handlers/join_lobby.go:24) – POST /lobbies/join
  - [handlers.GetLobbyHandler()](backend/services/LobbyService/internal/handlers/get_lobby.go:20) – GET /lobbies/{lobby_id}
  - [handlers.GetLobbyInternalHandler()](backend/services/LobbyService/internal/handlers/get_lobby_internal.go:18) – GET /internal/lobbies/{lobby_id}
  - [handlers.KickPlayerHandler()](backend/services/LobbyService/internal/handlers/kick_player.go:22) – POST /lobbies/{lobby_id}/kick
  - [handlers.UpdatePlayerActiveStatusHandler()](backend/services/LobbyService/internal/handlers/update_player_active_status.go:21) – PUT /internal/lobbies/{lobby_id}/players/{player_id}/active
- Validierung: Header- und UUID-Formatprüfungen in Handlern; Join-Code-Länge exakt 6 (A–Z, 0–9).
- Repository/DB: PostgreSQL-Implementierung in [backend/services/LobbyService/internal/repository/postgres.go](backend/services/LobbyService/internal/repository/postgres.go) mit Transaktionen.
- Join-Code: Generator [joincode.Generator](backend/services/LobbyService/internal/joincode/generator.go) erzeugt eindeutige 6-stellige Codes (Großbuchstaben/Ziffern).

## Endpunkte

Basis-URL: http://localhost:8083

- POST /lobbies
  - Headers (erforderlich): X-User-ID (uuid), X-Username (string, max 20)
  - 201: CreateLobbyResponse; 400, 500
- POST /lobbies/join
  - Headers (erforderlich): X-User-ID (uuid), X-Username (string, max 20)
  - Body: { "join_code": "ABC123" } (6 Zeichen)
  - 200: LobbyDetailResponse; 400, 404, 409, 500
- GET /lobbies/{lobby_id}
  - Headers (erforderlich): X-User-ID (uuid), X-Username (string)
  - 200: LobbyDetailResponse; 400, 401, 403, 404, 500
- POST /lobbies/{lobby_id}/kick
  - Headers (erforderlich): X-User-ID (uuid), X-Username (string)
  - Body: { "target_user_id": "uuid" }
  - 204; 400, 403, 404, 500
- PUT /internal/lobbies/{lobby_id}/players/{player_id}/active
  - Interner Endpunkt (keine Auth-Middleware auf Router-Ebene)
  - Body: { "is_active": true|false }
  - 204; 400, 404, 500
- GET /internal/lobbies/{lobby_id}
  - Interner Endpunkt (keine Auth-Middleware auf Router-Ebene)
  - 200: LobbyDetailResponse; 400, 404, 500

Hinweise:
- Keine Pagination-, Filter- oder Sortier-Parameter vorhanden.
- Keine Rate-Limits oder Idempotenz-Header implementiert. Create-Flow ist bzgl. Nutzeranlage idempotent (ON CONFLICT DO NOTHING).

## Datenmodelle

- Lobby: id (uuid), join_code (char[6], unique), leader_id (uuid), status (waiting|running|finished), created_at, updated_at. Siehe [models.Lobby](backend/services/LobbyService/internal/models/models.go:16) und Migration [00001_create_schema.sql](backend/services/LobbyService/internal/db/migrations/00001_create_schema.sql).
- Player: id (uuid), lobby_id (uuid), user_id (uuid), joined_at (ts), is_active (bool), left_at (nullable). Siehe [models.Player](backend/services/LobbyService/internal/models/models.go:26).
- Responses: [models.CreateLobbyResponse](backend/services/LobbyService/internal/models/models.go:53), [models.LobbyDetailResponse](backend/services/LobbyService/internal/models/models.go:63), PlayerInfo.

## Voraussetzungen und Installation

- Go: Version gemäß go.mod / Dockerfile (1.25.3). Siehe [backend/services/LobbyService/go.mod](backend/services/LobbyService/go.mod) und [backend/services/LobbyService/Dockerfile](backend/services/LobbyService/Dockerfile).
- PostgreSQL-Datenbank erreichbar.
- Optional: Docker, curl, Spectral (OpenAPI Linter).

Installation von Spectral (optional, lokal):
```bash
npm i -g @stoplight/spectral-cli
```

## Konfiguration (Environment)

Quelle/Defaults: [pkg/config/config.go](backend/services/LobbyService/pkg/config/config.go), [env.d/LobbyService.env.example](env.d/LobbyService.env.example)

| Variable | Typ | Default | Beispiel |
|---|---|---|---|
| LOG_COLOR | bool | true | true |
| SERVICE_NAME | string | LobbyService | LobbyService |
| PORT | string | 8083 | 8083 |
| DATABASE_HOST | string | Postgres | Postgres |
| DATABASE_PORT | string | 5432 | 5432 |
| DATABASE_USER | string | lobby | lobby |
| DATABASE_PASSWORD | string | secure | secure |
| DATABASE_NAME | string | lobby | lobby |
| DATABASE_SSLMODE | string | disable | disable |

## Lokale Entwicklung und Start

Direktstart (ohne Docker):
```bash
go run ./cmd/LobbyService
```
Alternativ expliziter Pfad: [main.go](backend/services/LobbyService/cmd/LobbyService/main.go)

Docker:
```bash
docker build -t lobby-service .
docker run -p 8083:8083 --env-file env.d/LobbyService.env.example lobby-service
```

## Migrationen

- Migrations sind in die Binary eingebettet und werden beim Start automatisch ausgeführt. Siehe [internal/db/migrations.go](backend/services/LobbyService/internal/db/migrations.go) und SQL [00001_create_schema.sql](backend/services/LobbyService/internal/db/migrations/00001_create_schema.sql).
- Für manuelle Entwicklung/Debug siehe Hinweise in [internal/db/migrations/README.md](backend/services/LobbyService/internal/db/migrations/README.md).

## Testen, Linting, Build & Release

- Tests: ```go test ./...```
- Build: ```go build ./cmd/LobbyService```
- OpenAPI Lint: ```spectral lint backend/services/LobbyService/openapi.yaml``` (keine Errors erwartet)

## Health- und Readiness-Checks

- GET /healthcheck → 200, Body: {"status":"ok"} (siehe [healthcheck.Mount()](backend/libs/healthcheck/healthcheck.go:11)).

## Observability & Logging

- Request-Logging via [logger.ChiMiddleware()](backend/libs/logger/middleware.go:19).
- Felder: method, path, status, duration_ms, request_id, remote_ip, user_agent.

## Beispielaufrufe (curl)

Create Lobby:
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

Get Lobby:
```bash
curl -sS "http://localhost:8083/lobbies/123e4567-e89b-12d3-a456-426614174000" \
  -H "X-User-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -H "X-Username: Alice"
```

Kick Player (Leader):
```bash
curl -sS -X POST "http://localhost:8083/lobbies/123e4567-e89b-12d3-a456-426614174000/kick" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -H "X-Username: Leader" \
  -d '{"target_user_id":"789e4567-e89b-12d3-a456-426614174111"}' -i
```

Update Player Active (internal):
```bash
curl -sS -X PUT "http://localhost:8083/internal/lobbies/123e4567-e89b-12d3-a456-426614174000/players/789e4567-e89b-12d3-a456-426614174111/active" \
  -H "Content-Type: application/json" \
  -d '{"is_active":false}' -i
```

Get Lobby Internal:
```bash
curl -sS "http://localhost:8083/internal/lobbies/123e4567-e89b-12d3-a456-426614174000" -i
```

## Bekannte Stolpersteine & Troubleshooting

- 400 Bad Request bei fehlenden/ungültigen Headers (X-User-ID muss UUID sein).
- 403 Forbidden bei GET /lobbies/{id}, wenn aufrufender Nutzer kein Mitglied ist.
- 409 Conflict beim Join: Lobby voll (6 aktive Spieler), bereits Mitglied, oder Status ≠ waiting.
- DB-Verbindungsfehler: prüfen Sie Konfiguration (Host, Port, Credentials).

## Bekannte Abweichungen

- Statuswerte: README (alt) erwähnte "in_game" und "closed". Implementierung nutzt Konstanten "waiting", "running", "finished" (siehe [models](backend/services/LobbyService/internal/models/models.go:37)). Remediation: Terminologie in abhängigen Komponenten auf "running" anpassen oder optional zusätzlichen Status ergänzen.
- Interner Pfad in Tests: Unit-Tests rufen Handler direkt mit Test-URLs (z. B. ohne /internal oder mit "/active-status"). Der tatsächlich gemountete Pfad lautet PUT /internal/lobbies/{lobby_id}/players/{player_id}/active (siehe [router.go](backend/services/LobbyService/internal/router.go:26)). Remediation: Tests ggf. aktualisieren, falls Router-Integration getestet wird.

## Verweis auf OpenAPI

- Spezifikation: [backend/services/LobbyService/openapi.yaml](backend/services/LobbyService/openapi.yaml) (OpenAPI 3.1.0, per Spectral validiert).
- Lint: ```spectral lint backend/services/LobbyService/openapi.yaml```

## Lizenz

Keine explizite Lizenzangabe in diesem Service. Falls benötigt, bitte im Repository-Wurzelverzeichnis pflegen.
