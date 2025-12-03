package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Save original env vars
	originalEnv := map[string]string{
		"PORT":              os.Getenv("PORT"),
		"LOG_LEVEL":         os.Getenv("LOG_LEVEL"),
		"SERVICE_NAME":      os.Getenv("SERVICE_NAME"),
		"LOG_COLOR":         os.Getenv("LOG_COLOR"),
		"AUTH_SERVICE_URL":  os.Getenv("AUTH_SERVICE_URL"),
		"LOBBY_SERVICE_URL": os.Getenv("LOBBY_SERVICE_URL"),
		"GAME_SERVICE_URL":  os.Getenv("GAME_SERVICE_URL"),
		"COOKIE_DOMAIN":     os.Getenv("COOKIE_DOMAIN"),
		"COOKIE_SAMESITE":   os.Getenv("COOKIE_SAMESITE"),
		"COOKIE_SECURE":     os.Getenv("COOKIE_SECURE"),
	}

	// Clean up after test
	defer func() {
		for k, v := range originalEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	tests := []struct {
		name     string
		env      map[string]string
		expected *Config
	}{
		{
			name: "all env vars set",
			env: map[string]string{
				"PORT":              "9090",
				"LOG_LEVEL":         "debug",
				"SERVICE_NAME":      "TestGateway",
				"LOG_COLOR":         "true",
				"AUTH_SERVICE_URL":  "http://auth:8081",
				"LOBBY_SERVICE_URL": "http://lobby:8082",
				"GAME_SERVICE_URL":  "http://game:8083",
				"COOKIE_DOMAIN":     "example.com",
				"COOKIE_SAMESITE":   "Strict",
				"COOKIE_SECURE":     "true",
			},
			expected: &Config{
				Port:            "9090",
				LogLevel:        "debug",
				ServiceName:     "TestGateway",
				LogColor:        true,
				AuthServiceURL:  "http://auth:8081",
				LobbyServiceURL: "http://lobby:8082",
				GameServiceURL:  "http://game:8083",
				CookieDomain:    "example.com",
				CookieSameSite:  "Strict",
				CookieSecure:    true,
			},
		},
		{
			name: "no env vars set - defaults",
			env:  map[string]string{},
			expected: &Config{
				Port:           "8080",
				LogLevel:       "info",
				ServiceName:    "APIGateway",
				LogColor:       false,
				CookieDomain:   "localhost",
				CookieSameSite: "Lax",
				CookieSecure:   false,
			},
		},
		{
			name: "partial env vars set",
			env: map[string]string{
				"PORT":            "7070",
				"LOG_LEVEL":       "warn",
				"COOKIE_DOMAIN":   "test.com",
				"COOKIE_SAMESITE": "None",
				"COOKIE_SECURE":   "false",
				"LOG_COLOR":       "invalid",
			},
			expected: &Config{
				Port:           "7070",
				LogLevel:       "warn",
				ServiceName:    "APIGateway",
				LogColor:       false, // invalid bool defaults to false
				CookieDomain:   "test.com",
				CookieSameSite: "None",
				CookieSecure:   false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Unset all env vars first
			for k := range originalEnv {
				os.Unsetenv(k)
			}
			// Set env vars
			for k, v := range tt.env {
				if v != "" {
					os.Setenv(k, v)
				}
			}

			// Load config
			cfg := Load()

			// Check each field
			if cfg.Port != tt.expected.Port {
				t.Errorf("Port = %v, want %v", cfg.Port, tt.expected.Port)
			}
			if cfg.LogLevel != tt.expected.LogLevel {
				t.Errorf("LogLevel = %v, want %v", cfg.LogLevel, tt.expected.LogLevel)
			}
			if cfg.ServiceName != tt.expected.ServiceName {
				t.Errorf("ServiceName = %v, want %v", cfg.ServiceName, tt.expected.ServiceName)
			}
			if cfg.LogColor != tt.expected.LogColor {
				t.Errorf("LogColor = %v, want %v", cfg.LogColor, tt.expected.LogColor)
			}
			if cfg.AuthServiceURL != tt.expected.AuthServiceURL {
				t.Errorf("AuthServiceURL = %v, want %v", cfg.AuthServiceURL, tt.expected.AuthServiceURL)
			}
			if cfg.LobbyServiceURL != tt.expected.LobbyServiceURL {
				t.Errorf("LobbyServiceURL = %v, want %v", cfg.LobbyServiceURL, tt.expected.LobbyServiceURL)
			}
			if cfg.GameServiceURL != tt.expected.GameServiceURL {
				t.Errorf("GameServiceURL = %v, want %v", cfg.GameServiceURL, tt.expected.GameServiceURL)
			}
			if cfg.CookieDomain != tt.expected.CookieDomain {
				t.Errorf("CookieDomain = %v, want %v", cfg.CookieDomain, tt.expected.CookieDomain)
			}
			if cfg.CookieSameSite != tt.expected.CookieSameSite {
				t.Errorf("CookieSameSite = %v, want %v", cfg.CookieSameSite, tt.expected.CookieSameSite)
			}
			if cfg.CookieSecure != tt.expected.CookieSecure {
				t.Errorf("CookieSecure = %v, want %v", cfg.CookieSecure, tt.expected.CookieSecure)
			}
		})
	}
}
