# AuthService

JWT issuing and validation microservice for KnuffelGame. Provides internal endpoints to create guest user tokens and validate existing tokens. Uses HS256 signed JSON Web Tokens with a shared secret.

## Features
- Issue 24h expiry JWTs for (guest) users
- Validate tokens (signature, expiry, issuer, required claims)
- Structured JSON logging (via shared `logger` lib) with request middleware
- Lightweight healthcheck endpoint (`GET /healthcheck` -> `200` / body `1`)
- Input validation using `go-playground/validator`

## Endpoints
Base path: service root (e.g. `http://auth-service:8081`).

| Method | Path                | Description                  |
|--------|---------------------|------------------------------|
| GET    | /healthcheck        | Liveness check               |
| POST   | /internal/create    | Create guest JWT token       |
| POST   | /internal/validate  | Validate a JWT token         |

### POST /internal/create
Request JSON:
```
{
  "user_id": "<uuid4>",
  "username": "<3-20 chars, letters/digits/spaces>"
}
```
Response 200 JSON:
```
{ "token": "<jwt>" }
```
Validation errors -> 400 with error schema:
```
{
  "error": "invalid_request",
  "message": "Validation failed",
  "details": { "fields": { "Username": "min" } }
}
```
Possible error codes:
- `invalid_request` (malformed JSON / validation failure)
- `token_generation_failed` (internal signing issues)

### POST /internal/validate
Request JSON:
```
{ "token": "<jwt>" }
```
Success 200 JSON:
```
{
  "valid": true,
  "user_id": "<uuid4>",
  "username": "<username>",
  "is_guest": true
}
```
Failure JSON (status 400 or 401):
```
{ "valid": false, "error": "<reason>" }
```
Error reasons returned:
- `invalid format` (malformed token / missing claims)
- `invalid signature`
- `token expired`
- `invalid issuer`
- `missing claims`

Status mapping:
- 400: malformed JSON body or structurally invalid token (`invalid format`)
- 401: signature/expiry/issuer/claims issues (except structural format)

## JWT Details
- Algorithm: HS256
- Issuer: `knuffel-auth-service`
- Claims:
  - `sub` (Subject): user id (UUID4)
  - `name`: username
  - `guest`: boolean (always `true` for now)
  - `iat`: issued at (unix)
  - `exp`: expires at (unix, 24h)
  - `iss`: issuer (see above)

Tokens must be validated with the same shared secret and expected issuer.

## Configuration
Environment variables:

| Variable      | Default                     | Required   | Description                                  |
|---------------|-----------------------------|------------|----------------------------------------------|
| PORT          | 8081                        | no         | Port to bind HTTP server                     |
| JWT_SECRET    | (empty)                     | yes*       | HS256 signing/validation secret (>=32 chars) |
| SERVICE_NAME  | AuthService (auto if empty) | no         | Name injected into logs                      |
| LOG_LEVEL     | info                        | no         | debug, info, warn, error (from logger lib)   |
| LOG_COLOR     | disabled                    | no         | Enable ANSI color in JSON logs               |

`JWT_SECRET` must be set; if missing or <32 chars, service logs warnings and token operations fail.

Example `.env`:
```
JWT_SECRET=change_me_32+chars_long_secret
PORT=8081
SERVICE_NAME=AuthService
LOG_LEVEL=info
LOG_COLOR=true
```

## Test create + validate:
```bash
curl -s -X POST localhost:8081/internal/create \
  -H 'Content-Type: application/json' \
  -d '{"user_id":"11111111-1111-4111-8111-111111111111","username":"GuestUser"}' | jq .token
# Use output token
TOKEN=$(curl -s -X POST localhost:8081/internal/create -H 'Content-Type: application/json' -d '{"user_id":"11111111-1111-4111-8111-111111111111","username":"GuestUser"}' | jq -r .token)
curl -s -X POST localhost:8081/internal/validate -H 'Content-Type: application/json' -d '{"token":"'$TOKEN'"}' | jq .
```

## Logging
Uses shared `logger` library. Each request logs on completion at INFO with attributes:
- http.method, http.path, http.status, http.duration_ms, http.request_id, http.remote_ip, http.user_agent
Handlers add contextual fields (e.g. `handler.action`). Token generation/validation logs DEBUG lines when enabled.

## Validation Rules
- `user_id`: UUID4
- `username`: 3-20 chars, letters/digits/spaces, must include at least one alphanumeric
- `token`: must match JWT structural regex three segments Base64URL

## Project Layout
```
cmd/AuthService/main.go      # Bootstrap
internal/router.go           # Chi router & route setup
internal/handlers            # HTTP handlers create/validate
internal/jwt                 # Generator & Validator
internal/models              # Request/response models & validation
pkg/config/config.go         # Env config loader
```

## Extending
Planned / suggested enhancements:
- Non-guest tokens with role claims
- Secret rotation support (multiple valid signing keys)
- Rate limiting on create endpoint
- OpenAPI spec generation

## Testing
Run unit tests for handlers/jwt/models:
```bash
go test ./...
```
(From service module root.)

## Security Notes
- Keep `JWT_SECRET` out of version control; use environment injection.
- Minimum 32 characters recommended (service warns if shorter).
- HS256 symmetric secret must match across services performing validation.
