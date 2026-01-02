-- +goose Up
-- ENUM-Typ für den Spielstatus
CREATE TYPE game_status_enum AS ENUM (
    'lobby',    -- Spiel ist in der Lobby, wartet auf Spieler
    'active',   -- Spiel läuft gerade
    'finished', -- Spiel ist regulär beendet
    'abandoned' -- Spiel wurde abgebrochen
);

-- ENUM-Typ für die Scorecard-Felder (Kniffel/Yahtzee-basiert)
CREATE TYPE scorecard_field_enum AS ENUM (
    'ones',
    'twos',
    'threes',
    'fours',
    'fives',
    'sixes',
    'bonus',
    'three_of_a_kind',
    'four_of_a_kind',
    'full_house',
    'small_straight',
    'large_straight',
    'kniffel', -- (oder 'yahtzee')
    'chance'
);

---
-- Tabelle: games
-- Speichert den Hauptstatus jedes Spiels
---
CREATE TABLE IF NOT EXISTS games (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lobby_id UUID NOT NULL,
    status game_status_enum NOT NULL DEFAULT 'lobby',

    -- Index des Spielers in 'turn_order', der gerade dran ist
    current_turn INT NOT NULL DEFAULT 0,

    -- JSON-Array von User-IDs, z.B. ["uuid-player1", "uuid-player2"]
    turn_order JSONB NOT NULL DEFAULT '[]'::jsonb,

    round INT NOT NULL DEFAULT 1,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ
);

-- Index für die schnelle Suche nach Lobby-ID
CREATE INDEX IF NOT EXISTS idx_games_lobby_id ON games(lobby_id);

---
-- Tabelle: turns
-- Speichert jeden einzelnen Zug (3 Würfe) eines Spielers
---
CREATE TABLE IF NOT EXISTS turns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Foreign Key zur games Tabelle
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,

    roll_count INT NOT NULL DEFAULT 0 CHECK (roll_count BETWEEN 0 AND 3),

    -- Aktuelle Würfelwerte, z.B. [5, 2, 1, 5, 3]
    dice_values JSONB NOT NULL DEFAULT '[]'::jsonb,

    kept_dice JSONB NOT NULL DEFAULT '[]'::jsonb,

    timeout BOOLEAN NOT NULL DEFAULT false,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    round INT NOT NULL DEFAULT 1
);

-- Index für die schnelle Suche nach Zügen eines Spiels
CREATE INDEX IF NOT EXISTS idx_turns_game_id ON turns(game_id);
CREATE INDEX IF NOT EXISTS idx_turns_user_id ON turns(user_id);

---
-- Tabelle: scorecards
-- Speichert die ausgefüllten Felder der Spieler
---
CREATE TABLE IF NOT EXISTS scorecards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Foreign Key zur games Tabelle
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,

    field_name scorecard_field_enum NOT NULL,
    value INT NOT NULL DEFAULT 0,
    round_filled INT, -- In welcher Runde wurde dies ausgefüllt?

    -- Wichtiger Constraint: Ein Spieler kann jedes Feld pro Spiel nur einmal ausfüllen
    UNIQUE(game_id, user_id, field_name)
);

-- Index für die schnelle Suche nach der Scorecard eines Spielers in einem Spiel
CREATE INDEX IF NOT EXISTS idx_scorecards_game_user ON scorecards(game_id, user_id);

COMMIT;