module github.com/KnuffelGame/KnuffelGame/backend/libs/auth

go 1.25.3

require (
	github.com/KnuffelGame/KnuffelGame/backend/libs/httpx v0.0.0
	github.com/KnuffelGame/KnuffelGame/backend/libs/logger v0.0.0
	github.com/google/uuid v1.6.0
)

require github.com/go-chi/chi/v5 v5.2.3 // indirect

replace github.com/KnuffelGame/KnuffelGame/backend/libs/httpx => ../httpx

replace github.com/KnuffelGame/KnuffelGame/backend/libs/logger => ../logger
