# SSE Service

Server-Sent Events (SSE) service for the Knuffel multiplayer Kniffel application.

## Overview

The SSE Service provides real-time event broadcasting for lobby and game updates in the Knuffel multiplayer game. It manages persistent HTTP connections to clients and broadcasts events from other services (Lobby Service, Game Service) to connected clients.

## Features

- **Real-time Event Broadcasting**: Send lobby and game events to connected clients
- **Connection Management**: Handle SSE connections with automatic cleanup
- **Event Routing**: Route events to specific lobby or game audiences
- **Authentication**: JWT-based authentication for client connections
- **Health Monitoring**: Health check endpoint for service monitoring

## Architecture

```
Services (Lobby/Game) → POST /internal/publish → SSE Service
                                              ↓
                                        Broadcast to
                                        Connections
                                              ↓
                                Clients ← SSE Stream
```

## Configuration

The service reads configuration from environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Service port | `8084` |
| `JWT_SECRET` | Secret for JWT validation | Required for auth |
| `SERVICE_NAME` | Service name for logging | `SSEService` |
| `LOG_LEVEL` | Logging level | `info` |

## API Endpoints

### Health Check
- `GET /healthcheck` - Service health status

### Client SSE Endpoints (Future Implementation)
- `GET /events/lobby/{lobby_id}` - Subscribe to lobby events
- `GET /events/game/{game_id}` - Subscribe to game events

### Internal Endpoints (Future Implementation)
- `POST /internal/publish` - Publish event to connections
- `POST /internal/register` - Register lobby/game
- `POST /internal/unregister` - Unregister lobby/game
- `GET /internal/connections` - Connection statistics

## Development

### Building

```bash
go build -o sse-service ./cmd/SSEService
```

### Running

```bash
# With default configuration
./sse-service

# With custom port
PORT=8085 ./sse-service

# With JWT secret
JWT_SECRET=your-secret ./sse-service
```

### Dependencies

The service uses the following shared libraries:
- `backend/libs/healthcheck` - Health check functionality
- `backend/libs/httpx` - HTTP utilities
- `backend/libs/logger` - Structured logging

### Testing

```bash
# Run health check
curl http://localhost:8084/healthcheck

# Expected response
{"status":"ok"}
```

## Future Implementation

This service is currently in the basic structure phase. The following features will be implemented in future tickets:

1. **SSE Connection Management**
   - WebSocket upgrade handling
   - Connection registry (in-memory)
   - Automatic connection cleanup

2. **Event Publishing**
   - Internal publish endpoint
   - Event broadcasting logic
   - Connection filtering

3. **Authentication**
   - JWT token validation
   - User authorization per lobby/game

4. **Event Types**
   - Lobby events (player_joined, player_left, etc.)
   - Game events (dice_rolled, turn_changed, etc.)

## Dockerfile

See `Dockerfile` for containerized deployment configuration.

## OpenAPI Specification

See `openapi.yaml` for detailed API specification and event schemas.