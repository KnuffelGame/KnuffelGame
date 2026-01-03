package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	"github.com/KnuffelGame/KnuffelGame/backend/services/GameService/internal/db"
	"github.com/KnuffelGame/KnuffelGame/backend/services/GameService/internal/handlers"
	"github.com/KnuffelGame/KnuffelGame/backend/services/GameService/internal/services"
	"github.com/KnuffelGame/KnuffelGame/backend/services/GameService/pkg/config"
	"github.com/gin-gonic/gin"

	// WICHTIG: Der Treiber muss hier importiert sein
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Server bündelt Abhängigkeiten
type Server struct {
	logger *slog.Logger
	dbConn *sql.DB // ÄNDERUNG: Hier nutzen wir direkt *sql.DB
	router *gin.Engine
}

func main() {
	// 1. Setup Logger & Config
	if os.Getenv("SERVICE_NAME") == "" {
		_ = os.Setenv("SERVICE_NAME", "GameService")
	}
	log := logger.FromEnv().With(slog.String("component", "bootstrap"))

	cfg := config.Load()

	// Connection String bauen (DSN)
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.DatabaseUser,
		cfg.DatabasePassword,
		cfg.DatabaseHost,
		cfg.DatabasePort,
		cfg.DatabaseName,
		cfg.DatabaseSSLMode,
	)

	log.Info("Connecting to database", slog.String("dsn_masked", fmt.Sprintf("postgres://%s:***@%s:%s/%s", cfg.DatabaseUser, cfg.DatabaseHost, cfg.DatabasePort, cfg.DatabaseName)))

	// 2. Datenbank initialisieren (NUR EINMAL)
	// Wir nutzen sql.Open mit dem "pgx" Treiber
	dbConn, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Error("failed to open database connection", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer dbConn.Close()

	// Testen ob die Verbindung wirklich steht (Ping)
	if err := dbConn.Ping(); err != nil {
		log.Error("failed to ping database", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 3. Migrationen ausführen (Nutzt dieselbe Verbindung)
	if err := db.RunMigrations(dbConn); err != nil {
		log.Error("failed to run migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 4. Gin-Router initialisieren
	router := gin.Default()

	// Server-Struktur erstellen
	srv := &Server{
		logger: log,
		dbConn: dbConn, // Das ist jetzt unser *sql.DB
		router: router,
	}

	// 5. Abhängigkeiten initialisieren

	// Repository erstellen
	// Hinweis: Unser 'NewRepository' aus dem vorherigen Schritt braucht nur *sql.DB.
	// Falls du den Logger im Repo brauchst, musst du NewRepository anpassen.
	// Ich gehe hier vom Standard aus:
	gameRepo := db.NewRepository(srv.dbConn)

	gameService := services.NewGameService(gameRepo)

	// Handler erstellen
	gameHandler := handlers.NewHandler(gameRepo, srv.logger, gameService)

	// 6. Routen registrieren
	srv.registerRoutes(gameHandler)

	// 7. Server starten
	port := "8082"
	log.Info("Game Service startet", "port", port)

	if err := router.Run(":" + port); err != nil {
		log.Error("Server konnte nicht gestartet werden", "error", err)
		os.Exit(1)
	}
}

// registerRoutes bündelt alle HTTP-Routen des Servers
func (s *Server) registerRoutes(gameHandler *handlers.Handler) {
	// Health-Check-Route
	s.router.GET("/healthcheck", s.healthCheckHandler) // Habe den Pfad auf /healthcheck angepasst (passend zum Dockerfile)

	// Game Routes
	// s.router.GET("/games/:game_id", gameHandler.GetGameState)
	s.router.POST("/internal/create", gameHandler.CreateGame)
	s.router.POST("/games/:game_id/roll", gameHandler.PostRollDice)
	s.router.POST("/games/:game_id/toggle-dice", gameHandler.PostSelectDice)
	s.router.POST("games/:game_id/select-field", gameHandler.PostSelectScoreField)
	s.router.GET("/games/:game_id", gameHandler.GetGameState)
}

// healthCheckHandler ist ein einfacher Handler für den Health-Check.
func (s *Server) healthCheckHandler(c *gin.Context) {
	// Wir prüfen hier auch kurz, ob die DB noch da ist
	if err := s.dbConn.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "db": "disconnected"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "game-service",
	})
}
