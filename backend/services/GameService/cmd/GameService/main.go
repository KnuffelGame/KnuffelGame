package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/KnuffelGame/KnuffelGame/backend/services/GameService/internal/database"
	"github.com/KnuffelGame/KnuffelGame/backend/services/GameService/internal/handlers"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Server bündelt Abhängigkeiten
type Server struct {
	logger *slog.Logger
	db     *pgxpool.Pool
	router *gin.Engine
}

func main() {
	// 1. Logger initialisieren
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 2. Datenbankverbindung
	logger.Info("Verbinde mit Datenbank 'game_db'...")
	pool, err := database.Connect()
	if err != nil {
		logger.Error("Datenbankverbindung fehlgeschlagen", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("Erfolgreich mit 'game_db' verbunden.")

	// 3. Gin-Router initialisieren
	router := gin.Default()

	// Server-Struktur erstellen
	srv := &Server{
		logger: logger,
		db:     pool,
		router: router,
	}

	// --- HIER IST DIE NEUE LOGIK ---
	// 4. Abhängigkeiten initialisieren (Repository und Handler)

	// Erstelle das Game-Repository
	gameRepo := database.NewRepository(srv.db, srv.logger)

	// Erstelle den Game-Handler
	gameHandler := handlers.NewHandler(gameRepo, srv.logger)

	// 5. Routen registrieren
	srv.registerRoutes(gameHandler) // Übergebe den Handler an die Routen
	// --- ENDE NEUE LOGIK ---

	// 6. Server starten
	port := "8083"
	logger.Info("Game Service startet auf Port", "port", port)

	if err := router.Run(":" + port); err != nil {
		logger.Error("Server konnte nicht gestartet werden", "error", err)
		os.Exit(1)
	}
}

// registerRoutes bündelt alle HTTP-Routen des Servers
// Wir übergeben den Handler, damit er die Routen registrieren kann.
func (s *Server) registerRoutes(gameHandler *handlers.Handler) { // NEU
	// Health-Check-Route
	s.router.GET("/health", s.healthCheckHandler)

	// Registriere die Route für GET /games/:game_id
	s.router.GET("/games/:game_id", gameHandler.GetGameState)
}

// healthCheckHandler ist ein einfacher Handler für den Health-Check.
func (s *Server) healthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "game-service",
	})
}
