# httpx Bibliothek

## Übersicht und Zweck

Hilfsbibliothek für HTTP-Handler in Go-Services. Stellt konsistente JSON-Antworten, ein standardisiertes Fehlerobjekt und kleine Komfortfunktionen bereit. Verwendet ausschließlich die Standardbibliothek.

## Exportoberfläche und API-Referenz

- Typen:
  - [httpx.ErrorPayload](backend/libs/httpx/httpx.go:17) – Standardisiertes Fehlerobjekt: { error, message, details? }
  - [httpx.LoggerProvider](backend/libs/httpx/httpx.go:11) – Interface zur Bereitstellung eines Request-Loggers (optional; derzeit nicht von den Funktionen verwendet)

- Response-Funktionen:
  - [httpx.WriteJSON()](backend/libs/httpx/httpx.go:25) — Signatur: func WriteJSON(w http.ResponseWriter, status int, payload interface{}, log *slog.Logger)
    - Setzt Header: Content-Type: application/json
    - Schreibt Statuscode und serialisiert payload als JSON (encoding/json).
    - Bei Encoding-Fehler: log.Error(...); Status kann nicht mehr geändert werden; kein Fallback-Body.
  - [httpx.WriteError()](backend/libs/httpx/httpx.go:37) — Signatur: func WriteError(w http.ResponseWriter, status int, code, message string, details map[string]interface{}, log *slog.Logger)
    - Schreibt ErrorPayload mit gegebenem HTTP-Status.
  - [httpx.WriteBadRequest()](backend/libs/httpx/httpx.go:42) — 400 Bad Request; error="bad_request"
  - [httpx.WriteUnauthorized()](backend/libs/httpx/httpx.go:47) — 401 Unauthorized; error="unauthorized" (ohne details)
  - [httpx.WriteForbidden()](backend/libs/httpx/httpx.go:52) — 403 Forbidden; error="forbidden"
  - [httpx.WriteNotFound()](backend/libs/httpx/httpx.go:57) — 404 Not Found; error="not_found"
  - [httpx.WriteInternalError()](backend/libs/httpx/httpx.go:62) — 500 Internal Server Error; error="internal_error" (mit optionalen details)
  - [httpx.WriteNoContent()](backend/libs/httpx/httpx.go:67) — 204 No Content (kein Body, kein Content-Type gesetzt)

- Request-Funktionen:
  - [httpx.DecodeJSON()](backend/libs/httpx/httpx.go:73) — Signatur: func DecodeJSON(r *http.Request, target interface{}) error
    - Dekodiert den Request-Body nach target via encoding/json Decoder.
    - Leerer Body führt zu einem Decode-Fehler (io.EOF); bei r.Body=nil würde ein leeres Objekt {} unmarshaled werden.
    - DisallowUnknownFields wird NICHT gesetzt; unbekannte Felder werden akzeptiert.
    - Es gibt KEIN Größenlimit im Decoder; Services müssen Limits selbst setzen (z. B. http.MaxBytesReader).

### Fehlerformat und Status-Mapping

ErrorPayload-Form:

```json
{
  "error": "<code>",
  "message": "<beschreibung>",
  "details": { /* optional, beliebige Schlüssel/Werte */ }
}
```

Konventionelle Zuordnung der Komfortfunktionen:
- 400: [httpx.WriteBadRequest()](backend/libs/httpx/httpx.go:42) → error="bad_request"
- 401: [httpx.WriteUnauthorized()](backend/libs/httpx/httpx.go:47) → error="unauthorized"
- 403: [httpx.WriteForbidden()](backend/libs/httpx/httpx.go:52) → error="forbidden"
- 404: [httpx.WriteNotFound()](backend/libs/httpx/httpx.go:57) → error="not_found"
- 500: [httpx.WriteInternalError()](backend/libs/httpx/httpx.go:62) → error="internal_error"

Für weitere Status (z. B. 409 Conflict) verwenden Services [httpx.WriteError()](backend/libs/httpx/httpx.go:37) mit eigenen Fehlercodes (siehe Beispiele).

## Unterstützte Laufzeitumgebungen

- Go gemäß [go.mod](backend/libs/httpx/go.mod): 1.25.3

## Abhängigkeiten

- Externe Module: keine (nur Standardbibliothek: [encoding/json](backend/libs/httpx/httpx.go:4), [net/http](backend/libs/httpx/httpx.go:6), [log/slog](backend/libs/httpx/httpx.go:5))
- Zusammenspiel: Logger-Middleware stellt einen [*slog.Logger](backend/libs/logger/middleware.go) über den Request-Context bereit (siehe Services); die httpx-Funktionen akzeptieren den Logger explizit als Parameter.

## Konfiguration

- Keine Umgebungsvariablen erforderlich.
- Keine eingebauten Limits im Paket. Empfohlen:
  - Request-Body begrenzen (z. B. 1MB) via http.MaxBytesReader im Handler.
  - JSON-Decoder auf strikten Modus setzen (DisallowUnknownFields), falls gewünscht.

## Verwendung (Praxisbeispiele)

