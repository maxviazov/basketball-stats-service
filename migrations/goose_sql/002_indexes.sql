-- +goose Up
-- Add indexes for performance on foreign keys and filtering
CREATE INDEX IF NOT EXISTS idx_players_team_id ON players(team_id);
CREATE INDEX IF NOT EXISTS idx_games_season ON games(season);
CREATE INDEX IF NOT EXISTS idx_games_date ON games(date);
CREATE INDEX IF NOT EXISTS idx_games_home_team_id ON games(home_team_id);
CREATE INDEX IF NOT EXISTS idx_games_away_team_id ON games(away_team_id);
CREATE INDEX IF NOT EXISTS idx_player_stats_player_id ON player_stats(player_id);
CREATE INDEX IF NOT EXISTS idx_player_stats_game_id ON player_stats(game_id);

-- +goose Down
DROP INDEX IF EXISTS idx_players_team_id;
DROP INDEX IF EXISTS idx_games_season;
DROP INDEX IF EXISTS idx_games_date;
DROP INDEX IF EXISTS idx_games_home_team_id;
DROP INDEX IF EXISTS idx_games_away_team_id;
DROP INDEX IF EXISTS idx_player_stats_player_id;
DROP INDEX IF EXISTS idx_player_stats_game_id;
