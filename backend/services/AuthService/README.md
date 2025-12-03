# AuthService

JWT-Ausstellungs- und Validierungsservice für KnuffelGame. Stellt interne HTTP-Endpunkte bereit, um Gast-Token (JWT) zu erzeugen und bestehende Tokens zu validieren. Verwendet HS256-signierte JSON Web Tokens mit gemeinsamem Geheimnis (JWT_SECRET).

## Übersicht und Zweck

- Ausgabe von Gast-JWTs mit 24h Gültigkeit
- Validierung von JWTs (Signatur, Ablauf, Issuer, erforderliche Claims)
- Einfache Liveness-Überprüfung über `/healthcheck`
- Strukturierte JSON-Logs per gemeinsamer Logger-Middleware

## Architekturüberblick

- Router: Chi-Router mit Request-Logging ([internal/router.go](backend/services/AuthService/internal/router.go))
- Handler:
  - POST `/internal/create` ([internal/handlers/create_token.go](backend/services/AuthService/internal/handlers/create_token.go))
  - POST `/internal/validate` ([internal/handlers/validate_token.go](backend/services/AuthService/internal/handlers/validate_token.go))
- Validierungsmodelle: ([internal/models/create_token.go](backend/services/AuthService/internal/models/create_token.go))
- JWT-Generator/Validator: ([internal/jwt/generator.go](backend/services/AuthService/internal/jwt/generator.go)), ([internal/jwt/validator.go](backend/services/AuthService/internal/jwt/validator.go))
- Middleware & Health:
  - Request-Logging: ([backend/libs/logger/middleware.go](backend/libs/logger/middleware.go))
  - Healthcheck: ([backend/libs/healthcheck/healthcheck.go](backend/libs/healthcheck/healthcheck.go))

## Endpunkte

Basis-URL: Service-Root (z. B. `http://auth-service:8081`)

| Methode | Pfad                | Beschreibung                    |
|--------:|---------------------|---------------------------------|
| GET     | /healthcheck        | Liveness-Check                  |
| POST    | /internal/create    | Gast-JWT erzeugen |
| POST    | /internal/validate  | JWT validieren                  |

### POST /internal/create

Request (JSON):
```json
{
  "username": "GuestUser"
}
```

Antworten:
- 200 OK:
  ```json
  {
    "token": "<jwt>",
    "username": "GuestUser",
    "user_id": "550e8400-e29b-41d4-a716-446655440000"
  }
  ```
- 400 Bad Request (ungültiges JSON oder Validierungsfehler):
  ```json
  {
    "error": "invalid_request",
    "message": "Validation failed",
    "details": { "fields": { "Username": "min" } }
  }
  ```
- 500 Internal Server Error (Signierung/Erzeugung fehlgeschlagen):
  ```json
  {
    "error": "token_generation_failed",
    "message": "Failed to generate JWT token"
  }
  ```

Validierungsregeln (Request):
- username: 3–20 Zeichen; erlaubte Zeichen: Buchstaben, Ziffern, Leerzeichen

Validierungsregeln (Response):
- username: 3–20 Zeichen; erlaubte Zeichen: Buchstaben, Ziffern, Leerzeichen
- user_id: UUID4

Header:
- Content-Type: `application/json` (erforderlich)
- Accept: `application/json` (empfohlen)

Rate-Limits & Idempotenz:
- Nicht implementiert

### POST /internal/validate

Request (JSON):
```json
{ "token": "<jwt>" }
```

Antworten:
- 200 OK (gültiges Token):
  ```json
  {
    "valid": true,
    "user_id": "usr_valid",
    "username": "Alice",
    "is_guest": true
  }
  ```
- 400 Bad Request (malformed JSON oder Token-Format ungültig):
  ```json
  { "valid": false, "error": "invalid format" }
  ```
- 401 Unauthorized (Signatur/Ablauf/Issuer/fehlende Claims):
  ```json
  { "valid": false, "error": "token expired" }
  ```
  Weitere Fehlergründe: `invalid signature`, `invalid issuer`, `missing claims`

Header:
- Content-Type: `application/json` (erforderlich)
- Accept: `application/json` (empfohlen)

Sicherheitsaspekte:
- Endpunkte sind intern und öffentlich erreichbar; kein Bearer-Auth erforderlich
- JWTs werden mit HS256 und gemeinsamem Geheimnis validiert

### GET /healthcheck

