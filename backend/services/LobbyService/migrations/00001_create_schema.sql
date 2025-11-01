-- +goose Up
-- +goose StatementBegin

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(20) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create lobbies table
CREATE TABLE IF NOT EXISTS lobbies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    join_code CHAR(6) NOT NULL UNIQUE,
    leader_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_leader FOREIGN KEY (leader_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Create players table
CREATE TABLE IF NOT EXISTS players (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lobby_id UUID NOT NULL,
    user_id UUID NOT NULL,
    joined_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN NOT NULL DEFAULT true,
    left_at TIMESTAMP NULL,
    CONSTRAINT fk_lobby FOREIGN KEY (lobby_id) REFERENCES lobbies(id) ON DELETE CASCADE,
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Create indices for performance
CREATE INDEX IF NOT EXISTS idx_lobbies_join_code ON lobbies(join_code);
CREATE INDEX IF NOT EXISTS idx_players_lobby_id ON players(lobby_id);
CREATE INDEX IF NOT EXISTS idx_players_user_id ON players(user_id);
CREATE INDEX IF NOT EXISTS idx_lobbies_status ON lobbies(status);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop indices
DROP INDEX IF EXISTS idx_lobbies_status;
DROP INDEX IF EXISTS idx_players_user_id;
DROP INDEX IF EXISTS idx_players_lobby_id;
DROP INDEX IF EXISTS idx_lobbies_join_code;

-- Drop tables in reverse order (respecting foreign keys)
DROP TABLE IF EXISTS players;
DROP TABLE IF EXISTS lobbies;
DROP TABLE IF EXISTS users;

-- +goose StatementEnd

