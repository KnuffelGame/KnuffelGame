package healthcheck

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Mount registers a GET /healthcheck route on the provided chi router.
// The handler responds with status 200 and body "1" to indicate the service is healthy.
func Mount(r chi.Router) {
	r.Get("/healthcheck", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("1"))
	})
}

// Handler returns a standalone http.Handler for the healthcheck endpoint.
// Useful when composing manually or testing without Mount.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("1"))
	})
}
