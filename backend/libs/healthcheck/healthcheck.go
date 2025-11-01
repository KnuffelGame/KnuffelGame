package healthcheck

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Mount registers a GET /healthcheck route on the provided chi router.
// The handler responds with status 200 and JSON {"status":"ok"}.
func Mount(r chi.Router) {
	r.Get("/healthcheck", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{\"status\":\"ok\"}"))
	})
}

// Handler returns a standalone http.Handler for the healthcheck endpoint.
// Useful when composing manually or testing without Mount.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{\"status\":\"ok\"}"))
	})
}
