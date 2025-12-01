# Logger

Übersicht und Zweck
- Strukturierte JSON-Logs für Services mit kontextbezogenem Logger und HTTP-Request-Middleware.
- Ziel: einheitliche, maschinenlesbare Logs mit Korrelation über Request-ID sowie einfache Verwendung in chi-Handlern.

Exportoberfläche und API-Referenz
- Konstruktion/Konfiguration
  - [logger.New()](backend/libs/logger/logger.go:15) – erzeugt einen neuen Logger gemäß Optionen und setzt ihn als Default.
  - [logger.Default()](backend/libs/logger/logger.go:23) – liefert den Paket-Default-Logger (lazy init).
  - [logger.FromEnv()](backend/libs/logger/logger.go:37) – liest Konfiguration aus Environment (LOG_LEVEL, SERVICE_NAME, LOG_COLOR).
- Middleware
  - [logger.ChiMiddleware()](backend/libs/logger/middleware.go:19) – chi-kompatible Middleware, die pro Request beim Abschluss auf INFO loggt. Der Logger wird via Context für Handler bereitgestellt.
    - Strukturierte Felder (unter der Gruppe "http"):
      - http.method, http.path, http.status, http.duration_ms, http.request_id, http.remote_ip, http.user_agent
    - Request-ID: Header X-Request-ID wird übernommen; fehlt er, wird eine zufällige 16-Byte Hex-ID generiert ([middleware](backend/libs/logger/middleware.go:45)).
    - Remote-IP: Präferenz X-Forwarded-For (erstes Element) > X-Real-IP > RemoteAddr ([middleware](backend/libs/logger/middleware.go:55)).
- Kontext-Hilfen
  - [logger.WithLogger()](backend/libs/logger/context.go:11) – injiziert Logger in einen context.Context.
  - [logger.Logger()](backend/libs/logger/context.go:16) – holt Logger aus dem Context (Fallback: Default).
- Optionen (Option-Pattern)
  - [logger.WithLevel()](backend/libs/logger/options.go:34) – Minimal-Level (slog.Level).
  - [logger.WithService()](backend/libs/logger/options.go:37) – Service-Name (Attribut "service").
  - [logger.WithColor()](backend/libs/logger/options.go:40) – ANSI-Farb-Ausgabe (pro Level).
  - [logger.WithAddSource()](backend/libs/logger/options.go:43) – Quelle file:line anhängen.
  - [logger.WithWriter()](backend/libs/logger/options.go:46) – Ziel-Writer (Default: os.Stderr).
- Interner Handler
  - Farb-/JSON-Handler implementiert in [colorJSONHandler](backend/libs/logger/handler.go:35) mit gruppierten Attributen, thread-sicherem Write und optionaler Farb-Ummantelung pro Zeile.

Unterstützte Laufzeitumgebungen
- Go-Version gemäß Modul: [go 1.25.3](backend/libs/logger/go.mod:3)

Abhängigkeiten
- Extern:
  - chi v5: [github.com/go-chi/chi/v5](backend/libs/logger/go.mod:5) (nur für ResponseWriter-Wrapper in der Middleware)
- Standardbibliothek:
  - log/slog, net/http, crypto/rand, encoding/hex, net, strings, time, sync, os, context, encoding/json, runtime, io
- Keine weiteren Env-Abhängigkeiten außer LOG_LEVEL, SERVICE_NAME, LOG_COLOR.

Konfiguration (Environment)
- [logger.FromEnv()](backend/libs/logger/logger.go:37) verarbeitet:
  - LOG_LEVEL (string) – unterstützte Werte: debug, info, warn, error. Default: info.
  - SERVICE_NAME (string) – optional; wird als Attribut "service" an jede Logzeile angefügt.
  - LOG_COLOR (bool/flag) – "1", "true", "yes", "on" aktivieren; "0", "false", "no", "off" deaktivieren. Default: deaktiviert.
- Beispiel:
  ```
  LOG_LEVEL=debug
  SERVICE_NAME=AuthService
  LOG_COLOR=on
  ```
- Hinweis: Services lesen diese Werte typischerweise aus ihren eigenen .env-Dateien und rufen dann [logger.FromEnv()](backend/libs/logger/logger.go:37) beim Start auf.

Verwendung (chi)
- Middleware-Einbindung (früh in der Kette):
  - Referenz: [logger.ChiMiddleware()](backend/libs/logger/middleware.go:19)
- Handler-Beispiel mit kontextuellem Logger und Gruppierung ("handler"):
  ```go
  import (
    "log/slog"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
  )

  func main() {
    // typischerweise: logger.FromEnv()
    log := logger.New(
      logger.WithService("ExampleService"),
      logger.WithLevel(slog.LevelInfo),
      logger.WithColor(false),
    )

    r := chi.NewRouter()
    r.Use(logger.ChiMiddleware(log))

    r.Get("/hello", func(w http.ResponseWriter, r *http.Request) {
      log := logger.Logger(r.Context()).
        WithGroup("handler").
        With(slog.String("action", "hello"))
      log.Info("handler invoked")
      w.WriteHeader(http.StatusOK)
      w.Write([]byte("hi"))
    })

    http.ListenAndServe(":8080", r)
  }
  ```
