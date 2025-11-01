# Lobby Service

The Lobby Service manages game lobbies, including creation, joining, and player management.

## Features

- Create new game lobbies
- Generate unique join codes
- Manage lobby participants
- Track lobby status (waiting, in_game, finished, closed)

## API Endpoints

### POST /lobbies

Creates a new lobby with the requesting user as the leader.

**Headers:**
- `X-User-ID` (required): UUID of the user (provided by API Gateway)
- `X-Username` (required): Username of the user (provided by API Gateway)

**Request:**
```http
POST /lobbies HTTP/1.1
X-User-ID: 550e8400-e29b-41d4-a716-446655440000
X-Username: Alice
```

**Response (201 Created):**
```json
{
  "lobby_id": "123e4567-e89b-12d3-a456-426614174000",
  "join_code": "ABC123",
  "leader_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "waiting",
  "players": [
    {
      "id": "789e4567-e89b-12d3-a456-426614174111",
      "user_id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "Alice",
      "joined_at": "2025-11-01T12:34:56Z",
      "is_active": true
    }
  ]
}
```

**Behavior:**
1. Creates user in database if not exists (using `ON CONFLICT DO NOTHING`)
2. Generates unique 6-character alphanumeric join code
3. Creates lobby with status "waiting"
4. Sets requesting user as lobby leader
5. Automatically adds user as first player

**Error Responses:**
- `400 Bad Request`: Missing or invalid headers
- `500 Internal Server Error`: Database error or join code generation failure

## Database Schema

### users
- `id` (UUID, PK): User identifier
- `username` (VARCHAR(20)): Username
- `created_at` (TIMESTAMP): Creation timestamp

### lobbies
- `id` (UUID, PK): Lobby identifier
- `join_code` (CHAR(6), UNIQUE): Join code for the lobby
- `leader_id` (UUID, FK -> users.id): Lobby leader
- `status` (VARCHAR(20)): Current status (waiting, in_game, finished, closed)
- `created_at` (TIMESTAMP): Creation timestamp
- `updated_at` (TIMESTAMP): Last update timestamp

### players
- `id` (UUID, PK): Player entry identifier
- `lobby_id` (UUID, FK -> lobbies.id): Associated lobby
- `user_id` (UUID, FK -> users.id): Associated user
- `joined_at` (TIMESTAMP): Join timestamp
- `is_active` (BOOLEAN): Active status
- `left_at` (TIMESTAMP, nullable): Leave timestamp

## Configuration

Environment variables:

- `PORT`: Service port (default: 8083)
- `DATABASE_HOST`: PostgreSQL host (default: Postgres)
- `DATABASE_PORT`: PostgreSQL port (default: 5432)
- `DATABASE_USER`: Database user (default: lobby)
- `DATABASE_PASSWORD`: Database password (default: secure)
- `DATABASE_NAME`: Database name (default: lobby)
- `DATABASE_SSLMODE`: SSL mode (default: disable)

## Dependencies

- PostgreSQL database
- Join code generator (internal/joincode)
- Healthcheck library (libs/healthcheck)
- Logger library (libs/logger)
- HTTP utilities library (libs/httpx)

## Running Tests

```bash
# Set up test database first
# Then run tests
go test ./...
```

## Building

```bash
go build ./cmd/LobbyService
```

## Docker

```bash
docker build -t lobby-service .
docker run -p 8083:8083 --env-file env.d/LobbyService.env lobby-service
```