Antwort:
- 200 OK:
  ```json
  { "status": "ok" }
  ```

Hinweise:
- Keine separaten Readiness-Probes; Container-Health nutzt diesen Endpunkt

## JWT Details

- Algorithmus: HS256
- Issuer: `knuffel-auth-service`
- Claims:
  - `sub`: Benutzer-ID (Subject)
  - `name`: Benutzername
  - `guest`: boolean (Service vergibt aktuell stets Gast-Token)
  - `iat`: Ausgabezeitpunkt (unix)
  - `exp`: Ablaufzeitpunkt (unix, +24h)
  - `iss`: Issuer (siehe oben)

Tokens müssen mit demselben Geheimnis und erwarteten Issuer validiert werden.

## Voraussetzungen und Installation

- Go-Version: 1.25.3 (siehe [go.mod](backend/services/AuthService/go.mod))
- Optional: Docker (Multi-Stage Build, siehe [Dockerfile](backend/services/AuthService/Dockerfile))

Installation/Abhängigkeiten:
- Modulabhängigkeiten werden via `go mod tidy`/`go build` aufgelöst.

## Konfiguration

Umgebungsvariablen (siehe Beispieldatei [env.d/AuthService.env.example](env.d/AuthService.env.example)):

| Variable     | Typ     | Default    | Erforderlich | Beschreibung                                                | Beispiel                          |
|--------------|---------|------------|--------------|-------------------------------------------------------------|-----------------------------------|
| PORT         | string  | 8081       | nein         | HTTP-Port                                                   | `8081`                            |
| JWT_SECRET   | string  | (leer)     | ja*          | HS256-Secret (>=32 Zeichen empfohlen)                       | `change_me_32_plus_chars_secret`  |
| SERVICE_NAME | string  | AuthService| nein         | Service-Name in Logs (Fallback auf „AuthService“)          | `AuthService`                     |
| LOG_LEVEL    | string  | info       | nein         | `debug`, `info`, `warn`, `error`                            | `info`                            |
| LOG_COLOR    | bool    | disabled   | nein         | ANSI-Farb-Codierung für JSON-Logs                           | `true`                            |

Hinweise:
- Ist `JWT_SECRET` leer oder kürzer als 32 Zeichen, werden Warnungen geloggt und Token-Operationen schlagen fehl.

Beispiel `.env`:
```
PORT=8081
JWT_SECRET=change_me_32_plus_chars_secret_ABCDEFG123456
SERVICE_NAME=AuthService
LOG_LEVEL=info
LOG_COLOR=true
```

## Lokale Entwicklung und Start

Direkt mit Go:
```bash
go run ./cmd/AuthService
```

Docker (Multi-Stage):
```bash
docker build -t authservice ./backend/services/AuthService
docker run --rm -p 8081:8081 --env-file ./env.d/AuthService.env.example authservice
```

Hinweis: Der Container definiert einen Healthcheck:
- Siehe [Dockerfile](backend/services/AuthService/Dockerfile) mit `curl -f http://localhost:8081/healthcheck`

## Migration/Seed

- Keine Datenbank, keine Migrationen/Seeds notwendig.

## Testen, Linting, Build und Release

Unit-Tests:
```bash
go test ./...
```

OpenAPI-Validierung (Spectral):
```bash
# einmalig: .spectral.yaml liegt im Repo
npx -y @stoplight/spectral-cli lint backend/services/AuthService/openapi.yaml
```

Build:
```bash
go build -o authservice ./cmd/AuthService
```

Release:
- Üblicherweise per Container-Image basierend auf dem Dockerfile.

## Health- und Readiness-Checks

- Liveness: `GET /healthcheck` → 200 + `{"status":"ok"}`
- Keine separate Readiness-Probe; Infrastruktur kann `/healthcheck` nutzen.

## Observability & Logging

Die Logger-Middleware ([backend/libs/logger/middleware.go](backend/libs/logger/middleware.go)) loggt für jede Anfrage (INFO) folgende Attribute:
- http.method, http.path, http.status, http.duration_ms, http.request_id, http.remote_ip, http.user_agent

Weitere Details:
- `X-Request-ID` wird übernommen, sonst generiert
- Handler loggen kontextuell (z. B. `handler.action`)
- Debug-Logs bei Token-Erzeugung/-Validierung möglich (abhängig von `LOG_LEVEL`)

