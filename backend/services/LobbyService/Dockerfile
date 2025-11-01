FROM golang:1.25.3-alpine AS builder

COPY . /app
WORKDIR /app/services/LobbyService
RUN go build -o lobbyservice ./cmd/LobbyService

FROM alpine:3.22.2

WORKDIR /app
RUN apk --no-cache add curl
COPY --from=builder /app/services/LobbyService/lobbyservice .

HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 CMD curl -f http://localhost:8083/healthcheck || exit 1

EXPOSE 8083

CMD ["./lobbyservice"]