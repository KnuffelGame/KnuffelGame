# 1. Builder Stage
FROM golang:1.25.3-alpine AS builder

WORKDIR /app

# 1. Caching: Erst nur go.mod/go.sum kopieren
# Wenn du im Ordner 'GameService' stehst, liegen diese Dateien direkt hier.
COPY go.mod go.sum ./
RUN go mod download

# 2. Source Code kopieren
COPY . .

# Jetzt sind wir in /app.
# Wenn deine Struktur so ist:
# GameService/
# ├── cmd/
# │   └── GameService/
# │       └── main.go
# └── go.mod
#
# Dann ist der Pfad ./cmd/GameService korrekt relativ zu /app

# 3. Build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o gameservice ./cmd/GameService

# --- Runtime Stage ---
FROM alpine:3.21

WORKDIR /app

RUN apk --no-cache add curl

COPY --from=builder /app/gameservice .

HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8082/healthcheck || exit 1

EXPOSE 8082

CMD ["./gameservice"]