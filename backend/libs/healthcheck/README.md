# Healthcheck Library

Provides a simple way to attach a `GET /healthcheck` endpoint to a chi router.

## Usage

```
import "github.com/KnuffelGame/KnuffelGame/backend/libs/healthcheck"

r := chi.NewRouter()
healthcheck.Mount(r)
```

The endpoint responds with HTTP 200 and body `1` plus `Content-Type: text/plain; charset=utf-8`.

You can also obtain a standalone handler:

```
h := healthcheck.Handler()
```

## Versioning
Internal library; use a replace directive pointing to the local path in service modules.

