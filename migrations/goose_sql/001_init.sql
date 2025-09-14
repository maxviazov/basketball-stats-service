-- +goose Up
-- Initial schema for basketball stats service

CREATE TABLE IF NOT EXISTS teams (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS players (
    id SERIAL PRIMARY KEY,
    team_id INT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    position TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS games (
    id SERIAL PRIMARY KEY,
    season TEXT NOT NULL,
    date TIMESTAMPTZ NOT NULL,
    home_team_id INT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    away_team_id INT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK (status IN ('scheduled', 'in_progress', 'finished')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS player_stats (
    id SERIAL PRIMARY KEY,
    player_id INT NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    game_id INT NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    points INT NOT NULL DEFAULT 0,
    rebounds INT NOT NULL DEFAULT 0,
    assists INT NOT NULL DEFAULT 0,
    steals INT NOT NULL DEFAULT 0,
    blocks INT NOT NULL DEFAULT 0,
    fouls INT NOT NULL DEFAULT 0 CHECK (fouls >= 0 AND fouls <= 6),
    turnovers INT NOT NULL DEFAULT 0,
    minutes_played NUMERIC(4,1) NOT NULL DEFAULT 0 CHECK (minutes_played >= 0 AND minutes_played <= 48),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (player_id, game_id)
);

-- +goose Down
DROP TABLE IF EXISTS player_stats;
DROP TABLE IF EXISTS games;
DROP TABLE IF EXISTS players;
DROP TABLE IF EXISTS teams;