## Bekannte Stolpersteine & Troubleshooting

Häufige Fehlerbilder:
- 400 `invalid_request`: Ungültiges JSON oder Validierungsfehler im Request
- 400 `invalid format`: Token-String entspricht nicht der strukturellen JWT-Form
- 401 `token expired` / `invalid signature` / `invalid issuer` / `missing claims`: Kryptografisch oder semantisch ungültig

Weitere Hinweise:
- Zu kurzes oder fehlendes `JWT_SECRET` → Token-Erzeugung schlägt fehl
- Zeitdrift kann zu `token expired` führen; Serverzeit prüfen
- Issuer muss exakt `knuffel-auth-service` sein

## Bekannte Abweichungen

1) Username-Validierung:
- Beschreibung sagt „mindestens ein alphanumerisches Zeichen“, die aktuelle Implementierung prüft dies nicht strikt (Regex erlaubt nur erlaubte Zeichen, aber nicht zwingend ein alphanumerisches Zeichen).
- Remediation: Die benutzerdefinierte Validierung `usernameFmt` in ([internal/models/create_token.go](backend/services/AuthService/internal/models/create_token.go)) so erweitern, dass mindestens eine alphanumerische Rune enthalten ist (zusätzliche Prüfung im Validator-Callback).

2) Format der `user_id` im Validation-Ergebnis:
- Der Validator erzwingt kein UUID4-Format für `claims.Subject`; das OpenAPI-Dokument belässt `user_id` deshalb ohne `format: uuid`.
- Remediation: Entweder UUID4-Format im Validator ([internal/jwt/validator.go](backend/services/AuthService/internal/jwt/validator.go)) prüfen und bei Verletzung `missing claims`/`invalid format` liefern, oder die Producer sicherstellen, dass `sub` immer UUID4 ist.

3) Healthcheck-Dokumentation:
- Frühere Dokumentation erwähnte Körper `1`; tatsächlich liefert der Endpunkt `{"status":"ok"}`.
- Remediation: Dokumentation ist aktualisiert (dieses README); Implementierung ist korrekt.

4) Rate Limiting:
- Kein Rate-Limit auf `/internal/create` implementiert.
- Remediation: Optional eine Middleware für Rate-Limiting einführen (z. B. IP-/Request-ID-basiert) und Grenzwerte dokumentieren.

## Verweis auf OpenAPI

- Spezifikation: [backend/services/AuthService/openapi.yaml](backend/services/AuthService/openapi.yaml)
- Version: OpenAPI 3.1.0
- Validierung: Spectral (fehlerfrei)
  ```bash
  npx -y @stoplight/spectral-cli lint backend/services/AuthService/openapi.yaml
  ```

## Beispielaufrufe

cURL – Token erzeugen:
```bash
curl -s -X POST 'http://localhost:8081/internal/create' \
  -H 'Content-Type: application/json' \
  -d '{"username":"GuestUser"}'
```

cURL – Token validieren:
```bash
TOKEN=$(curl -s -X POST 'http://localhost:8081/internal/create' -H 'Content-Type: application/json' -d '{"username":"GuestUser"}' | jq -r .token)
curl -s -X POST 'http://localhost:8081/internal/validate' \
  -H 'Content-Type: application/json' \
  -d '{"token":"'$TOKEN'"}' | jq .
```

Go (net/http) – Token erzeugen und validieren:
```go
package main

import (
  "bytes"
  "encoding/json"
  "fmt"
  "net/http"
)

func main() {
  // Create
  createBody := []byte(`{"username":"GuestUser"}`)
  resp, _ := http.Post("http://localhost:8081/internal/create", "application/json", bytes.NewReader(createBody))
  var createRes map[string]string
  json.NewDecoder(resp.Body).Decode(&createRes)
  token := createRes["token"]

  // Validate
  validateReq := map[string]string{"token": token}
  buf, _ := json.Marshal(validateReq)
  resp2, _ := http.Post("http://localhost:8081/internal/validate", "application/json", bytes.NewReader(buf))
  var validateRes map[string]interface{}
  json.NewDecoder(resp2.Body).Decode(&validateRes)
  fmt.Println(validateRes)
}
```

## Konsistenz der Terminologie und Versionen

- README und OpenAPI sind auf OpenAPI 3.1.0 abgestimmt.
- Terminologie (Issuer, Claims, Endpunkte, Statuscodes) entspricht der Implementierung.
