package handlers

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// ReverseProxy creates a reverse proxy handler that forwards requests to the specified target URL.
// It returns an http.Handler that can be used as a route handler.
func ReverseProxy(target string) http.Handler {
	targetURL, err := url.Parse(target)
	if err != nil {
		// If target is invalid, return a handler that returns 500
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Invalid proxy target", http.StatusInternalServerError)
		})
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Optionally, modify the request before forwarding
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// Remove hop-by-hop headers
		req.Header.Del("Connection")
		req.Header.Del("Keep-Alive")
		req.Header.Del("Proxy-Authenticate")
		req.Header.Del("Proxy-Authorization")
		req.Header.Del("TE")
		req.Header.Del("Trailers")
		req.Header.Del("Transfer-Encoding")
		req.Header.Del("Upgrade")
	}

	return proxy
}