- AuthService

  - Erzeugt konsistente Fehlercodes wie "invalid_request" und "token_generation_failed":

  ```go
  // Auszug aus Handler
  // invalid_request beim Decode/Validation
  // token_generation_failed bei Fehler im JWT-Generator
  ```

  Referenzen: [create_token.go (invalid_request)](backend/services/AuthService/internal/handlers/create_token.go:27), [create_token.go (Validation failed)](backend/services/AuthService/internal/handlers/create_token.go:33), [create_token.go (token_generation_failed)](backend/services/AuthService/internal/handlers/create_token.go:40), [create_token.go (WriteJSON)](backend/services/AuthService/internal/handlers/create_token.go:44)

  Beispielantworten:

  - 400 invalid_request

    ```json
    {
      "error": "invalid_request",
      "message": "Invalid JSON body",
      "details": { "detail": "<parse error>" }
    }
    ```

  - 500 token_generation_failed

    ```json
    {
      "error": "token_generation_failed",
      "message": "Failed to generate JWT token",
      "details": { "detail": "<generator error>" }
    }
    ```

- LobbyService

  - Verwendet 409-Conflicts über [httpx.WriteError()](backend/libs/httpx/httpx.go:37) mit service-spezifischen Codes:
    - "lobby_not_joinable" bei Status ≠ waiting ([join_lobby.go](backend/services/LobbyService/internal/handlers/join_lobby.go:85))
    - "lobby_full" bei Kapazitätsgrenze ([join_lobby.go](backend/services/LobbyService/internal/handlers/join_lobby.go:99))
    - "already_in_lobby" bei Doppelbeitritt ([join_lobby.go](backend/services/LobbyService/internal/handlers/join_lobby.go:113))
  - Autorisierungsfehler 401/403 via Komfortfunktionen ([authorizers.go](backend/services/LobbyService/internal/handlers/authorizers.go)).

  Beispielantwort (409 lobby_full):

  ```json
  {
    "error": "lobby_full",
    "message": "Lobby has reached maximum capacity (6 players)"
  }
  ```

- SSEService

  - Interner Test-Stub nutzt [httpx.WriteJSON()](backend/libs/httpx/httpx.go:25) für eine einfache OK-Antwort ([router.go](backend/services/SSEService/internal/router.go:31)).

## Performance-Hinweise

- Header-Setzung erfolgt vor dem Encoding; [httpx.WriteJSON()](backend/libs/httpx/httpx.go:25) nutzt encoding/json ohne zusätzliche Allokationsoptimierungen.
- Logging bei Encoding-Fehlern kann in Hotpaths vermieden werden, indem valide Payloads sichergestellt werden.
- Für hohen Durchsatz: Body-Limits und strikte Decoder im Handler setzen; Error-Details klein halten.

## Versionierungskompatibilität

- SemVer innerhalb des Monorepos. Öffentliche API-Oberfläche umfasst die in diesem Dokument referenzierten Symbole.
- Änderungen an [httpx.ErrorPayload](backend/libs/httpx/httpx.go:17) oder den Komfortfunktionen gelten als breaking.

## Bekannte Stolpersteine & Troubleshooting

- Falscher Content-Type auf Client-Seite: Antworten sind JSON; Client sollte Accept: application/json verwenden.
- Unbekannte Felder im Request-Body werden von [httpx.DecodeJSON()](backend/libs/httpx/httpx.go:73) akzeptiert: Bei Bedarf im Service [Decoder.DisallowUnknownFields()](backend/services/AuthService/internal/handlers/create_token.go:24) setzen.
- Große Payloads: Größe explizit begrenzen (z. B. 1MB) wie im AuthService mit [http.MaxBytesReader](backend/services/AuthService/internal/handlers/create_token.go:20).
- Statuscode-Zuordnung im aufrufenden Code prüfen (z. B. 409-Konflikte im LobbyService manuell via [httpx.WriteError()](backend/libs/httpx/httpx.go:37)).

## Bekannte Abweichungen

- Striktes Decoding: Ältere Beschreibungen erwähnten teilweise "DisallowUnknownFields" als Standard. Tatsächlich setzt [httpx.DecodeJSON()](backend/libs/httpx/httpx.go:73) dies NICHT. Remediation: Optional neue Helferfunktion "DecodeJSONStrict(r, target)" mit DisallowUnknownFields und Beispielnutzung in Services.
- Größe des Request-Bodys: Es gibt kein zentrales Limit im Paket. Services (z. B. AuthService) setzen das Limit selbst. Remediation: Optional Helfer "LimitBody(w, r, n)" oder Dokumentation eines empfohlenen Defaults (1MB).
- 409 Conflict Convenience: Es existiert keine [httpx.WriteConflict()](backend/libs/httpx/httpx.go:1). Services verwenden [httpx.WriteError()](backend/libs/httpx/httpx.go:37). Remediation: Optional [httpx.WriteConflict()](backend/libs/httpx/httpx.go:1) ergänzen (status=409, error="conflict").
- Details-Parameter bei 401/403/404: Komfortfunktionen setzen kein details. Remediation: Optional Überladungen mit details map hinzufügen oder konsequent [httpx.WriteError()](backend/libs/httpx/httpx.go:37) nutzen.

## Lizenz/Metadaten

- Modul: [backend/libs/httpx/go.mod](backend/libs/httpx/go.mod)
