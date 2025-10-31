package main

import (
	"log"
	"net/http"

	router "github.com/KnuffelGame/KnuffelGame/backend/services/AuthService/internal"
	"github.com/KnuffelGame/KnuffelGame/backend/services/AuthService/internal/jwt"
	"github.com/KnuffelGame/KnuffelGame/backend/services/AuthService/pkg/config"
)

func main() {
	cfg := config.Load()
	gen := jwt.NewGenerator(cfg.JWTSecret)
	if cfg.JWTSecret == "" {
		log.Println("warning: JWT_SECRET is empty; token generation will fail")
	}

	r := router.New(gen)
	log.Printf("auth service listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatal(err)
	}
}
