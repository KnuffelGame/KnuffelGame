# Database Migrations

This directory contains SQL migrations for the Lobby Service database.

## Migration Tool

We use [Goose](https://github.com/pressly/goose) for database migrations. Migrations are automatically run on application startup.

## Migration Files

### 00001_create_schema.sql

Creates the core database schema with the following tables:

#### Tables

##### users
- `id` (UUID, PRIMARY KEY) - Unique user identifier
- `username` (VARCHAR(20)) - User's display name
- `created_at` (TIMESTAMP) - User creation timestamp

##### lobbies
- `id` (UUID, PRIMARY KEY) - Unique lobby identifier
- `join_code` (CHAR(6), UNIQUE) - 6-character code for joining the lobby
- `leader_id` (UUID, FOREIGN KEY -> users.id) - User ID of the lobby leader
- `status` (VARCHAR(20)) - Current lobby status (waiting, in_game, finished, closed)
- `created_at` (TIMESTAMP) - Lobby creation timestamp
- `updated_at` (TIMESTAMP) - Last update timestamp

##### players
- `id` (UUID, PRIMARY KEY) - Unique player record identifier
- `lobby_id` (UUID, FOREIGN KEY -> lobbies.id) - Reference to the lobby
- `user_id` (UUID, FOREIGN KEY -> users.id) - Reference to the user
- `joined_at` (TIMESTAMP) - When the player joined the lobby
- `is_active` (BOOLEAN) - Whether the player is currently active
- `left_at` (TIMESTAMP, NULLABLE) - When the player left (NULL if still in lobby)

#### Indices

For optimal query performance, the following indices are created:

- `idx_lobbies_join_code` - Fast lookup by join code
- `idx_players_lobby_id` - Fast lookup of players in a lobby
- `idx_players_user_id` - Fast lookup of lobbies for a user
- `idx_lobbies_status` - Fast filtering by lobby status

#### Foreign Key Constraints

All foreign keys are configured with `ON DELETE CASCADE` to maintain referential integrity:

- `lobbies.leader_id` → `users.id`
- `players.lobby_id` → `lobbies.id`
- `players.user_id` → `users.id`

## Running Migrations

Migrations are automatically executed on application startup. The service will:

1. Connect to the database using configuration from environment variables
2. Run all pending migrations in order
3. Log success or fail with an error

## Manual Migration Management

To manually manage migrations (for development/debugging), you can use the goose CLI:

```bash
# Install goose CLI
go install github.com/pressly/goose/v3/cmd/goose@latest

# Run migrations up
goose -dir migrations postgres "host=localhost port=5432 user=lobby password=secure dbname=lobby sslmode=disable" up

# Check migration status
goose -dir migrations postgres "host=localhost port=5432 user=lobby password=secure dbname=lobby sslmode=disable" status

# Rollback last migration
goose -dir migrations postgres "host=localhost port=5432 user=lobby password=secure dbname=lobby sslmode=disable" down
```

## Creating New Migrations

To create a new migration:

```bash
cd migrations
goose create <migration_name> sql
```

This will create a new timestamped migration file. Edit the file to add your SQL statements in the `-- +goose Up` and `-- +goose Down` sections.

## Environment Variables

Required database configuration:

- `DATABASE_HOST` - Database host (default: "Postgres")
- `DATABASE_PORT` - Database port (default: "5432")
- `DATABASE_USER` - Database user (default: "lobby")
- `DATABASE_PASSWORD` - Database password (default: "secure")
- `DATABASE_NAME` - Database name (default: "lobby")
- `DATABASE_SSLMODE` - SSL mode (default: "disable")