- Downstream-Verwendung in Services:
  - AuthService: [internal/router.go](backend/services/AuthService/internal/router.go:13), Handler mit Gruppierung: [create_token.go](backend/services/AuthService/internal/handlers/create_token.go:21)
  - LobbyService: [internal/router.go](backend/services/LobbyService/internal/router.go:15)
  - SSEService: [internal/router.go](backend/services/SSEService/internal/router.go:15)

Observability & Logging
- Emittierte HTTP-Attribute: http.method, http.path, http.status, http.duration_ms, http.request_id, http.remote_ip, http.user_agent ([logger.ChiMiddleware()](backend/libs/logger/middleware.go:19)).
- Korrelation:
  - request-id über Header X-Request-ID oder automatisch generiert ([middleware](backend/libs/logger/middleware.go:45)).
- Level-Steuerung:
  - Via LOG_LEVEL oder [logger.WithLevel()](backend/libs/logger/options.go:34).
- Farb-Output:
  - Via LOG_COLOR oder [logger.WithColor()](backend/libs/logger/options.go:40). Hinweis: Farbige Ausgabe ist mit ANSI-Sequenzen umhüllt und somit für strikte JSON-Parser nicht valide. In CI/Docker-Logs i. d. R. deaktivieren oder Filter einsetzen.
- Source-Attribut:
  - Optional via [logger.WithAddSource()](backend/libs/logger/options.go:43) (file:line).

Performance-Hinweise
- Geringer Overhead pro Request:
  - Zeitmessung (time.Now), ResponseWriter-Wrapper (chi middleware), ein strukturierter Log-Eintrag bei Abschluss.
  - Keine blockierenden Netzwerk-/Datei-Operationen außer dem Write auf den konfigurierten io.Writer (Mutex-geschützt).
- Reihenfolge-Empfehlung:
  - Logger möglichst früh registrieren (äußere Middleware), danach Auth/Guards, damit auch frühe Abbrüche vollständig geloggt werden. Siehe auch Services.

Versionierung/Kompatibilität
- Modulpfad: [backend/libs/logger](backend/libs/logger/go.mod)
- Semantische Stabilität:
  - Signatur von [logger.ChiMiddleware()](backend/libs/logger/middleware.go:19) sowie Feldnamen der HTTP-Gruppe gelten als stabil.
  - Interne Implementierungsdetails (z. B. Farb-Codes) können sich ohne API-Bruch ändern.

Bekannte Stolpersteine & Troubleshooting
- Fehlender X-Request-ID:
  - Verhalten: automatische Generierung einer 16-Byte Hex-ID. Falls Upstream/Proxy eigene IDs erzwingt, Header korrekt durchreichen lassen.
- Ungeeignete LOG_LEVEL-Werte:
  - Unbekannte Werte werden auf info gemappt ([parseLevel](backend/libs/logger/logger.go:44)). Erwartete Strings verwenden.
- Farbige Logs in nicht-TTY-Umgebungen:
  - LOG_COLOR deaktivieren, wenn strikte JSON-Verarbeitung notwendig ist. Alternativ ANSI-Codes entfernen.
- Remote-IP hinter Proxies:
  - Korrekte Setzung von X-Forwarded-For bzw. X-Real-IP sicherstellen, sonst wird RemoteAddr verwendet.

Bekannte Abweichungen
- Feldnamen in Service-READMEs:
  - Einige Service-Dokumente listen die HTTP-Felder ohne Gruppierungspräfix (z. B. "method" statt "http.method"), siehe z. B. LobbyService-README. Tatsächlich werden die Attribute unter der Gruppe "http" emittiert ([logger.ChiMiddleware()](backend/libs/logger/middleware.go:32)) und erscheinen im JSON als Objekt http.{...}.
  - Remediation: Service-READMEs auf die Präfix-Variante "http.<feld>" anpassen oder explizit dokumentieren, dass die Felder innerhalb der Gruppe "http" erscheinen.
- Zeilenangaben in Querverweisen:
  - In manchen READMEs können die referenzierten Zeilennummern der Middleware variieren. Maßgeblich ist die Implementierung in [backend/libs/logger/middleware.go](backend/libs/logger/middleware.go:19). Bei Änderungen Verweise aktualisieren.

Praktische Hinweise
- Verwendung in Tests:
  - Tests validieren u. a. Status- und Pfad-Attribute ([logger_test.go](backend/libs/logger/logger_test.go:28)).
- Zusammenspiel mit weiteren Libs:
  - httpx akzeptiert einen Logger explizit als Parameter; in der Regel wird [logger.Logger()](backend/libs/logger/context.go:16) genutzt, um in Handlern den kontextuellen Logger zu beziehen.

Lizenz/Metadaten
- Keine expliziten Lizenzangaben im Modul. Es gelten die Repository-Richtlinien.
