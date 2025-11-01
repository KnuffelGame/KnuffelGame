module github.com/KnuffelGame/KnuffelGame/backend/services/AuthService

go 1.25.3

require (
	github.com/KnuffelGame/KnuffelGame/backend/libs/healthcheck v0.0.0
	github.com/KnuffelGame/KnuffelGame/backend/libs/logger v0.0.0
	github.com/go-chi/chi/v5 v5.2.3
	github.com/golang-jwt/jwt/v5 v5.3.0
	github.com/google/uuid v1.6.0
)

require (
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.22.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	golang.org/x/crypto v0.19.0 // indirect
	golang.org/x/net v0.21.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)

replace github.com/KnuffelGame/KnuffelGame/backend/libs/healthcheck => ../../libs/healthcheck

replace github.com/KnuffelGame/KnuffelGame/backend/libs/logger => ../../libs/logger
