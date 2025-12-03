package config

import (
	"os"
	"strconv"
)

// Config holds runtime configuration loaded from environment variables.
// PORT defaults to 8080 if unset.
// LOG_LEVEL defaults to "info" if unset.
// SERVICE_NAME defaults to "APIGateway" if unset.
// LOG_COLOR defaults to false if unset or invalid.
// COOKIE_DOMAIN defaults to "localhost" if unset.
// COOKIE_SAMESITE defaults to "Lax" if unset.
// COOKIE_SECURE defaults to false if unset or invalid.
// Other fields must be set via environment variables.

type Config struct {
	Port            string
	LogLevel        string
	ServiceName     string
	LogColor        bool
	AuthServiceURL  string
	LobbyServiceURL string
	GameServiceURL  string
	CookieDomain    string
	CookieSameSite  string
	CookieSecure    bool
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	serviceName := os.Getenv("SERVICE_NAME")
	if serviceName == "" {
		serviceName = "APIGateway"
	}
	logColorStr := os.Getenv("LOG_COLOR")
	logColor := false
	if logColorStr != "" {
		if b, err := strconv.ParseBool(logColorStr); err == nil {
			logColor = b
		}
	}
	authServiceURL := os.Getenv("AUTH_SERVICE_URL")
	lobbyServiceURL := os.Getenv("LOBBY_SERVICE_URL")
	gameServiceURL := os.Getenv("GAME_SERVICE_URL")
	cookieDomain := os.Getenv("COOKIE_DOMAIN")
	if cookieDomain == "" {
		cookieDomain = "localhost"
	}
	cookieSameSite := os.Getenv("COOKIE_SAMESITE")
	if cookieSameSite == "" {
		cookieSameSite = "Lax"
	}
	cookieSecureStr := os.Getenv("COOKIE_SECURE")
	cookieSecure := false
	if cookieSecureStr != "" {
		if b, err := strconv.ParseBool(cookieSecureStr); err == nil {
			cookieSecure = b
		}
	}
	return &Config{
		Port:            port,
		LogLevel:        logLevel,
		ServiceName:     serviceName,
		LogColor:        logColor,
		AuthServiceURL:  authServiceURL,
		LobbyServiceURL: lobbyServiceURL,
		GameServiceURL:  gameServiceURL,
		CookieDomain:    cookieDomain,
		CookieSameSite:  cookieSameSite,
		CookieSecure:    cookieSecure,
	}
}
