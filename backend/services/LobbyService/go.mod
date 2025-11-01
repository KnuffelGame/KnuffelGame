module github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService

go 1.25.3

require (
	github.com/KnuffelGame/KnuffelGame/backend/libs/healthcheck v0.0.0
	github.com/KnuffelGame/KnuffelGame/backend/libs/httpx v0.0.0
	github.com/KnuffelGame/KnuffelGame/backend/libs/logger v0.0.0
	github.com/go-chi/chi/v5 v5.2.3 // indirect
	github.com/lib/pq v1.10.9 // indirect
)

replace github.com/KnuffelGame/KnuffelGame/backend/libs/healthcheck => ../../libs/healthcheck

replace github.com/KnuffelGame/KnuffelGame/backend/libs/httpx => ../../libs/httpx

replace github.com/KnuffelGame/KnuffelGame/backend/libs/logger => ../../libs/logger