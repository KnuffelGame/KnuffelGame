Auth middleware
===============

Purpose
-------
This small package provides a minimal authentication middleware that parses and validates the HTTP headers the API Gateway injects (by default `X-User-ID` and `X-Username`), normalizes them into a `User` value and stores it in the request context for downstream handlers.

Design goals
------------
- Keep responsibilities narrow: parse headers, validate `X-User-ID` as a UUID, and inject a typed `User` into `context.Context`.
- Avoid DB access or business authorization logic; those belong to service-specific authorizers.
- Be lightweight and easy to reuse across services inside the repo.

Public API
----------
- type User struct {
    ID uuid.UUID
    Username string
  }

- func FromContext(ctx context.Context) (User, bool)
  - Retrieve the `User` previously injected by the middleware.

- func NewAuthMiddleware(userHeader, usernameHeader string) func(http.Handler) http.Handler
  - Create a middleware that reads the specified header names and injects the `User`.

- var AuthMiddleware = NewAuthMiddleware(DefaultHeaderUserID, DefaultHeaderUsername)
  - Ready-to-use middleware configured with the default headers.

- const DefaultHeaderUserID = "X-User-ID"
- const DefaultHeaderUsername = "X-Username"

Usage
-----
1) Mount the middleware in your router for routes that require an authenticated user (example using `chi`):

```go
import (
  "github.com/KnuffelGame/KnuffelGame/backend/libs/auth"
  "github.com/go-chi/chi/v5"
)

r := chi.NewRouter()
// mount auth for lobby routes
r.Route("/lobbies", func(r chi.Router) {
    r.Use(auth.AuthMiddleware) // injects auth.User into context
    r.Post("/", handlers.CreateLobbyHandler(...))
    r.With(handlers.RequireLobbyMember(db)).Get("/{lobby_id}", handlers.GetLobbyHandler(db))
})
```

2) Read the user in a handler:

```go
u, ok := auth.FromContext(r.Context())
if !ok {
    // either return 401/400 or fall back to header parsing for transitional code
}
// use u.ID, u.Username
```

3) Custom header names

If your gateway uses different headers, construct a middleware with the desired header names:

```go
r.Use(auth.NewAuthMiddleware("X-My-ID", "X-My-Name"))
```

Behavior and error handling
---------------------------
- If the user header or username header is missing, the middleware writes a 400 Bad Request and does not call the next handler.
- If `X-User-ID` is present but not a valid UUID, the middleware writes a 400 Bad Request.
- The middleware logs warnings using the repository `libs/logger` package.
- The middleware purposefully does not decide authorization; that is left to service authorizers (DB checks, role checks, etc.).

Testing
-------
Unit tests for the `auth` package are included under `backend/libs/auth`. Run them with:

```bash
cd backend/libs/auth
go test ./...
```

Migration notes (adopting the library in a service)
--------------------------------------------------
- Add a `require` for `github.com/KnuffelGame/KnuffelGame/backend/libs/auth` in your service `go.mod` and a `replace` directive to the local path, e.g.:

```text
replace github.com/KnuffelGame/KnuffelGame/backend/libs/auth => ../../libs/auth
```

- Update your router to use `auth.AuthMiddleware` and update handlers to prefer `auth.FromContext`.
- During a transition you can keep handler-level header fallbacks to avoid breaking tests; after migrating handlers you can remove fallbacks.

Extending the library
---------------------
- The library intentionally avoids heavy dependencies. If you want:
  - make header names and error handling pluggable via functional options
  - accept a custom logger interface instead of importing `libs/logger`
  - add integration tests that run against a full dev environment

Support
-------
If you run into build problems when importing this package from a service, ensure the service `go.mod` has a `replace` pointing to the local `../../libs/auth` directory (monorepo layout). If you'd like, I can add example `go.mod` snippets to specific services.
